# Comp HQ — Phase 1 report

**Version 0.1.0 is ready.**

## Summary

- **What's new:** Comp HQ now has a complete Knowledge Base — categories, a rich-text editor with tables and pictures, full version history, archiving, instant search, and Import from Word — plus the People picker, and a Settings area with Export everything, automatic nightly backups, and a Content check page.
- **Automatic checks:** all 52 Phase 1 gate promises (1.01–1.52) passed in one full test run, on every layer SPEC requires — a real browser, a real container on TrueNAS-style Compose, with the internet completely blocked, and an upgrade-then-rollback against the previous release. See `reports/gates-phase-1.md` for the full list, and `reports/phase-1/screens/` for a screenshot of every page type.
- **Tests I corrected:** see "Tests corrected" below — 8 items, all either a genuine bug this session's own tests caught before release, or a test that only worked by accident in one specific environment and needed hardening once the container-based full run actually exercised it.

## What I need from you

### 1. Install steps

Comp HQ has never been installed anywhere yet, so this is a fresh install, not an update. Follow the README's two sections, in order:

1. **"Installing on TrueNAS"** — confirm a fixed IP, create the `comphq` dataset, set up daily snapshots, install via YAML (use `deploy/truenas.yaml`, version `0.1.0`), and confirm it shows **Healthy**.
2. **"Setting up an office computer"** — open Comp HQ in Chrome or Brave, then create the desktop shortcut from **Settings → Set up this computer**.

Both are click-by-click in the README at the root of the Comp HQ project folder.

### 2. Owner checks (SPEC §A13 "👤 Owner checks — end of Phase 1")

| Check | What to try | What you should see |
|---|---|---|
| **O1.1** | Follow the README to install Comp HQ on the NAS, then open it from a desktop shortcut on an office computer. | It installs from the one settings file with only the pool path, time zone and (if needed) port changed; the shortcut opens Comp HQ in its own window, no browser tabs or address bar. |
| **O1.2** | Spend a few minutes in the Knowledge Base: browse categories, open the editor, publish something, and search for a word in it. | The frame, editor and search all feel easy to use; the deep navy frame with the signal-orange accent looks bold and distinctive, and that same orange shows up consistently (the current section in the sidebar, main buttons, "Publish"). |
| **O1.3** | Copy your HelpScout articles into Comp HQ, one at a time — see the README's "Moving your HelpScout articles". | Every article is in Comp HQ, ticked off against your own list, and **Settings → Content check** shows **0**. This one can take a while — reply "OK, start Phase 2" for the others if this is still in progress, and let me know separately once it's done. |
| **O1.4** | Import one of your own real Word documents that has pictures and a table (via **Import from Word** in the editor, or drag it onto the editor). | It comes across well enough to publish with little or no tidying — headings, lists, tables and pictures in place; anything that couldn't come across is clearly marked. |

Reply with **"all OK"**, or tell me in your own words what isn't right and I'll fix it and send a small update.

## Full gate checklist

See `reports/gates-phase-1.md` — every SPEC gate 1.01–1.52, its plain description, the actual test(s) that prove it, and the result from the full CI run: https://github.com/stas-comp/comphq/actions/runs/35250669689 (`go`, `browser`, `container`, `speed`, `screens` — all green).

## Tests corrected

Real bugs or test-only fixes found and fixed by this project's own tests during Phase 1, each recorded in full in `docs/decisions.md`:

1. **Picture-only paragraphs silently dropped** (D-31): a paragraph holding only an image had no text, so the "drop empty paragraphs" rule was quietly eating every picture. Fixed in the Word converter before it ever shipped.
2. **A stray run outside any paragraph could be silently lost** (fixture-generator bug found while building the converter, not a real editor path — fixed defensively regardless).
3. **A shape appearing in both a document's "modern" and "fallback" markup could be counted twice** (mc:AlternateContent, found by reasoning and confirmed with a dedicated test before it could ever manifest).
4. **A genuinely fresh installation would crash on its very first startup** (D-33): the pre-update backup ran before any migration and tried to write status into a table that didn't exist yet. Caught by this session's own integration test on the first try.
5. **The stale-backup banner would have shown on every page of every existing test** (D-33): fixed by seeding a fresh backup timestamp at startup in test mode, so only a test that deliberately asks for a stale one sees it.
6. **A backup-scheduler test was flaky on Linux** (found by real CI, not locally): it checked for a backup file's mere existence rather than the status record that follows it, so it could win a narrow timing race. Fixed to wait for the actual recorded status.
7. **Three End-to-end tests only worked because they happened to run against a private, loopback-addressed local server** (D-37): one needs a real internet connection to prove a *success* path and is now clearly marked as needing that; two others were rewritten to work identically whether there's a real network, no network at all, or a shared container — including "About shows the version" (which no longer assumes a private test's fixed value) and "Copy puts the exact command text on the clipboard" (which now also accepts the app's own already-correct fallback for when the Clipboard API isn't available — exactly what happens for a real owner too, since Comp HQ is reached over plain HTTP at a NAS address, not a secure "localhost" one).
8. **The container test's own data-integrity check broke the moment it checked a real uploaded picture** (D-37): the app correctly writes uploaded files so only the container itself can read them; the checking script needed to read them a different way, not the app.

None of these affect anything you've used or will use — they were all caught and fixed before this release, by the project's own automatic checks.

## New design decisions (`docs/decisions.md`)

37 entries, D-01 through D-37. In plain terms, they cover: naming and identifiers; how the database and file storage work (one connection, content-addressed images, embedded assets); how CI is organised to stay within GitHub's free minutes; the Word document converter's internal design (built from scratch, piece by piece, entirely from the Go standard library); how search, history, and the editor's paste/clean-up rules work; how the nightly backup and the "Export everything" download are built to never slow down or break an ordinary page load; and several small but important test-design choices (like never asserting a sitewide count in a test that shares a server with the rest of the test suite). None of them change anything described in `SPEC.md` — they're all about *how* Comp HQ was built to meet that spec, not what it does.

## Screenshots

Every page type, captured at 1366×768, in `reports/phase-1/screens/`: `who.png`, `briefing.png`, `kb.png`, `kb-categories.png`, `kb-new-article.png`, `kb-archived.png`, `kb-search.png`, `tasks.png`, `calendar.png`, `settings-people.png`, `settings-backups.png`, `settings-content-check.png`, `settings-about.png`, `settings-setup.png`, `404.png`.

## Version

**0.1.0** — `ghcr.io/stas-comp/comphq:0.1.0`.
