# Kernel 77 — Backup and Restore Proof

**Date:** 2026-08-01 / 2026-08-02
**Scope:** Goal C — encrypted off-host backup with a tested restore (Kernel 77 §9, §18).

No secrets, tokens, or credentials appear below, per §16.4/§22.5.

---

## 1. Backup destination

```
Backup destination: encrypted Google Drive crypt remote (rclone, victoryvtt-crypt:VictoryBackups)
Backup schedule:     quick (database) every 15 minutes; full (database + assets) daily at 03:00 UTC
Retention:           quick 48h, full 30d (retention.sh, dry-run by default; systemd timer weekly with --apply)
```

## 2. Live backup evidence

A full backup and a quick backup were each run twice — once directly via `backup.sh`, once
through the installed `systemd` service — and both succeeded:

```
=== Victory backup (full) starting at 2026-08-01T23:25:26Z ===
NOTICE: Encrypted drive 'victoryvtt-crypt:VictoryBackups/full/': 0 differences found
NOTICE: Encrypted drive 'victoryvtt-crypt:VictoryBackups/full/': 1 matching files
=== Victory backup (full) completed at 2026-08-01T23:25:37Z ===

local backup timestamp:  2026-08-01T23:25:26Z
database dump size:      part of a 15,653,787-byte (~14.9 MiB) full archive
asset count/size:        38 files, all checksum-verified
manifest version:        1 (commit c3a7a3a58c6f3a9e19216a7e0793c3d11821b7f1, 81 migrations)
remote upload result:    success (rclone copy, 0 errors)
crypt integrity result:  rclone check --one-way against the crypt remote: 0 differences, 1 matching file
```

```
=== Victory backup (quick) starting at 2026-08-02T04:27:28Z ===  (via systemctl start, not direct invocation)
=== Victory backup (quick) completed at 2026-08-02T04:27:34Z ===
systemctl status: code=exited, status=0/SUCCESS
```

**Failure behavior proven**, not just asserted — a deliberate failure (bad container name) was
run and confirmed to fail loudly without affecting Victory:

```
$ POSTGRES_CONTAINER=nonexistent-container bash scripts/backup/backup.sh quick
BACKUP FAILED (quick): pg_dump failed
exit code: 1
backup-status.json: {"status": "failed", "detail": "pg_dump failed", ...}

$ curl -s https://victory.amurray.family/api/session/me
{"ok":true,"signed_in":false}   <- Victory itself unaffected
```

Timers confirmed installed, enabled, and scheduled to survive reboot:

```
$ systemctl list-timers 'victory-backup-*'
NEXT                          UNIT                             ACTIVATES
Sun 2026-08-02 04:30:04 UTC   victory-backup-quick.timer       victory-backup-quick.service
Mon 2026-08-03 00:07:02 UTC   victory-backup-retention.timer   victory-backup-retention.service
Mon 2026-08-03 03:00:08 UTC   victory-backup-full.timer        victory-backup-full.service
```

## 3. Isolated restore proof

Restore source: the most recent `full-*` archive downloaded fresh from
`victoryvtt-crypt:VictoryBackups/full/`, decrypted transparently by the crypt remote.

Restore target: a throwaway `postgres:16-alpine` container (`victory-restore-test`), bound only
to `127.0.0.1:15432`/`15433`, entirely separate from the live `victory-postgres` container — not
the live database, and not the same container.

Steps proven, in order:

1. **Downloaded and checksum-verified** — every file in the archive passed `sha256sum -c`
   (database dump, all 38 asset files, manifest).
2. **Isolated PostgreSQL created** — `victory_restore_test` database in a separate container on
   an alternate port.
3. **Database restored** — `pg_restore --format=custom` into the isolated database. Verified
   `SELECT COUNT(*) FROM users` = 3 (matching live), `schema_migrations` = 81 (matching live at
   backup time).
4. **Assets restored** — all 38 files copied to an isolated `STORAGE_ROOT`; spot-checked files
   confirmed real, non-empty binary image content (hundreds of KB to ~1.6 MB each).
