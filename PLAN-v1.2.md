# Comp HQ — Build Plan for v1.2

**Checklists, and making Comp HQ feel like a real app.**

This plan follows on from `PLAN.md`. Everything in `PLAN.md` §2 (how the build agent works, the
progress log, the git workflow, the definition of done, the standing engineering rules) and §5 (the
release procedure) still applies unchanged — this document only adds Phase 5. Where this plan says
"the release procedure", it means `PLAN.md` §5.2.

---

## 1. For the owner (please read this part)

v1.2 does two things.

**1. Steps inside a job (the checklist you asked for.)** Open any job and, under its description,
there's a list of steps you can tick off — "Print exam papers", "Book the hall", "Email the
parents". It shows how far along you are ("3 of 7 done"), and on the Board a job with steps carries
a small **3/7** so you can see at a glance what's nearly finished. Anyone can add a step, tick one,
rename one, put them in order, or remove one (with an Undo, and nothing is ever really deleted).

**2. Comp HQ looking and opening like an app, not a website.** You already have a desktop shortcut
that opens Comp HQ in its own window with no address bar — that part works today. What's missing is
the **icon**: Comp HQ has never had one, so Windows has nothing to show and you get a blank page
icon or the browser's. v1.2 gives Comp HQ its own icon — the navy-and-orange **COMP HQ** mark — and
it then appears in the browser tab, on the desktop shortcut, in the Start menu, on the taskbar and
when you Alt-Tab between windows. The **Settings → Set up this computer** page is rewritten around
the easiest route: in Edge or Chrome, one menu item installs Comp HQ as an app, with its own icon
and its own window. The command-line shortcut stays as the fallback, and for Brave.

### The honest limit on "a real app"

There is one thing v1.2 cannot give you, and it's worth knowing why.

Browsers reserve their *full* app treatment for addresses with the padlock (`https://`). Comp HQ is
reached at `http://<NAS IP>:8080`, which has no padlock — a deliberate decision when the spec was
written (`SPEC.md` A12), because the padlock on a private network means installing a certificate on
every office computer, which is fragile and easy to break.

What that costs you: almost nothing you'd notice. You still get your own window with no address
bar, your own icon, and a proper taskbar entry. What you don't get is a branded "Comp HQ can't
reach the NAS" screen when the server is off — you'd get the browser's own error page instead — and
there's a chance a future browser version shows a thin strip with the address in it.

If that strip ever appears and bothers you, the padlock can be added later as its own small piece of
work. It is not worth doing now. **Recommendation: proceed without it.**

### What you'll do yourself

| When | What | How long |
|---|---|---|
| End of Phase 5 | Add steps to two or three real jobs on a Saturday and tick some off. | 15 min |
| End of Phase 5 | Set up one office computer from scratch with the new Setup page, and confirm the icon and the window look right. | 10 min |
| End of Phase 5 | Change the version number in TrueNAS to `1.2.0`, as with every update. | 2 min |

### Decisions I've made for you

Each of these is the simpler choice. Say so if you'd rather have it the other way, and the plan
changes before building starts.

- **D-70. One list of steps per job, not several named lists.** Trello lets you have "Before" and
  "On the day" as separate lists on one card. One flat list is quicker to read and much less to
  learn. If a job really needs two groups, it's usually two jobs.
- **D-71. Ticking a step does not go into History.** History keeps recording who added, renamed,
  removed or restored a step, but not every tick — otherwise a job with 20 steps buries its own
  history. Each ticked step shows who ticked it and when, right on the step itself, which is where
  you'd look anyway.
- **D-72. No reusable step templates in v1.2.** Tempting for the yearly concert, but it needs its
  own screen to manage the templates. Copy last year's steps by hand this once; if that turns out
  to be a nuisance, templates become a small v1.3.
- **D-73. Steps don't appear in the Saturday Briefing.** The Briefing is about what's due and what's
  on, not how far along things are. A job with unfinished steps shows up there exactly as today.
