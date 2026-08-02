# Victory — Security Notes

**Current as of:** Kernel 77, 2026-08-02
**Readiness:** Level 4 — Private paying clients, for account deletion, data export, and
encrypted off-host backup with a proven restore. Self-service password recovery is
code-complete and fully tested but **not yet proven in real-world delivery** — see §5.
**Supersedes:** `Security-notes-kernel-2-historical.md` (Kernel 2 posture, historical only).
Kernel 76's evidence in `Construction/Security/kernel-76-*.md` remains authoritative for
everything it covered; this document layers Kernel 77's changes on top rather than repeating
Kernel 76's verified guarantees, which the previous version of this file already recorded and
which remain true.

Full Kernel 77 evidence: `Construction/OperatorLogs/kernel-77-reportback.md`,
`Construction/Operations/kernel-77-backup-restore-proof.md`,
`Construction/Privacy/*.md`.

---

## 1. What Kernel 77 verified or built

**Account deletion** (`backend/internal/identity/account_deletion.go`)
- Private data hard-deletes via the existing, Kernel-76-verified CASCADE foreign keys.
- Shared history (`actions.actor_id` and eleven other previously-`RESTRICT` columns) reassigns
  to a dedicated, credential-less tombstone account (`deleted-user`) rather than blocking
  deletion or leaving a bare NULL — see `Construction/Privacy/account-deletion-policy.md`.
- Blocked when the account is the sole active Producer at a Location with Productions, or is
  the operator account. Both proven with negative tests.
- Six tests, all passing: private-only deletion, shared-Action anonymization, sole-producer
  block-then-clear, operator refusal, confirm-handle mismatch, unauthenticated refusal.

**Data export** (`backend/internal/identity/account_export.go`)
- Self-service, background-goroutine job, session-authenticated download (no token in the
  URL). JSON + Markdown + original files, scoped entirely to the requesting user; verified no
  secret-shaped string appears in a real generated archive.
- Three tests passing: full round trip, cross-user denial, expired-archive refusal.

**Encrypted off-host backup** (`scripts/backup/`, systemd timers)
- `rclone crypt` remote over a dedicated Google Drive account; contents and filenames both
  encrypted — verified by comparing the raw and decrypted views directly.
- Quick (database, 15 min) and full (database + assets, daily) timers installed, enabled, and
  proven to survive reboot (`Persistent=true`).
- A real isolated restore was performed end-to-end: downloaded, checksum-verified, restored
  into a throwaway PostgreSQL container (not the live one), booted a real backend against it,
  health-checked, and authenticated the real operator account via `victory-recover whoami`
  against the restored data. RPO ≤15 min (DB) / ≤24h (assets), RTO ≈69s measured. Full detail
  in `kernel-77-backup-restore-proof.md`.
- A deliberate failure was run to prove failure is visible and doesn't affect Victory: nonzero
  exit, `backup-status.json` records it, `victory-backend` unaffected.

**Session revocation now reaches open WebSockets** (K77-05, `internal/network/hub.go`)
- Previously, revoking a session blocked new WebSocket connections but did not close ones
  already open. A 30-second periodic sweep (`Hub.RevalidateSessions`) now closes any connected
  client whose user has no remaining valid session. Proven with a real dial-connect-revoke-
  observe-close test.

**Rate limiting extended beyond credential endpoints** (K77-06)
- Messages, note cards, uploads, warehouse asset operations, and command execution now share a
  more generous action-rate limiter (burst 30, 60/min) alongside the existing tight
  credential limiter.
- WebSocket connection *attempts* are now rate-limited per IP, and inbound WebSocket
  *messages* are rate-limited per connected user (dropped, not connection-closing, to tolerate
  a legitimate burst). Neither existed before Kernel 77.

**CSRF posture made explicit** (K77-07)
- The existing posture (`SameSite=Lax` + no GET route mutates) was already correct per Kernel
  76's manual inspection. A regression test
  (`backend/cmd/victory/kernel77_csrf_posture_test.go`) now enforces it going forward: it
  freezes the current set of routes registered without an explicit HTTP method prefix and
  fails if a new one appears unreviewed, forcing a deliberate choice rather than silent drift.

**WebSocket upgrade now authenticates before completing the handshake** (K77-08, was K76-L01)
- `ServeVenueWS`, `ServeCaveWS`, and `ServeProfileWS` all resolve session, venue admission, and
  (for venue sockets) active-session identity *before* calling `upgrader.Upgrade`. An
  unauthorized caller now gets a plain HTTP 401/403 and never completes a handshake at all —
  verified both by a Go test and live against the production host (`curl` to `/ws/*` now
  returns a clean HTTP status instead of what previously required a WebSocket client to see).

