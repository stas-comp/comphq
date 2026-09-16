# Handoff: Write the Staff HQ Build Plan

**To the agent reading this:** your job in this chat is to write the **build plan** for Staff HQ.
Don't write code, and don't start building. Save the plan as `PLAN.md` in this folder.

**Model:** the owner wants **Opus** to write direction documents like this one. **Sonnet** runs the plan
afterwards, on auto. If this chat is running on Sonnet, say so and suggest switching before you start.

---

## 1. What to read first

1. `SPEC.md`: the full specification. **Part A was approved by the owner on 14 September 2026.**
   - **Part A** (gates 1.01–1.52, 2.01–2.38, 3.01–3.13 and the 👤 Owner checks) is the contract.
     Don't change it or reopen it. If a gate turns out to be impossible or contradictory, stop and explain
     it to the owner in plain language, with a recommended fix.
   - **Part B** is technical direction. The plan may adjust it where that serves a gate better, as long as
     the change is recorded as a planned entry for `docs/decisions.md`.
2. `docs/intent/staff-hq.md`: the original confirmed intent, for background.
3. `HANDOFF - write the spec.md`: how the spec was framed, for background.

Earlier design conversations produced a clickable mockup. SPEC.md A5, A6 and A10 already describe
everything that mockup settled, so the plan doesn't depend on it.

## 2. Who the plan is for

- **The executor** is an automated Claude Sonnet agent running **on auto**, across many sessions, with its
  context reset between them. It must be able to open `PLAN.md` cold, find the next unfinished task, do it,
  prove it, record it, and move on, without asking anyone anything except at the stop points listed below.
- **The owner** has no coding experience and can't review code or pull requests. They read one short
  plain-language summary at the top of the plan and approve it. Everything else in the plan is for the agent.

## 3. Facts about the environment

Verify these, don't assume them:

- **Build computer:** Windows 11 Home. The project folder is `C:\Users\Knacker\Desktop\Staff HQ`.
  It is **not yet a git repository**. Both PowerShell and Git Bash are available.
  - Check which tools are installed: Go, Node, Git, `gh`, Docker.
  - Docker may not be available (Windows Home needs WSL2 for it). **Assume the container tests in B7 layer 4
    run only in GitHub Actions on Linux**, unless the check shows Docker working locally.
  - The plan's first task installs or confirms the local tooling. Installing standard developer tools
    (Go, Node LTS, Git, GitHub CLI) from their official sources is acceptable. List them for the owner in
    the summary.
- **GitHub:** the owner's existing account **`stas-comp`**. Put a private repository there. Images go to
  `ghcr.io/stas-comp/staffhq`.
  - The agent **never enters passwords or tokens**. When GitHub sign-in is needed (for example
    `gh auth login`), it stops and asks the owner to complete it, with click-by-click steps.
  - New ghcr packages start private. After the first image is pushed, the owner makes the package
    **Public** (SPEC B6). This is a stop point.
- **Target:** TrueNAS SCALE 25.04.2.6, where the owner installs the app with "Install via YAML" (SPEC B6).
  Office computers run Windows with Chrome or Brave.

## 4. Check current facts before fixing versions

The spec names technologies but deliberately leaves exact versions to the build. Before the plan pins
anything, check current official sources:

- The Go stable release, and `modernc.org/sqlite`, including confirming FTS5 is available in it.
- `bluemonday`.
- The rich-text editor. TipTap/ProseMirror is suggested: confirm the MIT licence and that the table extension
  exists. Then decide how its **one-time prebuilt bundle** gets made: a committed dev-only script (for example
  esbuild) that writes `web/static/vendor/`, with the version and checksum recorded. Nothing runs at app
  runtime or for the owner.
- SortableJS, if it's used.
- Playwright and `@axe-core/playwright`.
- The `gcr.io/distroless/static` digest.
- GitHub Actions for Go, Node, Docker buildx and pushing to ghcr (`GITHUB_TOKEN` with `packages: write`).
  Pin actions to commit SHAs.
- TrueNAS SCALE 25.04's "Install via YAML" behaviour: host-path volumes, `user`, `healthcheck`, and
  `read_only`.

Record what you found, with dates, in a short "Versions and sources" section of the plan. If something in
Part B is no longer true, put the adjustment in the plan and the planned decision-log entry.

## 5. Required structure of PLAN.md

### 5.1 For the owner (at the top, plain language, one screen at most)
- What will happen, in order, and roughly how many working sessions each phase takes.
- **Every moment the build will stop and wait for you**, taken from SPEC A13 and the stop points below.
- What you'll receive at the end of each phase (SPEC B8 rule 6).
- One question: "Is this plan OK to start?"

### 5.2 How the build agent works through this plan
Write these as fixed rules. Where there's a real choice, make it and give a one-line reason. Suggested
defaults are given.

- **Where it records progress:** a checklist in `PROGRESS.md`, with one line per task: status, commit, and
  the CI run link.
  - Starting a session: read `PLAN.md` rules, then `PROGRESS.md`, then the next task.
  - Ending a session: always leave `PROGRESS.md` accurate.
- **Git workflow.** Suggested: trunk-based on `main`, one commit or small group of commits per task, and
  push after each task. Pull requests aren't needed, because nobody reviews them. Before the next task
  starts, the task's CI run must be green.
  - Releases are tags `vX.Y.Z` taken only from a green commit.
  - Suggested versions: Phase 1 = `v0.1.0`, Phase 2 = `v0.2.0`, Phase 3 = `v1.0.0`, with patch releases for
    fixes.
- **Definition of done for a task:** its listed tests exist and pass locally and in CI, all earlier gates'
  tests still pass, `PROGRESS.md` is updated, and any `docs/decisions.md` entries are written.