- **D-74. Removing a step offers an Undo rather than a "Removed steps" screen.** Nothing is
  permanently deleted (SPEC A3) — a removed step stays in the database and in the export — but a
  whole screen for restoring them is more machinery than three volunteers need.

---

## 2. Before building: the spec

`SPEC.md` B9.10 says checklists "are v1.2 and will be specified separately". That spec work is part
of this plan and comes first, as **P5-00**. It is drafted by the owner's preferred model for
direction documents, not by the build agent, and it consists of these edits:

- **A11** — a new **Phase 5** gate list, being §3 of this document, moved in verbatim.
- **A12** — the line "Comments or chat on tasks, sub-tasks, and checklists" is split: comments and
  sub-tasks stay out; **checklists come in**, with a pointer to the new B10. Reusable step templates
  (D-72) join the "could be added later" list.
- **B3 (data model)** — the `task_checklist_items` table of §4.1 below.
- **B4 (precise behaviours)** — the step limits, what happens when two people act on the same step,
  and how steps carry across an upgrade and a rollback.
- **B10 (new)** — Checklists: the list inside the task window, the badge on a card, what goes in
  History (D-71), and the no-JavaScript path.
- **B11 (new)** — Looking like an app: the icon set, the manifest, what the Setup page says, and the
  padlock limit written down honestly so nobody re-litigates it later.

**Status: done, 2026-09-20.** All six edits are in `SPEC.md`, and D-70–D-74 are in
`docs/decisions.md`. `PLAN.md` §5.1 gained stop point S10 and §6 points at this document. The build
agent starts at **P5-01** and treats `SPEC.md` as the contract, this plan as the order of work.

---

## 3. Gates for Phase 5

Every gate below is proved by an automatic test, in the way `SPEC.md` B7 lays out, except the four
marked 👤 which the owner checks by hand.

### Steps inside a job

- **5.01** A job can carry a list of steps. The task window shows them under the description, headed
  **Steps**; a job with no steps shows an **Add a step** line and nothing else.
- **5.02** A step is added by typing its words and pressing Enter or **Add**. The box clears and
  stays ready, so several steps can be typed one after another without reaching for the mouse.
- **5.03** A step is ticked and unticked by pressing its box. A ticked step shows who ticked it and
  when, in the short form used everywhere else ("Jane, Sat 19 Sep").
- **5.04** The heading carries the count — **3 of 7 done** — with a small bar beside it, and reads
  **All 7 done** when every step is ticked.
- **5.05** A step's words can be changed in place, by anyone.
- **5.06** A step can be removed, and an **Undo** appears straight away and puts it back where it
  was. A removed step is never permanently deleted (A3): it stays in the database and in the export.
- **5.07** Steps can be put in order, by the same up and down icons used on cards (gate 4.11) and by
  dragging.
- **5.08** Every one of 5.02–5.07 still works with JavaScript switched off, as an ordinary form that
  reloads the page (A2 priority 2 — stability beats polish).
- **5.09** The job's own page at `/tasks/{id}` shows and edits the same steps the same way, so a
  refreshed window and a Calendar link still land somewhere whole (gate 4.27 stays true).
- **5.10** A card on the Board, on My jobs and on Team carries a small **3/7** when the job has
  steps, and carries nothing at all when it hasn't. It turns to a tick when all are done.
- **5.11** History records a step being added, renamed, removed and restored, and does **not** record
  ticks (D-71).
- **5.12** Two people on two computers: ticking a step somebody else ticked a moment ago does not
  produce an error, and acting on a step somebody else removed says so plainly instead of failing.
  The screen picks up the other person's change within a minute, or straight away on clicking back
  into the window, through the existing change counter.
- **5.13** A job holds up to **50** steps and a step holds up to **200** characters. Reaching either
  says so in plain words, and neither loses what was typed.
- **5.14** **Settings → Export everything** includes `steps.csv` — job, step, done or not, who ticked
  it, when, and whether it was removed — opening straight in Excel like the other two.
- **5.15** Steps survive an upgrade from 1.1.0 and a rollback back to 1.1.0: the rolled-back app
  ignores them, and upgrading again finds them all still there.
