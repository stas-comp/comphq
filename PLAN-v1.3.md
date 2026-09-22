# Comp HQ — Build Plan for v1.3

**Easier dates, a sidebar that stays put, a Sunday-first calendar, and search that keeps up with
your typing.**

This plan follows on from `PLAN.md` and `PLAN-v1.2.md`. Everything in `PLAN.md` §2 (how the build
agent works, the progress log, the git workflow, the definition of done, the standing engineering
rules) and §5 (the release procedure) still applies unchanged. This document only adds Phase 6.
Where this plan says "the release procedure", it means `PLAN.md` §5.2.

---

## 1. For the owner (please read this part)

v1.3 is a set of everyday fixes, all things you ran into after v1.2 came out. Nothing about your data
changes, and you can go back to 1.2.0 at any time.

**1. The sidebar stays put.** Today the dark sidebar scrolls away with the page, so on a long page
like the Board, **Settings** and the version number sit below the bottom of the screen until you
scroll all the way down. In v1.3 the sidebar stays still while the page scrolls, so every section
and Settings are always in view. The bar across the top, with the search box and "You: *name* ·
Change", stays put in the same way.

**2. A calendar for every date.** Every box where you type a date (a job's due date, and an event's
date, end date and "until" date) opens a small calendar when you click it. Pick a day and it's
filled in. It has **Today** and **This Saturday** buttons, since Saturday is the date you'll want
most. You can still type the date if you prefer, and now you can leave off the year: **25/9** means
the 25th of September nearest to today, and the box shows the full date as soon as you move on.

**3. Click a blank day to add an event.** Today only the small day number in a Calendar box is
clickable. In v1.3 clicking anywhere empty in the box starts a new event on that day. Clicking an
event or a job that's already in the box still opens it, as now.

**4. Weeks start on Sunday.** The Calendar's month view, and the new date calendar, start the week on
Sunday. Saturday, your office day, becomes the last column and stays highlighted.

**5. Search keeps up with your typing.** When you type into the search box, results appear as you go.
But some half-typed words find nothing until you finish them: "pay" doesn't find "payment",
"busin" doesn't find "business", and "voluntee" doesn't find "volunteer". The search trims words
down to their root so that "printers" finds "printer", and it trims a half-typed word as if it were
finished, which takes it off course. v1.3 looks up the word you're still typing exactly as you've
typed it, and keeps the root-matching for the words you've finished. Whole words work as they do now.

### What you'll do yourself

| When | What | How long |
|---|---|---|
| End of Phase 6 | Try the four owner checks (O6.1–O6.4 below) on an office computer. | 15 min |
| End of Phase 6 | Change the version number in TrueNAS to `1.3.0`, as with every update. | 2 min |

### Decisions I've made for you

Each of these is the simpler choice. Say so if you'd rather have it the other way, and the plan
changes before building starts.

- **D-80. The top bar stays put along with the sidebar.** You picked this. The red "backup is
  overdue" warning, when it appears, still sits above the top bar and scrolls away with the page:
  it's there to be noticed once, not to cover the screen.
- **D-81. We draw our own date calendar and don't use the browser's.** The browser's built-in
  calendar shows dates month-first on your computers, and nothing can change that (this is why v1.1
  went to a plain typed box, D-61). Our own calendar shows dates day-first and matches the rest of
  Comp HQ. The calendar opens when you click the date box, and a small calendar mark inside the box
  shows it's there. There's no extra button for it, which keeps icon-only buttons to the four the
  team already knows (gate 4.11).
- **D-82. "25/9" means the 25 September nearest to today.** On 22 September 2026, **25/9** is
  25/09/2026 and **1/7** is 01/07/2026 (earlier this year). In late December, **5/1** is next
  January, not the one just gone. The box always shows the full date once you move on, so you see
  which year it chose before you save.
- **D-83. Sunday first everywhere a week is drawn.** That's the month grid and the new date calendar.
  The Saturday Briefing doesn't change. It was always built around Saturday, not around where a week
  starts, so "this week" still runs from today to Friday.
- **D-84. A blank day starts an event, not a job.** Jobs come from Tasks. Their due dates still show
  on the Calendar and still open when clicked.
