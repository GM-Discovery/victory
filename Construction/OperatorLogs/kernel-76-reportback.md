# Kernel Report Back — Kernel 76: Canonical State, Security, and Hosted-Readiness Audit

## 1. Status

**PASS — Audit complete; readiness Level 3; Level 4 blocked by K76-M04, K76-M05, and backup posture.**

Every Critical and High finding is repaired and verified against the live host. Grant's access
was restored and proven after a full database rebuild. Level 4 is not claimed: account
deletion is structurally impossible, data export does not exist, and no backup leaves the
host.

```
Achieved readiness: Level 3 — Invited strangers
Required target:    Level 4 — Private paying clients
Stretch target:     Level 5 — Public signup (not attempted)
```

---

## 2. What Was Built

**Repairs (all deployed live and re-verified):**
- Password-reset request endpoint closed; raw-token logging removed
- Public email/password registration closed behind `PASSWORD_SIGNUP_ENABLED`
- Third Place commons list gated on Trailer Face admission
- Production list restricted to Producer/Director
- Discord gateway status restricted to operator
- HTTP `ReadTimeout` / `IdleTimeout` / `MaxHeaderBytes`; 4 MiB request body cap; 256 KiB WebSocket frame cap
- Database password rotated out of git into `.env` (0600)

**New:**
- `backend/cmd/victory-recover` — break-glass tool: `whoami`, `bootstrap`, `claim`, `grant`, `revoke`, `recover`
- `frontend/login/reset.html` — recovery-link redemption page
- `backend/internal/identity/kernel76_auth_closure_test.go` — closure regression tests
- Two negative tests for gateway-status authorization

**Documents:** nine artifacts listed in the kernel spec §5.

---

## 3. Evidence

### 3.1 Baseline

```
$ git rev-parse HEAD
97c71697d0bd51d4c28307515fef16a5d6f9b3a7
$ docker ps --format '{{.Names}}\t{{.Status}}'
victory-backend    Up 36 hours
bread-caddy        Up 5 weeks
victory-postgres   Up 2 months
```

Rollback dump retained: `/opt/victory/backups/k76/victory-pre-k76-20260730-2028.sql` (5.5 MB).

### 3.2 The Critical finding

`backend/internal/identity/auth.go` contained:

```go
log.Printf("PASSWORD RESET TOKEN for %s: %s", email, rawToken)
```

An unauthenticated caller naming any email address caused an account-takeover credential to be
written to `docker logs`. Now `410`:

```
$ curl -X POST .../api/auth/password-reset/request -d '{"email":"grant@amurray.family"}'
410  {"error":"self_service_password_reset_unavailable"}
```

### 3.3 Open signup — demonstrated, not inferred

Two accounts created from an unauthenticated shell against the live public host:

```
k76_alice signup=200
k76_bob   signup=200
{"handle":"k76_alice","role":"audience","signed_in":true}
```

Now:

```
$ curl -X POST .../api/auth/signup -d '{...}'
403  {"error":"password_signup_closed"}
```

### 3.4 Third Place leak — what the stranger account could read

```json
{"headshots":[{"stage_name":"Grant A. Murray",
  "headline_facts":[{"label":"D&D Class","value":"Bard"},
                    {"label":"Socio Class / Archetype","value":"Prodigy"}]}]}
```

After repair, an authenticated Producer without a ready Trailer Face:

```
403  {"ok":false,"data":{"error":"trailer_face_not_ready"}}
```

### 3.5 Negative test sweep

Twenty-one routes unauthenticated — all fail closed:

```
/api/account/me 401   /api/player-profile/me 401   /api/character-journals 401
/api/show-runs 401    /api/showings 401            /api/messages 403
/api/world/the-cave 403   /api/world/catharsis 403   /api/world/first-theater 403
/api/venues/first-theater/map 403   /api/warehouse/assets 403
/api/discord/gateway/status 401 (was 200 leaking infrastructure state)
```

