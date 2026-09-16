# Comp HQ — Decisions log

Non-obvious choices made while building Comp HQ, in the format required by `PLAN.md` §2.4: one short paragraph each, headed `D-NN Title (task ID, date)`.

## D-01 Name and identifiers (P1-02, 2026-09-16)

The app is named Comp HQ (owner, 14 Sep 2026). All identifiers use `comphq` / `COMPHQ_*` (repository `stas-comp/comphq`, binary `comphq`, container image `ghcr.io/stas-comp/comphq`, environment variables prefixed `COMPHQ_`). `SPEC.md` was updated accordingly.

## D-02 Single SQLite connection (P1-03, 2026-09-16)

`internal/db.Open` sets `sqlDB.SetMaxOpenConns(1)`. SQLite allows only one writer at a time; a single shared connection is the simplest way to avoid "database is locked" errors, at the cost of serialising reads too. Revisit if the layer 5 speed tests (P1-27) show read contention — a common fix is a second, read-only connection pool.

## D-03 Embedding at the repository root (P1-03, 2026-09-16)

`go:embed` patterns can't reach outside the directory of the file that declares them, but `web/` and `migrations/` are siblings of `internal/` and `cmd/` (SPEC B2's layout). So the embed directives live in a small package at the repository root (`assets.go`, `package comphq`), imported by `internal/app` and `cmd/comphq`. Every later section's templates and migrations are covered automatically since the patterns are `all:web/templates` and `all:migrations`.

## D-03 CI tiers and minutes (P1-05, 2026-09-16)

CI runs layers 1, 2 and 6 (`go` and `browser` jobs) on every push. Layers 4, 5 and 7 (`container`, `speed`, `screens`) run only when the commit message contains `[container]`/`[speed]`, on a full `workflow_dispatch`, or on a version tag — every release runs everything. Reason: GitHub Free private repos get 2,000 CI minutes/month on 2-vCPU runners (§3), and layers 4/5/7 are the slowest. The plan's path-based auto-trigger for `container` (when `Dockerfile`, `deploy/`, `cmd/`, `internal/db/`, or `migrations/` change) is deferred to P1-06, once those paths actually exist to watch — until then, use the `[container]` flag or a full dispatch/tag run.

## D-04 Command surface and integration build tag (P1-05, 2026-09-16)

npm scripts (`package.json`) are the single command surface for build and test, in both PowerShell and Bash, matching CI exactly. Go HTTP integration tests are gated behind the `integration` build tag (`//go:build integration`) so `go test ./...` (unit, fast) and `go test -tags integration ./...` (integration, spawns real servers/binaries) stay separate.

## D-02 Base image variant and digests (P1-06, 2026-09-16)

Final image: `gcr.io/distroless/static-debian13:nonroot`, pinned by index digest `sha256:e2e927ec666bae08560abb3c55d0659eceabb657f56b6782ab500a9fc7f555e3` (SPEC said `distroless/static`; `-debian13` is the current variant, confirmed against `docker buildx imagetools inspect`). `deploy/truenas.yaml`'s `user: "568:568"` overrides the image's built-in `nonroot` user to TrueNAS SCALE's default apps UID/GID. Build image: `golang:1.27.1-bookworm`, pinned by index digest `sha256:648f440f42a0958804efb24df176f806f9d353b41f1c0627f666428e40310f6b` (also confirmed via `imagetools inspect`).

## D-06b Skipped the `gh api .../versions` publish check (P1-06, 2026-09-16)

PLAN.md suggests confirming the published image with `gh api /users/stas-comp/packages/container/comphq/versions`, but that needs the `read:packages` scope, which the S1 sign-in didn't grant (it asked only for `workflow`) and adding it needs another interactive browser step — not one of the plan's stop points, so not something to ask the owner for here. The `release.yml` `publish` job (which runs `docker/build-push-action`) completing successfully is itself the automated proof the image reached `ghcr.io`; that CI result is the evidence for this task's "Done when", not a manual follow-up call. P1-07 (making the package public) can also confirm it, anonymously, with no extra scope needed.

## D-06a Container-test data proof before People exists (P1-06, 2026-09-16)

`deploy/test/container-test.sh`'s restart/recreate proof (gates 1.39, 1.40) needs some data to check survives. There's no HTTP endpoint that writes data yet — the People section (P1-13) is the first one — so the script writes a marker file directly into the bind-mounted `/data` folder and checksums the whole folder, instead of "create a person via HTTP" as later releases' copies of this proof will do once that's possible.

## D-08 Internal pre-release tags (P1-06, 2026-09-16)

`v0.0.1` (this task) and `v0.0.2` (P1-28, once the Knowledge Base has real data) are internal pre-release tags, never installed by the owner. They exist so the upgrade-and-rollback test (gate 1.42) has a previous release to roll back to well before the real `v0.1.0` phase release.

## D-04b Placeholder `e2e/speed` spec (P1-06, 2026-09-16)

Playwright exits non-zero on "No tests found", and the `speed` CI job (P1-05) runs on a full dispatch or a release tag — both needed for this task's `v0.0.1` release. Real speed tests arrive in P1-27 with the test-library seeder; until then, `e2e/speed/placeholder.spec.ts` keeps the job green, the same pattern as D-04a.

## D-04a Placeholder `tools/policy` package (P1-04, 2026-09-16)

`npm run check` (added in full at P1-04) includes `test:policy`, which runs `go test ./tools/policy/...`. That path must exist for the command to work at all — an absent package makes `go test` fail outright, not pass vacuously. P1-05 adds the real repository policy checks named in the plan; for now `tools/policy` holds one placeholder test so the command surface is complete and green from P1-04 onward, as the task requires.
