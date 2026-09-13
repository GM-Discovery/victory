# Kernel 76 — Finding Ledger

**Audit date:** 2026-07-30 / 2026-07-31
**Baseline commit:** `97c71697d0bd51d4c28307515fef16a5d6f9b3a7` (branch `main`)
**Target:** victory.amurray.family (live), plus `victory_k76_clean` scratch database
**Method:** code read of `backend/cmd/victory/main.go` route table and handlers, live negative
testing against the public host with real isolated accounts, WebSocket client tests, database
inspection.

Severity uses the Kernel 76 §7.7 scale. "Repaired" means the fix is deployed to the live
container and re-verified against the public host, not merely written.

---

## Summary

| Severity | Open | Repaired | Deferred to K77 |
|---|---|---|---|
| Critical | 0 | 1 | 0 |
| High | 0 | 3 | 0 |
| Medium | 0 | 3 | 2 |
| Low | 2 | 0 | 1 |
| Informational | 3 | 0 | 0 |

No unmitigated Critical finding remains. No High finding enabling account takeover,
cross-client data access, or arbitrary code/file compromise remains open.

---

## K76-C01 — Raw password-reset tokens written to the production log

- **Severity:** Critical
- **Surface:** `backend/internal/identity/auth.go`, `HandleForgotPassword`
- **Status:** Repaired (surface closed)

**Evidence.** The handler ended with:

```go
log.Printf("PASSWORD RESET TOKEN for %s: %s", email, rawToken)
```

The token stored in `auth.password_reset_tokens` is a SHA-256 hash, but the raw value — the
thing that actually redeems the reset — was printed verbatim to stdout, which Docker captures
and retains for the life of the container.

**Failure scenario.** `POST /api/auth/password-reset/request {"email":"grant@amurray.family"}`
is unauthenticated and needs no proof of address ownership. Anyone who could read
`docker logs victory-backend` — any process on the host, any future log shipper, anyone
handed logs during support — could take over any account by naming its email address. The
old `Security-notes.md` disclosed this as a temporary development measure; Kernel 76 §5.5
predicted it would still be live, and it was.

**Repair.** Victory has no email delivery, so the flow's only delivery mechanism *was* the
log line. `HandleForgotPassword` now returns `410 self_service_password_reset_unavailable`.
Reset tokens are minted by the operator-only break-glass tool and redeemed at the still-live
`/api/auth/password-reset/confirm`. Self-service recovery returns in Kernel 77 with a real
delivery channel.

**Retest.** `POST /api/auth/password-reset/request` → `410`. No occurrence of the token log
line remains in the source tree. Regression test:
`TestSelfServicePasswordResetIsClosed`.

---

## K76-H01 — Open public registration on an internet-facing host

- **Severity:** High
- **Surface:** `backend/internal/identity/auth.go`, `HandleSignup`; `POST /api/auth/signup`
- **Status:** Repaired (closed by default)

**Evidence.** Two accounts were created from an unauthenticated shell against the live public
host with no invite and no approval:

```
k76_alice signup=200
k76_bob   signup=200
{"data":{"handle":"k76_alice","is_operator":false,"role":"audience", ...},"signed_in":true}
```

The handler also grants every new account an active `audience` membership in the
`amurray-family` Location.

**Failure scenario.** Victory was, in practice, already running public signup. A stranger
could self-issue an account, receive a real Location membership, and reach every surface
gated on "is authenticated" rather than "is admitted" — which included the Third Place
roster (K76-H02) and the Production list (K76-M01).

**Repair.** Password signup is closed unless `PASSWORD_SIGNUP_ENABLED` is explicitly truthy;
nothing in the deployed configuration sets it. Account establishment goes through Discord.
`/signup/` now presents the Discord button and states plainly that email/password
registration is closed, rather than showing a form that fails.

**Retest.** `POST /api/auth/signup` → `403 password_signup_closed`. Regression tests:
`TestPasswordSignupClosedByDefault`, `TestPasswordSignupReopensForLocalDevelopment`.

---

## K76-H02 — Third Place commons exposed real-world identities to any account

- **Severity:** High
- **Surface:** `backend/internal/thirdplace/http.go`, `HandleCollection`;
  `GET /api/third-place/headshots`
- **Status:** Repaired

**Evidence.** The handler required only `requireAuthenticatedUser`. Read with the
freshly-created `k76_alice` account, which had no Trailer Face, no Production, and no
Show Run:

```json
{"headshots":[{"stage_name":"Grant A. Murray",
  "headline_facts":[{"label":"D&D Class","value":"Bard"},
                    {"label":"Socio Class / Archetype","value":"Prodigy"}], ...}]}
```

A source comment recorded the intent: *"Viewing the commons list, one's own current
Headshot, or history is left open."*

**Failure scenario.** Combined with K76-H01 this was a public endpoint returning a roster of
real people with legal names and personal facts. Even with signup closed it was wrong for
Level 4: any stranger with a Discord account would have enumerated every member.

