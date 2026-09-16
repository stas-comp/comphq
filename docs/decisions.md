# Comp HQ — Decisions log

Non-obvious choices made while building Comp HQ, in the format required by `PLAN.md` §2.4: one short paragraph each, headed `D-NN Title (task ID, date)`.

## D-01 Name and identifiers (P1-02, 2026-09-16)

The app is named Comp HQ (owner, 14 Sep 2026). All identifiers use `comphq` / `COMPHQ_*` (repository `stas-comp/comphq`, binary `comphq`, container image `ghcr.io/stas-comp/comphq`, environment variables prefixed `COMPHQ_`). `SPEC.md` was updated accordingly.

## D-02 Single SQLite connection (P1-03, 2026-09-16)

`internal/db.Open` sets `sqlDB.SetMaxOpenConns(1)`. SQLite allows only one writer at a time; a single shared connection is the simplest way to avoid "database is locked" errors, at the cost of serialising reads too. Revisit if the layer 5 speed tests (P1-27) show read contention — a common fix is a second, read-only connection pool.

## D-03 Embedding at the repository root (P1-03, 2026-09-16)

`go:embed` patterns can't reach outside the directory of the file that declares them, but `web/` and `migrations/` are siblings of `internal/` and `cmd/` (SPEC B2's layout). So the embed directives live in a small package at the repository root (`assets.go`, `package comphq`), imported by `internal/app` and `cmd/comphq`. Every later section's templates and migrations are covered automatically since the patterns are `all:web/templates` and `all:migrations`.

## D-04a Placeholder `tools/policy` package (P1-04, 2026-09-16)

`npm run check` (added in full at P1-04) includes `test:policy`, which runs `go test ./tools/policy/...`. That path must exist for the command to work at all — an absent package makes `go test` fail outright, not pass vacuously. P1-05 adds the real repository policy checks named in the plan; for now `tools/policy` holds one placeholder test so the command surface is complete and green from P1-04 onward, as the task requires.
