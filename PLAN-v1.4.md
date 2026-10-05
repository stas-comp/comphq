# Comp HQ — Build Plan for v1.4

**Phase 7 · gates 7.01–7.91 · SPEC A11 "Phase 7" and B13 · decisions D-89–D-100**
**Written 5 October 2026. The owner approved the spec the same day.**

This plan is built to run **unattended for about a week**. The owner is away and won't answer
messages. §2 says how the build agent works without them. It overrides the stop points in
`PLAN.md` §2.7 and §5.1 for this phase only.

---

## 1. For the owner (please read this part)

v1.4 is everything on the staff's list: eight fixes, weekly jobs, linked jobs, and a tidier
Knowledge Base. `SPEC.md` (Phase 7) is the full list you approved.

### Before you leave (about 10 minutes)

1. **Keep the computer awake and on the internet.** Plug it in. Go to Windows **Settings → System
   → Power** (or **Power & battery**) **→ Screen and sleep**. Set **"When plugged in, put my device
   to sleep after"** to **Never**. If the Claude app has a "keep computer awake" option, turn it on.
2. **Stop Windows restarting itself.** Go to **Settings → Windows Update → Pause updates** and choose
   **1 week**. A restart would end the build session.
3. **Optional:** put the Word document that lost pictures in the **samples\word** folder inside the
   Staff HQ folder (nothing private).
4. **Start the build.** Open a new Claude Code session in the Staff HQ folder. Choose **Sonnet 5.5**,
   set the permission mode to the one that **doesn't ask for approval** (auto), then paste the prompt
   from `docs/v1.4-build-prompt.md` (everything below its line) and press Enter.

### While you're away

The agent works through the tasks in §5 in order and never waits for you. Anything that needs you
goes into one file, **`reports/phase-7-owner-inbox.md`**, and the agent carries on with everything
that doesn't depend on it. If it's stuck on one task, it writes down why and moves on to the next.

### When you're back

1. Open **`reports/phase-7-report.md`**. Its first lines tell you whether v1.4.0 is published, and
   what's done.
2. Open **`reports/phase-7-owner-inbox.md`**. It lists anything waiting for you, each with a
   recommendation.
3. If v1.4.0 is published: in TrueNAS, go to **Apps → comphq → Edit**, change the number after
   `comphq:` to `1.4.0`, and click **Save**.
4. Try the six owner checks (O7.1–O7.6). The report has the steps.
5. If the session stopped early (for example after a power cut), open Claude Code in the Staff HQ
   folder on Sonnet 5.5 in auto mode, and paste the short **resume prompt** at the bottom of
   `docs/v1.4-build-prompt.md`.

---

## 2. Working without the owner (build agent: these rules override `PLAN.md` §2.7 and §5.1 for Phase 7)

Everything else in `PLAN.md` §2 applies unchanged: sessions, `PROGRESS.md`, the git workflow, the
definition of done, CI failures, commands and engineering rules.

### 2.1 Never stop, never wait
- **The owner isn't available until about 12 October 2026.** Don't send a stop-point message and
  wait. Don't end your turn with a question. Don't end your turn at all while any task in §5 can
  still be worked on.
- **Keep going until one of these is true:** P7-19 is `done`; or every remaining task is `blocked`
  or depends on a `blocked` task, and §2.5's second round is finished.
- Your last message is the summary from `reports/phase-7-report.md`, so it's the first thing the
  owner reads.

### 2.2 The owner inbox
`reports/phase-7-owner-inbox.md` replaces every stop-point message in Phase 7. Create it in P7-01
with the heading `# Waiting for the owner — Phase 7` and the line "Nothing yet." Then, for
anything that would have been S6, S7, S8 or a question, add a numbered entry:
- the date and task ID;
- **What happened**, in one or two plain sentences, with no jargon;
- **What I did meanwhile**;
- **What I recommend**, with options if there are any;
- **What you need to do**, click by click, or "nothing, just confirm".

Commit the inbox with the next task's commit (never a `[skip ci]` commit; see §2.6).

