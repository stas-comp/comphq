# Comp HQ — Build Plan

**Status:** Phases 1–3 built and released as v1.0.0. **Phase 4 (v1.1) added 19 September 2026** — draft for owner approval.
**Contract:** `SPEC.md` Part A (approved 14 Sep 2026). **Technical direction:** `SPEC.md` Part B, as adjusted in §7 of this plan.
**Name:** the app is **Comp HQ** (owner decision, 14 Sep 2026). All wording a person sees says "Comp HQ". All technical names use `comphq`: repository `stas-comp/comphq`, image `ghcr.io/stas-comp/comphq`, binary `comphq`, cookie `comphq_person`, env vars `COMPHQ_*`, database `comphq.db`, export `comphq-export-YYYY-MM-DD.zip`, TrueNAS app and dataset `comphq`. SPEC.md has already been updated to match. The build folder on this computer stays `C:\Users\Knacker\Desktop\Staff HQ`; don't rename it.

---

## 1. For the owner (please read this part)

**What happens, in order**

| Phase | What gets built | About how many working sessions | What you get at the end |
|---|---|---|---|
| Getting ready | Tools installed on this computer, your private GitHub storage for the code, and an automatic test-and-publish line, proven end to end with an empty app | 6–8 | Nothing to check. Two short sign-in moments (below). |
| **Phase 1** | The frame, name picker, Knowledge Base (editor, pictures, history, search, Import from Word), backups, export, and installing on the NAS | 35–45 | Version **v0.1.0** |
| **Phase 2** | Tasks (My jobs, Board, Team) and Calendar | 20–25 | Version **v0.2.0** |
| **Phase 3** | Saturday Briefing | 5–8 | Version **v1.0.0** |
| **Phase 4** | The new look right through the app (matching your design mockup), the task window you open by clicking a job, assigning people from a card, and the fix for jobs not showing in My jobs and Team | 25–35 | Version **v1.1.0** |

A "session" is one sitting of the build agent. It runs by itself; you don't need to watch.

**Every moment the build stops and waits for you**

1. **Right at the start: sign in to GitHub** (account **stas-comp**) on this computer. About 5 minutes, with click-by-click steps.
2. **Soon after: make the Comp HQ download "public"** in GitHub, so your NAS can fetch it without a password. It contains no data. About 3 minutes.
3. *(Not a stop, just a reminder during Phase 1)*: if you like, put 1–3 real Word documents (with pictures and tables, nothing private) in the `samples\word` folder inside this folder. The build won't wait for them.
4. **End of Phase 1:** install Comp HQ on the NAS, make desktop shortcuts, try it, move your HelpScout articles over, and do the four 👤 Owner checks. You can let Phase 2 start before the HelpScout move is finished.
5. **End of Phase 2:** try Tasks and Calendar, enter your yearly events and Trello cards, and do the two 👤 Owner checks.
6. **End of Phase 3:** check the Saturday Briefing, and do a final look around.
7. **End of Phase 4:** open the design mockup next to the app and agree each screen matches; make a task with someone on it and confirm it shows in their My jobs and Team; open a task by clicking it and try editing it; and try the new card icons. The task window is the one screen the mockup doesn't show, so it's the one worth looking at hardest.
8. **Only if something unexpected happens:** a promise in the spec turns out to be impossible, something would need a paid service, the internet or a login, the build is stuck on the same problem after three honest tries, or GitHub's free monthly testing time runs out. You'll get a plain explanation and a recommended choice.

**At the end of each phase you receive**
- A checklist of every promise (gate) for that phase and earlier ones, each marked passed by an automatic test.
- Screenshots of every screen.
- The version number to install.
- Click-by-click steps to install or update on the NAS.
- The exact 👤 Owner checks to do.

The next phase starts only after you say the checks are OK. If something isn't right, tell the agent in your own words. It turns that into fixes, sends you a small update (for example v0.1.1), and asks you to check again.

**Tools that will be installed on this computer** (free, from their official makers): Go (programming language), Node.js LTS (runs the automatic browser tests), and GitHub CLI (lets the agent talk to GitHub after you sign in). Git and Docker are already installed.

> **Is this plan OK to start?**

---

## 2. How the build agent works through this plan

These are fixed rules. You (the build agent, Claude Sonnet on auto) follow them exactly.

### 2.1 Starting and ending a session
1. **Start:** read §2 of this file, then `PROGRESS.md`, then the task entry for the **first task not marked `done`**, then the SPEC sections that task names. If a task is `in-progress`, check `git status`/`git log` to see how far it got, and continue it.
2. **Work one task at a time,** in the order listed. Don't start a task whose "Depends on" tasks aren't `done`.
3. **End:** always leave `PROGRESS.md` accurate before stopping for any reason, including partial progress. Put a one-line note in the task's Notes column saying what's left.
4. Tasks are sized for one session. If a task clearly won't fit, split it in `PLAN.md` (for example `P1-19a`, `P1-19b`) and in `PROGRESS.md`, keeping the same gates. Record the split in the task's Notes.

### 2.2 PROGRESS.md format
One table row per task: `ID | Title | Status | Commit(s) | CI run | Notes`.
- Status is one of `todo`, `in-progress`, `blocked`, or `done`.
- Commit is the short SHA of the last commit of the task.
- CI run is the URL printed by `gh run list --limit 1 --json url`.
- Blocked rows say who or what they're waiting for.

Below the table there's a **Stop log** (date, stop point, message sent, owner reply) and an **Owner feedback** list (see §5.3).

### 2.3 Git workflow
- **Trunk-based on `main`, no pull requests** (nobody reviews them). One commit, or a small group of commits, per task. Commit messages start with the task ID, for example `P1-13 People: picker and cookie`, and end with the attribution line `Co-Authored-By: Claude <noreply@anthropic.com>`.
- **Push after each task.** Wait for the CI run on that push (`gh run watch <id> --exit-status`). **The next task starts only when it's green.**
- **CI flags in the commit message** (see §2.6): add `[container]` or `[speed]` when the task's "Done when" says so.
- **Releases** are annotated tags `vX.Y.Z`, taken only from a commit whose full CI run (`gh workflow run ci.yml -f full=true`) is green.
  - Internal pre-releases (never installed by the owner): `v0.0.1` (skeleton) and `v0.0.2` (Knowledge Base data, for the rollback test).
  - Phase releases: Phase 1 = `v0.1.0`, Phase 2 = `v0.2.0`, Phase 3 = `v1.0.0`.
  - Fixes after owner feedback are patch releases (`v0.1.1`, …).
- Never force-push `main`, never rewrite pushed history, and never delete tags.
- Git identity for this repo (local config only): name `stas-comp`, email `<id>+stas-comp@users.noreply.github.com`, where `<id>` comes from `gh api user --jq .id`.

### 2.4 Definition of done (every task)
1. The task's listed tests exist and pass locally (`npm run check`) **and** in CI on the pushed commit.
2. All earlier tests still pass (CI runs the whole suite; nothing is skipped).
3. "Done when" conditions are all true.
4. Any `docs/decisions.md` entries named by the task are written: one short paragraph each, headed `D-NN Title (task ID, date)`.
5. `PROGRESS.md` is updated: status `done`, commit, and CI run link.
6. If the task changed anything the owner will do or see in TrueNAS or GitHub, `README.md` is updated in the same task.

### 2.5 When CI fails or a test seems wrong
Follow SPEC B8 rule 5:
- **Never skip, weaken or delete a test, add `test.skip`/`t.Skip`, loosen an assertion, raise a time limit, or add retries to hide a failure.**
- If a test is genuinely wrong (it contradicts SPEC Part A), fix the test, write a `docs/decisions.md` entry explaining why, and list it under "Tests corrected" in the next phase report.
- **The one allowed change:** gate 1.08's "Coming soon" test is replaced when that section is built, as SPEC B7 says.

**Retry limit.** An *attempt* is a distinct, reasoned change pushed or run to fix the same failure. Re-running flaky CI without a change doesn't count, but at most one re-run is allowed per failure.
- After **3 genuine attempts** on the same failure, stop.
- Set the task to `blocked`, and write `reports/blocker-<task ID>.md` in plain language: what isn't working, what was tried, and a recommended way forward.
- Send the owner the short version (stop point S7, §5.1).

**Flaky tests are failures.** Fix the cause, for example by waiting on `data-ready` or a specific element, using `page.clock`, or making data unique. Don't add retries.

### 2.6 Commands (same in PowerShell and Bash on Windows, and in CI)
All commands run from the repository root. `package.json` scripts are the single command surface. They call only `go`, `node`/`npx` and `playwright`, with no shell-specific syntax. Environment variables needed by tests are set inside `playwright.config.ts` or Go test code, never on the command line.

| Command | What it runs | B7 layer |
|---|---|---|
| `npm run build` | `go build -trimpath -o bin/` for `./cmd/comphq` (`bin/comphq.exe` on Windows) | — |
| `npm run test:unit` | `go vet ./... && go test ./...` | 1 |
| `npm run test:integration` | `go test -tags integration ./...` (HTTP tests carry `//go:build integration`) | 2 |
| `npm run test:e2e` | `playwright test --project=e2e` (global setup builds the binary; each worker gets a temp data folder) | 3 |
| `npm run test:a11y` | `playwright test --project=a11y` | 6 |
| `npm run test:speed` | `playwright test --project=speed` (seeds the test library with `go run ./e2e/seed`) | 5 |
| `npm run test:screens` | `playwright test --project=screens` (writes `reports/screens/`) | 7 |
| `npm run test:policy` | `go test ./tools/policy/...` (repository rules, see P1-05) | — |
| `npm run check` | build, unit, integration, policy, e2e, a11y (the pre-push gate) | 1,2,3,6 |
| `npm run vendor:editor` | `node scripts/vendor-editor/build.mjs` (dev only, rewrites `web/static/vendor/editor/`) | — |
| `npm run fixtures:docx` | `go run ./e2e/fixtures/docx/gen` (dev only, rewrites committed `.docx` fixtures) | — |

- **CI only:** layer 4 container tests: `bash deploy/test/container-test.sh`. This needs Linux Docker. Run it locally only if `docker info` succeeds from the Bash tool; don't start Docker Desktop yourself.
- **Speed runs** locally for information. The CI result is the one that counts.
- Install Playwright browsers once per machine with `npx playwright install chromium`.

**What CI runs** (`.github/workflows/ci.yml`, ubuntu-24.04, standard runner). The split exists because a private repo on GitHub Free gets **2,000 Linux minutes/month** on 2-vCPU runners (decision D-03):

| Job | Runs on every push to `main` | Also runs when |
|---|---|---|
| `go`: unit, integration, policy | yes | — |
| `browser`: e2e + a11y | yes | — |
| `container`: layer 4 | only if the push changes `Dockerfile`, `deploy/`, `cmd/`, `internal/db/`, `migrations/`, or the commit message contains `[container]` | `workflow_dispatch` with `full=true`; every tag |
| `speed`: layer 5 | only if the commit message contains `[speed]` | `workflow_dispatch` with `full=true`; every tag |
| `screens`: layer 7 | no | `workflow_dispatch` with `full=true`; every tag (uploaded as artifact `screens`) |

Other CI settings:
- Workflow-level `concurrency: {group: ci-${{ github.ref }}, cancel-in-progress: true}`.
- Cache Go modules and the Playwright browser folder.
- `release.yml` runs on tags `v*.*.*`. It re-runs the full suite, builds with `-ldflags "-X main.version=X.Y.Z"`, and pushes `ghcr.io/stas-comp/comphq:X.Y.Z`. It never pushes `latest`.

**Save CI minutes:** run `npm run check` locally before every push, push once per task where possible, and don't push work in progress.

### 2.7 Stop points
The agent pauses **only** at the stop points in §5.1. Everything else, you decide yourself following SPEC B8 rules 1–4, recording non-obvious choices in `docs/decisions.md`. Never ask the owner a technical question. Never enter a password or token.

### 2.8 Standing engineering rules (apply to every task)
- **SPEC B2 boundaries.** A section package never imports another section's internals. Each section registers routes, nav entry and migrations with `internal/app`.
- **Expand-only migrations from the first migration** (SPEC B6). Migration files live under `migrations/<section>/NNNN_name.sql` and are never edited after they're pushed.
  - The migration runner ignores rows for migrations it doesn't know (a newer app ran them).
  - A policy test fails if a pushed migration file changes, or if a migration contains `DROP`, `RENAME`, or `ALTER COLUMN`.
- **E2E tests must pass against a shared, already-populated server** (needed by the offline and upgrade runs in layer 4):
  - Each test creates its own uniquely named data, for example a name ending in a random suffix.
  - Tests never assume an empty database. Assert relative order and presence, not totals.
  - Tests that truly need a fresh database are tagged `@fresh`, run only in local/CI e2e with their own server, and are listed in `docs/decisions.md`.
- **Every page sets `data-ready` on `<body>`** when rendered and its scripts have initialised. Tests wait on it and never on fixed sleeps.
- **The time a page shows** comes from the server's clock (`TZ`). Tests control time only with `COMPHQ_TEST_MODE=1` plus `COMPHQ_TEST_TODAY`, fixture timestamps, or Playwright `page.clock` for browser timers.
- **Every drag has a button equivalent,** and E2E tests cover both.
- **User-facing wording lives in templates** (and in `web/static/<section>/messages.js` for script messages), never in Go code.
- **Every page type gets:** an axe test (layer 6), a no-sideways-scroll test at 1024 × 700 and 1920 × 1080, and a screenshot entry (layer 7), **in the same task that creates the page**.
- **Dependencies:** at most 5 direct Go modules. Planned: `modernc.org/sqlite`, `github.com/microcosm-cc/bluemonday`. Anything new needs a decision entry.

---

## 3. Versions and sources (checked 14 September 2026)

Pin exactly these. If a task finds one no longer installable, pick the nearest patch release, and record it in `docs/decisions.md` and here.

| Item | Pin | Source checked | Notes |
|---|---|---|---|
| Go | `go 1.27` + `toolchain go1.27.1` in `go.mod` | go.dev/dl JSON | winget offers 1.27.0. The `toolchain` line downloads 1.27.1 automatically at build time. CI uses `actions/setup-go` with `go-version-file: go.mod`. |
| Node.js | 24 LTS "Krypton", CI `node-version: 24.21.0`; `"engines": {"node": ">=24 <25"}` | nodejs.org/dist/index.json | winget `OpenJS.NodeJS.LTS` offers 24.19.0, which is fine locally. |
| GitHub CLI | winget `GitHub.cli` 2.100.0 | winget | Dev machine only. |
| Git | 2.37.1 already installed | local | Works. Upgrading isn't required. |
| Docker (local) | Docker Desktop 29.4.3 installed, **engine not running**; WSL2 (Ubuntu) present | local | **Container tests run only in GitHub Actions.** |
| `modernc.org/sqlite` | `v1.58.0` (bundles SQLite 3.53.4) | proxy.golang.org, pkg.go.dev, gitlab.com/cznic/sqlite source | **FTS5 confirmed:** `lib/sqlite_linux_amd64.go` is generated with `-DSQLITE_ENABLE_FTS5`. Driver name `"sqlite"`, pragmas via DSN `_pragma=`. windows/amd64 and linux/amd64 supported. P1-08 proves `snippet()`/`highlight()`/`bm25()` on both OSes. |
| `github.com/microcosm-cc/bluemonday` | `v1.0.27` | proxy.golang.org | Stable; last release Jul 2024. |
| TipTap | `@tiptap/core`, `@tiptap/pm`, `@tiptap/starter-kit`, `@tiptap/extension-table`, `@tiptap/extension-image`, `@tiptap/extension-file-handler`, all `3.31.3`, **MIT** | registry.npmjs.org | v3 has tables in `@tiptap/extension-table` (row, cell and header included). FileHandler (paste/drop hook) is MIT in v3. Link is in StarterKit v3. |
| esbuild | `0.28.2` (MIT), dev only | registry.npmjs.org | Builds the one-time editor bundle. |
| SortableJS | `1.15.7` (MIT) | registry.npmjs.org | Vendored single file for board/lane dragging (D-06). |
| Playwright | `@playwright/test 1.63.0` (Apache-2.0) | registry.npmjs.org | In CI, the offline run uses `mcr.microsoft.com/playwright:v1.63.0-noble`. P1-12 resolves and pins its digest. |
| `@axe-core/playwright` | `4.13.0` (MPL-2.0, dev only) | registry.npmjs.org | Dev only, never shipped. |
| Base image | `gcr.io/distroless/static-debian13:nonroot@sha256:e2e927ec666bae08560abb3c55d0659eceabb657f56b6782ab500a9fc7f555e3` | `docker buildx imagetools inspect` | Index digest. SPEC said `distroless/static`; debian13 is the current variant (D-02). |
| Build image | `golang:1.27.1-bookworm` (or `-trixie`), pinned by digest | resolve in P1-06 | Record the digest in `docs/decisions.md`. |
| `actions/checkout` | `v7.0.1` → `3d3c42e5aac5ba805825da76410c181273ba90b1` | `git ls-remote` | |
| `actions/setup-go` | `v7.0.0` → `b7ad1dad31e06c5925ef5d2fc7ad053ef454303e` | `git ls-remote` | |
| `actions/setup-node` | `v7.0.0` → `820762786026740c76f36085b0efc47a31fe5020` | `git ls-remote` | |
| `actions/cache` | `v6.1.0` → `55cc8345863c7cc4c66a329aec7e433d2d1c52a9` | `git ls-remote` | |
| `actions/upload-artifact` | `v7.0.1` → `043fb46d1a93c77aae656e7c1c64a875d1fc6a0a` | `git ls-remote` | |
| `docker/setup-buildx-action` | `v4.3.0` → `37fe631027851001ddb9b187196cc803df7f5f0e` | `git ls-remote` | |
| `docker/login-action` | `v4.6.0` → `dbcb813823bdd20940b903addbd779551569679f` | `git ls-remote` | ghcr login with `GITHUB_TOKEN`, `permissions: packages: write` in `release.yml` only. |
| `docker/build-push-action` | `v7.3.0` → `53b7df96c91f9c12dcc8a07bcb9ccacbed38856a` | `git ls-remote` | `platforms: linux/amd64`. |
| GitHub Actions allowance | GitHub Free, private repo: 2,000 Linux minutes/month; standard private runners are 2 vCPU / 8 GB | GitHub pricing pages, 2026 | Drives D-03. When the allowance is used up, jobs don't start (no charge without a payment method). That's stop point S8. |
| ghcr visibility | Package visibility is set separately from the repository; a public package doesn't expose a private repository | docs.github.com, Container registry | Supports stop point S2. |
| TrueNAS SCALE 25.04 "Install via YAML" | Plain Docker Compose. Host-path bind mounts, `user:`, `healthcheck` (shown as healthy in the Apps UI), `read_only`, `tmpfs`, `restart` all honoured. Default apps UID/GID 568. | truenas.com docs (25.04 Apps reference), apps.truenas.com "Installing Custom Apps", TrueNAS forums (Oct 2025) | Matches SPEC B6. Keep to standard Compose keys only. |

