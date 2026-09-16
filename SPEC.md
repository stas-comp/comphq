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
- Big, clear buttons with words, not just icons.
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
- A staff member can view a month at a time (weeks start Monday, with **Saturdays highlighted**), or a list of the next 12 weeks.
- A staff member can add an event with: title, date, optional end date (for events lasting several days), all-day or a start time (and optional end time), notes, how it repeats (never, weekly, monthly, yearly) with an optional "until" date, and **how far ahead it should appear in the Saturday Briefing** (for example "6 weeks").
- A staff member can change **just one occurrence** of a repeating event (for example, move this year's concert) without affecting other years, or change the whole series.
- A staff member can cancel just one occurrence.
- A staff member can remove an event, and restore it from **Removed events**.
- A staff member can see task due dates on the calendar, and click one to open the task.

**What it looks and feels like**
- A clean month grid. Events are solid accent-coloured chips. Task deadlines are outlined chips with a tick-box icon, so the two are never confused.
- Clicking a day starts a new event on that date. Clicking a repeating event asks **"Change just this one"** or **"Change all"**.

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
- **2.12** The Calendar opens on this month: weeks start Monday, Saturdays are highlighted, and today is marked. It has Previous, Next and Today buttons. The List view shows the next 12 weeks in date order.
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
- **2.33** My jobs shows two areas. **My jobs** holds the person's Working on now and Up next jobs (numbered by priority) with their workload blocks, matching their Team view column. **Up for grabs** holds jobs nobody is on (In progress and To do, in priority order), then ideas nobody is on. Jobs belonging only to other people, and finished jobs, don't appear.
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
- Comments or chat on tasks, sub-tasks, and checklists.
- Automatic "too busy" warnings, per-person workload limits, or a separate priority list for each person.
- Restoring backups from inside the app. Restores are done with TrueNAS snapshots, following the README.

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
`id` · `task_id` · `person_id` · `action` (`created`|`moved`|`assigned`|`unassigned`|`due_changed`|`size_changed`|`edited`|`reopened`|`removed`|`restored`) · `detail` (short JSON) · `at`

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