- **D-85. The half-typed word is looked up in a second word list, rebuilt every time Comp HQ
  starts.** The second list keeps words exactly as they're written. Rebuilding it at start-up means
  an upgrade, a rollback and an upgrade again can never leave it out of date.

### Not in this update

- **Safer Remove for names.** Remove on **Settings → People** still works in one click, with no
  "are you sure?" and no way to bring a name back (re-adding the name makes a new person). You chose
  to leave this for now. It remains a small, self-contained change for later.

---

## 2. Before building: the spec

This plan's spec work is **P6-00** and is done first, by the owner's preferred model for direction
documents, not by the build agent. It consists of these edits:

- **A4** is changed: the sidebar and the top bar stay in view while the page scrolls.
- **A7** is changed: weeks start on Sunday, and clicking anywhere empty in a day starts an event.
- **A11** gains a new **Phase 6** gate list, which is §3 of this document copied in word for word.
  Gate **2.12**'s "weeks start Monday" gains a note that gate 6.20 replaces it.
- **A12**: the safer Remove for names joins the "could be added later" list.
- **B4** is changed: "Article blocks and search" gains the half-typed word lookup, and there is a new
  **"Dates typed without a year"** entry.
- **B9.5**: its note on the date field points at the new B12.
- **B12** is new: the v1.3 design (the frame that stays put, the date calendar, Sunday-first, the
  blank-day click, and the second word list).

**Status: done, 2026-09-22.** All the edits are in `SPEC.md`, and D-80–D-85 are in
`docs/decisions.md`. `PLAN.md` §5.1 gained stop point S11 and §6 points at this document. The build
agent starts at **P6-01**. `SPEC.md` is the contract, and this plan is the order of work.

---

## 3. Gates for Phase 6

Every gate below is proved by an automatic test, in the way `SPEC.md` B7 lays out, except the four
marked 👤 which the owner checks by hand.

### The sidebar and top bar stay put

- **6.01** On every page, the sidebar stays in place while the page scrolls. With the test library
  loaded, scrolled to the very bottom of the Board in a 1024 × 700 window, every section link,
  **Settings** and the version number are on screen.
- **6.02** The top bar (the search box and "You: *name* · Change") also stays at the top while the
  page scrolls, and the live search results still open over the page, fully visible.
- **6.03** Nothing ends up hidden under the top bar. A search result's highlighted passage (gate
  1.28) scrolls into view *below* the bar, not behind it. The task window, the list of names on a
  card, the Undo on steps and drag-and-drop on the Board and Team all still work as before.
- **6.04** If a window is too short to show the whole sidebar, the sidebar scrolls on its own, so
  Settings can always be reached. The page itself still never scrolls sideways (gate 5.31).
- **6.05** The backup warning, when shown, sits above the top bar and scrolls away with the page
  (D-80).

### A calendar for every date

- **6.10** Every date box in the app opens a small calendar when clicked or tabbed into: a job's due
  date (in the add-a-task window and when editing), an event's date, end date and "until" date, and
  the dates of a "just this one" change. A small calendar mark inside the right-hand end of each
  date box shows the calendar is there.
- **6.11** The calendar shows one month, starting on **Sunday**, with Saturday's column shaded as on
  the Calendar page. Today is marked, the date already in the box is marked, and there are previous
  and next month controls.
- **6.12** Picking a day fills the box as day/month/year (**25/09/2026**) and closes the calendar.
- **6.13** The calendar has a **Today** button and a **This Saturday** button. This Saturday is today
  on a Saturday, and otherwise the coming Saturday: the same Saturday the Briefing is about.
- **6.14** Typing still works exactly as before. While you type, the calendar follows: typing
  25/12/2026 moves it to December with the 25th marked. Clicking elsewhere or pressing Escape closes
  it and leaves what was typed alone.
- **6.15** An empty end date or "until" date opens on the month of the start date, not on today's
  month.
- **6.16** It works from the keyboard alone. The arrow keys move a day or a week, Page Up and Page
  Down move a month, Enter picks, and Escape closes and returns to the box. A screen reader hears
  the month's name and each day's full date ("Friday 25 September 2026"). The accessibility check
  passes with the calendar open.