**Base image pinned** (K77-09)
- `postgres:16-alpine` is now pinned to a digest in `docker-compose.yml`, so a routine pull
  cannot silently swap the image. `caddy:2-alpine` was not touched — Victory's edge is a
  *shared* Caddy instance (`bread-caddy`) whose real config lives in
  `/opt/bread-exchange/Caddyfile`, serving `exchange.amurray.family` as well; pinning it is
  outside this repo's scope and was not attempted.

**`/health` decision made and implemented** (K77-10)
- Decided: expose it. `victory.amurray.family/health` now proxies to the real backend health
  check (`{"ok","service","time"}` — no version, config, or auth state) via a new `handle
  /health` block in the shared Caddyfile, gracefully reloaded (validated first) and confirmed
  live. External uptime monitoring now has something to poll.

---

## 2. Repaired in Kernel 76 (unchanged, still true)

See the previous version of this file (`git log` on this path) or
`Construction/Security/kernel-76-findings.md` for the full account: raw password-reset tokens
in logs, open public registration, Third Place identity leak, committed default database
password, Production-list disclosure, unauthenticated Discord gateway status, missing transport
limits. All seven repairs were re-verified live during this kernel's regression pass and remain
in effect — see §4.

---

## 3. Self-service password recovery: built, not yet proven live

`POST /api/auth/password-reset/request` is reopened, but only when `RECOVERY_EMAIL_ENABLED` is
true **and** SMTP is actually configured — otherwise it stays exactly as Kernel 76 left it
(`410`). As of this kernel, the live config is complete (Brevo SMTP relay, domain-verified
sender, IP-allowlisted) and the endpoint is genuinely open.

What's proven:
- Uniform response for known/unknown/unverified addresses (privacy-safe).
- A verified email only sends when `users.email_verified_at` is set; verification has its own
  request/confirm flow and token shape.
- Token hashing, 1-hour expiry, single-use, replay rejection, and full session revocation on
  reset — all identical to `victory-recover`'s own break-glass token, and covered by six
  automated tests.
- Rate limiting on the request endpoint.

What's **not** proven: an actual email landing in a real inbox. Every live send attempt failed
with `535 5.7.8 Authentication failed` from Brevo, despite three independently-correct fixes in
sequence (SMTP login format, a freshly-generated key, and an IP-allowlist entry that Brevo
itself confirmed recognizing). This pattern points to a pending account-level review on Brevo's
side, not a configuration or code defect — see the Kernel 77 reportback for the full sequence.

**Operator note, Grant's own words:** *"Remind me in kernel 78+ to check if I have been
approved yet with Brevo."*

Until delivery is proven, `victory-recover` remains the reliable path for any account,
including the operator's own.

---

## 4. Disabled surfaces

| Surface | State | Reversal |
|---|---|---|
| `POST /api/auth/signup` | `403 password_signup_closed` | set `PASSWORD_SIGNUP_ENABLED=true` |
| `POST /api/auth/password-reset/request` | Open when `RECOVERY_EMAIL_ENABLED=true` and SMTP is configured; `410` otherwise | already reversed, pending proven delivery — see §3 |
| `GET /api/discord/gateway/status` for non-operators | `401`/`403` | intentional; not to be reversed |

`/api/auth/password-reset/confirm` remains live — it is how break-glass recovery tokens are
redeemed, and now also how self-service tokens are redeemed, via the identical mechanism.

---

## 5. Account model

Unchanged from Kernel 76: Discord establishes accounts; email/password login is the recovery
fallback; email/password registration is closed; operator authority is `handle ==
OPERATOR_HANDLE` with no database flag. Kernel 77 adds: a credential-less `deleted-user`
tombstone account (never resolvable via login, holds no membership) that anonymized shared
history is reassigned to on deletion, and `users.email_verified_at`, gating whether an address
is trusted for recovery.

---

## 6. Remaining gaps

| Item | Status |
|---|---|
| Self-service recovery email real delivery | Blocked on Brevo account approval — see §3. Reminder recorded for Kernel 78+. |
| `caddy:2-alpine` digest pin | Not pinned — shared infrastructure outside this repo, see §1. |
| Orphaned files under `STORAGE_ROOT` | 38 files on disk with no corresponding `assets` row, discovered incidentally during the restore proof. Pre-existing, not a Kernel 77 defect. Worth a future cleanup sweep. |
| Live-over-live restore (§8 of the restore runbook) | Documented but not separately drilled — only the isolated-environment path was rehearsed. |

---

## 7. Operator privacy

Unchanged from Kernel 76: ordinary operator tooling does not render journals, relationship
notes, message bodies, or unpublished profiles. Kernel 77's backup and export tooling follow
the same rule — export is scoped to the requesting user only, and backup/restore proof used a
throwaway fixture-shaped verification (`victory-recover whoami`), not inspection of real
private user content.

---

## 8. Rule for future kernels

Unchanged from Kernel 76, and still the right lens: the question for a new surface is never
"is the caller authenticated," it's which boundary — Venue, Production, ownership, or now
account-lifecycle state (verified email, sole-producer, operator) — the action sits behind, and
whether the caller is inside it.
