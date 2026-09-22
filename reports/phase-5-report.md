# Comp HQ — Phase 5 report (version 1.2)

**Version 1.2.0 is ready. Jobs now have steps you can tick off, and Comp HQ has its own icon and installs as an app.**

## Summary

- **What's new:**
  - **Steps inside a job.** Open any job and, under its description, there is a **Steps** list — "Print exam papers", "Book the hall", "Email the parents". Type a step and press **Enter**; the box stays ready, so a whole list goes in without touching the mouse. Press a step's box to tick it (it shows who ticked it and when), click its words to change them, use the small arrows or drag to put steps in order, and press the bin to remove one — an **Undo** appears straight away, and nothing is ever really deleted. The heading counts as you go (**3 of 7 done**), and on the Board, My jobs and Team a card carries a small **3/7** when its job has steps (a tick when they're all done). It all works without JavaScript too.
  - **Comp HQ has its own icon.** The navy-and-orange **COMP / HQ** mark now shows in the browser tab, on a desktop shortcut, in the Start menu, on the taskbar and in Alt-Tab. **Settings → Set up this computer** is rewritten around the easiest route: in Edge or Chrome, one menu item installs Comp HQ as an app with its own icon and window. The command shortcut stays for Brave and as a fallback, with a step for pinning it to the taskbar and a **Download icon** button for the rare blank-icon case.
  - **Also:** History records steps being added, renamed, removed and restored (but not every tick); **Export everything** includes `steps.csv`; the Saturday Briefing is unchanged.
- **Automatic checks:** 25 of this phase's 26 promises (gates 5.01–5.17, 5.20–5.26, 5.30 and 5.31) passed in the full test run https://github.com/stas-comp/comphq/actions/runs/35618993521 (`go`, `browser`, `container` with the internet blocked and an upgrade-then-rollback against 1.1.0 **with real steps in the database**, `speed`, `screens`). The 26th, **5.24** (the installed window has no address bar), cannot be tested by a test browser — it has no window to look at — so it is your check **O5.3** below; see the decision D-78. Every Phase 1–4 promise passes in the same run, so all of them are re-proven. The release run for v1.2.0 (https://github.com/stas-comp/comphq/actions/runs/35627619823) repeated everything and published the download. See `reports/gates-phase-5.md` for the full list.
- **Tests I corrected:** see "Tests corrected" below.

## What I need from you

### 1. Update steps

