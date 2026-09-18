# Comp HQ — Phase 3 report (final)

**Version 1.0.0 is ready. The build is complete.**

## Summary

- **What's new:** The **Saturday Briefing** is now the first thing everyone sees when they open Comp HQ: what must be done by Saturday, what's happening this week, and what's coming up, with stamps like **OVERDUE**, **TODAY**, **MON 21 SEP**, **IN 6 WEEKS** and **YOURS**, a **Just mine** switch, and cards that open their job or event. It refreshes itself when anyone changes a job or an event.
- **Automatic checks:** all 103 promises (gates 1.01–3.13, across all three phases) passed in the full test run https://github.com/stas-comp/comphq/actions/runs/35394576555 (`go`, `browser`, `container` with the internet blocked and an upgrade-then-rollback against 0.2.0, `speed`, `screens`), and again in the release run https://github.com/stas-comp/comphq/actions/runs/35395930805 (which also published the download). See `reports/gates-phase-3.md` (with `reports/gates-phase-1.md` and `reports/gates-phase-2.md`) for the full list, and `reports/phase-3/screens/` for a screenshot of every page type.
- **Tests I corrected:** see "Tests corrected" below.

## What I need from you

### 1. Update steps

In TrueNAS, go to **Apps → comphq → Edit**, change the number after `comphq:` to **1.0.0**, and click **Save**. Wait for it to show **Healthy**. Your data carries across untouched.

### 2. Owner checks (SPEC §A13 "👤 Owner checks — end of Phase 3")

| Check | What to try | What you should see |
|---|---|---|
| **O3.1** | Open Comp HQ on a real Saturday (or a practice run — put a couple of jobs due this week, an event this week with a note, and a yearly event with a "show ahead" time into Comp HQ first). Read the README's "Using the Saturday Briefing" if you like. | The Briefing shows what you'd expect for the week: nothing missing, nothing surprising. Late jobs say OVERDUE, your own jobs say YOURS (and **Just mine** keeps only those), this week's events show their notes, and a far-off event appears under **Coming up** once its "show ahead" time has started. |
| **O3.2** | Spend a few minutes across the whole app: Briefing, Knowledge Base, Tasks (My jobs, Board, Team), Calendar, Settings. | The whole app feels like one product — the same deep navy frame and signal-orange accent everywhere — and nothing feels unfinished. |

**Still open from earlier phases** (you asked me to carry on without them, so they're recorded as pending, not skipped):

- **O1.1–O1.4** (Phase 1): installing on TrueNAS and the desktop shortcut; trying the Knowledge Base; moving your HelpScout articles (**Settings → Content check** should show 0); importing a real Word document.
- **O2.1–O2.2** (Phase 2): the look and feel of Tasks and the Calendar (including whether the Team view makes it easy to see who can take on more); entering your yearly events and moving your active Trello cards so Trello can be closed.

Every one of these is a real-content check that only you can do, and each is walked through in the README (see "Your hands-on moments").

Reply with **"all OK"**, or tell me in your own words what isn't right and I'll fix it and send a small update. You can also answer just some of them; the rest stay pending.

## Full gate checklist

See `reports/gates-phase-3.md` — every SPEC gate 3.01–3.13, its plain description, the test(s) that prove it, and the result (all Pass). Phase 1's and Phase 2's gates are in `reports/gates-phase-1.md` and `reports/gates-phase-2.md`, and are re-proven by the same full run (103 in all).

## Tests corrected

Found and fixed by this project's own tests during Phase 3 (each recorded in `docs/decisions.md`):

1. **A Knowledge Base search test failed by chance about one run in thirty** (D-58). It had already failed once during Phase 2 and passed on a re-run; this time I traced it instead. The test's made-up word sometimes ended in a way that the search's word-stemming (finding "printers" from "printer") treats differently, so the search legitimately found nothing. Only the test's data was changed; the check itself is the same.
2. **A second search test could type into a page that was about to be replaced** (D-58), found by running the search tests 15 times over. It now waits for the page to have really loaded, like its neighbours.
3. **Three earlier tests expected opening Comp HQ to land on the Knowledge Base** (the picker tests and the root-page test), and one expected "Coming soon" on the Briefing (gate 1.08). SPEC gate 3.01 changes both facts on purpose, so these were updated (allowed by the plan's rule for exactly this case; noted at the top of `reports/gates-phase-3.md`).

None of these affect anything you use — all were caught and fixed before release.

## New design decisions (`docs/decisions.md`)

4 entries, D-55 through D-58. In plain terms: Phase 3 started while Phase 2's checks were still pending, as you instructed (D-55); the Briefing owns the home page, and "Just mine" filters jobs only, since events belong to nobody (D-56); one combined "something changed" counter makes the Briefing refresh, with every Calendar change saved together with its counter (D-57); and the two search-test fixes above (D-58).

## Screenshots

Every page type, captured at 1366×768, in `reports/phase-3/screens/`: `who.png`, `briefing.png` (empty), `briefing-populated.png` (a Saturday with content in all three sections), `kb.png`, `kb-categories.png`, `kb-new-article.png`, `kb-archived.png`, `kb-search.png`, `tasks.png` (My jobs), `tasks-board.png`, `tasks-team.png`, `tasks-finished.png`, `tasks-removed.png`, `calendar.png`, `calendar-list.png`, `calendar-new.png`, `calendar-removed.png`, `settings-people.png`, `settings-backups.png`, `settings-content-check.png`, `settings-setup.png`, `settings-about.png`, `404.png`.

## Version

**1.0.0** — `ghcr.io/stas-comp/comphq:1.0.0` (release: https://github.com/stas-comp/comphq/releases/tag/v1.0.0).