- **6.17** With JavaScript switched off, every date box is exactly the typed box of v1.2, and still
  works.
- **6.18** A date can be typed without its year: **25/9**, **25.9** or **25-9** means the 25 September
  nearest to today (D-82). With JavaScript, the box shows the full date as soon as you move on. Without
  it, the saved date is the same, because the server applies the same rule. On 22 September 2026:
  25/9 → 25/09/2026, 1/7 → 01/07/2026. On 20 December 2026: 5/1 → 05/01/2027.
- **6.19** Everything D-61 refused is still refused, with the same plain message. That includes an
  American-order date such as **9/25/2026**, and a short date that is not a real day, such as **31/9**.

### The Calendar

- **6.20** The month view starts the week on **Sunday** (Sun, Mon … Sat). Saturday is the last column
  and is still highlighted. Previous, Next, Today and the greyed days from the months on either side
  work as before. *(This replaces "weeks start Monday" in gate 2.12. Nothing else in 2.12 changes.)*
- **6.21** Clicking anywhere empty in a day's box opens a new event with that date filled in. This
  includes the greyed days from the months on either side. Clicking an event or a job due in the
  box still opens that event or job, as before.
- **6.22** A day's box shows that it can be clicked: the pointer changes, and a small **+** appears in
  the box on hover.
- **6.23** From the keyboard, the day number is still a link that does the same thing, and its spoken
  name says what it does: "Add an event on Tuesday 22 September".

### Search keeps up with typing

- **6.30** A half-typed last word finds the articles containing words it's the start of. "pay" finds
  an article containing "payment", and "busin", "generat", "voluntee" and "happin" find "business",
  "generation", "volunteer" and "happiness". This holds in the live results and on the full results
  page.
- **6.31** Finished words still find their relatives exactly as before. "printers" finds "printer"
  and "ton" finds "toner" (gate 1.29, unchanged), and a match in the title still ranks above a match
  in the text (gate 1.26).
- **6.32** The word a half-typed word found is highlighted in the result's passage and on the article
  page, as any other match is (gates 1.27 and 1.28).
- **6.33** Publishing, editing and archiving an article are reflected in the very next search for
  half-typed words too (gates 1.30 and 1.31).
- **6.34** Search survives an upgrade from 1.2.0 and a rollback back to 1.2.0. The rolled-back app
  searches as 1.2.0 did. On upgrading again, a half-typed word finds an article that was published
  or edited *while* rolled back.
- **6.35** With the test library loaded, search still answers within 1 second (gate 4.53, unchanged),
  including for a two-letter last word.

### Still true afterwards

- **6.40** Every gate from Phases 1 to 5 (1.01–5.31) still passes, unchanged, at the same
  thresholds. The one exception is the three words of gate 2.12 that gate 6.20 replaces, and D-83
  records it.
- **6.41** Every screen still passes the accessibility check and still works from 1024 × 700 up to
  full HD, with no sideways scrolling that shouldn't be there.

### 👤 Owner checks — end of Phase 6

- **O6.1** Open the Board and scroll to the bottom. Confirm Settings is in view the whole time, and
  that nothing on the page looks cut off under the top bar.
- **O6.2** Add a job with a due date picked from the calendar, then another by typing just **25/9**.
  Confirm both feel natural, and that This Saturday picks the Saturday you expect.
- **O6.3** On the Calendar, click an empty part of a day and add an event. Confirm Sunday-first
  looks right to you and that Saturday still stands out.
- **O6.4** In the search box, slowly type a word from one of your articles. Confirm results appear
  before you finish the word.

---

## 4. Technical direction

SPEC B12 is the full design. This section is the short version, with the places in the code.

### 4.1 The frame (B12.1)

`web/static/app/frame.css` only. `.sidebar` becomes `position: sticky; top: 0; height: 100vh;
overflow-y: auto` (100vh, not a fixed pixel height, so gate 6.04 falls out of it). `.topbar` becomes
`position: sticky; top: 0` with its paper background and a z-index above page content but below the
task window, the live search panel and the name menus. Articles' `[data-b]` blocks and anything
scrolled to by fragment get `scroll-margin-top` equal to the bar's height (gate 6.03). Use a CSS
custom property for that height, set once in `frame.css`, rather than a number repeated in `kb.css`.
Nothing in `layout.html` needs to move.