**Repair.** Reading the commons now requires the same Trailer Face readiness that entering
it requires (Kernel 68 §3.2), via a shared `requireCommonsAdmission` helper. You can see the
room once you are in the room.

**Retest.** As an authenticated Producer with no ready Trailer Face:
`GET /api/third-place/headshots` → `403 {"error":"trailer_face_not_ready"}`.

---

## K76-H03 — Committed default database password in production use

- **Severity:** High
- **Surface:** `docker-compose.yml`; PostgreSQL role `victory`
- **Status:** Repaired (rotated)

**Evidence.** `docker-compose.yml`, tracked in git, contained
`POSTGRES_PASSWORD: change_this_now` and a `DATABASE_URL` embedding the same literal. That
value was the live credential, not a placeholder. Kernel 76 §5.15 names this string
explicitly.

**Failure scenario.** The production database credential was readable by anyone with
repository access and was a guessable default. PostgreSQL is bound to `127.0.0.1:5432` and
UFW does not expose 5432, so remote exploitation required host access first — which is what
keeps this High rather than Critical.

**Repair.** The role password was rotated to a 32-character random value. `docker-compose.yml`
now reads `${POSTGRES_PASSWORD:?...}` from `.env`, which is gitignored and `chmod 600`.
`.env.example`, `dev-workflow.md`, and `kernel-maker-field-guide.md` were updated to
reference the variable. Historical reportbacks retain the old string as a record of what was
true at the time.

**Retest.** Backend reconnected and served after rotation; `git grep change_this_now` returns
only historical reportbacks and a stale `.claude/worktrees/` copy.

---

## K76-M01 — Production list disclosed to every Location member

- **Severity:** Medium
- **Surface:** `backend/internal/identity/permissions.go`, `handleListProductions`
- **Status:** Repaired

**Evidence.** `resolveInviteAuthorityScope` resolves a Location for *any* membership role
including `audience`, so `GET /api/productions` returned every Production name, slug, and id
in the Location. Read with the fresh `k76_alice` audience account:

```json
[{"name":"Main Production","slug":"kernel-7-production", ...},
 {"name":"K69 Socio Test Production 1783981152", ...}, ...]
```

**Failure scenario.** Any member — under K76-H01, any stranger — enumerated the Location's
Production catalogue. Location is Victory's tenant boundary, so this did not cross between
clients, but within a Location it disclosed staff-side structure to the audience.

**Repair.** Listing now requires `producer` or `director`.

**Retest.** Audience/no-role → `403`; Producer → `200`.

---

## K76-M02 — Unauthenticated infrastructure status endpoint

- **Severity:** Medium
- **Surface:** `backend/internal/identity/discord_gateway.go`,
  `HandleDiscordGatewayStatus`; `GET /api/discord/gateway/status`
- **Status:** Repaired

**Evidence.** Anonymous request to the live host returned:

```json
{"configured":true,"enabled":true,"running":false,"connected":false,
 "intents":641,"active_thread_count":0,
 "last_error":"websocket: close 4004: Authentication failed."}
```

**Failure scenario.** An operations readout of Grant's infrastructure — including a verbatim
upstream error revealing that the bot token was failing authentication — served to the public
internet. Useful reconnaissance for timing an attack against a known-degraded component.

**Repair.** The route now requires an authenticated operator.

**Retest.** Anonymous → `401`; non-operator → `403`; operator → `200`. Regression tests:
`TestDiscordGatewayStatusRejectsAnonymousCallers`,
`TestDiscordGatewayStatusRejectsNonOperator`.

---

## K76-M03 — No transport limits: unbounded bodies, headers, idle connections, WS frames

- **Severity:** Medium
- **Surface:** `backend/cmd/victory/main.go` (`http.Server`);
  `backend/internal/network/ws.go` (`readPump`)
- **Status:** Repaired

**Evidence.** The server set only `ReadHeaderTimeout: 5s` — no `ReadTimeout`, `IdleTimeout`,
or `MaxHeaderBytes`. Every JSON handler decoded from an unbounded `r.Body`; only the three
upload handlers applied `http.MaxBytesReader`. Venue WebSockets never called `SetReadLimit`.

**Failure scenario.** A single client could hold connections open indefinitely, stream an
arbitrarily large JSON body into a decoder, or push an unbounded WebSocket frame into server
memory. No authentication needed for the HTTP paths.

**Repair.** `ReadTimeout 60s`, `IdleTimeout 120s`, `MaxHeaderBytes 64 KiB`; a
`requestBodyLimit` middleware caps non-upload, non-WebSocket bodies at 4 MiB (upload handlers
still set their own larger per-Location limit afterwards); venue sockets cap a frame at
256 KiB. `WriteTimeout` is deliberately left unset because it would sever long-lived
WebSocket upgrades sharing this server.