- **5.16** Steps follow the design system: no raw browser control anywhere (gate 4.05), icon-only
  buttons carry a tooltip and a spoken name (4.11), and the accessibility check passes with a long
  list open.
- **5.17** A job with 50 steps still opens within 1.5 seconds, and the Board with the test library
  loaded is no slower than today (gate 4.53 unchanged).

### Looking like an app

- **5.20** Comp HQ has its own icon: the stacked **COMP / HQ** wordmark, ink navy with the accent
  orange, drawn to read clearly down to 16 pixels. It is stored inside the app and never fetched
  from the internet (as with every font and icon, gates 4.01 and 4.12).
- **5.21** It is supplied in every form Windows and the browsers ask for: a multi-size `.ico`, PNGs
  at 16, 32, 48, 64, 128, 192, 256 and 512, and a maskable 512 with safe margins.
- **5.22** Every page links the icon, so it shows in the browser tab, in history and in bookmarks.
  Today no page links one at all, which is the whole reason shortcuts come out blank.
- **5.23** The manifest names Comp HQ, opens at the Briefing, asks for its own window, carries the
  navy as its window colour, and lists the icons of 5.21 — so an installed app takes the Comp HQ
  icon and colour rather than the browser's.
- **5.24** An installed Comp HQ window shows no address bar, no tabs and no bookmarks bar.
- **5.25** **Settings → Set up this computer** is rewritten to lead with the easiest route —
  installing Comp HQ as an app from the browser's own menu, in Edge or Chrome — then the
  command-line shortcut (as today, with its Copy button) for Brave and as a fallback, then pinning
  it to the taskbar, then a **Download icon** button and the three clicks that set the icon by hand
  if Windows ever shows a blank one. Each route names the browser it's for, and no route assumes the
  one before it worked.
- **5.26** The README's "Setting up an office computer" section matches the new page, step for step,
  with a picture of the finished window.

### Still true afterwards

- **5.30** Every gate from Phases 1 to 4 (1.01–4.54) still passes, unchanged, at the same
  thresholds.
- **5.31** Every screen still passes the accessibility check and still works from 1024 × 700 up to
  full HD, with no sideways scrolling that shouldn't be there.

### 👤 Owner checks — end of Phase 5

- **O5.1** On a real Saturday, add steps to two or three real jobs, tick some off, remove one and
  undo it. Confirm it's obvious enough that nobody needs telling how it works.
- **O5.2** Look at the Board with those jobs on it. Confirm the small **3/7** is useful rather than
  clutter.
- **O5.3** Set up one office computer from scratch, following only the new **Set up this computer**
  page. Confirm the icon on the desktop, in the Start menu and on the taskbar is the Comp HQ mark,
  and that the window has no address bar.
- **O5.4** Alt-Tab between Comp HQ and something else. Confirm Comp HQ looks like its own program.

---

## 4. Technical direction

### 4.1 Storage

One new table, in a new migration `migrations/tasks/0002_checklist.sql`. Adding a table is
rollback-safe on its own (SPEC B6): v1.1.0 never selects from it, so a rollback leaves the rows
untouched and a later upgrade finds them (gate 5.15).

```sql
CREATE TABLE task_checklist_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL REFERENCES tasks(id),
    text TEXT NOT NULL,
    position INTEGER NOT NULL,
    done_by INTEGER REFERENCES people(id),
    done_at TEXT,
    removed_at TEXT,
    created_by INTEGER NOT NULL REFERENCES people(id),
    created_at TEXT NOT NULL,
    updated_by INTEGER NOT NULL REFERENCES people(id),
    updated_at TEXT NOT NULL
);
CREATE INDEX task_checklist_items_task ON task_checklist_items (task_id, position);
```

Ticked is `done_at IS NOT NULL`, which carries who and when in the same row (gate 5.03) and needs no
separate flag. Removed is `removed_at IS NOT NULL`, the same soft-delete the rest of the app uses.
`position` is unique per task among non-removed rows, maintained by the move code the way
`tasks.position` already is — not by a constraint.

