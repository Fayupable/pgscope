# pgscope, backend

The backend is a small, self-contained Go service. It connects to PostgreSQL with a **read-only, least-privilege role**, polls session/lock/stat data, streams it live to the frontend over Server-Sent Events (SSE), and separately generates advisory insights by reading Postgres's statistics views on demand. It never writes to the database it monitors: that guarantee is enforced at three independent layers (Postgres role, session GUCs, and the Go driver config), not just by convention.

PostgreSQL setup (creating the database, the extension, and the least-privilege monitoring role) is documented in the [root README](../README.md); do that first if you haven't. This document assumes that role (`pgscope_agent`) already exists and works.

## Prerequisites

- Go 1.22+
- A running PostgreSQL instance with the setup from the root README completed
- (Optional, for generating test traffic) a **write-capable** Postgres user, see [Load generator](#load-generator-devtest-only) below

## Directory structure

    go/
      cmd/
        pgscope/        — the actual product: main.go wires everything together
        loadgen/         — separate dev/test-only binary, generates fake load (see below)
      internal/
        domain/          — pure Go types, no Postgres/MySQL-specific vocabulary, no external imports
        application/
          port/output/    — interfaces the application layer depends on (ISessionCollectorPort, IHistoryStorePort, IInsightsPort, ...)
          service/         — use cases: MonitoringService, Poller, Recorder, InsightsService
        infrastructure/
          postgres/        — Postgres-specific implementations of the output ports (SQL lives only here)
          history/          — in-memory ring buffer implementing IHistoryStorePort
          sse/              — Server-Sent Events broadcaster + publisher adapter
          config/           — environment variable loading
        presentation/
          http/            — router.go (routes only), handlers.go (thin handlers), response.go (JSON encoding), plus the access-control layer: auth_middleware.go, auth_handlers.go, rate_limit_middleware.go, ban_store.go, ban_middleware.go, attempt_tracker.go, request_timeout_middleware.go

This follows hexagonal architecture strictly: `domain` never imports anything outside itself. `application` depends only on `domain` and its own `port` interfaces, never on a concrete database driver. `infrastructure/postgres` is the only package in the whole codebase that knows any PostgreSQL-specific SQL or terminology. If MySQL or MSSQL support is added later, it becomes a new `infrastructure/mysql` package implementing the same `output` interfaces; `domain`, `application`, and the entire frontend stay untouched. See the root README for why other database engines are a long-term goal for this project, not just a hypothetical.

## Running it

Create `go/.env` (gitignored, never commit real credentials):

    PGSCOPE_DATABASE_URL=host=localhost port=5432 user=pgscope_agent password=<your-password> dbname=pgscope_demo sslmode=disable
    PGSCOPE_API_KEY=<a randomly generated secret, e.g. from openssl rand -base64 32>
    PGSCOPE_HTTP_PORT=8090
    PGSCOPE_POLL_INTERVAL_SECONDS=1

| Variable | Required | Default | Notes |
|---|---|---|---|
| `PGSCOPE_DATABASE_URL` | yes | none | Use the `key=value` DSN format, not `postgres://`. Passwords containing `+` or `/` break URL parsing unless percent-encoded; the keyword format avoids that entirely. |
| `PGSCOPE_API_KEY` | yes | none | The shared secret clients must send to `/api/v1/auth/login` to receive a session cookie. Treat it as a real credential, generate it randomly, never commit it. See the Security model section below for the full authentication flow. |
| `PGSCOPE_HTTP_PORT` | no | `8090` | Port the HTTP/SSE server listens on. |
| `PGSCOPE_POLL_INTERVAL_SECONDS` | no | `1` | How often the backend queries Postgres while monitoring is active. Shorter intervals catch short-lived queries more reliably but cost more; longer intervals are gentler on the target database. See the note on polling below. |

Run:

    go run ./cmd/pgscope

On startup the server does **nothing** database-side until told to, see [Monitoring lifecycle](#monitoring-lifecycle) below. This is intentional: pgscope should never query a production database just because the process happens to be running. It also requires a valid session before responding to almost any request at all, see Security model below.

## Monitoring lifecycle

Two independent, opt-in controls gate everything:

- **Monitor**, whether the backend queries Postgres at all and publishes live data over SSE. Off by default. Started with a duration (5/10/30 minutes) or indefinitely ("Full", `minutes=0`), stopped manually or automatically when the duration elapses.
- **Record**, whether, *while monitoring is active*, each tick is also written to an in-memory history buffer for later export/replay. Off by default, always bounded to one of `5, 10, 15, 30` minutes, never indefinite, since history exists for short-window export, not open-ended growth. Stopping Monitor also stops Recording (recording without monitoring makes no sense).

Both are implemented with the same small `Recorder` type (`internal/application/service/recorder.go`), a mutex-guarded active flag with an optional `time.AfterFunc` auto-stop timer. `Poller.tick()` checks `monitorControl.IsActive()` first; if false, it returns immediately without touching the database at all.

### Why polling can miss very short queries

`pg_stat_activity` is a snapshot view, it only reflects what's running *at the instant it's queried*. A query that starts and finishes entirely between two polling ticks is never observed, no matter how short you set `PGSCOPE_POLL_INTERVAL_SECONDS`. This is a fundamental limitation of polling, not a bug. pgscope is designed around this: it's meant to catch queries and locks that last long enough to matter (seconds, not milliseconds); anything shorter is better served by `pg_stat_statements`'s aggregate call/duration counters, which pgscope already relies on for query text masking and for the Insights panel.

## History and the ring buffer

`internal/infrastructure/history/ring_buffer_store.go` holds two separate in-memory slices:

- **Periodic snapshots**, capped at 300 entries (about 5 minutes at a 1-second poll interval). Oldest entries are evicted first (FIFO).
- **Incident snapshots**, capped at 50 entries, kept separately and evicted less aggressively. A tick is classified as an "incident" (`domain.SnapshotTriggerIncident`) when `Poller.classifyTrigger()` detects a session that is blocked *now* but wasn't blocked in the previous tick, meaning a new blocking relationship just appeared. Everything else is `periodic`.

`GET /api/v1/history` returns the periodic buffer's current contents as a JSON array; this is what the frontend's "Download JSON" button fetches and what the Replay feature loads back in.

## Insights

`GET /api/v1/insights` runs a separate, on-demand analysis pass: `internal/infrastructure/postgres/insights_collector.go` fans out to more than a dozen focused collectors, each reading one specific statistics view, and hands the raw numbers to a matching `domain` type that applies the actual judgment (thresholds, confidence scoring, and a plain-language explanation). The collectors never decide anything themselves; they only fetch. This split exists so every threshold used to decide "is this worth flagging" lives in one place per feature, in `domain`, independent of how the raw number was fetched, and is unit-testable without a database connection.

What it currently covers, each backed by its own pair of a raw-stats collector and a pure-judgment domain function:

- **Top queries and index candidates**, from `pg_stat_statements` and `pg_stat_user_tables`, following pgHero's proven thresholds for when a sequential-scan pattern is worth suggesting an index for.
- **Duplicate and unused indexes**, from the system catalog and `pg_stat_user_indexes`.
- **Function and trigger cost**, from `pg_stat_user_functions`, only populated if `track_functions` is set to `pl` or `all`; pgscope reads this setting but never changes it, and tells you in the UI if it's off.
- **Deep-offset pagination warnings**, detected from `OFFSET` usage patterns in `pg_stat_statements` combined with execution-time variance, always presented as an inference rather than a certainty since pgscope only sees aggregate statistics, never the actual `OFFSET` values used per call.
- **Database size, connection saturation, sequence overflow risk, invalid indexes and unvalidated constraints, vacuum health, and idle-in-transaction sessions**, each a straightforward read of the relevant catalog or stats view.
- **Checkpoint health and replication lag**, from `pg_stat_bgwriter` and `pg_stat_replication`.
- **Physical I/O hotspots**, only available if the optional `pg_stat_kcache` extension is installed; pgscope checks for its presence and never installs it itself.

Every suggestion Insights produces is worded as something to verify, never as a certainty, for the same reason described in the root README: pgscope only sees aggregate statistics, not your full application context, so it cannot know for certain that a suggestion is safe to apply.

`/api/v1/insights` is deliberately the most rate-limited endpoint in the whole API, because a single request to it triggers roughly fifteen sequential queries against a connection pool that only holds three connections. See Security model below for the exact limits.

## API

| Method | Path | Purpose |
|---|---|---|
| GET | `/healthz` | Liveness check. Returns `200 ok`. Not authenticated, since it's meant for infrastructure health checks that won't have a session cookie. |
| POST | `/api/v1/auth/login` | Body `{"key": "..."}`. If it matches `PGSCOPE_API_KEY`, sets a session cookie and returns `{"authenticated": true}`. Otherwise returns `401` and counts as a failed attempt toward the IP ban threshold. |
| POST | `/api/v1/auth/logout` | Clears the session cookie. |
| GET | `/api/v1/auth/status` | Returns `200` if the request's session cookie is currently valid, `401` otherwise. Used by the frontend on page load to decide whether to show the login screen. |
| GET | `/api/v1/sessions/stream` | SSE stream. Emits `sessions` (array of active sessions) and `db_stats` (commit/rollback rate) events, plus a `: heartbeat` comment every 15s to keep the connection alive. Only emits real data while monitoring is active, otherwise the connection stays open but silent. Requires a valid session. |
| POST | `/api/v1/monitor/start?minutes=N` | Start live polling. `minutes` omitted or `0` means run until stopped ("Full"). Negative values are rejected with `400`. Requires a valid session. |
| POST | `/api/v1/monitor/stop` | Stop live polling. Also stops recording, if it was active. Requires a valid session. |
| POST | `/api/v1/record/start?minutes=N` | Start writing ticks to history. `minutes` must be exactly one of `5, 10, 15, 30`, anything else returns `400`. Requires a valid session. |
| POST | `/api/v1/record/stop` | Stop recording (monitoring keeps running). Requires a valid session. |
| GET | `/api/v1/history` | Returns the current periodic snapshot buffer as a JSON array. Can be called at any time, even mid-recording; you don't have to wait for the window to finish. Requires a valid session. |
| GET | `/api/v1/insights` | Runs the full analysis pass described above and returns the result. Requires a valid session. Bounded to roughly one request per 5 seconds per IP, and to 5 seconds total server-side wall time before returning `503` if the connection pool is saturated. |

Example session:

    curl -c cookies.txt -X POST -H "Content-Type: application/json" \
      -d '{"key":"<your PGSCOPE_API_KEY>"}' http://localhost:8090/api/v1/auth/login
    curl -b cookies.txt -X POST "http://localhost:8090/api/v1/monitor/start?minutes=10"
    curl -b cookies.txt -X POST "http://localhost:8090/api/v1/record/start?minutes=5"
    curl -b cookies.txt http://localhost:8090/api/v1/history > snapshot.json
    curl -b cookies.txt -X POST "http://localhost:8090/api/v1/record/stop"
    curl -b cookies.txt -X POST "http://localhost:8090/api/v1/monitor/stop"

## Security model

- **Role level**: `pgscope_agent` has no write grants anywhere (`REVOKE ALL` on schema and database), only `CONNECT` and `pg_monitor`. It physically cannot run `INSERT`/`UPDATE`/`DELETE`/DDL even if the application code were compromised.
- **Session level**: the role has `default_transaction_read_only = on` and short `statement_timeout` / `lock_timeout` set via `ALTER ROLE`, applied automatically on every connection regardless of what the application does.
- **Driver level**: `internal/infrastructure/postgres/pool.go` sets the same read-only and timeout parameters again independently via `pgxpool.Config.ConnConfig.RuntimeParams`, and uses a small, isolated connection pool (`MaxConns = 3`) that is never shared with any other workload.
- **Query text**: every place pgscope shows a query, it comes from `pg_stat_statements` (parameterized, `$1`, `$2`, ...), never from `pg_stat_activity`'s raw `query` column. This means literal values (passwords, tokens, PII typed into a `WHERE` or `SET` clause) are never exposed through the tool, even by accident. Table and column names are still shown, since they're schema metadata, not user data.

**Authentication.** pgscope requires a single shared secret, `PGSCOPE_API_KEY`, to access anything beyond `/healthz`. `POST /api/v1/auth/login` compares the submitted key to it using `crypto/subtle.ConstantTimeCompare`, so the comparison takes the same amount of time regardless of how much of the key matches, which prevents an attacker from guessing the key one character at a time by measuring response times. On success it issues a cookie (`pgscope_session`) marked `HttpOnly` (JavaScript running on the page cannot read it, which limits what a cross-site-scripting bug could steal), `Secure` (never sent over plain HTTP), and `SameSite=Strict` (never sent along with a request that originated from another site, which is the main defense against cross-site request forgery). The cookie's value is the API key itself rather than a separately signed token; a dedicated session-token scheme was deliberately left out, since with only one shared credential in the whole system, a signed session token would add implementation complexity without meaningfully changing what an attacker could do with a stolen cookie.

**Rate limiting.** Every route is wrapped in a per-IP token bucket (`internal/presentation/http/rate_limit_middleware.go`), with a separate, smaller budget for `/api/v1/auth/login` than for everything else, since login attempts are the most sensitive to being guessed repeatedly. `/api/v1/insights` has its own, stricter budget on top of that, since a single request there triggers around fifteen sequential queries against a three-connection pool.

**Automatic IP banning.** Four failed login attempts, or four requests to a well-known vulnerability-scanner path (`/.env`, `/wp-login.php`, `/.git/config`, and similar; see `internal/presentation/http/ban_middleware.go` for the full list), ban that IP for seven days. A single mistake, such as mistyping the key once, never triggers a ban on its own; only a repeated pattern does. Bans are held in memory and do not survive a process restart, which is an accepted tradeoff at pgscope's intended scale (a single instance, not a fleet behind a load balancer); restarting the process is effectively "unban everyone."

**Request timeouts.** `internal/presentation/http/request_timeout_middleware.go` bounds the total time `/api/v1/insights` is allowed to take, including any time spent waiting for a free connection from the three-connection pool. If the pool is saturated and a connection can't be acquired in time, the request fails fast with `503 Service Unavailable` instead of the handler's goroutine blocking indefinitely.

**Reverse proxy note.** `clientIP()` (`internal/presentation/http/rate_limit_middleware.go`) trusts the `X-Real-IP` header for rate limiting and ban decisions. This is only safe if pgscope sits behind a reverse proxy (nginx, Caddy) configured to set that header itself from the real connecting address, overwriting any value a client tries to send, for example nginx's `proxy_set_header X-Real-IP $remote_addr;`. If pgscope is ever exposed directly, without a proxy in front of it doing this, remove that header check, since a client could otherwise spoof their own IP and bypass rate limiting and banning entirely.

**Error responses.** `writeError()` (`internal/presentation/http/response.go`) logs the real error server-side but only ever returns a generic status-text message to the client, so an internal error (a database driver message, a connection detail) is never leaked through an HTTP response body.

**What is deliberately not exposed.** pgscope has no endpoint that runs anything other than a `SELECT` against a statistics view. There is no way to kill a session, terminate a connection, run `VACUUM`, change a `postgresql.conf` setting, or execute arbitrary SQL through the API, even for an authenticated user. Someone who needs to take one of those actions already has, or should have, direct access to the database itself, through `psql` or their own admin console, and can do it there with full context of what else is running on that system at that moment. Adding any of those as an API endpoint would mean pgscope's `pgscope_agent` role would need write privileges it currently does not and should not have, turning a small, auditable read-only surface into something that needs the same level of security scrutiny as the database itself. That tradeoff was deliberately avoided.

## Load generator (dev/test only)

`cmd/loadgen` is a **completely separate binary** from `cmd/pgscope`, same Go module, different entrypoint, never shipped or run in production. It exists purely to generate realistic concurrent database activity so there's something worth watching in the graph while developing.

It connects with a **write-capable** role (e.g. your Postgres superuser), never `pgscope_agent`, which is read-only by design and would simply fail every write.

### Seed test data

    CREATE TABLE test_orders (id serial primary key, status text, amount numeric);
    INSERT INTO test_orders (status, amount) VALUES ('pending', 100), ('pending', 200), ('done', 50);

For a richer, more realistic graph (30+ nodes with hubs, chains, and isolated nodes), reseed with more rows:

    TRUNCATE test_orders RESTART IDENTITY;
    INSERT INTO test_orders (status, amount)
    SELECT 'pending', 100 FROM generate_series(1, 300);

### Run it

    export LOADGEN_DATABASE_URL="host=localhost port=5432 user=<superuser> password=<password> dbname=pgscope_demo sslmode=disable"
    go run ./cmd/loadgen

This starts an HTTP trigger server on port `9091` (configurable via `LOADGEN_HTTP_PORT`); it does nothing until you POST to it. This trigger server has no authentication of its own, since it is a dev-only tool never meant to run anywhere reachable outside your own machine.

### Trigger a run

    curl -X POST "http://localhost:9091/run?sessions=50"

- `sessions` is required, must be a positive integer, capped at 250.
- Each triggered run self-stops after 60 seconds regardless of how many sessions were spawned; it will never run indefinitely.
- Workers are split three ways by `workerID % 5`:
  - **Hub workers** (2 of every 5) all compete for a small pool of 5 "hot" rows, producing fan-in nodes that many sessions wait on at once.
  - **Chain workers** (2 of every 5) lock their own row plus the *next* worker's row, producing genuine multi-hop waiting chains (A waits on B waits on C).
  - **Independent workers** (1 of every 5) touch a row nobody else uses; they show up as isolated nodes with zero relationships.

This mix is what produces the varied, realistic-looking graphs (hubs, chains, and isolated nodes together) instead of everything either colliding on one row or never colliding at all.

## Testing changes

    go build ./...
    go vet ./...
    go test ./...
    golangci-lint run

There is a small, growing table-driven test suite, currently covering the pure-logic parts of `domain` (`IndexSignal.Confidence()`, `PaginationSignal.Suspected()`, `ClassifyOperation()`) and a couple of text-processing helpers in `infrastructure/postgres` (`extractSuspectedColumns`, `StatsResetReader`). New tests follow the same convention: table-driven, `_test.go` files alongside the code they test, and preferring to test the `domain` layer directly wherever possible, since it requires no database connection and no mocking.

`golangci-lint` (config in `.golangci.yml`) enforces the hexagonal architecture boundaries described in [CONTRIBUTING.md](../CONTRIBUTING.md) via the `depguard` linter: `domain` cannot import `infrastructure` or the Postgres driver, `application` cannot import `infrastructure`, and only `infrastructure/postgres` may import the Postgres driver. This runs automatically in CI, and a violation fails the build even if `go test` passes, since breaking these boundaries doesn't cause a test failure on its own.