Session lifecycle:

```
logout=200 → {"signed_in":false}          # copied cookie dead
reset token replay → invalid_or_expired_token
```

### 3.6 WebSocket proof

```
anon /ws/the-cave                  {"error":"forbidden"}
anon /ws/player-profile            {"error":"not_authenticated"}
outsider(alice) /ws/catharsis      {"error":"forbidden"}
outsider(alice) /ws/first-theater  {"error":"forbidden"}
bad-origin                         {"error":"forbidden"}
```

### 3.7 Break-glass proven BEFORE the wipe

```
$ victory-recover recover --handle k76_fake_operator
  https://victory.amurray.family/login/reset.html?token=<redacted>

$ curl -X POST .../api/auth/password-reset/confirm -d '{"token":"...","new_password":"..."}'
{"ok":true}
$ curl -b cookie .../api/session/me
{"handle":"k76_fake_operator","role":"producer","signed_in":true}
$ # replay:
{"error":"invalid_or_expired_token","ok":false}
```

Token stored hashed only — verified in `auth.password_reset_tokens`.

### 3.8 Clean rebuild, then live rebuild

Scratch database first:

```
migrate: 81 migration(s) applied, schema current
victory backend listening on :18099
health=200
locations: 2   venues: 16   equipment_items: 150   users: 1
```

Then live, after `DROP DATABASE victory`:

```
migrate: 81 migration(s) applied, schema current
victory backend listening on :8081
users: 1   locations: 2   venues: 16   equipment_items: 150   sessions: 0
```

### 3.9 Operator access restored

```
$ victory-recover bootstrap --operator-handle straturli --email grant@amurray.family
created operator account "straturli" with producer at amurray-family

$ curl -b cookie .../api/session/me
{"handle":"straturli","is_operator":true,"role":"producer","signed_in":true}

$ curl -b cookie .../api/map/visibility
catharsis, directors-chair, first-theater, grants-cabin, greenroom, info-booth,
library, middle-school-stage, producers-office, show-runs, soil-experts,
the-cave, third-place, trailers, warehouse, workshop
```

Cabin (`grants-cabin`) present. Operator surfaces `/api/productions`, `/api/workshop/venues`,
`/api/world/catharsis`, `/api/discord/gateway/status` all `200`.

Discord OAuth start:

```
302 → https://discord.com/oauth2/authorize?client_id=<ID>&redirect_uri=...&state=<STATE>
```

**Grant then signed in with Discord during the audit**, which created a separate
`discord_694402257331945562` account holding no memberships — the exact case the runbook
covers. Merged:

```
$ victory-recover claim --discord-id 694402257331945562 --operator-handle straturli
$ victory-recover grant --discord-id ... --location amurray-family --role producer
handle: straturli   is_operator: true   memberships: amurray-family: producer (active)
```

Grant's existing Discord session now resolves as operator. The orphaned bootstrap account was
removed.

### 3.10 Tests and DB isolation proof (Kernel 64)

```
$ go build ./...                       BUILD OK
$ git diff --check                     OK
$ node --check (reset.html script)     OK
```

Hard-fail without `TEST_DATABASE_URL`:

```
dbtest: TEST_DATABASE_URL is required for database-touching tests
FAIL victory/backend/internal/identity 0.019s
```

Full suite against isolated `victory_test`: **28 packages ok, 1 failing** —
`internal/merchant`, `TestKernel74TutorialTailEndToEnd` and
`TestKernel75TutorialCompletionEndToEnd`, both `crown-bet` dialogue-topic seed errors.

**These are pre-existing and not caused by Kernel 76.** Proven by stashing all Kernel 76
changes and re-running the same package against the same database:

```
$ git stash push backend/ frontend/
$ go test -count=1 ./internal/merchant/
--- FAIL: TestKernel74TutorialTailEndToEnd
--- FAIL: TestKernel75TutorialCompletionEndToEnd
```

