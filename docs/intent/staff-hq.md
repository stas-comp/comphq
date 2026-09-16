# Staff HQ — Confirmed Intent

Status: **Confirmed by the owner on 2026-09-14** (interview complete, explicit "yes").
This is the "what and why". It is not a spec. The spec is written from this document.

---

## Outcome

A no-login **Staff HQ** web app, hosted on our own TrueNAS server, opened from a desktop
shortcut so it looks and feels like a real app (its own window, no browser tabs or address bar).

It has four parts:

1. **Saturday Briefing** (the first screen you see)
   - Shows everything due before next Saturday, flagged as "must be done today"
     (we only meet on Saturdays, so this Saturday is the last chance).
   - Highlights upcoming events.
   - Shows big yearly items early, based on each event's "advance notice" setting
     (e.g. "show the Christmas card campaign 6 weeks ahead").
   - Example: "Exams are next week, so print exam papers today."

2. **Knowledge Base** (the everyday core, together with search)
   - Article library, organised into categories.
   - Search that shows the **matching passage with the search words highlighted**, and clicking a
     result opens the article **scrolled to that passage**, highlighted.
   - New or edited articles are searchable immediately.
   - Anyone can publish, edit, and archive. Archived articles are hidden from search but never deleted.
   - A simple editor with normal formatting and pasted photos/screenshots. No Markdown knowledge needed.
   - A simple edit history, showing who changed what, so mistakes can be undone.
   - Our existing HelpScout articles (fewer than 20) are moved over. Doing it by hand is fine
     if an automatic import would take a lot of extra code.
   - Content types: mailing SOPs, live event setup (e.g. the Christmas concert), office facts
     (which printer is where, which toner it takes).

3. **Projects / Tasks** (replaces Trello, but doesn't have to copy it)
   - Anyone can post a possible job or project.
   - Anyone can assign a job to anyone.
   - Jobs can be prioritised and tracked through to done.
   - The goal: a convenient place to keep track of projects, see that people are working
     on things, and prioritise.

4. **Calendar**
   - Deadlines and events. Task due dates appear on it.
   - **Recurring events**: yearly (main need, e.g. Christmas concert, card campaign, exams),
     plus monthly and weekly.
   - **Advance notice** per event: how early it should start appearing in the Saturday Briefing.
   - Change a single occurrence (e.g. this year's concert moved) without affecting future years.

## Users

- A small, trusted team of nonprofit volunteer staff who work together in the office on Saturdays.
- Internal only. No customers or public.
- **No logins.** Each computer asks "who are you?" once (pick your name from a list) so the app can
  show "my tasks" and record who made an edit. There are no passwords and no roles.

## Why now

- Logins make people slow to use our tools. Staff don't use them, and that is the real problem
  (HelpScout, Trello).
- HelpScout's free plan forces a login every couple of months and could change or disappear at any
  time, taking our articles with it. The same risk applies to Trello's free plan.

## Success looks like

- Staff open it every Saturday **without being told to**, because the briefing shows what matters
  this week.
- Search finds the right answer in seconds, down to the right paragraph.
- All HelpScout articles are in it, and we no longer depend on HelpScout or Trello.

## Constraints

- No logins, no paid services, no AI, no internet connection needed.
- Runs on our existing **TrueNAS SCALE 25.04.2.6 ("Fangtooth")**. All data lives on the NAS.
- Staff use it through a web browser, launched from a desktop shortcut that opens an app-like window.
- Office network only.
- The owner has **no coding experience** and cannot review code. It must be possible to keep it
  running without being a developer.

## Build order (confirmed)

1. App layout (the "HQ" frame with room for sections) + **Knowledge Base** (time-sensitive: get off HelpScout)
2. **Projects/Tasks** and **Calendar**
3. **Saturday Briefing** (it pulls from tasks and calendar)

## Out of scope

- AI question answering or AI anything.
- Email, phone, or push notifications. Reminders only appear when you open the app.
- Access from outside the office network.
- Logins, user roles, permissions.
- A public or customer-facing help centre.
- Separate programs installed on each computer.
- Syncing with Google Calendar or Outlook.
- A complex HelpScout import tool (unless it turns out to be easy).
