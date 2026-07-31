# Kernel 76 — HTTP Route Matrix

**Source of truth:** `backend/cmd/victory/main.go` lines 143–937, enumerated from code, not
documentation. **221 route registrations.**
**Live evidence:** response codes are real, taken against `https://victory.amurray.family`
with four identities described in §1. Codes marked *(post-repair)* were re-taken after the
Kernel 76 container rebuild.

---

## 1. Test identities used

| Identity | How established | Authority held |
|---|---|---|
| Anonymous | no cookie | none |
| Fresh account (`k76_alice`, `k76_bob`) | created via the then-open `/api/auth/signup` | `audience` at `amurray-family`, no Trailer Face, no Production |
| Producer | recovery-link redemption + `grant --role producer` | `producer` at `amurray-family` |
| Operator (`straturli`) | `bootstrap` + recovery link on the rebuilt database | operator (handle match) + `producer` |

The fresh account is the important one: it is what a stranger got, and it is the lens that
found K76-H01, K76-H02, and K76-M01.

---

## 2. Global controls

| Control | Value | Applies to |
|---|---|---|
| Credential rate limit | per-IP token bucket, burst 10, refill 10/min | `/api/auth/signup`, `/api/auth/login`, both password-reset routes, `/api/account/email` |
| Request body cap | 4 MiB *(added K76-M03)* | every route except multipart uploads and WebSocket upgrades |
| Upload body cap | per-Location `MaxUploadBytes` via `MaxBytesReader` | the three `/api/workshop/assets*` and warehouse upload paths |
| Header cap | 64 KiB *(added K76-M03)* | all |
| `ReadHeaderTimeout` / `ReadTimeout` / `IdleTimeout` | 5s / 60s / 120s *(last two added K76-M03)* | all |
| Session cookie | `victory_session`, `HttpOnly`, `Secure`, `SameSite=Lax`, 24h TTL | all authenticated routes |

Ordinary API routes are deliberately unthrottled beyond the body cap. See K77-04.

**CSRF posture.** Victory is cookie-authenticated with no CSRF token. `SameSite=Lax` blocks
cross-site POST/PUT/DELETE, which covers every state-changing route, and `CheckOrigin` on the
WebSocket upgrade is a strict same-host comparison. Lax does permit top-level cross-site GET
navigation, so the protection depends on no GET route being state-changing — verified true by
inspection of the route table, where every mutation is POST/PATCH/PUT/DELETE. This is
adequate but implicit; making it explicit is K77-05.

---

## 3. Public by intent

| Method | Path | Auth | Live result | Notes |
|---|---|---|---|---|
| GET | `/health` | none | `200` on container port, `404` via Caddy | not proxied — see K76-I03 |
| GET | `/api/session/me` | none | `200 {"signed_in":false}` | correct: discloses nothing when signed out |
| GET | `/api/auth/providers` | none | `200 {"discord_enabled":true}` | needed to render the login page |
| POST | `/api/auth/login` | none | `401` on bad credentials | rate limited |
| GET | `/auth/discord/start` | none | `302` to `discord.com/oauth2/authorize` with hashed single-use `state` | |
| GET | `/auth/discord/callback` | none | `400 invalid_or_expired_state` without valid state | |
| GET | `/api/map/visibility` | none | `200 {"data":[]}` | empty for anonymous; correct |

---

## 4. Closed surfaces

| Method | Path | Before | After | Finding |
|---|---|---|---|---|
| POST | `/api/auth/signup` | `200` + session + `audience` membership | `403 password_signup_closed` | K76-H01 |
| POST | `/api/auth/password-reset/request` | `200`, raw token to log | `410 self_service_password_reset_unavailable` | K76-C01 |
| POST | `/api/auth/password-reset/confirm` | live | **still live** — redeems break-glass tokens | K76-C01 |

---

## 5. Negative test results

Every row is an observed response code, not an inference.

### 5.1 Unauthenticated

| Path | Code | Verdict |
|---|---|---|
| `/api/account/me` | `401` | pass |
| `/api/player-profile/me` | `401` | pass |
| `/api/player-relationships` | `401` | pass |
| `/api/character-cards/me` | `401` | pass |
| `/api/character-journals` | `401` | pass |
| `/api/show-runs` | `401` | pass |
| `/api/scenes` | `401` | pass |
| `/api/showings` | `401` | pass |
| `/api/tickets/mine` | `401` | pass |
| `/api/third-place/headshots` | `401` | pass |
| `/api/commands/available` | `401` | pass |
| `/api/workshop/venues` | `401` | pass |
| `/api/messages` | `403` | pass |
| `/api/warehouse/assets` | `403` | pass |
| `/api/world/the-cave` | `403` | pass |
| `/api/world/catharsis` | `403` | pass |
| `/api/world/first-theater` | `403` | pass |
| `/api/venues/first-theater/map` | `403` | pass |
| `/api/discord/gateway/debug` | `401` | pass |
| `/api/discord/gateway/status` | `401` *(post-repair; was `200` with infrastructure state)* | K76-M02 |
| `/api/profiles/admin/save` | `410` | retired route |

