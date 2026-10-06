# Comp HQ — Phase 7 report (version 1.4.0)

## S12 — end of Phase 7 (v1.4.0)

**Phase 7 is done and v1.4.0 is published** (the image is public; I checked that it can be pulled).

- New jobs start in To do, and you can reorder your own list on My jobs.
- Ideas can be dragged into To do on Team, and every card has the people circles.
- Removing a job can be undone, and the search box on Tasks pages finds jobs.
- Weekly jobs come back every Saturday, and jobs can be linked ("do first" or "related").
- Articles can be put in order, Word pictures come across, and the Knowledge Base home, category pages and editor have their new look.

To update, change the version number in TrueNAS to `1.4.0` (see Updating in the README). *Inbox: 1 item waiting for you in `reports/phase-7-owner-inbox.md`.* There are six things to check yourself, **O7.1–O7.6**, each with what to try and what to expect, below. The full report follows.

## What I need from you

### 1. Update

In TrueNAS, go to **Apps → comphq → Edit**, change the number after `comphq:` to **1.4.0**, and click **Save**. Wait for it to show **Healthy**. Your data carries across untouched, and if you ever want to go back, change it to `1.3.0` (your articles, jobs and links all stay, and come back when you update again). The README's *Updating* and *Undoing an update* sections have the click-by-click steps.

### 2. The inbox: one item

`reports/phase-7-owner-inbox.md` has one item: **the Word document that lost pictures wasn't in `samples\word`** (the folder only held its README). I built the Word fixes for every cause the spec found, with test documents for each shape of picture, but I could not try your real document. Copy it into `samples\word` (nothing private) and tell me; the test suite checks every document in that folder automatically. If it still loses a picture, that becomes a v1.4.x fix.

### 3. The six checks (O7.1–O7.6)

**The Knowledge Base home page and category pages have no mockup** (the design only ever drew the article and the editor), so I built them from the mockup's own parts: the same buttons, cards, panel heads, dates and colours, with nothing new. **O7.5 matters most**: please look at them hardest, and tell me anything that looks out of place next to the rest of Comp HQ.

