# Kernel Report Back — Kernel 77: Private-Client Readiness

## 1. Status

**PASS — Kernel complete. Level 4 achieved for deletion, export, and backup/restore. Level 4
formally withheld on recovery email specifically, pending an external Brevo account approval.**

```
Achieved readiness: Level 4 — Private paying clients (deletion, export, backup/restore)
                     Level 3+ (recovery: code-complete, tested, deployed; real delivery unproven)
Required target:    Level 4 — Private paying clients
Stretch target:     Level 5 — Public signup (not attempted, out of scope)
```

This is the honest reading of Kernel 77 §19's "PASS — Kernel complete, Level 4 withheld"
category: all scoped recovery-email work is implemented and evidenced by six passing automated
tests plus confirmed-correct live configuration (SMTP host, verified sending domain, IP
allowlist recognized by Brevo); the one remaining gap — proof that a real email actually lands
in a real inbox — is blocked by what appears to be a pending account-level review on Brevo's
side, not by anything in Victory. Three independently-correct fixes (SMTP login format, a
freshly-generated key, and an IP-allowlist entry Brevo itself confirmed recognizing) all failed
identically (`535 5.7.8 Authentication failed`), which is the pattern of an account gate, not a
configuration error.

**Operator note, Grant's own words, recorded verbatim as requested:** *"Remind me in kernel
78+ to check if I have been approved yet with Brevo."*

---

## 2. Preflight: what was found before building anything

- **Kernel numbering conflict.** A "Kernel 77 — Proposed Hosted-User Stabilization Scope.md"
  already existed in `Construction/Kernels/`, distinct from and narrower than this kernel's
  spec. Per Grant's decision, both were merged into one canonical document
  (`Construction/Kernels/Kernel 77 — Private-Client Readiness.md`), with the detailed spec
  winning on conflicts and the proposal's extra hardening items (K77-05 through K77-12) folded
  in as required, not optional. The old file is marked superseded, not deleted.
- **Kernel 74/75 test root cause was misdiagnosed in the source material.** The Kernel 76
  reportback described the two failing tutorial tests as depending on undeclared seed data.
  Actual cause, found by tracing the dialogue-topic schema: migration
  `078_kernel75_ra_supervisor_handoff.sql` deliberately retired the `crown-bet` topic in favor
  of a supervisor-handoff ending, and the two tests were never updated to match. Fixed by
  updating the tests' assertions to the current design, not by adding fixture-seeding (Kernel
  77 §11 asked for the latter; the evidence pointed to the former, and the evidence won).
- **A stale agent worktree** (`.claude/worktrees/agent-ad8c3df7c85e39ce3`) contained ~700
  uncommitted lines confirmed fully superseded by later work in `main`, plus a copy of the
  pre-Kernel-76 hardcoded database password pattern (not a live credential). Removed.
- **The two competing Kernel 75 spec files** already cross-referenced each other in one
  direction; added the missing reverse pointer.
- **Privacy Policy / Terms verification** (Kernel 77 §10): found and corrected two real gaps —
  the age-eligibility language contradicted the kernel's 17+ decision (was 13+, COPPA-style,
  with a parental-consent carve-out Victory doesn't support), and the contact address was a
  literal unfilled `[EMAIL]` placeholder. Grant corrected both directly in the Google Docs
  (age language re-drafted himself; `privacy@amurray.family` mailbox created and added). Neither
  document previously linked from anywhere in the app; added links from `/account/` and
  `/login/`. Full detail in `Construction/Domains/Privacy/kernel-77-policy-terms-verification.md`.

---

## 3. What was built

### Goal A — Account deletion
`backend/internal/identity/account_deletion.go`, migrations 081–082, six tests. Private data
hard-deletes via existing CASCADE FKs; the twelve previously-`RESTRICT` shared-history columns
reassign to a new credential-less tombstone account rather than blocking deletion. Blocks only
on sole-producer-at-a-Location-with-Productions or the operator account. Full design in
`Construction/Domains/Privacy/account-deletion-policy.md` and
`Construction/Domains/Privacy/account-deletion-dependency-map.md`.

### Goal B — Self-service data export
`backend/internal/identity/account_export.go`, three tests. Background-goroutine job,
session-authenticated download, JSON+Markdown+original-files archive scoped to the requesting
user. Format documented in `Construction/Domains/Privacy/user-export-format.md`.

### Goal C — Encrypted off-host backup and tested restore
`scripts/backup/{backup.sh,retention.sh}`, three systemd timer pairs (installed and enabled,
survive reboot), `GET /api/operator/backup-status`. A real isolated restore was performed and
measured, not simulated. Full evidence in
`Construction/Domains/Operations/kernel-77-backup-restore-proof.md`; runbooks in
`victory-backup-runbook.md`, `victory-restore-runbook.md`, `victory-backup-secret-recovery.md`.

### Goal D — Self-service recovery email
`backend/internal/mailer/`, reopened `HandleForgotPassword`, new email-verification
request/confirm endpoints, six tests. See §1 for delivery status.
`Construction/Domains/Operations/victory-account-recovery-runbook.md` updated with the self-service
path.