### 2.3 Direction questions: decide, record, carry on
For anything that would have been S6 (a gate that seems impossible or contradictory, or a choice
SPEC doesn't settle):
1. Pick the simpler option, and the one that's easier to undo (SPEC B8 rule 3, A2 priority 2).
2. Build it, and write a `docs/decisions.md` entry headed `D-NN … (provisional — owner to confirm)`.
3. Add an inbox entry.

**Never** add a paid service, a runtime internet dependency or a login, even provisionally. If a
gate seems to need one, skip that gate, mark it `blocked` in the task's Notes, add an inbox entry,
and finish the rest of the task.

### 2.4 When CI can't run
This is S8's situation. It applies when jobs don't start, GitHub reports a billing, spending-limit
or minutes problem, or `gh` stops being signed in.
1. Add an inbox entry (S8's wording, adapted). Don't change any account setting yourself, and never
   type a password or token.
2. Keep building. For each task: get `npm run check` green locally, commit, and push if pushing still
   works (the code is then safe on GitHub). Set the task `blocked` with the note **"CI only: local
   check green at `<sha>`"**.
3. A task whose only blocker is "CI only" **counts as done for Depends on**. Carry on with the next
   task.
4. At every task boundary, check whether CI works again (`gh run list --limit 1`). When it does,
   run a full dispatch (`gh workflow run ci.yml --ref main -f full=true`) on the latest `main`. If
   it's green, mark every "CI only" task `done` with that run's URL. If something fails, fix it under
   `PLAN.md` §2.5 as the earliest affected task.
5. Don't tag a release without a green full dispatch (P7-19).

### 2.5 Stuck on a task: park it, move on, come back once
This is S7's situation: 3 genuine attempts at the same failure (`PLAN.md` §2.5).
1. Set the task `blocked`. Write `reports/blocker-P7-NN.md`. Add an inbox entry.
2. Make sure `main` is green without it: either the failing work isn't pushed, or it's reverted
   with a normal revert commit. Never force-push, and never delete or weaken a test.
3. Move to the next task in §5 that doesn't depend on it. Many tasks are independent; "Depends on"
   says which.
4. **Second round:** after P7-18 (or when nothing else can move), return to each `blocked` task once,
   with a fresh approach. That allows up to 3 more attempts. Then leave it parked, and say so in
   the report.
5. If a parked task holds up only part of the release, see §2.7.

### 2.6 Using time well, and the pushes that waste it
- CI takes 15–25 minutes a push, and about 40 for a full dispatch. Watch a run in the background
  (`gh run watch <id> --exit-status`). **While it runs, you may start the next task locally, but
  don't push it until the current run is green.** A push cancels the run in progress on `main`
  (`concurrency: cancel-in-progress`), and that has cost whole cycles before.
- **Never push a `[skip ci]` commit together with code.** It skips CI for the *whole* push. Fold
  `PROGRESS.md` and inbox updates into the next task's commit, as the previous phases did.
- Run `npm run check` locally before every push. A task that touches `migrations/`, `cmd/`,
  `internal/db/`, `deploy/` or `Dockerfile` runs the container job automatically. Add `[speed]` to
  the commit message where the task says so.

### 2.7 Releasing with something parked
P7-19 releases v1.4.0 only when the full dispatch is green and every gate's test passes.
- If a task is parked and its gates are all from **one feature group** (for example linked jobs),
  that feature can be left out of v1.4.0 (D-89). Revert or hide it, so v1.4.0 has no half-built
  screens. Release the rest, and list the missing feature in the report and the inbox as "will
  follow in v1.4.1". The release's gate report lists those gates as **deferred**, never as passed.
- If what's parked is a fix that other gates rely on, or the "Still true afterwards" gates
  (7.90–7.91), **don't release**. Write the report anyway, and say exactly what's left.

### 2.8 After a context reset
Long sessions get summarised. After any summary or restart, re-read this §2, `PROGRESS.md`, and the
current task's entry before doing anything else. `PROGRESS.md` and the inbox are the memory; keep
both accurate at the end of every task.

### 2.9 Things on this machine that have cost time before
- This session **can't photograph the screen or open a visible browser**. Use Playwright-rendered
  screenshots. The look is for the owner checks.
- **Line endings.** `SPEC.md` is CRLF; most other files are LF. Python's text mode writes CRLF on
  Windows. Read and write with `newline=''`, keep each file's own endings, and never run a
  line-ending fixer over images (it corrupted the icons once; regenerate with `npm run icons`).
- Write TypeScript and regex-heavy files with the file-writing tool. Shell heredocs can lose
  backslashes.
- **Drag tests** need real pauses between mouse waypoints (SortableJS runs on timers). Reuse the
  drag helper from Phase 5, which waits for the drag to start and retries the gesture.
- **D-86:** don't build tests with a 1-second "not yet" margin on `page.clock`.
- **Tests on a shared, already-filled server** (`PLAN.md` §2.8): use uniquely named data, and assert
  relative order, never totals.

---

## 3. Before building: the spec — **done, 2026-10-05**

**P7-00** consists of these edits, all already in the working tree. The build agent commits them
with P7-01 if they aren't already on `main`.
- `SPEC.md`: A5, A6, a new **A11 Phase 7** (gates 7.01–7.91, O7.1–O7.6), A12, A13, B3, B4, B9.8,
  and a new **B13**.
- `docs/decisions.md`: D-89 to D-100.
- `PROGRESS.md`: P6-08 marked done, with an S11 row.
- `PLAN.md`: §5.1 and §6 point at this file.

`SPEC.md` is the contract. **B13 is the technical direction for every task below**, with the
causes found in the v1.3.0 code, file by file. This plan doesn't repeat it, so the two can't drift
apart. Where this plan and SPEC disagree, SPEC wins, and you write a decision entry saying so.

---

## 4. Gates

Gates 7.01–7.91 and owner checks O7.1–O7.6 are in `SPEC.md` A11 "Phase 7". Every gate is proved by
an automatic test whose title starts with its number (`PLAN.md` §6's naming rule), checked as
SPEC B13.11 describes. The owner checks are **not** waited for in this phase. They go in the
report for the owner's return.

---

## 5. Tasks for Phase 7

The task format, definition of done and progress log are `PLAN.md` §2.2 and §2.4, with §2 above on
top. **Start by** adding rows `P7-01` … `P7-19` to `PROGRESS.md` with status `todo`, and P7-00 as
`done` (2026-10-05).

**Order.** Fixes come first (D-89), then the larger features. "Depends on" is the real constraint.
When a task is parked (§2.5), carry on with anything that doesn't depend on it.

### Which task proves which gate

| Gates | Task | Proved by (SPEC B13.11 has the detail) |
|---|---|---|
| 7.50–7.52 | P7-01 | Strengthened `no-side-scroll` helper on every page type; corner-pixel check; oversize-image article |
| 7.10, 7.11 | P7-02 | `Create` unit test; corrected E2E |
| 7.12–7.14 | P7-03 | Board drag E2E (dead zone, auto-scroll, filter kept) |
| 7.01–7.03 | P7-04 | My jobs E2E (drag and icons), Board order check |
| 7.04, 7.05 | P7-05 | Lane builder unit test; Team E2E |
| 7.06–7.09 | P7-06 | Team and My jobs E2E; menu-in-viewport test; no-JS E2E |
| 7.15–7.18 | P7-07 | Store unit tests; toast E2E; no-JS E2E |
| 7.20–7.26 | P7-08 | Query unit tests; E2E on every Tasks page; speed |
| 7.55–7.59 | P7-09 | Store unit tests; E2E; container rollback step |
| 7.60, 7.62–7.64, 7.67, 7.68 | P7-10 | `internal/kb/docx` unit fixtures; integration (every `src`); `samples/word` sweep; timing |
| 7.61, 7.65, 7.66 | P7-11 | Unit fixtures (group, canvas, VML group, Choice, EMF, header logo); E2E import |
| 7.31–7.37, 7.39 (reset half) | P7-12 | `ResetWeekly` table tests; 1.3.0-shaped DB integration test |
| 7.30, 7.38, 7.39 (survival half) | P7-13 | E2E (box, stamp, Briefing, History); export; container rollback step |
| 7.44–7.46, 7.48, 7.49 | P7-14 | Store unit tests (every refusal, the race); export; container rollback step |
| 7.40–7.43, 7.47 (+ 7.16 with links) | P7-15 | E2E from the window and the page; WAITING; no-JS; speed |
| 7.70–7.72, 7.77 | P7-16 | Design layers; axe; sizes; screenshots |
| 7.73–7.76, 7.77 | P7-17 | Design layers against the mockup's editor; the editor gates re-run; axe |
| 7.90, 7.91 | P7-18 | Full suite, every layer |
| O7.1–O7.6 | P7-19 | Listed in the report for the owner's return (not waited for) |

---

## Milestone 7.1 — The page and the Board

### P7-01 The page never scrolls sideways (the white areas)
- **Goal:** no blank strip beside or below any page. Every later task is then measured with the
  stronger check.
- **Gates:** 7.50, 7.51, 7.52.
- **Depends on:** P7-00
- **Touches:** `web/static/theme/theme.css` (`.visually-hidden`, `color-scheme`, image width),
  `web/static/tasks/team.css`, `web/static/kb/kb.css` and/or `editor.css` (image width),
  `e2e/helpers/no-side-scroll.ts`, `e2e/app/white-areas.spec.ts` (new). Also create
  `reports/phase-7-owner-inbox.md` (§2.2).
- **Behaviour:** SPEC B13.7. **First, write the failing test:** Team with 10 people at 1024 × 700,
  asserting `documentElement.scrollWidth <= clientWidth`. Run it against unchanged code and confirm
  it fails, then fix. If it *doesn't* fail, the reading in B13.7 is wrong: find the real cause with
  the same test on every page type, and record what you found.
- **Tests:** the helper asserts on both `documentElement` and `body`, on every page type at both
  sizes, with the test library loaded. A sideways wheel event over the Team lanes moves
  `.team-lanes.scrollLeft` and not `window.scrollX`. The corner-pixel check after scrolling each
  page to its bottom-right. An article with a 3,000-pixel-wide picture fits on the article page and
  in the editor.
- **Done when:** tests are green in CI.
- **Decisions:** D-97 (written). Add the measured cause, and anything else the sweep found.

### P7-02 New jobs start in To do
- **Gates:** 7.10, 7.11.
- **Depends on:** P7-00
- **Touches:** `internal/tasks/store.go`, `internal/tasks/new_handlers.go`,
  `internal/tasks/store_test.go`, `e2e/tasks/board.spec.ts`, and a new E2E for 7.11.
- **Behaviour:** SPEC B13.2 "Default column". The two existing tests that assert Ideas are
  **corrected, not deleted**. They name gate 7.10, and the report lists them under "Tests corrected"
  with D-90 as the reason.
- **Tests:** unit: `Create` with no stage → `todo`, at the bottom; an explicit `idea` still works.
  E2E: the add-a-task window opens with To do chosen; a job created with a person appears in their
  Up next on My jobs and Team, and their workload blocks grow by its size.
- **Done when:** tests are green in CI.
- **Decisions:** D-90 (written).

### P7-03 Drop anywhere in a column, scroll while dragging, keep the filter
- **Gates:** 7.12, 7.13, 7.14.
- **Depends on:** P7-01
- **Touches:** `web/static/tasks/board.css`, `board.js`, `team.css`, `team.js`, `myjobs.css`,
  `myjobs.js` (Sortable options only), `e2e/tasks/board-drag.spec.ts` (new).
- **Behaviour:** SPEC B13.2 "The dead zone", "Auto-scroll", "Filter kept". Use one shared set of
  Sortable options for every Tasks list, so P7-04 and P7-05 pick them up.
- **Tests:** SPEC B13.11's 7.10–7.14 row. Also re-run every existing drag test on the Board, My jobs
  and Team unchanged.
- **Done when:** tests are green in CI.
- **Decisions:** one new entry: native drag or `forceFallback`, what was measured, and the
  correction to the old note that Sortable uses pointer events.

---

## Milestone 7.2 — My jobs and Team

### P7-04 Reorder your own Up next; ideas into To do on My jobs
- **Gates:** 7.01, 7.02, 7.03.
- **Depends on:** P7-03
- **Touches:** `web/static/tasks/myjobs.js`, `web/templates/tasks/myjobs.html`, and possibly
  `internal/tasks/lanes.go` (the move-icon data is already there), `e2e/tasks/myjobs-order.spec.ts`
  (new).
- **Behaviour:** SPEC B13.1 "Reordering My jobs" and "Ideas into To do". No server change is
  expected. **Move to To do** posts the existing move endpoint.
- **Tests:** drag within Up next, then refresh and check a second browser context; the move icons;
  on the Board, only the moved job's relative position changes (as in gate 2.29's test); dragging an
  idea from Ideas I'm on into Up next at a chosen spot; the Move to To do button; History rows; both
  routes with JavaScript off where they're forms.
- **Done when:** tests are green in CI.
- **Decisions:** D-91 (written).

### P7-05 Team: unassigned ideas, and ideas into Up next
- **Gates:** 7.04, 7.05.
- **Depends on:** P7-03
- **Touches:** `internal/tasks/lanes.go` (and `lanes_test.go`), `web/templates/tasks/team.html`,
  `web/static/tasks/team.js`, `e2e/tasks/team-ideas.spec.ts` (new).
- **Behaviour:** SPEC B13.1 "Ideas into To do". Across lanes: move first, then assign. If either
  fails, refresh to the truth and show the plain message.
- **Tests:** unit: the Unassigned lane has an Ideas group, and its workload is unchanged by ideas.
  E2E: drag an idea into its own lane's Up next, into another person's (handed over, now To do, at
  the drop spot), and into an empty Up next; the Move to To do button; History; refresh in a second
  context.
- **Done when:** tests are green in CI.

### P7-06 People circles on Team and My jobs
- **Gates:** 7.06, 7.07, 7.08, 7.09.
- **Depends on:** P7-04, P7-05
- **Touches:** `internal/tasks/lanes.go` (assignee list), `internal/tasks/people_handlers.go` and
  `web/templates/tasks/people-menu.html` (redirect target), `web/templates/tasks/team.html` and
  `myjobs.html`, a shared script moved out of `board.js` (for example
  `web/static/tasks/people-menu.js`), `e2e/tasks/team-assign.spec.ts` (rewritten button test),
  `e2e/tasks/people-circles.spec.ts` (new). It's over five files, but each change is small and they
  belong together. Split into a/b only if it doesn't fit one session.
- **Behaviour:** SPEC B13.1 "People circles everywhere", including the clipping fix (7.07). The
  Board's own behaviour (gate 4.20) must not change, and its tests are re-run unchanged.
- **Tests:** SPEC B13.11's 7.01–7.09 row; add a second person from a person's lane (impossible at
  v1.3); remove one of two; the menu on the last card of the right-most lane, scrolled, at
  1024 × 700, with its box inside the viewport; no-JS; axe with the menu open on Team and on My
  jobs.
- **Done when:** tests are green in CI. The rewritten `team-assign` test is listed under "Tests
  corrected" (gate 2.28 wording, D-92).
- **Decisions:** D-92 (written), plus the menu-positioning choice.

---

## Milestone 7.3 — Undo and search

### P7-07 Undo after removing a job
- **Gates:** 7.15, 7.16, 7.17, 7.18.
- **Depends on:** P7-03
- **Touches:** `internal/tasks/lifecycle.go` (and tests), `internal/tasks/lifecycle_handlers.go`,
  `web/templates/tasks/board.html` (the toast, outside the refreshed fragment),
  `web/static/theme/theme.css` (`.toast`), `web/static/app/toast.js` (new), `board.js`,
  `e2e/tasks/undo-remove.spec.ts` (new).
- **Behaviour:** SPEC B13.3. Store the neighbour in the `removed` activity row's `detail` (new rows
  only; older removals simply fall back to their old position).
- **Tests:** SPEC B13.11's 7.15–7.18 row: undo into the same place; undo after the neighbour moved;
  undo after somebody else restored it (no error); people and steps intact; History shows both rows;
  the toast is gone after 10 s (`page.clock`, with comfortable margins, not D-86's); a screen
  reader live region; from the task page; no-JS; Removed tasks unchanged.
- **Done when:** tests are green in CI.
- **Decisions:** D-93 (written).

### P7-08 Search finds jobs on Tasks pages
- **Gates:** 7.20 to 7.26.
- **Depends on:** P7-00
- **Touches:** `internal/app/registry.go` and `frame.go` (per-section search setting),
  `web/templates/app/layout.html`, the search script (made data-driven), `internal/tasks/search.go`
  (new, with tests), `internal/tasks/section.go` (routes), `web/templates/tasks/search.html` (new),
  `e2e/tasks/search.spec.ts` (new), and the speed spec. Commit with `[speed]`.
- **Behaviour:** SPEC B13.4. The Knowledge Base search must behave byte-for-byte as before. Gates
  1.26–1.32 and 6.30–6.35 are re-run unchanged and must pass.
- **Tests:** SPEC B13.11's 7.20–7.26 row; also the `exclude=` parameter P7-15 will use, and the box
  reading "Search articles" again on a non-Tasks page.
- **Done when:** tests are green in CI, the speed job included.
- **Decisions:** D-94 (written).

---

## Milestone 7.4 — Knowledge Base fixes

### P7-09 Articles in order
- **Gates:** 7.55 to 7.59.
- **Depends on:** P7-00
- **Touches:** `migrations/kb/0006_article_order.sql` (new), `internal/kb/articles.go`,
  `history.go`, `export.go`, the start-up renumbering, `internal/kb/section.go` (route),
  `web/templates/kb/category.html` (move icons and drag only; the new look is P7-16), a small
  category script, `e2e/kb/article-order.spec.ts` (new), `e2e/smoke/` plus
  `deploy/test/container-test.sh` (the 7.59 step, tagged like D-85's `@search-…` pair).
- **Behaviour:** SPEC B13.8.
- **Tests:** SPEC B13.11's 7.55–7.59 row. The container step: on the rolled-back 1.3.0, publish an
  article into a category; after upgrading again, it's at the bottom of that category, and no
  category shows an article twice.
- **Done when:** tests are green in CI, the container job included (it runs automatically because
  `migrations/` changed).
- **Decisions:** D-98 (written).

### P7-10 Every picture from Word, part 1: the broken tenth picture and the silent losses
- **Gates:** 7.60, 7.62, 7.63, 7.64, 7.67, 7.68.
- **Depends on:** P7-00
- **Touches:** `internal/kb/import_docx_handlers.go`, `internal/kb/docx/document.go`, `images.go`,
  `docx.go`, `result.go` (and `docx_test.go`), `e2e/fixtures/docx/gen` (new fixtures, run
  `npm run fixtures:docx`, commit the output), the import integration test, and
  `e2e/kb/import-word.spec.ts`.
- **Behaviour:** SPEC B13.9 items 1, 5, 6, 7 and 8, plus gate 7.67's counting check. **Look in
  `samples/word` first.**
  - If it has documents, run the counting check on them before changing anything, and record which
    pictures go missing and why. That is the owner's real problem. If it's a cause B13.9 doesn't
    list, fix that too and record it.
  - If it's empty, add an inbox entry ("the owner's document wasn't provided, so the fixes are
    based on reading the code"), and carry on.
- **Tests:** the 25-picture fixture with each `src` hash asserted, in order (fails at v1.3.0; write
  it first); absolute, `%20` and upper-case targets; a picture in the title paragraph; a forced
  save failure becomes a placeholder plus a note; the counting check over every fixture and every
  `samples/word/*.docx`; gate 1.52's timing.
- **Done when:** tests are green in CI.

### P7-11 Every picture from Word, part 2: groups, canvases and old formats
- **Gates:** 7.61, 7.65, 7.66.
- **Depends on:** P7-10
- **Touches:** `internal/kb/docx/document.go` (drawing walk, `mc:Choice`, `scanVMLPict`),
  `images.go`, `docx.go` (the header note), the placeholder wording in templates or `messages.js`,
  `e2e/fixtures/docx/gen` (more fixtures), tests.
- **Behaviour:** SPEC B13.9 items 2, 3, 4, 9 and 10. The `drawing` placeholder kind carries 7.65's
  exact wording. Add it to the sanitiser's allowed `data-missing-kind` values, and to the content
  check (B4).
- **Tests:** unit fixtures: an anchored picture, a DrawingML group of 3, a canvas, a VML group of 3
  (all three kept, in order), a `v:fill` picture, a Choice-only picture, an EMF (the new wording),
  and a header logo (the note counts it). An E2E importing the group fixture shows three pictures.
- **Done when:** tests are green in CI.
- **Decisions:** D-99 (written). Add anything found in `samples/word`.

---

## Milestone 7.5 — Weekly and linked jobs

### P7-12 Weekly jobs, part 1: storage and the Saturday reset
- **Gates:** 7.31, 7.32, 7.33, 7.34, 7.35, 7.36, 7.37, and 7.39's reset half.
- **Depends on:** P7-02 (the reset puts jobs in To do)
- **Touches:** `migrations/tasks/0003_weekly.sql` (new), `internal/tasks/weekly.go` (new:
  `ResetWeekly`, set and unset), `store.go` and `lifecycle.go` (the 14-day and Finished rules),
  `details.go` (`activityText` for `repeat_on`, `repeat_off` and `weekly_reset`), the calls at
  start-up and in the Tasks, Briefing and version handlers, and tests.
- **Behaviour:** SPEC B13.5. `today` is always `app.Today`.
- **Tests:** SPEC B13.11's 7.30–7.39 row, as Go table tests: Friday→Saturday; Saturday run twice
  (second is a no-op); two missed Saturdays (one reset, due the latest); unfinished (unchanged,
  marked reset); removed (untouched); steps unticked; position at the bottom of To do; History text;
  the change counter bumped only when something changed; two concurrent calls. **Rollback half:** an
  integration test on a 1.3.0-shaped database holding a weekly job finished 20 days ago (tucked away
  on 1.3.0) shows `ResetWeekly` bringing it back.
- **Done when:** tests are green in CI, the container job included.
- **Decisions:** D-95 (written).

### P7-13 Weekly jobs, part 2: the tick-box, the stamp and the export
- **Gates:** 7.30, 7.38, and 7.39's survival half (also re-proves 7.31–7.37 end to end).
- **Depends on:** P7-12
- **Touches:** `web/templates/tasks/details.html` (`tasks-details-form`), the handlers that read the
  form, `web/static/theme/theme.css` (`.stamp.weekly`), the card templates (Board, My jobs, Team,
  Briefing), `internal/tasks/export.go`, `e2e/tasks/weekly.spec.ts` (new), `e2e/smoke/` plus the
  container script.
- **Behaviour:** SPEC B13.5 "Setting it" and "Everywhere else". The WEEKLY stamp follows B9's stamp
  rules: words, not an icon.
- **Tests:** E2E with `COMPHQ_TEST_TODAY`: tick the box on a new job (the due date becomes This
  Saturday); finish it; move the test date to the next Saturday and assert it's back at the bottom of
  To do, due that day, steps unticked, in the Briefing's Must be done today, with the History line.
  The stamp on all four screens. Untick. The export column. Container step: seed a weekly job on
  the current build, roll back (it still opens as an ordinary job), upgrade, and it's still weekly.
- **Done when:** tests are green in CI.

### P7-14 Linked jobs, part 1: storage, rules and export
- **Gates:** 7.44, 7.45, 7.46, 7.48, 7.49.
- **Depends on:** P7-00
- **Touches:** `migrations/tasks/0004_links.sql` (new), `internal/tasks/links.go` (new, with
  tests), `internal/tasks/section.go` (routes), `details.go` (`activityText`),
  `internal/tasks/export.go` plus the export wiring (`links.csv`), `e2e/smoke/` plus the container
  script.
- **Behaviour:** SPEC B13.6 "Storage", "Endpoints", "Rules" and "Export". The WAITING query is built
  here, as a store function with its own test, and used on screen in P7-15.
- **Tests:** SPEC B13.11's 7.40–7.49 row, store half: every refusal of 7.45, including a three-job
  loop and two contexts linking at once; a removed job hides its links and restore brings them back;
  unlinking on both sides; History on both jobs with the title kept after a rename; `links.csv`. The
  container step: links seeded on the current build survive a rollback and an upgrade.
- **Done when:** tests are green in CI, the container job included.
- **Decisions:** D-96 (written).

### P7-15 Linked jobs, part 2: on screen
- **Gates:** 7.40, 7.41, 7.42, 7.43, 7.47; also re-proves 7.16 (Undo keeps links).
- **Depends on:** P7-14, P7-08 (the picker uses job search), P7-07
- **Touches:** `web/templates/tasks/window-task.html` and `details.html` (the Linked jobs section),
  a small script for the picker, the card templates (the WAITING stamp), `theme.css`
  (`.stamp.waiting`), the Board, lane and My jobs handlers (one WAITING query per list),
  `e2e/tasks/links.spec.ts` (new). Commit with `[speed]`.
- **Behaviour:** SPEC B13.6 "Showing them" and "Picking a job". The remove-link control is gate
  4.11's **remove** icon, so the count of icon-only buttons stays at four.
- **Tests:** link with each of the three kinds from the window and from the page; both sides show
  it in the right group; WAITING appears on all three screens, lists its titles in the tooltip and
  spoken name, clears when the first job is Done, and comes back on Reopen; a waiting job can still
  be taken, moved and finished; no-JS linking and unlinking; Undo a removal and the links are still
  there; axe with the section open; speed with links seeded.
- **Done when:** tests are green in CI, the speed job included.

---

## Milestone 7.6 — A tidier Knowledge Base

### P7-16 Knowledge Base home and category pages
- **Gates:** 7.70, 7.71, 7.72, 7.77 (for these two screens).
- **Depends on:** P7-09
- **Mockup:** not drawn. Build these screens from the mockup's own parts (SPEC B13.10). Owner check
  **O7.5** covers them.
- **Touches:** `web/templates/kb/home.html`, `category.html`, `web/static/kb/kb.css`,
  `internal/kb/home.go` and the category handler (first three per category, opening words, the
  `?category=` preselect), `e2e/kb/home.spec.ts` and `e2e/kb/category.spec.ts` (new or extended),
  the screens spec.
- **Tests:** the content of each part (tile articles follow the order; the "All *n*" link; Recently
  updated still the 10 newest, gate 1.13; opening words cut at a word; New article preselects the
  category; empty states). B9.9's token and drift rules. axe. Both window sizes. Screenshots added.
- **Done when:** tests are green in CI.
- **Decisions:** D-100 (written).

### P7-17 The editor
- **Gates:** 7.73, 7.74, 7.75, 7.76, 7.77 (for the editor).
- **Depends on:** P7-11 (the `drawing` placeholder kind)
- **Mockup:** `docs/design/mockup.html`, `<section data-screen="editor">` and the CSS rules it uses
  (`.edit-head`, `.bar`, `.tb`, `.word`, `.imported`, `.title-in`, `.missing`). Read them before
  writing anything (B9.1).
- **Touches:** `web/templates/kb/editor.html`, `web/static/kb/editor.css`, `editor.js` (toolbar,
  table controls shown only in a table), `messages.js` ("Came across"), and the converter's counts if
  needed. Also `e2e/design/` (computed styles against the mockup) and the editor E2E.
- **Behaviour:** SPEC B13.10 "Editor". Form field names and every behaviour stay the same. Only the
  arrangement and look change.
- **Tests:** computed styles of each drawn part against the mockup (B9.9's first layer); the head
  and toolbar stay in view after scrolling a long article at 1024 × 700; the table controls appear
  only inside a table; placeholder look for every kind; the "Came across" line; **every gate from
  1.14 to 1.25 and from 1.45 to 1.52 re-run unchanged**; axe; screenshots.
- **Done when:** tests are green in CI.

---

## Milestone 7.7 — Finish

### P7-18 Full sweep: every gate, README, CHANGES, screenshots
- **Gates:** 7.90, 7.91.
- **Depends on:** every earlier P7 task that isn't parked
- **Touches:** `e2e/`, `reports/screens/`, `README.md`, `CHANGES.md`.
- **Behaviour:**
  - Run a full dispatch. Every gate from 1.01 to 6.41 must pass unchanged, apart from the planned
    corrections in 7.90, each pointing at its D-entry.
  - README, in plain words:
    - **Tasks:** new jobs start in To do; reordering My jobs; ideas into To do; the circles on
      Team; Undo; searching jobs; weekly jobs (what happens on Saturday); linked jobs (what WAITING
      means).
    - **Knowledge Base:** putting articles in order; the Word-picture tip for EMF/WMF.
  - Add screenshots: the Board with WEEKLY and WAITING stamps, the Undo message, the job search
    results, the Linked jobs section, and the three Knowledge Base screens.
  - Then do §2.5's second round on any parked task.
- **Done when:** one full CI run is green with every gate passing, or §2.7 says what's deferred.

### P7-19 Release v1.4.0 and the Phase 7 report — **no stop**
- **Gates:** O7.1–O7.6 (owner, listed for their return, not waited for); final confirmation of every
  gate.
- **Depends on:** P7-18
- **Steps:** `PLAN.md` §5.2 with version `1.4.0`, upgrading from and rolling back to `v1.3.0`,
  steps 1–7, but with these differences:
  - **Step 8 doesn't wait.** Commit the reports and push. Set this task to `done` once the release is
    published. The owner checks are logged under Owner feedback as `pending`.
  - If §2.4 or §2.7 prevents a release, don't tag. Write the report anyway: what's done, what's
    left, and the exact next step. Leave the task `blocked`.
- **The report** (`reports/phase-7-report.md`) starts with S12's summary below. Then it lists:
  - every gate with its test;
  - tests corrected (at least: 7.10's two, 7.06's `team-assign`, 7.50's helper);
  - new decisions, in plain words, including any marked provisional;
  - what was found in `samples/word`, or that it was empty;
  - deferred features, if any;
  - the six owner checks, each with what to try and what to expect.

  Say plainly that the Knowledge Base home and category pages have no mockup, so O7.5 matters
  most.
- **Done when:** v1.4.0 is published, or the report says exactly why not; and the report and inbox
  are committed and pushed.

---

## 6. The stop point (S12, report only)

Added to `PLAN.md` §5.1, after S11. In Phase 7 it is **written, not waited for**. It's the first
section of `reports/phase-7-report.md`, and the agent's final message:

> **S12 — end of Phase 7 (v1.4.0).** "Phase 7 is done and v1.4.0 is published *(or: what isn't
> yet, and why)*.
>
> - New jobs start in To do, and you can reorder your own list on My jobs.
> - Ideas can be dragged into To do on Team, and every card has the people circles.
> - Removing a job can be undone, and the search box on Tasks pages finds jobs.
> - Weekly jobs come back every Saturday, and jobs can be linked ("do first" or "related").
> - Articles can be put in order, Word pictures come across, and the Knowledge Base home, category
>   pages and editor have their new look.
>
> To update, change the version number in TrueNAS to `1.4.0` (see Updating in the README).
> *Inbox: n items waiting for you in `reports/phase-7-owner-inbox.md`, or none.* There are six
> things to check yourself: *O7.1–O7.6, each with what to try and what to expect.* The full report
> is in `reports/phase-7-report.md`."

---

## 7. Not in v1.4

See `SPEC.md` A12's v1.4 lines. In short:
- Repeats other than weekly-on-Saturday.
- A "do first" link that blocks a job, and links in the Briefing.
- A separate priority order per person.
- EMF/WMF, charts and SmartArt turned into pictures, and Word headers and footers.
- A new look for the Categories, Archived and History pages.
- Searching events.

Everything in `PLAN-v1.3.md` §7 is still out, except search covering jobs, which v1.4 adds.
