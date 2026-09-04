# pgscope

pgscope is an open-source, real-time database monitoring tool for PostgreSQL and MySQL/MariaDB. It lets you see, live, what queries are running on a database, what locks each one holds, and who is waiting on whom, both as a dense table and as a visual blocking graph. Beyond live monitoring, it also generates advisory insights (slow queries, missing/duplicate/unused indexes, deep-offset pagination, lock contention, and more) by reading each engine's own statistics views.

## Why this project, and how it differs from existing tools

Tools like pg_activity, pgHero, and pgAdmin already cover parts of what pgscope does, and pgscope's Insights module deliberately borrows pgHero's proven thresholds and query patterns for things like unused index detection, since there was no reason to reinvent a heuristic that has already been validated in production by a widely used tool. What pgscope adds on top is not a replacement for any of them, it targets a different gap.

pg_activity is terminal based, which is great for someone who already lives in a database console but not something you can hand to a teammate who has never opened `psql`. pgAdmin is a full database administration console: powerful, but it assumes the person using it already knows what they are looking for and is comfortable running arbitrary SQL. pgHero is closer in spirit to pgscope, a web dashboard aimed at surfacing problems without requiring deep Postgres expertise, but it does not visualize blocking relationships as an interactive graph, and it does not have a record and replay mode for going back to a specific incident after the fact.

pgscope's actual goal is to take the kind of information a database administrator would normally read directly off `pg_stat_activity`, `pg_locks`, and similar system views, information that requires knowing which view to query, how to join it with the others, and how to interpret the result, and present it in a form that someone without that background can still understand. A developer who has never administered a database should be able to open pgscope, see that a checkout endpoint is slow, and see for themselves whether that slowness is caused by their own application code or by something happening inside the database (a blocked query, a missing index, table bloat, a stuck transaction) without needing a DBA to translate `pg_stat_activity` for them first. If the bottleneck does turn out to be database side, pgscope's Insights module gives that same non-expert user a plain-language explanation and a concrete, testable suggestion, not just a wall of raw numbers. Everything pgscope surfaces is also directly useful to an actual DBA, but the tool is written so that it does not require being one.

## What pgscope is, and is not

pgscope is deliberately **read-only**. Every piece of information it shows comes from a `SELECT` query against the connected engine's own statistics views (Postgres's `pg_stat_activity`, `pg_stat_statements`, `pg_stat_user_tables`, `pg_locks`; MySQL's `performance_schema`, `sys`, `information_schema`, and similar), the same views a database administrator would already query manually from a terminal or a database console. pgscope does not add any capability that was not already available to someone with direct database access; it only makes that same information reachable through a web interface, without requiring the terminal or console access itself.

pgscope never writes, alters, or deletes anything in the database it monitors, and on Postgres it never runs `EXPLAIN` against a live query either, since even a read-only `EXPLAIN` can force Postgres to actually execute parts of a statement depending on the plan. There are no "kill query" or "terminate connection" buttons, no "apply this suggested index" button, and no way to change a database setting from the UI, and there never will be by default. This is an observation and advisory tool, not a control panel.

This is a deliberate scope decision, not a missing feature. Any action beyond reading statistics (killing a session, dropping an index, changing a GUC, running an administrative function) requires a level of judgment about the current state of a live production system that pgscope, or any automated tool, cannot safely have. A wrong decision made automatically at the wrong moment can take a database down. Rather than trying to build that judgment into the tool and risk getting it wrong, pgscope leaves every action step to the people who already have direct database access and already know how to take it safely: the actual database administrators, running the actual command themselves, in their own console, with full context of what else is happening on that system at that moment. pgscope's job stops at showing them, or a less experienced teammate, that something needs attention in the first place.

