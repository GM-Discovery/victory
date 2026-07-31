# Victory — Backup and Restore Assessment

Assessed during Kernel 76 (§5.16). States what exists, what it actually protects against, and
what must be true before paying clients.

---

## 1. What exists now

| Mechanism | Trigger | Location | Covers |
|---|---|---|---|
| Pre-migration dump | automatic, before applying pending migrations (Kernel 72) | `/opt/victory/backups/*.dump` | database only |
| Manual pre-kernel dump | a builder remembering | same directory | database only |
| Uploaded files | none | `/opt/victory/storage` (16 MB) | **not backed up** |
| Off-host copy | none | — | **nothing leaves the host** |
| Encryption at rest | none | — | dumps are plaintext SQL |
| Scheduled job | none — no cron entry, no systemd timer | — | — |
| Tested restore | none on record | — | — |

Eleven dumps totalling 68 MB, spanning 2026-07-16 to 2026-07-30. Every one was produced by a
migration or a person, never by a schedule.

## 2. What this actually protects against

**Protects against:** a bad migration, a bad deploy, a destructive mistake made *while
someone was paying attention* — which is a real and frequently-realised risk, and the
mechanism has genuinely earned its place.

**Does not protect against:** disk failure, host loss, ransomware, accidental
`docker volume rm`, or any incident where the host itself is the casualty. Every copy lives on
the machine being protected. Uploaded files have no copy at all.

**Honest recovery point:** since backups are event-triggered rather than scheduled, the
demonstrated maximum data-loss window is *unbounded* — it is however long since someone last
ran a migration. Between 2026-07-20 and 2026-07-30 no dump was taken at all: a ten-day gap.

Kernel 76 §5.16 forbids claiming "zero data loss" without architecture proving it. The
measured recovery point is: **unbounded, and in practice up to ten days.**

## 3. What Kernel 76 verified

The Kernel 76 rebuild proved the *reconstruction* path rather than the restore path, and the
distinction matters:

- A dump of the pre-Kernel-76 database was taken and retained
  (`/opt/victory/backups/k76/victory-pre-k76-20260730-2028.sql`, 5.5 MB).
- An **empty database was rebuilt from migrations alone**: 81 migrations applied cleanly,
  the backend booted, and seeds produced 2 Locations, 16 Venues, and the 150-item equipment
  catalogue. Verified twice — on the `victory_k76_clean` scratch database and then on the live
  `victory` database.
- Operator access was re-established from nothing via `victory-recover bootstrap` plus a
  recovery link.

So Victory can be reconstructed from source and migrations. What has **not** been tested is
restoring a `pg_dump` into a running deployment and confirming the application works against
recovered data.

## 4. Required before paying clients

| # | Requirement | Status |
|---|---|---|
| 1 | Scheduled automated PostgreSQL backup (daily minimum) | **missing** |
| 2 | Asset backup covering `/opt/victory/storage` | **missing** |
| 3 | Encrypted off-machine copy | **missing** |
| 4 | Retention policy | **missing** — eleven dumps kept by accident |
| 5 | Written restore instructions | **missing** |
| 6 | A restore actually performed and verified | **missing** |
| 7 | Stated maximum data-loss window | currently unbounded |
| 8 | Stated maximum restoration time | unmeasured |

Items 1–3 are the ones that change the risk category. Items 5–6 are the ones that turn a
backup into a recovery capability — an untested backup is a hypothesis.

## 5. Recommended shape (Kernel 77)

Deliberately modest, because the failure mode to avoid is a backup system elaborate enough
that nobody maintains it:

1. **Nightly `pg_dump`** via systemd timer, custom format, 30-day retention, into
   `/opt/victory/backups/daily/`.
2. **Nightly asset snapshot** — `tar` of `/opt/victory/storage`, same retention. At 16 MB this
   is trivially cheap.
3. **Encrypted off-host sync** — `age` or `gpg` to a symmetric key held outside the host, then
   `rclone`/`rsync` to any second location. Off-host matters more than which destination.
4. **A quarterly restore drill** with the elapsed time recorded, producing the two numbers §4
   asks for.
5. **Restore runbook** alongside the recovery runbook, including how to restore assets and
   database together so they are consistent with each other.

Target after implementation: recovery point ≤ 24 hours, recovery time ≤ 1 hour, both measured
rather than asserted.

## 6. Interaction with account deletion

When deletion lands (K76-M04), backups become a retention question: a user who deletes their
account still exists in every dump taken before that moment. The retention policy in item 4
should therefore be short enough to bound that window, and the deletion path should state
plainly that backups age out over the retention period rather than being rewritten.
