# Comp HQ — Decisions log

Non-obvious choices made while building Comp HQ, in the format required by `PLAN.md` §2.4: one short paragraph each, headed `D-NN Title (task ID, date)`.

## D-01 Name and identifiers (P1-02, 2026-09-16)

The app is named Comp HQ (owner, 14 Sep 2026). All identifiers use `comphq` / `COMPHQ_*` (repository `stas-comp/comphq`, binary `comphq`, container image `ghcr.io/stas-comp/comphq`, environment variables prefixed `COMPHQ_`). `SPEC.md` was updated accordingly.

## D-02 Single SQLite connection (P1-03, 2026-09-16)

`internal/db.Open` sets `sqlDB.SetMaxOpenConns(1)`. SQLite allows only one writer at a time; a single shared connection is the simplest way to avoid "database is locked" errors, at the cost of serialising reads too. Revisit if the layer 5 speed tests (P1-27) show read contention — a common fix is a second, read-only connection pool.

## D-03 Embedding at the repository root (P1-03, 2026-09-16)

`go:embed` patterns can't reach outside the directory of the file that declares them, but `web/` and `migrations/` are siblings of `internal/` and `cmd/` (SPEC B2's layout). So the embed directives live in a small package at the repository root (`assets.go`, `package comphq`), imported by `internal/app` and `cmd/comphq`. Every later section's templates and migrations are covered automatically since the patterns are `all:web/templates` and `all:migrations`.

## D-16 Sanitiser link/image rules run as a custom pass after bluemonday (P1-18, 2026-09-16)

`internal/kb/html` uses `bluemonday` for the structural allowlist (tags and simple attributes like `colspan`/`rowspan`) but not for its own URL scheme/path checks. bluemonday's `RequireParseableURLs`/`AllowURLSchemes` validate `href` and `src` the same way across every element and reject relative URLs unless `AllowRelativeURLs(true)` is set globally — there's no way to say "keep the `<a>` tag but drop a `javascript:` href" (its default is to unwrap the whole element) while separately saying "drop the whole `<img>` unless `src` starts with `/images/`" (an attribute-value regex on `src` still leaves `alt` behind, so a bare `<img alt="...">` survives instead of disappearing). So `Sanitize` lets bluemonday allow `href`/`src`/`alt` unconditionally, then runs its own tree walk afterward: `fixLink` strips `href` unless its scheme is http/https/mailto and always adds `rel="noopener"`; `pruneInvalidImages` removes the `<img>` node outright unless `src` matches `^/images/`.

## D-12 Paste/drop testing method (P1-10, 2026-09-16)

Chose a synthetic `ClipboardEvent`/`DragEvent` with a hand-built `DataTransfer` (approach (b) in the plan) over the real-clipboard approach (`context.grantPermissions(['clipboard-read','clipboard-write'])` + `navigator.clipboard.write`). It needs no OS clipboard or extra browser permissions, so it isn't sensitive to headless Chromium's clipboard support varying between Windows (dev) and Linux (CI). `e2e/helpers/paste.ts` exports `pasteFile`, `pasteHTML` and `dropFile`; all three ran green 40/40 (`--repeat-each=10` across the 4 paste/drop specs) locally before being committed at the normal `repeat-each=1`. The oversize-image fixture (`e2e/helpers/images.ts`) generates a real ~21–23MB greyscale PNG at test time with a small hand-rolled PNG writer (zlib deflate + CRC32) rather than committing one or depending on an image library.

## D-15 Section registry shape (P1-14, 2026-09-16)