### 4.2 Endpoints

All under the existing tasks section (SPEC B2 — sections stay independent), all POST, all answering
either a redirect back (no JavaScript) or a fragment (with it), the way `details_handlers.go`
already does for the task window:

| Endpoint | Does |
|---|---|
| `POST /tasks/{id}/steps` | add a step at the end |
| `POST /tasks/{id}/steps/{stepID}/tick` | tick or untick (the form carries which) |
| `POST /tasks/{id}/steps/{stepID}` | rename |
| `POST /tasks/{id}/steps/{stepID}/remove` | remove |
| `POST /tasks/{id}/steps/{stepID}/restore` | the Undo |
| `POST /tasks/{id}/steps/{stepID}/move` | up or down |

Every one of them bumps the tasks change counter (`bumpTasksVersion`, `internal/tasks/version.go`)
inside its own transaction, which is what makes gate 5.12's other-computer refresh work with no new
machinery.

### 4.3 Where it appears on screen

One template partial, `web/templates/tasks/steps.html`, included by **both** the task window
(`window-task.html`) and the task page (`details.html`). One partial, two homes — that is what keeps
gate 5.09 honest rather than aspirational, and it means the no-JavaScript path is the same code, not
a second implementation.

The list is built from the mockup's existing parts (`.card`, `.btn`, `.mini`, the icon buttons of
gate 4.11), so it looks as though it had always been there. Like the task window itself, there is no
mockup drawing for it — owner check O5.1 covers that.

### 4.4 The icon

The icon is drawn as SVG first, then rendered to every size at build time by a small tool under
`tools/`, so there is one source of truth and no hand-exported PNGs drifting out of step. The
placeholder `icon-192.png` and `icon-512.png` in `web/static/theme/icons/` — currently plain orange
squares — are replaced.

`web/templates/app/layout.html` gains the icon links it has never had, which is the actual fix
behind gates 5.21–5.22.

### 4.5 What the Setup page tells people

Three routes, in this order, each independently workable:

1. **Edge or Chrome, from the menu.** The browser's own "install this site as an app" — Edge's
   **Apps → Install this site as an app**, Chrome's **⋮ → Save and share → Install page as app**.
   Both work on a plain `http://` address, both create a Start-menu entry and a desktop icon, and
   both take the icon from the page — which is why 5.21–5.22 have to land first. This is the route
   the page leads with.
2. **The command shortcut** (what the page does today, kept for Brave and as a fallback): the
   `--app=` command, its Copy button, and New → Shortcut.
3. **If the icon comes out blank:** a **Download icon** button for `comphq.ico`, then right-click
   the shortcut → Properties → Change Icon → Browse.

The exact menu wording in routes 1 and 2 is **checked against the browsers actually on the office
computers during P5-08 and written down from what is seen there**, not from this plan — browser
menus move, and the README is only useful if it matches the screen in front of the owner.

---

## 5. Milestones and tasks

The task format, the definition of done and the progress log are `PLAN.md` §2.2 and §2.4, unchanged.

### Which task proves which gate

| Gates | Task | Proved by |
|---|---|---|
| Storage for 5.01–5.07, 5.13 | P5-01 | `internal/tasks` unit tests; a migration test |
| 5.08, 5.12, 5.13 | P5-02 | Go integration tests per route; E2E with JavaScript disabled |
| 5.01–5.07, 5.09, 5.16 | P5-03 | `tasks/steps.spec.ts` in the window and on the page; axe with 50 steps |
| 5.10, 5.11, 5.14, 5.17 | P5-04 | `tasks/steps-elsewhere.spec.ts`; an export test; `speed/` with a 50-step job |
| 5.20, 5.21 | P5-05 | Go test that every declared size exists at its declared dimensions |
| 5.22, 5.23, 5.24 | P5-06 | E2E that every icon address answers with an image; a manifest test |
| 5.25 | P5-07 | `BuildSetupCommands` unit tests; an E2E for the icon download; axe |
| 5.26, 5.30, 5.31 | P5-08 | Full suite at every layer; std checks; the real-browser walk-through |
| O5.1–O5.4 | P5-09 (S10) | Owner reply, logged in `PROGRESS.md` |