Keeping the tool to `SELECT`-only queries against statistics views also keeps its security surface small and easy to reason about on purpose. A tool that only ever reads has a fundamentally smaller attack surface than one that can also write, and that difference is enforced here at the database role level, not just in the application code, so it holds even if the application itself had a bug. On Postgres this guarantee is enforced at multiple independent layers (database role privileges, session settings, and the application driver itself); on MySQL it currently rests on database role privileges alone, since MySQL has no per-user session-settings layer equivalent to Postgres's `default_transaction_read_only` (see the MySQL setup section below). Each layer is documented in [go/README.md](go/README.md)'s security model section.

## What it targets today, and where it is going

- **Backend**: a small, self-contained Go service that connects to Postgres or MySQL/MariaDB with a least-privilege, read-only role and streams live activity over Server-Sent Events.
- **Frontend**: a lightweight React + Vite dashboard that consumes that live stream: a session table, a blocking graph, live commit/rollback stats, a record/replay mode for going back and watching a past incident unfold second by second, and an Insights panel for advisory suggestions. The dashboard knows which engine it is connected to and hides suggestions that engine cannot produce, instead of showing a Postgres-only instruction against a MySQL database.

Postgres was the first engine pgscope supported, and MySQL/MariaDB is the second. The backend is built with hexagonal architecture so that everything engine-specific (SQL, system view names, driver calls) lives in its own isolated package (`internal/infrastructure/postgres`, `internal/infrastructure/mysql`) behind a shared set of interfaces (`internal/application/port/output`). The domain layer and the application layer that use those interfaces have no idea which engine, or how many, are actually connected. Adding support for another database means writing a new `infrastructure/<engine>` package that implements the same interfaces; nothing in `domain`, `application`, or the frontend has to change. The MySQL adapter is the proof that this actually works rather than just a claim about the architecture: it was written without touching a single line in `domain` or `application`.

MySQL support is not a full duplicate of every Postgres insight, and it should not be treated as one. Some things Postgres exposes have no real MySQL equivalent and are simply not shown when connected to a MySQL database, rather than faked or approximated: vacuum health, checkpoint health, replication lag and replication slot health, physical I/O hotspots (which depend on the Postgres-only `pg_stat_kcache` extension), invalid indexes and unvalidated constraints (Postgres-specific catalog states left behind by an aborted `CREATE INDEX CONCURRENTLY` or a `NOT VALID` constraint), and function/trigger cost tracking (MySQL's `performance_schema` does not track a stored routine's own execution time separately from the statement that called it, and does not track trigger execution at all). Function and trigger cost tracking specifically is not a paid-tier limitation, it is a genuine gap in what MySQL's own instrumentation exposes, at least in the community edition. MySQL does have one insight Postgres does not: row-lock wait detection via `sys.innodb_lock_waits`, showing which session is blocked on which and for how long, since Postgres's own equivalent (`pg_locks` plus `pg_blocking_pids()`) is already covered by pgscope's live blocking graph rather than being a separate advisory.

The relational engines planned after MySQL are Oracle and MSSQL, since both expose comparable session, lock, and statistics views that a similar read-only role and similar `SELECT`-based queries can reach. MongoDB is also a longer-term goal, but since it is not relational, its monitoring surface (`db.currentOp()`, profiler data, `serverStatus()`) works differently enough that it needs its own research pass before an adapter can be designed properly rather than assuming the relational model maps over directly. The same applies to cache-oriented stores like Redis: their relevant statistics (memory usage, eviction rate, slow log) come from a different set of commands entirely, and supporting them well means reading their documentation first to understand what is actually worth surfacing, not just porting the relational approach over.

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

## MySQL/MariaDB setup

pgscope needs a MySQL 8.0+ (or compatible MariaDB) instance reachable over TCP. This is the full setup, typically run against a `mysql:8.0` container via Docker Compose.

### 1. Server-level config

Add this to your `docker-compose.yaml` mysql service:

    command:
      - "--performance-schema=ON"

`performance_schema` ships on by default on most MySQL 8.0 builds, but pass it explicitly rather than assuming, since it is what pgscope's session/lock collectors and normalized query text both depend on. Unlike Postgres's `pg_stat_statements`, this needs no extension install and no per-database enable step, it is a server-wide setting only.

