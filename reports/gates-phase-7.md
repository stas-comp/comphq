# Phase 7 gate checklist

Each row is one Phase 7 gate from SPEC.md A11 (7.01 to 7.91), with its plain description and the test file(s) that prove it. All rows passed in the full CI run https://github.com/stas-comp/comphq/actions/runs/37402952243 (`go`, `browser`, `container` with an upgrade and rollback against 1.3.0, `speed`, `screens`). The tests are named after their gates (`gate 7.xx`), so a gate's title can be found in the file with a search.

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

Owner checks O7.1–O7.6 are pending; see `reports/phase-7-report.md`.
