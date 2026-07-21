# pgscope

pgscope is an open-source, real-time PostgreSQL monitoring tool. It lets you see, live, what queries are running on a database, what locks each one holds, and who is waiting on whom, both as a dense table and as a visual blocking graph. Beyond live monitoring, it also generates advisory insights (slow queries, missing/duplicate/unused indexes, expensive functions and triggers, deep-offset pagination, vacuum and checkpoint health, replication lag, and more) by reading Postgres's own statistics views.

## Why this project, and how it differs from existing tools

Tools like pg_activity, pgHero, and pgAdmin already cover parts of what pgscope does, and pgscope's Insights module deliberately borrows pgHero's proven thresholds and query patterns for things like unused index detection, since there was no reason to reinvent a heuristic that has already been validated in production by a widely used tool. What pgscope adds on top is not a replacement for any of them, it targets a different gap.

pg_activity is terminal based, which is great for someone who already lives in a database console but not something you can hand to a teammate who has never opened `psql`. pgAdmin is a full database administration console: powerful, but it assumes the person using it already knows what they are looking for and is comfortable running arbitrary SQL. pgHero is closer in spirit to pgscope, a web dashboard aimed at surfacing problems without requiring deep Postgres expertise, but it does not visualize blocking relationships as an interactive graph, and it does not have a record and replay mode for going back to a specific incident after the fact.

pgscope's actual goal is to take the kind of information a database administrator would normally read directly off `pg_stat_activity`, `pg_locks`, and similar system views, information that requires knowing which view to query, how to join it with the others, and how to interpret the result, and present it in a form that someone without that background can still understand. A developer who has never administered a database should be able to open pgscope, see that a checkout endpoint is slow, and see for themselves whether that slowness is caused by their own application code or by something happening inside the database (a blocked query, a missing index, table bloat, a stuck transaction) without needing a DBA to translate `pg_stat_activity` for them first. If the bottleneck does turn out to be database side, pgscope's Insights module gives that same non-expert user a plain-language explanation and a concrete, testable suggestion, not just a wall of raw numbers. Everything pgscope surfaces is also directly useful to an actual DBA, but the tool is written so that it does not require being one.

## What pgscope is, and is not

pgscope is deliberately **read-only**. Every piece of information it shows comes from a `SELECT` query against Postgres's own statistics views (`pg_stat_activity`, `pg_stat_statements`, `pg_stat_user_tables`, `pg_locks`, and similar), the same views a database administrator would already query manually from a terminal or a database console. pgscope does not add any capability that was not already available to someone with direct database access; it only makes that same information reachable through a web interface, without requiring the terminal or console access itself.

pgscope never writes, alters, or deletes anything in the database it monitors, and it never runs `EXPLAIN` against a live query either, since even a read-only `EXPLAIN` can force Postgres to actually execute parts of a statement depending on the plan. There are no "kill query" or "terminate connection" buttons, no "apply this suggested index" button, and no way to change a Postgres setting from the UI, and there never will be by default. This is an observation and advisory tool, not a control panel.

This is a deliberate scope decision, not a missing feature. Any action beyond reading statistics (killing a session, dropping an index, changing a GUC, running an administrative function) requires a level of judgment about the current state of a live production system that pgscope, or any automated tool, cannot safely have. A wrong decision made automatically at the wrong moment can take a database down. Rather than trying to build that judgment into the tool and risk getting it wrong, pgscope leaves every action step to the people who already have direct database access and already know how to take it safely: the actual database administrators, running the actual command themselves, in their own console, with full context of what else is happening on that system at that moment. pgscope's job stops at showing them, or a less experienced teammate, that something needs attention in the first place.

Keeping the tool to `SELECT`-only queries against statistics views also keeps its security surface small and easy to reason about on purpose. A tool that only ever reads has a fundamentally smaller attack surface than one that can also write, and that difference is enforced here at the database role level, not just in the application code, so it holds even if the application itself had a bug. This guarantee is enforced at multiple independent layers (database role privileges, session settings, and the application driver itself), and each layer is documented in [go/README.md](go/README.md)'s security model section, redundant with the others on purpose: if one layer were ever misconfigured, the others still hold.