No anonymous path reached a private Venue, membership list, profile, Character, relationship,
message, journal, or asset.

### 5.2 Authenticated outsider — the fresh account

| Path | Code | Verdict |
|---|---|---|
| `/api/world/{the-cave,catharsis,first-theater}` | `403` | pass |
| `/api/workshop/venues` | `403` | pass |
| `/api/showings` | `403` | pass |
| `/api/warehouse/assets` | `403` | pass |
| `/api/venues/first-theater/map` | `403` | pass |
| `/api/discord/gateway/debug` | `403` | pass |
| `/api/show-runs` | `200 {"show_runs":null}` | pass — empty, not forbidden |
| `/api/messages` | `200 {"messages":null}` | pass — empty |
| `/api/player-relationships` | `200 {"relationships":[]}` | pass — own scope only |
| `/api/player-profile/{other_user_id}` | `{"ok":false}` | pass |
| `/api/third-place/headshots` | `200` **leaked real names** → `403 trailer_face_not_ready` | **K76-H02** |
| `/api/productions` | `200` **full Production list** → `403` | **K76-M01** |
| `POST /api/invites {"role":"producer"}` | `400 target_role_required`, then `403 producer_membership_required` | pass — no privilege escalation |
| `/api/messages/1` | `400` | pass — malformed id rejected |

The map visibility resolver returned only `audition-hall` and `trailers` to the fresh account,
both marked `visible_because: authenticated_surface` — correct fail-closed behaviour, and the
reason the two leaks above stood out as anomalies rather than policy.

### 5.3 Client-supplied identity

`POST /api/profiles/admin/save` with `{"user_id":"00000000-…"}` → `410` (route retired).
No handler in the route table accepts `user_id`, `actor_id`, `role`, `location_id`, or
`character_id` from the client as an authority claim; identity is resolved from the session
cookie in every case checked. `handleCreateProduction` carries an explicit source comment that
`location_id` is always resolved server-side, with an Operator-only exception that still
resolves a slug to an id rather than trusting it verbatim.

### 5.4 Session lifecycle

| Test | Result |
|---|---|
| Copied cookie after logout | `{"signed_in":false}` — revoked server-side |
| Reset token replay | `invalid_or_expired_token` — single use |
| Recovery redemption | establishes a working session immediately |
| Session after database rebuild | all sessions gone; `auth.sessions` count 0 |

---

## 6. Route families and their authority model

Grouped by prefix; counts are registrations, not distinct paths.

| Family | Count | Authority | Audited |
|---|---|---|---|
| `/api/shows/*` | 33 | Show → Show Run → Production → Location membership | negative-tested via outsider `403`s |
| `/api/show-runs/*` | 22 | roster membership or Producer/Director | outsider sees empty list |
| `/api/venues/*` | 12 | `access.UserCanAccessVenueSlug` | outsider `403` |
| `/api/participant-interactions/*` | 12 | interaction owner or Director | Kernel 73/74 coverage retained |
| `/api/discord/*` | 12 | operator | `401`/`403` confirmed |
| `/api/character-cards/*` | 11 | card ownership + `CanEditCard` | Kernel 73 tests retained |
| `/api/player-profile/*` | 9 | session user only; no route names another user | own-scope by construction |
| `/api/auth/*` | 8 | public, rate limited | see §3–4 |
| `/api/scenes/*`, `/api/stage-elements/*` | 11 | Production scope | outsider `401` |
| `/api/profiles/*` | 6 | self, or retired admin routes (`410`) | |
| `/api/warehouse/*`, `/api/workshop/*` | 8 | Producer scope | outsider `403` |
| `/api/world/*` | 3 | venue role resolution | anonymous + outsider `403` |
| `/ws/*` | 4 | see `kernel-76-websocket-matrix.md` | proven |

The venue and world families are the ones that carry Victory's real tenancy boundary, and they
were the ones that held up. The failures were all in surfaces that had been reasoned about as
"just a list" — the commons roster and the Production catalogue.