### Goal E — Fresh-database merchant test repair
See §2. Both tests proven passing against a genuinely from-scratch rebuilt `victory_test`
(`scripts/test/reset-test-database.sh` with `CONFIRM_TEST_DB_RESET=1`), with live database row
counts unchanged before/after.

### Merged hardening (K77-05 through K77-10)
- **K77-05** Session revocation now reaches open WebSockets (30s periodic revalidation sweep).
- **K77-06** Rate limiting extended to messages, note cards, uploads, warehouse operations,
  command execution, WebSocket connection attempts, and WebSocket messages.
- **K77-07** CSRF posture made explicit via a regression test that freezes the reviewed set of
  bare (no-method-prefix) routes and fails on unreviewed additions.
- **K77-08** WebSocket upgrades now authenticate before completing the handshake (was K76-L01).
- **K77-09** `postgres:16-alpine` pinned to a digest. `caddy:2-alpine` intentionally not
  touched — shared infrastructure outside this repository (see §5).
- **K77-10** `/health` decision: expose it. Proxied through the shared Caddyfile to the real
  backend health check.

(K77-11 stale worktree and K77-12 duplicate Kernel 75 docs — see §2.)

---

## 4. Evidence

### 4.1 Commands

```bash
$ cd /opt/victory && git rev-parse HEAD && git branch --show-current
c3a7a3a58c6f3a9e19216a7e0793c3d11821b7f1
main
$ docker ps --format '{{.Names}}\t{{.Status}}'
victory-backend    Up (redeployed this kernel)
victory-postgres   Up (recreated this kernel — see §5, data preserved via named volume)
bread-caddy        Up (restarted this kernel — see §5)
bread-exchange     Up 2 months (unaffected)
```

### 4.2 Test suite

Full suite run fresh (`-count=1`, no cache) against a genuinely from-scratch rebuilt
`victory_test`, twice — once for the merchant tests specifically, once for everything:

```
$ go test -count=1 ./...
ok all packages, 0 failures
```

Live database row count before and after the entire run: unchanged (`users` count 4 both
times — confirming test isolation held throughout, including the new Kernel 77 test suites).

### 4.3 Static checks

```
$ git diff --check
(clean)
```

Every new/changed inline `<script>` block checked with `node --check`: clean.

### 4.4 Backup/restore evidence

See `Construction/Domains/Operations/kernel-77-backup-restore-proof.md` in full. Summary:

```
Backup destination: encrypted Google Drive crypt remote
Backup schedule:    quick every 15 min, full daily 03:00 UTC
Retention:          quick 48h, full 30d (dry-run by default)
Last successful off-host backup: confirmed live, both modes, both direct and via systemd
Integrity check:    rclone check against the crypt remote, 0 differences
Restore source:     most recent full-* archive, downloaded fresh from Drive
Restore target:     isolated throwaway PostgreSQL container, not the live one
RPO:                ≤15 min (database), ≤24h (assets)
RTO:                ≈69s measured end-to-end in a single continuous timed run
```

### 4.5 Email evidence

```
Provider/mechanism:      Brevo SMTP relay (smtp-relay.brevo.com:587, STARTTLS)
Controlled delivery:     NOT achieved -- 535 5.7.8 Authentication failed, three fixes attempted
Request endpoint status: open (200, generic response) when RECOVERY_EMAIL_ENABLED and SMTP configured
Confirmation status:     proven via automated test (fake mailer capturing the real token/URL shape)
Replay status:           rejected -- proven
Session revocation:      proven -- pre-reset session dies on successful reset
Rate limit result:       proven -- 429 after repeated requests
```

Addresses, tokens, and credentials are not reproduced here per Kernel 77 §16.5.

### 4.6 Browser/live evidence

```
$ curl https://victory.amurray.family/health
{"ok":true,"service":"victory-backend","time":"..."}          <- now publicly proxied (K77-10)

$ curl -X POST https://victory.amurray.family/api/auth/signup ...
403 password_signup_closed                                     <- Kernel 76 closure intact

$ curl https://victory.amurray.family/api/account/me (anon)
401
$ curl https://victory.amurray.family/api/third-place/headshots (anon)
401
$ curl https://victory.amurray.family/api/productions (anon)
401
$ curl https://victory.amurray.family/api/discord/gateway/status (anon)
401

$ curl https://victory.amurray.family/ws/the-cave (plain GET, anon)
403   <- previously required a WS client to observe; now a plain HTTP status (K77-08)
$ curl https://victory.amurray.family/ws/catharsis (anon)      -> 403
$ curl https://victory.amurray.family/ws/first-theater (anon)  -> 403
$ curl https://victory.amurray.family/ws/player-profile (anon) -> 401

$ curl -X POST https://victory.amurray.family/api/account/deletion-plan (anon) -> 401
$ curl -X POST https://victory.amurray.family/api/account/export (anon)        -> 401
$ curl https://victory.amurray.family/api/operator/backup-status (anon)        -> 401

$ victory-recover whoami --handle straturli
is_operator: true, discord_id linked, memberships: amurray-family producer (active),
live_sessions: 1   <- Grant's own Discord-authenticated session, confirmed unaffected throughout
```

