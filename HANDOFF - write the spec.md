# Handoff: Write the Staff HQ Spec

**To the agent reading this:** your job in this chat is to write the **specification** for Staff HQ.
Do not write code. Do not write the task-by-task build plan. That comes in a later chat.

Start by reading `docs/intent/staff-hq.md`. It is the confirmed statement of what the owner wants,
agreed through a full interview. Treat it as settled. Don't reopen anything in it unless you find a
real contradiction or an impossibility, and if you do, explain it in plain language.

Save the spec as `SPEC.md` in this folder.

---

## 1. Who you are writing for

The owner **has no coding experience and cannot review code.** This shapes everything:

- They will approve the spec by reading it, so the direction must be understandable to them.
- The build will later be done by an automated coding agent (Claude Sonnet) running **on auto**.
  That agent must not keep stopping to ask the owner to run checks or answer technical questions.
- So the spec has to do two jobs at once:
  1. Tell the owner, in plain language, **what will exist and how they'll know it works.**
  2. Give the coding agent **fixed goals and checks it can verify itself**, and leave it free to
     solve the technical details however it likes, as long as it meets them.

## 2. Required structure of SPEC.md

Split the spec into two clearly labelled parts.

### Part A: Direction and Gates (for the owner, plain language)

No jargon. If a technical word is unavoidable, explain it in one short phrase the first time.
The owner should be able to read this part and say "yes, that's what I want" or "no, change X."

Include:

- **What we're building**: a short summary (reuse the intent doc).
- **Design priorities, in the owner's order** (see section 3 below), and how each one shows up in real decisions.
- **Each section of the app** (Briefing, Knowledge Base, Tasks, Calendar, plus the shared app frame and name picker):
  - What a staff member can do, written as short statements like "A staff member can…".
  - What it looks and feels like.
- **Gates**: numbered, checkable statements, grouped by build phase. Each gate must be:
  - **Observable**: describes something a person could see happen, not code internals.
    ✅ "Searching 'toner' shows the printer article with the word 'toner' highlighted, and clicking it
    scrolls to that paragraph."
    ❌ "The search index uses full-text search."
  - **Pass/fail**: there's no "feels fast" or "looks nice" without a concrete meaning.
  - **Verifiable by the coding agent on its own** (automated tests, driving a real browser, etc.).
    Mark the few gates that truly need a human (e.g. "the owner agrees the theme looks good") as
    **👤 Owner check**, and keep them to a handful, bundled at the end of each phase.
- **What's not included** (reuse the out-of-scope list).
- **The owner's hands-on moments**: the full, short list of times the owner must physically do something
  (e.g. installing the app on TrueNAS, creating desktop shortcuts, the phase-end look-and-feel check).
  Keep it as short as possible, and promise click-by-click instructions for each.

### Part B: Technical Direction (for the coding agent)

The owner doesn't need to read this part. Include:

- Chosen technology and **why it serves Stability first** (see section 4 for starting recommendations).
- Data model, the way data is stored on the NAS, and backup/restore approach.
- How it's deployed to TrueNAS SCALE 25.04 and how updates and rollbacks work.
- How each Part A gate is verified automatically (the test approach), so the build agent has a clear "done".
- **Rules for the build agent** (see section 5).

## 3. The owner's design priorities (their words, their order)

1. **Ease of use**: an intuitive, attractive interface.
2. **Stability**: *prioritise simple code that works all the time* over fancy features. No constant
   troubleshooting, no confusing pages, and no breakage when systems (TrueNAS, browsers, Windows) update.
3. **Flexibility**: it must be easy to adjust things and add features a few months in, once real use
   shows what's needed.
4. **Design**: it should look attractive, themed, and distinctive, "not just a generic blob". It matters
   to the owner, but **less than the functional side.**

When priorities conflict, the spec should say which one wins. The owner's intent is that stability and
function win over polish, but simple does not have to mean ugly.

## 4. Starting technical recommendations (verify, then decide)

These came out of the interview. They are sensible defaults, not decisions. Confirm them against current
documentation before committing.

- **Hosting:** TrueNAS SCALE 25.04 runs apps as Docker containers, and custom apps can be installed from a
  Docker Compose file ("Install via YAML" / custom app). Aim for **one container**, with its data kept in a
  normal TrueNAS dataset folder, so the NAS's snapshots and backups protect it. It must start automatically
  after the NAS reboots, and keep working after TrueNAS updates.
- **Boring, well-supported technology** with few dependencies and **pinned versions**. Avoid anything
  that needs a separate database server, a build step the owner has to run, or internet access at runtime.
- **Search:** a single-file database with built-in full-text search (e.g. SQLite FTS5) can provide ranked
  results, highlighted snippets, and immediate indexing on save, with no extra service. Clicking a
  result must land on and highlight the matching passage in the article.
- **App-like window:** make the site installable (a web app manifest) so Chrome and Edge on Windows can
  create a desktop shortcut that opens in its own window. Check what the office computers use.
- **Images:** pasted and imported images must be **stored on the NAS**. No image should still point
  to HelpScout's servers after the move.
- **Changes to how data is stored** must happen automatically and safely when the app updates, without
  losing data.

## 5. Rules for the build agent (put these in Part B)

- Meet the gates. Beyond that, solve technical problems independently. Don't ask the owner technical questions.
- Stop and ask the owner **only** for: a Part A gate that seems impossible or contradictory, a change
  that would add a paid service, internet dependency, or login, or a 👤 Owner check.
- Prefer the simplest approach that passes the gates. Don't add features that aren't in the spec.
- Each app section should be self-contained, so adding or changing one later doesn't break the others (Flexibility).
- Never report something as done without the automated check for that gate passing.
- Leave a plain-language `README` for the owner: how to open it, update it, back it up, and restore it.

## 6. Open questions to raise with the owner (one at a time, with your best guess)

These were **not** settled in the interview. Ask them in plain language, each with a recommended answer.
Don't invent answers silently.

1. **Backup and "never lose our articles again":** is relying on TrueNAS snapshots enough, or do they
   also want an in-app "export everything" (e.g. articles as readable documents) as a second safety net?
   *Suggested guess: yes, include a simple export. Losing articles is the original fear.*
2. **Theme and look:** any colours, logo, or feel they want (e.g. the organisation's colours, warm or
   calm)? Offer 2–3 concrete theme directions to choose from.
3. **Name list:** who adds and removes staff names from the "who are you?" picker, and how?
4. **Tasks shape:** what the stages/categories are (e.g. Ideas → To do → In progress → Done, or their
   current Trello columns), and whether finished tasks stay visible or get tucked away.
5. **Office computers:** Windows? Which browser (Chrome or Edge)? This affects the desktop-shortcut setup.
6. **Address:** is a friendly address (e.g. `http://staffhq.local`) wanted, or is the NAS's number
   address fine if the shortcut hides it?
7. **HelpScout move:** confirm a manual copy-paste move is fine (fewer than 20 articles), with the gate
   being "all articles present, images stored locally."

## 7. Background facts

- TrueNAS: **SCALE 25.04.2.6 ("Fangtooth")**, already running on the office network.
- Current tools being replaced: HelpScout Docs (free plan) and Trello (free plan).
- Usage pattern: volunteer staff all work in the office **on Saturdays**. That's why the opening screen is a
  Saturday briefing of "due before next Saturday".
- The team is small and trusted. Anyone can edit anything and assign anyone, and nobody needs gatekeeping.

## 8. When the spec is done

- Walk the owner through Part A in plain language and get an explicit yes.
- Then suggest the next step: a separate chat to turn the spec into a build plan that Sonnet can run on auto.
