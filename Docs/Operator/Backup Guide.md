# Backup Guide

This guide explains what Victory data actually is, how to back it up, and how to restore it. Nothing here is aspirational — every mechanism described is real and has been demonstrated against a live install.

---

## 1. What counts as Victory data

Two things, and only two things, need to survive a disaster:

1. **The PostgreSQL database** — every Show, Character, message, Storyboard, roster, and account. This is the vast majority of what matters.
2. **Uploaded assets on disk** — images, tokens, and other files an Operator or Producer uploaded, stored under Victory's configured storage directory (not inside the database itself).

Everything else — the application binaries, the `.env`/configuration, Caddy/reverse-proxy config — is either replaceable from the install itself or small enough to note down once and keep with your backups. A generated `.env` should be treated as data worth keeping too, since it holds the install's actual secrets (database password, session signing key); losing it without a copy means generating new ones and re-authenticating everyone.

---

## 2. Automatic backups (if you set them up)

Victory ships a real, working automated backup system (`scripts/backup/backup.sh` and `scripts/backup/retention.sh`), built and proven in Kernel 77. **It is not active by default on a fresh install** — it requires you to point it at your own off-host storage before it does anything.

What it does once configured:

| Timer | Schedule | Contents |
|---|---|---|
| Quick backup | every 15 minutes | database dump only |
| Full backup | daily | database dump + uploaded assets + a manifest with checksums |
| Retention cleanup | weekly | deletes remote archives past your configured retention window |

Backups are encrypted before they leave your machine and uploaded to a remote you control (the reference setup uses `rclone` against a cloud storage provider, but any `rclone`-supported remote works). Victory's own application code never holds your cloud storage credentials — only the backup scripts' local `rclone` configuration does.

**Setting this up requires server/terminal access** — it's an Operator-level task, not something available from the Victory UI today. See `scripts/backup/` in the repository and its accompanying runbook for the exact systemd timer setup. If you're not comfortable with this, the manual path below is always available and requires nothing pre-configured.

Once running, you can check backup health from inside Victory itself: an authenticated Operator can query the backup-status endpoint for the timestamp, mode, and outcome of the last run — no filenames or restore controls are exposed there, just enough to confirm it's working.

---

## 3. Manual backup (works on any install, right now)

If you haven't set up automated backups, or just want an on-demand copy before a risky change:

```bash
# Database
pg_dump "$DATABASE_URL" > victory-backup-$(date +%Y%m%d-%H%M%S).sql

# Uploaded assets — copy your configured storage directory
cp -r /path/to/victory/storage victory-assets-backup-$(date +%Y%m%d)
```

Store both somewhere off the machine running Victory — a backup that lives only on the same disk as the thing it's backing up doesn't protect you from a disk failure.

---

## 4. Verifying a backup is good

A backup file existing is not the same as a backup file being restorable. At minimum:

- Confirm the `.sql` dump is non-empty and starts with a valid `pg_dump` header (`head` the file).
- Periodically — not every time — actually restore a dump into a scratch database and confirm Victory can start against it. This is the only way to be certain a backup is real, and it's the same method used to prove the automated system's restore path.

---

## 5. Restoring

```bash
# Create a fresh database, then:
psql "$DATABASE_URL" < victory-backup-20260101-030000.sql

# Restore assets
cp -r victory-assets-backup-20260101/* /path/to/victory/storage/
```

Restart Victory afterward so it picks up the restored state. A full restore (database + assets) from the automated backup system has been measured end-to-end at roughly 69 seconds on the reference deployment — restoring by hand will take longer depending on dump size and how quickly you can provision a target database, but the mechanism itself is identical.

---

## 6. What happens during updates

An update replaces Victory's application files; it does not touch your database or asset storage. On the Windows consumer build specifically, application files live in a directory the updater manages and fully replaces, while all persistent data (database, generated secrets, storage, backups, exports, logs) lives in a separate location the updater never writes to. This separation is deliberate — it means an update cannot corrupt or delete your data even if the update itself fails partway through.

**Not yet independently verified:** a representative-data survival check across a real version-to-version update (i.e., proving specific Show/Character data is byte-identical before and after an update) hasn't been performed as its own dedicated test. The install/data separation this relies on has been verified by inspection and by the update mechanism succeeding repeatedly in practice, but the stronger end-to-end data-integrity proof is still an open item.

---

## 7. What happens on uninstall

Uninstalling Victory removes only the application files. Your database, assets, and configuration are left in place by construction — the uninstaller has no code path that reaches your data directory at all, so this isn't a "please don't delete my data" setting, it's structurally not possible for ordinary uninstall to do it.

If you genuinely want to delete everything — not just uninstall the application, but erase the lot itself — the Windows build has a separate, explicit "Delete Victory Data" action (not the uninstaller) that requires two confirmations, including typing the lot's own name back, before it stops Victory and removes the data directory. This exists specifically so a full wipe can happen safely rather than as a risky manual delete while Victory might still be running. As of this writing, this specific action has been built but not yet exercised end-to-end by an Operator.

---

## 8. Gaps, honestly

- Automated off-host backup is real and proven, but requires manual Operator setup (systemd timers + your own cloud remote) — there is no in-product "turn on backups" button yet.
- The A→B update data-survival proof described in §6 is an open item, not a completed proof.
- The "Delete Victory Data" safety action exists in code but has not yet been exercised by an Operator outside of development.