---

## K76-M04 — Account deletion is structurally impossible

- **Severity:** Medium
- **Surface:** database foreign keys; no deletion route exists
- **Status:** Open — deferred to Kernel 77

**Evidence.** No `/api/account/delete` or export route exists anywhere in the tree. Attempting
the deletion directly during the account purge failed:

```
ERROR: update or delete on table "users" violates foreign key constraint
"actions_actor_id_fkey" on table "actions"
```

**Failure scenario.** A paying client asking for deletion cannot be served without hand-written
SQL, and the attempt fails partway, risking inconsistent state. Kernel 76 §1.6 requires a
defined and implementable path before private paying clients.

**Disposition.** Kernel 77. The dependency map and per-class disposition (hard delete /
anonymize / detach / preserve-with-anonymous-author) is in
`kernel-76-data-classification.md`. `actions.actor_id` is the load-bearing case: authored
Actions are shared Production history and should survive with an anonymised author rather
than cascade-delete.

---

## K76-M05 — No self-service data export

- **Severity:** Medium
- **Surface:** none — the capability is absent
- **Status:** Open — deferred to Kernel 77

Kernel 76 §1.6 requires a defined path for a user to export their own data. Nothing exists.
Scope for the minimum viable export (JSON plus files) is in the Kernel 77 proposal.

---

## K76-L01 — WebSocket upgrade completes before authentication

- **Severity:** Low
- **Surface:** `backend/internal/network/ws.go`, `ServeVenueWS` / `ServeCaveWS`
- **Status:** Open

The handler calls `upgrader.Upgrade` first, then checks the session and venue admission,
writing `{"type":"error","error":"forbidden"}` and closing. Authorization is correctly
enforced — no unauthorized subscription occurs — but an anonymous client can still drive a
full WebSocket handshake before rejection, which is cheap flood capacity. Rejecting before
upgrade would be tidier. Not a data-exposure issue.

---

## K76-L02 — Stale host development process shared the live database

- **Severity:** Low
- **Surface:** operational
- **Status:** Repaired (process stopped)

A `go run ./cmd/victory` from 2026-07-28 was still listening on `:18081` against the live
database, alongside the container Caddy actually proxies (`backend:8081`). Two builds of
different vintage were writing to production data, and it blocked the database rebuild until
terminated. Kernel 76 §4.4 asks for exactly this to be resolved before attributing behaviour
to current code.

---

## K76-I01 — Positive findings worth recording

These were tested and found sound; they are recorded so a later kernel does not re-litigate
them.

- **Session handling.** Tokens are 32 random bytes, stored SHA-256-hashed, never raw. Cookies
  are `HttpOnly`, `Secure`, `SameSite=Lax`. Logout revokes server-side — a copied cookie is
  dead after logout (verified live). Reset consumes its token; replay returns
  `invalid_or_expired_token` (verified live).
- **Discord OAuth.** State is hashed, single-use, and expiring, consumed by an atomic
  `UPDATE ... WHERE consumed_at IS NULL AND expires_at > NOW() RETURNING`. Accounts key on the
  stable Discord user ID, never the display name. A Discord identity cannot silently take over
  an existing account by matching its email address.
- **Uploads.** Producer-scoped, `MaxBytesReader`-capped, magic-byte sniffed against a
  PNG/JPEG/WebP allowlist, decoded and re-encoded with dimension caps. SVG is rejected. This
  is the strongest subsystem in the audit.
- **WebSocket authorization.** Anonymous and outsider connections are rejected on every
  private venue socket; `CheckOrigin` is a strict same-host comparison and rejects an empty
  Origin. Proven with a Go client — see `kernel-76-websocket-matrix.md`.
- **Invite authority.** Correctly scoped: Producers may invite any role, Directors may not
  create Directors, and the target role is validated against an allowlist.
- **Logs.** Request logging is method, path, and duration only. No private bodies, journals,
  messages, or credentials appear in normal operation (K76-C01 was the sole exception).
- **Database exposure.** PostgreSQL binds `127.0.0.1:5432` only; UFW allows 22/80/443/53 and
  does not expose 5432.

---

## K76-I02 — Discord bot gateway token was failing authentication

At audit start the gateway reported `websocket: close 4004: Authentication failed.` After the
container rebuild it reconnected (`discord gateway dial connected`, `identifying
intents=641 token_set=true`). Recorded because it was visible in the pre-repair evidence and
explains the `last_error` string in K76-M02.

---

## K76-I03 — `/health` is not reachable through Caddy

`GET https://victory.amurray.family/health` → `404`. The Caddyfile proxies `/api/*`, `/ws/*`,
and `/auth/*` only; `/health` is served on the container port and is not exposed publicly.
This is safe, and arguably correct, but it means external uptime monitoring has no endpoint
to poll. Kernel 77 should decide whether to expose a deliberately minimal public health route.
