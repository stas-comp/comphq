# Comp HQ — Phase 4 report (version 1.1)

**Version 1.1.0 is ready. Every screen now follows the design mockup, and there are a few new things on the Tasks Board.**

## Summary

- **What's new:**
  - **The look.** Every screen — the frame, Board, My jobs, Team, Calendar, Knowledge Base, Briefing and the name picker — now follows `docs/design/mockup.html`: the tall condensed headings, the readable body text, dates in a typewriter-style face, the orange used only as a fill, and the deep orange for orange text.
  - **A proper "Add task" window.** **+ Add task** opens a window where you type the title, a description, a size, the column, a due date and who's on it, all in one go. Dates are typed day first (25/09/2026).
  - **A window for reading and editing a job.** Click a job's title and it opens laid out for reading; **Edit** turns it into the form; **History** sits at the foot.
  - **Card icons.** Every card has small icons with tooltips: move up, move down, remove, and give back (take yourself off a job).
  - **People on a job.** Press the little circles (or **Assign**) to open a short list of names; several people can be on one job. An idea can have people on it too and still shows for them (and doesn't count toward their workload).
  - **A bug fixed.** Ideas that belong to someone now appear in that person's Team column instead of vanishing.
- **Automatic checks:** all 54 Phase 4 promises (gates 4.01–4.54) passed in the full test run https://github.com/stas-comp/comphq/actions/runs/35478917941 (`go`, `browser`, `container` with the internet blocked and an upgrade-then-rollback against 1.0.0, `speed`, `screens`). Every Phase 1–3 promise passes in the same run, so all of them are re-proven. The release run for v1.1.0 (https://github.com/stas-comp/comphq/actions/runs/35480026380) repeated the whole thing and published the download; it passed too. See `reports/gates-phase-4.md` for the full list.
- **How "looks like the mockup" was checked:** I didn't compare pictures. The tests open the mockup and the running app in one browser and compare the *actual* fonts, sizes, colours, spacing and borders of matching parts, one by one. Where the mockup and my earlier wording disagreed, the mockup won.
- **Tests I corrected:** see "Tests corrected" below.

## What I need from you

### 1. Update steps

In TrueNAS, go to **Apps → comphq → Edit**, change the number after `comphq:` to **1.1.0**, and click **Save**. Wait for it to show **Healthy**. Your data carries across untouched.

### 2. Please look hardest at the task window

**The window that opens when you click a job (and the one for adding a job) is the one screen the mockup does *not* show.** Everything else I could check against a picture; this I had to build out of the mockup's own parts so it looks as if it had always been there. It is the part most likely to feel wrong to you, so that is where I most want your eyes (check O4.3 below).

### 3. Owner checks (SPEC §A13 "👤 Owner checks — end of Phase 4")

| Check | What to try | What you should see |
|---|---|---|
| **O4.1** | Open `docs/design/mockup.html` in Brave or Chrome (double-click the file in the Comp HQ folder, inside `docs` → `design`). Open Comp HQ in another window beside it. Go through the screens one by one: Briefing, Knowledge Base, Board, My jobs, Team, Calendar, the name picker. If it helps, the same screens as I saw them are in `reports/phase-4/screens/` (pictures with the mockup's own example jobs). | Each screen reads as the same design as the mockup. |
| **O4.2** | On **Tasks → Board**, press **+ Add task**. Type a title, write a description, and pick someone (maybe yourself) under people, then press **Add task**. Then open **My jobs** as that person, and **Team**. | The new job is on the Board, in that person's **My jobs**, and in their column on **Team**. |
| **O4.3** | *This is the one to look at hardest.* Click a job's title on the Board. Read it, press **Edit**, change something (say the size or the description) and press **Save**. Then close the window (the **×**, **Cancel**, Escape, or a click on the page behind it). | The window is as easy to use as you hoped. Closing it puts you back on the Board exactly where you were. If you've typed something and try to close, it asks before throwing it away. |
| **O4.4** | Use the little card icons for a few minutes: move a job up (the up arrow), give one back (the curling arrow, on a job you're on), and remove one (the bin). Hover to see the tooltips. | The icons are clear enough that your team would use them without being told. (A removed job is in **Removed tasks** and **Restore** brings it back.) |

**Still open from earlier phases** (you asked me to carry on without them, so they're recorded as pending, not skipped):

- **O1.1–O1.4** (Phase 1): installing on TrueNAS and the desktop shortcut; trying the Knowledge Base; moving your HelpScout articles; importing a real Word document.
- **O2.1–O2.2** (Phase 2): the look and feel of Tasks and the Calendar; entering your yearly events and moving your active Trello cards.
- **O3.1** (Phase 3): trying the Saturday Briefing with real jobs and events. (O3.2, your feedback on the design, is what this phase answers.)

Reply with **"all OK"**, or tell me in your own words what isn't right and I'll fix it and send a small update. You can also answer just some of them; the rest stay pending.

## Full gate checklist

See `reports/gates-phase-4.md` — every SPEC gate 4.01–4.54, its plain description, the test that proves it, and the result (all Pass). Phase 1, 2 and 3 gates are in `reports/gates-phase-1.md`, `-2.md` and `-3.md`, and are re-proven by the same full run (gate 4.54).

## Tests corrected

Existing tests that were changed during Phase 4, each because Phase 4 changed the behaviour on purpose or the test itself was wrong (each is recorded in `docs/decisions.md`). No test was skipped, weakened or deleted, no time limit was raised and no retry was added.

1. **The Team view test that said ideas are left out** (D-59). The old rule was a bug: an idea someone was on vanished from everyone's view. The test now checks that assigned ideas do show (and don't count as workload) while finished jobs still don't.
2. **Date displays.** Dates on the Calendar and in a job's details are now shown day first, so the tests that typed or read ISO-style dates (2026-10-01) now use day-first (01/10/2026).
3. **Card and window wording and structure.** Tests that clicked a card title now open the job through the shared helper (the window opens over the Board); tests that looked for `.task-avatar` now look for the new circle part; the "Move up / Move down" test finds the icon buttons by their names; the Team number test reads the figure rather than the text "1."; the Briefing's "Everything" became "Everyone", and its people are read from the circle's title; the Calendar month-name test reads the page heading; the filter test uses the single Everyone / My tasks / person select; the "Move to…" test opens its expander first; and one drag test now scrolls the card into view before dragging.
4. **The design-token tests** were rewritten around the new palette and fonts (D-60 supersedes D-09): the colour-contrast test now lives with the people palette.
5. **Two things my new tests found** that I fixed in the app rather than the tests: the Calendar's greyed-out days outside the month failed the readability check (D-63), and the Knowledge Base editor's text area and file buttons had no accessible names.
6. **One CI failure, fixed at its cause** (D-61c). The Team page's accessibility test timed out because the test run had piled up hundreds of made-up people. I stopped tests leaving people behind, and made the "who's on this job" list load only when you press it. No limit was raised and there is no retry.

None of these affect anything you use.

## New design decisions (`docs/decisions.md`)

D-59 to D-63c. In plain terms:

- **D-59:** an idea can have people on it; it shows for them but doesn't add to their workload.
- **D-60:** the palette and fonts (orange as a fill only, deep orange as text, ink-navy text on orange), replacing the old font choice.
- **D-61 / D-61b / D-61c:** the day-first date box and the shared card parts; the new Add-task page and one filter box on the Board; short dates on cards; people lists load only when asked.
- **D-62 / D-62b:** the task window (reading, editing, adding), and My jobs and Team.
- **D-63 / D-63b / D-63c:** the Calendar (one small colour departure from the mockup so text stays readable), the Knowledge Base, the Briefing and the name picker.

## Screenshots

The screens as they look in use, at 1366×768, in `reports/phase-4/screens/` (jobs and events taken from the mockup's own example, so you can hold each beside `docs/design/mockup.html`): `briefing.png`, `briefing-populated.png`, `who.png`, `who-populated.png`, `tasks-board.png`, `tasks-board-populated.png`, `tasks-board-people-menu.png`, `tasks-new.png`, `task-window-new.png`, `task-window-reading.png`, `task-window-reading-history.png`, `task-window-editing.png`, `tasks.png`, `tasks-populated.png`, `tasks-team.png`, `tasks-team-populated.png`, `tasks-finished.png`, `tasks-removed.png`, `calendar.png`, `calendar-populated.png`, `calendar-list.png`, `calendar-new.png`, `calendar-removed.png`, `kb.png`, `kb-article.png`, `kb-article-highlighted.png`, `kb-search.png`, `kb-search-open.png`, `kb-categories.png`, `kb-new-article.png`, `kb-editor-imported.png`, `kb-archived.png`, the five Settings pages, and `404.png`.

## Version

**1.1.0** — `ghcr.io/stas-comp/comphq:1.1.0` (release: https://github.com/stas-comp/comphq/releases/tag/v1.1.0).
