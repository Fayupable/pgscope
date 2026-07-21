## What does this PR do

## Why

## How was this tested

- [ ] `go build ./... && go vet ./... && go test ./...` passes
- [ ] `npx tsc -b` passes (if frontend changed)
- [ ] Manually tested in the browser (describe what you clicked through, if UI changed)

## Checklist

- [ ] New backend code follows the hexagonal architecture rules in `CONTRIBUTING.md` (no SQL outside `infrastructure/postgres`, no Postgres vocabulary in `domain`)
- [ ] Any new Insights category's SQL filter/order matches the domain layer's actual judgment criteria (see `CONTRIBUTING.md`'s note on this recurring bug class)
- [ ] This does not add any capability beyond read-only `SELECT` against statistics views