# Comp HQ

Comp HQ is a small, private staff hub: a Knowledge Base, a task Board, a Team view, a Calendar, and a daily Briefing — self-hosted on your own TrueNAS, with no logins and no outside services.

This README is written for you, the owner — plain language, no coding knowledge needed. It's updated as the app is built; the Briefing section is still to come.

## Opening Comp HQ

Once it's installed on your TrueNAS (see "Installing on TrueNAS" below), open it from any computer on your network by going to:

```
http://<NAS IP>:8080
```

Replace `<NAS IP>` with your NAS's address (for example `http://192.168.1.50:8080`). The first time anyone opens it on a given computer, they'll be asked to pick their name from a list, or add it if it's not there. That computer remembers the name after that.

## Setting up an office computer

For the best experience, open Comp HQ in its own window (like a real app) instead of a browser tab, using a desktop shortcut.

1. Open Comp HQ in Chrome or Brave, as above.
2. Go to **Settings → Set up this computer**.
3. Under your browser (Chrome or Brave), click **Copy** to copy the shortcut command.
4. Right-click an empty spot on the Desktop, then choose **New → Shortcut**.
5. Paste the copied command into the box, click **Next**, then **Finish**.
6. Double-click the new shortcut. Comp HQ opens in its own window, with its own icon.

**If Windows says it can't find the program:** right-click your browser's existing icon (from the Start menu or Taskbar), choose **Properties**, and look at the **Target** box — that's the real install location. The Setup page in Comp HQ explains how to adjust the shortcut command to match.

## Installing on TrueNAS

These steps use TrueNAS SCALE 25.04's menu names.

1. **Confirm your NAS has a fixed IP address**, so Comp HQ's address never changes. Go to **Network → Interfaces** and check your NAS's IP is set to static, not DHCP. If it's not, your network administrator (or TrueNAS's own documentation) can help you set one.
2. **Create a storage location for Comp HQ's data:**
   - Go to **Datasets → Add Dataset**.
   - Name it `comphq`.
   - Choose the **Apps** preset.
   - Save.
3. **Set up daily backups of that data:**
   - Go to **Data Protection → Periodic Snapshot Tasks → Add**.
   - Choose the `comphq` dataset.
   - Set it to run **daily**, and keep snapshots for **30 days**.
   - Save.
4. **Install Comp HQ:**
   - Go to **Apps → Discover Apps**.
   - Click the **⋮** (three-dot) menu in the top corner, then **Install via YAML**.
   - Open the file `deploy/truenas.yaml` from the Comp HQ project (ask whoever set up the build for this file, or find it in the project folder), and paste its entire contents into the box.
   - Change the three lines marked `# ← CHANGE THIS`:
     - The storage pool path — replace `YOUR-POOL` with your pool's name (visible under **Storage**).
     - The time zone — replace `Europe/London` with your own. Common examples: `America/New_York`, `America/Chicago`, `America/Los_Angeles`, `Europe/London`, `Australia/Sydney`. If you're not sure of yours, search "IANA time zone" plus your city.
     - The port (`8080`) — only change this if TrueNAS tells you it's already used by something else. Never use `80` or `443` (TrueNAS's own screen uses those).
   - Click **Save** / **Install**.
5. **Check it started properly:** in **Apps**, Comp HQ should show as **Healthy** within about a minute. If it doesn't, see "It won't open — what now?" below.
6. **Open it:** go to `http://<NAS IP>:8080` in your browser (see "Opening Comp HQ" above).

## Updating

When a new version is ready, you'll be told the version number and a short summary of what's new.

1. Go to **Apps → comphq → Edit**.
2. Find the line with the image name (it ends in a version number, like `comphq:0.1.0`) and change the number to the new version.
3. Click **Save**.

TrueNAS downloads the new version and restarts Comp HQ automatically. Your data is never affected by an update.

## Undoing an update

If something looks wrong after an update:

1. Go to **Apps → comphq → Edit**.
2. Change the version number back to the previous one.
3. Click **Save**.

If the release notes for that update said it wasn't safe to simply undo this way, you'll also be told to restore the snapshot taken just before the update — see "Restoring" below.

## Backing up

Comp HQ protects your data two ways, and you don't need to do anything for either to keep working:

1. **TrueNAS snapshots** (the main safety net). Set up once, during "Installing on TrueNAS" above — a snapshot of the whole `comphq` dataset is taken every day and kept for 30 days.
2. **Automatic nightly backups inside Comp HQ.** Every night at 3am (your NAS's time zone), and once whenever the app has been off for more than a day, Comp HQ copies its own database to `backups/daily` inside the `comphq` folder, keeping the last 30 copies. Go to **Settings → Backups** to see when the last one happened. If it's ever more than 2 days old, a red warning banner appears at the top of every page until it's fixed — if you see that banner, check that the app is actually running (see "It won't open — what now?" below) and let whoever helps you with Comp HQ know.

**A second safety net you can trigger yourself, any time:**

- Go to **Settings → Export everything**. Your browser downloads one `.zip` file (named something like `comphq-export-2026-09-17.zip`).
- Save that file somewhere safe — a USB drive, another computer, a cloud drive, anywhere outside the NAS. It contains every article as a readable page (with its pictures), plus a full copy of the database.
- This doesn't need TrueNAS or the network at all to open later — see "Restoring" below.

## Restoring

There are two different ways to undo a problem, depending on how big it is.

**For a single bad edit to one article (no NAS access needed):**

1. Open the article in Comp HQ.
2. Click **History**.
3. Find the version you want (each one shows who changed it and when) and click it.
4. Click **Restore this version**.

This immediately publishes that older text again, as a brand new version — nothing is lost, and you can always go back further if needed.

**For anything bigger — the whole app looks wrong, or data seems to have disappeared (preferred: roll back a snapshot):**

1. In TrueNAS, go to **Datasets**.
2. Find the `comphq` dataset and click **Snapshots**.
3. Pick a snapshot from before the problem started, and choose **Rollback**.
4. Restart Comp HQ: go to **Apps → comphq**, click the **⋮** menu, then **Restart**.

Two more copies of the database also live inside the `comphq` folder, purely so that anyone helping you (a family member, a future agent, a technical friend) has a clean file to work from if needed: `backups/daily` (one per day, last 30 kept) and `backups/pre-update` (one from just before each update, last 10 kept). You won't normally need to touch these yourself — the two methods above cover everything.

## It won't open — what now?

Work through these in order; most problems are fixed by the first or second step.

1. **Check its status.** In TrueNAS, go to **Apps**. Comp HQ should show as **Healthy** (a green dot). If it shows anything else (Stopped, Crashed, Deploying for a long time), that tells you where the problem is.
2. **Restart it.** Click the **⋮** menu next to Comp HQ, then **Restart**. Wait about a minute and check **Apps** again.
3. **Check the version number.** Go to **Apps → comphq → Edit** and look at the image line — does it match the version you expect (Comp HQ also shows its own version at **Settings → About**)? If a recent update seems to be the cause, see "Undoing an update" above.
4. **Still stuck? Roll back to the last known-good snapshot.** Follow "Restoring" above (the "anything bigger" steps), then try opening Comp HQ again.

If none of this helps, the NAS's own **Apps → comphq → Logs** screen shows what the app is doing, which is useful information to pass along to whoever helps you with Comp HQ.

## Moving your HelpScout articles

Import works one way, from HelpScout into Comp HQ — there's no link back, and no ongoing sync. Once an article is in Comp HQ, you edit it there from then on.

For each article in HelpScout (there should be fewer than 20):

1. Open the article in HelpScout, in your browser.
2. Select all of its content (click inside the article body, then Ctrl+A) and copy it (Ctrl+C).
3. In Comp HQ, go to **Knowledge Base**, choose the right category (or create a new one), and click **New article**.
4. Type the title, click into the content area, and paste (Ctrl+V). Formatting, tables and pictures come across automatically.
5. Click **Publish**.

Once you've moved everything, go to **Settings → Content check**. It should show 0 — if anything's listed, open it, look for a red dashed box (something that didn't come across cleanly), fix or remove it, and publish again.

## Using Tasks

Click **Tasks** in the left-hand list. Across the top there's a switch: **My jobs · Board · Team**.

**My jobs** is where Tasks opens. On the left are your own jobs: what you're **Working on now**, then **Up next** in priority order, with a row of little blocks showing how much you've got on (a small job is one block, a medium two, a large four). On the right is **Up for grabs** — jobs, then ideas, that nobody has taken yet.

- **To take a job**, press **Take it**, or drag it from Up for grabs into your Up next list. Dragging puts it exactly where you drop it; **Take it** keeps the job's place in the priority order. Taking an idea moves it to To do.
- If two people go for the same job at the same moment, the second person is told who got there first, and nothing changes for them.
- **Start** moves a job to In progress, **Done** finishes it, and **Give back** takes you off a job you can't do — it returns to Up for grabs if nobody else is on it.

The **Board** shows every job in four columns: **Ideas → To do → In progress → Done**. Add a job at the top, drag cards between columns or use the small buttons (they do the same thing). In **To do**, the top of the column is the most important. Click a card's title to open it: you can change its title, notes, people, due date and size, remove it, and see the **Activity** list of who changed what. Jobs that have sat in Done for more than two weeks tuck away into **Finished tasks** (a link at the top of the Board), where **Reopen** brings one back. **Removed tasks** lists removed ones, where **Restore** brings them back. Nothing is ever permanently deleted.

The **Team** view has one column for each person, plus **Unassigned**, each showing that person's jobs and how much they have on. It's the quick way to see who can take on more. Drag a job onto someone else's column (or use **Assign to…**) to hand it over.

Any page that shows the same jobs on more than one computer refreshes itself within a minute, or straight away when you click back into the window.

## Using the Calendar

Click **Calendar**. It opens on this month, weeks starting on Monday, with Saturdays shaded and today outlined. Use **Previous**, **Next** and **Today** to move around, or **List** for the next twelve weeks in date order. Jobs with a due date appear on their day as outlined boxes with a tick-box; events are solid orange. Click a job to open it.

- **To add an event**, click a day's number (or **Add event**). Give it a title and a date. You can also give it an end date (for something lasting several days), a start and end time (leave the start blank for an all-day event), notes, how often it **Repeats** (Never, Weekly, Monthly or Yearly, with an optional **Until** date), and **Show in briefing ahead** — how many days or weeks before it should start appearing in the Saturday Briefing.
- Weekly events fall on the same weekday. Monthly ones fall on the same date, and an event on the 31st shows on the last day of shorter months. Yearly ones fall on the same date, and 29 February shows on 28 February in other years.
- **To change a repeating event**, click any of its dates. You're asked **Change just this one** or **Change all**. *Just this one* only offers a new date and time — for example, moving this year's concert a week — or **Cancel just this one**; every other date is untouched. *Change all* edits the whole series, including its title and notes, everywhere. A date you moved on its own keeps its own date but takes on any new title.
- **Remove event** (at the bottom of an event) hides the whole event. **Removed events**, linked at the top, lists them and lets you **Restore** one.
- Each event shows who last changed it and when.

## Entering your yearly events

This is your Phase 2 check with real content. Set aside about half an hour.

1. Go to **Calendar** and press **Add event**.
2. For each yearly event (the concert, the card campaign, exams, and so on): type the title, pick the date of the *next* one, set **Repeats** to **Yearly**, and choose **Show in briefing ahead** — for example 6 weeks for a big campaign. Add notes if it helps ("Print exam papers").
3. Press **Save**. Look at the month it falls in, then at the same month next year, to check it's there.
4. If one year is different, click that year's date, choose **Change just this one**, and move it — next year stays as it was.

## Moving your Trello cards

As with HelpScout, this is one way and one time. Only move cards that are still active — finished ones can stay behind in Trello.

1. In **Tasks**, open the **Board**.
2. For each active Trello card, type its title in the box at the top, pick the column it's in now (**Ideas**, **To do** or **In progress**), add the due date if it has one, choose the people, and press **Add task**.
3. Open the new card and paste in any notes from Trello, and set its size (small, medium or large) so the Team view shows people's load fairly.
4. Put the **To do** cards in priority order, most important at the top.
5. When you've been through them all, look at the **Team** view and check it matches who's really doing what. If nothing is missing, you can close Trello.

## Keeping a copy of your tasks and events

**Settings → Export everything** now also includes two spreadsheets, `tasks.csv` and `events.csv`, that open straight in Excel with clear column names and readable dates (removed items are included and marked). The zip's own `index.html` is still the place to read your articles offline.

## Word samples

If you have real Word documents (with pictures and tables, nothing private) you'd like Import from Word tested against, put 1–3 of them in the `samples\word` folder inside the Comp HQ project folder. This is entirely optional.

## Your hands-on moments

Everything you're asked to do yourself, start to finish, and where to find the steps:

| When | What | Where to find the steps |
|---|---|---|
| Once, before building starts | Sign in to GitHub so the build agent can create the project. | Handled live with the build agent when it happens; nothing to prepare. |
| Once, after the first release | Switch the Comp HQ download to "public" on GitHub. | The build agent gives you the exact click-by-click steps when the first release is ready. |
| Optional, during Phase 1 | Add 1–3 real Word documents for testing. | "Word samples" above. |
| Once, end of Phase 1 | Install Comp HQ on TrueNAS, with automatic snapshots. | "Installing on TrueNAS" above. |
| Once per office computer | Create the desktop shortcut and pick a name. | "Setting up an office computer" above. |
| Once, end of Phase 1 | Copy your HelpScout articles into Comp HQ. | "Moving your HelpScout articles" above. |
| End of Phase 2 | Enter your yearly events, and move your active Trello cards. | "Entering your yearly events" and "Moving your Trello cards" above. |
| End of each phase (3 times) | Check what's been built so far. | Walked through live with the build agent at each phase's close; nothing to prepare. |
| Each future update | Change the version number in TrueNAS. | "Updating" above. |