Verify it took effect:

    docker exec <container_name> mysql -uroot -p -e "SHOW VARIABLES LIKE 'performance_schema';"

### 2. Create a dedicated database and a monitoring user

The standard `mysql:8.0` Docker image's `MYSQL_USER`/`MYSQL_PASSWORD`/`MYSQL_DATABASE` environment variables create that user with **every privilege** on that database by default, which is the opposite of what pgscope needs. Either create the user manually instead of through those variables, or create it that way and then revoke everything, as step 4 below does.

    CREATE DATABASE pgscope_demo;

Generate a strong password first:

    openssl rand -base64 48

    CREATE USER 'pgscope_agent'@'%' IDENTIFIED BY '<paste-generated-password-here>';

### 3. Grant read access to the statistics views

    GRANT SELECT ON performance_schema.* TO 'pgscope_agent'@'%';
    GRANT SELECT ON sys.* TO 'pgscope_agent'@'%';
    GRANT EXECUTE ON sys.* TO 'pgscope_agent'@'%';
    GRANT PROCESS ON *.* TO 'pgscope_agent'@'%';

**Easy to miss:** the `sys` schema's own views (`sys.innodb_lock_waits`, `sys.schema_unused_indexes`, `sys.schema_redundant_indexes`) are `SQL SECURITY INVOKER` views that call stored functions defined in `sys` itself (`sys.format_statement`, `sys.quote_identifier`, and similar). `GRANT SELECT ON sys.*` alone is not enough to use them; without `GRANT EXECUTE ON sys.*` too, every query against those views fails with `Error 1356: View ... references invalid table(s) or column(s) or function(s) or definer/invoker of view lack rights to use them`, a confusing error that does not obviously point at a missing `EXECUTE` grant. `PROCESS` is what makes every session visible in `performance_schema.threads`, not just the monitoring connection's own, MySQL's rough equivalent of Postgres's `pg_monitor` role.

### 4. Lock the user down

If the user was created through the Docker image's `MYSQL_USER` variables (see step 2), revoke the default grant first:

    REVOKE ALL PRIVILEGES ON pgscope_demo.* FROM 'pgscope_agent'@'%';

Either way, confirm the user has no table-level access to your actual application data, only to the statistics schemas granted in step 3:

    SHOW GRANTS FOR 'pgscope_agent'@'%';

This is what makes the read-only claim actually true rather than just a UI restriction: `pgscope_agent` should show grants on `performance_schema`, `sys`, and `PROCESS` only, nothing on the application database's own tables.

### 5. Timeouts

Postgres lets a role carry its own default `statement_timeout` and `lock_timeout` (`ALTER ROLE ... SET ...`), so pgscope's Postgres role enforces short timeouts on itself automatically, on every connection, with no cooperation needed from the application. MySQL has no equivalent per-user default: `max_execution_time` is either a server-wide global or something a client sets for its own session after connecting, not something a role carries with it. pgscope does not currently set it per-session on the MySQL connections it opens (see `internal/infrastructure/mysql/pool.go`), so this layer of defense that exists on the Postgres side is not yet duplicated on the MySQL side. Read-only enforcement (step 3 and 4 above, grant-level) still holds regardless; this is specifically about a runaway monitoring query being killed automatically, which today only happens on the Postgres adapter.

### 6. Verify

Connect as `pgscope_agent` with a **fresh** connection:

    SELECT id, `user`, db, command FROM information_schema.processlist; -- should show all sessions
    CREATE TABLE test_write (id int); -- must fail: no privilege on the application schema

If both of those behave as expected, the database side is done. Point `PGSCOPE_DATABASE_URL` at this user and set `PGSCOPE_DB_ENGINE=mysql` (it defaults to `postgres` otherwise); backend and frontend setup are documented separately, since those are about running the actual application, not preparing the database.

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