---

## 5. Deviations from kernel and unplanned live actions

1. **`docker compose up -d --build backend` recreated `victory-postgres`, not just `backend`.**
   Pinning the postgres image to a digest (K77-09) changed its config from Compose's point of
   view, and `--build backend` still reconciles the full dependency graph. Data was preserved
   (named volume, independent of container lifecycle) — verified immediately after: user count,
   `straturli`'s full record, and the new tombstone account all correct. Confirmed no data loss,
   but flagging the mechanism since it wasn't the intended scope of that command.
2. **Editing `/opt/bread-exchange/Caddyfile` and restarting `bread-caddy`** (K77-10) touches
   infrastructure shared with `bread-exchange`, a different live service. Confirmed with Grant
   before restarting. The edit itself only added a block to the `victory.amurray.family` site;
   `exchange.amurray.family`'s block was untouched.
3. **Restarting `bread-caddy` surfaced a pre-existing, unrelated problem**: `exchange.amurray.family`
   has no TLS certificate in Caddy's persistent storage at all (only a stale issue-lock file),
   and Let's Encrypt validation fails with a connection timeout. This is not caused by anything
   in this kernel — the edit didn't touch the exchange block, and Victory's own certificate
   (separately stored, separately managed) was confirmed unaffected and serving correctly
   throughout. Grant will handle it separately.
4. **`docker-compose.yml` was missing the Kernel 77 environment variables entirely**
   (`RECOVERY_EMAIL_*`, `SMTP_*`, `EXPORTS_ROOT`) — they existed in `.env` for Compose's own
   variable substitution but were never listed in the `backend` service's `environment:` block,
   so the container never actually received them until this was caught and fixed during live
   verification. Worth remembering for future kernels: a new `.env` variable is not
   automatically available inside the container without an explicit line in
   `docker-compose.yml`.

---

## 6. Blockers & workarounds

**BLOCKER:** Self-service recovery email cannot be proven to deliver.
**CAUSE:** Brevo SMTP authentication fails (`535`) despite three independently-correct fixes.
**WORKAROUND:** `victory-recover` remains fully functional for every account, including the
operator's own.
**OPERATOR ACTION REQUIRED:** Check Brevo account approval status; see Grant's own note in §1,
recorded for Kernel 78+.

**BLOCKER (pre-existing, not this kernel's):** `exchange.amurray.family` has no valid TLS
certificate.
**CAUSE:** Unknown — predates this kernel, surfaced by an unrelated restart.
**WORKAROUND:** None attempted; explicitly out of scope, Grant is handling separately.
**OPERATOR ACTION REQUIRED:** Grant's own follow-up, not Victory's.

**BLOCKER (informational, not blocking):** 38 files under `STORAGE_ROOT` have no corresponding
`assets` row.
**CAUSE:** Unknown — pre-existing, surfaced incidentally during the restore proof's asset
verification.
**WORKAROUND:** None needed; backup/restore correctly captures whatever is actually on disk
regardless.
**OPERATOR ACTION REQUIRED:** Worth a cleanup sweep in a future kernel; not urgent.

---

## 7. Required Reportback Additions

### 7.1 Deletion proof
Six tests passing: private-only deletion (credentials, journal, session all removed), shared
Action preserved with attribution reassigned to the tombstone account, sole-producer blocker
raised then cleared on transfer, operator account refused, confirm-handle mismatch refused,
unauthenticated request refused.

### 7.2 Export proof
Three tests passing: full request→status→download round trip against a real fixture account
(archive reopens as valid zip, contains expected files, private journal entry present, manifest
counts correct, no secret-shaped string anywhere in the archive bytes); cross-user download and
status both correctly return 404 rather than another user's data; expired export refused (410)
and file removed from disk.

### 7.3 Recovery proof
See §4.5. Full mechanism proven via automated tests with a fake mailer capturing real
URLs/tokens; live SMTP delivery not proven — see §1, §6.

### 7.4 Backup proof
See §4.4 and the full standalone evidence document.

### 7.5 Legal-surface verification
`Construction/Domains/Privacy/kernel-77-policy-terms-verification.md`. Privacy Policy: age language
corrected to 17+ (Grant's own edit), contact placeholder replaced with a real mailbox, both now
linked from the app. Terms of Service: still has no contact line — flagged as an open operator
wording decision, not fixed by this kernel (per §10.2, factual corrections yes, new legal
drafting no).

### 7.6 Remaining blockers
Recovery-email real delivery (external, Brevo). Everything else in Kernel 77's scope is
complete and proven.

---

## 8. The one thing worth remembering

Every piece of this kernel that actually broke something did it the same way: a value that
looked complete in isolation but was silently missing from the one place that actually mattered
at runtime. The Kernel 77 environment variables existed in `.env` but not in
`docker-compose.yml`'s explicit list, so the container never saw them. The Caddyfile edit was
correct on disk but the running container held an orphaned inode from a single-file bind mount,
so it never saw the change either, until a restart. Both looked done and weren't, and both were
only caught by testing the live path directly rather than trusting that a correct-looking source
file meant the running system matched it.