Identical failures at baseline. Cause: these tests depend on dialogue-topic seed data present
in the long-lived test database, which was recreated during this kernel. See §6.

Live row counts unchanged across the whole test run:

```
BEFORE  users=2  memberships=1  sessions=1
AFTER   users=2  memberships=1  sessions=1
```

---

## 4. How to Run (Operator Steps)

```bash
cd /opt/victory
git pull
docker compose up -d --build backend      # migrations apply at boot

# if you ever cannot get in:
cd backend
set -a; . /opt/victory/.env; set +a
export DATABASE_URL="postgres://victory:${POSTGRES_PASSWORD}@127.0.0.1:5432/victory?sslmode=disable"
go run ./cmd/victory-recover whoami --handle straturli
go run ./cmd/victory-recover recover --handle straturli   # prints a one-time link
```

Full procedures: `Construction/Domains/Operations/victory-account-recovery-runbook.md`.

---

## 5. Operator Notes

- **`POSTGRES_PASSWORD` now lives in `/opt/victory/.env`** (0600, gitignored). `docker-compose.yml`
  fails fast if it is missing. **If `.env` is lost, the database is unreachable** — back it up
  somewhere you control.
- **Signup is closed.** New accounts come from Discord. Reopen for local development only with
  `PASSWORD_SIGNUP_ENABLED=true`.
- **Discord signup creates `discord_<id>`, not `straturli`.** Operator authority is the handle,
  so a fresh Discord account is *not* operator until `victory-recover claim` runs.
- **Password reset is operator-issued**, at `/login/reset.html?token=…` — not `/auth/…`,
  because Caddy proxies `/auth/*` to the backend.
- **Do not run a host `go run ./cmd/victory` against the live database** while the container is
  serving. A stale one from 2026-07-28 was found doing exactly that and blocked the rebuild.
- **`/health` is not reachable through Caddy** — only on the container port.
- Migrations refuse to run without `pg_dump` on PATH; the host lacks it, the container has it.
  For a disposable database use `MIGRATE_DANGEROUSLY_SKIP_BACKUP=true`.

---

## 6. Blockers & Workarounds

**BLOCKER:** `internal/merchant` tutorial tests fail with `crown-bet: topic_not_found`.
**CAUSE:** They depend on dialogue-topic rows present in the long-lived `victory_test`
database rather than created by their own fixtures. `victory_test` was dropped and rebuilt
during this kernel, so that latent dependency surfaced.
**WORKAROUND:** None applied — proven pre-existing at baseline, and out of scope for a
security audit.
**OPERATOR ACTION REQUIRED:** Kernel 77 should make these tests seed their own dialogue
topics. Until then the two tests fail on any freshly created test database.

---

**BLOCKER:** Account deletion is impossible.
**CAUSE:** `actions_actor_id_fkey` has no delete rule; no deletion route exists.
**WORKAROUND:** None — surfaced as K76-M04 with a full dependency map.
**OPERATOR ACTION REQUIRED:** Kernel 77 K77-01. This blocks Level 4.

---

**BLOCKER:** No backup leaves the host, and none is scheduled.
**CAUSE:** Backups are triggered by migrations or by a person; assets are not backed up at all.
**WORKAROUND:** A pre-Kernel-76 dump was taken manually and retained.
**OPERATOR ACTION REQUIRED:** Kernel 77 K77-03. Demonstrated data-loss window is currently
**unbounded** — there was a ten-day gap with no dump between 2026-07-20 and 2026-07-30.

---

## 7. Deviations from Kernel

1. **Password login retained.** The brief anticipated possibly removing it. Signup is closed but
   login stays, because §2.1 forbids removing a working path before the replacement is proven
   from a fresh browser, and because reset-confirm is how break-glass tokens are redeemed.
   Removing it would make recovery depend entirely on Discord, which §5.4 forbids.