### 4.2 Dates typed without a year (B4, B12.2)

`internal/app/format` gains `ParseDayFirstNear(s string, today time.Time)`, which accepts everything
`ParseDayFirst` accepts plus `d/m`, `d.m` and `d-m` without a year, choosing the year per D-82. The
nearest-year rule compares the three candidates (last year, this year, next year) and takes the
closest; a tie can't happen with whole days. A day that doesn't exist in the chosen year (29/2) is
tried in the other two candidates before being refused. `NormaliseDate` gains the same `today`
argument, and every caller passes `app.Today(...)`, which honours test mode's fixed date. The typed
forms stay the no-JavaScript path, and they're done first (P6-02).

### 4.3 The date calendar (B12.2)

One shared script, `web/static/app/datepicker.js`, loaded from `layout.html` like `refresh.js`, and
attached to every `input.field-date` on the page, including ones added later inside the task window
(listen at the document, don't bind once at load). Its styles are a shared part in `theme.css`
(SPEC B9.5), built from the existing tokens and the Calendar's own Saturday wash. No new library:
it is one small month grid, and a vendored date-picker would bring its own look and its own locale
rules, which is the problem D-61 already solved once. The pairing for 6.15 is declared in the markup:
`data-date-after="event-start-date"` on the end and "until" boxes. The script never becomes the only
way to enter a date. It writes into the text box, and the form posts the text box as it does today.

### 4.4 Sunday-first (B12.3)

`internal/calendar/handlers.go`: `mondayOnOrBefore`/`sundayOnOrAfter` become
`sundayOnOrBefore`/`saturdayOnOrAfter`; the heading row in `month.html` becomes Sun … Sat. The
Saturday class is already per-day, not per-column, so it follows. The pop-up shares the same rule.
The Briefing is untouched (D-83), and a test says so.

### 4.5 The blank-day click (B12.4)

A few lines added to an existing Calendar script, or a new `web/static/calendar/month.js`. A click
on a `td.calendar-day` whose target is not inside an `a` goes to that cell's day-number link.
Nothing new on the server: `/calendar/new?date=` already fills in the date. The day number's link
gains an `aria-label` built on the server from the cell's date.

### 4.6 Search: the half-typed word (B4, B12.5)

The approach, chosen so ranking, snippets and highlighting stay in the one `kb_search` table that
already does them:

1. A new migration, `migrations/kb/0005_search_words.sql`, adds `kb_search_words`, an FTS5 table
   with `tokenize = 'unicode61'` (no stemming) over the same title and body text, plus an `fts5vocab`
   table over it, so the app can list the whole words the knowledge base contains. Adding tables is
   rollback-safe: 1.2.0 never reads them.
2. Wherever `kb_search` rows are rewritten (publish, edit, archive, restore), `kb_search_words` is
   rewritten in the same transaction.
3. **At every start-up, `kb_search_words` is rebuilt from `kb_search`.** That's what makes gate 6.34
   hold: articles changed while rolled back to 1.2.0 are picked up on the next upgrade. With fewer
   than a few hundred articles this takes a moment. Measure it and write the time in the decision
   entry.
4. `BuildQuery` stays pure. A new step before it looks up the last word in the vocabulary, taking at
   most 30 whole words that start with it, most common first. The last term then becomes
   `("pay"* OR "payment" OR "payments" …)`. Those whole words go through the stemmer as usual and
   match `kb_search`. Finished words are untouched.
5. Everything is still quoted, and raw input still never reaches MATCH (B4). The expansion list comes
   from the database's own vocabulary, not from the input.

If the build agent finds a simpler way that passes 6.30–6.35, it may use it and record why in a
decision entry. The rebuild-at-start-up requirement is not optional.

---

## 5. Tasks for Phase 6

The task format, the definition of done and the progress log are `PLAN.md` §2.2 and §2.4, unchanged.

### Which task proves which gate

| Gates | Task | Proved by |
|---|---|---|
| 6.01–6.05 | P6-01 | `e2e/app/frame-sticky.spec.ts` at 1024 × 700 and full HD; the gate 1.28 scroll test re-run |
| 6.18, 6.19 | P6-02 | `format` unit tests (the year table); Go integration tests; E2E with JavaScript off |
| 6.10–6.17 | P6-03 | `e2e/app/datepicker.spec.ts` on every date box; keyboard-only test; axe with it open |
| 6.20 | P6-04 | Calendar handler unit test; the corrected design and calendar E2E; a Briefing-unchanged test |
| 6.21–6.23 | P6-05 | `e2e/calendar/blank-day.spec.ts`; axe |
| 6.30–6.35 | P6-06 | `search` unit and integration tests; E2E; the rollback step of the container test; speed |
| 6.40, 6.41 | P6-07 | Full suite at every layer; screenshots |
| O6.1–O6.4 | P6-08 (S11) | Owner reply, logged in `PROGRESS.md` |

### P6-00 Spec edits for v1.3 — **done, 2026-09-22**
- **Goal:** `SPEC.md` describes v1.3 before anything is built against it.
- **Touched:** `SPEC.md` (A4, A7, A11 Phase 6 and the note on 2.12, A12, B4, B9.5, new B12),
  `docs/decisions.md` (D-80–D-85), `PLAN.md` (§5.1 stop point S11, §6 table, P5 pointer).
- **Note:** this isn't a build-agent task. The owner's spec model drafts it, as with every spec
  change.

---

## Milestone 6.1 — The frame

### P6-01 The sidebar and top bar stay put
- **Goal:** Settings is always on screen.
- **Gates:** 6.01, 6.02, 6.03, 6.04, 6.05.
- **Depends on:** P6-00
- **Touches:** `web/static/app/frame.css`, `web/static/kb/kb.css` (the scroll margin),
  `e2e/app/frame-sticky.spec.ts` (new).
- **Behaviour:** §4.1. The one real risk is 6.03: every overlay (task window, live search panel,
  name menu, drag ghost, steps Undo) must still sit above the bar, and the gate 1.28 passage must
  land below it. Check each one. Don't assume any of them.
- **Tests:** at 1024 × 700 and at 1920 × 1080: scroll the Board (test library) to the bottom and
  assert Settings and the version are inside the viewport; assert the top bar's box stays at y = 0
  after scrolling; open the live search panel after scrolling and assert it is fully visible; re-run
  gate 1.28's scroll test and assert the highlighted block's top is below the bar's bottom; open the
  task window and a card's name menu after scrolling; at 1024 × 500 assert the sidebar scrolls and
  Settings can be reached; the no-sideways-scroll check on every page.
- **Done when:** tests are green in CI.
- **Decisions:** D-80.

---

## Milestone 6.2 — Dates

### P6-02 Dates typed without a year, and the typed path first
- **Goal:** the server understands **25/9**, so the pop-up in P6-03 is never the only way to enter a
  date.
- **Gates:** 6.18, 6.19.
- **Depends on:** P6-00
- **Touches:** `internal/app/format/format.go` and its tests, every handler that calls
  `NormaliseDate` (tasks new and details, calendar event and occurrence), and the date-box
  placeholder text if it now reads better as `dd/mm/yyyy or dd/mm` (the builder's judgement; the
  message wording of D-61 is kept).
- **Behaviour:** §4.2. The rule depends on "today", so every call goes through `app.Today`, and the
  tests pin it.
- **Tests:** a table of unit tests: the three examples of 6.18, both sides of the new year, 29/2 in a
  leap and a non-leap year, 31/9, 0/5, 9/25/2026 still refused, every v1.2 accepted form still
  accepted. Go integration tests that a job and an event saved with **25/9** store the expected ISO
  date. An E2E with JavaScript off that does the same through the form.
- **Done when:** tests are green in CI.
- **Decisions:** D-82.

### P6-03 The date calendar
- **Goal:** click a date box and pick a day.
- **Gates:** 6.10, 6.11, 6.12, 6.13, 6.14, 6.15, 6.16, 6.17.
- **Depends on:** P6-02
- **Mockup:** not drawn. Build it from the Calendar page's own look (the Saturday wash, the today
  mark, the mono date type) and the shared parts, so it looks as if it had always been there. Owner
  check **O6.2** covers it.