`internal/app.Section` is a small struct (`MigrationName`, `Nav *NavItem`, `RegisterRoutes func(*http.ServeMux)`), not an interface — a section package exposes one `Section(srv *app.Server) app.Section` function, and `cmd/comphq/main.go` (which is the only place that may import every section, per SPEC B2 boundaries) adds one line per section: `srv.Registry().Add(briefing.Section(srv))`. `internal/app` itself only ever imports `internal/people` (an elevated, foundational dependency alongside `db` and `app` itself, per SPEC B2's boundary list), never a content section — the reverse dependency (section imports `app` to build handlers) would be a cycle otherwise. This supersedes D-14's "direct wiring" for every section except People and the app's own migration, which `NewServer` still registers itself since they're foundational rather than content sections.

`/` redirects to `/kb` in Phase 1 (SPEC P1-14), so `/kb` needs *something* to redirect to before the real Knowledge Base lands in P1-17 — it gets the same "Coming soon" placeholder as Briefing/Tasks/Calendar, to be replaced the moment P1-17 builds the real page (the one allowed test change, SPEC B7 rule 5). Settings gets a nav entry now (SPEC gate 1.07 needs it to show) but no route yet — P1-15 is the very next task and adds it.

## D-09 Fonts, colour tokens, contrast adjustments (P1-14, 2026-09-16)

Headings: Archivo (variable, weights 100–900 in one file). Body: Atkinson Hyperlegible Next (variable, same). Both OFL-1.1, fetched from Google Fonts' own hosting (`fonts.gstatic.com`, not linked live — the files are vendored under `web/static/theme/fonts/` with their own `VENDOR.md`, generalising the P1-09 checksum mechanism to more than one asset group).

Colour tokens (`web/static/theme/theme.css`): ink-navy `#14213d` (sidebar), signal-orange `#c2410c` (accent), paper `#fbf6ee` (reading background). The orange is deliberately darker than a "pure" bright orange (which would read closer to `#e8590c`): computing WCAG relative luminance by hand, white text on `#e8590c` only reaches ~3.6:1 (fails the 4.5:1 AA text threshold), while white on `#c2410c` reaches ~5.2:1. Orange text directly on the navy sidebar would only reach ~3.1:1 (large-text/non-text territory, not body text) — SPEC A10's own fallback ("use it for fills and borders, with dark or white text") is exactly what's used: the current-section nav item is an orange *fill* with white text, and the picker's error message is white text on a darker orange chip, never orange text on navy. Orange text on the light paper background (e.g. links) does clear 4.5:1, so that combination is used freely in body content. All of this is now proven, not just computed: every page type passes its own axe (contrast included) check in CI.

## D-14 Direct section wiring before the registry exists (P1-13, 2026-09-16)

SPEC B2 wants sections to register their own routes, nav entry and migrations with `internal/app`, and that registry is built in P1-14. Since People (P1-13) lands first, `cmd/comphq/main.go` lists section names directly (`for _, section := range []string{"app", "people"}`) and `internal/app.Server` builds a `people.Handlers` and mounts its routes itself, rather than through a registry that doesn't exist yet. P1-14 should fold this into the real registry rather than leaving two wiring styles side by side.

Middleware order is security headers → `OriginCheck` (SPEC B4 request safety) → the person-identity middleware (SPEC B4 person identity) → the route mux. A request needs to pass the origin check before the identity middleware ever renders the picker for it. Both middlewares exempt `/who`, `/healthz`, `/static/` and `/__test/` so the picker itself, health checks, assets and test-only harness routes are always reachable.

`POST /__test/people/deactivate` (test-mode only) calls the same `Store.Deactivate` that Settings → People (P1-15) will use, so gate 1.06 ("removing the current person shows the picker next time") is testable now — the same pattern as the P1-06 container-test marker file and the P1-12 offline-egress route.

## D-13 Offline test, upgrade/rollback method, smoke tag scheme (P1-12, 2026-09-16)

**Offline:** `deploy/test/offline.override.yaml` puts the app container on an `internal: true` network and adds a `runner` service (the digest-pinned `mcr.microsoft.com/playwright:v1.63.0-noble` image, on that same network only, with the repo mounted read-only via `${REPO_ROOT}`) that runs `npx playwright test --project=smoke` with `BASE_URL=http://comphq:8080`. `container-test.sh` also curls the test-mode-only `GET /__test/egress` (added to `internal/app`) from the host — published ports still work on an `internal: true` network (only the container's own outbound route is blocked), so this proves the block from outside the container too, not just via the smoke suite.

**Upgrade/rollback:** `PREV` is simply the latest existing release tag — during ordinary development HEAD is always untagged, so the plan's fallback ("the latest tag if HEAD is untagged") is what actually applies in practice. `git worktree add` checks it out; each phase runs *that tag's own* `e2e/smoke` tests (with its own `npm ci`), per PLAN.md's "runs each tag's own smoke tests ... from a git worktree." Since `v0.0.1` predates this harness and has no smoke tests at all, `container-test.sh` skips smoke assertions for a phase when the checked-out tag's `e2e/smoke/` is empty, rather than failing on "no tests found" — expected only until `v0.0.2` (P1-28). The pre-update-backup assertion is deferred to P1-34 (backups don't exist yet) and left as a TODO in the script.

**Smoke tag scheme:** `@seed`, `@verify`, `@verify-prev`, exactly as PLAN.md names them. Playwright's `--grep` does substring-style regex matching, so a plain `@verify` pattern also matches `@verify-prev` — fixed with a negative lookahead (`@verify(?!-prev)`) rather than renaming the tags away from the plan's wording.

## D-07 Fixture generator instead of LibreOffice (P1-11, 2026-09-16)

`.docx` fixtures (`e2e/fixtures/docx/{sample,googledocs,big-20pages}.docx` and `bad/*`) come from a committed Go generator (`e2e/fixtures/docx/gen`, stdlib `archive/zip` only, fixed zip timestamps), not LibreOffice headless — no heavy CI dependency, and byte-reproducible (checked in CI via re-run + `git diff --exit-code`, the same pattern as the P1-09 editor bundle). `sample.docx` hand-builds realistic WordprocessingML for every SPEC B4 mapping rule at once (localised style IDs, `basedOn` inheritance, nested lists, a hyperlink and a legacy HYPERLINK field, a table with `gridSpan`/`vMerge`, three images, a chart, SmartArt as `mc:AlternateContent`, a shape, an equation, an EMF picture, a text box, tracked insert/delete, a comment, a footnote, and a header/footer); `googledocs.docx` is deliberately smaller, using Google Docs' un-localised style IDs, to prove the converter isn't secretly tied to Word's own conventions. Realism beyond that is checked against the owner's own samples (`samples/word/`, N1) and the O1.4 owner check, not against these self-made fixtures.

`internal/kb/docx.Convert` reads any zip part in two passes: first discarding it through a size-limited reader to find its true decompressed length (bounded, constant memory — this is what lets `zipbomb.docx`, a real 320MB-of-zeros bomb compressing to ~330KB, be rejected using well under the plan's 64MB budget), then actually reading it only once confirmed within the 300MB package-wide limit. XML depth (max 256) is checked the same way, via a first pass over raw tokens, before ever calling `xml.Unmarshal`.

## D-05 Editor bundle: packages, bundling, CSP (P1-09, 2026-09-16)

TipTap 3.31.3 (`@tiptap/core`, `@tiptap/starter-kit`, `@tiptap/extension-table`, `@tiptap/extension-image`, `@tiptap/extension-file-handler`) bundled once via `scripts/vendor-editor/build.mjs` (esbuild 0.28.2, IIFE, minified, no sourcemap) into `web/static/vendor/editor/editor-3.31.3.js`, reproducible byte-for-byte (checked in CI by re-running and `git diff --exit-code`). StarterKit is configured down to the SPEC A5 toolbar: headings `[2, 3]` only, link protocols `http`/`https`/`mailto`; `blockquote`, `code`, `codeBlock`, `horizontalRule`, `strike` and `underline` are turned off since they're not in the toolbar. Table is `resizable: false` (no per-column inline width styles to fight the CSP over). `injectCSS: false`; base ProseMirror styles live in `web/static/kb/editor.css` instead. **D-11 was not needed**: the P1-09 E2E test drives every toolbar command (headings, marks, lists, link, table insert/add/remove row/column, image) plus a synthetic paste under the real production CSP (`default-src 'self'; img-src 'self' data: blob:; object-src 'none'; frame-ancestors 'none'`) with a `securitypolicyviolation` listener attached before the page's own scripts run, and zero violations were reported — no relaxation required.

## D-10 FTS5 query shape and markers (P1-08, 2026-09-16)

Confirmed `modernc.org/sqlite` v1.58.0 supports FTS5 `bm25()`, `snippet()` and `highlight()` identically to upstream SQLite, on Windows (dev) and Linux (CI) — no caveats found. Query building (`internal/kb/search.BuildQuery`): extract letter/digit runs only (`[\p{L}\p{N}]+`), discarding everything else, so raw user input can never inject FTS5 operator syntax; quote every run (`"word"`), which also defuses bareword operators like `AND`/`OR`/`NEAR`; suffix the last run with `*` for a prefix match — FTS5 allows `*` directly after a quoted string's closing quote for exactly this. Fewer than 2 significant characters yields no query at all (`ok = false`), rather than asking FTS5 to match everything or erroring. Snippet/highlight markers are the private-use characters U+E000/U+E001: they can't appear in real content and pass through `html.EscapeString` untouched, so the safe order is always escape-the-text-first, then replace markers with `<mark>`/`</mark>`. One gotcha: `snippet()` supports column index `-1` ("pick the column with the most matches"), but `highlight()` does not — `HighlightBlock` picks the column itself (0 for the title block, 1 for a body block) since SPEC B3 already tells us which one has content per row.

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
