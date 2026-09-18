# Comp HQ — Changes

## 0.2.0

Phase 2 is complete: **Tasks** and the **Calendar** are here, replacing Trello. Tasks opens on **My jobs** — your own jobs next to **Up for grabs**, where you can take a job (or an idea) by pressing **Take it** or dragging it across; if two people go for the same job at once, the second is told who got there first. The **Board** has four columns (Ideas, To do, In progress, Done) with drag-and-drop or buttons, a search and filters, and a card for each job showing people, due date, size and an OVERDUE stamp; each card has its own page with an Activity list. The **Team** view shows a column per person with a row of blocks for how much they have on, so it's easy to see who can take on more, and jobs can be handed over by dragging. Finished and removed jobs tuck away into their own lists and can always be brought back, and every screen refreshes itself when someone else makes a change. The **Calendar** shows a month (weeks start Monday, Saturdays shaded) or the next twelve weeks, with events that can repeat weekly, monthly or yearly, be moved or cancelled **just this once**, or be changed for every date at once; jobs with due dates appear on their day. **Export everything** now also includes two spreadsheets, `tasks.csv` and `events.csv`, that open straight in Excel. The README has new sections on using Tasks and the Calendar, entering your yearly events, and moving your Trello cards over.

## 0.1.0

The first real release — Phase 1 is complete. Comp HQ now has a full **Knowledge Base**: categories, a rich-text editor (headings, lists, links, tables, pictures), publishing with full version history and one-click restore, archiving, and instant search with highlighted matches. Pictures can be pasted, dropped, or copied in from a web page, and whole Word documents can be imported directly into the editor — tables, lists, pictures and all, with anything that can't come across clearly marked so nothing silently goes missing. Everyone on your network picks their name once per computer; no logins, no passwords. Settings now has **Export everything** (a one-file backup you can download any time), automatic nightly backups with a warning banner if one is ever overdue, a **Content check** page listing anything that needs a second look, and a **Set up this computer** page for a proper desktop shortcut. The README covers installing on TrueNAS, updating, backing up and restoring, and moving your existing HelpScout articles over.

## 0.0.2 (internal)

Second internal build. Adds real Knowledge Base content (categories, articles, edits, pasted images) for the first time, so the upgrade/rollback test (gate 1.42) has real data to carry across an upgrade, not just an empty database. This version is for testing only — nothing to install yet.

## 0.0.1 (internal)

First internal build. Proves the deployment path end to end: the app builds into a container image, TrueNAS can run it from `deploy/truenas.yaml`, and it restarts, recreates and updates without losing data. This version is for testing only — nothing to install yet.
