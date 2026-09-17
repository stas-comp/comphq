# Comp HQ — Changes

## 0.0.2 (internal)

Second internal build. Adds real Knowledge Base content (categories, articles, edits, pasted images) for the first time, so the upgrade/rollback test (gate 1.42) has real data to carry across an upgrade, not just an empty database. This version is for testing only — nothing to install yet.

## 0.0.1 (internal)

First internal build. Proves the deployment path end to end: the app builds into a container image, TrueNAS can run it from `deploy/truenas.yaml`, and it restarts, recreates and updates without losing data. This version is for testing only — nothing to install yet.
