# Victory — Security Notes

**Current as of:** Kernel 76, 2026-07-31
**Readiness:** Level 3 — Invited strangers. Level 4 (private paying clients) is blocked by
the items in §5.
**Supersedes:** `Security-notes-kernel-2-historical.md`, which described the Kernel 2 posture
and remained the only security document through Kernel 75. Read it for historical rationale
only. Nothing in it should be treated as a current guarantee.

Full evidence: `Construction/Security/kernel-76-*.md`.

---

## 1. Verified current guarantees

Each of these was tested during Kernel 76, not merely read.

**Authentication**
- Session tokens are 32 random bytes, stored SHA-256-hashed. The raw token exists only in the
  cookie.
- Cookies: `HttpOnly`, `Secure`, `SameSite=Lax`, 24-hour TTL.
- Logout revokes server-side. A copied cookie is dead immediately after logout — verified live.
- Passwords use Argon2id (t=1, m=64 MiB, p=4, 16-byte salt, 32-byte key) with constant-time
  comparison.
- Discord OAuth state is hashed, expiring, and consumed atomically. Accounts key on the stable
  Discord user id, never the display name. A Discord identity cannot take over an existing
  account by matching its email.
- Reset tokens are hashed, 1-hour, single-use. Replay is rejected — verified live.

**Authorization**
- Anonymous requests reach no private Venue, Production, profile, Character, relationship,
  message, journal, or asset. Twenty-one routes were negative-tested.
- An authenticated outsider is rejected from every private Venue exactly as an anonymous one
  is — holding an account grants nothing.
- Every WebSocket endpoint authenticates and checks venue admission before subscription.
  Anonymous and outsider connections are rejected on all four.
- Actor identity and role are resolved server-side from the session. No route accepts
  `user_id`, `actor_id`, `role`, `location_id`, or `character_id` from the client as an
  authority claim.
- Invite authority is correctly scoped: Directors cannot create Directors.

**Content**
- Uploads are Producer-scoped, size-capped, magic-byte sniffed against a PNG/JPEG/WebP
  allowlist, and decoded then re-encoded with dimension caps. SVG is rejected. File paths are
  built from server-generated UUIDs, never client filenames.
- Request logging is method, path, and duration only. No private bodies in normal operation.

**Infrastructure**
- PostgreSQL binds `127.0.0.1` only. UFW exposes 22, 80, 443, 53 — not 5432, not 8081.
- Caddy proxies only `/api/*`, `/ws/*`, `/auth/*`.
- Secrets live in `/opt/victory/.env` (0600, gitignored). Nothing usable remains in the
  repository.

---

## 2. Repaired in Kernel 76

| ID | Severity | Was |
|---|---|---|
| K76-C01 | Critical | Raw password-reset tokens printed to the production log — account takeover by anyone who could read `docker logs` |
| K76-H01 | High | Open public registration, granting every stranger an active Location membership |
| K76-H02 | High | Third Place commons served real names and personal facts to any account |
| K76-H03 | High | `change_this_now` committed to git and in use as the live database credential |
| K76-M01 | Medium | Production list disclosed to every Location member including audience |
| K76-M02 | Medium | Discord gateway status served infrastructure state to anonymous callers |
| K76-M03 | Medium | No read/idle timeouts, header cap, body cap, or WebSocket frame limit |

All seven are deployed and re-verified against the live host.

---

## 3. Disabled surfaces

| Surface | State | Reversal |
|---|---|---|
| `POST /api/auth/signup` | `403 password_signup_closed` | set `PASSWORD_SIGNUP_ENABLED=true` |
| `POST /api/auth/password-reset/request` | `410` | Kernel 77, once email delivery exists |
| `GET /api/discord/gateway/status` for non-operators | `401`/`403` | intentional; not to be reversed |

`/api/auth/password-reset/confirm` remains live — it is how break-glass recovery tokens are
redeemed.

---

## 4. Account model

Discord establishes accounts. Email/password **login** is retained as the recovery path so
that losing Discord access does not destroy account ownership; email/password **registration**
is closed.

Operator authority is `handle == OPERATOR_HANDLE` (`straturli`) — there is no database flag,
which keeps operator status visible in the `users` table and auditable.

Recovery of last resort is `backend/cmd/victory-recover`, documented in
`Construction/Operations/victory-account-recovery-runbook.md`. It runs from the server shell,
needs no Victory login, cannot read private content, and cannot create a hidden superuser.

---

## 5. Open gaps blocking Level 4

| ID | Gap |
|---|---|
| K76-M04 | Account deletion is structurally impossible — `actions.actor_id` blocks it and no route exists |
| K76-M05 | No self-service data export |
| — | No scheduled, off-host, or encrypted backup; no tested restore; recovery point currently unbounded |
| K77-03 | Revoking a session does not disconnect already-open WebSockets |
| K77-04 | No rate limiting outside credential endpoints |
| K76-L01 | WebSocket upgrade completes before authentication |

---

## 6. Operator privacy

Ordinary operator tooling does not render journals, relationship notes, message bodies, or
unpublished profiles, and logs do not reproduce them. Root and database access remain
available to the server owner by necessity; the practical guarantee is that reaching private
content requires a deliberate, exceptional act rather than a routine one.

---

## 7. Rule for future kernels

Victory's authorization failures have all had the same shape: code asking *"are you signed
in?"* where it should ask *"are you admitted to this room?"* Both leaks of real user data
found in Kernel 76 (K76-H02, K76-M01) were that mistake.

When adding a read endpoint, the question is not whether the caller is authenticated. It is
which Venue, Production, or ownership boundary the data sits behind, and whether the caller is
inside it.