### P5-00 Spec edits for v1.2 — **done, 2026-09-20**
- **Goal:** `SPEC.md` describes v1.2 before anything is built against it.
- **Depended on:** the owner confirming §1's decisions D-70–D-74.
- **Touched:** `SPEC.md` (A11 Phase 5, A12, B3, B4, B9.10, new B10 and B11), `docs/decisions.md`
  (D-70–D-74), `PLAN.md` (§5.1 stop point S10, §6 table, P4-15's closing line).
- **Note:** not a build-agent task — drafted by the owner's spec model, as with every spec change.

---

## Milestone 5.1 — Steps inside a job

### P5-01 Storage and the store functions
- **Goal:** steps exist in the database, with every operation the screens will need.
- **Gates:** groundwork for 5.01–5.07, 5.13.
- **Depends on:** P5-00
- **Touches:** `migrations/tasks/0002_checklist.sql`, `internal/tasks/checklist.go` (new),
  `internal/tasks/checklist_test.go` (new).
- **Behaviour:** add, tick, untick, rename, remove, restore, move up, move down, list for a task.
  Every write bumps the change counter in the same transaction (§4.2). Limits of 50 steps and 200
  characters are enforced here, in the store, so no route can get round them. Position stays unique
  among non-removed rows of a task; restoring a step puts it back at its old position, and pushes
  the others down if that position is taken.
- **Tests:** Go unit tests for every operation, both limits, restore into a taken position, and
  ordering after a sequence of moves and removals. A migration test that the table exists and that
  the 1.1.0 schema is still readable.
- **Done when:** tests are green in CI.
- **Decisions:** D-70, D-74.

### P5-02 Endpoints, and the no-JavaScript path first
- **Goal:** every step operation works as a plain form, before any script exists.
- **Gates:** 5.08, 5.12, 5.13.
- **Depends on:** P5-01
- **Touches:** `internal/tasks/checklist_handlers.go` (new), `internal/tasks/section.go`,
  `web/templates/tasks/steps.html` (new), `web/templates/tasks/details.html`.
- **Behaviour:** the six endpoints of §4.2, each answering a redirect back to the task page. Acting
  on a step another person removed says so in plain words rather than failing (5.12). Ticking a
  step already ticked is not an error — it stays ticked, with the first person's name.
  **Building this before the script is deliberate:** it is the same order P4-08 used, and it is what
  makes gate 5.08 a fact rather than a retrofit.
- **Tests:** Go integration tests per endpoint; E2E with JavaScript disabled covering add, tick,
  rename, remove, undo and reorder from the task page; the two conflict cases of 5.12.
- **Done when:** tests are green in CI.

### P5-03 The list on screen, in the window and on the page
- **Goal:** what people actually see and use.
- **Gates:** 5.01, 5.02, 5.03, 5.04, 5.05, 5.06, 5.07, 5.09, 5.16.
- **Depends on:** P5-02
- **Mockup:** not drawn — built from the mockup's existing parts (`.card`, `.btn`, `.mini`, the
  gate-4.11 icon buttons), as the task window was. Owner check **O5.1** covers it.
- **Touches:** `web/templates/tasks/steps.html`, `web/templates/tasks/window-task.html`,
  `web/static/tasks/steps.js` (new), `web/static/tasks/board.css` or a new `steps.css`,
  `web/static/theme/theme.css`.
- **Behaviour:** the same partial in the window and on the page (§4.3). Enter adds and leaves the
  box ready (5.02). Ticking is immediate, without a page reload, and the count and bar update with
  it. Renaming is in place. Removing shows **Undo** beside the heading until the next action or
  about ten seconds. Reordering uses the up and down icons and the already-vendored drag library.
  Nothing here is required: with the script gone, P5-02's forms are what's left, and they work.
