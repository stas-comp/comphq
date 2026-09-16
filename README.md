# Comp HQ

Comp HQ is a small, private staff hub: a Knowledge Base, a task Board, a Team view, a Calendar, and a daily Briefing — self-hosted on your own TrueNAS, with no logins and no outside services.

This README is written for you, the owner — plain language, no coding knowledge needed. It's updated as the app is built; some sections below are still placeholders until later phases.

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
5. **Check it started properly:** in **Apps**, Comp HQ should show as **Healthy** within about a minute. If it doesn't, see "It won't open — what now?" (added in a later update to this README).
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

If the release notes for that update said it wasn't safe to simply undo this way, you'll also be told to restore the snapshot taken just before the update — see "Restoring" (added in a later update to this README).

## Backing up

*(Added once the export feature is built — Phase 1, later task.)*

## Restoring

*(Added once the export feature is built — Phase 1, later task.)*

## It won't open — what now?

*(Added once this is fully covered — Phase 1, later task.)*
