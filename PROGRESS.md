# Comp HQ — Build Progress

Build agent: follow `PLAN.md` §2. Work on the first row that isn't `done`. Keep this file accurate at the end of every session.
Status values: `todo` · `in-progress` · `blocked` · `done`.

## Tasks

| ID | Title | Status | Commit(s) | CI run | Notes |
|---|---|---|---|---|---|
| P1-01 | Local tooling | done | | | go1.27.0, node v24.19.0, npm 11.17.0, git 2.37.1, gh 2.101.0; docker engine: not running (expected, container tests run in CI only). Installed via winget. Note: this machine's terminal tools (used by the build agent) don't pick up a freshly-installed program's PATH until the agent's own session restarts, so each command in this session that needs go/node/npm/gh explicitly adds their folders to PATH first; this doesn't affect the owner's own use of the computer. |
| P1-02 | GitHub sign-in and repository (STOP S1) | done | 90a9caf | | Repo stas-comp/comphq created (private), pushed to main. D-01 recorded. |
| P1-03 | Go skeleton: binary, config, health, migrations runner | done | 759f02f | no CI yet (added in P1-05) | go vet + go test and integration tests pass locally; `go run ./cmd/comphq` serves `/` with `<body data-ready>` and `/healthz` returns 200. D-02, D-03 recorded. |
| P1-04 | Playwright and axe skeleton | done | 0939f46 | no CI yet (added in P1-05) | `npm run check` passes locally from both PowerShell and Bash. Added placeholder `tools/policy` package so the command surface works before P1-05 (see D-04a). |
| P1-05 | CI workflow and repository policy tests | done | d8cd122 | https://github.com/stas-comp/comphq/actions/runs/35113839555 | First CI run (12f0ef7) and the follow-up PROGRESS.md update (d8cd122) both green: `go` and `browser` jobs pass; `container`/`speed`/`screens` correctly skipped (no flag/tag). `.github/workflows/ci.yml` added. Real `tools/policy` checks replace the P1-04 placeholder. D-03, D-04 recorded. |
| P1-06 | Dockerfile, TrueNAS YAML, container tests, tag v0.0.1 | done | 2e8094b | https://github.com/stas-comp/comphq/actions/runs/35115590961 | Container job green (2 fixes needed: sudo for uid-568-owned files; excluded SQLite -wal/-shm from the data checksum, see docs/decisions.md). Full dispatch green (run 35116031926). Tag v0.0.1 pushed; both CI (35116427243) and Release (35116427158, incl. publish) green. CHANGES.md has the 0.0.1 entry. D-02, D-04b, D-06a, D-06b, D-08 recorded. |
| P1-07 | Make the image public (STOP S2) | done | c3028bc | | Owner made the ghcr.io package public. Verified anonymously: token request + HEAD/GET on the 0.0.1 manifest both return 200 with no auth header. |
| P1-08 | Spike: FTS5 search with block ids | done | 7263076 | https://github.com/stas-comp/comphq/actions/runs/35126541585 | `internal/kb/search` (query builder, ranking, snippet/highlight) and `migrations/kb/0001_search.sql` added. All 6 unit tests green on Windows locally and Linux CI. D-10 recorded. |
| P1-09 | Spike: editor bundle | done | (pending) | (pending) | TipTap 3.31.3 bundled via esbuild (`web/static/vendor/editor/editor-3.31.3.js`, reproducible, checked in CI). Added global CSP/Referrer-Policy headers, `/static/` file serving, and the `COMPHQ_TEST_MODE`-gated `/__test/editor` harness. 12 E2E tests green (every toolbar command, paste callback, zero CSP violations). D-05 recorded; D-11 not needed. |
| P1-10 | Spike: clipboard paste in headless Chromium (send N1) | todo | | | |
| P1-11 | Spike: Word converter and fixture generator | todo | | | |
| P1-12 | Spike: offline test and upgrade/rollback harness | todo | | | |
| P1-13 | People: picker, add name, cookie | todo | | | |
| P1-14 | Frame, theme, section registry, Coming soon, 404, headers | todo | | | |
| P1-15 | Settings shell: People, About, Set up this computer | todo | | | |
| P1-16 | README part 1 | todo | | | |
| P1-17 | Categories and Knowledge Base home | todo | | | |
| P1-18 | Sanitiser and block ids | todo | | | |
| P1-19 | Article store, publish, read, versions, search rows | todo | | | |
| P1-20 | Editor page: toolbar, publish/cancel, leave warning | todo | | | |
| P1-21 | Image store, upload, paste and drop | todo | | | |
| P1-22 | Paste clean-up for web pages and Word | todo | | | |
| P1-23 | External images copied on publish | todo | | | |
| P1-24 | History, restore, archive | todo | | | |
| P1-25 | Edit conflicts | todo | | | |
| P1-26 | Search: live panel, results page, scroll-to-passage | todo | | | |
| P1-27 | Test-library seeder and Knowledge Base speed tests | todo | | | |
| P1-28 | Internal release v0.0.2 | todo | | | |
| P1-29 | Word converter: text, styles, headings, title, revisions, fields | todo | | | |
| P1-30 | Word converter: lists and links | todo | | | |
| P1-31 | Word converter: tables, text boxes, alternate content | todo | | | |
| P1-32 | Word converter: pictures, unsupported items, limits, unreadable files | todo | | | |
| P1-33 | Import from Word in the editor | todo | | | |
| P1-34 | Nightly backups and banner | todo | | | |
| P1-35 | Export everything | todo | | | |
| P1-36 | Content check | todo | | | |
| P1-37 | README part 2 | todo | | | |
| P1-38 | Phase 1 hardening and full gate sweep | todo | | | |
| P1-39 | Release v0.1.0 and Phase 1 report (STOP S3) | todo | | | |
| P2-01 | Tasks data, Board columns, add task, cards | todo | | | |
| P2-02 | Ordering engine and move buttons | todo | | | |
| P2-03 | Dragging on the Board | todo | | | |
| P2-04 | Task details dialog and Activity | todo | | | |
| P2-05 | Board filters | todo | | | |
| P2-06 | Finished tasks and Removed tasks | todo | | | |
| P2-07 | Refresh without live sockets; removed people on tasks | todo | | | |
| P2-08 | Team view: lanes and workload | todo | | | |
| P2-09 | Team view: assigning and reordering | todo | | | |
| P2-10 | My jobs: the Tasks home, and Take it | todo | | | |
| P2-11 | My jobs: take conflicts, moving jobs on, Give back | todo | | | |
| P2-12 | Seeder and speed for Board, Team and My jobs | todo | | | |
| P2-13 | Recurrence engine | todo | | | |
| P2-14 | Events: add and edit, month view, list view | todo | | | |
| P2-15 | Repeating events: just this one, change all, cancel, remove | todo | | | |
| P2-16 | Task deadlines on the calendar | todo | | | |
| P2-17 | Calendar speed, and Tasks/Calendar accessibility and size sweep | todo | | | |
| P2-18 | Export: tasks and events spreadsheets | todo | | | |
| P2-19 | README for Phase 2 and full Phase 2 sweep | todo | | | |
| P2-20 | Release v0.2.0 and Phase 2 report (STOP S4) | todo | | | |
| P3-01 | Briefing rules as a pure function | todo | | | |
| P3-02 | Briefing page | todo | | | |
| P3-03 | Briefing refresh, speed and checks | todo | | | |
| P3-04 | Final README and full sweep | todo | | | |
| P3-05 | Release v1.0.0 and final report (STOP S5) | todo | | | |

## Stop log

| Date | Stop point | Message sent | Owner reply |
|---|---|---|---|
| 2026-09-15 | S1 (P1-02) | GitHub sign-in instructions sent in chat | done (confirmed 2026-09-16) |
| 2026-09-16 | S2 (P1-07) | Make-the-package-public instructions sent in chat | done |

## Owner feedback

| Date | Phase / check | Owner's words | Fix task(s) | Status |
|---|---|---|---|---|