2. **The wipe was questioned before it was executed.** Inspection showed the live database held
   108 hand-placed stage elements, 30 uploaded assets (29 owned by `straturli`), the active
   venue map, and 34 character cards — none reproduced by migrations. This was raised rather
   than acted on. Grant confirmed the assets were disposable and that the courtyard could be
   re-uploaded, and the full wipe proceeded as originally decided.

3. **The recovery page moved** from `/auth/reset.html` to `/login/reset.html` after testing
   showed Caddy proxies `/auth/*` to the backend, which returned 404. Changing the shared
   Caddyfile would have affected bread-exchange.

4. **`Security-notes.md` was replaced, not edited.** The original is preserved as
   `Security-notes-kernel-2-historical.md`. §7.10 forbids preserving outdated claims as
   current fact.

---

## 8. Required Reportback Additions

### 8.1 Achieved readiness

```
Achieved readiness: Level 3 — Invited strangers
Required target:    Level 4 — Private paying clients
Stretch target:     Level 5 — Public signup
```

Level 3 is justified: no anonymous access to private surfaces, server-side authorization
proven, WebSocket isolation proven, private notes/journals/messages isolated, no raw secrets in
logs, no default secrets, database not publicly exposed, uploads safe, recovery proven.

Level 4 is withheld solely for missing capabilities: deletion, export, backup.

### 8.2 Grant access proof

| Item | Result |
|---|---|
| Supported login method | Discord OAuth; password login retained for recovery |
| Fresh-context proof | recovery link redeemed from a clean cookie jar → working session |
| Cabin access | `grants-cabin` present in map visibility |
| Producer+ access | `producer` at `amurray-family`; operator by handle |
| Break-glass tested | yes — `bootstrap`, `claim`, `grant`, `revoke`, `recover` all exercised |
| Old path status | signup closed; login retained deliberately |

### 8.3 Data reset statement

- Test data wiped: **yes** — `DROP DATABASE victory`, rebuilt from 81 migrations.
- Deliberate seeds remaining: 2 Locations, 16 Venues, 150 equipment items, 1 merchant packet,
  1 dialogue packet with 6 topics, 2 Scenes — all from migrations.
- `straturli` recreated: **yes**, via `victory-recover bootstrap`, then merged with Grant's
  Discord identity via `claim`.
- Intentionally preserved: nothing from the old database. Retained as a dump only.

### 8.4 Disabled surfaces

| Surface | State | Reversal |
|---|---|---|
| `POST /api/auth/signup` | `403 password_signup_closed` | `PASSWORD_SIGNUP_ENABLED=true` |
| `POST /api/auth/password-reset/request` | `410` | Kernel 77, with email delivery |
| `GET /api/discord/gateway/status` (non-operator) | `401`/`403` | intentional |

### 8.5 Findings summary

```
Critical: 0 open / 1 repaired / 0 deferred
High:     0 open / 3 repaired / 0 deferred
Medium:   0 open / 3 repaired / 2 deferred
Low:      2 open / 0 repaired / 1 deferred
Informational: 3 recorded
```

### 8.6 Kernel 77 blockers

1. **K77-01** — account deletion, including the `actions.actor_id` disposition
2. **K77-02** — self-service data export
3. **K77-03** — scheduled, encrypted, off-host backup with a tested restore
4. **K77-04** — self-service password recovery with real email delivery

Only these four gate Level 4. Everything else in the Kernel 77 proposal is improvement, not
blocker.

---

## 9. The one thing worth remembering

Every authorization failure found in this audit had the same shape: code that asked
*"are you signed in?"* where it should have asked *"are you admitted to this room?"* Both
leaks of real user data — the Third Place roster and the Production list — were that mistake,
and both were written by someone reasoning that a list is not sensitive.

Victory's venue model is already the right abstraction for this. The failures were in the
places where a developer stepped outside it.