- **Tests:** E2E for each of 5.01–5.07 in the window and again on the page; an E2E that the page
  still answers after a refresh mid-edit (5.09); axe with a 50-step list open and with a step being
  renamed; the style rules of gate 4.05 and 4.11 applied to the new controls.
- **Done when:** tests are green in CI.

### P5-04 The badge, History, export and speed
- **Goal:** steps show up everywhere else they should, and nowhere they shouldn't.
- **Gates:** 5.10, 5.11, 5.14, 5.17, and 5.13's messages.
- **Depends on:** P5-03
- **Touches:** `internal/tasks/export.go`, `internal/settings/export.go`,
  `internal/tasks/checklist.go`, `web/templates/tasks/board.html`, `myjobs.html`, `team.html`,
  `e2e/speed/`.
- **Behaviour:** the **3/7** badge on a card, absent when there are no steps, a tick when all are
  done (5.10). History rows for add, rename, remove and restore, and none for ticks (D-71) — proved
  by a test that ticks ten steps and asserts History is unchanged. `steps.csv` joins `tasks.csv` and
  `events.csv` in the export (5.14). The Briefing is explicitly untouched (D-73), with a test that
  asserts a job with unfinished steps briefs exactly as it does today.
- **Tests:** E2E for the badge on all three screens; the History test above; an export test opening
  `steps.csv` and checking its columns and a removed step; the speed suite with a 50-step job.
- **Done when:** tests are green in CI.

---

## Milestone 5.2 — Looking like an app

### P5-05 The Comp HQ icon
- **Goal:** Comp HQ has a mark of its own, at every size, from one source.
- **Gates:** 5.20, 5.21.
- **Depends on:** P5-00 (independent of Milestone 5.1)
- **Touches:** `web/static/theme/icons/comphq.svg` (new, the source), `tools/icons/` (new renderer),
  `web/static/theme/icons/` (generated `.ico` and PNGs), `assets.go`.
- **Behaviour:** the stacked **COMP / HQ** wordmark of gate 4.13, navy field, orange HQ, rendered
  from the SVG at build time. At 16 and 32 pixels the wordmark is unreadable, so those sizes drop to
  the **HQ** alone — checked by eye in P5-08's screenshots and by a test that every declared size
  exists and is the size it claims. The maskable 512 keeps the mark inside the safe circle so
  Windows and Android can crop it without cutting letters.
- **Tests:** Go test that every file in the manifest and in `layout.html` exists, is a real image and
  has the declared dimensions; a repository test that no icon is fetched from the internet (the
  existing gate 4.12 rule, extended to the new files).
- **Done when:** tests are green in CI.

### P5-06 Linking it, and the manifest
- **Goal:** the browser and Windows can both find the icon.
- **Gates:** 5.22, 5.23, 5.24.
- **Depends on:** P5-05
- **Touches:** `web/templates/app/layout.html`, `web/static/app/manifest.webmanifest`,
  `internal/app/server.go` (a `/favicon.ico` route, because some browsers ask for that address and
  nothing else).
- **Behaviour:** the head gains the `.ico`, the PNG sizes and the apple touch icon; the manifest
  gains the full icon list, the maskable entry, the Briefing as its start address, standalone
  display and the navy theme colour. **The manifest's `start_url` is `/`, which is the Briefing
  (SPEC A8)** — an installed app should open where the app opens, not on a settings page.
- **Tests:** E2E that every icon address in the head and the manifest answers 200 with an image
  type; a test that the manifest parses and carries name, start address, display and theme colour;
  an E2E in a browser context that reports the page's resolved icon.
- **Done when:** tests are green in CI.

### P5-07 Set up this computer, rewritten
- **Goal:** one page that gets a non-technical person from a browser to an app icon.
- **Gates:** 5.25.
- **Depends on:** P5-06
- **Touches:** `internal/settings/setup.go`, `internal/settings/setup_test.go`,
  `web/templates/settings/setup.html`, `web/static/settings/copy.js`.