## What it targets today, and where it is going

- **Backend**: a small, self-contained Go service that connects to Postgres with a least-privilege, read-only role and streams live activity over Server-Sent Events.
- **Frontend**: a lightweight React + Vite dashboard that consumes that live stream: a session table, a blocking graph, live commit/rollback stats, a record/replay mode for going back and watching a past incident unfold second by second, and an Insights panel for advisory suggestions.

Postgres is the first engine pgscope supports, but not the only one it is meant to support. The backend is deliberately built with hexagonal architecture so that everything Postgres-specific (SQL, system view names, `pgx` driver calls) lives in one isolated package (`internal/infrastructure/postgres`) behind a set of interfaces (`internal/application/port/output`). The domain layer and the application layer that use those interfaces have no idea Postgres exists. Adding support for another database means writing a new `infrastructure/<engine>` package that implements the same interfaces; nothing in `domain`, `application`, or the frontend has to change.

The relational engines planned after Postgres are MySQL, Oracle, and MSSQL, since all three expose comparable session, lock, and statistics views that a similar read-only role and similar `SELECT`-based queries can reach. MongoDB is also a longer-term goal, but since it is not relational, its monitoring surface (`db.currentOp()`, profiler data, `serverStatus()`) works differently enough that it needs its own research pass before an adapter can be designed properly rather than assuming Postgres's model maps over directly. The same applies to cache-oriented stores like Redis: their relevant statistics (memory usage, eviction rate, slow log) come from a different set of commands entirely, and supporting them well means reading their documentation first to understand what is actually worth surfacing, not just porting the Postgres approach over.

## Access control

pgscope requires a single shared API key, set via `PGSCOPE_API_KEY`, to access anything beyond the `/healthz` endpoint. This is intentionally simple rather than a full user/password/role system: pgscope is built for a single operator or a small trusted team looking at one database, not a multi-tenant product, so the usual reasons to build out full user management do not apply here. Authentication details (how the key is checked, how the session cookie is issued, rate limiting, and automatic IP banning after repeated failures) are documented in [go/README.md](go/README.md)'s security model section.

If you plan to run pgscope somewhere reachable from outside your own machine, whether that is a shared internal network or the public internet, treat `PGSCOPE_API_KEY` as a real secret: generate it randomly (for example `openssl rand -base64 32`), never commit it to a repository, and put pgscope behind HTTPS. The cookie pgscope issues after login is marked `Secure`, meaning browsers will refuse to send it over a plain HTTP connection, so HTTPS is not optional for a deployment reachable outside `localhost`.

## PostgreSQL setup

pgscope needs a PostgreSQL instance reachable over TCP. This is the full setup, typically run against a `postgres:16-alpine` container via Docker Compose.

### 1. Cluster-level config

Add this to your `docker-compose.yaml` postgres service:

    command:
      - "postgres"
      - "-c"
      - "shared_preload_libraries=pg_stat_statements"
      - "-c"
      - "track_activity_query_size=4096"
      - "-c"
      - "track_io_timing=on"
      - "-c"
      - "log_min_duration_statement=1000"

This is cluster-wide, meaning it applies to every database in the container, and requires a restart (`docker compose up -d`) since `shared_preload_libraries` only loads at server start. `pg_stat_statements` is what lets pgscope show masked, parameterized query text (`$1`, `$2`) instead of raw SQL with literal values baked in, and it is also the source of most of the Insights panel's data (top queries, index candidates, pagination warnings).

Verify it took effect:

    docker exec <container_name> psql -U <superuser> -c "SHOW shared_preload_libraries;"

### 2. Create a dedicated database

    CREATE DATABASE pgscope_demo;

Connect to `pgscope_demo` specifically (not the default `postgres` database) for the next step.

