# Kernel 77 — Proposed Hosted-User Stabilization Scope

> **SUPERSEDED 2026-08-01.** This proposal has been merged into
> `Construction/Kernels/Kernel 77 — Private-Client Readiness.md`, the canonical Kernel 77
> specification, per Grant's decision to merge both documents with that one winning on
> conflicts. Items K77-05 through K77-12 below are carried forward there as §25 "Merged scope."
> Kept here for history only — do not implement from this file.

**Status:** PROPOSAL — not a final specification (superseded, see above)
**Scope source:** Kernel 76 findings (`Construction/Security/kernel-76-findings.md`)
**Purpose:** advance Victory from readiness Level 3 (invited strangers) to Level 4 (private
paying clients)

---

## 0. What this kernel is for

Kernel 76 closed every Critical and High finding it found. Level 4 is not blocked by an
exposure any more — it is blocked by **absent capabilities**: a client cannot delete their
account, cannot export their data, and Victory cannot demonstrate it would survive losing the
host.

Kernel 77 should therefore build a small number of missing things rather than harden existing
ones. Resist scope growth: the Medium and Low findings left open are genuinely lower priority
than the four items below.

---

## 1. Blockers — required for Level 4

### K77-01 — Account deletion (from K76-M04)

Deletion currently fails at the database level:
`actions_actor_id_fkey` blocks removing any user who has ever acted.

Build:
1. A migration giving every blocking foreign key an explicit disposition. The load-bearing
   decision: `actions.actor_id` becomes nullable with an `anonymized_at` marker, so authored
   Actions survive as shared Production history with no attributable author. Do **not**
   cascade-delete Actions — that would erase other people's Show record.
2. `POST /api/account/delete`, self-service, requiring re-authentication.
3. A blocking rule: a user who solely owns a Production or Show Run cannot delete until
   ownership transfers. Return the specific blocker, not a generic refusal.
4. Session revocation and credential destruction on deletion.

Per-record dispositions are mapped in `Construction/Security/kernel-76-data-classification.md`
§3. The controlling principle: a user may erase themselves but not other people's history of a
shared performance.

### K77-02 — Data export (from K76-M05)

`GET /api/account/export` producing one JSON document plus a files directory: identity,
profile and workbook, Characters, journals as Markdown, relationships and private notes,
messages sent and received, Show/Session participation, aftercare submissions, uploaded assets.

Never included: password hashes, session tokens, reset tokens, OAuth state, other users'
private content. Rate limit it — it is the most expensive endpoint Victory will have.

### K77-03 — Backup and restore

From `Construction/Operations/victory-backup-restore-assessment.md` §5:

1. Nightly `pg_dump` via systemd timer, 30-day retention.
2. Nightly `/opt/victory/storage` snapshot.
3. Encrypted off-host copy (`age` or `gpg` plus any second location). **Off-host is the item
   that changes the risk category** — everything currently lives on the machine being
   protected.
4. A restore actually performed, with elapsed time recorded.
5. A restore runbook covering database and assets together.

Target: recovery point ≤ 24h, recovery time ≤ 1h, both measured rather than asserted.

### K77-04 — Self-service recovery

Kernel 76 closed `/api/auth/password-reset/request` because its only delivery mechanism was
printing the token to the log. Restore it properly:

1. Email delivery (transactional provider, credentials in `.env`).
2. Reset revokes all existing sessions for the account.
3. Uniform response whether or not the address exists.
4. Rate limited per address as well as per IP.

Until this ships, recovery runs through `victory-recover` and the operator.

---

## 2. Strongly recommended

### K77-05 — Session revocation reaches open WebSockets

Revoking a session today prevents new connections but does not tear down sockets already open
under it. Add a periodic re-validation or a revocation broadcast to the hub.

### K77-06 — Rate limiting beyond credential endpoints

Only `/api/auth/*` and `/api/account/email` are throttled. Extend to message sending, note
cards, uploads, command execute, WebSocket connections and messages, and the new export
endpoint.

Two constraints from Kernel 76 §5.18: derive the client IP correctly behind Caddy rather than
trusting arbitrary forwarded headers, and ensure no limiter can lock the operator out without
recovery.

### K77-07 — Explicit CSRF posture

Protection currently rests implicitly on `SameSite=Lax` plus the fact that no GET route
mutates state. That is adequate but undocumented and easy to break by accident. Either adopt
a token for state-changing routes or add a test asserting no GET handler mutates.

### K77-08 — Reject WebSocket upgrades before completing them (K76-L01)

Authenticate before `upgrader.Upgrade` rather than after.

---

## 3. Worth doing, not blocking

- **K77-09** — pin `postgres:16-alpine` and `caddy:2-alpine` to digests.
- **K77-10** — decide whether to expose a deliberately minimal public `/health` (K76-I03).
- **K77-11** — `.claude/worktrees/agent-ad8c3df7c85e39ce3/` is a stale agent worktree still
  containing the old database credential. Delete it.
- **K77-12** — reconcile the two competing Kernel 75 spec files.

---

## 4. Explicitly out of scope

Victory Documents (Kernel 78 — but its Markdown security requirements are already written in
`kernel-76-data-classification.md` §5 and should be honoured there), payment processing,
public signup (Level 5), a general observability platform, end-to-end encryption, and any
map or Venue redesign.

---

## 5. Definition of done

Kernel 77 succeeds when a private paying client can be told, truthfully:

1. Only people you invite reach your Production.
2. Your private material is visible to you alone, including to the operator in ordinary use.
3. You can leave, and take your data with you.
4. If the server dies, your work comes back — and here is the measured window.

Items 1 and 2 are true today. Items 3 and 4 are what Kernel 77 buys.
