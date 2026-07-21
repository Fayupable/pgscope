# Contributing to pgscope

Thanks for considering a contribution. This document covers how to get set up locally, what's expected of a pull request, and the conventions this codebase follows.

## Branching model

`main` is the stable, released branch. Active development happens on a versioned `dev/*` branch (e.g. `dev/2026-07`) — check the repository's branch list for the most recently active `dev/*` branch. Please branch off the current `dev/*` branch and open your pull request against it, not `main`. Periodically, the current `dev/*` branch is merged into `main` as a release, and a new `dev/*` branch is opened to continue development.

## Getting set up

Follow the setup steps in the root [README.md](README.md) (PostgreSQL role setup), then [go/README.md](go/README.md) (backend) and [web/README.md](web/README.md) (frontend). If you'd rather use Docker, see the `docker-compose.yml` and `docker-compose.local.yml` files, both documented in the same READMEs.

## Before opening a pull request

Run these and make sure they pass:

```
cd go && go build ./... && go vet ./... && go test ./...
cd web && npx tsc -b
```

CI runs these automatically on every pull request targeting `main` or a `dev/*` branch (see `.github/workflows/`), but please still run them yourself before pushing — it's faster to catch a failure locally than to wait for CI. CI also runs `golangci-lint` (backend, enforces the architecture rules below via `depguard`) and `eslint`/`stylelint` (frontend, enforces the conventions in [web/README.md](web/README.md)).

## Architecture rules (please read before writing backend code)

The Go backend follows hexagonal architecture strictly, and pull requests that break these rules will be asked to change before merging:

- `internal/domain` contains pure Go types and logic only. No Postgres-specific vocabulary, no SQL, no imports outside the standard library and other `domain` code. This is what keeps the project able to support other database engines later without rewriting this layer.
- `internal/application` depends only on `domain` and its own `port/output` interfaces, never on a concrete database driver or infrastructure package.
- `internal/infrastructure/postgres` is the only package allowed to contain SQL or PostgreSQL-specific terminology. If you're adding a new statistic pgscope should surface, the pattern is: a raw-stats collector in `infrastructure/postgres` that only fetches numbers, plus a pure judgment function in `domain` that decides what those numbers mean and writes the explanation text. Look at any existing pair (e.g. `unused_index_collector.go` + `domain/unused_index.go`) as a template.
- A SQL query's `WHERE`/`ORDER BY`/`LIMIT` must match the actual threshold the corresponding `domain` function uses to judge relevance. A mismatch here (ranking by one metric while filtering by a different one) has been the single most common bug class in this codebase's history — it silently drops valid candidates before the domain layer ever sees them.

## What pgscope will never do

Every suggestion pgscope surfaces is advisory. Pull requests that add the ability to execute anything beyond a read-only `SELECT` against a statistics view (killing a session, running `VACUUM`, applying a suggested index, changing a Postgres setting, running arbitrary SQL) will not be merged, regardless of how well-intentioned or well-implemented. See the root README's "What pgscope is, and is not" section for the reasoning. If you want pgscope to do more than observe and suggest, that's a fork, not a pull request here.

## Testing conventions

- Table-driven Go tests, `_test.go` files alongside the code they test.
- Prefer testing the `domain` layer directly wherever possible — it requires no database connection and no mocking, since it's pure logic over plain Go structs.
- Frontend tests use `vitest`; keep new pure-logic utilities (like anything under `shared/utils/graph/`) covered the same way `edges.ts` already is.

## Code style

- No comments explaining *what* code does — names should already make that clear. A comment is only warranted for a non-obvious *why* (a hidden constraint, a workaround, an invariant that isn't visible from the code itself).
- No premature abstractions. Three similar lines are better than a shared helper built for a single caller.
- Constructor injection only in Go — no service locators, no globals.

## Reporting bugs and requesting features

Please use the issue templates — they ask for the context needed to act on a report without a lot of back-and-forth.

## Roadmap

Multi-database engine support (MySQL, MSSQL, Oracle, and eventually non-relational stores) is a long-term goal — see the root README.