### 3. Enable the extension inside that database

    CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

Unlike step 1, this is per-database, not cluster-wide.

### 4. Create a least-privilege monitoring role

Generate a strong password first:

    openssl rand -base64 48

    CREATE ROLE pgscope_agent WITH
        LOGIN
        NOSUPERUSER
        NOCREATEDB
        NOCREATEROLE
        INHERIT
        NOBYPASSRLS
        CONNECTION LIMIT 10
        PASSWORD '<paste-generated-password-here>';

    GRANT pg_monitor TO pgscope_agent WITH INHERIT TRUE;

**Important gotcha:** on PostgreSQL 16+, `WITH INHERIT TRUE` must be stated explicitly on the `GRANT` itself, since inherit behavior is stored per-grant, not derived from the role's own `INHERIT` attribute. If you grant `pg_monitor` before this is set correctly, `pg_stat_activity` will silently return `<insufficient privilege>` for every session except the role's own. Verify with:

    SELECT r.rolname AS member, g.rolname AS granted_role, m.inherit_option
    FROM pg_auth_members m
    JOIN pg_roles r ON m.member = r.oid
    JOIN pg_roles g ON m.roleid = g.oid
    WHERE r.rolname = 'pgscope_agent';

`inherit_option` must be `true`. If it isn't, re-run the `GRANT ... WITH INHERIT TRUE` and reconnect with a **fresh** session, since privilege changes never apply to connections that were already open.

### 5. Lock the role down

    REVOKE ALL ON SCHEMA public FROM pgscope_agent;
    REVOKE ALL ON DATABASE pgscope_demo FROM pgscope_agent;
    GRANT CONNECT ON DATABASE pgscope_demo TO pgscope_agent;
    ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE SELECT ON TABLES FROM pgscope_agent;
    REVOKE CONNECT ON DATABASE postgres FROM pgscope_agent;

This is what makes the read-only claim actually true rather than just a UI restriction: `pgscope_agent` has no table-level grants at all. `pg_monitor` gives it access to statistics views specifically, not to your application's actual data tables.

### 6. Enforce read-only and timeouts at the role level

    ALTER ROLE pgscope_agent SET default_transaction_read_only = on;
    ALTER ROLE pgscope_agent SET statement_timeout = '250ms';
    ALTER ROLE pgscope_agent SET lock_timeout = '100ms';
    ALTER ROLE pgscope_agent SET idle_in_transaction_session_timeout = '1s';

The short timeouts exist so that pgscope itself can never become the thing it is trying to help you diagnose: a long-running query or a session sitting idle in a transaction, holding locks. If a monitoring query somehow takes longer than 250ms, Postgres kills it rather than letting it pile up.

### 7. Verify

Connect as `pgscope_agent` with a **fresh** connection:

    SELECT pid, usename, state, query FROM pg_stat_activity; -- should show all sessions
    CREATE TABLE test_write (id int); -- must fail: read-only transaction

If both of those behave as expected, the database side is done. Backend and frontend setup are documented separately, since those are about running the actual application, not preparing Postgres.

## Project layout

    go/     — backend (Go)
    web/    — frontend (React + Vite)

- Backend setup and details: [go/README.md](go/README.md)
- Frontend setup and details: [web/README.md](web/README.md)

## References

pgscope's approach and specific thresholds were developed by reading the actual PostgreSQL source and documentation rather than assuming how a statistics view behaves, and by studying how an existing, production-proven tool (pgHero) approaches the same problems:

- [PostgreSQL documentation](https://www.postgresql.org/docs/)
- [PostgreSQL source code](https://github.com/postgres/postgres)
- [The Cumulative Statistics System](https://www.postgresql.org/docs/current/monitoring-stats.html), the reference for every `pg_stat_*` view pgscope reads from
- [pgHero](https://github.com/ankane/pgHero), whose unused-index and slow-query heuristics pgscope's Insights module deliberately follows rather than reinventing