- **Touches:** `web/static/app/datepicker.js` (new), `web/static/theme/theme.css` (the shared part
  and the calendar mark inside `.field-date`), `web/templates/app/layout.html` (one script line),
  `web/templates/calendar/event-form.html` and `occurrence.html` (`data-date-after`),
  `e2e/app/datepicker.spec.ts` (new).
- **Behaviour:** §4.3. With JavaScript, when the box is left, it's rewritten to the full date using
  the same rule as P6-02, so the owner sees the year before saving. Sunday first (D-83). It opens
  below the box, or above it when there's no room below, and never makes the page scroll sideways.
  Inside the task window it sits above the window's content and isn't cut off by it.
- **Tests:** for each of the date boxes of 6.10 (the add-a-task window, editing a job, the job's own
  page, the event form's three, the "just this one" form's two): open, pick, assert the value.
  Today and This Saturday on a pinned Saturday and a pinned Wednesday. Typing moves the month.
  Escape keeps the typing. An empty end date opens on the start date's month. The whole flow
  keyboard-only. axe with the calendar open, in the task window and on the event form. With
  JavaScript off, the v1.2 date tests pass as they are. The gate 4.05 style rules apply to the new
  controls. The no-sideways-scroll check with the calendar open at 1024 × 700.
- **Done when:** tests are green in CI.
- **Decisions:** D-81.

