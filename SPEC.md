# Comp HQ — Specification

**Status:** Part A approved by the owner · 14 September 2026
**Based on:** `docs/intent/staff-hq.md` (confirmed by the owner, 14 September 2026)

This document has two parts:

- **Part A — Direction and Gates** is for you, the owner. It says what will exist, how it will look and behave, and the exact checks that prove each piece works. Please read all of Part A.
- **Part B — Technical Direction** is for the automated build agent. You don't need to read it.

---

## Decisions made while writing this spec

The interview left seven questions open. Here is what you decided, plus a few things that came up along the way.

| # | Question | Decision |
|---|---|---|
| 1 | Backup | TrueNAS snapshots **plus** an in-app **"Export everything"** button as a second safety net. |
| 2 | Look and feel | **Bold and distinctive**, with a **signal orange** accent (details in A10). |
| 3 | Name list | **Anyone** can add, rename or remove names from a simple Settings screen. |
| 4 | Tasks | Columns **Ideas → To do → In progress → Done**. Finished tasks tuck away after a while but are never deleted. **To do can be put in priority order** (top = most important). |
| 5 | Office computers | Windows, with **Brave or Chrome**. |
| 6 | Address | You wanted a friendly address, or no fuss if that was hard. **It's harder than it sounds**, because the NAS's own admin page already uses the normal web address slot, and giving the app a second name needs extra add-ons that could break. So the desktop shortcut uses the NAS's number address. Nobody types it or sees it day to day. |
| 7 | HelpScout move | Manual copy-and-paste is fine (fewer than 20 articles). |
| — | Delegating | A **Team view** in Tasks, open to anyone. It shows what each person is working on and their workload. Workload is judged by job **size** (Small, Medium, Large), not a count, because one big job can outweigh five small ones. There are no automatic "too busy" warnings; the person in charge judges. Reordering someone's jobs uses the same shared priority order as the board. |
| — | Tasks home screen | Tasks opens on **My jobs**: your own jobs, next to **Up for grabs** (jobs and ideas nobody has taken). Anyone can drag one into their own list to take it. The full Board and the Team view are one click away. |
| — | Word documents | Articles can come from Word. **Import from Word** turns a Word document (.docx) into an article in the editor, keeping its headings, lists, tables and pictures, ready to check and publish. It works one way: once imported, the article is edited in Comp HQ. |
| — | Where updates come from | New versions of the app are delivered through your existing **GitHub** account, **stas-comp** (a website that stores the app's code and publishes updates). Staff never need it, and the app never needs it to run. It's only used when an update is installed. |
| — | "Not secure" label | Office-network apps like this one don't use the padlock certificate that public websites use, because certificates expire and break things. The app window may show a small **"Not secure"** note. It's expected and harmless on your own network. Stability wins here. |
| — | App name (14 Sep 2026, while planning the build) | The app is called **Comp HQ**, not Staff HQ, everywhere: the wordmark, window name, messages, README, and the names you see in TrueNAS and GitHub (`comphq`). Only the name changed; every gate means the same as before. The build folder on the build computer keeps its old name. |

---

# Part A — Direction and Gates

## A1. What we're building

**Comp HQ** is your team's own web app, running on your TrueNAS server in the office. It replaces HelpScout Docs and Trello. There are no logins: staff open it from a desktop shortcut, and it appears in its own window like a normal program, with no browser tabs or address bar.

It has four sections:

1. **Saturday Briefing**: the first screen. It shows what must be done this Saturday (everything due before next Saturday), what's happening this week, and big events coming up early enough to prepare.
2. **Knowledge Base**: your library of how-to articles and office facts, with search that takes you straight to the right paragraph.
3. **Tasks**: a board where anyone can post jobs, assign them, put them in priority order, and follow them through to done.
4. **Calendar**: events and deadlines, including yearly, monthly and weekly repeats. Each event has its own setting for how early it appears in the briefing.

Everything is stored on the NAS. It works with no internet connection, costs nothing to run, and uses no AI.

**How you'll know it works:** at the end of each build phase, the build agent gives you a checklist of the gates below (A11), each marked passed by an automatic test, plus screenshots. You only personally check the few items marked **👤 Owner check**.

## A2. Design priorities, in your order

**1. Ease of use: intuitive and attractive.**
- Every page shares the same frame, so nobody gets lost.
- **Buttons say what they do.** Icons alone are allowed only for four small, repeated controls on a
  card or row — **move up**, **move down**, **remove** and **give back** — each with a hover tooltip
  and a spoken name. Everything with a consequence (Take it, Start, Done, Publish, Archive, Save,
  Add task, Add event) keeps its words. *(Changed at v1.1, 19 Sep 2026, at the owner's request; the
  rule is the one the design mockup already follows.)*
- No page ever needs special knowledge. The article editor works like Word or Google Docs.
- Anything you can do by dragging, you can also do with a button.
- Messages are in plain English ("Someone else changed this article while you were editing"), never error codes.

**2. Stability: simple code that works all the time.**
- The app is one self-contained program, stored in one folder on the NAS.
- It uses well-established technology that's known for not changing under your feet, with every version fixed so nothing updates by surprise.
- No outside services, internet, fonts or images are loaded from elsewhere.
- Features are kept deliberately modest. When in doubt, the simpler version is built.

**3. Flexibility: easy to adjust in a few months.**
- Each section is built as its own separate piece, so changing Tasks can't break the Knowledge Base.
- Colours, fonts and wording live in one place each.
- Updates change your stored information safely and automatically, and can be undone.

**4. Design: bold, themed, distinctive.**
- A strong, recognisable "HQ" look (see A10), applied to the frame, buttons and labels.
- Reading areas stay calm and very readable.

**When priorities clash, the higher one wins.** Examples already decided:

| Clash | Winner | What that means |
|---|---|---|
| Friendly address vs. reliability | Stability | The shortcut uses the NAS's number address. |
| Padlock certificate vs. certificates that expire and break | Stability | A small "Not secure" note may appear. |
| Live, instant updates between computers vs. simple code | Stability | The board and briefing refresh within 60 seconds, or straight away when you click back into the window. |
| Bold colours everywhere vs. easy reading | Ease of use | Bold colours go in the frame, buttons and labels. Article text sits on a calm background. |
| Fancy animations vs. speed and simplicity | Stability | Movement is minimal and never makes you wait. |
| Search that forgives typos vs. simple code | Stability | Search understands word forms ("printers" finds "printer") and part-typed words, but not misspellings. |

## A3. Things that are true everywhere

- **Anyone can do anything.** There are no roles and no permissions.
- **Nothing is permanently deleted from inside the app.** Articles are archived, and tasks and events are removed into a "Removed" list. All of them can be brought back.
- **Every change records who made it and when**, using the name picked on that computer.
- **Dates look like "Sat 19 Sep 2026"** and times like **"14:30"**. "Today" follows the time zone chosen when the app is installed.
- **Works in Chrome and Brave on Windows**, in windows from small laptop size (1024 × 700) up to a full HD screen. Phones and tablets are not supported.
- **Speed:** with far more content than you'll ever have (500 articles, 2,000 tasks, 300 events), every page is ready within 1.5 seconds and search results appear within 1 second.

## A4. The app frame and name picker

**What a staff member can do**
- A staff member can open Comp HQ from a desktop shortcut into its own window.
- The first time a computer opens Comp HQ, a staff member picks their name from a list. That computer then remembers them.
- A staff member can add their own name if it's missing.
- A staff member can switch to a different name at any time ("Change" next to their name).
- A staff member can reach every section, Settings, and search from any page.

**What it looks and feels like**
- A bold dark sidebar on the left holds the Comp HQ wordmark and the sections: Briefing, Knowledge Base, Tasks, Calendar, and Settings at the bottom. The current section is lit up in the accent colour.
- A slim bar across the top holds the search box (always there) and "You: Sam · Change".
- *(v1.3)* The sidebar and the top bar stay in place while the page scrolls, so every section, Settings and the search box are always in view, however long the page (gates 6.01–6.05, B12.1).
- The name picker is a full screen reading **"Who's using this computer?"**, with each name as a large button in alphabetical order, and a smaller **"My name isn't here"** link underneath.

## A5. Knowledge Base

**What a staff member can do**
- A staff member can browse articles by category, and see what was recently updated and by whom.
- A staff member can create, rename, reorder and delete categories. A category can only be deleted when it's empty, and the app explains why otherwise.
- A staff member can write and publish an article using a toolbar: two heading sizes, bold, italic, bulleted and numbered lists, links, simple tables, and images.
- A staff member can paste screenshots or drag in photos. Images are saved on the NAS.
- A staff member can paste content from a web page or Word document and keep the basic formatting (headings, bold, lists, links, tables). Odd fonts, colours and sizes are removed so everything matches. Pictures come across from web pages. Pictures in pasted Word content often can't (Word doesn't hand them to the browser), so the editor marks them and suggests Import from Word.
- A staff member can **Import from Word**: choose a Word document (.docx), or drag one onto the editor. It appears in the editor with its headings, bold and italic, lists, links, tables and pictures, ready to check and publish. Pictures are saved on the NAS. Anything that can't come across, such as a chart, is marked in place so it can be fixed. Importing into an existing article keeps the old version in its history.
- A staff member can edit any article, see its history (who changed it and when), view an older version, and restore it.
- A staff member can archive an article, which hides it from browsing and search. They can also see all archived articles and restore any of them.
- A staff member can search from any page and see the matching passage with their search words highlighted. Clicking a result opens the article scrolled to that passage, with it highlighted.

**What it looks and feels like**
- The Knowledge Base home shows category tiles with article counts, and a "Recently updated" list beside them.
- Articles read like a clean, well-set document: comfortable text size, clear headings, images that fit the page.
- The editor looks like the article itself, with a toolbar across the top (ending in a bold **Import from Word** button), **Publish** and **Cancel** buttons, and a warning if you try to leave without saving.
- Search results appear in a panel under the search box as you type. Each shows the title, category, and a short passage with your words highlighted in the accent colour.

## A6. Tasks

**What a staff member can do**
- A staff member can add a task with a title, and optionally notes, one or more people, a due date, a size (Small, Medium or Large; Medium if not chosen), and a starting column.
- A staff member can move a task between **Ideas**, **To do**, **In progress** and **Done**, by dragging or with buttons.
- A staff member can put **To do** in priority order: drag cards up or down, or use "Move up / Move down". The top card is the most important, and the order is the same on every computer.
- A staff member can assign any task to anyone, and change it later.
- A staff member can show just **My tasks**, one person's tasks, or tasks matching a word.
- A staff member can open the **Team view** to see, for each person, what they're working on now and what's up next, with a picture of how much work that adds up to (a large job counts as four small ones).
- A staff member can drag an unassigned job onto a person to hand it to them, or move a job from one person to another.
- A staff member can reorder a person's upcoming jobs to set their priorities for them. This changes the same shared priority order the To do column uses.
- A staff member opening Tasks lands on **My jobs**: their own jobs and workload, next to **Up for grabs**, the jobs and ideas nobody has taken yet.
- A staff member who has finished their jobs can take a new one by dragging it from Up for grabs into their own list, or by pressing **Take it**. Taking an idea also moves it to To do.
- A staff member can **Give back** a job they can't do. If nobody else is on it, it returns to Up for grabs.
- A staff member can open a task to edit it and see its activity ("Sam moved this to In progress · Sat 19 Sep").
- A staff member can find finished tasks in **Finished tasks**, and reopen one.
- A staff member can remove a task, and restore it from **Removed tasks**.

**What it looks and feels like**
- Four columns side by side. Cards show the title, people's initials in coloured circles, and the due date. A bold **OVERDUE** stamp appears when a task is past its date.
- The **To do** header says "Top = most important".
- Cards that have sat in Done for more than 14 days quietly leave the board, which keeps it tidy.
- **Team view:** one column per person, with an **Unassigned** column first. Each column starts with the person's name and a row of blocks showing their workload (small job = 1 block, medium = 2, large = 4). Below that are **Working on now** (their In progress jobs) and **Up next** (their To do jobs, numbered in priority order). There are no warning colours or limits: the person in charge decides who can take more.
- **My jobs** (the screen Tasks opens on): two areas side by side. On the left is **My jobs**, which looks like your own column from the Team view (your workload blocks, Working on now, Up next). On the right is **Up for grabs**, listing jobs nobody has taken, then ideas nobody has taken, each with a **Take it** button. A switch at the top reads **My jobs · Board · Team**.

## A7. Calendar