5. **Victory launched against the restored state** — a real build of `cmd/victory`, pointed at
   the isolated database and storage path, on an isolated port. Two Kernel 77 migrations
   (081, 082 — not yet applied live at backup time) applied cleanly on top of the restored
   dump, proving restored state can catch forward to current schema.
6. **Health check passed**: `{"ok":true,"service":"victory-backend",...}`.
7. **Account authentication proven** against the restored data via `victory-recover whoami`:

   ```
   handle:        straturli
   display_name:  Grant A. Murray
   discord_id:    694402257331945562
   is_operator:   true
   memberships:   amurray-family: producer (active)
   ```

   — an exact match to the live account, restored from an off-host encrypted copy.
8. **Representative records verified**: `locations`, `venues`, `equipment_items` counts matched
   live values exactly. `assets`/`character_cards` were correctly 0 in both live and restored
   databases — see the note below.
9. **Isolated environment fully torn down** afterward: container removed, temporary process
   killed, staging directories deleted. The live `victory-backend`/`victory-postgres` containers
   were confirmed running and undisturbed throughout.

### A finding surfaced by this proof, not caused by it

The live `assets` and `character_cards` tables are currently empty (0 rows) — expected, since
Kernel 76's rebuild wiped both intentionally. However, `STORAGE_ROOT` on the host contains 38
real image files with no corresponding database rows. The backup faithfully captured this
mismatch (it backs up whatever is actually on disk); it is a pre-existing storage-hygiene gap,
not a Kernel 77 defect. Recorded here rather than silently fixed, since Kernel 77's scope is
deletion/export/backup/recovery, not asset-table reconciliation. Worth a follow-up sweep in a
future kernel to identify and clean up orphaned files under `STORAGE_ROOT`.

## 4. RPO / RTO

```
RPO (database):  demonstrated ≤ 15 minutes — the quick timer's own interval, confirmed running
RPO (assets):    demonstrated ≤ 24 hours — the full timer's own interval, confirmed running
RTO:             ≈ 69 seconds, end-to-end, in a single uninterrupted timed run covering:
                   Drive download of a ~14.9 MiB archive, checksum verification, isolated
                   Postgres container start, pg_restore, asset copy, and backend startup
                   through auto-migration to a healthy /health response.
```

The backend itself reached a healthy `/health` response **5 seconds** after being launched
against the fully-restored database — most of the measured 69 seconds was the one-time
`go run` compile step and the network download, neither of which a production restore against
a prebuilt binary and a warm local cache would necessarily repeat in full. 69 seconds is
reported as the honest, conservative figure actually measured, not an optimistic estimate.

An earlier attempt at this same measurement, spread across several separate tool invocations
in this session, produced a nonsensical ~6-hour delta due to an environment clock adjustment
between those invocations (unrelated to actual restore performance) — discarded in favor of
the single continuous timed run reported above.

## 5. What this proves for Level 4

- Database and assets leave the host encrypted: **yes**, verified via `rclone check` and
  manual decryption through the crypt remote.
- Drive receives only encrypted content: **yes** — the raw `victoryvtt-drive:` remote shows
  scrambled filenames and unreadable content; only `victoryvtt-crypt:` presents plaintext.
- Backup runs on schedule: **yes**, systemd timers installed, enabled, survive reboot.
- Failures are observable: **yes**, demonstrated with a deliberate failure.
- Retention is controlled, not automatic destruction: **yes**, `retention.sh` defaults to
  dry-run; a real remote delete requires `--apply`.
- Restore from Drive succeeds in isolation: **yes**, full path proven above.
- RPO/RTO measured, not asserted: **yes**, see §4.
- Required recovery secrets preserved outside the host: **yes** for the rclone crypt
  password/password2 (confirmed in Bitwarden); the Drive account and Brevo account credentials
  themselves are Grant's to store the same way — see `victory-backup-secret-recovery.md`.