Pin the SHA-256 of every vendored file in `web/static/vendor/VENDOR.md`: name, version, source (npm package), licence, and checksum. The policy test recomputes them.

---

## 4. Milestones and tasks

Each task lists **Gates** (SPEC A11), **Depends on**, **Touches** (SPEC B2 paths), **Tests** (B7 layer), **Done when**, and any **Decisions** to record. "Standard page checks" means: an axe test (layer 6), a no-sideways-scroll test at 1024 × 700 and 1920 × 1080, and a screenshot entry, all for every new page type (rule 2.8).

# Phase 1 — Frame, names, Knowledge Base, installing on the NAS (→ v0.1.0)

## Milestone 1.0 — Tooling and walking skeleton

### P1-01 Local tooling
- **Goal:** confirm or install the build tools on the Windows build computer.
- **Gates:** none (plumbing).
- **Depends on:** —
- **Touches:** machine only; creates `PROGRESS.md` entries.
- **Steps:**
  - `winget install --id GoLang.Go --exact --accept-source-agreements --accept-package-agreements`, and the same for `OpenJS.NodeJS.LTS` and `GitHub.cli`. Skip any that `Get-Command` already finds.
  - Open a new shell so PATH updates, then record versions.
  - Run `docker info` and record whether the engine runs. Don't start Docker Desktop.
- **Tests:** none.
- **Done when:** `go version`, `node --version`, `npm --version`, `git --version` and `gh --version` all succeed from both PowerShell and Bash. The Notes column records the versions and "docker engine: running/not running".

### P1-02 GitHub sign-in and repository — **STOP S1**
- **Goal:** get an authenticated `gh`, a private repository, and a local git repo on `main`.
- **Gates:** none (plumbing).
- **Depends on:** P1-01
- **Touches:** `.gitignore`, `.gitattributes`, `README.md` (stub), `docs/decisions.md` (header), `PLAN.md`, `PROGRESS.md`, `SPEC.md`, `docs/intent/`, the two HANDOFF files (committed as history).
- **Steps:**
  1. If `gh auth status` fails, send message **S1** (§5.1) and wait for the owner's "done".
  2. Confirm `gh auth status` shows account `stas-comp` with the `repo` and `workflow` scopes.
  3. `git init -b main`, set the local git identity (rule 2.3), and run `gh auth setup-git`.
  4. `gh repo create stas-comp/comphq --private --source . --remote origin`.
  5. `.gitignore` covers `bin/`, `node_modules/`, `test-results/`, `playwright-report/`, `reports/screens/`, `*.db*` outside fixtures, and `/data/`. **Commit `samples/word/`** (the owner was told to put nothing private there).
  6. `.gitattributes`: `* text=auto eol=lf`; `*.docx *.png *.jpg *.gif *.webp *.woff2 *.db binary`.
- **Tests:** none.
- **Done when:** `git push -u origin main` succeeds, `gh repo view stas-comp/comphq --json visibility` shows `PRIVATE`, and the Stop log records S1.
- **Decisions:** D-01 (name and identifiers).

### P1-03 Go skeleton: binary, config, health, migrations runner
- **Goal:** a Go binary that opens SQLite, runs section migrations, serves one frame page with `data-ready`, and answers `healthcheck`.
- **Gates:** none directly (foundation for 1.38–1.42).
- **Depends on:** P1-02
- **Touches:** `go.mod`/`go.sum`, `cmd/comphq/`, `internal/app/` (router, section registry, error/recover middleware, embedded templates and static files), `internal/db/` (open with WAL, `foreign_keys=ON`, `busy_timeout=5000`; migration runner; `app_meta`; `schema_migrations(section, version, applied_at, app_version)`), `web/templates/app/`, `migrations/app/0001_meta.sql`.
- **Behaviour:**
  - Config from env: `COMPHQ_DATA_DIR` (default `/data`), `COMPHQ_ADDR` (default `:8080`), `TZ`. Embed `time/tzdata`.
  - `comphq healthcheck` does a GET to `http://127.0.0.1:<port>/healthz` with a 3s timeout, and exits 0 or 1.
  - `/healthz` returns 200 only when the DB answers `SELECT 1`.
  - `main.version` is set by ldflags, defaulting to `dev`.
- **Tests:**
  - Unit: migration runner (applies in order, one transaction each, ignores unknown newer rows, refuses to start on a failed migration with a clear log line), config defaults.
  - Integration: `/healthz` 200; `healthcheck` subcommand against a live test server.
- **Done when:** `npm run test:unit` and `npm run test:integration` pass (this task also adds a minimal `package.json` with those two scripts), and `go run ./cmd/comphq` with `COMPHQ_DATA_DIR=<temp>` serves `/` with `<body data-ready>`.

### P1-04 Playwright and axe skeleton
- **Goal:** the browser test harness, used by every later task.
- **Gates:** none directly (the harness for 1.09 and 1.10).
- **Depends on:** P1-03
- **Touches:** `package.json` (all §2.6 scripts; exact devDependency versions from §3), `package-lock.json`, `playwright.config.ts` (projects `e2e`, `a11y`, `speed`, `screens`, `smoke`; global setup builds the binary; a per-worker server fixture with its own temp data dir and a free port; `BASE_URL` mode skips starting a server and filters out `@fresh`), `e2e/helpers/` (`ready(page)`, `expectNoSideScroll(page)` at both sizes, `axeCheck(page)` failing on serious/critical, `uniqueName()`), `e2e/frame/skeleton.spec.ts`, `e2e/a11y/pages.spec.ts` (page list registry), `e2e/screens/pages.spec.ts`.
- **Tests:**
  - E2E: frame page loads and is ready.
  - Layer 6: axe on the frame page.
  - Layer 7: a screenshot at 1366 × 768.
- **Done when:** `npm run check` passes locally from both PowerShell and Bash.