**What a staff member can do**
- A staff member can view a month at a time (weeks start **Sunday** since v1.3, with **Saturdays highlighted**), or a list of the next 12 weeks.
- A staff member can add an event with: title, date, optional end date (for events lasting several days), all-day or a start time (and optional end time), notes, how it repeats (never, weekly, monthly, yearly) with an optional "until" date, and **how far ahead it should appear in the Saturday Briefing** (for example "6 weeks").
- A staff member can change **just one occurrence** of a repeating event (for example, move this year's concert) without affecting other years, or change the whole series.
- A staff member can cancel just one occurrence.
- A staff member can remove an event, and restore it from **Removed events**.
- A staff member can see task due dates on the calendar, and click one to open the task.

**What it looks and feels like**
- A clean month grid. Events are solid accent-coloured chips. Task deadlines are outlined chips with a tick-box icon, so the two are never confused.
- Clicking a day (since v1.3, anywhere empty in its box, not only its number) starts a new event on that date. Every date box in the app opens a small day-first calendar to pick from, and typing still works. Clicking a repeating event asks **"Change just this one"** or **"Change all"**.

## A8. Saturday Briefing

**The rules, in plain language**
- The briefing is always **for a Saturday**: today if it's Saturday, otherwise the coming Saturday.
- **Must be done today** (on other days: "Must be done this Saturday"): every unfinished task due on or before the Friday after that Saturday, including overdue ones. That Saturday is the last chance before they're due.
- **This week**: events happening from today up to that Friday.
- **Coming up**: later events whose "show ahead" time has started. For example, a campaign on 31 October with 6 weeks' notice first appears on the briefing for Saturday 19 September.

**What a staff member can do**
- A staff member sees the briefing as soon as they open Comp HQ.
- A staff member can see which items are theirs (marked **YOURS**) and switch to "Just mine".
- A staff member can click any item to open its task or event.

**What it looks and feels like**
- A big date headline: **"Saturday 19 September — today's briefing"**.
- Three bold sections, each item a card with a stamp: **TODAY**, **OVERDUE**, **MON 21 SEP**, **IN 5 WEEKS**.
- Event notes show on the card, so "Exams Monday — print exam papers today" is right there.
- Empty sections say something friendly, for example "Nothing due before next Saturday."

## A9. Settings

- **People**: add, rename, or remove names. A removed name disappears from the picker and assignment lists, but old history still shows it.
- **Export everything**: downloads one file containing every article as a readable page (with images), the tasks and calendar as spreadsheets, and a full copy of the data.
- **Backups**: shows when the last automatic nightly backup happened. If it's more than 2 days old, a warning appears across the top of every page until it's fixed.
- **Content check**: lists any article with an image that isn't stored on the NAS. Once HelpScout is moved over, this should say 0.
- **Set up this computer**: step-by-step shortcut instructions for Chrome and Brave, with the exact text to paste and a **Copy** button.
- **About**: the version number.

## A10. Look and feel — "Bold and distinctive"

- **Signature look:** a deep ink-navy frame with **one bold accent colour** for the current section, main buttons, and stamps. You chose a vivid **signal orange**. It's set in one place, so it can be changed later in minutes.
- **Reading areas:** a warm off-white "paper" background with dark text at a generous size, so long procedures are easy on the eyes.
- **Type:** a confident, characterful typeface for headings and a very readable one for body text. Both are stored inside the app.
- **HQ motifs:** a "Comp HQ" wordmark, stamp-style labels (TODAY, OVERDUE, YOURS), highlighted Saturdays, and one icon per section.
- **Avoided:** tiny grey text, busy gradients, stock illustrations, animations that make you wait.
- Light theme only.
- **From v1.1 the look is no longer described in words alone.** `docs/design/mockup.html` is the
  agreed design, screen by screen; it opens in any browser. Where this spec and the mockup disagree
  about appearance, **the mockup wins**. Where they disagree about *behaviour*, **the spec wins** —
  the mockup is filled with made-up content and doesn't show every state. The exact colours, fonts
  and parts taken from it are fixed in B9.

## A11. Gates

Each gate is a pass/fail statement. The build agent checks every gate itself with automatic tests, except the **👤 Owner checks**, which are grouped at the end of each phase. A phase is only finished when all of its gates pass, **and all gates from earlier phases still pass**.

"Test library" means the large set of made-up content used for speed tests: 500 articles, 2,000 tasks, and 300 events.

### Phase 1 — The frame, names, Knowledge Base, and installing on the NAS

**Names and the frame**
- **1.01** Opening Comp HQ on a computer that hasn't used it before shows "Who's using this computer?" with every current name as a button, in alphabetical order.
- **1.02** After picking a name, the top bar shows "You: *name*". Closing and reopening the app on that computer goes straight in with the same name.
- **1.03** Clicking "Change" shows the name list again, and picking a different name updates the top bar.
- **1.04** "My name isn't here" lets you type a name. It's added, selected, and shows in the picker on other computers.
- **1.05** In Settings → People, renaming a person changes the name everywhere it appears, including old history. Removing a person takes them off the picker and assignment lists, but old history still shows their name.
- **1.06** If the name chosen on a computer is removed, that computer shows the picker the next time a page opens.
- **1.07** Every page shows the same frame: wordmark, the four sections, Settings, search box, and current name. The current section is visibly marked.
- **1.08** In this phase, Briefing, Tasks and Calendar each show a friendly "Coming soon" page, never an error.
- **1.09** No page needs sideways scrolling at any window size from 1024 × 700 to 1920 × 1080.
- **1.10** Every page passes an automatic accessibility check with no serious problems (readable colour contrast, labelled buttons and fields, usable by keyboard).
- **1.11** A web address that doesn't exist shows a friendly "Page not found" inside the normal frame, with a link to the Knowledge Base.

**Knowledge Base library**
- **1.12** A staff member can create, rename and reorder categories. Deleting a category with articles in it is refused with a plain explanation, and deleting an empty one works.
- **1.13** The Knowledge Base home shows categories in the chosen order with article counts, plus the 10 most recently updated articles with who updated them and when.
- **1.14** Choosing a category, typing a title and content, and pressing Publish makes the article appear in its category straight away.

**Editor**
- **1.15** The toolbar offers two heading sizes, bold, italic, bulleted list, numbered list, link, table (with add/remove rows and columns), and image. Each one shows correctly in the published article.
- **1.16** Pasting a screenshot shows it in the editor within 3 seconds. After publishing, the image displays in the article and is served by Comp HQ itself, not any other website.
- **1.17** Dragging in a JPEG, PNG, GIF or WebP up to 20 MB inserts it. A larger file, or a file that's neither an image nor a Word document, shows a plain message and nothing else changes.
- **1.18** Pasting a sample web page keeps headings, bold, lists, links, tables and images. Pasting from a sample Word document keeps headings, bold, lists, links and tables. Both lose their fonts, colours and text sizes. Pictures in pasted Word content that can't come across are marked in place, with the message "Some pictures from Word couldn't be pasted. Use Import from Word to bring them in."
- **1.19** When pasted content includes images hosted on another website, publishing saves copies on the NAS and the article no longer points to that website. If a copy can't be made, the editor names the failed images and marks them in the article, and the text is still saved.
- **1.20** Leaving the editor with unsaved changes asks "Leave without saving?" first.
- **1.21** If two people edit the same article and both publish, the second person sees "Someone else changed this article while you were editing". Their text is still on screen, and "Publish mine anyway" saves it. The other person's version remains in history.

**History and archive**
- **1.22** Every publish is recorded. The article's History lists each version with name, date and time, newest first.
- **1.23** Opening an old version shows it as it was. "Restore this version" makes it current and adds a new history entry with the restorer's name.
- **1.24** Archiving an article removes it from its category and from search, and lists it in "Archived articles". Restoring puts it back in both.
- **1.25** Nowhere in the app can an article be permanently deleted.

**Search**
- **1.26** The search box is on every page. After typing at least 2 letters, up to 20 matching articles appear within 1 second of stopping typing, most relevant first. A match in the title ranks above a match in the text.
- **1.27** Each result shows the article title, its category, and the matching passage with the search words highlighted.
- **1.28** Searching "toner" shows the printer article with the word "toner" highlighted, and clicking it scrolls to that paragraph, with the paragraph and the word "toner" highlighted.
- **1.29** "printers" finds an article containing "printer", and "ton" finds "toner".
- **1.30** A newly published article, or a new word added in an edit, is found by the very next search. Words removed in the edit no longer find it.
- **1.31** Archived articles never appear in search results.
- **1.32** A search with no matches says "No articles match *words*". Typing symbols such as `" * ( - :` never causes an error.
- **1.33** With the test library loaded, every Knowledge Base page is ready within 1.5 seconds, and search results appear within 1 second.

**Safety net**
- **1.34** Settings → Export everything downloads one file. Unpacked on a computer with no network, its `index.html` opens a contents page listing every article by category (archived ones marked). Every article opens with its formatting and images.
- **1.35** Comp HQ makes an automatic backup every night and keeps the last 30. Settings shows the time of the last backup. If the last backup is more than 2 days old, a warning shows at the top of every page.
- **1.36** Settings → Content check shows how many articles contain images not stored on the NAS, and lists them.
- **1.37** Settings → About shows the version number.

**Running on the NAS**
- **1.38** Comp HQ installs by pasting one provided settings file into TrueNAS's "Install via YAML" screen and filling in the folder and time zone. It runs as a single app with nothing else to install.
- **1.39** After the app restarts (including a NAS reboot), Comp HQ is back on its own within 60 seconds, with all articles, images, history and names intact.
- **1.40** All data (articles, images, history, names, backups) lives in the one Comp HQ folder on the NAS. Removing and reinstalling the app, pointed at the same folder, brings everything back.
- **1.41** With all internet access blocked, every Phase 1 gate still passes (Import from Word included), except copying images from other websites (1.19), which reports clearly that it couldn't.
- **1.42** Installing a newer version keeps all data. Switching back to the previous version still works with that data.
- **1.43** Settings → Set up this computer shows the exact shortcut text for Chrome and for Brave, with this Comp HQ's address filled in and a Copy button. Launching the browser with that text opens Comp HQ in a window with no tabs or address bar.
- **1.44** A plain-language `README` exists covering: opening Comp HQ, setting up a computer, updating, undoing an update, backing up, restoring, and "it won't open — what now?"

**Import from Word**
- **1.45** The editor toolbar has an **Import from Word** button that opens a file chooser for Word documents (.docx). Dragging a .docx file onto the editor does the same. A .docx downloaded from Google Docs imports too.
- **1.46** Importing the sample Word document puts it in the editor **without publishing anything**. It keeps Heading 1 as the large heading, Heading 2 and smaller as the small heading, bold, italic, bulleted and numbered lists (including indented levels), links, and tables (including merged cells). Fonts, colours, underlining and text sizes are removed.
- **1.47** Pictures in the document appear in the editor in their places. After publishing, each one is served by Comp HQ itself, and its Word "alt text" (the picture description) is kept.
- **1.48** If the document starts with a paragraph in Word's Title or Heading 1 style, that text becomes the article title. Otherwise the file name (without ".docx") does. The title can be changed before publishing.
- **1.49** Anything that can't come across is shown in its place as a red dashed box saying what it was. This covers charts, SmartArt, drawn shapes, equations, and pictures in a format such as EMF. A message lists what came across and what didn't, for example "1 thing couldn't be brought in: a chart." Text in text boxes is kept, and tracked changes appear as if accepted. Comments, footnotes, headers and footers are left out, and the message says so. Until the boxes are removed, Settings → Content check lists the article.
- **1.50** Importing when the editor already has content first asks "Replace what's in the editor with this document?" Cancel changes nothing. Importing into an existing article and publishing adds a new version to its History, so the old version can be restored.
- **1.51** A file Comp HQ can't read (an old .doc, a password-protected document, a PDF, or a damaged file) shows: "Comp HQ can't open this file. Open it in Word, remove any password, choose File → Save As → Word Document (.docx), then try again." A file over 50 MB is refused with a plain message. Nothing else changes.
- **1.52** A 20-page document with 10 pictures is in the editor within 10 seconds of choosing it.

**👤 Owner checks — end of Phase 1**
- **O1.1** Following the README, you install Comp HQ on the NAS and open it from a desktop shortcut on an office computer, in its own window.
- **O1.2** Look and feel: using the frame, Knowledge Base, editor and search, you agree it's easy to use and looks bold and distinctive. The signal orange accent works across the app.
- **O1.3** HelpScout move: every HelpScout article is in Comp HQ (ticked off against your list), and Settings → Content check shows **0**. After this, HelpScout can be closed.
- **O1.4** Word import: you import one of your real Word documents that has pictures and a table, and agree it came across well enough to publish with little or no tidying.

### Phase 2 — Tasks and Calendar

**Tasks**
- **2.01** The Tasks **Board** view shows four columns, in order: Ideas, To do, In progress, Done.
- **2.02** Adding a task with just a title puts it at the bottom of Ideas (or the column chosen) for everyone. Notes, people and due date are optional. The creator is recorded.
- **2.03** Cards show the title, assigned people, and due date. An unfinished task past its due date shows an OVERDUE stamp.
- **2.04** Dragging a card to another column moves it, and dragging within a column reorders it. Both are the same on every computer after refresh. Every card also has Move up, Move down and Move to… buttons that do the same without dragging.
- **2.05** The To do header reads "Top = most important". A card moved into To do by button goes to the bottom, and a dragged card lands where it's dropped.
- **2.06** Opening a card allows editing all its details. Its Activity list shows who created it, and who moved it, changed its people, or changed its due date, with dates.
- **2.07** "My tasks" shows only cards assigned to the current person. Choosing a person shows only theirs. Typing a word shows only cards with that word in the title.
- **2.08** A card in Done for more than 14 days no longer shows on the board. It appears in "Finished tasks", newest first, where "Reopen" returns it to the bottom of To do.
- **2.09** "Remove task" hides it from the board and calendar and lists it in "Removed tasks", where it can be restored. Nothing can be permanently deleted.
- **2.10** A change made on one computer appears on another computer's board within 60 seconds, or straight away when that window is clicked back into. A card someone is part-way through editing is never wiped out by this.
- **2.11** A removed person no longer appears when assigning, but still shows on their old tasks, marked "(removed)".

**Calendar**
- **2.12** The Calendar opens on this month: weeks start Monday *(replaced by gate 6.20 in v1.3: weeks start Sunday; D-83)*, Saturdays are highlighted, and today is marked. It has Previous, Next and Today buttons. The List view shows the next 12 weeks in date order.
- **2.13** An event can be added with a title (required), date, optional end date, all-day or start time with optional end time, notes, repeats (Never / Weekly / Monthly / Yearly) with optional "until" date, and "Show in briefing ahead" (not early, or a number of days or weeks; the default is 1 week).
- **2.14** Weekly events repeat on the same weekday. Monthly events repeat on the same date, and an event on the 31st shows on the last day of shorter months. Yearly events repeat on the same date, and 29 February shows on 28 February in other years. Nothing appears after the "until" date.
- **2.15** A yearly Christmas concert on Sat 12 Dec 2026, moved with "Change just this one" to Sat 19 Dec 2026, shows on 19 Dec 2026 and still shows on 12 Dec 2027.
- **2.16** "Cancel just this one" hides only that occurrence, and the others remain.
- **2.17** "Change all" (for example a new title or notes) shows on every occurrence. An occurrence moved on its own keeps its own date.
- **2.18** "Remove event" hides the whole event and lists it in "Removed events", where it can be restored.
- **2.19** Unfinished tasks with due dates show on their date, looking different from events. Clicking one opens the task. Finished and removed tasks don't show.
- **2.20** An event's details show who last changed it and when.

**Both**
- **2.21** With the test library loaded, the Tasks board and a Calendar month are each ready within 1.5 seconds.
- **2.22** Export everything now also includes a tasks spreadsheet and an events spreadsheet. Both open in Excel with clear column names and readable dates.
- **2.23** Every Tasks and Calendar page passes the accessibility and window-size checks (as 1.09 and 1.10).

**Team view (for delegating)**
- **2.24** The Tasks page has a **My jobs / Board / Team** switch. Team shows an **Unassigned** column first, then one column per current person in alphabetical order.
- **2.25** Each person's column lists their unfinished jobs in two groups: **Working on now** (In progress) and **Up next** (To do, numbered in priority order). Ideas and finished jobs don't appear. The Unassigned column lists To do and In progress jobs that nobody is on.
- **2.26** Every task has a size: Small, Medium or Large. New tasks are Medium unless changed, and the size can be changed when editing. Cards on the board and in the Team view show the size, and a size change appears in the task's Activity.
- **2.27** Each person's column shows their workload as a row of blocks (small = 1, medium = 2, large = 4) and a line such as "1 large · 1 medium · 1 small". Someone with one large job shows the same number of blocks as someone with four small jobs. No warning colours or limits appear.
- **2.28** Dragging a job from Unassigned onto a person assigns it to them. Dragging a job from one person to another takes the first person off it and puts the second on, and anyone else on the job stays. The job's column and priority don't change. An "Assign to…" button does the same without dragging.
- **2.29** Reordering a person's Up next jobs (by dragging or with Move up/down) changes the shared priority order. On the board, the moved job sits directly above or below the job it was placed next to, and all other jobs keep their order.
- **2.30** A job with two people shows in both of their columns and counts towards both workloads.
- **2.31** With the test library loaded (2,000 tasks, 10 people), the Team view is ready within 1.5 seconds and passes the accessibility and window-size checks. When there are more people than fit, the columns scroll sideways inside their own area and the page itself doesn't.

**My jobs (the screen Tasks opens on)**
- **2.32** Opening Tasks always shows **My jobs** first, for the name picked on that computer.
- **2.33** My jobs shows two areas. **My jobs** holds the person's Working on now and Up next jobs (numbered by priority) with their workload blocks, matching their Team view column. **Up for grabs** holds jobs nobody is on (In progress and To do, in priority order), then ideas nobody is on. Jobs belonging only to other people, and finished jobs, don't appear. *(From v1.1: ideas the person is on appear in a third group, **Ideas I'm on** — gate 4.48.)*
- **2.34** Dragging a To do job from Up for grabs into My jobs puts the person on it, placed where it was dropped in their Up next. Pressing **Take it** does the same but keeps the job's current place in the priority order.
- **2.35** Taking an idea (by dragging or Take it) puts the person on it and moves it to To do, where it was dropped, or at the bottom when Take it is used. The task's Activity records both changes.
- **2.36** A taken job leaves Up for grabs on every computer within 60 seconds, or straight away when that window is clicked back into. If two people try to take the same job at almost the same moment, the second person sees "Sam has just taken this job", and the job isn't added to their list.
- **2.37** From My jobs a person can move a job to In progress or Done with buttons. A job marked Done leaves My jobs and appears in Finished tasks. **Give back** takes the person off a job, and if nobody else is on it, it returns to Up for grabs.
- **2.38** With the test library loaded, My jobs is ready within 1.5 seconds and passes the accessibility and window-size checks.

**👤 Owner checks — end of Phase 2**
- **O2.1** Look and feel: you agree Tasks (My jobs, Board and Team view) and Calendar are easy to use and match the rest of the app, and that the Team view makes it easy to see who can take on more.
- **O2.2** Real content: you add your yearly events (concert, card campaign, exams, …) with their "show ahead" times, and move your active Trello cards over. You confirm nothing's missing, and Trello can be closed.

### Phase 3 — Saturday Briefing

In these examples, Saturday 19 September 2026 is the briefing Saturday.

- **3.01** Opening Comp HQ shows the Briefing, and the Briefing link in the frame leads there too. The "Coming soon" page is gone.
- **3.02** On a Saturday, the headline reads "Saturday 19 September — today's briefing". On any other day it reads "Briefing for Saturday 19 September", naming the coming Saturday.
- **3.03** "Must be done today" lists every unfinished task due on or before Friday 25 September, overdue ones included and stamped OVERDUE, earliest first, each with its people. A task due Wed 23 Sep is listed, and a task due Sat 26 Sep is not.
- **3.04** On other days of the week the same section is titled "Must be done this Saturday" and follows the same rule for the coming Saturday.
- **3.05** Tasks assigned to the current person are stamped YOURS. "Just mine" shows only those.
- **3.06** "This week" lists events from today up to Friday 25 September, in date order, with their notes. "Exams" on Mon 21 Sep with the note "Print exam papers today" appears with that note visible.
- **3.07** "Coming up" lists later events whose "show ahead" time has started, with how far away they are. A card campaign on Sat 31 Oct with 6 weeks' notice appears on the briefing for 19 September ("in 6 weeks") and not on the briefing for 12 September.
- **3.08** Single changed or cancelled occurrences are respected: a cancelled one never appears, and a moved one appears at its new date.
- **3.09** An event lasting several days that has already started shows in "This week" as "until *date*".
- **3.10** Empty sections show a friendly message instead of disappearing.
- **3.11** Every item links to its task or event.
- **3.12** Changes made elsewhere appear within 60 seconds, or straight away when the window is clicked back into.
- **3.13** With the test library loaded, the Briefing is ready within 1.5 seconds, and it passes the accessibility and window-size checks.

**👤 Owner checks — end of Phase 3**
- **O3.1** On a real Saturday (or a practice run), the briefing shows what you'd expect for the week: nothing missing, nothing surprising.
- **O3.2** Final look-and-feel check across the whole app.

### Phase 4 — Look and feel (v1.1)

Phase 4 changes how the app *looks*, not what it does, with two exceptions: the task window at
4.22–4.29, and the assigned-ideas fix at 4.48–4.51. Every gate below is checked against
`docs/design/mockup.html`. Nothing in Phases 1–3 changes meaning.

**The design system (true on every screen)**
- **4.01** Headings are set in **Big Shoulders Display** — the tall, narrow, heavy capitals of the mockup. The font file is stored inside the app; no font is ever fetched from the internet.
- **4.02** Dates, counts and version numbers are set in **IBM Plex Mono**, with figures all the same width, so a column of dates lines up.
- **4.03** The accent is the vivid orange **#FF6B1A**. Text on an accent fill is ink navy, never white. Accent-coloured *text* on the paper background uses the deeper **#A94000**. Every text-and-background pair in the app passes the 4.5:1 readability check.
- **4.04** Buttons come in three kinds and no others: **primary** (accent fill, ink text, ink outline), **secondary** (white, ink outline), and **small** (compact, for cards and rows). No unstyled browser button appears anywhere.
- **4.05** No raw browser control appears anywhere, including inside the task window. Drop-downs, date fields, text boxes and the search box are all styled to match, and the date field shows and accepts UK order (day first).
- **4.06** Every switch between views — My jobs / Board / Team, Month / List, Everyone / Just mine — uses one shared segmented control: joined boxes with the current one filled ink.
- **4.07** Stamps (TODAY, OVERDUE, YOURS, IMPORTED) are condensed capitals on a coloured field: TODAY accent, OVERDUE red on a pale red field, YOURS outlined in the deep accent.
- **4.08** A person is a coloured circle holding their initials. The same person is the same colour on every screen, and the person using the app gets the accent circle. Several people on one item stack with a slight overlap.
- **4.09** Job size shows as a chip reading SMALL, MEDIUM or LARGE, in three increasingly dark shades.
- **4.10** Workload shows as small blocks — one for a small job, two for medium, four for large — and carries a spoken equivalent ("Workload: 7 blocks — large, medium, small").
- **4.11** Icon-only buttons exist for exactly four things: move up, move down, remove, give back. Each has a hover tooltip and a spoken name. Every other button keeps its words (A2).
- **4.12** Icons are simple line drawings stored inside the app, all the same weight and size. No icon is fetched from the internet.

**The frame**
- **4.13** The sidebar carries the stacked wordmark: a small wide-spaced **COMP** above a large accent **HQ**. The section you're in is a solid accent block with ink text.
- **4.14** The top bar holds the styled search box with a "/" key hint, and on the right "You: *name*" with that person's circle and a Change link.

**Tasks — Board**
- **4.15** The board's controls are one row: the My jobs / Board / Team switch, the Everyone / My tasks / Person filter, the word filter, and a primary **+ Add task** at the right. There is no add-a-task form sitting on the board.
- **4.16** A card shows, in this order: its priority number (To do only, in large accent figures), the title, the due date, the people circles, the size chip, and its icon buttons.
- **4.17** The **To do** column is tinted with the pale accent wash and headed "Top = most important". Column headings are condensed capitals with a count beside them.
- **4.18** Every card carries a **remove** icon. Removing from the card does exactly what removing from the task's own page does: it goes to Removed tasks and can be brought back (A3 — nothing is ever permanently deleted).
- **4.19** A card carries a **give back** icon, shown only to a person who is on that job. Pressing it takes **only that person** off. Anyone else on the job stays on it, and the job returns to Up for grabs only if nobody is left (same behaviour as gate 2.37).
- **4.20** Pressing the people circles on a card opens a short list of names. Picking a name puts that person on the job; picking a selected name takes them off. Several people can be on one job. The board updates without leaving the page, and the change is recorded in the task's Activity like any other assignment.
- **4.21** A finished card shows its title struck through in grey.

**Tasks — the task window**
- **4.22** Pressing **+ Add task** opens the task window, empty and ready for a new job: title, description, size, starting column, due date and people, ending in a primary **Add task** and a Cancel. Description and size can now be set while creating a job, which the board form never allowed.
- **4.23** Pressing a card opens the same window **to read**: the title, the description laid out as text rather than sitting in a box, who's on it, its size, its column and its due date. A primary **Edit** turns it into the form of 4.22, filled in.
- **4.24** The task's history ("Sam moved this to In progress") sits at the foot of the window behind a **History** expander, closed when the window opens.
- **4.25** The window closes with Escape, with a close icon, or by pressing the page behind it, returning you to exactly where you were on the board. Closing it with unsaved changes asks first.
- **4.26** Opening the window moves the keyboard into it and keeps it there while it's open — Tab cannot wander onto the board behind. Closing it puts the keyboard back on the card you opened.
- **4.27** Every task still has its own page at its own address, unchanged. A Calendar link to a task still lands there, and refreshing the window on that page still works.
- **4.28** If the window can't open, nothing is lost: **+ Add task** and a card's title are ordinary links to that page, which does the same job (A2 priority 2 — stability).
- **4.29** Saving from the window updates the board without reloading the whole page, and records the change in the task's history exactly as editing from its page does.

**Tasks — My jobs and Team**
- **4.30** On My jobs, the **My jobs** side is tinted with the accent wash and the **Up for grabs** side has a dashed outline, so it reads as a holding area rather than someone's list. The drop target reads "Drop a job here to take it".
- **4.31** On Team, the **Unassigned** lane has a dashed outline and no fill, your own lane is tinted with a YOU stamp, and every lane head shows the person's circle, their name in condensed capitals, their workload blocks and a summary line ("1 large · 1 medium · 1 small").
- **4.32** Take it, Give back, Start and Done ✓ stay as worded small buttons on both screens, and a card on either screen opens the same task window as 4.23.

**Calendar**
- **4.33** Saturday's column is tinted with the accent wash and its day heading is a solid accent block — office day is unmistakable at a glance.
- **4.34** Ordinary days, Saturdays and days outside the month are three clearly different shades, and no two of them can be confused. *(Today they are near-identical — this gate fixes a real readability fault.)*
- **4.35** Today's date number sits in a filled ink-navy chip.
- **4.36** Events are solid ink-navy chips. Task due dates are white chips outlined in ink with a small empty square, red when overdue. The later days of a multi-day event are a lighter navy.
- **4.37** A legend under the grid explains the two kinds of chip and the Saturday tint.
- **4.38** Previous, Today and Next are secondary buttons, beside a Month / List segmented switch and a primary **+ Add event**.

**Knowledge Base**
- **4.39** An article's title is large condensed capitals; its section headings are the same face, smaller. The body stays the calm, very readable face at a generous size on the paper background (A2 priority 1 beats priority 4 inside reading areas).
- **4.40** Article tables have a solid ink-navy header row with white condensed capitals.
- **4.41** The byline row reads "Updated by *name* · *date*", with History on the right, then Archive, then a primary **Edit**.
- **4.42** Search results appear below the box as a bordered panel with a shadow: each result shows its category in small capitals, its title, and the matching words highlighted in accent. The selected result is tinted. Opening it highlights and outlines the found paragraph in the article, with a "Clear highlights" link.
- **4.43** The editor's formatting controls sit in one bordered bar, with **Import from Word** as a primary button with its icon at the right-hand end. The import summary appears as a bordered notice with an IMPORTED stamp. Anything that couldn't be brought in is a red dashed box in place.

**Saturday Briefing**
- **4.44** The date is a very large condensed heading with the day and month in the deeper accent.
- **4.45** Each panel heading is condensed capitals over a thick ink rule, with its count beside it.
- **4.46** Items are white cards: title, then date and people, with stamps aligned right. An event's note sits in a pale accent strip across the bottom of its card. An overdue item's card is outlined in red.

**Name picker**
- **4.47** The name picker is a full ink-navy screen with the large wordmark, the question in condensed capitals, and the names as a grid of large buttons that turn accent when pointed at.

**Assigned ideas (the fix)**
- **4.48** A job in the **Ideas** column that has someone on it appears in that person's My jobs, in a third group headed **Ideas I'm on**, below Working on now and Up next. It does not count towards their workload blocks, because it isn't work yet.
- **4.49** The same job appears in that person's Team column under an **Ideas** heading, and does **not** appear in Up for grabs, because somebody is already on it.
- **4.50** An idea nobody is on still appears in Up for grabs exactly as before (gate 2.33), and taking it still moves it to To do (gate 2.34).
- **4.51** Creating a task and choosing people in the same step puts those people on it whatever column it starts in, and it appears straight away in each of their My jobs and in their Team column. *(This is the reported fault: an idea created with people on it was visible on the Board but nowhere else.)*

**Still true afterwards**
- **4.52** Every screen still passes the accessibility check and still works from a small laptop window (1024 × 700) up to a full HD screen, with no sideways scrolling that shouldn't be there.
- **4.53** With the test library loaded, every page is still ready within 1.5 seconds and search still answers within 1 second.
- **4.54** Every gate from Phases 1, 2 and 3 still passes, unchanged.

**👤 Owner checks — end of Phase 4**
- **O4.1** Open `docs/design/mockup.html` in Brave or Chrome next to the real app and go through the screens one by one. You agree each one now reads as the same design.
- **O4.2** Make a task with the **+ Add task** button, giving it a description and putting someone on it. Confirm it appears in their My jobs and in their column on Team.
- **O4.3** Click a task to read it, press Edit, change something and save. Confirm the window is as easy to use as you hoped, and that closing it puts you back where you were.
- **O4.4** Use the new card icons for a few minutes — move a job up, give one back, remove one — and confirm they're clear enough for your team without being explained.

### Phase 5 — Steps inside a job, and looking like an app (v1.2)

Phase 5 adds one feature (steps inside a job — the checklist of the owner's request 4) and finishes
one that was half-built: Comp HQ opens in its own window today, but has never had an icon of its
own. Nothing in Phases 1–4 changes meaning. The design lives in B10 and B11.

**Steps inside a job**
- **5.01** A job can carry a list of steps. The task window shows them under the description, headed **Steps**; a job with no steps shows an **Add a step** line and nothing else.
- **5.02** A step is added by typing its words and pressing Enter or **Add**. The box clears and stays ready, so several steps can be typed one after another without reaching for the mouse.
- **5.03** A step is ticked and unticked by pressing its box. A ticked step shows who ticked it and when, in the short form used everywhere else ("Jane, Sat 19 Sep").
- **5.04** The heading carries the count — **3 of 7 done** — with a small bar beside it, and reads **All 7 done** when every step is ticked.
- **5.05** A step's words can be changed in place, by anyone.
- **5.06** A step can be removed, and an **Undo** appears straight away and puts it back where it was. A removed step is never permanently deleted (A3): it stays in the database and in the export.
- **5.07** Steps can be put in order, by the same up and down icons used on cards (gate 4.11) and by dragging.
- **5.08** Every one of 5.02–5.07 still works with JavaScript switched off, as an ordinary form that reloads the page (A2 priority 2 — stability beats polish).
- **5.09** The job's own page at `/tasks/{id}` shows and edits the same steps the same way, so a refreshed window and a Calendar link still land somewhere whole (gate 4.27 stays true).
- **5.10** A card on the Board, on My jobs and on Team carries a small **3/7** when the job has steps, and carries nothing at all when it hasn't. It turns to a tick when all are done.
- **5.11** History records a step being added, renamed, removed and restored, and does **not** record ticks.
- **5.12** Two people on two computers: ticking a step somebody else ticked a moment ago does not produce an error, and acting on a step somebody else removed says so plainly instead of failing. The screen picks up the other person's change within a minute, or straight away on clicking back into the window, through the existing change counter (B4 "Refresh without live sockets").
- **5.13** A job holds up to **50** steps and a step holds up to **200** characters. Reaching either says so in plain words, and neither loses what was typed.
- **5.14** **Settings → Export everything** includes `steps.csv` — job, step, done or not, who ticked it, when, and whether it was removed — opening straight in Excel like the other two.
- **5.15** Steps survive an upgrade from 1.1.0 and a rollback back to 1.1.0: the rolled-back app ignores them, and upgrading again finds them all still there.
- **5.16** Steps follow the design system: no raw browser control anywhere (gate 4.05), icon-only buttons carry a tooltip and a spoken name (4.11), and the accessibility check passes with a long list open.
- **5.17** A job with 50 steps still opens within 1.5 seconds, and the Board with the test library loaded is no slower than it is at v1.1.0 (gate 4.53 unchanged).

**Looking like an app**
- **5.20** Comp HQ has its own icon: the stacked **COMP / HQ** wordmark, ink navy with the accent orange, drawn to read clearly down to 16 pixels. It is stored inside the app and never fetched from the internet (as with every font and icon, gates 4.01 and 4.12).
- **5.21** It is supplied in every form Windows and the browsers ask for: a multi-size `.ico`, PNGs at 16, 32, 48, 64, 128, 192, 256 and 512, and a maskable 512 with safe margins.
- **5.22** Every page links the icon, so it shows in the browser tab, in history and in bookmarks. At v1.1.0 no page links one at all, which is why desktop shortcuts come out blank.
- **5.23** The manifest names Comp HQ, opens at the Briefing, asks for its own window, carries the navy as its window colour, and lists the icons of 5.21 — so an installed app takes the Comp HQ icon and colour rather than the browser's.
- **5.24** An installed Comp HQ window shows no address bar, no tabs and no bookmarks bar.
- **5.25** **Settings → Set up this computer** is rewritten to lead with the easiest route — installing Comp HQ as an app from the browser's own menu, in Edge or Chrome — then the command-line shortcut (as at v1.1.0, with its Copy button) for Brave and as a fallback, then pinning it to the taskbar, then a **Download icon** button and the three clicks that set the icon by hand if Windows ever shows a blank one. Each route names the browser it's for, and no route assumes the one before it worked.
- **5.26** The README's "Setting up an office computer" section matches the new page, step for step, with a picture of the finished window.

**Still true afterwards**
- **5.30** Every gate from Phases 1 to 4 (1.01–4.54) still passes, unchanged, at the same thresholds.
- **5.31** Every screen still passes the accessibility check and still works from a small laptop window (1024 × 700) up to a full HD screen, with no sideways scrolling that shouldn't be there.

**👤 Owner checks — end of Phase 5**
- **O5.1** On a real Saturday, add steps to two or three real jobs, tick some off, remove one and undo it. Confirm it's obvious enough that nobody needs telling how it works.
- **O5.2** Look at the Board with those jobs on it. Confirm the small **3/7** is useful rather than clutter.
- **O5.3** Set up one office computer from scratch, following only the new **Set up this computer** page. Confirm the icon on the desktop, in the Start menu and on the taskbar is the Comp HQ mark, and that the window has no address bar.
- **O5.4** Alt-Tab between Comp HQ and something else. Confirm Comp HQ looks like its own program.

### Phase 6 — Easier dates, a sidebar that stays put, Sunday first, and search that keeps up (v1.3)

Phase 6 is a set of everyday fixes the owner asked for after v1.2. Nothing in Phases 1–5 changes meaning except the three words of gate 2.12 that gate 6.20 replaces (D-83). The design is in B12.

**The sidebar and top bar stay put**
- **6.01** On every page, the sidebar stays in place while the page scrolls. With the test library loaded, scrolled to the very bottom of the Board in a 1024 × 700 window, every section link, **Settings** and the version number are on screen.
- **6.02** The top bar (the search box and "You: *name* · Change") also stays at the top while the page scrolls, and the live search results still open over the page, fully visible.
- **6.03** Nothing ends up hidden under the top bar. A search result's highlighted passage (gate 1.28) scrolls into view *below* the bar, not behind it. The task window, the list of names on a card, the Undo on steps and drag-and-drop on the Board and Team all still work as before.
- **6.04** If a window is too short to show the whole sidebar, the sidebar scrolls on its own, so Settings can always be reached. The page itself still never scrolls sideways (gate 5.31).
- **6.05** The backup warning, when shown, sits above the top bar and scrolls away with the page (D-80).

**A calendar for every date**
- **6.10** Every date box in the app opens a small calendar when clicked or tabbed into: a job's due date (in the add-a-task window and when editing), an event's date, end date and "until" date, and the dates of a "just this one" change. A small calendar mark inside the right-hand end of each date box shows the calendar is there.
- **6.11** The calendar shows one month, starting on **Sunday**, with Saturday's column shaded as on the Calendar page. Today is marked, the date already in the box is marked, and there are previous and next month controls.
- **6.12** Picking a day fills the box as day/month/year (**25/09/2026**) and closes the calendar.
- **6.13** The calendar has a **Today** button and a **This Saturday** button. This Saturday is today on a Saturday, and otherwise the coming Saturday: the same Saturday the Briefing is about.
- **6.14** Typing still works exactly as before. While you type, the calendar follows: typing 25/12/2026 moves it to December with the 25th marked. Clicking elsewhere or pressing Escape closes it and leaves what was typed alone.
- **6.15** An empty end date or "until" date opens on the month of the start date, not on today's month.
- **6.16** It works from the keyboard alone. The arrow keys move a day or a week, Page Up and Page Down move a month, Enter picks, and Escape closes and returns to the box. A screen reader hears the month's name and each day's full date ("Friday 25 September 2026"). The accessibility check passes with the calendar open.
- **6.17** With JavaScript switched off, every date box is exactly the typed box of v1.2, and still works.
- **6.18** A date can be typed without its year: **25/9**, **25.9** or **25-9** means the 25 September nearest to today (D-82). With JavaScript, the box shows the full date as soon as you move on. Without it, the saved date is the same, because the server applies the same rule. On 22 September 2026: 25/9 → 25/09/2026, 1/7 → 01/07/2026. On 20 December 2026: 5/1 → 05/01/2027.
- **6.19** Everything D-61 refused is still refused, with the same plain message. That includes an American-order date such as **9/25/2026**, and a short date that is not a real day, such as **31/9**.

**The Calendar**
- **6.20** The month view starts the week on **Sunday** (Sun, Mon … Sat). Saturday is the last column and is still highlighted. Previous, Next, Today and the greyed days from the months on either side work as before. *(This replaces "weeks start Monday" in gate 2.12. Nothing else in 2.12 changes.)*
- **6.21** Clicking anywhere empty in a day's box opens a new event with that date filled in. This includes the greyed days from the months on either side. Clicking an event or a job due in the box still opens that event or job, as before.
- **6.22** A day's box shows that it can be clicked: the pointer changes, and a small **+** appears in the box on hover.
- **6.23** From the keyboard, the day number is still a link that does the same thing, and its spoken name says what it does: "Add an event on Tuesday 22 September".

**Search keeps up with typing**
- **6.30** A half-typed last word finds the articles containing words it's the start of. "pay" finds an article containing "payment", and "busin", "generat", "voluntee" and "happin" find "business", "generation", "volunteer" and "happiness". This holds in the live results and on the full results page.
- **6.31** Finished words still find their relatives exactly as before. "printers" finds "printer" and "ton" finds "toner" (gate 1.29, unchanged), and a match in the title still ranks above a match in the text (gate 1.26).
- **6.32** The word a half-typed word found is highlighted in the result's passage and on the article page, as any other match is (gates 1.27 and 1.28).
- **6.33** Publishing, editing and archiving an article are reflected in the very next search for half-typed words too (gates 1.30 and 1.31).
- **6.34** Search survives an upgrade from 1.2.0 and a rollback back to 1.2.0. The rolled-back app searches as 1.2.0 did. On upgrading again, a half-typed word finds an article that was published or edited *while* rolled back.
- **6.35** With the test library loaded, search still answers within 1 second (gate 4.53, unchanged), including for a two-letter last word.

**Still true afterwards**
- **6.40** Every gate from Phases 1 to 5 (1.01–5.31) still passes, unchanged, at the same thresholds. The one exception is the three words of gate 2.12 that gate 6.20 replaces, and D-83 records it.
- **6.41** Every screen still passes the accessibility check and still works from 1024 × 700 up to full HD, with no sideways scrolling that shouldn't be there.

**👤 Owner checks — end of Phase 6**
- **O6.1** Open the Board and scroll to the bottom. Confirm Settings is in view the whole time, and that nothing on the page looks cut off under the top bar.
- **O6.2** Add a job with a due date picked from the calendar, then another by typing just **25/9**. Confirm both feel natural, and that This Saturday picks the Saturday you expect.
- **O6.3** On the Calendar, click an empty part of a day and add an event. Confirm Sunday-first looks right to you and that Saturday still stands out.
- **O6.4** In the search box, slowly type a word from one of your articles. Confirm results appear before you finish the word.

## A12. What's not included

From the confirmed intent:
- AI question-answering, or AI of any kind.
- Email, phone or push notifications. Reminders only appear when the app is open.
- Access from outside the office network.
- Logins, user roles, permissions.
- A public or customer-facing help centre.
- Separate programs installed on each computer.
- Syncing with Google Calendar or Outlook.
- An automatic HelpScout import tool.

Decided while writing this spec (any of these could be added later if real use shows the need):
- A friendly address such as `comphq.local`.
- The padlock "secure" certificate.
- Phone and tablet layouts, and a dark theme.
- Search that forgives misspellings. Search covering tasks and events (it covers articles).
- Repeat rules like "second Saturday of December". Repeats are by date, and single occurrences can be moved.
- Attaching files other than images (such as PDFs) to articles.
- Sending articles back out to Word, or keeping a Word file linked to its article. Import works one way; after that, the article is edited in Comp HQ. (Export everything still gives readable copies of every article.)
- Importing old-style .doc files, PDFs or other formats. These are saved as .docx in Word first.
- Comments or chat on tasks, and sub-tasks that are jobs in their own right. *(Checklists were on this list until v1.2, which adds them as **steps inside a job** — see A11 Phase 5 and B10.)*
- Automatic "too busy" warnings, per-person workload limits, or a separate priority list for each person.
- Restoring backups from inside the app. Restores are done with TrueNAS snapshots, following the README.
- Several named lists of steps on one job, and reusable step templates for jobs that come round every year (v1.2, D-70 and D-72). Steps in the Saturday Briefing, and a "Removed steps" screen (D-73, D-74).
- Giving a step its own person or its own due date. A step is a tick-box; if it needs an owner and a date, it is a job.
- A confirmation before removing a name, and a Removed people list to bring one back (considered for v1.3; the owner left it for later). A blank Calendar day offering "add a job" as well as an event (D-84).

## A13. Your hands-on moments

This is the complete list. Each one comes with click-by-click instructions in the README.

| When | What you do | About how long |
|---|---|---|
| Once, before building starts | Sign in to your existing GitHub account (**stas-comp**) once on the build computer when the build agent asks. | 5 min |
| Once, after the first release | In GitHub, switch the Comp HQ download to "public" so TrueNAS can fetch it without a password. It holds no data. | 3 min |
| Optional, during Phase 1 | Put 1–3 real Word documents (with pictures and tables, nothing private) in the `samples\word` folder inside the Comp HQ folder, so Import from Word is tested on your real documents. | 5 min |
| Once, end of Phase 1 | On TrueNAS: create a folder (a "dataset") for Comp HQ, install the app by pasting the settings file, and set up automatic nightly snapshots (saved copies the NAS can roll back to). | 30 min |
| Once per office computer | Create the desktop shortcut and pick a name. | 3 min each |
| Once, end of Phase 1 | Copy your HelpScout articles into Comp HQ (fewer than 20). | 1–2 hours |
| End of each phase (3 times) | The 👤 Owner checks above, including entering your yearly events and Trello cards in Phase 2. | 30–60 min each |
| Each future update | In TrueNAS, change one version number in Comp HQ's settings. | 2 min |

---

# Part B — Technical Direction

*For the build agent. Part A is the contract. Everything here serves Part A's gates and may be adjusted if a gate is better served another way, as long as the change is recorded in `docs/decisions.md`.*

## B1. Technology choices

| Concern | Choice | Why it serves Stability first |
|---|---|---|
| Server | **Go** (current stable release at build start, pinned with `go` and `toolchain` lines in `go.mod`). Standard-library `net/http` (method and path routing) and `html/template`. | One static binary with no runtime to patch. The Go 1 compatibility promise means little breakage. The standard library covers HTTP, templates, zip, CSV, time zones (`time/tzdata` embedded) and content sniffing. |
| Database | **SQLite** through `modernc.org/sqlite` (pure Go, no C compiler, **FTS5 built in**). WAL mode, `foreign_keys=ON`, `busy_timeout` of at least 5s. | A single file with no database server. FTS5 gives ranked search, snippets and highlighting with no extra service. |
| Pages | Server-rendered HTML with small, plain JavaScript files per section. **No single-page-app framework.** | Fewer moving parts. It survives browser updates, and later agents can change it easily. |
| Rich-text editor | One mature, permissively licensed (MIT/BSD) editor that supports headings, bold, italic, lists, links, tables, a custom image-upload hook for paste and drop, and paste clean-up. For example, TipTap/ProseMirror with the table extension. **Bundle it once into a prebuilt file committed under `web/static/vendor/`**, with its version and checksum recorded. | No build step at runtime or for the owner. The pinned version never changes by surprise. |
| HTML sanitising | `github.com/microcosm-cc/bluemonday`, server-side, with a strict allowlist. The server is the authority and never trusts the editor's output. | Guards against pasted HTML that breaks pages or runs scripts. |
| Word import | Go standard library only (`archive/zip`, `encoding/xml`), in a pure `internal/kb/docx` package. **No new dependency**, and no LibreOffice or pandoc in the image. | Keeps one small binary. The converter is a pure function that's easy to test and can't affect other sections. |
| Drag and drop | Native HTML5 drag-and-drop, or one small vendored library (for example SortableJS, MIT). Buttons must provide the same actions. | Minimal dependency, and the buttons make it testable. |
| Fonts and icons | Font files (SIL OFL) and SVG icons bundled in the binary. **No Google Fonts or CDNs.** | Works offline and never changes. |
| Assets | All templates, CSS, JS, fonts and migrations embedded with `go:embed`. | One file to ship. |
| Container | Multi-stage `Dockerfile`: Go build stage, then a `gcr.io/distroless/static` (or `scratch`) final stage. Base images pinned **by digest**. The binary provides its own `healthcheck` subcommand. | Tiny, with nothing inside that needs updating. |
| Code hosting, CI, images | Private GitHub repository under the owner's existing **`stas-comp`** account. GitHub Actions runs every test on every push, and publishes a **public** image to `ghcr.io` on version tags `vX.Y.Z`. The image contains no data. | Free. TrueNAS pulls a public image with no credentials, and internet is only needed when updating. |
| Tests | Go `testing`, plus **Playwright** (TypeScript, pinned in `package-lock.json`, development only) driving Chromium, with `@axe-core/playwright` for accessibility. | Real-browser proof of every observable gate. |

**Rejected:** Node/Next.js and similar (large, fast-moving dependency trees). Python/Django (fine, but a separate runtime plus more packages to keep patched). PostgreSQL or MySQL (separate server). Meilisearch or Elasticsearch (extra service). SPA frameworks (build complexity, churn). HTTPS with self-signed certificates (installing certificates on each PC, expiry failures). mDNS name publishing on TrueNAS 25.04 (needs extra privileged containers and is experimental).

**Dependency rule:** the Go module list stays short (target at most 5 direct dependencies). Every new dependency needs a one-line reason in `docs/decisions.md`.

## B2. Architecture: independent sections

```
cmd/comphq/            main: config, open DB, run migrations, start scheduler, serve; `healthcheck` subcommand
internal/app/           router, frame layout, middleware (person, origin check, errors), section registry
internal/db/            open, migration runner, backups (VACUUM INTO), schema-version checks
internal/people/        names, picker, person cookie
internal/kb/            categories, articles, versions, blocks/search, images, article export, Word import
internal/kb/docx/       pure .docx → HTML converter (no DB, network or disk)
internal/tasks/         board, ordering, activity, finished/removed
internal/calendar/      events, exceptions, recurrence expansion (pure functions)
internal/briefing/      pure Build(...) function + page; reads tasks & calendar only via small query interfaces
internal/settings/      people screen, export orchestration, backup status, content check, setup page, about
web/templates/<section>/  web/static/<section>/  web/static/vendor/  web/static/theme/
migrations/<section>/   numbered SQL files per section
e2e/                    Playwright tests, fixtures, test-library seeder
deploy/truenas.yaml     the Compose file the owner pastes into TrueNAS
README.md               owner guide (plain language)
docs/decisions.md       short technical decision log
```

**Boundaries:**
- Sections never import each other's internals. Allowed dependencies: every section → `people`, `db`, `app`. `briefing` → the query interfaces exported by `tasks` and `calendar`. `settings` → the export and status interfaces of the other sections.
- Each section registers its routes, navigation entry and migrations with `app`. Adding a section means adding one package and one registration line.
- **All theme values** (colours, fonts, radii, spacing) live as CSS custom properties in `web/static/theme/theme.css`. Swapping the accent colour means editing one file.
- User-facing wording lives in the templates, not scattered through Go code.
- Pages expose a `data-ready` attribute on `<body>` once rendered and their scripts have initialised. Tests and speed checks wait on it.

## B3. Data model

Timestamps are stored as UTC ISO-8601 text. **Calendar dates** (due dates, event dates) are stored as local `YYYY-MM-DD` text with no time zone. "Today" is computed in the container's `TZ`.

**people**
`id` · `name` (unique among active) · `active` (bool) · `created_at`

**kb_categories**
`id` · `name` · `sort_order` · `created_at` · `updated_at`

**kb_articles**
`id` · `category_id` → categories · `title` · `body_html` (sanitised, with block ids) · `status` (`published`|`archived`) · `version_no` · `created_by` → people · `created_at` · `updated_by` · `updated_at`

**kb_article_versions**
`id` · `article_id` · `version_no` · `title` · `category_id` · `body_html` · `action` (`created`|`edited`|`archived`|`unarchived`|`restored`) · `restored_from_version` (nullable) · `edited_by` · `edited_at`

**kb_search** (FTS5 virtual table, `tokenize='porter unicode61'`)
`title` · `body` · `article_id UNINDEXED` · `block_id UNINDEXED`
One row per block of each **published** article: block 0 holds the title, and blocks 1..n hold the text of each block element. Rows are rewritten in the **same transaction** as the article save. Archiving deletes the rows, and unarchiving re-adds them.

**images**
`sha256` (primary key) · `ext` · `mime` · `bytes` · `created_by` · `created_at`
Files are stored at `/data/images/<first 2 hex>/<sha256>.<ext>` and served at `/images/<sha256>.<ext>`. They are never deleted.

**tasks**
`id` · `title` · `notes` (plain text) · `size` (`S`|`M`|`L`, default `M`) · `stage` (`idea`|`todo`|`doing`|`done`) · `position` (integer, unique per stage among visible) · `due_date` (nullable) · `done_at` (nullable) · `removed_at` (nullable) · `created_by` · `created_at` · `updated_by` · `updated_at`

**task_assignees**
`task_id` · `person_id` (composite primary key)

**task_activity**
`id` · `task_id` · `person_id` · `action` (`created`|`moved`|`assigned`|`unassigned`|`due_changed`|`size_changed`|`edited`|`reopened`|`removed`|`restored`|`step_added`|`step_renamed`|`step_removed`|`step_restored`) · `detail` (short JSON) · `at`
The four `step_*` actions arrive in v1.2. There is deliberately **no** action for ticking a step (D-71): a job with twenty steps would otherwise bury its own history, and who ticked what is carried on the step itself.

**task_checklist_items** *(v1.2)*
`id` · `task_id` → tasks · `text` (plain text, ≤ 200 characters) · `position` (integer, unique per task among non-removed rows) · `done_by` → people (nullable) · `done_at` (nullable) · `removed_at` (nullable) · `created_by` · `created_at` · `updated_by` · `updated_at`
Ticked is `done_at IS NOT NULL`, which carries who and when in the same row and needs no separate flag. Removed is `removed_at IS NOT NULL`, the soft delete used everywhere else (A3). `position` is maintained by the move code the way `tasks.position` is, not by a constraint. Indexed on (`task_id`, `position`).
Adding a table is rollback-safe on its own (B6): v1.1.0 never selects from it, so rolling back leaves the rows untouched and upgrading again finds them (gate 5.15).

**events**
`id` · `title` · `notes` · `start_date` · `end_date` (nullable) · `start_time` · `end_time` (nullable, `HH:MM`; null start means all day) · `recurrence` (`none`|`weekly`|`monthly`|`yearly`) · `until_date` (nullable) · `notice_days` (integer, 0 = not early; weeks stored as days ×7, displayed in weeks when divisible) · `removed_at` · `created_by` · `created_at` · `updated_by` · `updated_at`

**event_exceptions**
`event_id` · `original_date` (composite primary key) · `kind` (`moved`|`cancelled`) · `new_start_date` · `new_end_date` · `new_start_time` · `new_end_time` (nullable) · `updated_by` · `updated_at`

**app_meta**
Key/value store: `last_backup_at`, `last_backup_ok`, `last_backup_error`, `min_app_version`.

**schema_migrations**
`section` · `version` · `applied_at` · `app_version`

## B4. Precise behaviours

**Person identity**
- A cookie `comphq_person=<id>` with `SameSite=Lax`, `HttpOnly`, `Max-Age` 400 days, refreshed on every visit.
- Middleware: if the cookie is missing, unknown or inactive, a GET redirects to `/who` (preserving the destination) and a POST shows the picker with a message.
- This is attribution only, not security.

**Article blocks and search**
- On save: sanitise, then assign `data-b="<n>"` to every block element (`p`, `h2`, `h3`, `li`, `blockquote`, `tr`, `figcaption`) in document order, then extract plain text per block, then rewrite the `kb_search` rows.
- Query building: take only Unicode letter and digit runs from the input, wrap each word in double quotes, and give the **last word a `*` prefix**. Words are joined by implicit AND. Fewer than 2 characters in total returns no results. **Never pass raw input to MATCH.**
- Ranking: `bm25(kb_search, 10.0, 1.0)` (title weighted). Take the best-ranked block per article, then the top 20 articles.
- Snippet: `snippet()` on the best block, using private-use Unicode characters as markers. HTML-escape the text, then replace the markers with `<mark>`.
- Result link: `/kb/articles/<id>?q=<query>#b-<block>`. On the article page, the server marks the target block, and a small script scrolls it into view, adds a highlight class, and wraps the matched words within it in `<mark>`. Matched words come from the server via `highlight()` on that block, passed as a data attribute. A "Clear highlights" control removes them.
- Live results: debounce at 250 ms. Pressing Enter goes to a full results page at `/kb/search?q=` with the same content.
- *(v1.3)* The half-typed last word: before `BuildQuery`'s output reaches MATCH, the last word is looked up in the unstemmed vocabulary of `kb_search_words` and becomes `("word"* OR "whole1" OR … )` with at most 30 whole words, most common first. Finished words are unchanged. `kb_search_words` is written wherever `kb_search` is, and rebuilt from `kb_search` at every start-up (D-85, B12.5).

**Dates typed without a year** *(v1.3)*
- Every date field also accepts `d/m`, `d.m` and `d-m`. The year is whichever of last year, this year and next year puts the date nearest to today (`app.Today`, so test mode's fixed date applies). A day that doesn't exist in that year (29/2) is tried in the other two before being refused. Everything D-61 refuses is still refused, with the same message (D-82, gates 6.18–6.19).

**Images**
- Upload: `POST /kb/images`. Limit 20 MB. Type is decided by content sniffing, and only JPEG, PNG, GIF and WebP are accepted (**no SVG**). Store under the SHA-256 name (duplicates collapse to one file).
- Serve with `Content-Type` from the stored mime type, `X-Content-Type-Options: nosniff`, and immutable caching.
- **External images on publish:** for every `<img>` whose `src` isn't `/images/...`:
  - `data:` URLs are decoded and stored.
  - `http(s)` URLs are fetched server-side with a 10s timeout and 20 MB cap. The fetch refuses loopback, private, link-local and multicast destinations (check the resolved IP at connect time) and uses at most 3 redirects. The result is sniffed, stored, and the `src` rewritten.
  - On failure, the image is replaced with a placeholder element (`data-missing-src` holds the original URL). The editor response lists the failures.
- After this step, the sanitiser **drops every `img` not pointing at `/images/`**, so no stored article ever references another host.
- Content check: count published and archived articles containing `data-missing-src`, `data-missing-kind`, or any non-local `img`.
- Pasting (not importing): an `img` whose `src` starts with `file:` (Word's clipboard pictures) is replaced client-side at paste time with a `data-missing-kind="picture"` placeholder, and the editor shows the 1.18 message.

**Import from Word**
- Entry points: the **Import from Word** toolbar button (file input accepting `.docx`), and dropping a `.docx` onto the editor. Both call `POST /kb/import/docx` (multipart; the origin check applies). The endpoint **never creates or changes an article**. It stores the pictures and returns JSON `{title, html, notes}`, and the editor inserts the result. If the editor isn't empty, confirm "Replace what's in the editor with this document?" first. Publishing then follows the normal path (sanitise, block ids, version row, search rows).
- Conversion lives in `internal/kb/docx` as a pure function over the uploaded bytes. It returns the title, HTML, images (bytes plus alt text) and counts of skipped items. The handler stores images through the **same image-store function** as uploads (sniffed; JPEG, PNG, GIF and WebP only), rewrites them to `/images/…`, and runs the result through the **same sanitiser** as publishing.
- Accepted input: a zip whose `[Content_Types].xml` declares a WordprocessingML main document (`.docx`; `.docm` is read as plain content, and macros are never run or kept). Find the main part through `_rels/.rels`, not a fixed path. Anything else returns the single plain message in 1.51. That covers old `.doc` files, password-protected documents (which aren't zips), PDFs and damaged zips.
- Limits:
  - Upload ≤ 50 MB, and at most 5,000 zip entries.
  - Total uncompressed size ≤ 300 MB, enforced while reading (`io.LimitReader`) rather than trusted from the zip headers.
  - XML element depth ≤ 256, and each image ≤ 20 MB.
  - Relationship targets resolve only inside the zip. Images with `TargetMode="External"` are **never fetched**; they become placeholders.
- Mapping. Resolve paragraph and character styles through the `styles.xml` `basedOn` chains. Match style **names** (`heading 1`, `Title`), not style IDs, because Word localises IDs. `w:outlineLvl` also marks a heading.
  - `Title`, or a first non-empty paragraph in `heading 1`, becomes the article title and is removed from the body. Otherwise the title is the file name without its extension.
  - `heading 1` → `h2`; `heading 2`–`heading 9` → `h3`. No `strong` inside headings.
  - Runs: bold → `strong`, italic → `em`, honouring `w:val="0"`/`"false"` and style inheritance. Underline, strike, colour, highlight, font and size are dropped. `w:tab` → space; `w:br` → `br`, with page and column breaks ignored.
  - Lists: `w:numPr`, directly or through the paragraph style. A `numbering.xml` level with `numFmt` `bullet` → `ul`; any other format → `ol`. `ilvl` → nesting. Consecutive paragraphs with the same `numId` form one list, and `numId` 0 is not a list.
  - Links: `w:hyperlink` with an external relationship, and `HYPERLINK` simple or complex fields → `a href`. The sanitiser allows only `http`, `https` and `mailto`. Internal anchors keep their text and drop the link.
  - Tables: `w:tbl` → `table`. Rows marked `w:tblHeader` → `th`. `w:gridSpan` → `colspan`; `w:vMerge` restart/continue → `rowspan`. A table nested in a cell is flattened into paragraphs in that cell.
  - Images: `w:drawing` (`wp:inline`/`wp:anchor`) with `a:blip r:embed`, and legacy VML `v:imagedata r:id`, → `img`, with `alt` from `wp:docPr` `descr` (or `title`). Word's SVG icons carry a PNG in `a:blip`; use it. Inside `mc:AlternateContent`, use the one branch that yields an image or text, never both.
  - Text boxes (`w:txbxContent`): output their paragraphs after the paragraph that anchors them.
  - Tracked changes: include `w:ins` and `w:moveTo`; skip `w:del`, `w:delText` and `w:moveFrom`. Descend into `w:sdt`/`w:sdtContent`, `w:smartTag`, `w:customXml` and `w:fldSimple`. Skip field instruction text. Drop empty paragraphs.
  - Unsupported items each become one placeholder, `<div data-missing-kind="chart|diagram|shape|equation|picture|object">`, containing the 1.49 wording. This covers charts (`c:chart`), SmartArt (`dgm:`), drawn shapes with no text, equations (`m:oMath`), embedded objects with no usable image, and pictures in other formats (EMF, WMF, TIFF, BMP) or external links.
  - Left out, and counted in `notes`: comments, footnotes and endnotes, headers and footers.
- The sanitiser allows `colspan` and `rowspan` on `td`/`th`, `alt` on `img`, and the `data-missing-kind` placeholder. The editor and the article page show placeholders as a red dashed box.

**Edit conflicts**
- The editor form carries `version_no`. If the stored `version_no` differs on publish, return the editor with the user's content intact, a plain message, and a "Publish mine anyway" button (sends `override=1`).
- Every publish creates a version row, so nothing is lost.

**Task ordering**
- Moves use `POST /tasks/{id}/move` with `stage` and one of `before_id`, `after_id` or `to_bottom`. The server renumbers `position` for the affected stage in one transaction.
- "Done more than 14 days" means `done_at < now − 14 days` in local time. Such tasks are excluded from the board query.
- Reopening sets the stage to `todo`, puts the card at the bottom, and clears `done_at`.

**Checklist steps** *(v1.2)*
- A job holds at most **50** non-removed steps and a step at most **200** characters. Both limits are enforced in the store, not in a route, so nothing can get round them. Hitting one answers with the step text still in the box and a plain message ("A job can have up to 50 steps" / "A step can be up to 200 characters").
- **Two people at once.** Ticking a step somebody else ticked a moment ago is not an error: it stays ticked, and keeps the *first* person's name and time. Unticking clears both. Renaming, removing, restoring or moving a step somebody else has already removed answers "Somebody else removed that step" and changes nothing. None of these produce a failure page.
- Removing sets `removed_at` and closes the gap in `position`. **Undo** (the restore route) puts the step back at its old position, pushing later steps down if that position has since been taken. A removed step is never deleted from the database and always appears in `steps.csv`.
- Every write to a step bumps the tasks change counter in the same transaction, so other computers pick it up through the existing refresh (see "Refresh without live sockets"). No new refresh machinery.
- The Briefing does not read steps (D-73). A job with unfinished steps briefs exactly as it did at v1.1.0.

**Team view**
- Lanes: first an Unassigned lane (tasks that aren't removed, are in `todo` or `doing`, and have no assignees), then one lane per active person, in alphabetical order. Ideas and Done are not shown.
- Each lane lists `doing` tasks ("Working on now", ordered by `position`), then `todo` tasks ("Up next", ordered by `position`, numbered 1..n).
- Load is the sum of weights S=1, M=2, L=4 over the lane's tasks, drawn as blocks grouped per task. There are no thresholds, colours or limits.
- Reordering within a lane's Up next uses the existing move endpoint. `before_id`/`after_id` is the neighbouring card **in that lane**, so the task lands directly next to that card in the shared To do order and unrelated tasks keep their positions.
- Assigning by drag: `POST /tasks/{id}/assign` with `from_person` (nullable) and `to_person`. It replaces `from_person` with `to_person`, keeps other assignees, records `unassigned`/`assigned` activity, and doesn't change stage or position. Every drag has an equivalent "Assign to…" button.
- Lanes have a minimum width of 232 px. When there are more people than fit, the lanes scroll sideways inside their own container.

**My jobs view** (default for `GET /tasks`; Board at `/tasks/board`, Team at `/tasks/team`)
- The left side is the current person's lane, built by the same code as their Team lane.
- The right side is Up for grabs: unassigned `doing` and `todo` tasks ordered by stage then `position`, followed by unassigned `idea` tasks by `position`.
- Take: `POST /tasks/{id}/take`, with an optional `before_id`/`after_id` from the drop position. In one transaction: if the task already has any assignee, return 409 with the assignees' names and change nothing. Otherwise add the current person. If the stage is `idea`, set it to `todo`, placed at the drop position or at the bottom. Record activity.
- Without a drop position, a `todo`/`doing` task keeps its `position`.
- Give back: the existing assign endpoint with `from_person` = current person and `to_person` = null.

**Refresh without live sockets**
- The Tasks board, My jobs, the Team view and the Briefing poll a small `…/version` endpoint every 60s while the window is visible, and immediately on `focus`/`visibilitychange`. The endpoint returns the max `updated_at` or a change counter.
- If it has changed, re-fetch and re-render, **unless a drag is active or a card or event editor is open**. In that case, show a small "Updated — refresh" notice.

**Recurrence**
- Pure function: `Occurrences(event, exceptions, from, to) []Occurrence`.
- Weekly: every 7 days from `start_date`.
- Monthly: same day of month, clamped to the month's last day.
- Yearly: same month and day, with 29 Feb becoming 28 Feb in non-leap years.
- Multi-day events keep their length (`end_date − start_date`).
- `until_date` is inclusive on the occurrence start.
- Exceptions are keyed by the **original** date. `cancelled` removes the occurrence, and `moved` replaces its dates and times.
- "Change all" edits the series row. Moved exceptions keep their own dates and times, but title and notes always come from the series.
- "Change just this one" only offers date and time changes plus cancel. Any title or notes edit applies to the series. **This keeps the model simple, and satisfies 2.15–2.17.**

**Briefing** (pure function, `Build(today, tasks, occurrences) Briefing`)
- `S` = `today` if today is Saturday, else the next Saturday after today. `F` = `S + 6 days` (the Friday).
- **Must be done:** tasks not done and not removed, with `due_date ≤ F`, sorted by due date then board position. Mark as overdue if `due_date < today`, and mark YOURS if assigned to the current person.
- **This week:** occurrences (after exceptions) overlapping `[today, F]`, sorted by start.
- **Coming up:** occurrences with `start > F` and `start − notice_days ≤ S`, sorted by start, with a label "in N weeks" or "in N days" measured from `S`. Each series only shows its next qualifying occurrence.
- Test mode only: `COMPHQ_TEST_MODE=1` with `COMPHQ_TEST_TODAY=YYYY-MM-DD` overrides `today`. The app logs a loud warning when this is set. It is never set in `deploy/truenas.yaml`, and a test asserts that.

**Request safety**
- Every state-changing request must be a POST whose `Origin` header (or, failing that, `Referer`) host matches `Host`. Otherwise return 403.
- Templates auto-escape. Article HTML is rendered only after sanitising.
- Set `Content-Security-Policy: default-src 'self'; img-src 'self' data: blob:; object-src 'none'; frame-ancestors 'none'` and `Referrer-Policy: same-origin`.
- **No outbound network calls** except the publish-time image copy. No telemetry.

## B5. Storage on the NAS, backups, export, restore

**Data folder:** a TrueNAS dataset (for example `/mnt/<pool>/comphq`, created with the **Apps** dataset preset so the `apps` user, UID/GID 568, can write to it), mounted at `/data`:

```
/data/comphq.db (+ -wal, -shm)
/data/images/ab/abcdef….png
/data/backups/daily/comphq-2026-09-14.db        (keep 30)
/data/backups/pre-update/comphq-v1.2.0-20260914T030000Z.db   (keep 10)
```

Use a **host path** mount, not an ixVolume. This way data survives deleting and reinstalling the app, and it's covered by the dataset's own snapshot task.

**Nightly backup:**
- An in-process scheduler runs at startup if no backup has been made in the last 24h, and daily at 03:00 local time.
- It runs `VACUUM INTO` to a temporary name, then renames the file, prunes old copies, and records the result in `app_meta`.
- Images are content-addressed and never deleted, so the NAS snapshots cover them without copying.
- A banner appears on every page when `now − last successful backup > 48h` (gate 1.35).

**Export everything:**
- `GET /settings/export` streams a zip named `comphq-export-YYYY-MM-DD.zip` containing:
  - `index.html` (contents page).
  - `articles/<Category>/<Title>.html` (self-contained readable page with inline CSS, and images linked relatively to `images/`).
  - `articles/_Archived/…`.
  - `images/…`.
  - `tasks.csv` and `events.csv` (UTF-8 with BOM so Excel reads it correctly; the dates are readable).
  - `comphq.db` (a fresh `VACUUM INTO` copy).
- Filenames are made safe for Windows.

**Restore (owner, via README):**
1. **Preferred:** TrueNAS → Datasets → comphq → Snapshots → roll back to a chosen snapshot, then restart the app.
2. For a single bad edit: use article History → Restore (no NAS access needed).

The daily and pre-update copies exist so a future agent helping the owner has clean files to recover from. The README says so in one sentence.

## B6. Deployment to TrueNAS SCALE 25.04, updates, rollbacks

**`deploy/truenas.yaml`** uses only standard Docker Compose keys, so it survives TrueNAS changes (custom YAML apps have run on Docker since 24.10 and remain in 25.10):
- One service `comphq`. `image: ghcr.io/stas-comp/comphq:<exact version>` (never `latest`).
- `restart: unless-stopped`. `user: "568:568"`.
- `ports: "8080:8080"`. The README explains how to pick another host port if TrueNAS reports a conflict. Never port 80 or 443, which the TrueNAS UI uses.
- `volumes`: host path → `/data`. `environment`: `TZ=<owner's time zone>`.
- `healthcheck`: `["CMD", "/comphq", "healthcheck"]`.
- `read_only: true` root filesystem, `tmpfs: /tmp`, no privileged mode, no host networking.
- The owner only edits the pool path, the time zone and, if needed, the host port. Those lines are marked in the file with plain-English comments.

**App address:** `http://<NAS IP>:8080`. The README tells the owner to confirm the NAS has a fixed IP address (TrueNAS → Network), and how to set one if not.

**Desktop shortcut (gate 1.43):**
- The Setup page generates the shortcut text for both browsers:
  - `"C:\Program Files\Google\Chrome\Application\chrome.exe" --app=http://<host>:<port>/`
  - `"C:\Program Files\BraveSoftware\Brave-Browser\Application\brave.exe" --app=http://<host>:<port>/`
- It uses the address the page was loaded from, and gives steps for "Desktop → right-click → New → Shortcut → paste", plus how to find the browser's real path if Windows can't find it (right-click the existing browser icon → Properties → Target).
- The `--app` flag is used rather than the browsers' "Install page as app" menus, because those menus move between versions (and Brave's has disappeared more than once), and browser installability requires HTTPS.
- Also serve a web app manifest and icons, so the window gets the Comp HQ name and icon.

**Releases:**
- Semantic versions. CI builds, runs every test, and pushes the image only if everything is green.
- A plain-language `CHANGES.md` entry is written per release.
- The build agent gives the owner the new version number, a two-sentence plain summary, and the update steps.
- New `ghcr.io` packages start private. After the first push, ask the owner to set the `comphq` package to **Public** (GitHub → stas-comp → Packages → comphq → Package settings → Change visibility), with click-by-click steps. The agent never changes account settings itself.

**Update (owner):** TrueNAS → Apps → comphq → Edit → change the version number in the image line → Save. TrueNAS pulls the image and restarts the app.

**Migrations:**
- Run automatically at startup, one transaction per migration, **after** a `VACUUM INTO` pre-update backup (only when migrations are pending).
- **Expand-only rule:** a release may add tables, columns and indexes. It may **not** drop, rename or change the meaning of existing ones. Contraction is allowed only in a release at least two versions after the one that stopped using them, and must be recorded in `docs/decisions.md`.
- As a result, the **previous version runs correctly on data migrated by the new version** (gate 1.42). Unknown newer migration rows must not stop an older app from starting.
- For a rare release that truly can't be backwards compatible, set `app_meta.min_app_version`. An older app seeing a higher value shows a plain page: "This data was updated by Comp HQ *x.y.z*. Put the version number back to *x.y.z* in TrueNAS, or roll back to the snapshot taken before the update (README → Undo an update)." Avoid this.

**Undo an update (owner, via README):** change the version number back. If the README's release note says the release was not backwards compatible, roll back the dataset snapshot as well.

## B7. How each gate is verified

**Test layers** (all run in CI on every push; the local run uses the same commands):
1. **Go unit tests:** recurrence, briefing `Build`, search-query builder, sanitiser and block ids, the Word import converter, ordering, backup pruning, migration runner, filename sanitising.
2. **Go HTTP integration tests:** handlers against a temporary data folder. These cover the origin check, image upload limits and types, SSRF refusals (fetch to a local test server on a private address is refused, while a test override allows the stub "external" server), and conflict detection.
3. **Playwright end-to-end tests (Chromium):** against the real binary with a temporary `/data`, seeded fixtures, and `COMPHQ_TEST_TODAY`. Tests wait on `data-ready`.
4. **Container tests (Linux CI):** build the image, then run `deploy/truenas.yaml` with Docker Compose (placeholders substituted), then run an end-to-end smoke test against the container. This stage includes:
   - Restart test.
   - Remove-and-recreate test.
   - Offline test (Compose network with `internal: true`, and the test runner attached to it).
   - Upgrade and rollback test against the **previous released image**.
5. **Speed tests:** seed the test library (500 articles of ~800 words with images, 2,000 tasks, 300 events of which 100 repeat). Measure navigation to `data-ready` (limit 1,500 ms, median of 5), time from last keystroke to the results panel rendering (limit 1,000 ms), and search endpoint p95 over 50 queries (limit 200 ms).
6. **Accessibility:** `@axe-core/playwright` on every page type. Fail on `serious` or `critical`.
7. **Screenshots for owner checks:** at each phase end, Playwright captures every page type at 1366 × 768 into a CI artifact. The agent passes these to the owner with the gate checklist.

| Gates | Verified by |
|---|---|
| 1.01–1.07, 1.11 | E2E: fresh browser context (no cookie), picker, cookie persistence across new pages in the context, rename and remove reflected in history and pickers, frame elements present on every route, 404 route. |
| 1.08 | E2E: each placeholder route returns 200 with the "Coming soon" heading (removed in later phases). |
| 1.09, 2.23, 3.13 (size) | E2E: every page type at 1024 × 700 and 1920 × 1080 has `document.documentElement.scrollWidth <= clientWidth`. |
| 1.10, 2.23, 3.13 (a11y) | Axe on every page type. Keyboard-only E2E for picker, publish article, move task by buttons, add event. |
| 1.12–1.14 | E2E plus integration. |
| 1.15 | E2E: apply each toolbar action, publish, and assert the matching elements exist in the published article. |
| 1.16–1.17 | E2E: write a PNG to the clipboard (or synthesise a paste event with a file) and use `setInputFiles` / drop events. Assert the `img src` starts with `/images/` and the file exists in `/data/images`. Oversize and wrong-type fixtures show a message. |
| 1.18 | E2E: paste fixture HTML captured from a sample web page and from Word (`e2e/fixtures/paste/`; the Word fixture includes `file:///…/clip_image001.png` pictures). Assert the allowed elements are kept, `style`, `font` and `class` are stripped, and Word's `file:` pictures become placeholders with the Import from Word message. |
| 1.19 | Integration plus E2E with a local stub "external" image server. Success rewrites to `/images/`. With the stub down, the failure is listed, the placeholder is present and the text is saved. |
| 1.20–1.21 | E2E: `beforeunload` dialog. Two browser contexts edit the same article, and the second publish shows the conflict message, keeps the text, and "Publish mine anyway" works. Both versions are in history. |
| 1.22–1.25 | E2E plus integration. 1.25 also asserts that no route issues DELETE-style removal of articles (route table test). |
| 1.26–1.32 | E2E with fixture articles: printer article with "toner", title versus body ranking, stemming, prefix, immediate indexing after edit, archived exclusion, and symbol fuzzing (`" * ( - : NEAR AND OR ^`) with no error. **1.28 test:** type "toner" in the search box; the first result contains `<mark>toner</mark>`; click it; the URL has `#b-N`; that block is in the viewport, has the highlight class, and contains `mark` with the text "toner". |
| 1.33, 2.21, 3.13 (speed) | Speed tests (layer 5). |
| 1.34, 2.22 | E2E: download the export, unzip, open `index.html` via `file://` in an **offline** browser context, follow every article link, and assert headings and images load with no failed requests. Parse the CSVs and assert the headers and a known row. |
| 1.35 | Integration: fake clock and scheduler run a backup, create the file, prune to 30, update status. E2E: set `last_backup_at` to 3 days ago and the banner shows on every page. |
| 1.36–1.37 | E2E. |
| 1.38–1.42 | Container tests (layer 4). 1.38: the Compose file validates (`docker compose config`) and the container becomes healthy with only the documented edits. 1.39: `docker compose restart` and a full `down`/`up`, healthy within 60s, data checksums match. 1.40: `down`, remove the container and image, `up` with the same folder, data intact. 1.41: the full E2E smoke suite passes on an internal-only network. 1.42: seed with the previous release, upgrade, smoke-test, downgrade to the previous release, smoke-test. |
| 1.43 | Integration: Setup page text matches the expected commands for the request host and port. E2E: launch Chromium with `--app=<url>`, and the app page loads as the first window (Playwright `launchPersistentContext` with `args`). The owner confirms the real window in O1.1. |
| 1.44 | Test: `README.md` exists and contains each required section heading. |
| 1.45–1.52 | **Unit:** `internal/kb/docx` with small hand-built `.docx` fixtures, one per B4 mapping rule. Cover a localised style ID, `basedOn` inheritance, numbering levels, `mc:AlternateContent`, runs split mid-word, tracked changes, merged cells, text boxes, each unsupported kind, an external image, and the zip-size and depth limits. **E2E:** import `e2e/fixtures/docx/sample.docx` by button and by drop. This is a realistic document made by a committed generator script (for example LibreOffice headless in CI, or a Go writer), with the output committed. Assert headings, nested lists, table spans, `img src` under `/images/` with the files on disk, alt text, the title rule, placeholders, the notes message, the replace confirmation, and that publishing adds a History version. If `samples/word/*.docx` files exist, import each one and assert no error, at least one block, and only local images. **1.51:** fixtures for an OLE/CFB file (standing in for `.doc` and password-protected files), a PDF, a truncated zip and an oversize file each show the exact message. **1.52:** a timed import of a generated 20-page, 10-picture fixture. |
| 2.01–2.11 | E2E, using two browser contexts for 2.04 and 2.10 (a change in A appears in B within 65s of polling, or instantly after a focus event; an open editor in B is not replaced). Fake clock for 2.08. |
| 2.24–2.31 | E2E with fixtures covering unassigned, one-person, shared and mixed-size tasks. Checks: lane contents and grouping; block counts (one L = four S); assigning by drag and by button; reordering inside a lane, then asserting that on the board only the moved task's relative position changed; size default and activity entry. Speed and axe as layers 5–6. Sideways-scroll check on the lane container at 1024 × 700. |
| 2.32–2.38 | E2E: fresh name on a computer lands on My jobs; fixtures with own, shared, other-person, unassigned `todo`/`doing` and unassigned ideas assert exactly what shows on each side. Take by drag (drop position respected) and by button (position kept); taking an idea moves it to To do. Two browser contexts take the same job: the first succeeds, the second gets the "has just taken" message and no change. Refresh within 65s in the other context. Give back returns the job to Up for grabs. Speed and axe as layers 5–6. |
| 2.12–2.20 | Unit tests (recurrence table: weekly, month-end clamp for 31 Jan → 28 Feb 2027, leap day, `until`, multi-day, moved, cancelled) plus E2E for the UI flows, including the exact 2.15 dates. |
| 3.01–3.12 | Unit tests on `Build` with the exact example dates from Part A, plus E2E with `COMPHQ_TEST_TODAY` set to 2026-09-19 (Saturday), 2026-09-16 (Wednesday) and 2026-09-12 for 3.07, and two contexts for 3.12. |
| 4.01–4.21, 4.30–4.47 | Five automatic layers, per B9.9. The first opens `docs/design/mockup.html` beside the app and asserts the app's computed styles match the mockup's, element for element, so the mockup is the contract rather than its description. Then: a token test that reads `theme.css` against the B9.2 table and rejects any colour or font literal in a section stylesheet; Playwright assertions on **computed** styles for real rendered elements (resolved font families, the accent fill, the three distinct day shades for 4.34, the tint on To do, today's ink chip); and drift rules — no unstyled `button`/`select`/`input`, no icon-only button without both `title` and `aria-label`, no white text on the accent, no text in `--color-signal-orange`. Contrast comes from the existing axe layer. Screenshots in `reports/screens/` refreshed for O4.1. |
| 4.22–4.29 | E2E on the task window: open from **+ Add task** and from a card; create with a description and a size and assert both stored; read-only then Edit then save, asserting the board updates and the right history row appears; close by Escape, by the close icon and by the backdrop, asserting focus returns to the card; Tab cycles inside the window only. **With JavaScript disabled**, assert `+ Add task` and a card title still reach a working page and a task can still be created and edited there (gate 4.28). Assert `/tasks/{id}` still answers directly and a Calendar link still lands on it (4.27). Axe runs with the window open. |
| 4.48–4.51 | Written failing-first (B9.8): a unit test creating an `idea` with a person on it, asserting it appears in that person's My jobs group and Team column and **not** in Up for grabs, and that their workload blocks are unchanged by it. Plus E2E: create a task choosing people in the same step, then assert it on the Board, on My jobs and on Team. |
| 4.52–4.54 | The Phase 1–3 suites re-run unchanged, at the same thresholds. A weakened, skipped or deleted Phase 1–3 test fails the phase (B8 rule 5). |

Also asserted in CI: `deploy/truenas.yaml` doesn't set `COMPHQ_TEST_MODE`; image tags are pinned; base images are pinned by digest; no `http(s)://` asset URLs appear in templates or CSS other than in the Setup page text.

## B8. Rules for the build agent

1. **Meet the gates.** Beyond them, solve technical problems yourself. **Don't ask the owner technical questions.**
2. **Stop and ask the owner only for:** a Part A gate that seems impossible or contradictory; any change that would add a paid service, an internet dependency at runtime, or a login; or a 👤 Owner check. Ask in plain language, with your recommended answer.
3. **Prefer the simplest approach that passes the gates.** Don't add features that aren't in this spec. If a detail isn't specified, pick the simpler option and note it in `docs/decisions.md`.
4. **Keep sections self-contained** (B2 boundaries), so changing one later can't break another.
5. **Never report a gate as done unless its automated check passes in CI.** Never skip, weaken or delete a test, or raise a threshold, to get to green. If a gate's test is wrong, fix the test *and* say so plainly in the phase report.
6. Build in phase order (1 → 2 → 3). At the end of each phase, give the owner a plain-language report: the gate checklist with results, screenshots, the version number to install, and the exact 👤 Owner checks with click-by-click steps.
7. **Keep the owner's `README.md` current** in plain language: opening Comp HQ, setting up a computer, installing on TrueNAS, updating, undoing an update, backing up (snapshots and export), restoring, and "it won't open — what now?" (check the app in TrueNAS → Apps, restart it, check the version, roll back). Include click-by-click steps for every owner hands-on moment in A13.
8. Pin every version: Go toolchain, Go modules (`go.sum`), vendored JS with recorded version and checksum, `package-lock.json`, base-image digests, and GitHub Actions pinned to commit SHAs.
9. **No runtime calls to the internet** apart from the publish-time image copy. No CDNs, no analytics, no telemetry.
10. **Follow the expand-only migration rule (B6)** and keep the previous-version rollback test green.
11. Never enter passwords or tokens on the owner's behalf. When GitHub sign-in is needed, ask the owner to complete it, with steps.

## B9. The v1.1 design system

### B9.1 Where the design lives

`docs/design/mockup.html` is the agreed design, saved into this repository on 19 Sep 2026 from the
owner's design conversation. It is a reference, not code to copy: it uses made-up content, loads its
fonts from the internet, and carries a fake browser window around the app. Take from it the colours,
type, spacing, parts and per-screen layout — nothing else.

`docs/design/mockup.html` is **reference material, not shipped code**. It is never served by the app,
never embedded in the binary, and never linted or tested. Its Google Fonts link is expected and must
not be "fixed"; if the no-external-asset check in B7 ever grows to scan the whole repository, exclude
`docs/` rather than editing the mockup. Leave the file exactly as saved, so it stays a faithful record
of what the owner approved.

**Read the mockup; don't work from this spec's prose alone.** Every Phase 4 task in `PLAN.md` names
the part of `docs/design/mockup.html` it implements — a screen's `<section data-screen="...">` and the
CSS rules it uses. Open the file and read that markup and those rules before writing anything. The
gates in A11 say what must be **true**; the mockup says what it must **look like**. Where a spacing,
weight, size or radius isn't named in a gate, the mockup's value is the answer — don't invent one, and
don't round it off.

The mockup does not show the task window (4.22–4.29) — that was agreed after it was drawn. Build the
window from the mockup's own parts, so it looks as though it had always been there.

`web/static/theme/theme.css` stays the single place where colours, fonts, radii and spacing are
defined (A2 priority 3, B2). Section stylesheets may only do layout; a section stylesheet that
introduces its own colour or font value is a defect.

### B9.2 Tokens

Replace the current palette with the mockup's. Names stay as they are where they already match.

| Token | Value | Used for |
|---|---|---|
| `--color-ink-navy` | `#141B2D` | sidebar, name picker, event chips, table headers |
| `--color-ink-navy-light` | `#1F2842` | sidebar hover |
| `--color-ink-navy-3` | `#2C3656` | continuing multi-day event chips, default avatar |
| `--color-signal-orange` | `#FF6B1A` | accent **fills** only |
| `--color-signal-orange-deep` | `#A94000` | accent **text** on paper; priority numbers |
| `--color-accent-soft` | `#FFE2CF` | focus rings, outlines of tinted areas |
| `--color-accent-wash` | `#FFF3EA` | tinted columns and panels (To do, My jobs, Saturday) |
| `--color-paper` | `#F5F3EE` | page background |
| `--color-paper-muted` | `#ECE9E1` | board columns, team lanes |
| `--color-card` | `#FFFFFF` | cards, article surfaces, the task window |
| `--color-ink-text` | `#1A1F2B` | body text |
| `--color-text-muted` | `#5A6072` | secondary text |
| `--color-border` | `#DEDAD0` | hairlines |
| `--color-border-strong` | `#CBC6B9` | control outlines, dashed holding areas |
| `--color-danger` | `#C62D2D` | overdue |
| `--color-danger-soft` | `#FBE3E1` | overdue stamp field |
| `--font-display` | `"Big Shoulders Display"` | headings, stamps, column and panel titles, size chips |
| `--font-body` | `"Atkinson Hyperlegible Next"` | all body and control text (unchanged) |
| `--font-mono` | `"IBM Plex Mono"` | dates, counts, version |

**The contrast rule (supersedes D-09).** The old palette dulled the orange to `#c2410c` so that white
text on it would pass 4.5:1. The mockup solves the same problem without dulling anything: the bright
accent is only ever a **fill**, and the text on it is ink navy. Where orange is the text itself, it is
the deep accent on paper. Measured:

| Pair | Ratio | |
|---|---|---|
| ink navy on accent `#141B2D` / `#FF6B1A` | 6.02:1 | pass |
| **white on accent** `#FFFFFF` / `#FF6B1A` | **2.85:1** | **fails — never do this** |
| deep accent on paper `#A94000` / `#F5F3EE` | 5.54:1 | pass |
| deep accent on card `#A94000` / `#FFFFFF` | 6.14:1 | pass |
| body on paper `#1A1F2B` / `#F5F3EE` | 14.86:1 | pass |
| muted on paper `#5A6072` / `#F5F3EE` | 5.65:1 | pass |
| muted on board column `#5A6072` / `#ECE9E1` | 5.17:1 | pass |
| overdue stamp `#C62D2D` / `#FBE3E1` | 4.50:1 | pass, but exactly on the line |
| sidebar text on navy `#C9CEDC` / `#141B2D` | 10.90:1 | pass |

So:

- **Never** white text on the accent. Ink navy on accent, always.
- **Never** `--color-signal-orange` as a text colour. Use `--color-signal-orange-deep`.
- The overdue stamp sits exactly on 4.50:1. Don't lighten either of its two colours; if one has to
  change, darken the text rather than the field, and re-measure.

Record this in `docs/decisions.md` as a new decision superseding D-09, and edit D-09 to say it has
been superseded rather than deleting it.

### B9.3 Fonts

Add **Big Shoulders Display** and **IBM Plex Mono**, both under the SIL Open Font Licence, so they may
be stored inside the app. Keep **Atkinson Hyperlegible Next** for body text. **Retire Archivo** and
delete its file — nothing may reference it once Phase 4 is done.

- Self-hosted `woff2` in `web/static/theme/fonts/`, subset to Latin, variable weight where the family
  offers it, served with a long cache lifetime and `font-display: swap`, exactly as the existing two are.
- Commit the licence text for each family alongside the files.
- The existing CI assertion that no `http(s)://` asset URL appears in templates or CSS (B7) already
  proves nothing is fetched from the internet. Do not weaken it.

### B9.4 Icons

One small line-art set stored in `web/static/theme/icons/`, drawn on a 24 × 24 grid, stroke width 2,
round caps and joins, `fill: none`, `stroke: currentColor`, so an icon takes the colour of its button.
Needed for v1.1: `arrow-up`, `arrow-down`, `trash`, `give-back`, `plus`, `search`, `check`, `chevron`,
`close`, plus the five section icons already present, redrawn to match if they don't.

Every icon-only button (gate 4.11) needs both a `title` for the hover tooltip and an `aria-label`, and
they must say the same thing, naming the task: "Move *Print exam papers* up".

### B9.5 Shared parts

These live in `theme.css` and are used by every section rather than being re-invented per screen. The
mockup's own class names are a reasonable starting point:

`.btn` / `.btn.primary` · `.mini` (small card buttons) · `.seg` (segmented switch) · `.stamp` (+
`.today` `.overdue` `.yours`) · `.size` (+ `.s` `.m` `.l`) · `.av` / `.people` (avatars and stacks) ·
`.load` (workload blocks) · `.card` · `.panel-head` · `.date` (mono) · `.icon-btn`.

Styled form controls (gate 4.05) belong here too, and are shared by the board and the task window. The
date field must show day-first; a native `<input type=date>` follows the computer's locale and
currently shows `mm/dd/yyyy`, which is wrong for this office — fix it however is simplest and most
stable, and note what you chose in `docs/decisions.md`. *(Done: D-61, a typed box. v1.3 adds a day-first calendar over it, drawn by Comp HQ, not the browser. See B12.2.)*

### B9.6 Assigning from a card (gate 4.20)

No new storage and no new endpoint. The people circles open a small menu listing current people; each
choice posts to the existing assign endpoint (`from_person` / `to_person`, `internal/tasks/assign.go`),
which already keeps every other assignee untouched and already records Activity. Give back on a card
(gate 4.19) is the same endpoint with `from_person` = self and no `to_person`, exactly as My jobs
already does it. Remove on a card (gate 4.18) posts to the existing `/tasks/{id}/remove`.

### B9.7 The task window (gates 4.22–4.29)

**No new endpoints and no new storage.** Everything the window needs already exists:

- `handleCreateTask` (`internal/tasks/handlers.go`) already accepts `title`, `notes`, `size`, `stage`,
  `due_date` and repeated `person_id`. The board's form simply never offered description or size, which
  is why gate 4.22 can add both without touching the server.
- `POST /tasks/{id}` (`internal/tasks/details.go`, P2-04) already edits the same set in one transaction
  and already records the right activity row per kind of change.
- `GET /tasks/{id}` already renders the full page the window is a nicer face for, and `ListActivity`
  already produces the history for gate 4.24.

So the window is one shared piece of the interface over endpoints that are done. Build it as **one**
window with three states — new, reading, editing — not three separate things.

**It is an enhancement, never a requirement (gate 4.28).** Server-rendered pages stay the ground truth:
`+ Add task` is a link to a new-task page and a card's title is a link to `/tasks/{id}`. The window
intercepts those links when it can, and when it can't, the links still work on their own. This is A2
priority 2 (stability) beating priority 4 (design), and it is not optional.

**It must behave like a dialog, not a floating box.** Use the platform's own `<dialog>` element rather
than building one: it gives Escape, the keyboard trap of gate 4.26, the backdrop and the return of
focus without hand-written code, and it is the simpler and more stable choice (A2 priority 2). The
accessibility layer of B7 covers the rest.

**Don't remove the task page.** v1.2 keeps it too — Calendar links point at it, and a refreshed browser
window must still land somewhere real (gate 4.27).

### B9.8 The assigned-ideas fix (gates 4.48–4.51)

The cause is known. `ListForTeamView` (`internal/tasks/lanes.go`) selects only `stage IN ('doing',
'todo')`, while `Create` defaults a new task to `idea` (`internal/tasks/store.go`). An idea with people
on it therefore appears on the Board and nowhere else: not in My jobs, not in Team, and not in Up for
grabs either, because `ListUpForGrabs` excludes anything somebody is on.

Direction:

- The team/my-jobs query includes `idea` as well, and the lane builder puts those tasks in their own
  group rather than mixing them into Working on now or Up next.
- Workload blocks count `todo` and `doing` only. An idea must not change anybody's workload.
- `ListUpForGrabs` is unchanged: unassigned ideas still surface there.
- Guard it with a test that fails against today's code before the fix: create a task in `idea` with a
  person on it, and assert it appears in that person's My jobs and Team column. Write that test first
  and watch it fail, so the fix is proven rather than assumed.

### B9.9 How Phase 4 gates are checked

"Looks like the mockup" is made testable in five layers, all automatic:

1. **Against the mockup itself.** The test suite opens `docs/design/mockup.html` in the same browser
   as the app and reads its **computed** styles, then asserts the app's matching element agrees:
   font family, weight, size, letter-spacing and text-transform for each kind of heading, stamp, size
   chip and date; background and border for a card, a board column, the To do column, a team lane, the
   Unassigned lane, a Saturday cell and an out-of-month cell; and the fill and text colour of a primary
   button. This is what makes the mockup the contract rather than my description of it — if the two
   disagree, the mockup wins (A10) and the app is wrong.

   This works with no internet. `getComputedStyle` reports the *declared* font stack whether or not the
   font file loaded, so the mockup's Google Fonts link never needs to resolve in CI. Compare the first
   family named, not a resolved file. Don't compare whole screenshots — the two hold different content
   and that test would fail forever for no useful reason.

2. **Tokens.** A test reads `theme.css` and asserts the B9.2 table exactly — every name present, every
   value as written, and no colour or font literal anywhere in a section stylesheet.
3. **What the browser actually renders.** Playwright asserts computed styles on real elements: the
   resolved font family of a heading, of a date, of body text; the background of a primary button; the
   backgrounds of an ordinary day, a Saturday and an out-of-month day being three measurably different
   values (gate 4.34); the tint on the To do column; the ink chip on today's date. These are the gates
   an agent can prove without a human eye.
4. **Rules that catch drift.** No `<button>`, `<select>` or `<input>` renders without a theme class
   (gate 4.05); no icon-only button lacks both `title` and `aria-label` (4.11); no element uses white
   text on the accent, and no text uses `--color-signal-orange` (B9.2).
5. **Nothing broke.** The existing accessibility, window-size and speed suites run unchanged, and
   every Phase 1–3 gate still passes (gate 4.54). Contrast is checked by the accessibility layer, not
   by eye.

Then refresh every screenshot in `reports/screens/` so the owner can hold them beside the mockup for
O4.1, adding one of the task window open.

### B9.10 Not in v1.1

**Checklists** (the remainder of the owner's request 4) are **v1.2**. They need storage of their own,
so they are deliberately not started in Phase 4. The task window is built so that a checklist can
later be added inside it without rearranging anything. **They are now specified, in B10** — as
*steps inside a job*, which is what the app calls them on screen.

Settings does not appear in the mockup. It inherits the shared parts of B9.5 and the new tokens, but
its layout is unchanged.

## B10. Steps inside a job (v1.2, gates 5.01–5.17)

### B10.1 What it is, and what it deliberately isn't

A job carries **one flat list of steps** — not Trello's several named lists (D-70). One list is
quicker to read and much less to learn, and a job that genuinely needs "Before" and "On the day" is
usually two jobs. A step is a tick-box with words on it: no person of its own, no date of its own,
no notes (A12). There are no reusable templates in v1.2 (D-72).

On screen the word is **Steps**, never "checklist" — it is what the owner's team would say, and it
matches the plain vocabulary of the rest of the app (A2 priority 1).

### B10.2 Storage and endpoints

The `task_checklist_items` table of B3 and the behaviours of B4 "Checklist steps" are the whole
data story. No other table changes; `task_activity` gains four action names.

Six routes, all in the existing tasks section (B2 — sections stay independent), all `POST`:

| Route | Does |
|---|---|
| `POST /tasks/{id}/steps` | add a step at the end |
| `POST /tasks/{id}/steps/{stepID}/tick` | tick or untick (the form carries which) |
| `POST /tasks/{id}/steps/{stepID}` | rename |
| `POST /tasks/{id}/steps/{stepID}/remove` | remove |
| `POST /tasks/{id}/steps/{stepID}/restore` | the Undo |
| `POST /tasks/{id}/steps/{stepID}/move` | up or down |

Each answers a **redirect back to the task page** when asked for a page, and a **fragment** when
asked for one, exactly as `internal/tasks/details_handlers.go` already does for the task window.
Each bumps the tasks change counter inside its own transaction (gate 5.12).

### B10.3 One partial, two homes

The list is **one template partial**, included by both the task window and the task page. One
partial, two homes — that is what makes gate 5.09 a fact rather than an aspiration, and it means
the no-JavaScript path is the same code rather than a second implementation of it.

**Build the forms before the script.** The routes and the plain-form version come first and are
tested with JavaScript off; the script is then layered over them, intercepting where it can. This is
the order the task window was built in (B9.7), and it is what makes gate 5.08 true by construction
rather than by retrofit. A2 priority 2 (stability) over priority 4 (design), as everywhere.

### B10.4 On screen

Under the description, headed **Steps**, with the count and a small bar: **3 of 7 done**, or
**All 7 done** when every one is ticked. Each step is its box, its words, and — once ticked — who
ticked it and when, in the short form used everywhere ("Jane, Sat 19 Sep"). Each carries the
gate-4.11 icon buttons for move up, move down and remove, each with its tooltip and spoken name.

**Add a step** sits at the foot: type, press Enter or **Add**, and the box clears and stays ready,
so a list can be typed straight through without reaching for the mouse (gate 5.02). Renaming is in
place. Removing shows **Undo** beside the heading until the next action or about ten seconds.
Reordering uses the icons and the already-vendored drag library.

There is **no mockup drawing for this**, as there was none for the task window. Build it from the
mockup's existing parts (`.card`, `.btn`, `.mini`, the icon buttons) so it looks as though it had
always been there. Owner check O5.1 is what stands in for a drawing.

### B10.5 Everywhere else

- **On a card** (Board, My jobs, Team): a small **3/7** when the job has steps, nothing at all when
  it hasn't, a tick when all are done (gate 5.10). It is a count, not a bar — a bar at card size is
  decoration that can't be read.
- **In History:** added, renamed, removed, restored. Not ticks (D-71). Prove it with a test that
  ticks ten steps and asserts History is unchanged.
- **In the export:** `steps.csv` beside `tasks.csv` and `events.csv`, same plain column names, same
  readable dates, removed steps included and marked (gate 5.14, B5).
- **In the Briefing:** nothing (D-73).

## B11. Looking like an app (v1.2, gates 5.20–5.26)

### B11.1 What is already true, and what is missing

Comp HQ has opened in its own window since v0.1.0: the Setup page hands out a `--app=` command, and
a browser started that way shows no address bar, no tabs and no bookmarks bar. That half works.

What is missing is the **icon**. No page in the app links one, and the `icon-192.png` and
`icon-512.png` that the manifest names are plain orange squares left over from the first build. A
browser and Windows therefore have nothing to show, which is why a shortcut comes out blank. This is
the whole of the problem, and fixing it is most of B11.

### B11.2 The icon

One source of truth: the mark is drawn as **SVG**, and every raster size is rendered from it by a
small tool under `tools/` at build time. Hand-exported PNGs drift out of step with each other; a
renderer cannot.

The mark is the stacked **COMP / HQ** wordmark of gate 4.13 — ink navy field, accent orange HQ. At
16 and 32 pixels the two-line wordmark is unreadable, so those sizes drop to the **HQ** alone. The
maskable 512 keeps the mark inside the safe circle, so Windows and Android can crop it without
cutting letters.

Sizes: a multi-size `.ico`, and PNGs at 16, 32, 48, 64, 128, 192, 256, 512, plus the maskable 512.
Every one lives inside the app and is served from it; **no icon is ever fetched from the internet**,
the same rule as the fonts and the section icons (gates 4.01 and 4.12).

`layout.html` gains the links it has never had, and the app answers `/favicon.ico` directly, because
some browsers ask for that address and nothing else.

### B11.3 The manifest

Name **Comp HQ**, start address `/` (which is the Briefing, A8 — an installed app should open where
the app opens, not on a settings page), `display: standalone`, the ink navy as `theme_color`, the
paper as `background_color`, and the full icon list of B11.2 including the maskable entry.

### B11.4 What the Setup page says

Three routes, in this order, each of which must work without the one before it having worked:

1. **Edge or Chrome, from the browser's own menu** — "install this site as an app". Both do this for
   a plain `http://` address, both create a Start-menu entry and a desktop icon, and both take the
   icon from the page, which is why B11.2 has to land first. This is the route the page leads with:
   it is one menu item, with nothing to paste.
2. **The command shortcut** — the `--app=` command of v0.1.0, its Copy button, and New → Shortcut.
   Kept for Brave, and as the fallback that is certain to give a window with no address bar.
3. **If Windows still shows a blank icon** — a **Download icon** button serving `comphq.ico` as
   `Comp HQ.ico`, then right-click the shortcut → Properties → Change Icon → Browse.

**The exact menu wording in routes 1 and 2 is not to be taken from this spec.** Browser menus move.
It is checked on the browsers actually installed on the office computers, at the end of the phase,
and the page and README are corrected to match what is on the screen — with the browser versions and
the date written into `PROGRESS.md` so the next person knows how stale the wording is.

### B11.5 The padlock, and why it stays out

Browsers keep their full app treatment — installing over a service worker, an offline screen of the
app's own — for `https://` addresses. Comp HQ is reached at `http://<NAS IP>:8080`. A12 ruled the
padlock out deliberately: on a private network it means a certificate installed and trusted on every
office computer, which is fragile and easy to break, for three computers and no outside access.

What that costs, stated plainly so nobody re-opens it without new information: no offline screen of
our own (the browser's error page shows instead when the NAS is off), and the possibility that a
future browser version shows a thin strip carrying the address in an installed window. What it does
**not** cost: the app window, the icon, the taskbar entry, or anything the team does day to day.

Revisit only if that strip actually appears, and then as its own small piece of work.

## B12. Everyday fixes (v1.3, gates 6.01–6.41)

Phase 6 changes no data and adds no section. It touches the shared frame, the shared date field, the
Calendar's month grid, and search. `PLAN-v1.3.md` §4 names the files.

### B12.1 The frame stays put (gates 6.01–6.05)

Up to v1.2, `.frame` is a flex row of `.sidebar` and `.main-column`, and the whole page scrolls, so a
long page carries the sidebar's foot (Settings, the version) off the bottom of the window. The fix
is CSS in `frame.css` only:

- `.sidebar`: `position: sticky; top: 0; height: 100vh; overflow-y: auto`. It's exactly one window
  tall and scrolls on its own when the window is shorter than its contents (6.04).
- `.topbar`: `position: sticky; top: 0`, keeping its paper background and bottom rule. Its z-index
  sits above page content and below every overlay: the task window, the live search panel, a card's
  name menu, drag ghosts, the steps Undo and the date calendar.
- The bar's height is one custom property. Anything the app scrolls to (article blocks for gate
  1.28, fragment targets) uses it as `scroll-margin-top`, so it lands below the bar.
- The backup banner keeps its place above the top bar and scrolls away (D-80).

### B12.2 The date calendar (gates 6.10–6.19)

The typed box of D-61 stays the real form field. One shared script (`web/static/app/datepicker.js`)
adds a calendar to every `input.field-date`, delegated from the document so boxes inside the task
window are covered too. No library.

- **Opens** on click or focus, below the box, or above it when there's no room below. It never
  takes focus while the person is typing. Down arrow moves focus into the grid.
- **Shows** one month, **Sunday first** (D-83): Saturday's column in the Calendar's Saturday wash,
  today marked as on the Calendar, the box's current date marked as selected. It has previous and next
  month controls, and **Today** and **This Saturday** buttons (This Saturday is B4's Briefing
  Saturday: today if it's Saturday, otherwise the coming one).
- **Follows typing:** whenever the box parses (by B4's rules, including a date without its year),
  the grid moves to that month and marks the day.
- **Pairs:** a box with `data-date-after="<id>"` opens, when empty, on the month of the box it names
  (event end and "until" follow the start date).
- **Leaving the box** rewrites a parseable value to the full `dd/mm/yyyy`, so the chosen year is
  visible before saving. An unparseable value is left exactly as typed for the server to refuse.
- **Keyboard and screen readers:** a dialog-like popover holding a grid. Arrows move by a day or a
  week, Page Up and Page Down by a month, Home and End to the week's ends, Enter picks, and Escape
  closes and returns focus to the box. The month heading is a live region, each day's accessible
  name is its full date ("Friday 25 September 2026"), and the picked day carries `aria-selected`.
- **The calendar mark** inside the box's right-hand end is a background image from the icon set
  (`calendar`), decoration only. Gate 4.11's four icon-only buttons stay four (D-81).
- **Without JavaScript** nothing is added, and the box is v1.2's box (6.17).

### B12.3 Sunday first (gate 6.20)

The month grid starts on the Sunday on or before the 1st and ends on the Saturday on or after the
last day. Headings read Sun … Sat. Saturday is marked per day, as now, so its shading follows it to
the last column. The date calendar uses the same rule. The Briefing and every B4 Briefing rule are
unchanged, and a test pins a Briefing to its v1.2 output.

### B12.4 A blank day adds an event (gates 6.21–6.23)

A click on a month-grid day cell whose target isn't inside a link follows that cell's day-number
link to `/calendar/new?date=<day>`, which already fills in the date. The cell shows a pointer and a
small **+** on hover. The day-number link's accessible name becomes "Add an event on *Weekday D
Month*". Without JavaScript the day number still does the job.

### B12.5 Search: the half-typed word (gates 6.30–6.35)

**The fault.** `kb_search` is `porter unicode61`. FTS5 stems a prefix query as it stems any term,
and a stemmed half-word is often not the start of the stemmed whole word: `pay` → `pai`, which is not
a prefix of `payment`. The same happens for `happin`, `busin`, `generat` and `voluntee` (probe,
2026-09-22). Finished words are fine.

**The fix** (D-85):

- `migrations/kb/0005_search_words.sql`: `kb_search_words`, FTS5 over `title` and `body` with
  `tokenize = 'unicode61'` (no stemmer), plus `kb_search_words_vocab`, an `fts5vocab` table of type
  `row` over it. Add-only, so rollback-safe.
- Every write to `kb_search` (publish, edit, archive, restore) makes the same write to
  `kb_search_words` in the same transaction.
- **At every start-up**, after migrations, `kb_search_words` is emptied and refilled from
  `kb_search`. v1.2.0 never writes it, so this is what keeps it right after a rollback and an
  upgrade again (6.34).
- The last word `w` of the query is expanded: `SELECT term FROM kb_search_words_vocab WHERE term >= w
  AND term < w || char(0x10FFFF) ORDER BY cnt DESC LIMIT 30`. The last MATCH term becomes
  `("w"* OR "t1" OR "t2" …)`, each whole word quoted exactly as B4 quotes words. These go through
  `kb_search`'s stemmer as any other word does, so ranking, `snippet()` and `highlight()` work
  unchanged, and the found word is highlighted (6.32).
- The expansion only adds whole words from the database's own vocabulary. Raw input still never
  reaches MATCH. The 2-character minimum of gate 1.26 is unchanged.

Another approach is allowed if it passes 6.30–6.35 and is recorded in `docs/decisions.md`. The
rebuild at start-up (or an equivalent guarantee for 6.34) is not optional.

### B12.6 How Phase 6 gates are checked

As B7, per gate: 6.01–6.05 by E2E at 1024 × 700 and 1920 × 1080 with the test library; 6.10–6.17 by
E2E on every date box, keyboard-only, and axe with the calendar open; 6.18–6.19 by a table of
`format` unit tests and a no-JavaScript E2E; 6.20 by grid unit tests and the corrected Calendar
E2E; 6.21–6.23 by E2E and axe; 6.30–6.33 by `search` unit and integration tests and E2E; 6.34 by the
container test's rollback step; 6.35 by the speed suite; 6.40–6.41 by the full suite.