---

## Milestone 6.3 — The Calendar

### P6-04 Weeks start on Sunday
- **Goal:** the month grid runs Sunday to Saturday.
- **Gates:** 6.20.
- **Depends on:** P6-00 (independent of Milestone 6.2)
- **Touches:** `internal/calendar/handlers.go`, `web/templates/calendar/month.html`,
  `web/static/calendar/calendar.css` if the Saturday heading style is tied to a column position,
  and the tests below.
- **Behaviour:** §4.4. This is the only v1.3 change to an earlier gate's meaning. The tests that
  assert Monday-first are **corrected, not deleted**. Each one now asserts Sunday-first, names gate
  6.20 in its title, and is listed under "Tests corrected" in the phase report, with D-83 as the
  reason. Currently those are in `e2e/calendar/calendar.spec.ts` and `e2e/design/calendar.spec.ts`.
  Search for others.
- **Tests:** a unit test of the grid for September 2026 (starts on a Tuesday, so the grid starts Sun
  30 Aug), August 2026 (starts on a Saturday: the grid starts Sun 26 Jul and runs six rows) and
  February 2026 (starts on a Sunday: exactly four rows); the corrected
  E2E; a Briefing test asserting that the Briefing for Sat 26 Sep 2026 is exactly what it was at
  v1.2 (D-83).
- **Done when:** tests are green in CI.
- **Decisions:** D-83.

### P6-05 Click a blank day to add an event
- **Goal:** the whole day box is the way in, not just its number.
- **Gates:** 6.21, 6.22, 6.23.
- **Depends on:** P6-04
- **Touches:** `web/templates/calendar/month.html`, `internal/calendar/handlers.go` (the spoken
  label), `web/static/calendar/calendar.css`, a small script (§4.5), `e2e/calendar/blank-day.spec.ts`
  (new).
- **Behaviour:** §4.5. On a day crowded with chips the empty part is small. That's fine, because the
  day number is always there as well.
- **Tests:** click the empty lower part of a current-month day, and of a greyed next-month day, and
  assert the new-event form has that date. Click an event chip and a job chip in the same box and
  assert they open as before. The keyboard route and its spoken name. axe on the month view. With
  JavaScript off, the day number still works.
- **Done when:** tests are green in CI.
- **Decisions:** D-84.

---

## Milestone 6.4 — Search

### P6-06 Search finds half-typed words
- **Goal:** results keep up with typing.
- **Gates:** 6.30, 6.31, 6.32, 6.33, 6.34, 6.35.
- **Depends on:** P6-00 (independent of Milestones 6.1–6.3)
- **Touches:** `migrations/kb/0005_search_words.sql` (new), `internal/kb/search/` (the vocabulary
  lookup, and the rebuild), `internal/kb/articles.go` and `history.go` (write the second table
  wherever `kb_search` is written), the start-up path that runs the rebuild, `deploy/test/container-test.sh`
  (the 6.34 step), tests.