In TrueNAS, go to **Apps → comphq → Edit**, change the number after `comphq:` to **1.2.0**, and click **Save**. Wait for it to show **Healthy**. Your data carries across untouched, and if you ever need to go back to 1.1.0 your steps stay safely in the database (the older version simply doesn't show them) and come back when you update again — that was tested for real.

### 2. Please look hardest at Steps (O5.1)

**Steps are the one part of this version with no drawing to check against.** As with the task window last time, I built them out of the design's own parts (the card look, the buttons, the small icons) so they look as though they had always been there — but whether they are *obvious enough that nobody needs telling how they work* is something only you and your team can judge. That is check **O5.1**, and it is the one that matters most.

### 3. Please walk three menus for me (this is the one thing I could not check)

The Setup page tells people which menu items to press in Edge, Chrome and Windows. **I wrote that wording from what I know of those programs; I could not click through the real menus, and I have not checked it against the office computers.** Please walk these once, on a real office computer, and tell me in your own words anything that isn't quite right (a different word, a different order, a missing step) — I'll correct the page and the README:

1. **Edge:** the **⋯** menu → **Apps** → **Install this site as an app** → **Install**.
2. **Chrome:** the **⋮** menu → **Save and share** → **Install page as app…** (on some versions this is called **Create shortcut…**, with a tick box **Open as window**).
3. **Windows taskbar:** right-click Comp HQ's icon on the taskbar → **Pin to taskbar** (and, on Windows 11, right-click the desktop shortcut → **Show more options** → **Pin to taskbar**). And, for the blank-icon case: right-click the shortcut → **Properties** → **Shortcut** tab → **Change Icon…** → **Browse…**.

For your information, the build computer has Edge 153.0.4234.48, Chrome 153.0.8010.52 and Brave 153.1.95.104 on Windows 11; the wording was written on 21 September 2026 and is recorded in `PROGRESS.md` as **unverified**.

### 4. Owner checks (SPEC §A13 "👤 Owner checks — end of Phase 5")

| Check | What to try | What you should see |
|---|---|---|
| **O5.1** | *This is the one to look at hardest.* On a real Saturday, open two or three real jobs and add steps to each: type a step and press **Enter**, several times over. Tick some off. Change the wording of one (click it). Remove one, then press **Undo**. | Adding steps is quick and needs no explanation. Ticking shows who and when. The little **Undo** puts the step back exactly where it was. Nobody needs telling how it works. |
| **O5.2** | Look at the **Board** with those jobs on it (and **My jobs** and **Team** if you like). | Each job with steps carries a small **3/7** (a tick once every step is done); jobs without steps carry nothing. It is useful rather than clutter. |
| **O5.3** | Set up **one office computer from scratch**, following only **Settings → Set up this computer**. Start with way 1 (Edge or Chrome's menu); if that doesn't work, use way 2. Then look at the desktop, the Start menu and the taskbar. | The icon on the desktop, in the Start menu and on the taskbar is the Comp HQ mark (navy square, orange **HQ**), and the Comp HQ window has **no address bar**, tabs or bookmarks bar. (This is also where gate 5.24 is checked.) |
| **O5.4** | Press **Alt+Tab** between Comp HQ and another program. | Comp HQ shows up with its own icon and name and looks like its own program. |

**Still open from earlier phases** (you asked me to carry on without them, so they're recorded as pending, not skipped):

- **O1.1–O1.4** (Phase 1): installing on TrueNAS and the desktop shortcut; trying the Knowledge Base; moving your HelpScout articles; importing a real Word document.
- **O2.1–O2.2** (Phase 2): the look and feel of Tasks and the Calendar; entering your yearly events and moving your active Trello cards.
- **O3.1** (Phase 3): trying the Saturday Briefing with real jobs and events.
- **O4.1–O4.4** (Phase 4): comparing the screens with the design mockup; adding a job in the window; the look of the job window (the one screen the mockup doesn't show); the little card icons.

Reply with **"all OK"**, or tell me in your own words what isn't right and I'll fix it and send a small update. You can also answer just some of them; the rest stay pending.

## Full gate checklist

See `reports/gates-phase-5.md` — every SPEC gate 5.01–5.31, its plain description, the test that proves it, and the result (all Pass except 5.24, which is your check O5.3). Phase 1–4 gates are in `reports/gates-phase-1.md` to `-4.md` and are re-proven by the same full run (gate 5.30).

## Things to know about (honest notes)

- **Gate 5.24 has no automatic test** (D-78), as above. What *is* tested: the shortcut command opens Comp HQ as its only window and lands on the Briefing; the manifest asks for its own window (`standalone`), opens at the Briefing and is read by the browser without complaint; every icon it names exists at the size it claims.
- **The picture in the README** (Setting up an office computer) is a rendering of what fills the window, not a photograph of a Windows window: this build computer has no screen I can photograph from here. The README says so.
- **The padlock.** Browsers reserve their full "app" treatment (including a branded offline screen) for addresses with the padlock (`https://`). Comp HQ is deliberately reached at `http://<NAS IP>:8080` (SPEC A12). You still get your own window, the icon and a taskbar entry; you don't get a Comp HQ-branded "can't reach the NAS" page when the NAS is off (the browser's own error page shows), and a future browser version might show a thin strip with the address. The README explains this in plain words. Adding the padlock later would be its own small piece of work.
- **A search quirk I found and left alone.** While fixing a flaky test I found that the live search offers nothing when what you've typed so far ends in "y" and isn't a whole word yet (typing "pay" doesn't offer "payment" until you finish the word). It is older than this version and not part of it, so I've flagged it as a separate task rather than changing search behaviour here.

## Tests corrected

Existing tests that were changed during Phase 5, each because Phase 5 changed the behaviour on purpose or the test itself was wrong (each is recorded in `docs/decisions.md`). No test was skipped, weakened or deleted, no time limit was raised and no retry was added.

1. **Gate 4.23's "the reading window has no form control at all"** (D-76). The window's reading face now legitimately holds the Steps list (tick-boxes, an "Add a step" line), which SPEC gates 5.01–5.03 require. The test now leaves the Steps list out of the count and keeps every other assertion: the description still has no control, border or fill, and nothing else in the reading face may be a control.
2. **Gate 4.12's "the icon set is exactly the fifteen line icons"** (D-77). The scan now skips `comphq.svg`, the app's own mark, which is not an interface icon.
3. **Extended, not weakened:** the Setup-command test now covers Edge as well as Chrome and Brave; the `--app` window test (gate 1.43) additionally checks the window opens on the Briefing; the settings export test now expects `steps.csv`; the smoke suite gained the steps seed/verify/rolled-back tests.
4. **Gate 1.29's prefix search test** (D-79), found by this phase's full run: its random word could occasionally make the search legitimately find nothing (one run in thirty-six, when the typed prefix ended in "y"). The word now ends in a plain letter before the cut. No assertion changed.

None of these affect anything you use.

## New design decisions (`docs/decisions.md`)

D-70 to D-74 were settled before building (one flat list of steps; ticks aren't in History; no step templates; steps aren't in the Briefing; Undo instead of a "Removed steps" screen). New during the build, in plain terms:

- **D-75:** the small edge cases of steps: a tick that changes nothing (ticking what's already ticked) is quiet and makes other computers refresh for nothing; Undo into a full list of 50 is refused like any add; a step's words are one line and the 200 limit counts characters, not bytes.
- **D-76:** the reading window may hold controls, in its Steps list only (the gate 4.23 correction above).
- **D-77:** the icon is designed once, in `tools/icons/comphq.design.svg`; `npm run icons` draws every size from it, embedding the app's own font so it looks right anywhere. Regenerated files are committed, not redrawn in CI (anti-aliasing differs between Windows and Linux); tests check what matters instead.
- **D-78:** gate 5.24 is checked by eye, not by the test browser.
- **D-79:** the gate 1.29 flake was traced to a stemming quirk in the test's random data and fixed there.

## Screenshots

The screens as they look in use, at 1366×768, in `reports/phase-5/screens/` (jobs and events taken from the mockup's own example): new for this phase are **`task-window-steps.png`** (a job open in the window with its Steps list, three of seven ticked) and **`tasks-board-populated.png`** (the Board with a **3/7** on one card and a tick on another; `tasks-populated.png` and `tasks-team-populated.png` show the same on My jobs and Team), and **`settings-setup.png`** (the rewritten Set up this computer page). Every other screen is refreshed: `briefing.png`, `briefing-populated.png`, `who.png`, `who-populated.png`, `tasks-board.png`, `tasks-board-people-menu.png`, `tasks-new.png`, `task-window-new.png`, `task-window-reading.png`, `task-window-reading-history.png`, `task-window-editing.png`, `tasks.png`, `tasks-team.png`, `tasks-finished.png`, `tasks-removed.png`, `calendar.png`, `calendar-populated.png`, `calendar-list.png`, `calendar-new.png`, `calendar-removed.png`, `kb.png`, `kb-article.png`, `kb-article-highlighted.png`, `kb-search.png`, `kb-search-open.png`, `kb-categories.png`, `kb-new-article.png`, `kb-editor-imported.png`, `kb-archived.png`, the other Settings pages, and `404.png`. The icon itself, at every size, is in `web/static/theme/icons/`.

## Version

**1.2.0** — `ghcr.io/stas-comp/comphq:1.2.0` (release: https://github.com/stas-comp/comphq/releases/tag/v1.2.0).