### P1-05 CI workflow and repository policy tests
- **Goal:** every push runs layers 1, 2, 3 and 6, and the repository rules are enforced automatically.
- **Gates:** none (plumbing; supports SPEC B7's "also asserted in CI").
- **Depends on:** P1-04
- **Touches:** `.github/workflows/ci.yml` (jobs per §2.6; `container`, `speed` and `screens` defined but conditional; actions pinned to the §3 SHAs; `permissions: contents: read`), `tools/policy/` (Go tests).
- **Policy tests** (each fails with a clear message):
  - every `uses:` is pinned to a 40-hex SHA;
  - `Dockerfile` `FROM` lines are pinned by digest;
  - `deploy/truenas.yaml` has no `COMPHQ_TEST_MODE`, and its image tag is an exact `X.Y.Z` (never `latest`);
  - no `http://` or `https://` URLs in `web/templates/**` or `web/static/**` (except `web/static/vendor/**` licence headers, and the setup-page template's placeholder text);
  - `web/static/vendor/VENDOR.md` checksums match the files;
  - migration files are unchanged once committed (compare against `git show HEAD~:path` where the file existed), and contain no `DROP`, `RENAME` or `ALTER COLUMN`;
  - `go.mod` has at most 5 direct requirements.
- **Tests:** layer-1-style Go tests in `tools/policy`.
- **Done when:** a push shows `go` and `browser` jobs green in `gh run watch`, and a deliberate local break (e.g. an unpinned action in a scratch branch, never pushed) makes `npm run test:policy` fail.
- **Decisions:** D-03 (CI tiers and minutes), D-04 (command surface and integration build tag).

### P1-06 Dockerfile, TrueNAS YAML, container tests, first image — tag v0.0.1
- **Goal:** prove the deployment path end to end, from source through the image, ghcr and Compose, to a healthy app.
- **Gates:** 1.38, 1.39, 1.40 (first proof; re-proved at every release).
- **Depends on:** P1-05
- **Touches:**
  - `Dockerfile`: multi-stage; build stage is the digest-pinned Go image with `CGO_ENABLED=0`, `-trimpath`, and ldflags version; final stage is the §3 distroless digest; `COPY` the binary to `/comphq`; `ENTRYPOINT ["/comphq"]`; `EXPOSE 8080`.
  - `deploy/truenas.yaml` (SPEC B6):
    - service `comphq`, `image: ghcr.io/stas-comp/comphq:0.0.1`, `restart: unless-stopped`, `user: "568:568"`, `ports: ["8080:8080"]`;
    - volume `/mnt/YOUR-POOL/comphq:/data`, environment `TZ: "Europe/London"`;
    - `healthcheck` `["CMD","/comphq","healthcheck"]`, interval 10s, timeout 3s, retries 3, start_period 15s;
    - `read_only: true`, `tmpfs: [/tmp]`;
    - the three owner-editable lines marked with `# ← CHANGE THIS:` plain-English comments.
  - `deploy/test/container-test.sh`: substitutes a temp folder (chown 568) and `TZ=Europe/London`, builds the image locally, and tags it as the YAML's image reference. It then runs these sub-tests:
    1. `docker compose config` validates;
    2. `up -d` becomes healthy within 60s;
    3. **restart:** create a person via HTTP and checksum the data folder; `docker compose restart` and `down` + `up`; healthy ≤ 60s each time; data checksum and person still present;
    4. **recreate:** `down`, `docker rm` and `docker image rm`, re-tag and `up` with the same folder; data intact.
  - `.github/workflows/ci.yml` `container` job; `.github/workflows/release.yml` (on tag: full suite, then build and push `ghcr.io/stas-comp/comphq:${tag#v}`, `permissions: packages: write, contents: read`).
  - `README.md` stub section "Installing on TrueNAS" (filled in P1-16).
- **Tests:** layer 4 as above; policy tests now cover the real files.
- **Done when:**
  - A push with `[container]` has a green `container` job.
  - Tag `v0.0.1` (from green `main`): `release.yml` is green, and `gh api /users/stas-comp/packages/container/comphq/versions` lists `0.0.1`.
  - `CHANGES.md` has a `0.0.1 (internal)` entry.
- **Decisions:** D-02 (base image variant and digests), D-08 (internal pre-release tags).

### P1-07 Make the image public — **STOP S2**
- **Goal:** TrueNAS (and the offline and upgrade tests) can pull images anonymously.
- **Gates:** none (plumbing for O1.1).
- **Depends on:** P1-06
- **Steps:** send message **S2** (§5.1) and wait for "done". Then verify anonymously, with no auth header: `curl -s "https://ghcr.io/token?scope=repository:stas-comp/comphq:pull&service=ghcr.io"` returns a token, and a `HEAD` to `https://ghcr.io/v2/stas-comp/comphq/manifests/0.0.1` using it returns 200.
- **Done when:** anonymous manifest `HEAD` returns 200, and the Stop log records S2.

## Milestone 1.1 — Risk spikes (each ends in a decision entry and committed tests)

### P1-08 Spike: FTS5 search with block ids on modernc
- **Goal:** prove ranked, stemmed, prefix search with per-block snippets and highlights works in `modernc.org/sqlite` on Windows and Linux, before any Knowledge Base code depends on it.
- **Gates:** foundation for 1.26–1.32.
- **Depends on:** P1-05
- **Touches:** `internal/kb/search/` (query builder: letter/digit runs only, each quoted, last word prefixed `*`, fewer than 2 characters → no query; SQL helpers for `bm25(kb_search, 10.0, 1.0)`, best block per article, `snippet()` and `highlight()` with private-use markers converted to escaped HTML with `<mark>`), `migrations/kb/0001_search.sql` (FTS5 `tokenize='porter unicode61'`).
- **Tests (unit, in-memory DB):**
  - "printers" matches "printer"; "ton" matches "toner";
  - a title hit outranks a body hit;
  - the snippet contains `<mark>toner</mark>` and escapes `<script>` text;
  - symbol fuzz inputs `" * ( - : NEAR AND OR ^ '` and empty or 1-character input never error;
  - deleting an article's rows removes it from results.
- **Done when:** tests are green locally (Windows) and in CI (Linux).
- **Decisions:** D-10 (the exact FTS5 query shape, marker characters, and any modernc caveat found).

### P1-09 Spike: editor bundle
- **Goal:** build the one-time TipTap bundle, and prove it runs under the app's CSP with every toolbar feature and a paste/drop upload hook.
- **Gates:** foundation for 1.15–1.18, 1.45.
- **Depends on:** P1-04
- **Touches:**
  - `scripts/vendor-editor/build.mjs` and `entry.js`: import the §3 TipTap packages; configure StarterKit with headings `[2, 3]` only and link protocols `http`, `https`, `mailto`; Table with `resizable: false`; Image; FileHandler with `onPaste`/`onDrop` callbacks exposed to app code. Export one global `ComphqEditor.create(el, options)`, set `injectCSS: false`, and bundle with esbuild as an IIFE, minified, no sourcemap.
  - The output goes to `web/static/vendor/editor/editor-3.31.3.js`, plus `web/static/vendor/editor/LICENSES.txt`; add an entry to `web/static/vendor/VENDOR.md` (versions and SHA-256).
  - `web/static/kb/editor.css` holds the ProseMirror base styles (no injected CSS).
  - A test-mode-only harness route `GET /__test/editor`, registered only when `COMPHQ_TEST_MODE=1`, served with the production CSP.
- **Tests (E2E against the harness):**
  - each toolbar command produces its element (h2, h3, strong, em, ul, ol, a, table with add/remove row and column, img);
  - a FileHandler paste callback fires for a synthetic image paste;
  - **no CSP violations** are reported (listen for `securitypolicyviolation` and console errors).
- **Done when:**
  - `npm run vendor:editor` reproduces a byte-identical file (CI checks this by re-running it and `git diff --exit-code`);
  - E2E tests are green in CI;
  - the policy checksum test passes.
- **Decisions:**
  - D-05 (editor packages, bundling, `injectCSS: false`, no column resizing).
  - D-11, only if the CSP had to be relaxed (e.g. `style-src 'self' 'unsafe-inline'` for ProseMirror style attributes). Prefer not relaxing it. If you do, record exactly why.

### P1-10 Spike: clipboard paste in headless Chromium (CI)
- **Goal:** choose a reliable way to test pasting screenshots and HTML in CI.
- **Gates:** foundation for 1.16 and 1.18.
- **Depends on:** P1-09
- **Touches:** `e2e/helpers/paste.ts`, `e2e/fixtures/paste/web-page.html`, `e2e/fixtures/paste/word.html` (Word's clipboard HTML with `mso-` styles, `<font>`, classes, and `<img src="file:///C:/Users/x/AppData/Local/Temp/msohtmlclip1/01/clip_image001.png">`), `e2e/fixtures/images/` (small PNG/JPEG/GIF/WebP, a 21 MB PNG generated at test time rather than committed, and a `.txt`).
- **Approach:** try (a) `context.grantPermissions(['clipboard-read','clipboard-write'])` with `navigator.clipboard.write`, and (b) dispatching a synthetic `ClipboardEvent('paste')` with a `DataTransfer` carrying `text/html` or a `File`. Pick the one that is green 10/10 locally and in CI (run the spec with `--repeat-each=10` once, only in this task). Drop events use `DataTransfer` the same way.
- **Tests:** E2E on the harness: pasted PNG reaches the FileHandler callback within 3s; pasted HTML reaches the editor.
- **Done when:** the chosen helper is committed and green in CI with repeat-each 10 (then reset to 1).
- **Decisions:** D-12 (paste/drop testing method).
- **Also:** send the optional **N1** reminder message about `samples\word` (§5.1). Don't wait for a reply.

### P1-11 Spike: Word converter on realistic markup, and the fixture generator
- **Goal:** prove the pure `.docx` converter can read realistic Word and Google Docs packages, and settle how fixtures are made.
- **Gates:** foundation for 1.45–1.52.
- **Depends on:** P1-05
- **Touches:**
  - `e2e/fixtures/docx/gen/` (Go program, dev only, stdlib `archive/zip` and `text/template`). It writes WordprocessingML exactly as Word 365 and Google Docs structure it: `[Content_Types].xml`, `_rels/.rels`, `word/document.xml`, `styles.xml` with localised style IDs (e.g. `berschrift1` named `heading 1`), `numbering.xml`, `word/_rels`, `media/`, `docProps`. Outputs:
    - `sample.docx`: title, H1–H3, bold/italic incl. `w:val="0"`, nested bullets and numbers, hyperlink and HYPERLINK field, table with gridSpan and vMerge and header row, 3 PNG/JPEG images with alt text, a chart, SmartArt, a shape, an equation, an EMF picture, a text box, tracked insert and delete, a comment, a footnote, header and footer;
    - `googledocs.docx`: Google Docs export style;
    - `big-20pages.docx`: 20 pages, 10 pictures;
    - `bad/`: `oldword.doc` (minimal CFB header), `protected.docx` (CFB, as Word writes encrypted files), `file.pdf`, `truncated.docx`, `zipbomb.docx` (entry claiming small, inflating past limit), `deep.docx` (XML depth 300).
    - Outputs are committed; `npm run fixtures:docx` reproduces them byte-identically (fixed zip timestamps).
  - `internal/kb/docx/`: package skeleton with `Convert(r io.ReaderAt, size int64, filename string) (Result, error)`, content-type and `_rels` discovery, style resolution (`basedOn`, name matching), paragraphs to `p`/`h2`/`h3`, the title rule, and the `ErrUnreadable` / `ErrTooLarge` error kinds.
- **Tests (unit):** `sample.docx` and `googledocs.docx` yield the expected title and headings; each `bad/` fixture returns `ErrUnreadable` (or the limit error) without panic, within 2s and under 64 MB allocation (`testing.AllocsPerRun` is not needed; use a size-limited reader test).
- **Done when:** tests are green in CI; the generator reproduces committed files (CI: re-run and `git diff --exit-code`).
- **Decisions:** D-07 (fixture generator instead of LibreOffice; how realism is checked: owner samples plus O1.4).

### P1-12 Spike: offline test and upgrade/rollback harness in Compose
- **Goal:** complete layer 4 with the offline run (1.41) and the previous-release upgrade and rollback run (1.42), so both are green from now on.
- **Gates:** 1.41, 1.42 (harness; content grows with each release).
- **Depends on:** P1-07
- **Touches:** `deploy/test/container-test.sh` (new sub-tests), `deploy/test/offline.override.yaml` (the app's network `internal: true`; a `runner` service using the digest-pinned Playwright image, attached only to that network, running `npx playwright test --project=smoke` with `BASE_URL=http://comphq:8080`), `e2e/smoke/` (tests tagged `@seed` and `@verify`).
- **Offline sub-test:**
  - pre-pull images, then `up` with the override;
  - assert that the app container can't reach the internet (`/comphq healthcheck` still works, and a test-mode-only `GET /__test/egress` that tries `https://example.com` reports failure; this route is registered only when `COMPHQ_TEST_MODE=1` and is set only by the override);
  - the smoke suite passes. From P1-38 the full Phase 1 E2E suite runs here.
- **Upgrade/rollback sub-test:**
  - `PREV` = the highest `vX.Y.Z` tag lower than the version in `deploy/truenas.yaml` (or the latest tag if HEAD is untagged). Check out `PREV` into a git worktree.
  - Run: `PREV` image with `PREV`'s smoke `@seed @verify`; current image with the current smoke `@verify-prev` (reads `PREV`-seeded data) then `@seed @verify`; `PREV` image again with `PREV`'s smoke `@verify`.
  - Each phase must be healthy within 60s. Assert that a pre-update backup file appears in `/data/backups/pre-update/` when migrations ran.
- **Tests:** layer 4.
- **Done when:** a `[container]` push is green with all four container sub-tests (compose/healthy, restart, recreate, offline, upgrade-rollback against `v0.0.1`).
- **Decisions:** D-13 (offline method, upgrade/rollback method, smoke tag scheme).
- **Rule from here on:** the upgrade/rollback sub-test is mandatory and never removed. It becomes meaningful with `v0.0.2` (P1-28). **Gate 1.42 is reported passed only from the `v0.1.0` release run** (upgrade from `v0.0.2`, with articles, images and history, then roll back).

## Milestone 1.2 — Names and the frame

### P1-13 People: picker, add name, cookie
- **Goal:** the first-visit name picker, "My name isn't here", "Change", and person middleware.
- **Gates:** 1.01, 1.02, 1.03, 1.04, 1.06.
- **Depends on:** P1-04
- **Touches:** `internal/people/` (store, handlers, middleware), `migrations/people/0001_people.sql`, `web/templates/people/`, `web/static/people/`.
- **Behaviour** (SPEC B4 person identity):
  - cookie `comphq_person`;
  - a GET without a valid cookie redirects to `/who?next=`;
  - a POST shows the picker with a message;
  - names are unique among active people (case-insensitive), with a plain duplicate message.
  - Also add the **origin-check middleware** here (SPEC B4 request safety) in `internal/app`: a POST whose `Origin` (or `Referer`) host doesn't match `Host` gets a 403 with a plain page.
- **Tests:**
  - Integration: middleware redirects, cookie attributes (`SameSite=Lax`, `HttpOnly`, `Max-Age` 400 days, refreshed), origin check 403 cases.
  - E2E: 1.01 (alphabetical buttons), 1.02 (new page in the same context goes straight in), 1.03, 1.04 (second browser context sees the new name), 1.06 (deactivate the person via store helper, and the next page shows the picker).
  - Keyboard-only E2E for the picker.
  - Standard page checks for `/who`.
- **Done when:** tests are green in CI.

### P1-14 Frame, theme, section registry, Coming soon, 404, headers
- **Goal:** the full shared frame and the bold theme, so every later page drops into it.
- **Gates:** 1.07, 1.08, 1.09, 1.10, 1.11.
- **Depends on:** P1-13
- **Touches:**
  - `internal/app` (layout template with dark sidebar, wordmark "Comp HQ", nav Briefing, Knowledge Base, Tasks, Calendar, Settings; top bar with search box placeholder and "You: *name* · Change"; the current section marked with `aria-current="page"` plus the accent style; security headers per SPEC B4, CSP including the P1-09 outcome; 404 page with a link to `/kb`);
  - `web/static/theme/theme.css` (all tokens: ink-navy frame, signal-orange accent, paper background, type scale, radii, spacing), `web/static/theme/fonts/`;
  - two OFL font families (suggested: *Archivo* for headings, *Atkinson Hyperlegible Next* for body; confirm the OFL licence and add them to `VENDOR.md`), `web/static/theme/icons/*.svg`;
  - `web/manifest.webmanifest` and PNG icons (name "Comp HQ");
  - placeholder sections `internal/briefing`, `internal/tasks` and `internal/calendar`, each registering a "Coming soon" page only.
  - `/` redirects to `/kb` in Phase 1 (to the Briefing from P3-02).
  - Contrast: accent text on navy and paper must reach WCAG AA. If signal orange fails for text, use it for fills and borders, with dark or white text.
- **Tests:**
  - E2E: 1.07, frame elements on every registered route (route list from a test-mode `GET /__test/routes`); 1.08, each placeholder shows its "Coming soon" heading with status 200; 1.11, unknown URL shows "Page not found" inside the frame with a working link.
  - Standard page checks on all current pages.
  - Integration: headers present on HTML responses.
- **Done when:** tests are green in CI.
- **Decisions:** D-09 (fonts, colour tokens, contrast adjustments).

### P1-15 Settings shell: People, About, Set up this computer
- **Goal:** Settings with People management, version display, and shortcut instructions.
- **Gates:** 1.05 (picker and assignment-list part; history part completed in P1-24), 1.37, 1.43.
- **Depends on:** P1-14
- **Touches:** `internal/settings/` (routes `/settings`, `/settings/people`, `/settings/about`, `/settings/setup`), `web/templates/settings/`, `web/static/settings/copy.js`.
- **Behaviour:**
  - Rename and remove (set `active=0`). History and author names always resolve through `people.id`, so renames show everywhere and removed people still show.
  - The Setup page builds the SPEC B6 Chrome and Brave commands from the request's `Host`. It has Copy buttons (Clipboard API with a select-text fallback) and the click-by-click steps.
  - About shows `main.version`.
- **Tests:**
  - Integration: setup text for `Host: 192.168.1.50:8080` matches exactly.
  - E2E:
    - rename shows in the picker and top bar;
    - a removed person leaves the picker;
    - About shows the version (test build uses `-X main.version=0.0.0-test`);
    - Copy puts the exact text on the clipboard;
    - 1.43: `chromium.launchPersistentContext(tmp, {args: ['--app=' + url]})` opens the app as the first page and it becomes ready.
  - Standard page checks.
- **Done when:** tests are green in CI.

### P1-16 README part 1
- **Goal:** the owner guide covers opening Comp HQ, setting up a computer, installing on TrueNAS, updating, and undoing an update, with click-by-click steps.
- **Gates:** 1.44 (partial; completed in P1-37).
- **Depends on:** P1-15, P1-12
- **Touches:** `README.md`, `internal/app/readme_test.go` (or `tools/policy`), which asserts required headings.
- **Content** (plain language, numbered steps, TrueNAS 25.04 menu names):
  - **Opening Comp HQ**
  - **Setting up an office computer**: shortcut via the Setup page, and finding the browser path.
  - **Installing on TrueNAS**:
    - confirm a fixed IP (Network → Interfaces);
    - create dataset `comphq` with the **Apps** preset (Datasets → Add Dataset);
    - set up a periodic snapshot task, daily and kept 30 days (Data Protection → Periodic Snapshot Tasks);
    - Apps → Discover Apps → ⋮ → Install via YAML, then paste `deploy/truenas.yaml` and change the marked lines (pool path, time zone from a short list with how to find yours, port if in use);
    - check it shows healthy; open `http://<NAS IP>:8080`.
  - **Updating**: Apps → comphq → Edit → change the version number → Save.
  - **Undoing an update**
  - Placeholders for the part-2 headings.
- **Tests:** a README heading test (currently checks the part 1 headings).
- **Done when:** the test is green in CI.

## Milestone 1.3 — Knowledge Base core

### P1-17 Categories and Knowledge Base home
- **Goal:** manage categories, and show the KB home with tiles and "Recently updated".
- **Gates:** 1.12, 1.13 (the recent list fills once articles exist in P1-19).
- **Depends on:** P1-14
- **Touches:** `internal/kb/` (categories store and handlers), `migrations/kb/0002_categories.sql`, `web/templates/kb/home.html`, `web/templates/kb/categories.html`, `web/static/kb/`.
- **Behaviour:**
  - Create, rename, and reorder with Move up and Move down buttons (no drag needed here).
  - Delete is refused if the category has any article, published or archived, with a plain explanation ("This category still has 3 articles. Move or archive them first." Archived articles still count, because they can be restored).
  - The home page shows tiles in `sort_order` with published counts, plus the 10 most recently updated published articles with name and date/time in A3 format.
- **Tests:**
  - Integration: delete refusal, reorder renumbering.
  - E2E: 1.12 and 1.13, with the recent list asserted after P1-19 adds articles (a test marked `// completed in P1-19` is not allowed, so write the recent-list E2E in P1-19).
  - Standard page checks.
  - Unit: date formatting helper `Sat 19 Sep 2026` / `14:30` in `internal/app/format`.
- **Done when:** tests are green in CI.

### P1-18 Sanitiser and block ids
- **Goal:** the single server-side HTML policy and the block-id and plain-text pass that search and scroll-to rely on.
- **Gates:** foundation for 1.15, 1.18, 1.19, 1.28, 1.46.
- **Depends on:** P1-08
- **Touches:** `internal/kb/html/` (bluemonday policy, `Sanitize`, `AssignBlocks`, `BlockTexts`).
- **Policy:**
  - Allowed tags: `p h2 h3 strong em ul ol li a table thead tbody tr th td br img figure figcaption blockquote div`.
  - `a[href]` is limited to http, https and mailto, with `rel="noopener"` added.
  - `td/th[colspan,rowspan]`; `img[src,alt]` where `src` must start with `/images/`.
  - `div[data-missing-kind]` and `[data-missing-src]` placeholders.
  - No `style`, `class`, `font` or `id`. `h1` is mapped to `h2`, and `h4`–`h6` to `h3`, before sanitising.
- **Tests (unit):**
  - Table-driven fixtures: scripts and handlers removed; styles, classes and fonts stripped; external `img` dropped; placeholders kept.
  - `data-b` is numbered in document order across `p h2 h3 li blockquote tr figcaption`.
  - Block texts extracted; nested `li` inside `li` gets its own id.
  - The function is idempotent (re-sanitising gives the same output).
- **Done when:** tests are green in CI.

### P1-19 Article store, publish, read, versions, search rows
- **Goal:** publishing an article saves the sanitised body, a version row, and search rows in one transaction, and the article page renders it.
- **Gates:** 1.14, 1.22 (listing), 1.30 (indexing part), 1.13 (recent list).
- **Depends on:** P1-17, P1-18
- **Touches:** `internal/kb/` (articles store: `Publish(ctx, input, person) (Article, error)` in one transaction covering sanitise, block ids, `kb_articles` upsert, `kb_article_versions` insert, delete and insert `kb_search` rows for that article; handlers `GET /kb/articles/{id}`, `GET /kb/new`, `GET /kb/articles/{id}/edit`, `POST /kb/articles`), `migrations/kb/0003_articles.sql`, `web/templates/kb/article.html`, `web/templates/kb/category.html`, a minimal editor form (a plain contenteditable stub is fine until P1-20).
- **Tests:**
  - Integration: publish creates version 1; an edit creates version 2 and replaces the search rows; the transaction rolls back fully if the search insert fails (inject failure); a removed word no longer matches, and a new word matches.
  - E2E: 1.14 (appears in category straight away); 1.13 recent list with name and time.
  - Standard page checks for the article and category pages.
- **Done when:** tests are green in CI.

### P1-20 Editor page: toolbar, publish/cancel, leave warning
- **Goal:** the real editor built on the P1-09 bundle, looking like the article, with the full toolbar.
- **Gates:** 1.15, 1.20.
- **Depends on:** P1-19, P1-09
- **Touches:** `web/templates/kb/editor.html`, `web/static/kb/editor.js`, `web/static/kb/messages.js`, `internal/kb/` handlers (category picker, title field, hidden `version_no`). Remove the `/__test/editor` harness usage from E2E and keep the harness only if still used by P1-10 tests (otherwise delete the route).
- **Behaviour:**
  - Toolbar buttons with text labels: Large heading, Small heading, Bold, Italic, Bullets, Numbers, Link, Table (with Add row, Remove row, Add column, Remove column when inside a table), Image (file chooser), and **Import from Word** (disabled with a tooltip until P1-33; gate 1.45 is proven in P1-33).
  - Publish and Cancel buttons.
  - A `beforeunload` prompt when dirty; Cancel on a dirty editor asks "Leave without saving?"
- **Tests:**
  - E2E 1.15: apply each action, publish, and assert the element in the published article.
  - E2E 1.20: dirty then navigate triggers the dialog (`page.on('dialog')`), and Cancel asks first.
  - Keyboard-only E2E: publish an article.
  - Standard page checks for the editor.
- **Done when:** tests are green in CI.

## Milestone 1.4 — Images

### P1-21 Image store, upload, paste and drop
- **Goal:** screenshots and photos go into articles and are stored and served by Comp HQ.
- **Gates:** 1.16, 1.17.
- **Depends on:** P1-20, P1-10
- **Touches:** `internal/kb/images/` (`Store(ctx, r io.Reader, person) (Image, error)`: 20 MB cap via `io.LimitReader`, `http.DetectContentType` plus WebP signature check, JPEG/PNG/GIF/WebP only, SHA-256 name, write to temp then rename into `/data/images/ab/`, insert `images` row idempotently), `migrations/kb/0004_images.sql`, `POST /kb/images`, `GET /images/{sha}.{ext}` (nosniff, `Cache-Control: public, max-age=31536000, immutable`), `web/static/kb/editor.js` (FileHandler paste and drop upload, then insert `img`; plain messages for too large and wrong type; a `.docx` drop is routed to import from P1-33 and until then shows a "coming soon" message).
- **Tests:**
  - Unit: sniffing, SVG refused, a renamed `.txt` refused, duplicate collapse.
  - Integration: 20 MB boundary (exactly 20 MB accepted, 20 MB + 1 refused), headers, the origin check on upload.
  - E2E: 1.16, paste PNG, visible within 3s, `src` starts `/images/`, file exists on disk, and after publish the image loads with no request to another host (`page.on('request')` host check). 1.17, drop each of the four types; 21 MB and `.txt` show the message and the editor content is unchanged.
- **Done when:** tests are green in CI.

### P1-22 Paste clean-up for web pages and Word
- **Goal:** pasted content keeps structure and loses styling; Word's unreachable pictures become marked placeholders.
- **Gates:** 1.18.
- **Depends on:** P1-21
- **Touches:** `web/static/kb/editor.js` (a `transformPastedHTML` hook: replace `img[src^="file:"]` with a `data-missing-kind="picture"` placeholder node and show the exact 1.18 message; `data:` and `http(s)` images are kept for publish-time handling in P1-23), a TipTap placeholder node in `scripts/vendor-editor/entry.js` if needed (re-vendor and update the checksum), editor CSS for red dashed boxes.
- **Tests:**
  - E2E with `e2e/fixtures/paste/web-page.html`: headings, bold, lists, links, table and images kept; no `style`, `class` or `font` in the published HTML.
  - E2E with `word.html`: the same minus images; placeholders present; exact message text.
  - Unit (Go): the sanitiser strips the same fixture's styles.
- **Done when:** tests are green in CI.

### P1-23 External images copied on publish
- **Goal:** publishing copies `data:` and web images onto the NAS, safely, and reports any failures.
- **Gates:** 1.19.
- **Depends on:** P1-22
- **Touches:** `internal/kb/images/fetch.go` (an HTTP client with a custom `net.Dialer.Control` refusing loopback, private, link-local, multicast and unspecified IPs at connect time; 10s timeout; 20 MB cap; max 3 redirects; `http`/`https` only; a test-only hook to allow the stub server's loopback address, settable only from Go tests and `COMPHQ_TEST_MODE=1` with `COMPHQ_TEST_ALLOW_FETCH_HOST`), publish pipeline integration (before sanitise: rewrite or replace with a `data-missing-src` placeholder), the editor response listing failed images with plain text.
- **Tests:**
  - Integration: `data:` decode and store; stub success rewrites to `/images/`; refusal for `127.0.0.1`, `10.x`, `169.254.x`, `[::1]`, and a redirect to a private address; timeout; oversize; non-image content.
  - E2E: paste HTML pointing at the Playwright-started stub image server, publish, `src` is local. Stop the stub, publish again: failures are named, placeholders are shown, and the text is saved.
- **Done when:** tests are green in CI.

## Milestone 1.5 — History, conflicts, search

### P1-24 History, restore, archive
- **Goal:** every version is viewable and restorable; articles can be archived and restored, and never deleted.
- **Gates:** 1.22, 1.23, 1.24, 1.25, 1.05 (history part).
- **Depends on:** P1-19
- **Touches:** `internal/kb/` (`GET /kb/articles/{id}/history`, `GET /kb/articles/{id}/versions/{n}`, `POST …/restore`, `POST …/archive`, `POST …/unarchive`, `GET /kb/archived`; archive deletes search rows and unarchive re-adds them, in the same transaction), templates.
- **Tests:**
  - Integration: the version sequence and actions; the route table has no DELETE route or delete handler for articles (test iterates the registered routes).
  - E2E: 1.22 newest first with name and time; 1.23 old version displayed and restored, adding a "restored" entry with the restorer's name; 1.24 archive hides from category and search, is listed in Archived, and restore brings both back; 1.05 rename a person and their name changes in History, remove them and History still shows the name.
  - Standard page checks for History, a version view, and Archived.
- **Done when:** tests are green in CI.

### P1-25 Edit conflicts
- **Goal:** a second publisher sees a plain conflict message and keeps their text.
- **Gates:** 1.21.
- **Depends on:** P1-20
- **Touches:** `internal/kb/` publish handler (compare `version_no`; on mismatch re-render the editor with the posted content, the message, and a "Publish mine anyway" button posting `override=1`), templates.
- **Tests:** E2E with two browser contexts, per SPEC B7 1.20–1.21: the exact message, text still present, override saves, and both versions are in History. Integration: mismatch returns the editor rather than a redirect.
- **Done when:** tests are green in CI.

### P1-26 Search: live panel, results page, scroll-to-passage
- **Goal:** the search box on every page shows highlighted passages, and clicking one lands on the highlighted paragraph.
- **Gates:** 1.26, 1.27, 1.28, 1.29, 1.30, 1.31, 1.32.
- **Depends on:** P1-24, P1-08, P1-14
- **Touches:** `internal/kb/search/` (handlers `GET /kb/search.json?q=` and `GET /kb/search?q=`; top 20 by best block), `internal/app` layout (search box wired on every page), `web/static/kb/search.js` (250 ms debounce, panel under the box, keyboard navigation, Enter goes to the results page, "No articles match *words*"), the article page (server marks target block `data-b` from `#b-N`/`?q=`, passes `highlight()` terms in `data-hl`; `web/static/kb/highlight.js` scrolls into view, adds the highlight class, wraps terms in `<mark>`; "Clear highlights" control).
- **Tests:**
  - E2E: the exact 1.28 script from SPEC B7; 1.26 result count ≤ 20 and title-above-body ordering; 1.27 title, category and marked passage; 1.29 stemming and prefix; 1.30 publish then search, edit to add and remove a word then search; 1.31 archived never shown; 1.32 no-match text, and the symbol fuzz list typed into the box gives no error and no console error.
  - Integration: `search.json` shape and escaping.
  - Standard page checks for the results page, and axe with the panel open.
- **Done when:** tests are green in CI.

### P1-27 Test-library seeder and Knowledge Base speed tests
- **Goal:** catch slow designs early, with the SPEC B7 layer-5 harness and the KB limits.
- **Gates:** 1.33.
- **Depends on:** P1-26, P1-21
- **Touches:** `e2e/seed/` (Go program using internal store functions; deterministic PRNG; `-articles 500 -words 800 -images` using 20 generated PNGs reused across articles; extended in P2-12 and P2-17 for tasks, people and events), `e2e/speed/` (`kb.speed.spec.ts`: KB home, a category, an article, the editor, History, Archived, search results page, each navigation to `data-ready` median of 5 ≤ 1,500 ms; last keystroke to panel render ≤ 1,000 ms; `search.json` p95 over 50 queries ≤ 200 ms), `playwright.config.ts` speed project (seeds once into its own data dir).
- **Tests:** layer 5.
- **Done when:** a `[speed]` push has a green `speed` job. If a limit fails, fix the design (indexes, query shape, template work). **Never raise the limit.**
- **Decisions:** D-14 (seeder design, timing method).

### P1-28 Internal release v0.0.2 (rollback test with real data)
- **Goal:** create a previous release that has Knowledge Base data, so gate 1.42 is proven for real before v0.1.0.
- **Gates:** 1.42 (enables the proof).
- **Depends on:** P1-27, P1-12
- **Touches:**
  - `e2e/smoke/` `@seed`: create a person, a category, an article with a pasted image and an edit (2 versions).
  - `@verify`: the article opens with its image, and History has 2 entries.
  - `@verify-prev` in the current branch: data seeded by the previous release is readable.
  - `deploy/truenas.yaml` image `0.0.2`; `CHANGES.md`.
- **Done when:** full CI (`gh workflow run ci.yml -f full=true`) is green; tag `v0.0.2` and `release.yml` are green; the package lists `0.0.2`; the next `[container]` push shows upgrade/rollback green against `v0.0.2`.

## Milestone 1.6 — Import from Word

### P1-29 Word converter: text, styles, headings, title, revisions, fields
- **Goal:** convert paragraphs and runs correctly across realistic Word structures.
- **Gates:** 1.46 (headings, bold, italic, formatting removal), 1.48, 1.49 (tracked changes, left-out parts counted).
- **Depends on:** P1-11
- **Touches:** `internal/kb/docx/` (streaming `encoding/xml` decoder with a depth counter ≤ 256; style resolution through `basedOn` for paragraph and character styles by **name**; `w:outlineLvl`; the title rule; bold/italic toggles including `w:val="0"`/`false` and style inheritance; `w:tab` to space; `w:br` to `br`, ignoring page and column breaks; include `w:ins` and `w:moveTo`, skip `w:del`, `w:delText`, `w:moveFrom`; descend into `w:sdt`, `w:smartTag`, `w:customXml`, `w:fldSimple`; skip field instructions; drop empty paragraphs; count comments, footnotes, endnotes, headers and footers into `notes`).
- **Tests (unit):** one small generated fixture per rule (add cases to the generator; no hand-edited binaries): localised style IDs, a `basedOn` chain, runs split mid-word, bold off in a bold style, tracked insert and delete, `sdt` content, the title from Title style, from first Heading 1, and from the filename fallback.
- **Done when:** tests are green in CI; `npm run fixtures:docx` reproduces the fixtures.

### P1-30 Word converter: lists and links
- **Goal:** bulleted and numbered lists with nesting, and hyperlinks.
- **Gates:** 1.46 (lists including indented levels, links).
- **Depends on:** P1-29
- **Touches:** `internal/kb/docx/` (numbering: `w:numPr` direct or via style; `numbering.xml` abstract levels; `bullet` gives `ul`, others `ol`; `ilvl` nesting; consecutive same `numId` in one list; `numId` 0 is not a list; links: `w:hyperlink` with external relationship; `HYPERLINK` simple and complex fields; internal anchors keep text only).
- **Tests (unit):** nested three levels mixed bullets and numbers, list interrupted by a paragraph, list defined via paragraph style, a complex-field hyperlink spanning runs, a `mailto` link, an internal anchor.
- **Done when:** tests are green in CI.

### P1-31 Word converter: tables, text boxes, alternate content
- **Goal:** tables with header rows and merged cells, text-box text, and correct `mc:AlternateContent` choice.
- **Gates:** 1.46 (tables including merged cells), 1.49 (text boxes kept).
- **Depends on:** P1-30
- **Touches:** `internal/kb/docx/` (`w:tbl` to `table`; `w:tblHeader` rows to `th`; `gridSpan` to `colspan`; `vMerge` restart/continue to `rowspan` computed per column; nested tables flattened into cell paragraphs; `w:txbxContent` paragraphs output after the anchoring paragraph; `mc:AlternateContent` uses exactly one branch yielding image or text).
- **Tests (unit):** 3×3 with horizontal and vertical merges (exact colspan and rowspan), header row, nested table, a text box in `mc:Choice` with VML fallback (text appears once).
- **Done when:** tests are green in CI.

### P1-32 Word converter: pictures, unsupported items, limits, unreadable files
- **Goal:** pictures with alt text; clear placeholders for what can't come across; safe handling of hostile or wrong files.
- **Gates:** 1.47 (converter part), 1.49, 1.51 (converter part).
- **Depends on:** P1-31
- **Touches:** `internal/kb/docx/` (`w:drawing` inline and anchor `a:blip r:embed`; VML `v:imagedata r:id`; alt from `wp:docPr descr` or `title`; SVG icons use the PNG blip; EMF, WMF, TIFF, BMP and `TargetMode="External"` become `picture` placeholders and are never fetched; charts, SmartArt, textless shapes, equations and objects without images become the matching `data-missing-kind` placeholders with the 1.49 wording; `Result.Images` hold bytes, alt and a placeholder token; limits: 50 MB input checked by handler, ≤ 5,000 entries, ≤ 300 MB total uncompressed enforced by counting readers, ≤ 20 MB per image, relationship targets resolved inside the zip only; unreadable detection for non-zip (including CFB/OLE and PDF signatures), a missing main document, or a truncated zip, all giving `ErrUnreadable`).
- **Tests (unit):** `sample.docx` gives the exact expected placeholders and counts; alt text; the `bad/` fixtures; zip bomb stopped by limit; path traversal target `../../x` ignored; a fuzz test (`go test -fuzz` seed corpus from fixtures, run 30s in this task locally, then committed as a regular seed-corpus test) never panics.
- **Done when:** tests are green in CI.

### P1-33 Import from Word in the editor
- **Goal:** the button and drop import `.docx` into the editor without publishing, with pictures stored and a clear summary.
- **Gates:** 1.45, 1.46 (end to end), 1.47, 1.48 (title field), 1.49 (message), 1.50, 1.51, 1.52.
- **Depends on:** P1-32, P1-23, P1-25
- **Touches:**
  - `internal/kb/` `POST /kb/import/docx` (multipart; 50 MB limit with the plain message; calls `docx.Convert`, stores images via `images.Store`, rewrites tokens to `/images/…` with alt, runs `html.Sanitize`; returns `{title, html, notes}`; never touches articles; on `ErrUnreadable` returns the exact 1.51 text from the template).
  - `web/static/kb/editor.js` (enable the button; file input `accept=".docx"`; `.docx` drop; the "Replace what's in the editor with this document?" confirm when not empty; set the title field; show the summary "N things couldn't be brought in: …" plus left-out parts).
  - Content check listing is covered in P1-36.
- **Tests:**
  - E2E: import `sample.docx` by button and by drop, asserting headings, nested lists, colspan and rowspan, `img src` under `/images/` and the files on disk, alt text, title, placeholders, summary text, and that **no article exists** before Publish.
  - E2E: `googledocs.docx` imports.
  - E2E: import into an existing article, confirm the replace (Cancel changes nothing), publish, and History gains a version whose previous version restores.
  - E2E: each `bad/` fixture and a generated 51 MB file shows the exact message and the editor is unchanged.
  - E2E: 1.52 time from `setInputFiles` of `big-20pages.docx` to editor populated ≤ 10,000 ms.
  - E2E: for each `samples/word/*.docx` present, no error, at least one block, and all `img src` local.
- **Done when:** tests are green in CI.

## Milestone 1.7 — Safety net, docs and release

### P1-34 Nightly backups and banner
- **Goal:** automatic nightly backups kept to 30, status in Settings, and a warning banner when stale.
- **Gates:** 1.35.
- **Depends on:** P1-15
- **Touches:** `internal/db/backup.go` (`VACUUM INTO` temp then rename; prune daily to 30 and pre-update to 10; record `last_backup_at`, `last_backup_ok`, `last_backup_error`), `internal/db/scheduler.go` (injectable clock; run at startup if > 24h since last, then daily 03:00 local), `internal/app` layout banner when `now − last success > 48h`, `internal/settings` Backups section. The pre-update backup in the migration runner uses the same function.
- **Tests:**
  - Unit: pruning order and counts; scheduler next-run across a DST change in `Europe/London`.
  - Integration: a fake clock runs a backup, the file exists and opens as SQLite, and status updates.
  - E2E: set `last_backup_at` to 3 days ago (test-mode helper), banner on every registered page; fresh value, no banner.
- **Done when:** tests are green in CI.

### P1-35 Export everything
- **Goal:** one zip that opens offline with every article readable.
- **Gates:** 1.34.
- **Depends on:** P1-24, P1-21
- **Touches:** `internal/kb/export.go` (exposes an export interface: articles by category including `_Archived`, self-contained HTML with inline CSS, relative `../../images/…` paths), `internal/settings/export.go` (`GET /settings/export` streams `comphq-export-YYYY-MM-DD.zip`: `index.html` contents page with archived ones marked, `articles/…`, `images/…`, `comphq.db` via fresh `VACUUM INTO` to temp; `tasks.csv`/`events.csv` added in P2-18), a Windows-safe filename function (reserved names, `<>:"/\|?*`, trailing dots and spaces, length ≤ 100, duplicate suffixes).
- **Tests:**
  - Unit: filename sanitising table.
  - E2E per SPEC B7 1.34: download, unzip to temp, a new context with `offline: true` opens `file://…/index.html`, follows every article link, asserts headings and images `naturalWidth > 0`, and no failed requests.
- **Done when:** tests are green in CI.

### P1-36 Content check
- **Goal:** Settings shows how many articles have images not on the NAS, or Word items still to fix.
- **Gates:** 1.36, 1.49 (the Content check part).
- **Depends on:** P1-33, P1-34
- **Touches:** `internal/kb` (content-check query interface over published and archived bodies: `data-missing-src`, `data-missing-kind`, or `img` not under `/images/`), `internal/settings` Content check section with count and links.
- **Tests:** E2E: a clean library shows 0; an article published with a failed external image is listed; an imported article with a chart placeholder is listed until the box is removed and republished, then it's gone.
- **Done when:** tests are green in CI.

### P1-37 README part 2
- **Goal:** complete the owner guide for Phase 1.
- **Gates:** 1.44.
- **Depends on:** P1-35, P1-36, P1-16
- **Touches:** `README.md`, the README heading test (all SPEC 1.44 and B8 rule 7 headings).
- **New sections** (click-by-click): Backing up (snapshots, Export everything), Restoring (roll back a snapshot; restoring one article from History; the one sentence that daily and pre-update copies exist in the `backups` folder for anyone helping), "It won't open — what now?" (Apps → comphq status, restart, check version, roll back), Moving your HelpScout articles (copy and paste per article, Content check should reach 0), Word samples (`samples\word`), and a table of every A13 hands-on moment pointing to its section.
- **Tests:** the README test.
- **Done when:** tests are green in CI.

### P1-38 Phase 1 hardening and full gate sweep
- **Goal:** prove every Phase 1 gate in every environment that SPEC requires, before releasing.
- **Gates:** 1.39, 1.41 (full), plus a re-check of all of 1.01–1.52.
- **Depends on:** P1-37, P1-28
- **Touches:**
  - `deploy/test/container-test.sh`: the offline sub-test now runs the full E2E suite, excluding `@fresh`, in `BASE_URL` mode against the container; 1.19 must report the plain failure.
  - The restart sub-test times healthy ≤ 60s and asserts article, image, history and names intact.
  - `e2e/screens/`: every Phase 1 page type.
  - `reports/gates-phase-1.md`: generated gate checklist mapping gate to test file and test name (from §6), with results filled from the CI run.
- **Tests:** layer 4 full; layer 7.
- **Done when:** `gh workflow run ci.yml -f full=true` is green (all jobs: go, browser, container, speed, screens), and every Phase 1 row in §6 points to a test that ran in that run. Record the run URL.

### P1-39 Release v0.1.0 and Phase 1 report — **STOP S3**
- **Goal:** publish v0.1.0 and hand the owner a plain-language report and install steps.
- **Gates:** O1.1, O1.2, O1.3, O1.4 (owner); final confirmation of 1.01–1.52.
- **Depends on:** P1-38
- **Steps:** follow the release procedure in §5.2 with version `0.1.0`, then send **S3** (§5.1).
- **Done when:** the tag and image exist; `reports/phase-1-report.md` and screenshots are written; S3 is sent; the owner has confirmed O1.1, O1.2 and O1.4 (O1.3 may follow later, see §5.3). Record the replies in the Stop log.

# Phase 2 — Tasks and Calendar (→ v0.2.0)

Starts only after S3 is answered (§5.3). **Don't assume any real content exists.** The owner may not have entered events or Trello cards yet, and HelpScout articles may still be arriving. All tests use their own fixtures.

## Milestone 2.1 — Board

### P2-01 Tasks data, Board columns, add task, cards
- **Goal:** the Board with four columns, adding tasks, and cards showing title, people, due date, size and OVERDUE.
- **Gates:** 2.01, 2.02, 2.03, 2.26 (size default and card display).
- **Depends on:** P1-39 (owner go-ahead)
- **Touches:**
  - `internal/tasks/` (store, handlers, section registration replacing the Coming soon page; routes `GET /tasks/board`, `POST /tasks`; `GET /tasks` temporarily redirects to the board until P2-10),
  - `migrations/tasks/0001_tasks.sql` (tasks, task_assignees, task_activity per SPEC B3),
  - `web/templates/tasks/`, `web/static/tasks/`.
- **Behaviour:**
  - New tasks go to the bottom of the chosen stage (default Ideas) with size `M`, and record `created` activity.
  - Initials circles use a colour derived from the person id, with contrast checked.
  - OVERDUE shows when `due_date < today` and the task isn't done.
- **Tests:**
  - Integration: position assignment, creator recorded.
  - E2E: 2.01 column order; 2.02 title-only add, visible in a second context after reload; 2.03 card content and OVERDUE with `COMPHQ_TEST_TODAY`.
  - Replace the 1.08 Tasks "Coming soon" assertion with a Tasks page check (rule 2.5 exception).
  - Standard page checks.
- **Done when:** tests are green in CI.

### P2-02 Ordering engine and move buttons
- **Goal:** the one move endpoint with Move up, Move down and Move to… buttons.
- **Gates:** 2.04 (button part), 2.05 (button to bottom; header text).
- **Depends on:** P2-01
- **Touches:**
  - `internal/tasks/order.go`: pure renumbering functions.
  - `POST /tasks/{id}/move` with `stage` plus one of `before_id`, `after_id` or `to_bottom`. Renumbering happens in one transaction, with `moved` activity when the stage changes.
  - The card menu, and the "Top = most important" header.
- **Tests:**
  - Unit: renumber tables (move within a stage, across stages, to bottom, before or after a missing id → error), and positions stay unique.
  - Integration: concurrency via two transactions ends consistent.
  - E2E: buttons move and reorder; the order is the same in a second context after reload; a card moved into To do by button lands at the bottom.
  - Keyboard-only E2E: move a task by buttons.
- **Done when:** tests are green in CI.

### P2-03 Dragging on the Board
- **Goal:** drag between columns and within a column, landing where dropped.
- **Gates:** 2.04 (drag part), 2.05 (dragged card lands where dropped).
- **Depends on:** P2-02
- **Touches:**
  - `web/static/vendor/sortable/Sortable-1.15.7.min.js` (from npm `sortablejs`, copied by `scripts/vendor-sortable.mjs`), plus a `VENDOR.md` entry.
  - `web/static/tasks/board.js`: on drop, call the move endpoint with the neighbour id, then re-render from the server response.
- **Tests:** E2E: `locator.dragTo` (or explicit `mouse` steps if SortableJS needs them; decide in this task) for a cross-column move and within-column reorder, asserting server order after reload.
- **Done when:** tests are green in CI.
- **Decisions:** D-06 (SortableJS and drag-test method).

### P2-04 Task details dialog and Activity
- **Goal:** open a card to edit every detail and see its Activity.
- **Gates:** 2.06, 2.26 (size editable, size change in Activity).
- **Depends on:** P2-02
- **Touches:**
  - `internal/tasks/`: `GET /tasks/{id}` (a full page, also used by calendar links) and `POST /tasks/{id}` (title, notes, people multi-select of active people, due date, size, stage).
  - Activity actions per SPEC B3, with the rendering "Sam moved this to In progress · Sat 19 Sep". People resolve by id (renames show).
- **Tests:**
  - Integration: each change writes the right activity rows.
  - E2E: edit all fields; Activity lists created, moved, assigned and unassigned, due changed, and size changed, with names and dates.
  - Keyboard-only E2E for the dialog.
  - Standard page checks.
- **Done when:** tests are green in CI.

### P2-05 Board filters
- **Goal:** My tasks, a single person, or a word.
- **Gates:** 2.07.
- **Depends on:** P2-04
- **Touches:** `internal/tasks/` board query params `?mine=1`, `?person=`, `?q=` (title `LIKE`, case-insensitive; the filter state is in the URL), and the filter bar.
- **Tests:** E2E: each filter shows exactly the matching cards among fixtures.
- **Done when:** tests are green in CI.

### P2-06 Finished tasks and Removed tasks
- **Goal:** Done cards older than 14 days leave the board and appear in Finished tasks; tasks can be removed and restored.
- **Gates:** 2.08, 2.09.
- **Depends on:** P2-04
- **Touches:**
  - `internal/tasks/`: board query excludes `done_at < now − 14 days` and `removed_at IS NOT NULL`.
  - `GET /tasks/finished` (newest first) with Reopen (to the bottom of To do, `done_at` cleared, `reopened` activity).
  - `POST /tasks/{id}/remove` and `/restore`, `GET /tasks/removed`.
  - No delete route.
- **Tests:**
  - Integration: the 14-day boundary using fixture `done_at` values of 14d−1m and 14d+1m against a fixed clock; a route-table test for no delete.
  - E2E: 2.08 and 2.09 flows.
  - Standard page checks for Finished and Removed.
- **Done when:** tests are green in CI.

### P2-07 Refresh without live sockets; removed people on tasks
- **Goal:** other computers see changes within 60s, or instantly on focus, without wiping an open editor; removed people are marked.
- **Gates:** 2.10, 2.11.
- **Depends on:** P2-06, P2-03
- **Touches:**
  - `internal/tasks/`: `GET /tasks/version` (a change counter in `app_meta` key `tasks_version`, bumped in every task write transaction).
  - `web/static/app/refresh.js`, a shared module:
    - poll every 60s while `document.visibilityState === 'visible'`, and immediately on `focus` and `visibilitychange`;
    - re-fetch the section fragment and swap it in;
    - skip when a drag is active or a dialog/editor is open, and show "Updated — refresh" instead.
  - Assignment lists show only active people; cards and details show removed people as "Name (removed)".
- **Tests:**
  - E2E with two contexts: a change in A appears in B after `page.clock.runFor(61_000)` (clock installed before load); a change appears immediately after dispatching `focus`; with the dialog open in B, content isn't replaced and the notice shows.
  - E2E 2.11.
- **Done when:** tests are green in CI.
- **Decisions:** D-15 (poll counter design; `page.clock` in place of 65s real waits, which is still faithful to "within 60 seconds").

## Milestone 2.2 — Team view and My jobs

### P2-08 Team view: lanes and workload
- **Goal:** Unassigned plus one lane per person, with Working on now, numbered Up next, and workload blocks.
- **Gates:** 2.24, 2.25, 2.27, 2.30.
- **Depends on:** P2-07
- **Touches:**
  - `internal/tasks/lanes.go`: a pure function `BuildLanes(tasks, people) []Lane`, reused by My jobs.
  - `GET /tasks/team`, and the switch **My jobs · Board · Team** in the Tasks header.
  - Lane minimum width 232px; the lanes container is `overflow-x: auto`.
  - Blocks are grouped per task (S=1, M=2, L=4), with the line "1 large · 1 medium · 1 small". No thresholds or colours.
- **Tests:**
  - Unit: `BuildLanes` table (unassigned only todo/doing; ideas and done excluded; shared task in two lanes; block counts: one L equals four S).
  - E2E: lane order, grouping, numbering, blocks and text.
  - Standard page checks.
- **Done when:** tests are green in CI.

### P2-09 Team view: assigning and reordering
- **Goal:** hand jobs over by drag or "Assign to…", and reorder a person's Up next in the shared order.
- **Gates:** 2.28, 2.29, 2.31 (sideways scroll inside lanes; accessibility).
- **Depends on:** P2-08
- **Touches:**
  - `internal/tasks/`: `POST /tasks/{id}/assign` (`from_person` nullable, `to_person` nullable; keeps other assignees; records activity; stage and position unchanged).
  - `web/static/tasks/team.js`: drag between lanes calls assign; drag within Up next calls move with the lane neighbour as `before_id`/`after_id`; "Assign to…" and Move up/down buttons.
- **Tests:**
  - Integration: assign semantics.
  - E2E:
    - drag from Unassigned to a person, and from person A to B with a third assignee kept;
    - the button equivalents;
    - reorder in a lane, then on the Board **only the moved task's relative position changed** (compare the full To do order before and after);
    - with 10 people at 1024 × 700 the lanes container scrolls and the page doesn't (`scrollWidth` checks on both).
- **Done when:** tests are green in CI.

### P2-10 My jobs: the Tasks home, and Take it
- **Goal:** Tasks opens on My jobs, showing the person's lane beside Up for grabs, and jobs can be taken.
- **Gates:** 2.32, 2.33, 2.34, 2.35.
- **Depends on:** P2-09
- **Touches:**
  - `internal/tasks/`: `GET /tasks` becomes My jobs (left: `BuildLanes` lane for the current person; right: unassigned doing and todo by stage then position, then unassigned ideas).
  - `POST /tasks/{id}/take` with optional `before_id`/`after_id`, per SPEC B4, in one transaction.
  - `web/static/tasks/myjobs.js`: drag from Up for grabs into Up next, and the **Take it** button.
- **Tests:**
  - Integration: take on unassigned todo keeps position; take with drop position; take idea → todo at drop or bottom; activity rows for both changes.
  - E2E:
    - a fresh name on a computer lands on My jobs;
    - fixtures with own, shared, other-person, unassigned todo/doing and unassigned ideas, asserting exactly what shows on each side;
    - take by drag (drop position respected) and by button (position kept);
    - an idea taken moves to To do.
  - Standard page checks.
- **Done when:** tests are green in CI.

### P2-11 My jobs: take conflicts, moving jobs on, Give back
- **Goal:** the second taker gets a plain message; jobs move to In progress or Done; Give back returns jobs.
- **Gates:** 2.36, 2.37.
- **Depends on:** P2-10
- **Touches:**
  - `internal/tasks/`: take returns 409 with names, and the page shows "*Name* has just taken this job" with no change.
  - My jobs buttons: Start (to In progress), Done, and **Give back** (assign with `from_person` = me, `to_person` = null).
  - My jobs uses `refresh.js`.
- **Tests:**
  - E2E: two contexts take the same job, the first succeeds, the second sees the exact message and nothing is added; the taken job leaves Up for grabs in the other context after `page.clock.runFor(61_000)` and immediately on focus; Done leaves My jobs and appears in Finished tasks; Give back with no other assignee returns it to Up for grabs, and with another assignee it stays off Up for grabs.
- **Done when:** tests are green in CI.

### P2-12 Seeder and speed for Board, Team and My jobs
- **Goal:** the Tasks pages meet the limits with the test library.
- **Gates:** 2.21 (Board), 2.31 (speed), 2.38.
- **Depends on:** P2-11, P1-27
- **Touches:** `e2e/seed/` (2,000 tasks spread across stages, sizes and due dates; 10 people; assignees including shared tasks; activity rows), `e2e/speed/tasks.speed.spec.ts` (Board, Team, My jobs, a task page, Finished, Removed: median of 5 ≤ 1,500 ms), indexes in a new migration if needed.
- **Tests:** layer 5; axe and size checks re-run with the seeded library for Team and My jobs.
- **Done when:** a `[speed]` push is green.

## Milestone 2.3 — Calendar

### P2-13 Recurrence engine
- **Goal:** the pure occurrence expansion that Calendar and Briefing share.
- **Gates:** 2.14, and the logic parts of 2.15, 2.16, 2.17.
- **Depends on:** P1-39
- **Touches:** `internal/calendar/recur/` (pure: `Occurrences(event, exceptions, from, to) []Occurrence` per SPEC B4; multi-day length kept; `until_date` inclusive on start; exceptions keyed by original date; title and notes always from the series).
- **Tests (unit table):**
  - weekly on the same weekday;
  - monthly on the 31st clamped (31 Jan → 28 Feb 2027 → 31 Mar 2027);
  - yearly on 29 Feb → 28 Feb 2027 → 29 Feb 2028;
  - `until` boundary;
  - multi-day across month end;
  - moved: the exact 2.15 dates (12 Dec 2026 moved to 19 Dec 2026; 12 Dec 2027 unchanged);
  - cancelled;
  - a moved exception outside the window still appears when its new date is inside;
  - a range across years.
- **Done when:** tests are green in CI.

### P2-14 Events: add and edit, month view, list view
- **Goal:** the Calendar pages and the full event form.
- **Gates:** 2.12, 2.13, 2.20.
- **Depends on:** P2-13
- **Touches:**
  - `internal/calendar/`: store, handlers, section registration replacing Coming soon.
  - `migrations/calendar/0001_events.sql` (events and event_exceptions per SPEC B3).
  - Routes: `GET /calendar?month=YYYY-MM` (Monday-first grid, Saturdays highlighted, today marked, Previous/Next/Today), `GET /calendar/list` (next 12 weeks), `GET /calendar/new?date=`, `POST /calendar/events`, `GET /calendar/events/{id}`, `POST /calendar/events/{id}`.
  - Form fields per 2.13. "Show in briefing ahead" is Not early, or N days, or N weeks (default 1 week), stored as `notice_days`.
  - Details show "Last changed by *name* · *date time*".
  - Clicking a day starts a new event on that date.
- **Tests:**
  - Integration: validation (title required; end date not before start; end time needs start time).
  - E2E: 2.12 with `COMPHQ_TEST_TODAY=2026-09-16` (Saturdays highlighted and today marked, navigation works, the list view is in date order across 12 weeks); 2.13 every field saved and redisplayed; 2.20.
  - Replace the 1.08 Calendar assertion (rule 2.5).
  - Keyboard-only E2E: add an event.
  - Standard page checks for the month, list, event form and event details.
- **Done when:** tests are green in CI.

### P2-15 Repeating events: just this one, change all, cancel, remove
- **Goal:** single-occurrence moves and cancels, series edits, and removing and restoring events.
- **Gates:** 2.15, 2.16, 2.17, 2.18.
- **Depends on:** P2-14
- **Touches:**
  - `internal/calendar/`: clicking a repeating occurrence asks **"Change just this one"** or **"Change all"**.
  - Just this one: a date and time form plus "Cancel just this one", writing an `event_exceptions` row keyed by the original date.
  - Change all: edits the series row; title and notes are always series-wide.
  - `POST /calendar/events/{id}/remove` and `/restore`, `GET /calendar/removed`. No delete route.
- **Tests:**
  - E2E: the exact 2.15 script (create yearly Sat 12 Dec 2026, move just this one to Sat 19 Dec 2026, then assert Dec 2026 and Dec 2027 month views); 2.16; 2.17 (change all title, and the moved one keeps its date but shows the new title); 2.18.
  - A route-table test for no delete.
  - Standard page checks for Removed events.
- **Done when:** tests are green in CI.

### P2-16 Task deadlines on the calendar
- **Goal:** unfinished tasks with due dates show on their date as outlined tick-box chips linking to the task.
- **Gates:** 2.19.
- **Depends on:** P2-14, P2-06
- **Touches:**
  - `internal/tasks`: an exported query interface `DueTasks(ctx, from, to) []DueTask` (not done, not removed).
  - `internal/calendar`: consumes it through an interface injected at registration, per the B2 boundary.
  - Chip styles.
- **Tests:** E2E: a due task shows with the outlined style (`data-kind="task"`) and opens the task page; finished and removed tasks don't show. Unit: a fake `DueTasks` in calendar handler tests.
- **Done when:** tests are green in CI.

### P2-17 Calendar speed, and the Tasks and Calendar accessibility and size sweep
- **Goal:** Calendar meets the limits with 300 events, and every Phase 2 page passes the checks with seeded data.
- **Gates:** 2.21 (Calendar), 2.23.
- **Depends on:** P2-16, P2-12
- **Touches:** `e2e/seed/` (300 events, 100 repeating, some exceptions), `e2e/speed/calendar.speed.spec.ts` (month and list views median ≤ 1,500 ms), `e2e/a11y/pages.spec.ts` and the size checks covering every Tasks and Calendar page type, with and without the seeded library.
- **Tests:** layers 5 and 6, and size checks.
- **Done when:** a `[speed]` push is green.

### P2-18 Export: tasks and events spreadsheets
- **Goal:** Export everything also includes `tasks.csv` and `events.csv`, readable in Excel.
- **Gates:** 2.22.
- **Depends on:** P2-17, P1-35
- **Touches:**
  - `internal/tasks/export.go` and `internal/calendar/export.go`, exposing export interfaces.
  - `internal/settings/export.go` adds them: UTF-8 with BOM, CRLF, header names in plain English.
  - Tasks columns: Title, Column, Size, People, Due date, Created by, Created, Finished, Removed.
  - Events columns: Title, Date, End date, Start time, End time, Repeats, Until, Show ahead, Notes, Removed.
  - Dates are formatted `Sat 19 Sep 2026`, and additionally ISO in hidden-safe columns if needed for sorting. Decide and record.
- **Tests:** E2E per SPEC B7: download, unzip, parse CSVs, assert the BOM, headers, and a known row with readable dates; 1.34 still passes.
- **Done when:** tests are green in CI.

### P2-19 README for Phase 2 and full Phase 2 sweep
- **Goal:** the owner guide covers Tasks, Calendar, and moving events and Trello cards; everything is re-proved.
- **Gates:** 1.44 (kept current); all 1.xx and 2.xx re-checked.
- **Depends on:** P2-18
- **Touches:** `README.md` (Using Tasks: My jobs, Board, Team; Using the Calendar, including repeats and "just this one"; Entering your yearly events; Moving Trello cards), the README test, `e2e/screens/` (all Phase 2 pages), the offline container run including the Phase 2 E2E suite, `reports/gates-phase-2.md`.
- **Done when:** `gh workflow run ci.yml -f full=true` is green, and every Phase 1 and 2 row in §6 maps to a test in that run.

### P2-20 Release v0.2.0 and Phase 2 report — **STOP S4**
- **Goal:** publish v0.2.0 and report.
- **Gates:** O2.1, O2.2 (owner); final confirmation of 1.01–2.38.
- **Depends on:** P2-19
- **Steps:** follow the release procedure in §5.2 with version `0.2.0` (the upgrade and rollback test now runs against `v0.1.x`), then send **S4**.
- **Done when:** the release exists, the report is written, S4 is sent, and the owner has confirmed O2.1 and O2.2 (and O1.3 if it was still open).

# Phase 3 — Saturday Briefing (→ v1.0.0)

Starts only after S4 is answered.

### P3-01 Briefing rules as a pure function
- **Goal:** `Build(today, tasks, occurrences, person)` in `internal/briefing`, exactly per SPEC A8 and B4, fed through small query interfaces.
- **Gates:** logic for 3.02, 3.03, 3.04, 3.05, 3.06, 3.07, 3.08, 3.09.
- **Depends on:** P2-20 (owner go-ahead)
- **Touches:**
  - `internal/briefing/build.go` (pure).
  - `internal/tasks`: exported `BriefingTasks(ctx, dueOnOrBefore) []BriefingTask`, returning not done, not removed, with assignees and board position.
  - `internal/calendar`: exported `Occurrences(ctx, from, to)` and `UpcomingWithNotice(ctx, after, saturday)`, using `recur` and returning only each series' next qualifying occurrence.
  - Interfaces are declared in `internal/briefing` and wired at registration (B2 boundaries).
- **Rules:**
  - `S` = today if today is a Saturday, else the next Saturday. `F = S + 6`.
  - Must be done: `due ≤ F`, sorted by due date then position. OVERDUE when `due < today`. YOURS when the current person is assigned.
  - This week: occurrences overlapping `[today, F]`, sorted by start. Multi-day events already started get the label "until *date*".
  - Coming up: `start > F` and `start − notice_days ≤ S`, labelled from `S` as "in N weeks" when N days is a multiple of 7, else "in N days".
  - Stamps: TODAY (dated today), OVERDUE, a weekday date such as MON 21 SEP (this week), and IN N WEEKS / IN N DAYS (coming up).
- **Tests (unit, exact SPEC dates):**
  - `S`/`F` for every weekday from 13–19 Sep 2026.
  - 3.03: due Wed 23 Sep is included, Sat 26 Sep is excluded, an overdue item is included and stamped.
  - 3.05: YOURS.
  - 3.06: Exams Mon 21 Sep with note.
  - 3.07: campaign Sat 31 Oct, 42 days' notice, appears for `S` = 19 Sep ("in 6 weeks") and not for `S` = 12 Sep.
  - 3.08: cancelled never appears; moved appears at the new date.
  - 3.09: a multi-day event started on Thu 17 Sep, viewed on Sat 19 Sep, shows "until Tue 22 Sep".
  - A repeating series shows only its next qualifying occurrence in Coming up.
- **Done when:** tests are green in CI.
- **Decisions:** D-19 (label rule for weeks and days; stamp rules).

### P3-02 Briefing page
- **Goal:** the Briefing is the first screen, with headline, three sections, stamps, YOURS and Just mine, links, and friendly empty messages.
- **Gates:** 3.01, 3.02, 3.03, 3.04, 3.05, 3.06, 3.07, 3.08, 3.09, 3.10, 3.11.
- **Depends on:** P3-01
- **Touches:**
  - `internal/briefing/` handlers: `GET /` and `/briefing` both render the Briefing, replacing the Phase 1 redirect and the Coming soon page; `?mine=1` for Just mine.
  - `web/templates/briefing/`: headline "Saturday 19 September — today's briefing" or "Briefing for Saturday 19 September"; sections "Must be done today" or "Must be done this Saturday", "This week", "Coming up"; cards show notes, people and stamps, and link to `/tasks/{id}` or `/calendar/events/{id}?date=`; the empty messages, e.g. "Nothing due before next Saturday."
  - `web/static/briefing/`
- **Tests:**
  - E2E with `COMPHQ_TEST_TODAY` = 2026-09-19, 2026-09-16 and 2026-09-12, with fixtures matching SPEC A11 examples: every gate 3.01–3.11 asserted by visible text, stamp and link target.
  - The 1.08 Briefing assertion is replaced (rule 2.5).
  - Standard page checks, including with empty sections.
- **Done when:** tests are green in CI; no route in the app shows "Coming soon" any more (asserted).

### P3-03 Briefing refresh, speed and checks
- **Goal:** the Briefing refreshes like the Board, and meets the speed and accessibility limits with the test library.
- **Gates:** 3.12, 3.13.
- **Depends on:** P3-02, P2-17
- **Touches:** `internal/briefing` `GET /briefing/version` (combines `tasks_version` and `calendar_version`; calendar writes bump `calendar_version` in the same transaction), `refresh.js` wiring, `e2e/speed/briefing.speed.spec.ts` (median of 5 ≤ 1,500 ms with the full seeded library and `COMPHQ_TEST_TODAY`).
- **Tests:**
  - E2E with two contexts: a task due this week, added in A, appears in B after `page.clock.runFor(61_000)` and immediately on focus. The same for an event edit.
  - Layers 5 and 6, and size checks.
- **Done when:** a `[speed]` push is green.

### P3-04 Final README and full sweep
- **Goal:** complete owner guide; every gate from all phases proven in one full run.
- **Gates:** 1.44 (final); all gates re-checked.
- **Depends on:** P3-03
- **Touches:** `README.md` (Saturday Briefing section, including "Just mine", the "Show ahead" setting explained, and a check that the A13 table and each section match the current app), the README test, `e2e/screens/` (all page types), the offline container run including all E2E, `reports/gates-phase-3.md`.
- **Done when:** `gh workflow run ci.yml -f full=true` is green, and every row in §6 maps to a test in that run.

### P3-05 Release v1.0.0 and final report — **STOP S5**
- **Goal:** publish v1.0.0 and report.
- **Gates:** O3.1, O3.2 (owner); final confirmation of all gates.
- **Depends on:** P3-04
- **Steps:** follow the release procedure in §5.2 with version `1.0.0` (upgrade and rollback against `v0.2.x`), then send **S5**.
- **Done when:** the release exists, the report is written, S5 is sent, and the owner has confirmed O3.1 and O3.2. The build is then complete; mark every row `done`.

---

## Milestone 4.0 — The reported fault, fixed first

### P4-01 Assigned ideas appear in My jobs and Team
- **Goal:** fix the fault the owner reported — a task created with people on it shows on the Board and nowhere else. Done before any design work, because it's independent of it and it's the thing that actually bothers them.
- **Gates:** 4.48, 4.49, 4.50, 4.51.
- **Depends on:** nothing.
- **Touches:** `internal/tasks/lanes.go` (`ListForTeamView`, `buildLane`, `LaneForPerson`, `workloadFor`), `web/templates/tasks/myjobs.html`, `web/templates/tasks/team.html`.
- **Behaviour:**
  - `ListForTeamView` currently selects `stage IN ('doing','todo')` while `Create` defaults a new task to `idea` (`internal/tasks/store.go`), so an assigned idea falls through every list. Include `idea`, and group it separately rather than mixing it into Working on now or Up next.
  - My jobs gains a third group, **Ideas I'm on**, below Up next. Team gains an **Ideas** sub-heading in each person's lane.
  - **Workload counts `todo` and `doing` only.** An idea must not move anybody's blocks — that is the whole reason it's a separate group and not just another row.
  - `ListUpForGrabs` is untouched: an idea nobody is on still appears there (gate 4.50).
- **Tests:**
  - **Write the failing test first** (SPEC B9.8). Unit, in-memory DB: create a task in `idea` with one person on it; assert it appears in that person's My jobs "Ideas I'm on" group and in their Team lane, is absent from Up for grabs, and leaves their workload total unchanged. Watch it fail against today's code before touching `lanes.go`.
  - Unit: an idea with nobody on it still appears in Up for grabs; taking it still moves it to To do (gate 2.34 unchanged).
  - E2E `tasks/create-assigned.spec.ts`: create a task from the board choosing a person, in each of the four columns in turn; assert it appears on the Board, in that person's My jobs, and in their Team lane, every time.
  - Std checks on My jobs and Team.
- **Done when:** tests are green in CI, and the Phase 2 task suites still pass unchanged.
- **Decisions:** D-59 (where assigned ideas sit, and that they don't count towards workload).

---

## Milestone 4.1 — Design foundations

Nothing in this milestone changes what the app does. It is the layer every later task draws on, so it comes first and it is proven mechanically rather than by eye.

### P4-02 Fonts, tokens, and the mockup-comparison harness
- **Goal:** the new palette and type in `theme.css`, and the test harness that makes the mockup the contract for everything after it.
- **Gates:** 4.01, 4.02, 4.03.
- **Depends on:** nothing (may run alongside P4-01).
- **Mockup:** read `docs/design/mockup.html` — the `:root` block and the `.studio[data-accent="orange"]` block are the whole palette; the `--display`, `--body` and `--mono` declarations are the type.
- **Touches:** `web/static/theme/theme.css` (tokens per SPEC B9.2, exactly), `web/static/theme/fonts/` (add Big Shoulders Display and IBM Plex Mono as subset variable `woff2`; **delete `archivo-variable.woff2`**), `tools/VENDOR.md` or `web/static/vendor/VENDOR.md` (both families' OFL licence text and version), `e2e/design/`.
- **Behaviour:**
  - Both new families are SIL OFL, so they ship inside the app. Subset to Latin, `font-display: swap`, same `@font-face` shape as the two already there. **Nothing may reference Archivo when this task is done.**
  - Apply SPEC B9.2's contrast rule: the bright accent is a fill only, ink navy sits on it, `--color-signal-orange-deep` is the only orange used as text. White on the accent is 2.85:1 and is banned.
- **The harness (SPEC B9.9 layer 1), built here because every later task needs it:** a Playwright helper that opens `docs/design/mockup.html` and the running app in the same browser, reads `getComputedStyle` for a named pair of elements, and asserts they agree on the properties given. It must work with no internet: `getComputedStyle` reports the *declared* font stack whether or not the font file loaded, so compare the **first family named**, never a resolved file. Never compare whole screenshots — the two hold different content and such a test would fail forever without teaching anyone anything.
- **Tests:**
  - Unit/policy: a test parses `theme.css` and asserts the B9.2 table exactly — every token present, every value as written — and that no section stylesheet under `web/static/*/` declares its own colour or font literal.
  - Policy: no file references Archivo; no `http(s)://` asset URL in templates or CSS (the existing B7 assertion, unweakened).
  - Harness self-test: the helper correctly reports a match and a deliberate mismatch.
  - Contrast is left to the axe layer, not asserted by hand.
- **Done when:** tests are green in CI.
- **Decisions:** D-60 (**adjusts Part B**) the accent contrast rule, superseding D-09; edit D-09 to say it is superseded rather than deleting it. Also record the font versions and subsetting method.

### P4-03 Shared parts and styled controls
- **Goal:** every reusable piece of the interface, defined once in `theme.css`, so no screen invents its own.
- **Gates:** 4.04, 4.05, 4.06, 4.07, 4.08, 4.09, 4.10.
- **Depends on:** P4-02
- **Mockup:** the `.btn`, `.mini`, `.seg`, `.stamp`, `.size`, `.av`/`.people`, `.load`, `.card`, `.panel-head`, `.date` and `.search` rules, plus the `.load` builder at the foot of its `<script>` for the workload blocks and their spoken label.
- **Touches:** `web/static/theme/theme.css`, `web/templates/app/` (shared partials for stamp, avatar, people stack, size chip, workload blocks, segmented switch), `internal/tasks/avatar.go` (confirm the existing colour-per-person rule matches `.av`/`.av.b`/`.av.c`/`.av.d`/`.av.me`).
- **Behaviour:**
  - Three button kinds and no others: primary, secondary, small. The person using the app always gets the accent avatar (`.av.me`).
  - Styled `select`, `input`, `textarea` and the search box, shared by the board and the task window (P4-08).
  - The date field must show **day-first**. A native `<input type=date>` follows the computer's locale and currently shows `mm/dd/yyyy`, which is wrong for this office. Pick the simplest and most stable fix and record it.
  - Workload blocks carry the spoken equivalent from gate 4.10 ("Workload: 7 blocks — large, medium, small").
- **Tests:**
  - Mockup comparison (P4-02 harness) for each part: font, weight, size, letter-spacing, transform, background, border, radius.
  - Unit: workload blocks for S/M/L are 1/2/4 and the spoken label reads correctly for a mixed set.
  - Policy: no `<button>`, `<select>` or `<input>` renders anywhere without a theme class (gate 4.05) — a repository-wide template assertion, so later screens can't regress it.
  - Std checks on a harness page showing every part at once.
- **Done when:** tests are green in CI.
- **Decisions:** D-61 (how the day-first date field is done).

### P4-04 Icons and the icon-only button rule
- **Goal:** one line-art icon set, and the rule that keeps icon-only buttons usable.
- **Gates:** 4.11, 4.12.
- **Depends on:** P4-03
- **Mockup:** the inline `<svg>` elements in `.nav`, `.search`, `.bar .word` and `.card .mv` — 24 × 24, stroke width 2, round caps and joins, `fill:none`, `stroke:currentColor`.
- **Touches:** `web/static/theme/icons/` (add `arrow-up`, `arrow-down`, `trash`, `give-back`, `plus`, `search`, `check`, `chevron`, `close`; redraw the five section icons to match if they differ), `web/templates/app/` (an icon-button partial), `web/static/theme/theme.css` (`.icon-btn`).
- **Behaviour:**
  - An icon takes its button's colour (`currentColor`), so one file serves a dark sidebar and a white card.
  - **Icon-only buttons exist for exactly four things:** move up, move down, remove, give back (SPEC A2, as amended at v1.1). Every one carries both a `title` and an `aria-label` saying the same thing and naming the task: "Move *Print exam papers* up".
- **Tests:**
  - Policy: every icon-only button in every template has both `title` and `aria-label`, and they match; no icon-only button exists outside the four permitted actions.
  - Mockup comparison: stroke width, size and colour inheritance.
  - Axe on a page using them.
- **Done when:** tests are green in CI.

### P4-05 The frame
- **Goal:** the sidebar and top bar as drawn, seen on every screen.
- **Gates:** 4.13, 4.14.
- **Depends on:** P4-04
- **Mockup:** `.side`, `.mark`, `.nav`, `.top`, `.search`, `.you` — note `.mark` is a small wide-spaced word above a large accent word, and `.nav[aria-current="page"]` is an accent block with **ink** text, not white.
- **Touches:** `internal/app` (layout template), `web/static/app/frame.css`.
- **Behaviour:** the wordmark is **COMP** above **HQ** (the mockup says "Staff"; the app is Comp HQ — SPEC A1). The top bar keeps "You: *name* · Change" and gains that person's avatar.
- **Tests:** mockup comparison on the sidebar, the wordmark, the current-section block and the top bar; existing frame E2E (gates 1.07–1.11) still green; std checks.
- **Done when:** tests are green in CI.

---

## Milestone 4.2 — Tasks

### P4-06 Board: controls, columns and cards
- **Goal:** the Board as drawn, with the add-a-task form gone from the page.
- **Gates:** 4.15, 4.16, 4.17, 4.21.
- **Depends on:** P4-05
- **Mockup:** `<section data-screen="board">` with `.tools`, `.board`, `.col`, `.col.todo`, `.col-head`, `.card`, `.card .rank`, `.card.done`.
- **Touches:** `web/templates/tasks/board.html`, `web/static/tasks/board.css`.
- **Behaviour:**
  - The board's controls become one row: the My jobs/Board/Team switch, the Everyone/My tasks/Person filter, the word filter, and a primary **+ Add task** at the right. **The inline add-a-task form is removed** — until P4-08 lands, + Add task is a plain link to a new-task page, which is also the permanent no-JavaScript path (gate 4.28).
  - Card order is fixed by gate 4.16: rank (To do only), title, due date, people, size chip, icon buttons.
- **Tests:** mockup comparison on a column, the To do column's tint, a card and a done card; E2E that no add-task form exists on the board and that + Add task reaches a working page; Phase 2 board suites still green; std checks.
- **Done when:** tests are green in CI.

### P4-07 Card actions: remove, give back, assign
- **Goal:** the three card controls the owner asked for, over endpoints that already exist.
- **Gates:** 4.18, 4.19, 4.20.
- **Depends on:** P4-06
- **Touches:** `web/templates/tasks/board.html`, `web/static/tasks/board.js`, `web/static/theme/theme.css`.
- **Behaviour (SPEC B9.6 — no new endpoints, no new storage):**
  - **Remove** posts to the existing `/tasks/{id}/remove`; it goes to Removed tasks and can be brought back (A3).
  - **Give back** is the existing assign endpoint with `from_person` = self and no `to_person` (`internal/tasks/assign.go`), exactly as My jobs already does it. It takes **only that person** off; anyone else stays on, and the job returns to Up for grabs only if nobody is left. The icon shows only to a person who is on that job.
  - **Assigning** from the people circles opens a short menu of current people; each choice posts to the same endpoint, which already leaves other assignees untouched and already writes the Activity row.
  - Each must work as a plain form submit if its script doesn't run.
- **Tests:**
  - E2E: remove from a card, then find it in Removed tasks and restore it; give back with two people on a job, asserting the other stays and the job does **not** return to Up for grabs; give back as the last person, asserting it does; assign and unassign from the circles and assert the Activity row.
  - E2E: the same three actions with JavaScript disabled.
  - Axe with the assign menu open; std checks.
- **Done when:** tests are green in CI.

### P4-08 The task window: the dialog, and creating a task
- **Goal:** one dialog, built once, that later serves reading and editing too.
- **Gates:** 4.22, 4.25, 4.26, 4.28.
- **Depends on:** P4-07
- **Mockup:** not drawn — the window was agreed after the mockup (SPEC B9.1). Build it from the mockup's own parts (`.card`, `.btn`, `.mini`, `.size`, `.av`, styled controls) so it looks as though it had always been there. **This is the one screen with no picture to check against; owner check O4.3 covers it.**
- **Touches:** `web/templates/tasks/window.html` (new partial), `web/static/tasks/window.js`, `web/static/theme/theme.css`, `web/templates/tasks/board.html`.
- **Behaviour (SPEC B9.7):**
  - Use the platform's own `<dialog>` element rather than building one. It gives Escape, the keyboard staying inside (gate 4.26), the backdrop and the return of focus without hand-written code — the simpler and more stable choice (A2 priority 2).
  - **+ Add task** opens it empty: title, description, size, starting column, due date, people, ending in a primary **Add task** and a Cancel. `handleCreateTask` already accepts every one of those fields (`internal/tasks/handlers.go`), including `notes` and `size`, which the board form never offered — so nothing server-side changes.
  - Closing with unsaved changes asks first. Closing returns the keyboard to whatever opened the window.
  - **If the window can't open, nothing is lost:** + Add task stays an ordinary link to the new-task page (gate 4.28). This is not optional.
- **Tests:**
  - E2E: open from + Add task; create a job with a description and a size and assert both stored; close by Escape, by the close icon and by the backdrop; type something then close and assert it asks; assert focus returns.
  - E2E: Tab cycles inside the window only and never reaches the board behind.
  - E2E **with JavaScript disabled**: + Add task reaches a page where a task can still be created, with the same fields.
  - Axe with the window open; std checks.
- **Done when:** tests are green in CI.
- **Decisions:** D-62 (the `<dialog>` approach and how the no-JavaScript path is kept honest).

### P4-09 The task window: reading, editing and history
- **Goal:** clicking a task opens it to read, with Edit and its history.
- **Gates:** 4.23, 4.24, 4.27, 4.29.
- **Depends on:** P4-08
- **Touches:** `web/templates/tasks/window.html`, `web/static/tasks/window.js`, `internal/tasks/details_handlers.go` (a fragment response for the window; the page response is unchanged).
- **Behaviour:**
  - Clicking a card opens the same window **to read**: description laid out as text, not sitting in a box. A primary **Edit** turns it into P4-08's form, filled in. `POST /tasks/{id}` already edits the whole set in one transaction and already records the right activity row per kind of change (P2-04).
  - History sits at the foot behind an expander, closed on opening, from the existing `ListActivity`.
  - **The task page stays exactly as it is** (gate 4.27): Calendar links point at it and a refreshed window must land somewhere real. v1.2 keeps it too.
  - Saving updates the board without reloading the whole page.
- **Tests:**
  - E2E: open a card, read it, press Edit, change the due date and people, save, and assert the board updated and the right history row appeared — the same row editing from the page produces.
  - E2E: `/tasks/{id}` still answers directly; a Calendar link still lands on it; refreshing that page still works.
  - E2E with JavaScript disabled: a card's title still reaches that page.
  - Axe with the window open and the history expanded; std checks.
- **Done when:** tests are green in CI.

### P4-10 My jobs and Team
- **Goal:** both screens as drawn, including P4-01's new Ideas group.
- **Gates:** 4.30, 4.31, 4.32.
- **Depends on:** P4-09
- **Mockup:** `<section data-screen="tasks">` (`.myjobs`, `.area.mine`, `.area.grabs`, `.area-head`, `.drop`, `.sub`) and `<section data-screen="team">` (`.team`, `.lane`, `.lane.unassigned`, `.lane.me`, `.lane-head`).
- **Touches:** `web/templates/tasks/myjobs.html`, `web/templates/tasks/team.html`, `web/static/tasks/myjobs.css`, `web/static/tasks/team.css`.
- **Behaviour:** Up for grabs and the Unassigned lane are **dashed outlines with no fill**, so they read as holding areas rather than someone's list — currently Unassigned looks exactly like a person. Take it, Give back, Start and Done ✓ keep their words. A card on either screen opens the P4-09 window. Style P4-01's **Ideas I'm on** group and the Team **Ideas** heading to match.
- **Tests:** mockup comparison on both areas, both lane kinds and a lane head; E2E that a card opens the window; Phase 2 suites for 2.24–2.38 still green; std checks.
- **Done when:** tests are green in CI.

---

## Milestone 4.3 — The other sections

### P4-11 Calendar
- **Goal:** the month grid as drawn, and the Saturday that's actually visible.
- **Gates:** 4.33, 4.34, 4.35, 4.36, 4.37, 4.38.
- **Depends on:** P4-05 (independent of the Tasks milestone)
- **Mockup:** `<section data-screen="calendar">` with `.cal-grid`, `.dow`, `.dow.sat`, `.day`, `.day.sat`, `.day.out`, `.day.today .num`, `.chip.ev`, `.chip.ev.cont`, `.chip.task`, `.chip.task.late`, `.legend`.
- **Touches:** `web/templates/calendar/`, `web/static/calendar/calendar.css`.
- **Behaviour:** gate 4.34 fixes a real readability fault — today an out-of-month day and a Saturday are near-identical beige, and Saturday is the point of this screen. Ordinary, Saturday and out-of-month must be three clearly different shades. Today's number sits in a filled navy chip.
- **Tests:** mockup comparison on each of the three day kinds, the Saturday heading, today's chip and both chip kinds; **an explicit assertion that the three day backgrounds are three distinct values** (gate 4.34); Phase 2 calendar suites still green; std checks.
- **Done when:** tests are green in CI.

### P4-12 Knowledge Base
- **Goal:** article, search results and editor as drawn.
- **Gates:** 4.39, 4.40, 4.41, 4.42, 4.43.
- **Depends on:** P4-05
- **Mockup:** `<section data-screen="kb">` (`.article`, `.article h1`, `.article h2`, `.article table`/`th`, `.by`, `.results`, `.hit`, `mark`) and `<section data-screen="editor">` (`.bar`, `.tb`, `.word`, `.imported`, `.title-in`, `.article.doc`, `.missing`).
- **Touches:** `web/templates/kb/`, `web/static/kb/kb.css`, `web/static/kb/editor.css`.
- **Behaviour:** reading areas stay calm — the body keeps the readable face at a generous size on paper (A2 priority 1 beats priority 4 inside an article). The display face is for the title, headings and table headers only.
- **Tests:** mockup comparison on the title, headings, table header, byline row, the results panel and a highlighted hit; Phase 1 KB suites still green; std checks and axe on the results panel.
- **Done when:** tests are green in CI.

### P4-13 Briefing and the name picker
- **Goal:** the two remaining screens.
- **Gates:** 4.44, 4.45, 4.46, 4.47.
- **Depends on:** P4-05
- **Mockup:** `<section data-screen="briefing">` (`.brief-head`, `.brief-grid`, `.panel-head`, `.item`, `.item .note`, `.item.late`) and `#picker` (`.picker`, `.picker .mark`, `.names`).
- **Touches:** `web/templates/briefing/`, `web/static/briefing/briefing.css`, `web/templates/people/`, `web/static/people/who.css`.
- **Tests:** mockup comparison on the date heading, a panel head, an ordinary item, an overdue item and a note strip, and on the picker's grid and hover state; Phase 3 briefing suites and Phase 1 picker suites still green; std checks.
- **Done when:** tests are green in CI.

---

## Milestone 4.4 — Finish

### P4-14 Full sweep: checks, speed, screenshots, README
- **Goal:** prove nothing regressed, and give the owner what they need to check it.
- **Gates:** 4.52, 4.53, 4.54.
- **Depends on:** P4-10, P4-11, P4-12, P4-13
- **Touches:** `e2e/a11y/`, `e2e/speed/`, `e2e/screens/`, `reports/screens/`, `README.md`.
- **Behaviour:**
  - Every Phase 1–3 gate re-run unchanged, at the same thresholds. A weakened, skipped or deleted Phase 1–3 test fails the phase (§2.5 and SPEC B8 rule 5).
  - Refresh **every** screenshot in `reports/screens/`, adding one of the task window open, so the owner can hold them beside the mockup for O4.1.
  - README: the new card icons and what each does, and the task window.
- **Tests:** the full suite at every layer, including speed with the test library (gate 4.53) and the accessibility and window-size checks at 1024 × 700 and full HD (gate 4.52).
- **Done when:** every gate 1.01–4.54 passes in one full CI run.

### P4-15 Release v1.1.0 and Phase 4 report — **STOP S9**
- **Goal:** publish v1.1.0 and report.
- **Gates:** O4.1, O4.2, O4.3, O4.4 (owner); final confirmation of all gates.
- **Depends on:** P4-14
- **Steps:** follow the release procedure in §5.2 with version `1.1.0` (upgrade and rollback against `v1.0.0`), then send **S9**. In the report, tell the owner plainly that the task window is the one screen the mockup doesn't show, and ask them to look at it hardest (O4.3).
- **Done when:** the release exists, the report is written, S9 is sent, and the owner has confirmed O4.1–O4.4. **v1.2 (steps inside a job, and the app icon) is now specified — SPEC A11 Phase 5, B10 and B11 — and planned in `PLAN-v1.2.md`.**

---

## 5. Stop points, releases and owner feedback

### 5.1 Stop points, in order, with the message to send
Send each message as plain text in the chat. Log it in `PROGRESS.md` → Stop log, and wait for the reply, except for N1. Replace *italics* with real values.

**S1 — GitHub sign-in (P1-02)**
> I need you to sign in to GitHub once on this computer, so I can store Comp HQ's code in your **stas-comp** account. It takes about 5 minutes.
> 1. Click **Terminal**, next to this chat.
> 2. Paste this line and press **Enter**: `gh auth login --hostname github.com --git-protocol https --web --scopes workflow`
> 3. If it asks "Authenticate Git with your GitHub credentials?", type **Y** and press **Enter**.
> 4. It shows a code like **ABCD-1234**. Copy it, then press **Enter**. Your browser opens.
> 5. Sign in as **stas-comp** if asked. Paste the code, click **Continue**, then **Authorize GitHub CLI**.
> 6. When the Terminal says **Logged in as stas-comp**, come back here and type **done**.
>
> I never see or type your password.

**S2 — Make the download public (P1-07)**
> The first test version of Comp HQ has been published to your GitHub account. Please make it **public**, so your NAS can download it without a password. It contains the program only, never your data. About 3 minutes.
> 1. Open https://github.com/users/stas-comp/packages/container/package/comphq (sign in as stas-comp if asked).
> 2. On the right, click **Package settings**.
> 3. Scroll to **Danger Zone** and click **Change visibility**.
> 4. Choose **Public**, type **comphq** in the box, and click **I understand the consequences, change package visibility**.
> 5. Come back here and type **done**.
>
> Your code itself stays private.

**N1 — Optional Word samples (P1-10, no waiting)**
> Optional, whenever you like: if you put 1–3 of your real Word documents (ones with pictures and tables, nothing private) in the **samples\word** folder inside the Staff HQ folder on your Desktop, I'll test Import from Word on them too. I'm carrying on either way.

**S3 — End of Phase 1 (P1-39)**, **S4 — End of Phase 2 (P2-20)**, **S5 — End of Phase 3 (P3-05)**, **S9 — End of Phase 4 (P4-15)**, **S10 — End of Phase 5 (P5-09, see `PLAN-v1.2.md`)**, **S11 — End of Phase 6 (P6-08, see `PLAN-v1.3.md`)**
Send the summary part of `reports/phase-N-report.md` (§5.2 step 7):
> **Phase *N* is finished: version *X.Y.Z* is ready.**
> - **What's new:** *two plain sentences*.
> - **Automatic checks:** all *count* promises for this phase and earlier phases passed. The full list and screenshots are in the **reports\phase-*N*** folder inside the Staff HQ folder on your Desktop.
> - **Tests I corrected:** *none, or one line each*.
>
> **What I need from you:**
> 1. *Install or update steps, click by click. Phase 1: the README "Installing on TrueNAS" and "Setting up an office computer" steps, copied in. Later phases: "In TrueNAS, go to Apps → comphq → Edit, change the number after `comphq:` to X.Y.Z, and click Save."*
> 2. *Each 👤 Owner check for this phase, with what to try and what you should see.*
>
> **Reply with** "all OK", or tell me in your own words what isn't right; I'll fix it and send a small update. *(Phase 1 only:)* Moving your HelpScout articles can take a while. If the other checks are fine, you can reply "OK, start Phase 2" and tell me when the HelpScout move is finished.

**S6 — Spec problem or rule conflict (any time)**
Use this when a Part A gate is impossible or contradictory, or a change would add a paid service, a runtime internet dependency, or a login.
> I've hit something in the plan that needs your decision. **What happened:** *plain explanation*. **Options:** *A (recommended): …* / *B: …*. **What changes for you:** *…*. Reply with A or B, or ask me anything.

Keep working on other tasks that don't depend on the answer, if any. Otherwise wait.

**S7 — Stuck after 3 tries (rule 2.5)**
> I'm stuck on *task title*. **What isn't working:** *one or two plain sentences*. **What I tried:** *three short bullets*. **What I recommend:** *…*. Details are in **reports\blocker-*ID*.md**. Reply "go ahead" to accept my recommendation, or tell me what you'd prefer.

**S8 — GitHub's free testing time used up**
Use this when CI jobs don't start, or fail with a billing, spending-limit or minutes message.
> Automatic testing has paused, because this month's free testing time on GitHub is used up. It resets on *date*. **Recommended:** wait until then; nothing is lost, and I'll carry on automatically when you restart me after that date. **Other option:** make the code storage public, which gives unlimited free testing, but anyone could then read the app's code (it contains no data or passwords). Reply "wait" or "make it public".

If the owner chooses public: `gh repo edit stas-comp/comphq --visibility public --accept-visibility-change-consequences` is an account-setting change. **Ask the owner to do it** in GitHub → comphq → Settings → General → Danger Zone → Change visibility, with steps. Don't do it yourself.

### 5.2 Release procedure (used by P1-06 for `0.0.1`, P1-28 for `0.0.2`, and every phase and patch release)
1. Update `deploy/truenas.yaml` to `image: ghcr.io/stas-comp/comphq:X.Y.Z`, add a plain-language `CHANGES.md` entry, and update the README if anything owner-facing changed. Commit `Release vX.Y.Z` and push; wait until CI is green.
2. `gh workflow run ci.yml --ref main -f full=true`, then `gh run watch` until **all** jobs (go, browser, container, speed, screens) are green. The upgrade and rollback test runs against the previous tag.
3. `git tag -a vX.Y.Z -m "Comp HQ X.Y.Z"` and `git push origin vX.Y.Z`. Wait until `release.yml` is green.
4. Confirm publication: `gh api /users/stas-comp/packages/container/comphq/versions --jq '.[].metadata.container.tags[]'` includes `X.Y.Z`, and an anonymous manifest `HEAD` (as in P1-07) returns 200. Skip the anonymous check for `0.0.1`, which is before S2.
5. Skip steps 6–8 for internal releases (`0.0.x`).
6. `gh run download <full run id> -n screens -D reports/phase-N/screens`, then generate `reports/gates-phase-N.md`. For every gate in this and earlier phases, list the gate, a short plain description, the test file and title, and pass. Do the same for the `container` sub-tests. A gate without a passing test **blocks the release**.
7. Write `reports/phase-N-report.md`:
   - summary (S3–S5 text);
   - full gate checklist;
   - tests corrected;
   - plain summary of new `docs/decisions.md` entries;
   - screenshot list;
   - version;
   - install or update steps;
   - owner checks with steps.
8. Commit the reports (`reports/phase-N/screens` included), push, send S3/S4/S5, set the task `blocked` ("waiting for owner checks"), and stop.

### 5.3 When the owner replies to a phase report
- **"All OK":** log the reply, mark the release task `done`, and start the next phase.
- **"OK, start Phase 2" with O1.3 pending:** log O1.3 as `pending` under Owner feedback, and start Phase 2. Ask about O1.3 again in S4.
- **Something isn't right:**
  1. Copy the owner's words exactly under Owner feedback.
  2. For each distinct issue, add a fix task right after the release task in `PLAN.md` and `PROGRESS.md`: `P1-F01`, `P1-F02`, … Each fix task names the gate or owner check it serves, and, where the issue is automatable, **adds a test that fails before the fix**.
  3. Run the §5.2 procedure as a patch release (`vX.Y.Z+1`).
  4. Send a short report covering only what changed, and the checks to repeat.
- **Feedback asking for something not in SPEC.md:** don't build it. Reply in plain language that it's a new feature, not a fix. Recommend noting it for later, and add it to `docs/later.md`. Only if the owner then explicitly asks to change the spec: add a dated row to SPEC.md's decisions table and the gate(s) to Part A, add tasks, and continue.
- **Feedback that contradicts a gate:** use S6.

---

## 6. Gate traceability

**Test naming rule:** every test that proves a gate has the gate number at the start of its title: Playwright `test('1.28 …')`, Go `t.Run("gate 1.25 …")`, or a `// Gate: 1.25` comment on the test function. Container sub-tests print `GATE 1.39 PASS`. The gate reports in §5.2 are generated by searching for these, so a missing tag blocks the release.

Unless noted, E2E paths are under `e2e/`, and "std checks" means `e2e/a11y/pages.spec.ts` plus `e2e/a11y/size.spec.ts` entries.

| Gate | Delivered by | Proven by |
|---|---|---|
| 1.01 | P1-13 | `people/picker.spec.ts` |
| 1.02 | P1-13 | `people/picker.spec.ts`; integration cookie test `internal/people` |
| 1.03 | P1-13 | `people/picker.spec.ts` |
| 1.04 | P1-13 | `people/picker.spec.ts` (two contexts) |
| 1.05 | P1-15, P1-24 | `settings/people.spec.ts`; `kb/history.spec.ts` |
| 1.06 | P1-13 | `people/picker.spec.ts` |
| 1.07 | P1-14 | `frame/frame.spec.ts` (every registered route) |
| 1.08 | P1-14; replaced in P2-01, P2-14, P3-02 | `frame/frame.spec.ts` |
| 1.09 | P1-14 and every page task | `a11y/size.spec.ts` |
| 1.10 | P1-14 and every page task | `a11y/pages.spec.ts`; `a11y/keyboard.spec.ts` |
| 1.11 | P1-14 | `frame/frame.spec.ts` |
| 1.12 | P1-17 | `kb/categories.spec.ts`; `internal/kb` categories integration |
| 1.13 | P1-17, P1-19 | `kb/home.spec.ts` |
| 1.14 | P1-19 | `kb/publish.spec.ts` |
| 1.15 | P1-20 | `kb/editor-toolbar.spec.ts` |
| 1.16 | P1-21 | `kb/images.spec.ts` |
| 1.17 | P1-21 | `kb/images.spec.ts`; `internal/kb/images` unit + integration |
| 1.18 | P1-22 | `kb/paste.spec.ts`; `internal/kb/html` unit |
| 1.19 | P1-23 | `kb/external-images.spec.ts`; `internal/kb/images` fetch integration |
| 1.20 | P1-20 | `kb/editor-leave.spec.ts` |
| 1.21 | P1-25 | `kb/conflict.spec.ts` |
| 1.22 | P1-19, P1-24 | `kb/history.spec.ts` |
| 1.23 | P1-24 | `kb/history.spec.ts` |
| 1.24 | P1-24 | `kb/archive.spec.ts` |
| 1.25 | P1-24 | `internal/kb` route-table integration test |
| 1.26 | P1-26 | `kb/search.spec.ts` |
| 1.27 | P1-26 | `kb/search.spec.ts` |
| 1.28 | P1-26 | `kb/search.spec.ts` (the SPEC B7 script) |
| 1.29 | P1-08, P1-26 | `internal/kb/search` unit; `kb/search.spec.ts` |
| 1.30 | P1-19, P1-26 | `internal/kb` integration; `kb/search.spec.ts` |
| 1.31 | P1-24, P1-26 | `kb/search.spec.ts` |
| 1.32 | P1-08, P1-26 | `internal/kb/search` unit; `kb/search.spec.ts` |
| 1.33 | P1-27 | `speed/kb.speed.spec.ts` |
| 1.34 | P1-35 | `settings/export.spec.ts` (offline `file://`) |
| 1.35 | P1-34 | `internal/db` backup integration; `settings/backup-banner.spec.ts` |
| 1.36 | P1-36 | `settings/content-check.spec.ts` |
| 1.37 | P1-15 | `settings/about.spec.ts` |
| 1.38 | P1-06 | `deploy/test/container-test.sh` → `compose_healthy` |
| 1.39 | P1-06, P1-38 | `container-test.sh` → `restart` (≤ 60s, data intact) |
| 1.40 | P1-06 | `container-test.sh` → `recreate` |
| 1.41 | P1-12, P1-38 | `container-test.sh` → `offline` (full E2E in `BASE_URL` mode) |
| 1.42 | P1-12, P1-28, P1-39 | `container-test.sh` → `upgrade_rollback` (v0.1.0 run vs v0.0.2) |
| 1.43 | P1-15 | `internal/settings` setup integration; `settings/setup.spec.ts` (`--app` launch) |
| 1.44 | P1-16, P1-37 (kept by P2-19, P3-04) | `tools/policy` README test |
| 1.45 | P1-33 | `kb/import-word.spec.ts` |
| 1.46 | P1-29, P1-30, P1-31, P1-33 | `internal/kb/docx` unit; `kb/import-word.spec.ts` |
| 1.47 | P1-32, P1-33 | `internal/kb/docx` unit; `kb/import-word.spec.ts` |
| 1.48 | P1-29, P1-33 | `internal/kb/docx` unit; `kb/import-word.spec.ts` |
| 1.49 | P1-29, P1-31, P1-32, P1-33, P1-36 | `internal/kb/docx` unit; `kb/import-word.spec.ts`; `settings/content-check.spec.ts` |
| 1.50 | P1-33 | `kb/import-word.spec.ts` |
| 1.51 | P1-11, P1-32, P1-33 | `internal/kb/docx` unit; `kb/import-word.spec.ts` |
| 1.52 | P1-33 | `kb/import-word.spec.ts` (timed) |
| 2.01 | P2-01 | `tasks/board.spec.ts` |
| 2.02 | P2-01 | `tasks/board.spec.ts`; `internal/tasks` integration |
| 2.03 | P2-01 | `tasks/board.spec.ts` |
| 2.04 | P2-02, P2-03 | `tasks/move.spec.ts`; `internal/tasks` order unit |
| 2.05 | P2-02, P2-03 | `tasks/move.spec.ts` |
| 2.06 | P2-04 | `tasks/details.spec.ts`; `internal/tasks` activity integration |
| 2.07 | P2-05 | `tasks/filters.spec.ts` |
| 2.08 | P2-06 | `internal/tasks` 14-day integration; `tasks/finished.spec.ts` |
| 2.09 | P2-06 | `tasks/removed.spec.ts`; route-table test |
| 2.10 | P2-07 | `tasks/refresh.spec.ts` (two contexts, `page.clock`) |
| 2.11 | P2-07 | `tasks/removed-people.spec.ts` |
| 2.12 | P2-14 | `calendar/month.spec.ts` |
| 2.13 | P2-14 | `calendar/event-form.spec.ts` |
| 2.14 | P2-13 | `internal/calendar/recur` unit |
| 2.15 | P2-13, P2-15 | `internal/calendar/recur` unit; `calendar/repeats.spec.ts` |
| 2.16 | P2-13, P2-15 | `internal/calendar/recur` unit; `calendar/repeats.spec.ts` |
| 2.17 | P2-13, P2-15 | `internal/calendar/recur` unit; `calendar/repeats.spec.ts` |
| 2.18 | P2-15 | `calendar/removed.spec.ts`; route-table test |
| 2.19 | P2-16 | `calendar/task-deadlines.spec.ts` |
| 2.20 | P2-14 | `calendar/event-form.spec.ts` |
| 2.21 | P2-12, P2-17 | `speed/tasks.speed.spec.ts`; `speed/calendar.speed.spec.ts` |
| 2.22 | P2-18 | `settings/export.spec.ts` |
| 2.23 | P2-01…P2-17 (page tasks), P2-17 sweep | std checks; `a11y/keyboard.spec.ts` |
| 2.24 | P2-08 | `tasks/team.spec.ts` |
| 2.25 | P2-08 | `internal/tasks` lanes unit; `tasks/team.spec.ts` |
| 2.26 | P2-01, P2-04 | `tasks/board.spec.ts`; `tasks/details.spec.ts` |
| 2.27 | P2-08 | `internal/tasks` lanes unit; `tasks/team.spec.ts` |
| 2.28 | P2-09 | `tasks/team-assign.spec.ts`; `internal/tasks` assign integration |
| 2.29 | P2-09 | `tasks/team-assign.spec.ts` |
| 2.30 | P2-08 | `internal/tasks` lanes unit; `tasks/team.spec.ts` |
| 2.31 | P2-09, P2-12 | `tasks/team-assign.spec.ts` (scroll); `speed/tasks.speed.spec.ts`; std checks |
| 2.32 | P2-10 | `tasks/myjobs.spec.ts` |
| 2.33 | P2-10 | `tasks/myjobs.spec.ts` |
| 2.34 | P2-10 | `tasks/myjobs.spec.ts`; `internal/tasks` take integration |
| 2.35 | P2-10 | `tasks/myjobs.spec.ts`; `internal/tasks` take integration |
| 2.36 | P2-11 | `tasks/myjobs-conflict.spec.ts` |
| 2.37 | P2-11 | `tasks/myjobs-conflict.spec.ts` |
| 2.38 | P2-12 | `speed/tasks.speed.spec.ts`; std checks |
| 3.01 | P3-02 | `briefing/briefing.spec.ts` |
| 3.02 | P3-01, P3-02 | `internal/briefing` unit; `briefing/briefing.spec.ts` |
| 3.03 | P3-01, P3-02 | `internal/briefing` unit; `briefing/briefing.spec.ts` |
| 3.04 | P3-01, P3-02 | `internal/briefing` unit; `briefing/briefing.spec.ts` |
| 3.05 | P3-01, P3-02 | `internal/briefing` unit; `briefing/briefing.spec.ts` |
| 3.06 | P3-01, P3-02 | `internal/briefing` unit; `briefing/briefing.spec.ts` |
| 3.07 | P3-01, P3-02 | `internal/briefing` unit; `briefing/briefing.spec.ts` (12 and 19 Sep) |
| 3.08 | P3-01, P3-02 | `internal/briefing` unit; `briefing/briefing.spec.ts` |
| 3.09 | P3-01, P3-02 | `internal/briefing` unit; `briefing/briefing.spec.ts` |
| 3.10 | P3-02 | `briefing/briefing.spec.ts` |
| 3.11 | P3-02 | `briefing/briefing.spec.ts` |
| 3.12 | P3-03 | `briefing/refresh.spec.ts` |
| 3.13 | P3-02, P3-03 | `speed/briefing.speed.spec.ts`; std checks |
| O1.1–O1.4 | P1-39 (S3) | Owner reply, logged in `PROGRESS.md` |
| O2.1–O2.2 | P2-20 (S4) | Owner reply, logged in `PROGRESS.md` |
| O3.1–O3.2 | P3-05 (S5) | Owner reply, logged in `PROGRESS.md` |
| 4.01–4.03 | P4-02 | `theme.css` token test; policy test (no Archivo, no external asset URL); `design/` mockup-comparison harness self-test |
| 4.04–4.10 | P4-03 | `design/parts.spec.ts` (mockup comparison per part); workload unit tests; template policy test for unstyled controls |
| 4.11–4.12 | P4-04 | `design/icons.spec.ts`; template policy test for `title` + `aria-label` |
| 4.13–4.14 | P4-05 | `design/frame.spec.ts`; existing `frame/` suites |
| 4.15–4.17, 4.21 | P4-06 | `design/board.spec.ts`; `tasks/board.spec.ts` |
| 4.18–4.20 | P4-07 | `tasks/card-actions.spec.ts`, including a no-JavaScript run |
| 4.22, 4.25, 4.26, 4.28 | P4-08 | `tasks/window-create.spec.ts`, including a no-JavaScript run |
| 4.23, 4.24, 4.27, 4.29 | P4-09 | `tasks/window-read-edit.spec.ts` |
| 4.30–4.32 | P4-10 | `design/myjobs-team.spec.ts`; existing `tasks/` suites |
| 4.33–4.38 | P4-11 | `design/calendar.spec.ts` (including the three-distinct-day-shades assertion for 4.34) |
| 4.39–4.43 | P4-12 | `design/kb.spec.ts`; existing `kb/` suites |
| 4.44–4.47 | P4-13 | `design/briefing.spec.ts`, `design/picker.spec.ts` |
| 4.48–4.51 | P4-01 | `internal/tasks` unit tests (written failing-first); `tasks/create-assigned.spec.ts` |
| 4.52–4.54 | P4-14 | Full suite at every layer; std checks; `speed/` with the test library |
| O4.1–O4.4 | P4-15 (S9) | Owner reply, logged in `PROGRESS.md` |
| 5.01–5.31, O5.1–O5.4 | P5-01–P5-09 | See `PLAN-v1.2.md` §5, which carries its own gate-to-task table |
| 6.01–6.41, O6.1–O6.4 | P6-01–P6-08 | See `PLAN-v1.3.md` §5, which carries its own gate-to-task table |

---

## 7. Planned `docs/decisions.md` entries (Part B adjustments and choices)

The named task writes each entry. Entries marked **(adjusts Part B)** change SPEC Part B's direction; the rest fill details Part B left open.

| ID | Task | Decision |
|---|---|---|
| D-01 | P1-02 | **(adjusts Part B)** The app is named Comp HQ (owner, 14 Sep 2026); all identifiers `comphq`/`COMPHQ_*`. SPEC.md was updated accordingly. |
| D-02 | P1-06 | **(adjusts Part B)** Final image `gcr.io/distroless/static-debian13:nonroot`, pinned by digest (SPEC said `distroless/static`). Compose `user: "568:568"` overrides the image user. Build image pinned by digest. |
| D-03 | P1-05 | **(adjusts Part B)** CI runs layers 1, 2, 3 and 6 on every push; layers 4, 5 and 7 on relevant paths, `[container]`/`[speed]` flags, full dispatch and tags. **Every release runs everything.** Reason: GitHub Free private repos get 2,000 minutes/month on 2-vCPU runners. |
| D-04 | P1-05 | npm scripts are the single command surface; Go HTTP tests use the `integration` build tag. |
| D-05 | P1-09 | TipTap 3.31.3 bundle via esbuild 0.28.2 IIFE; headings 2/3 only; `injectCSS: false`; table column resizing off (CSP). |
| D-06 | P2-03 | SortableJS 1.15.7 vendored for board and lane dragging; how Playwright drives it. |
| D-07 | P1-11 | `.docx` fixtures come from a committed Go generator, not LibreOffice (no heavy CI dependency; byte-reproducible). Realism is covered by Word 365 and Google Docs structures, owner samples, and O1.4. |
| D-08 | P1-06 | Internal pre-release tags `v0.0.1`/`v0.0.2` exist so the upgrade and rollback test proves 1.42 before `v0.1.0`. |
| D-09 | P1-14 | Fonts (OFL) and colour tokens; contrast adjustments to signal orange if needed. |
| D-10 | P1-08 | FTS5 query shape, marker characters, ranking SQL, and any modernc caveats. |
| D-11 | P1-09 | *Only if needed:* CSP relaxation for editor styles, with exact reason. |
| D-12 | P1-10 | Paste and drop testing method in headless Chromium. |
| D-13 | P1-12 | **(adjusts Part B)** Offline test uses an `internal: true` network plus a test-mode egress probe. Upgrade and rollback runs each tag's own smoke tests (`@seed`, `@verify`, `@verify-prev`) from a git worktree. |
| D-14 | P1-27 | Seeder design and timing method for speed tests. |
| D-15 | P2-07 | Change counters for `…/version`; tests use Playwright `page.clock` instead of 65s real waits. |
| D-16 | P1-04 | E2E tests must pass against a shared populated server; `@fresh` exceptions are listed. |
| D-17 | P1-09 | Test-mode-only routes under `/__test/` (editor harness, route list, egress probe, backup-age helper) are registered only when `COMPHQ_TEST_MODE=1`. A policy test asserts `deploy/truenas.yaml` never sets it, and an integration test asserts the routes return 404 without it. |
| D-18 | P1-39 | O1.3 (HelpScout move) may finish during Phase 2, if the owner chooses. |
| D-19 | P3-01 | Briefing labels ("in N weeks" when a multiple of 7 days, else "in N days") and stamp rules. |
| D-59 | P4-01 | Where an assigned idea sits (My jobs "Ideas I'm on", a Team **Ideas** heading) and that it never counts towards workload blocks. |
| D-60 | P4-02 | **(adjusts Part B)** The accent contrast rule of SPEC B9.2, superseding D-09: the bright `#FF6B1A` is a fill only with ink navy on it (6.02:1); `#A94000` is the only orange used as text (5.54:1); white on the accent (2.85:1) is banned. Also the font versions and subsetting method. |
| D-61 | P4-03 | How the date field is made day-first rather than following the computer's locale. |
| D-62 | P4-08 | The `<dialog>` approach for the task window, and how the no-JavaScript path is kept honest. |

---

## 8. Risks

| Risk | How the plan reduces it | Fallback |
|---|---|---|
| **Editor bundle** (TipTap under a strict CSP; paste, drop, tables) | Spike P1-09 comes before any KB page. The bundle is reproducible and checksummed. `injectCSS: false`; no column resizing. | Allow `style-src 'unsafe-inline'` only for styles (D-11), or drop to TipTap's core packages without the kit. The editor is isolated in one vendored file plus `editor.js`. |
| **Word import fidelity** | Converter built rule by rule on a realistic generator (P1-11), including localised styles and Google Docs output; fuzzed; owner samples tested automatically when present; O1.4 checks a real document. | Owner feedback becomes fix tasks with a new generator case per issue (§5.3). Unsupported items are always visibly marked, never silently lost. |
| **Flaky timing tests** (60s refresh, 3s paste, 10s import, speed) | `page.clock` for polls; `focus` events; `data-ready` waits; unique data; paste method proven 10/10 in CI (P1-10); speed uses the median of 5. | Fix the cause (rule 2.5). No retries, and no raised limits. |
| **Speed on 2-vCPU CI runners** | Seeder and KB speed tests come early (P1-27), while design changes are cheap; Tasks and Calendar speed in P2-12/P2-17. | Indexes, lighter templates, pagination of Finished/Removed lists. Limits stay fixed. |
| **ghcr and TrueNAS pulls** | Exact version tags; anonymous-pull check after S2 and at each release; standard Compose keys only; README troubleshooting. | README: if TrueNAS can't pull, check the package is still Public and the version number matches exactly. Rollback is changing the number back. |
| **CI minutes run out** | D-03 tiers, local `npm run check` before push, one push per task, caching, cancel-in-progress. | S8: wait for the monthly reset (recommended), or the owner makes the repo public. |
| **No Docker locally** | Container tests are CI-only by design; container changes carry `[container]`. | None needed. If Docker Desktop later runs, the script also works locally from Bash. |
| **Owner's time zone or port** | Marked lines in `deploy/truenas.yaml` plus README steps; the app refuses to start with a plain log line if `TZ` is invalid. | README "It won't open": check Apps logs, fix `TZ`, or change the host port. |
| **SQLite file on NAS storage** | WAL on a local dataset, host path mount (not SMB/NFS), `busy_timeout`, and a single writer process. | Nightly `VACUUM INTO` copies plus TrueNAS snapshots (README restore steps). |
