# Comp HQ — Build Plan for v1.6

**Phase 8 · gates 8.01–8.91 · SPEC A11 "Phase 8" and B14 · planned decisions D-104–D-111**
**Written 21 November 2026. The owner approved the spec the same day.**

This plan runs **on its own, from start to release**, the same way v1.4 did: the build agent
never stops to wait for the owner. §2 says how. It overrides the stop points in `PLAN.md` §2.7
and §5.1 for this phase only.

---

## 1. For the owner (please read this part)

v1.6 is the staff's second list: Done in finishing order, articles and COSMOS folders on jobs,
steps from the Add task window and on My jobs, yearly jobs, US public holidays, and a new
Questions section. `SPEC.md` (Phase 8) is the full list you approved.

### Before you start (about 10 minutes)

1. **MC HQ must be released first.** v1.6 is built on top of MC HQ's version, 1.5.0. **Done:
   1.5.0 was released on 21 November 2026.** (The build agent still checks this first, and stops
   straight away with a plain message if it's missing.)
2. **Keep the computer awake and on the internet.** Plug it in. Go to Windows **Settings → System
   → Power** (or **Power & battery**) **→ Screen and sleep**. Set **"When plugged in, put my device
   to sleep after"** to **Never**.
3. **Stop Windows restarting itself.** Go to **Settings → Windows Update → Pause updates** and
   choose **1 week**.
4. **Start the build.** Open a new Claude Code session in the Staff HQ folder. Choose **Sonnet
   5.5**, set the permission mode to the one that **doesn't ask for approval** (auto), then paste
   the prompt from `docs/v1.6-build-prompt.md` (everything between its two lines) and press Enter.

### While it runs

You don't need to be there. The agent works through 14 tasks in order, roughly in this sequence,
over about 3 to 5 days, most of it waiting for the automatic tests on GitHub:

1. Done in finishing order (1 task)
2. Articles and COSMOS places on jobs (2 tasks)
3. Steps in the Add task window, and on My jobs (2 tasks)
4. Yearly jobs (2 tasks)
5. US public holidays (2 tasks)
6. Questions (3 tasks)
7. A final check of everything, then the release (2 tasks)

Anything that needs you goes into one file, **`reports/phase-8-owner-inbox.md`**. The agent then
carries on with everything that doesn't depend on it.

### When it's finished

1. Open **`reports/phase-8-report.md`**. Its first lines say whether v1.6.0 is published and
   what's done.
2. Open **`reports/phase-8-owner-inbox.md`**. It lists anything waiting for you, each with a
   recommendation.
3. If v1.6.0 is published: in TrueNAS, go to **Apps → comphq → Edit**, change the number after
   `comphq:` to `1.6.0`, and click **Save**. Do the same for **mchq**.
4. Try the six owner checks (O8.1–O8.6). The report has the steps.
5. If the session stopped early (for example after a power cut), open Claude Code in the Staff HQ
   folder on Sonnet 5.5 in auto mode, and paste the short **resume prompt** at the bottom of
   `docs/v1.6-build-prompt.md`.

---

## 2. Working on its own (build agent: these rules override `PLAN.md` §2.7 and §5.1 for Phase 8)

Everything else in `PLAN.md` §2 applies unchanged: sessions, `PROGRESS.md`, the git workflow, the
definition of done, CI failures, commands and engineering rules. **`PLAN-v1.4.md` §2.1–§2.9
also apply in full, with every "Phase 7" read as "Phase 8", `P7-` as `P8-`, the inbox as
`reports/phase-8-owner-inbox.md`, the report as `reports/phase-8-report.md`, and its task numbers
mapped as follows:** P7-18 → P8-13 (full sweep), P7-19 → P8-14 (release), and 7.90–7.91 →
8.90–8.91. They're not repeated here, so the two can't drift apart. The differences:

### 2.1 The one check that does stop
Before anything else, run `git tag --list v1.5.0` and `git status --short`.
- **No `v1.5.0` tag:** MC HQ hasn't been released. Don't commit, change or build anything. Your
  only message is: "I can't start v1.6 yet. It's built on MC HQ's version (1.5.0), which hasn't
  been released. Please finish and release MC HQ, then paste the same prompt again." Then end your
  turn. This is the one stop in Phase 8, and it happens while the owner is still there.
- **`v1.5.0` exists, but files other than this phase's planning files are uncommitted:** don't
  commit them as part of P8-00, and don't discard them. Add an inbox entry listing them, and
  carry on. The planning files are `SPEC.md`, `PLAN-v1.6.md`, `docs/v1.6-build-prompt.md` and
  `docs/intent/v1.6-requests.md`. Stage those by name, never with `git add -A` or `git add .`.

### 2.2 Releases and rollbacks
- Every rollback gate (8.21, 8.53, 8.82, and 8.05's "nothing extra") is about **1.5.0**. The
  container test's upgrade and rollback step already compares with the highest tag below the
  version being built (commit 0b7f59c).
- MC HQ is the same app. Every gate passes with `COMPHQ_BRAND` unset, and 8.91's screen sweep runs
  once more with `COMPHQ_BRAND=mc`. MC HQ needs no separate release: TrueNAS pulls the same image
  for both.

### 2.3 Releasing with something parked
`PLAN-v1.4.md` §2.7 applies with these **feature groups**, each of which can be left out of
v1.6.0 on its own: `done-order` (P8-01), `job-references` (P8-02–P8-03), `steps-everywhere`
(P8-04–P8-05), `yearly-jobs` (P8-06–P8-07), `public-holidays` (P8-08–P8-09) and `questions`
(P8-10–P8-12).
- **Exception:** P8-04 (the Add task window) uses P8-03's section. If `job-references` is parked,
  P8-04 still ships steps in the Add task window and leaves out the Articles and places area. Its
  gates 8.30–8.34 are then reported as passed for steps, and deferred for articles and places.
- Leave a group out by reverting its commits, never by hiding half a screen. A migration that has
  already been pushed stays in place (expand-only, SPEC B6). It's harmless unused.

### 2.4 Things specific to this phase
- **`today` is always `app.Today`.** This matters most for yearly jobs and holidays. Test both with
  `COMPHQ_TEST_TODAY`.
- **Holiday dates come from the spec's table, not from memory.** If you have internet, check them
  against opm.gov's federal holiday pages and the USPS notices (SPEC B14.5). If anything differs,
  build what the official source says. Then write a decision entry and an inbox entry, because the
  owner approved the spec's wording.
- **Gate 4.11 stays at exactly four icon-only buttons.** Copy, Show steps, Edit steps, Mark
  answered, Not answered yet and Ask are worded. Removing an article, place or answer uses the
  existing **remove** icon. HOLIDAY and YEARLY are worded stamps.
- **New sections and cross-section reads follow B2.** `tasks` and `kb` never import each other.
  `cmd/comphq` wires the interfaces (B14.2). `questions` depends only on `people`, `db` and `app`.
  The repository policy tests in `tools/policy` should prove this. If they don't check imports
  between sections yet, add that check in P8-02.

---

## 3. Before building: the spec — **done, 2026-11-21**

**P8-00** is these edits, all already in the working tree. The build agent commits them as
`P8-00 Spec and plan for v1.6`, staging the files by name (§2.1):
- `SPEC.md`: A1, A4, A5, A6, A7, A8, a new A8a (Questions), a new **A11 Phase 8** (gates
  8.01–8.91, O8.1–O8.6), A12, B2, B3, a new **B14**, and the decisions table at the top.
- `docs/intent/v1.6-requests.md`: the requests, the capability map and the owner's answers.
- This file and `docs/v1.6-build-prompt.md`.

In the same commit, the build agent:
- **Copies §8's planned decisions into `docs/decisions.md`**, newest first, as D-104 to D-111, each
  headed `(planned, P8-00, 2026-11-21)`. They're kept in this file until then, so they can't get
  mixed into MC HQ's unfinished commit.
- Adds rows `P8-00` (done, 2026-11-21) and `P8-01` … `P8-14` (`todo`) to `PROGRESS.md`.
- Creates `reports/phase-8-owner-inbox.md` with the heading `# Waiting for the owner — Phase 8`
  and the line "Nothing yet."

`SPEC.md` is the contract. **B14 is the technical direction for every task below.** This plan
doesn't repeat it. Where this plan and SPEC disagree, SPEC wins, and you write a decision entry
saying so.

---

## 4. Gates

Gates 8.01–8.91 and owner checks O8.1–O8.6 are in `SPEC.md` A11 "Phase 8". Every gate is proved by
an automatic test whose title starts with its number (`PLAN.md` §6's naming rule), checked as SPEC
B14.7 describes. The owner checks are **not** waited for. They go in the report.

**Tests that change on purpose** (SPEC gate 8.90). Correct each one, never delete it. Put the gate
number and D-entry in its title or commit message, and list it under "Tests corrected" in the
report:
- gate 2.04's within-column drag test and gate 4.11's move icons, **inside Done only** (D-104);
- gate 7.30's tick-box tests, which become the Repeats choice (D-108);
- tests that list the sidebar's sections, or the Briefing's three sections (D-110);
- gate 7.20's list of Tasks pages and 7.24's "instead" link (D-108, D-110);
- gates 2.08 and 7.34 for yearly jobs (D-108).

Nothing else is weakened, skipped or deleted.

---

## 5. Tasks for Phase 8

The task format, definition of done and progress log are `PLAN.md` §2.2 and §2.4, with §2 above
on top. **Order** follows the six pieces in SPEC B14. "Depends on" is the real constraint. When a
task is parked, carry on with anything that doesn't depend on it.

### Which task proves which gate

| Gates | Task | Proved by (SPEC B14.7 has the detail) |
|---|---|---|
| 8.01–8.05 | P8-01 | Store order unit test; Board E2E (drop low, button, My jobs, within-Done drag, no icons) |
| 8.15, 8.16, 8.17, 8.20, 8.21 | P8-02 | Store unit tests (every refusal, normalising); policy import test; export; container rollback step |
| 8.10–8.14, 8.18, 8.19 | P8-03 | E2E from the window and the page; Copy; article panel; no-JS |
| 8.30–8.34 | P8-04 | E2E (build and save, every refusal keeps the form, discard prompt); no-JS `steps_text` |
| 8.35–8.39 | P8-05 | My jobs E2E (open defaults, kept state, ticks seen in a second context); no-JS; speed |
| 8.44–8.47, 8.51, 8.53 (pass half) | P8-06 | `ResetYearly` table tests; every-list exclusion test; move-order test; 1.5.0-shaped DB test |
| 8.40–8.43, 8.48–8.50, 8.52, 8.53 (survival half) | P8-07 | E2E (Repeats choice, message, Yearly jobs page, stamp, Calendar, search); export; container step |
| 8.66, 8.67 | P8-08 | `holidays_test.go` fixed table and year-boundary test |
| 8.60–8.65 | P8-09 | Calendar E2E (Nov 2026, Jul 2027, Dec 2027); Briefing E2E on five test dates; axe |
| 8.70–8.74, 8.80 (these parts), 8.82 | P8-10 | Store unit tests; E2E for ask, list, answer, mark; sidebar and icon in both brands; no-JS; container step |
| 8.75, 8.76, 8.78, 8.81, 8.80 (these parts) | P8-11 | Store unit tests (edit, remove, restore, search order); E2E; Undo; no-JS; export |
| 8.77, 8.79, 8.83 | P8-12 | Briefing E2E (5 + "more", empty state, Just mine); two-context refresh; speed |
| 8.90, 8.91 | P8-13 | Full dispatch, every layer; screen sweep with both brands |
| O8.1–O8.6 | P8-14 | Listed in the report (not waited for) |

---

## Milestone 8.1 — `done-order`

### P8-01 Done in finishing order
- **Goal:** the Done column always shows the most recently finished job on top, and can't be
  reordered by hand.
- **Gates:** 8.01–8.05.
- **Depends on:** P8-00
- **Touches:** `internal/tasks/store.go` (board query order), the move code for Done
  (`internal/tasks/store.go` `Move`), `web/templates/tasks/board.html` (or its card partial: no move
  icons in Done), `web/static/tasks/board.js` (Done list `sort: false`, dropped card to the top),
  `internal/tasks/store_test.go`, `e2e/tasks/done-order.spec.ts` (new), and the corrected 2.04/4.11
  tests.
- **Behaviour:** SPEC B14.1. **First, write the failing test:** finish job A, then B, then reorder
  Done by drag. On the unchanged code the test must fail, because the dragged order is kept.
- **Tests:** SPEC B14.7's 8.01–8.05 row. Re-run every existing Board drag test unchanged, apart
  from the planned corrections.
- **Done when:** tests are green in CI.
- **Decisions:** D-104.
- **Size:** S.

### Checkpoint 8.1
`main` green in CI, and `PROGRESS.md` shows P8-01 done with its run URL.

---

## Milestone 8.2 — `job-references`

### P8-02 Articles and places, part 1: storage, rules, wiring and export
- **Goal:** jobs can hold article links and COSMOS places in the database, with every rule
  enforced, readable across sections without either importing the other.
- **Gates:** 8.15, 8.16, 8.17 (History rows), 8.20, 8.21.
- **Depends on:** P8-00
- **Touches:** `migrations/tasks/0005_references.sql` (new), `internal/tasks/references.go` (new:
  store functions, `ArticleFinder` interface, path normalising) with tests,
  `internal/tasks/details.go` (`activityText` for the four new actions), `internal/kb` (an
  implementation of `ArticleFinder`, and the `JobsUsing` interface it consumes),
  `cmd/comphq/main.go` (wiring), `internal/tasks/export.go` plus the export wiring
  (`job-articles.csv`, `job-places.csv`), `tools/policy` (the import-boundary check if missing),
  `e2e/smoke/` plus the container script.
- **Behaviour:** SPEC B14.2 "Storage", "Crossing between Tasks and the Knowledge Base", the
  endpoints' rules, and "Export". The endpoints themselves can land here or in P8-03, but their
  store functions and refusals land here.
- **Tests:** store unit tests with fake interfaces: every 8.15 path case (UNC, drive letter,
  forward slashes, trailing spaces, too long, not a path) and every 8.16 refusal (duplicate
  article, case-insensitive duplicate path, 21st of each, a job removed meanwhile), plus two
  contexts racing for the 20th slot. History text keeps the old title after an article rename.
  An integration test with the real `kb` wiring. Export columns. **Container step:** an article
  link and a place seeded on this build survive a rollback to 1.5.0 and an upgrade again.
- **Done when:** tests are green in CI, the container job included.
- **Decisions:** D-105.
- **Size:** M.

### P8-03 Articles and places, part 2: on screen
- **Goal:** people can link articles, add and copy COSMOS places from a job, and see which jobs use
  an article.
- **Gates:** 8.10, 8.11, 8.12, 8.13, 8.14, 8.18, 8.19.
- **Depends on:** P8-02
- **Touches:** a new partial `web/templates/tasks/references.html` (drawn by `window-task.html`
  and `details.html` under Linked jobs), `web/static/tasks/references.js` (new, built like
  `links.js`), `internal/tasks/references_handlers.go` (new: `articles.json` and the forms), the
  shared Copy script (moved from `web/static/settings/copy.js` to `web/static/app/` if both
  sections now use it, with the Setup page's tests unchanged), `web/templates/kb/article.html` (or
  its view template: **Jobs using this article**), `internal/kb` article handler,
  `e2e/tasks/references.spec.ts` (new), `e2e/kb/article-jobs.spec.ts` (new).
- **Behaviour:** SPEC B14.2 "The section" and "The article's side".
- **Tests:** SPEC B14.7's 8.10–8.21 row, screen half. Include an assertion that linking leaves the
  article's version and **Recently updated** unchanged, axe with the section open, and both
  window sizes.
- **Done when:** tests are green in CI.
- **Size:** M.

### Checkpoint 8.2
`main` green, the container job included in one of P8-02 or P8-03's runs. A job's window shows
Linked jobs, then Articles and places.

---

## Milestone 8.3 — `steps-everywhere`

### P8-04 Steps, articles and places in the Add task window
- **Goal:** a new job can be set up in one go, with its steps, articles and places, from the Add
  task window.
- **Gates:** 8.30, 8.31, 8.32, 8.33, 8.34.
- **Depends on:** P8-03 (or see §2.3 if `job-references` is parked)
- **Touches:** `web/templates/tasks/new.html` (`tasks-new-form`), a new
  `web/static/tasks/new-task.js` (the client-only lists), `internal/tasks/new_handlers.go` and
  `store.go` (`Create` with steps, articles and places in one transaction),
  `internal/tasks/details.go` (`created` with `"steps": N`), the window script's close and Cancel
  (the discard prompt), `e2e/tasks/add-with-steps.spec.ts` (new).
- **Behaviour:** SPEC B14.3 "The add form", "Saving" and "Discard prompt". The new step list uses
  the same markup and classes as `steps.html`, so it looks identical, but posts nothing until
  **Add task**.
- **Tests:** SPEC B14.7's 8.30–8.39 row, add-window half. Also: a transaction test proving a
  refusal on the 3rd place leaves no job, no steps and no links behind. Re-run every existing
  add-a-task test (2.02, 4.22, 4.28, 7.10, 7.11, 7.30) unchanged.
- **Done when:** tests are green in CI.
- **Decisions:** D-106.
- **Size:** M.

### P8-05 Steps on My jobs
- **Goal:** people see and tick their jobs' steps on My jobs without opening the job.
- **Gates:** 8.35, 8.36, 8.37, 8.38, 8.39.
- **Depends on:** P8-00
- **Touches:** `internal/tasks/lanes.go` (steps for every card in one query), the My jobs card
  partial in `web/templates/tasks/myjobs.html` (compact tick-only list, Show/Hide steps, Edit
  steps), `web/static/tasks/myjobs.js` (open state in `localStorage`, re-applied after refresh),
  `internal/tasks/checklist_handlers.go` (`redirect_to=myjobs` and the card fragment),
  `e2e/tasks/myjobs-steps.spec.ts` (new), `e2e/speed/` (8.39). Commit with `[speed]`.
- **Behaviour:** SPEC B14.3 "My jobs".
- **Tests:** SPEC B14.7's 8.30–8.39 row, My jobs half. Assert the count of step queries per page
  load is 1, not one per card. Re-run gates 2.32–2.38 and 7.01–7.09 unchanged.
- **Done when:** tests are green in CI, the speed job included.
- **Decisions:** D-107.
- **Size:** M.

### Checkpoint 8.3
`main` green, speed job included. My jobs and the Add task window screenshots are in
`reports/screens/` (via `npx playwright test --project=screens`) and look right when opened with
the Read tool.

---

## Milestone 8.4 — `yearly-jobs`

### P8-06 Yearly jobs, part 1: storage, waiting, and the yearly pass
- **Goal:** a yearly job can wait out of sight, show on its date, and be put away for next year,
  with every list and every move ignoring it while it waits.
- **Gates:** 8.44, 8.45, 8.46, 8.47, 8.51, and 8.53's pass half.
- **Depends on:** P8-01 (Done ordering is touched by the put-away rule)
- **Touches:** `migrations/tasks/0006_yearly.sql` (new), `internal/tasks/yearly.go` (new:
  `ResetYearly`, the waiting predicate, set and unset), `weekly.go` and `section.go`
  (`withWeeklyReset` → `withRepeats`), `store.go`, `lanes.go`, `briefing.go`, `links.go` (the
  waiting predicate in every list; the move code over non-waiting jobs only), `lifecycle.go`
  (`ListFinished` excludes yearly), `details.go` (`activityText` for `yearly_wait` and
  `yearly_shown`), tests.
- **Behaviour:** SPEC B14.4 "Storage", "Waiting", "Order" and "The yearly pass". **First, check
  `0001_tasks.sql`/`0003_weekly.sql` for a CHECK constraint on `repeat`** (B14.4 assumes none).
- **Tests:** SPEC B14.7's 8.40–8.53 row, store half, as Go table tests over `COMPHQ_TEST_TODAY`
  dates. The **every-list exclusion test**: enumerate each store function that lists jobs for a
  screen, and assert a waiting job is in none of them, except search and Calendar, where it must
  appear. The **move-order test**: waiting jobs interleaved in To do, and every move gives the same
  result as with them absent. **Rollback half:** an integration test on a 1.5.0-shaped database.
- **Done when:** tests are green in CI, the container job included.
- **Decisions:** D-108.
- **Size:** L. This is the riskiest task. If it runs long, split it into "storage, predicate and
  order" and "the yearly pass", and record the split in `PROGRESS.md`.

### P8-07 Yearly jobs, part 2: the Repeats choice, the page, the stamp and the export
- **Goal:** people can make a job yearly, see every yearly job on one page, and spot them by their
  stamp.
- **Gates:** 8.40, 8.41, 8.42, 8.43, 8.48, 8.49, 8.50, 8.52, and 8.53's survival half.
- **Depends on:** P8-06, P8-04 (both change the add form; doing them in order avoids clashes)
- **Touches:** `web/templates/tasks/details.html` (`tasks-details-form`) and `new.html` (the
  Repeats select and weeks box), the form handlers, a new `web/templates/tasks/yearly.html` and
  handler (`GET /tasks/yearly`), the Tasks page links beside Finished and Removed tasks,
  `web/static/theme/theme.css` (`.stamp.yearly`), the card templates (Board, My jobs, Team,
  Briefing), `internal/tasks/search.go` (the "Yearly · appears …" mark), `export.go`, the
  corrected 7.30 tests, `e2e/tasks/yearly.spec.ts` (new), and the container script.
- **Behaviour:** SPEC B14.4 "Setting it", "Everywhere else" and "Rollback". Add the rollback note
  to `CHANGES.md`'s 1.6.0 draft entry now, so P8-13 doesn't forget it.
- **Tests:** E2E with `COMPHQ_TEST_TODAY`: make a yearly job due in 10 weeks with 6 weeks' notice.
  It's on Yearly jobs, not Up for grabs, and the message names the date. Move the test date to the
  show date: it's in Up for grabs (nobody on it), stamped YEARLY. Finish it, move the date on 15
  days: it's back on Yearly jobs, due next year, with steps unticked. Also cover changing the
  notice so it shows at once, switching to No, search marking, the Calendar chip and the export
  columns. **Container step:** a waiting yearly job seeded on this build appears as an ordinary
  job on 1.5.0, and is waiting again after upgrading.
- **Done when:** tests are green in CI, the container job included.
- **Size:** M.

### Checkpoint 8.4
`main` green with the container job. Gates 7.30–7.39 (weekly jobs) still pass unchanged, apart
from 7.30's planned correction.

---

## Milestone 8.5 — `public-holidays`

### P8-08 Holidays, part 1: the rules
- **Goal:** Comp HQ can work out every US federal holiday and its observed day for any year, with
  no internet.
- **Gates:** 8.66, 8.67.
- **Depends on:** P8-00
- **Touches:** `internal/calendar/holidays/holidays.go` and `holidays_test.go` (new). Nothing else.
- **Behaviour:** SPEC B14.5 "A pure package", "Postal practice" and "Fixed test table". Build the
  full 2026–2030 table first, as the failing test, from the official lists (opm.gov) if internet
  is available, or from the rules plus B14.5's spot checks if not. Say which in the decision entry.
- **Tests:** the fixed table (every holiday and observed day, 2026–2030), Juneteenth absent before
  2021, and `Days` across the 2027/2028 boundary returning Fri 31 Dec 2027 as `observed_open`.
- **Done when:** tests are green in CI.
- **Decisions:** D-109.
- **Size:** S.

### P8-09 Holidays, part 2: the Calendar and the Briefing
- **Goal:** the Calendar and the Saturday Briefing show holidays, each saying whether there's mail.
- **Gates:** 8.60, 8.61, 8.62, 8.63, 8.64, 8.65.
- **Depends on:** P8-08
- **Touches:** `internal/calendar` (the month and list handlers, and an exported
  `HolidaysBetween` query), `web/templates/calendar/` (the `.chip.holiday` markup),
  `web/static/calendar/` (the day click ignores chips, if it doesn't already) and its CSS,
  `web/static/theme/icons/` (a `flag` icon, B9.4 rules) and the icon policy tests,
  `internal/briefing` (`Build`'s fourth input, the HOLIDAY stamp, Coming up's 14 days),
  `web/templates/briefing/`, the corrected Briefing-sections tests,
  `e2e/calendar/holidays.spec.ts` and `e2e/briefing/holidays.spec.ts` (new).
- **Behaviour:** SPEC B14.5 "Calendar" and "Briefing". Use existing tokens only.
- **Tests:** SPEC B14.7's 8.60–8.67 row. Re-run gates 2.12–2.20, 3.01–3.13 and 6.21–6.23
  unchanged. Add screenshots of Nov 2026 and Dec 2027 for O8.5.
- **Done when:** tests are green in CI.
- **Size:** M.

### Checkpoint 8.5
`main` green. Calendar screenshots for November 2026 and December 2027 are in `reports/screens/`
and look right when opened with the Read tool.

---

## Milestone 8.6 — `questions`

### P8-10 Questions, part 1: the section, asking, answering and marking answered
- **Goal:** a working Questions section where anyone can ask, answer and mark a question answered.
- **Gates:** 8.70, 8.71, 8.72, 8.73, 8.74, 8.82, and 8.80 for these screens.
- **Depends on:** P8-00
- **Touches:** `internal/questions/` (new: `section.go`, `store.go`, `handlers.go`, tests),
  `migrations/questions/0001_questions.sql` (new), `web/templates/questions/` (list and question
  pages), `web/static/questions/questions.css`, `cmd/comphq/main.go` (registration),
  `web/static/theme/icons/questions.svg` (B9.4 rules) and the icon policy tests, the corrected
  sidebar-sections tests, `e2e/questions/ask.spec.ts` (new), and the container script.
- **Behaviour:** SPEC B14.6 "A new section package", "Storage", "Routes" (ask, view, answer, mark,
  unmark), and "Icon".
- **Tests:** store unit tests (length limits with text kept, mark and unmark with who and when, the
  list's two groups and their order, the 50-then-Show-all rule). E2E for each flow, axe, both
  window sizes, the sidebar entry in Comp HQ and MC HQ, and no-JS. **Container step:** a question
  and an answer seeded on this build survive a rollback to 1.5.0 and an upgrade again.
- **Done when:** tests are green in CI, the container job included.
- **Decisions:** D-110.
- **Size:** L. If it runs long, split off the icon and sidebar into their own commit inside the
  task.

### P8-11 Questions, part 2: editing, removing, search and export
- **Goal:** questions and answers can be corrected, removed and brought back, searched, and
  exported.
- **Gates:** 8.75, 8.76, 8.78, 8.81, and 8.80 for these parts.
- **Depends on:** P8-10
- **Touches:** `internal/questions/` (edit, remove, restore, search, the export interface),
  `web/templates/questions/` (edit forms, Removed questions, search results),
  `internal/app` (the section's `NavItem.Search`), `internal/kb` and `internal/tasks` search
  results ("Search questions instead"), `internal/settings/export.go` (wiring),
  `e2e/questions/manage.spec.ts` and `e2e/questions/search.spec.ts` (new).
- **Behaviour:** SPEC B14.6 "Routes" (the rest), "Search" and "Export". Undo uses the shared undo
  message part (D-93).
- **Tests:** store unit tests (edit marks, remove and restore, search word matching with the
  half-typed last word, Unicode folding, order: waiting first, then question text before details
  or answer matches). E2E for edit, Undo of an answer within 10 seconds (`page.clock`, avoiding
  D-86's tight margins), Removed questions, live search and the results page, all three "instead"
  links, no-JS and export. Re-run gates 1.26–1.32, 6.30–6.35 and 7.20–7.26 with only 7.24's
  planned correction.
- **Done when:** tests are green in CI.
- **Size:** M.

### P8-12 Questions, part 3: the Briefing, refresh and speed
- **Goal:** waiting questions show in the Saturday Briefing and on other computers within a
  minute, and stay fast with a large library.
- **Gates:** 8.77, 8.79, 8.83.
- **Depends on:** P8-10, P8-09 (both change the Briefing; doing them in order avoids clashes)
- **Touches:** `internal/questions` (`Waiting` query interface, `GET /questions/version`),
  `internal/briefing` (the fourth section, and the version endpoint reflecting the questions
  counter), `web/templates/briefing/`, `web/static/questions/` (the 60-second poll, reusing the
  shared refresh script), `e2e/seed` (1,000 questions, 3,000 answers, 200 yearly jobs),
  `e2e/briefing/questions.spec.ts` (new), `e2e/speed/`. Commit with `[speed]`.
- **Behaviour:** SPEC B14.6 "Briefing" and "Test library".
- **Tests:** SPEC B14.7's 8.70–8.83 row, Briefing and speed half: 0, 3, 5 and 8 waiting questions
  (empty message, list, the "and 3 more" link), answered never shown, Just mine still showing
  them, two contexts for 8.79, and the speed layer with the new library.
- **Done when:** tests are green in CI, the speed job included.
- **Size:** M.

### Checkpoint 8.6
`main` green with the speed job. Every new screen has a screenshot in `reports/screens/`.

---

## Milestone 8.7 — Finish

### P8-13 Full sweep: every gate, both brands, README, CHANGES, screenshots
- **Gates:** 8.90, 8.91.
- **Depends on:** every earlier P8 task that isn't parked
- **Touches:** `e2e/`, `reports/screens/`, `README.md`, `CHANGES.md`.
- **Behaviour:**
  - Run a full dispatch. Every gate from 1.01 to 7.91 must pass unchanged, apart from §4's planned
    corrections, each pointing at its D-entry.
  - Run the screen sweep (accessibility, both sizes, no sideways scroll) once more with
    `COMPHQ_BRAND=mc`.
  - README, in plain words:
    - **Tasks:** Done's order; articles and COSMOS places (including pasting a path into File
      Explorer); steps in the Add task window and on My jobs; yearly jobs (when they appear, and
      where they wait).
    - **Calendar:** what the holiday chips mean, and the weekend rule.
    - A new **Questions** section.
  - `CHANGES.md`: a 1.6.0 entry in the same style as 1.4.0, with the rollback note from P8-07.
  - Screenshots: the Board's Done column, a job's Articles and places, the Add task window with
    steps, My jobs with steps open, Yearly jobs, the Calendar in December 2027, the Briefing with
    holidays and questions, and both Questions pages.
  - Then do `PLAN-v1.4.md` §2.5's second round on any parked task.
- **Done when:** one full CI run is green with every gate passing, or §2.3 says what's deferred.
- **Size:** M.

### P8-14 Release v1.6.0 and the Phase 8 report — **no stop**
- **Gates:** O8.1–O8.6 (listed for the owner, not waited for); final confirmation of every gate.
- **Depends on:** P8-13
- **Steps:** `PLAN.md` §5.2 with version `1.6.0`, upgrading from and rolling back to `v1.5.0`,
  steps 1–7, but with these differences:
  - **Step 8 doesn't wait.** Commit the reports and push. Set this task to `done` once the release
    is published. Log the owner checks under Owner feedback as `pending`.
  - If CI can't run or §2.3 prevents a release, don't tag. Write the report anyway, saying what's
    done, what's left and the exact next step. Leave the task `blocked`.
- **The report** (`reports/phase-8-report.md`) starts with S13's summary (§6). Then it lists:
  - every gate with its test (also written to `reports/gates-phase-8.md`, as for Phase 7);
  - the tests corrected (§4);
  - new decisions, in plain words, including any marked provisional;
  - whether the holiday table was checked against the official lists online, or built from the
    rules only;
  - deferred features, if any;
  - the six owner checks, each with what to try and what to expect.

  Say plainly that O8.2 (pasting a COSMOS path into File Explorer) and O8.5 (the holiday wording)
  can only be checked in the office.
- **Done when:** v1.6.0 is published, or the report says exactly why not; and the report and inbox
  are committed and pushed.
- **Size:** S.

---

## 6. The stop point (S13, report only)

Added to `PLAN.md` §5.1's list, after S12. If MC HQ's release already used S13, use the next free
number throughout. In Phase 8 it is **written, not waited for**. It's the first section of
`reports/phase-8-report.md`, and the agent's final message:

> **S13 — end of Phase 8 (v1.6.0).** "Phase 8 is done and v1.6.0 is published *(or: what isn't
> yet, and why)*.
>
> - Done now always shows the most recently finished job on top.
> - Jobs can link Knowledge Base articles and COSMOS folders (with a Copy button for File
>   Explorer).
> - Steps can be added in the Add task window, and ticked on My jobs.
> - Yearly jobs wait out of sight and appear in To do ahead of their date.
> - The Calendar and Briefing show US federal holidays, saying whether there's mail.
> - The new **Questions** section lets anyone ask or answer, and waiting questions show in the
>   Briefing.
>
> To update, change the version number in TrueNAS to `1.6.0` for both Comp HQ and MC HQ (see
> Updating in the README). *Inbox: n items waiting for you in `reports/phase-8-owner-inbox.md`, or
> none.* There are six things to check yourself: *O8.1–O8.6, each with what to try and what to
> expect.* The full report is in `reports/phase-8-report.md`."

---

## 7. Risks

| Risk | Impact | How the plan reduces it | Fallback |
|---|---|---|---|
| A waiting yearly job leaks into a list, or upsets the To do order | High: jobs appear early, or moves seem broken | One shared predicate; the every-list exclusion test and the move-order test (P8-06) | Park `yearly-jobs` and release the rest (§2.3) |
| Holiday dates or the weekend rule wrong | Medium: wrong "No mail" advice | Fixed 2026–2030 table, official sources checked when online, the USPS notices recorded in B14.5 | Fix as a 1.6.1 patch; the rules live in one pure package |
| Copy doesn't work on the NAS's plain `http://` address | Medium: places can't be pasted easily | Reuse the Setup page's proven Copy script (P8-03) | The path stays selectable text (8.19); report it in the inbox |
| Tasks and the Knowledge Base end up importing each other | Medium: breaks B2's independence | Interfaces wired in `cmd/comphq`; a policy test on imports (P8-02) | Fix before P8-03, never work round it |
| The Briefing is changed by three tasks (P8-09, P8-12, P8-07's stamp) | Low: merge clashes | Ordered dependencies; each re-runs gates 3.01–3.13 | Ordinary fix inside the later task |
| Speed limits missed (My jobs with many steps, a large Questions library) | Medium | One-query loading; seed data and speed runs in P8-05 and P8-12 | B14.6 allows a search index for Questions; record it |
| MC HQ (1.5.0) not released, or its files still uncommitted | High: v1.6 built on unfinished work | §2.1's check before anything is committed; files staged by name | Stop at once with the plain message |
| CI timing flakes in new drag and refresh tests | Low | Phase 5's drag helper, `page.clock`, D-86's margins, unique test data | `PLAN-v1.4.md` §2.5 |

---

## 8. Planned decisions (copied into `docs/decisions.md` in P8-00)

Each is copied as written, with the heading `## D-NNN … (planned, P8-00, 2026-11-21)`. The task
that builds it adds a **Built (P8-NN, date):** paragraph, as Phase 7 did.

**D-104 Done is in finishing order and can't be reordered by hand (owner request 3).** The staff
asked for the newest finished job on top, always. A hand-made order in Done had no use: nobody
prioritises finished work. So the Board orders Done by `done_at`, newest first. A drop anywhere in
Done lands at the top, and Done cards lose their move icons. This changes gate 2.04 and gate
4.11's icons inside Done only. `position` is still written there, so the move code needs no special
case, and nothing changes in storage. SPEC B14.1.

**D-105 Articles and COSMOS places on a job; Tasks and the Knowledge Base meet through interfaces
(owner requests 4 and 7).** COSMOS is a shared folder (owner, 17 Nov 2026). Browsers won't open
`file:` or UNC links from an `http://` page, so a place is a folder path with a **Copy** button,
not a link. The path is checked loosely (UNC or drive letter) because COSMOS's real layout isn't
known. Linking an article isn't an edit of it. Tasks needs article titles, and an article's page
needs its jobs. Importing each other would break B2, so each section declares the small interface
it needs, and `cmd/comphq` wires the other's implementation in. Limits of 20 each match linked
jobs (D-96). SPEC B14.2.

**D-106 The Add task window keeps new steps, articles and places on the page until one save (owner
request 1).** The alternative was to save the job first and then switch the window to it. That
needs less code, but it's two steps for something the staff asked to be one. The lists live in
the form, and `POST /tasks` writes everything in one transaction through the same store checks as
the single-item routes, so no limit can be got round and a refusal leaves nothing behind. Without
JavaScript, steps come from a one-per-line box. SPEC B14.3.

**D-107 Steps on My jobs are tick-only; Working on now opens by default (owner request 2).** "Show
up on My jobs" means seeing and ticking. Adding and editing stay in the job's window, so My jobs
doesn't grow a second editor. Jobs you're working on show their steps open, because that's what
you're doing. Up next and Up for grabs stay compact, with **Show steps**. The open or closed
choice is kept per computer in browser storage, which is a convenience only, so losing it just
means the defaults. SPEC B14.3.

**D-108 A yearly job is one job that waits, shows ahead, and is put away again (owner request 6;
owner's answer 17 Nov 2026).** The owner chose "appears ahead, then waits" over a list someone
starts by hand.
- **Waiting is computed** (`shows_from > today`), not a fifth column or stage. A new stage would
  break every list on rollback to 1.5.0, and computing it needs no clock job.
- **Put away after 14 days in Done**, the moment an ordinary job would go to Finished tasks. So a
  finished yearly job stays visible for a while, and Undo and Reopen still make sense.
- **Next year is counted from the due date**, not from when it was finished, and the anchor date
  keeps 29 February.
- **People stay on it** from year to year (like weekly jobs). With nobody on it, it shows in Up for
  grabs.
- **The weekly tick-box becomes a Repeats choice**, since a job can't be both. This changes gate
  7.30's wording, and gate 7.20's list of Tasks pages gains Yearly jobs.

SPEC B14.4.

**D-109 US federal holidays are worked out inside Comp HQ, with the Postal Service's weekend rule
(owner request 8; owner's answer 17 Nov 2026).** No download and no internet (A2, intent). The 11
federal holidays follow fixed rules, so a small pure package can compute any year. The
office cares about mail, and the Postal Service treats weekend holidays differently from other
federal offices. A Saturday holiday is a no-mail Saturday, with mail as normal on the Friday that
federal offices observe. A Sunday holiday means no mail on the Monday. This matters to a Saturday
team: Christmas 2027 and New Year's Day 2028 are Saturdays. Holidays show in the Briefing's
existing This week and Coming up sections (from 2 weeks ahead), and add no section of their own.
Holidays aren't stored, so there's nothing to export or migrate.
SPEC B14.5.

**D-110 Questions is a new section; anyone answers and anyone marks answered (owner request 5;
owner's answer 17 Nov 2026).** There are no roles (A3), so the subprefect answers like anyone.
Marking answered is a separate press, because a first answer is often not the final one. The text
is plain, with no editor, so the section stays small. Waiting questions show in the Briefing for
everyone, as the owner chose, as a fourth section. Each section's top search box searches that
section (D-94), now with three "instead" links. The sidebar and Briefing tests that list sections
are corrected. SPEC B14.6.

**D-111 v1.6 ships as one update, after MC HQ's 1.5.0 (planning, 21 Nov 2026).** As with v1.4
(D-89): one release, with six feature groups that can each be left out if one gets stuck
(`PLAN-v1.6.md` §2.3). It's built on 1.5.0 so that MC HQ and Comp HQ get it together, and every
rollback gate means 1.5.0. The planning files are kept apart from MC HQ's uncommitted work until
1.5.0 is released.

---

## 9. Not in v1.6

See `SPEC.md` A12's v1.6 lines. In short:
- Clicking a COSMOS folder to open it, checking that a path exists, editing a place in place, and
  articles or places on cards.
- Editing steps from My jobs, and steps opened on the Board or Team.
- Yearly patterns other than "same date every year", and linked jobs in the Add task window.
- Other countries' or states' holidays, choosing which holidays show, your own holidays, and
  non-federal days off.
- Notifications, asking a named person, best answers, votes, comments, formatting in questions,
  and turning an answer into an article.

Everything in `PLAN-v1.4.md` §7 is still out.