| Check | What to try | What to expect |
|---|---|---|
| **O7.1** | On **My jobs**, reorder your *Up next* by dragging and with the arrows. On **Team**, drag an idea into someone's *Up next*, and put two people on a job with the circles. | All three feel as natural as on the Board; a dragged card scrolls the page when you hold it near the top or bottom; the list of names is never cut off. |
| **O7.2** | Remove a job and press **Undo**. Then, on the Board, type part of a job's name into the top search box. | The job comes back exactly where it was (the message lasts about ten seconds); the box says **Search jobs** and lists matching jobs as you type. |
| **O7.3** | Set up one real **weekly job** (tick *Repeats every week* when you add it) and finish it. On the next Saturday, look at it. | It is back at the bottom of **To do**, due that Saturday, with its steps unticked, and it is in the Saturday Briefing. (If Comp HQ wasn't opened until later, it goes back the first time it is.) |
| **O7.4** | **Link** two real jobs with **That one first**. | The second shows an orange **WAITING** stamp (hover over it: "Waiting on: …"); the stamp goes when the first is Done; and the second can still be started or finished meanwhile. |
| **O7.5** | Look at the Knowledge Base **home**, a **category** and the **editor**. Put a category's articles in order. Import the Word document that lost pictures before. | It all looks tidier and feels better; every picture is there or clearly marked in a red dashed box that says what it was and what to try; the head and toolbar stay in view while a long article scrolls. |
| **O7.6** | Scroll around the **Board**, **Team** and a long article, down and across. | No white areas anywhere. If you still see one, note the page and what you did and it becomes a fix. |

## Summary

- **What's new:**
  - **Jobs:** a new job starts in **To do**. On **My jobs**, *Up next* can be put in order (arrows or drag), and an idea you're on becomes a job with **Move to To do**; **Team** lists unassigned ideas too, lets you drag an idea into anyone's *Up next*, and has the **people circles** on every card (My jobs too), opening one list of names that is never cut off.
  - **The Board:** a card can be dropped anywhere in a column, the page scrolls while you drag near its top or bottom, and the filter you chose survives a drag.
  - **Undo** after removing a job, with a message at the bottom of the screen; **Search jobs** in the top bar on every Tasks page.
  - **Weekly jobs** (a **WEEKLY** stamp; each Saturday back to the bottom of *To do*) and **linked jobs** (*Do first*, *Then*, *Related*, and a **WAITING** stamp that is only a label).
  - **Knowledge Base:** articles have their own **order** in a category (editing no longer moves them), **every picture in a Word document** comes across, and the **home**, **category pages** and **editor** have the new look.
  - **No white areas:** the strips at the side and bottom of some pages are gone, and the checks that missed them now measure the whole page.
  - **Also:** *Export everything* gains `links.csv` and a **Repeats** column in `tasks.csv`; the README and CHANGES are updated.
- **Automatic checks:** All 72 of this phase's promises (gates 7.01–7.91) passed in the release run https://github.com/stas-comp/comphq/actions/runs/37402952243 (`go`, `browser`, `container` with the internet blocked and an upgrade-then-rollback against 1.3.0 with real links, weekly jobs and ordered articles in the data, `speed`, `screens`, `publish`), and in the full run https://github.com/stas-comp/comphq/actions/runs/37400524492 on the same commit.
- **Tests I corrected:** see "Tests corrected" below. None was weakened.

## Gate checklist (7.01 – 7.91)

| Gate | What it promises | Where it is tested |
|---|---|---|
| 7.01 | On My jobs, a job in Up next can be dragged up or down, and stays where it was dropped after a refresh and on every computer. | `e2e/tasks/myjobs-order.spec.ts` |
| 7.02 | Reordering on My jobs changes the one shared priority order, exactly as reordering on Team does (gate 2.29). | `e2e/tasks/myjobs-order.spec.ts` |
| 7.03 | An idea in Ideas I'm on can be dragged into Up next, which moves it to To do where it's dropped. | `e2e/tasks/myjobs-order.spec.ts` |
| 7.04 | Team's Unassigned column also lists ideas nobody is on, in their own Ideas group below Up next. | `e2e/tasks/team-ideas.spec.ts`, `internal/tasks/lanes_test.go` |
| 7.05 | An idea on Team can be dragged into any column's Up next, its own or another person's. | `e2e/tasks/team-ideas.spec.ts` |
| 7.06 | Every card on Team (Working on now, Up next and Ideas) and on My jobs (including Up for grabs) shows the same people circles as the Board, or… | `e2e/tasks/people-circles.spec.ts`, `e2e/tasks/team-assign.spec.ts` |
| 7.07 | The list of names always opens fully in view: on the last card of a long column, in the right-most column when the columns scroll sideways, and in… | `e2e/tasks/people-circles.spec.ts` |
| 7.08 | Changing people from the list updates the screen without reloading the page and records History exactly as on the Board. | `e2e/tasks/people-circles.spec.ts` |
| 7.09 | Without JavaScript, the circles are a link to a page where people can be put on and taken off, as the Board's already are. | `e2e/tasks/people-circles.spec.ts` |
| 7.10 | A new job starts in To do, at the bottom, unless another column is chosen. | `e2e/tasks/board.spec.ts`, `e2e/tasks/new-default.spec.ts` |
| 7.11 | A new job with people on it appears straight away in their Up next on My jobs and Team, and counts towards their workload. | `e2e/tasks/new-default.spec.ts` |
| 7.12 | On the Board, every column accepts a dropped card anywhere from its heading down to the bottom of the longest column. | `e2e/tasks/board-drag.spec.ts` |
| 7.13 | While a card is being dragged, holding it near the bottom of the window, or near the top just under the top bar, scrolls the page. | `e2e/tasks/board-drag.spec.ts` |
| 7.14 | After a drag on the Board, any filter in use (My tasks, one person, or a word) is still applied, and the filter box still shows it. | `e2e/tasks/board-drag.spec.ts` |
| 7.15 | Removing a job shows a message at the bottom of the screen: Removed "Order toner" · Undo. | `e2e/tasks/undo-remove.spec.ts`, `internal/tasks/handlers_integration_test.go` |
| 7.16 | Pressing Undo puts the job back as it was: the same column, the same place in the order, and the same people, steps and links. | `e2e/tasks/links.spec.ts`, `e2e/tasks/undo-remove.spec.ts` |
| 7.17 | This works when removing from a Board card and from a job's own page (which returns to the Board). | `e2e/tasks/undo-remove.spec.ts` |
| 7.18 | Removed tasks is unchanged. | `e2e/tasks/undo-remove.spec.ts`, `internal/tasks/undo_test.go` |
| 7.20 | On every Tasks page (My jobs, Board, Team, a job's own page, Finished tasks and Removed tasks), the search box in the top bar reads Search jobs… | `e2e/design/frame.spec.ts`, `e2e/tasks/search.spec.ts` |
| 7.21 | Results appear as you type, as article results do. | `e2e/tasks/search.spec.ts`, `internal/tasks/search_test.go` |
| 7.22 | Finished jobs are found too, including those tucked away after 14 days, and are marked Finished. | `e2e/tasks/search.spec.ts`, `internal/tasks/search_test.go` |
| 7.23 | Clicking a result opens the job's window (or its page, without JavaScript). | `e2e/tasks/search.spec.ts` |
| 7.24 | The bottom of the results offers Search articles instead, which runs the same words as an article search. | `e2e/tasks/search.spec.ts` |
| 7.25 | With the test library loaded, job results appear within 1 second of the last keystroke. | `e2e/speed/tasks.speed.spec.ts`, `internal/tasks/search_test.go` |
| 7.26 | The Board's own Filter by word box works exactly as before (gate 2.07). | `e2e/tasks/search.spec.ts` |
| 7.30 | The add-a-task window and the job's edit form have a tick-box: Repeats every week (back in To do each Saturday). | `e2e/tasks/weekly.spec.ts` |
| 7.31 | Ticking it on a job with no due date sets the due date to This Saturday (today, on a Saturday). | `e2e/tasks/weekly.spec.ts`, `internal/tasks/weekly_test.go` |
| 7.32 | At the start of each Saturday, every weekly job in Done goes back to the bottom of To do, due that Saturday, with the same people and all its… | `e2e/smoke/weekly.spec.ts`, `e2e/tasks/weekly.spec.ts` |
| 7.33 | A weekly job that wasn't finished by Saturday stays where it is, with its old due date, so it shows OVERDUE. | `e2e/tasks/weekly.spec.ts`, `internal/tasks/weekly_test.go` |
| 7.34 | A weekly job never goes to Finished tasks. | `e2e/tasks/weekly.spec.ts`, `internal/tasks/weekly_test.go` |
| 7.35 | The reset happens even if Comp HQ was switched off at midnight. | `e2e/tasks/weekly.spec.ts`, `internal/tasks/weekly_test.go` |
| 7.36 | History shows each reset: "Comp HQ put this back in To do for Sat 10 Oct (weekly)". | `e2e/tasks/weekly.spec.ts`, `internal/tasks/weekly_test.go` |
| 7.37 | Unticking the box makes it an ordinary job again, from then on. | `e2e/tasks/weekly.spec.ts` |
| 7.38 | Export everything gains a Repeats column in `tasks.csv` ("weekly", or empty). | `e2e/settings/export.spec.ts`, `internal/tasks/export_test.go` |
| 7.39 | Weekly jobs survive an upgrade from 1.3.0 and a rollback back to 1.3.0. | `e2e/smoke/weekly.spec.ts`, `internal/tasks/weekly_test.go` |
| 7.40 | A job's window and its page have a Linked jobs section under Steps, with a Link a job control. | `e2e/tasks/links.spec.ts`, `e2e/tasks/task-window-read.spec.ts` |
| 7.41 | Linked jobs are shown in three groups: Do first (jobs this one is waiting on), Then (jobs waiting on this one) and Related. | `e2e/tasks/links.spec.ts`, `internal/tasks/links_test.go` |
| 7.42 | A job with an unfinished Do first job shows a WAITING stamp on its card on the Board, My jobs and Team. | `e2e/tasks/links.spec.ts` |
| 7.43 | WAITING is only a label. | `e2e/tasks/links.spec.ts` |
| 7.44 | A link can be removed with the remove icon (gate 4.11), and both jobs lose it. | `e2e/tasks/links.spec.ts` |
| 7.45 | These are refused with a plain message, and nothing changes: linking a job to itself; a second link between the same two jobs; a "do first" link… | `e2e/tasks/links.spec.ts`, `internal/tasks/links_test.go` |
| 7.46 | A removed job's links are hidden while it's removed, and come back when it's restored. | `e2e/tasks/links.spec.ts`, `internal/tasks/links_test.go` |
| 7.47 | Without JavaScript, linking and unlinking work from the job's own page as ordinary forms. | `e2e/tasks/links.spec.ts` |
| 7.48 | Export everything includes `links.csv`: the job, the linked job, what the link means, who linked them and when, and whether the link was removed. | `e2e/smoke/links.spec.ts`, `internal/settings/export_integration_test.go`, `internal/tasks/links_test.go` |
| 7.49 | Links survive an upgrade from 1.3.0 and a rollback back to 1.3.0. | `e2e/smoke/links.spec.ts` |
| 7.50 | On every page, at 1024 × 700 and 1920 × 1080 with the test library loaded, the page as a whole never scrolls sideways. | `e2e/app/white-areas.spec.ts`, `e2e/speed/no-side-scroll.speed.spec.ts`, `e2e/tasks/team-assign.spec.ts` |
| 7.51 | Scrolled to the bottom, and as far right as the page allows, every page shows its paper background and its frame. | `e2e/app/white-areas.spec.ts` |
| 7.52 | A picture wider than the article (for example 3,000 pixels wide) is shrunk to fit, both on the article page and in the editor. | `e2e/app/white-areas.spec.ts` |
| 7.55 | A category's page lists its articles in their own order, the same on every computer. | `e2e/kb/article-order.spec.ts`, `internal/kb/article_order_test.go` |
| 7.56 | Editing an article no longer moves it to the top of its category. | `e2e/kb/article-order.spec.ts` |
| 7.57 | A new article goes to the bottom of its category. | `e2e/kb/article-order.spec.ts` |
| 7.58 | After the upgrade, every category first shows its articles in the order it showed them on 1.3.0 (most recently edited first), so nothing appears… | `internal/kb/article_order_test.go` |
| 7.59 | The order survives an upgrade from 1.3.0 and a rollback back to 1.3.0. | `deploy/test/container-test.sh`, `e2e/smoke/order.spec.ts`, `internal/kb/article_order_test.go` |
| 7.60 | A Word document with 25 different pictures imports with all 25 showing: each is the right picture, in the right place. | `e2e/kb/import-word.spec.ts`, `internal/kb/docx/pictures_test.go` |
| 7.61 | Pictures that sit beside the text (wrapped or floating), pictures grouped together or placed on a drawing canvas, and several pictures in one… | `e2e/kb/import-word.spec.ts`, `internal/kb/docx/pictures_test.go` |
| 7.62 | Pictures in documents made by other programs (Google Docs, LibreOffice, and programs that store pictures under unusual names, with spaces or… | `e2e/kb/import-word.spec.ts` |
| 7.63 | A picture in the same paragraph as the document's title is kept, at the top of the article. | `e2e/kb/import-word.spec.ts`, `internal/kb/docx/pictures_test.go` |
| 7.64 | A picture that can't be brought in is never lost without a trace. | `internal/kb/docx/pictures_test.go`, `internal/kb/import_docx_integration_test.go` |
| 7.65 | Pictures in old Windows drawing formats (EMF and WMF, common for Visio drawings, Excel charts pasted as pictures, and clip art) are still marked… | `e2e/kb/import-word.spec.ts`, `internal/kb/docx/docx_test.go` |
| 7.66 | Pictures in page headers and footers, such as a letterhead logo, are still left out, and the import message now says so: "Left out: the page… | `e2e/kb/import-word.spec.ts`, `internal/kb/docx/pictures_test.go` |
| 7.67 | Every Word document in `samples\word` imports with each of its pictures either shown or marked in place. | `internal/kb/docx/pictures_test.go`, `internal/kb/import_docx_integration_test.go` |
| 7.68 | The 20-page, 10-picture document still imports within gate 1.52's time. | `e2e/kb/import-word.spec.ts`, `internal/kb/docx/pictures_test.go` |
| 7.70 | Knowledge Base home: a heading row with Knowledge Base, a New article main button, and Categories and Archived buttons. | `e2e/kb/home-category.spec.ts`, `e2e/screens/populated.spec.ts`, `internal/kb/article_order_test.go` |
| 7.71 | Recently updated is a panel like the Briefing's sections. | `e2e/kb/articles.spec.ts`, `e2e/kb/home-category.spec.ts` |
| 7.72 | A category's page: a "Knowledge Base › Office facts" trail, the category name with its article count, and a New article button that starts an… | `e2e/kb/home-category.spec.ts`, `e2e/screens/populated.spec.ts`, `internal/kb/article_order_test.go` |
| 7.73 | The editor follows the mockup's editor screen. | `e2e/design/kb-home.spec.ts`, `e2e/kb/editor-look.spec.ts` |
| 7.74 | The red dashed boxes for things that couldn't come across look like the mockup's: a bold sentence saying what it was, and a tip underneath. | `e2e/kb/editor-look.spec.ts`, `e2e/kb/import-word.spec.ts` |
| 7.75 | The import message gains the mockup's first line: "Came across: 2 headings, 1 list, 1 table, 3 pictures." | `e2e/kb/editor-look.spec.ts` |
| 7.76 | Nothing the editor does changes. | `e2e/kb/editor-look.spec.ts` |
| 7.77 | The three screens use only the design's shared parts, colours and fonts (B9). | `e2e/design/kb-home.spec.ts`, `e2e/kb/home-category.spec.ts` |
| 7.90 | Every gate from Phases 1 to 6 still passes, unchanged, apart from the planned corrections listed below. | the whole suite | the full run: `go`, `browser` (e2e + a11y), `container` (offline full E2E, upgrade and rollback), `speed`, `screens` |
| 7.91 | Every screen still passes the accessibility check, works from 1024 × 700 up to full HD, and meets the speed limits. | `e2e/a11y/pages.spec.ts`, `e2e/speed/*.speed.spec.ts` | the a11y project (24 pages), the speed project (40 tests, the seeded library now has linked jobs), and axe inside the new specs |

## Owner checks (👤)

| Check | What it covers | Status |
|---|---|---|
| O7.1 | Reorder My jobs, drag an idea into Up next on Team, put two people on a job with the circles. | Pending — see "What I need from you". |
| O7.2 | Remove a job and Undo; search for a job from the top bar. | Pending. |
| O7.3 | A real weekly job, finished, is back in To do and in the Briefing on the next Saturday. | Pending. |
| O7.4 | Link two real jobs; WAITING makes sense and goes away. | Pending. |
| O7.5 | The Knowledge Base home, a category and the editor; ordering; the Word document that lost pictures. | Pending. |
| O7.6 | No white areas on the Board, Team or a long article. | Pending. |

## Tests corrected

Every one of these follows a planned change in SPEC A11 (gate 7.90) or its own gate, and its commit message names the gate. None was loosened to make a failure go away.

| Test | What changed | Why |
|---|---|---|
| Gate 2.02 (`e2e/tasks/board.spec.ts`, `internal/tasks/store_test.go`) | A new job goes to the bottom of **To do**, not Ideas. | 7.10, D-90. |
| Gate 2.25 (`internal/tasks/lanes_test.go`, Team E2E) | **Unassigned** also lists unassigned ideas. | 7.04. |
| Gate 2.28 (`e2e/tasks/team-assign.spec.ts`); gate 2.34's selector | The **Assign to…** button became the people circles; the card-title selector is `.task-card-title a` because the circles are links too. | 7.06, D-92. |
| The sideways-scroll helper (`e2e/helpers/no-side-scroll.ts`) and gate 2.31's test | Measure the whole page (`documentElement` as well as `body`). | 7.50, D-97. The old helper could not see the very overflow it was meant to catch. |
| Gate 4.14 (`e2e/design/frame.spec.ts`) | The search box says **Search jobs** on Tasks pages. | 7.20, D-94. |
| Gate 4.23 (`e2e/tasks/task-window-read.spec.ts`) | The reading face's "no controls" count leaves out the Linked jobs box, as it already left out Steps. | 7.40, D-96. |
| Gate 1.13 (`e2e/kb/articles.spec.ts`) and the Knowledge Base speed test | A Recently updated row reads "Jo · date", not "updated by Jo, date"; the tile's link is its heading, as the tile is no longer one big link. | 7.71, 7.70. |
| Gate 1.49 / 7.65 (`internal/kb/docx` sample test, `e2e/kb/import-word.spec.ts`) | The sample's EMF picture is marked with its own `drawing` kind, and the box is two parts (a bold sentence and a tip). | 7.65, 7.74, D-99. |
| `tasks.csv` header tests (unit and E2E), the 1.1.0-schema migration test | The export has a **Repeats** column; the old-schema test writes its jobs with SQL because of the new column. | 7.38. |
| The Remove / Restore HTTP test | Looks for the card, as removing a job now lands on the Board with the Undo message. | 7.17. |

Not rule changes, but changed so the checks stop failing for the wrong reason (all found by CI): the drag helper waits for the list to be a Sortable list; gate 5.12's steps test waits for the tick to settle; three heavy tests (My jobs and Team with cards, 50 steps in the window, the Team lanes) have a 90 s limit because the container job's machine is slower; gate 7.10's order check and gate 2.04's keyboard test no longer assume nobody else adds jobs to the shared test server's To do column at the same moment.

## New decisions (`docs/decisions.md`)

D-89 to D-100 were settled with you before the build and were not reopened. Two new ones came out of building, and four of the settled ones gained a "Built" note saying how they turned out. **Nothing is marked provisional: there was no question that needed your direction.**

- **D-101** — Task lists drag with SortableJS's own pointer handling. The browser's own drag-and-drop never scrolled the page when a card was held near the top bar, so a card could not be carried up past it (gate 7.13). This corrects a wrong note in D-41.
- **D-102** — The list of names that opens from the people circles is placed against the window by one script shared by the Board, Team and My jobs, so it is never cut off by the lanes' own scrolling (gate 7.07).
- *Built* notes: D-94 (job search is done in Go so accented letters match), D-96 (links in one write, WAITING worked out for a whole list in one query), D-99 (what the Word import counts and how), D-100 (the Knowledge Base screens, the sticky head and the "Came across" counts).

## `samples\word`

The folder held only its own README: **no Word document**. The Word fixes are therefore based on reading the converter and on documents I generated for each shape of problem (25 pictures in a row; names that are prefixes of one another; unusual part names; a picture in the title; pictures in groups, drawing canvases and VML groups; pictures chosen from a `Choice`/`Fallback` pair; an EMF; a header logo). A counting check now runs on every document in that folder: the pictures the document refers to must equal the pictures shown plus the ones marked in place. That is item 1 in the inbox.

## Deferred, and not in this version

Nothing from the plan was deferred or parked; every task P7-01 to P7-19 was built. Not in v1.4 by design (SPEC A12): repeats other than weekly on Saturday; a "do first" link that blocks a job, and links in the Briefing; a separate priority order per person; EMF/WMF, charts and SmartArt turned into pictures, and Word headers and footers; a new look for the Categories, Archived and History pages; searching events.

## Things you should know

- **I could not look at the real screens as you will.** I looked at the pictures the test browser draws (they are in `reports/phase-7/screens/`) and compared computed styles with the mockup, but not on a real monitor. That is what O7.1–O7.6 are for.
- **The upgrade from 1.3.0 and rollback back to it** are proved by the container job in CI (it runs a real 1.3.0 image, writes articles, jobs, weekly jobs and links, upgrades, rolls back and upgrades again); this computer cannot run Docker, so I could not run that part myself.
- **CI was slow and the container job needed several rounds** of fixes to tests (listed above); each failure was a test assuming a quiet shared server or a slow-machine time limit, not a fault in Comp HQ.
- **The first `v1.4.0` tag was withdrawn before anything was published.** Its release run failed because the upgrade-and-rollback test took "the latest tag" as the previous release, and once the new tag existed that was the build itself; one drag test also let go too soon on the slow machine. I fixed the script (it now takes the highest tag below the version being built, which is what its own comment said), re-ran the full checks, and tagged again on the fixed commit. Nothing had reached the public registry from the first attempt.
- One line, `http: superfluous response.WriteHeader`, appears once in about 500 local tests; it comes from a page whose browser had already gone away. I did not chase it further.

## Screenshots

`reports/phase-7/screens/` (from the full CI run): `tasks-board-populated.png` (WEEKLY and WAITING stamps), `tasks-undo-message.png`, `tasks-search-results.png`, `task-window-links.png` (the Linked jobs section), `tasks-board-scrolled.png`, `kb-home.png`, `kb-category.png`, `kb-editor-imported.png` (the new editor with the "Came across" line and the new boxes), and every other screen as before.

## Version and updating

Version **1.4.0**. In TrueNAS change the number after `comphq:` to `1.4.0` (README, *Updating*); to go back, set it to `1.3.0` (README, *Undoing an update*). The release is the tag `v1.4.0`, run https://github.com/stas-comp/comphq/actions/runs/37402952243.