- **Behaviour:** the three routes of §4.5, in that order, each headed by the browser it's for. The
  existing `BuildSetupCommands` keeps building the `--app=` commands from the request's host and
  keeps its Copy buttons; Edge joins Chrome and Brave in that list. A **Download icon** button
  serves `comphq.ico` as an attachment named `Comp HQ.ico`, so the Change Icon dialog shows a
  sensible name. No route tells the reader to do anything they can't see on their own screen.
- **Tests:** `BuildSetupCommands` unit tests extended to Edge; an E2E that the download button
  serves a real `.ico` with a download filename; axe and the style rules on the rewritten page.
- **Done when:** tests are green in CI.

---

## Milestone 5.3 — Finish

### P5-08 Full sweep: browser check, screenshots, README
- **Goal:** prove nothing regressed, and give the owner what they need to check it.
- **Gates:** 5.26, 5.30, 5.31.
- **Depends on:** P5-04, P5-07
- **Touches:** `e2e/`, `reports/screens/`, `README.md`, `CHANGES.md`.
- **Behaviour:**
  - Every gate 1.01–4.54 re-run unchanged, at the same thresholds. A weakened, skipped or deleted
    earlier test fails the phase (`PLAN.md` §2.5, SPEC B8 rule 5).
  - **The one thing CI cannot prove:** that route 1 of the Setup page matches the menus in Edge,
    Chrome and Brave as they stand on the office computers. Walk it through on a real Windows
    machine, write down the menu wording that is actually there, and correct the page and README to
    match. Note in `PROGRESS.md` which browser versions were checked and on what date, so the next
    person knows how stale the wording is.
  - Refresh every screenshot in `reports/screens/`, adding one of a job with steps open in the
    window, one of the Board with a **3/7** badge, and one of the installed app window with its icon
    and no address bar (gate 5.26).
  - README: "Setting up an office computer" rewritten around the new page; a new "Steps inside a
    job" section under Using Tasks; the hands-on table updated.
- **Tests:** the full suite at every layer, including accessibility and window sizes at 1024 × 700
  and full HD, and the speed suite with the test library.
- **Done when:** every gate 1.01–5.31 passes in one full CI run.

### P5-09 Release v1.2.0 and the Phase 5 report — **STOP S10**
- **Goal:** publish v1.2.0 and report.
- **Gates:** O5.1, O5.2, O5.3, O5.4 (owner); final confirmation of every gate.
- **Depends on:** P5-08
- **Steps:** follow the release procedure in `PLAN.md` §5.2 with version `1.2.0`, upgrading from and
  rolling back to `v1.1.0` and confirming gate 5.15 with real steps in the database. Then send
  **S10**.
- **In the report, say plainly:** that steps are the one part with no mockup to check against, so
  O5.1 is the check that matters most; and that the app-icon routes were confirmed on real office
  browsers on a named date, with the wording that was seen.
- **Done when:** the release exists, the report is written, S10 is sent, and the owner has confirmed
  O5.1–O5.4.

---

## 6. The stop point

Added to `PLAN.md` §5.1, after S9:

> **S10 — end of Phase 5 (v1.2.0).** "Phase 5 is done and v1.2.0 is published. Jobs now have steps
> you can tick off, and Comp HQ has its own icon and installs as an app. To update, change the
> version number in TrueNAS to `1.2.0` — see Updating in the README. There are four things to check
> yourself: *O5.1–O5.4, each with what to try and what to expect.* The full report is in
> `reports/phase-5-report.md`."

---

## 7. Not in v1.2

Named here so nobody has to wonder:

- Several named lists on one job (D-70), and reusable step templates (D-72).
- Comments or chat on a job, and sub-tasks that are jobs in their own right — still out (SPEC A12).
- Steps in the Saturday Briefing (D-73), and a "Removed steps" screen (D-74).
- Assigning a step to a person, or giving a step its own due date. A step is a tick-box, not a
  small job; if it needs an owner and a date, it is a job.
- The padlock (`https://`), a friendly address like `comphq.local`, and the offline screen that
  depends on the padlock (§1). Revisit only if a browser starts showing the address strip.
- Phone and tablet layouts, and a dark theme — unchanged from SPEC A12.