- **When CI fails or a test seems wrong:** follow SPEC B8 rule 5. Never weaken, skip or delete a test to get
  green. Also define a retry limit: after N genuine attempts on the same failure, stop and write a
  plain-language note to the owner explaining the blocker.
- **Commands:** the exact local commands for unit, integration, E2E, speed and accessibility tests, and
  what only CI runs. The same commands must work from the PowerShell and Bash tools on Windows.
- **Stop points:** the only times the agent pauses (see 5.5).

### 5.3 Milestones and tasks, grouped by phase (1 → 2 → 3)

Every task has:
- **ID and title** (for example `P1-07 Article editor: toolbar and publish`).
- **Goal**, in one or two sentences.
- **Gates served**: gate numbers from SPEC A11. Plumbing tasks serve none; say so.
- **Depends on**: task IDs.
- **Touches**: packages, folders and files, matching SPEC B2.
- **Tests to add**, naming the B7 layer (unit, integration, E2E, container, speed, accessibility).
- **Done when**: specific, checkable conditions, such as commands to run and results to expect.
- **Size**: small enough for one Sonnet session. Split anything bigger.

Ordering principles:
1. **Walking skeleton first.** Early in Phase 1, prove the whole pipeline end to end with almost no
   features: repository, CI, Go binary with `healthcheck`, the Dockerfile, a ghcr push on a tag,
   `deploy/truenas.yaml` passing `docker compose config` and becoming healthy in CI, a frame page with
   `data-ready`, Playwright and axe running in CI. The deployment path must work before the features pile up.
2. **Front-load risk**, with small spikes that end in a recorded decision:
   - Bundling the editor (paste, drop, tables, image-upload hook).
   - `modernc.org/sqlite` FTS5 with `snippet()`/`highlight()` and block ids.
   - The `.docx` converter on realistic Word markup, including how the `sample.docx` fixture generator runs
     in CI.
   - The offline network test in Compose.
   - Playwright clipboard paste in CI.
3. **Vertical slices after that.** Each feature task delivers database, handler, page and tests together, in
   the section's own package (SPEC B2 boundaries).
4. **Expand-only migrations from day one** (SPEC B6). The **previous-version rollback test** can't run until a
   previous release exists. State exactly when it becomes mandatory (for example from `v0.1.1`, or by
   cutting a throwaway `v0.0.1` early) so gate 1.42 is actually proven before Phase 1 is reported done.
5. **Speed tests and the test-library seeder** come in early enough that slow designs are caught while
   they're still cheap to change.
6. **README tasks** (gate 1.44 and SPEC B8 rule 7) are scheduled, not left to the end. Include the
   click-by-click steps for every A13 moment.

### 5.4 Gate traceability table
A table listing **every** Part A gate (1.01–1.52, 2.01–2.38, 3.01–3.13) with the task or tasks that deliver
it and the test that proves it. Every gate must appear, and no gate is left without a task. Owner checks
(O1.1–O1.4, O2.1–O2.2, O3.1–O3.2) are mapped to the phase-end tasks.

### 5.5 Stop points and phase-end tasks
List every stop in order. At each one, the agent sends a plain-language message and waits:
- GitHub sign-in on the build computer.
- Making the ghcr package public, after the first image push.
- **Optional, during Phase 1:** a reminder that the owner can put 1–3 real Word documents in `samples\word`.
  Don't block on it.
- The end of each phase. The release task runs the full suite, tags, confirms the image was published, then
  writes the SPEC B8 rule 6 report, including screenshots from B7 layer 7, the version to install,
  click-by-click install or update steps, and the exact 👤 Owner checks. **The next phase only starts after
  the owner confirms the checks.** If an owner check fails, the plan should say how the feedback becomes
  fix tasks.
- Any Part A gate found impossible or contradictory, or any change that would add a paid service, a runtime
  internet dependency or a login (SPEC B8 rule 2).

Remember the owner's hands-on work in between phases: installing on TrueNAS and creating shortcuts (end of
Phase 1), moving the HelpScout articles over (O1.3), and entering events and Trello cards (O2.2). Phase 2
work must not assume that content already exists.

### 5.6 Risks
Keep it short: each risk, how the plan reduces it, and the fallback. At minimum cover the editor bundle,
Word import fidelity, CI flakiness in timing-based tests (the 60s refresh gates should use a fake clock or
focus events where the spec allows), and ghcr/TrueNAS image pulls.

## 6. Rules for writing the plan

- **Don't write code.** Short illustrative commands in "Done when" are fine.
- **Don't add features** that aren't in SPEC.md, and don't drop any.
- Keep the plan self-contained enough that a fresh Sonnet session can run any task by reading only
  `PLAN.md`, `PROGRESS.md` and the SPEC sections the task names.
- Use plain language in 5.1 and in every message the agent will send to the owner. Technical language is
  fine everywhere else.
- If you find a real problem in SPEC.md (a contradiction, an impossibility, or a Part B detail that's now
  wrong), don't silently work around it. Part B issues are fixed in the plan and logged. Part A issues go to
  the owner, in plain language, with a recommendation.

## 7. When the plan is done

1. Also create `PROGRESS.md`, containing just the empty checklist generated from the plan's task list.
2. Walk the owner through section 5.1 in plain language, and get an explicit yes.
3. Then give the owner **the exact message to paste into a new chat** to start the build with Sonnet on
   auto. For example: "Read PLAN.md and PROGRESS.md in this folder and carry out the build, starting from the
   next unfinished task. Follow the plan's rules and stop points." Also tell them how to restart it after an
   interruption: the same message works, because progress is tracked in `PROGRESS.md`.