- **Behaviour:** §4.6. The old gate 1.29 test and its D-79 note stay as they are. They're still
  true, and they were the first sign of this problem.
- **Tests:** unit tests for the expansion (the five words of 6.30, a prefix with more than 30
  matches takes the 30 most common, a prefix matching nothing leaves the query unchanged, symbols
  never reach MATCH). An integration test that archiving an article removes its words from the
  lookup. E2E typing "pay" and "busin" character by character into the live search. Gate 1.28's
  highlight check with a half-typed word. The container test: roll back to v1.2.0, publish an
  article containing "zephyrine" through the 1.2.0 app, upgrade again, and assert "zephyr" finds it.
  The speed suite with a two-letter last word.
- **Done when:** tests are green in CI.
- **Decisions:** D-85, plus the measured rebuild time.

---

## Milestone 6.5 — Finish

### P6-07 Full sweep: screenshots, README, CHANGES
- **Goal:** prove nothing regressed, and give the owner what they need to check it.
- **Gates:** 6.40, 6.41.
- **Depends on:** P6-01, P6-03, P6-05, P6-06
- **Touches:** `e2e/`, `reports/screens/`, `README.md`, `CHANGES.md`.
- **Behaviour:**
  - Every gate 1.01–5.31 re-run unchanged, at the same thresholds, apart from the corrected
    Monday-first assertions of P6-04. A weakened, skipped or deleted earlier test fails the phase
    (`PLAN.md` §2.5, SPEC B8 rule 5).
  - Refresh the screenshots in `reports/screens/`, adding one of the Board scrolled to the bottom
    with the sidebar in view, one of the date calendar open in the task window, and one of the
    Sunday-first month.
  - README: the Tasks and Calendar sections mention the date calendar, typing **25/9** and clicking
    a blank day. The Knowledge Base section drops any wording that implies a half-typed word may
    find nothing.
- **Tests:** the full suite at every layer, including accessibility and window sizes at 1024 × 700
  and full HD, and the speed suite with the test library.
- **Done when:** every gate 1.01–6.41 passes in one full CI run.

### P6-08 Release v1.3.0 and the Phase 6 report — **STOP S11**
- **Goal:** publish v1.3.0 and report.
- **Gates:** O6.1, O6.2, O6.3, O6.4 (owner); final confirmation of every gate.
- **Depends on:** P6-07
- **Steps:** follow the release procedure in `PLAN.md` §5.2 with version `1.3.0`, upgrading from and
  rolling back to `v1.2.0`, and confirm gate 6.34 in that run. Then send **S11**.
- **In the report, say plainly:** that the date calendar has no mockup to check against, so O6.2 is
  the check that matters most; which tests were corrected for Sunday-first; and the measured
  start-up cost of rebuilding the search word list.
- **Done when:** the release exists, the report is written, S11 is sent, and the owner has confirmed
  O6.1–O6.4.

---

## 6. The stop point

Added to `PLAN.md` §5.1, after S10:

> **S11 — end of Phase 6 (v1.3.0).** "Phase 6 is done and v1.3.0 is published. The sidebar now
> stays put, every date box has a calendar (and takes **25/9**), the Calendar starts on Sunday and a
> blank day adds an event, and search finds half-typed words. To update, change the version number
> in TrueNAS to `1.3.0` (see Updating in the README). There are four things to check yourself:
> *O6.1–O6.4, each with what to try and what to expect.* The full report is in
> `reports/phase-6-report.md`."

---

## 7. Not in v1.3

Listed here so nobody has to wonder:

- The safer Remove for names (a confirmation, and a Removed people list to bring someone back). The
  owner left it for later (§1).
- A blank day that offers "add a job due this day" as well as an event (D-84).
- Returning to the month after saving a new event, rather than to the event's own page. Unchanged
  from v1.2.
- The browser's own date calendar (D-81), and a native time picker change. Times stay as they are.
- Search that forgives misspellings, and search covering jobs and events. Both are still out (SPEC A12).
- Everything in `PLAN-v1.2.md` §7 is still out.
