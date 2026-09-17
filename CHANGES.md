# Comp HQ — Changes

## 0.1.0

The first real release — Phase 1 is complete. Comp HQ now has a full **Knowledge Base**: categories, a rich-text editor (headings, lists, links, tables, pictures), publishing with full version history and one-click restore, archiving, and instant search with highlighted matches. Pictures can be pasted, dropped, or copied in from a web page, and whole Word documents can be imported directly into the editor — tables, lists, pictures and all, with anything that can't come across clearly marked so nothing silently goes missing. Everyone on your network picks their name once per computer; no logins, no passwords. Settings now has **Export everything** (a one-file backup you can download any time), automatic nightly backups with a warning banner if one is ever overdue, a **Content check** page listing anything that needs a second look, and a **Set up this computer** page for a proper desktop shortcut. The README covers installing on TrueNAS, updating, backing up and restoring, and moving your existing HelpScout articles over.

## 0.0.2 (internal)

Second internal build. Adds real Knowledge Base content (categories, articles, edits, pasted images) for the first time, so the upgrade/rollback test (gate 1.42) has real data to carry across an upgrade, not just an empty database. This version is for testing only — nothing to install yet.

## 0.0.1 (internal)

First internal build. Proves the deployment path end to end: the app builds into a container image, TrueNAS can run it from `deploy/truenas.yaml`, and it restarts, recreates and updates without losing data. This version is for testing only — nothing to install yet.
