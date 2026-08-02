# Victory Backup Runbook

**Purpose.** Operate the Kernel 77 encrypted off-host backup system: what runs, on what
schedule, how to check it, and how to run it by hand.

**Scripts.** `scripts/backup/backup.sh`, `scripts/backup/retention.sh`.
**Destination.** A dedicated Google Drive account (`victoryvtt.ops@gmail.com`), reached through
two chained `rclone` remotes: `victoryvtt-drive` (Drive) and `victoryvtt-crypt` (encrypting
`crypt` remote layered over it, pointed at `VictoryBackups/`). Victory's application code has
no Drive credentials; only the host's `rclone` config does.

This document contains no secrets. The crypt password, Drive OAuth token, and everything else
that must survive independently of this host are listed (not stored) in
`victory-backup-secret-recovery.md`.

---

## 1. What runs, and when

| Timer | Schedule | Script | Contents |
|---|---|---|---|
| `victory-backup-quick.timer` | every 15 minutes | `backup.sh quick` | PostgreSQL dump only |
| `victory-backup-full.timer` | daily, 03:00 UTC | `backup.sh full` | PostgreSQL dump + uploaded assets + manifest + checksums |
| `victory-backup-retention.timer` | weekly | `retention.sh --apply` | deletes remote archives past their retention window |

All three are systemd timers, installed from `scripts/backup/systemd/*.service`/`*.timer` into
`/etc/systemd/system/`, `daemon-reload`'d, and `enable --now`'d. `Persistent=true` on each timer
means a missed run (host was down) fires as soon as the host is back, rather than waiting for
the next scheduled slot.

Retention targets (§9.6 of the kernel spec): quick backups 48 hours, full backups 30 days.
`retention.sh` reads these from `RETAIN_QUICK_HOURS`/`RETAIN_FULL_DAYS` env if you need to
override them; defaults live in the script.

---

## 2. Check it's working

```bash
systemctl list-timers 'victory-backup-*' --no-pager
cat /opt/victory/backups/backup-status.json
journalctl -u victory-backup-quick.service -n 20 --no-pager
journalctl -u victory-backup-full.service -n 20 --no-pager
```

`backup-status.json` is written after every run (success or failure) with `mode`, `status`,
`timestamp`, and a short `detail` — the same non-sensitive fields exposed at
`GET /api/operator/backup-status` (operator-authenticated only; no tokens, filenames, or
restore controls are exposed there).

`rclone lsl victoryvtt-crypt:VictoryBackups/quick/` and `.../full/` list what has actually
landed remotely, with real (decrypted-view) filenames, sizes, and timestamps — the crypt remote
handles decryption transparently for anyone with the working rclone config.

---

## 3. Run it by hand

```bash
cd /opt/victory
bash scripts/backup/backup.sh quick
bash scripts/backup/backup.sh full

# or via systemd, which is what the timers do:
systemctl start victory-backup-quick.service
systemctl start victory-backup-full.service
```

A manual run is safe to do any time — it never touches the live database beyond a read-only
`pg_dump`, and upload is additive (`rclone copy`, never `sync`).

`retention.sh` defaults to a dry run:

```bash
bash scripts/backup/retention.sh          # lists what would be deleted
bash scripts/backup/retention.sh --apply  # actually deletes it
```

---

## 4. What's inside a backup archive

Each run produces one immutable, timestamped `.tar.gz` (e.g. `full-20260801-232526.tar.gz`),
staged locally under `/opt/victory/backups/scheduled/` before upload, containing:

```text
database.dump       pg_dump --format=custom output
assets/              (full mode only) a copy of STORAGE_ROOT, symlinks never followed
manifest.json        commit hash, migration count, timestamp, format version, restore command
checksums.sha256      sha256sum of every file above
```

The manifest deliberately does not include `.env` or any credential.

---

## 5. Failure behavior

A failed run:

- writes `{"status":"failed","detail":"<reason>"}` to `backup-status.json`;
- exits nonzero (visible in `systemctl status` / `journalctl`);
- leaves its staged directory in place for diagnosis rather than deleting it;
- does **not** touch or restart Victory itself — `victory-backend` and `victory-postgres` are
  untouched by a backup run regardless of outcome.

There is no alerting service wired up. Checking `backup-status.json` or
`GET /api/operator/backup-status` periodically (or after any host maintenance) is currently a
manual operator habit, not automated paging.

---

## 6. Related documents

- `victory-restore-runbook.md` — how to actually restore from one of these archives.
- `victory-backup-secret-recovery.md` — what must be preserved outside this host for a restore
  to be possible if the host itself is lost.
- `kernel-77-backup-restore-proof.md` — the evidence that a real restore was performed and
  measured.
