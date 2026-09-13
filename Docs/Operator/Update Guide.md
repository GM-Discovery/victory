# Update Guide

---

## 1. How updates work

On Windows, Victory checks for updates automatically — every 4 hours ordinarily, or every 15 minutes once an update has already downloaded and is just waiting for a safe moment to apply. You can also check manually at any time from the tray icon's "Check for Updates."

Updates are fetched over HTTPS directly from Victory's own release feed. **Use the built-in updater, not Git**, to update a Windows installation — the updater manages stopping and restarting everything (including your database) safely; pulling code with Git directly does not.

---

## 2. Default update behavior and live Shows

Victory won't interrupt a live Show to update itself:

- A **frontend-only** update (no backend/database change) applies immediately — it causes no outage, since nothing running needs to restart.
- A **backend-changing** update waits for either a quiet 2:30 AM local-time window, or for every session on your installation to reach `closed` status — whichever comes first. If you're mid-Show when an update is ready, it simply waits.

If you need an update to apply sooner and a Show is holding it back, end the Show properly (via the `/showtime <code> end` command, which is the one path that actually marks a session closed) rather than looking for a way to force the update through while still live.

Every update-check outcome — including a silent "waiting for quiet window" decision — is written to the launcher's log, not just shown as a notification balloon, since Windows notification settings can suppress the balloon with no other record left behind.

---

## 3. Checking your current version

The Status screen shows your currently running version. Version numbers during this alpha period are sequential build identifiers rather than a traditional semantic version (like `1.2.0`) — that's expected for now, not a sign anything is wrong.

---

## 4. Release channels

There is currently one release channel. If Victory later adds separate stable/beta channels, this guide will be updated to describe how to choose between them.

---

## 5. Rollback / recovery behavior

There is no one-click rollback to a previous version today. If an update ever causes a real problem, your data itself is safe regardless — see the Backup Guide for how to restore your data independently of which Victory version is currently installed, and the Troubleshooting Guide for what to check first.

---

## 6. Linux/server installs

If you're running Victory via the Linux/server path rather than the Windows installer, there is no automatic updater — updating means pulling the new code and re-running your deployment process yourself, following your own deployment path's documented steps. The live-Show-aware, quiet-window update timing described above is specific to the Windows consumer installer.
