# Kernel 76 — Canonical Current-State Inventory

Reconciles what the repository, the database, and the running deployment actually contain, as
of commit `97c7169` on `main`.

---

## 1. Kernel number verification (§4.2)

Searched `Construction/` for prior use of the number: the only occurrence of "Kernel 76" was
the forward-looking section in `Victory_Canonical_Roadmap_v2.md`. No spec, reportback, branch,
or migration had consumed it. **76 is the correct number**, and this kernel proceeds under it.

The next free migration number is **081** — Kernel 76 added none.

---

## 2. Kernel sequence 53–76

Reconciled from `Construction/Kernels/`, `Construction/OperatorLogs/`, and git history.

| Kernel | Spec present | Reportback | Status | Notes |
|---|---|---|---|---|
| 53–60 | yes | yes | shipped | kernel-60 is marked DRAFT in its filename |
| 61 / 61A | yes | yes | shipped | Player Workbook, Trailer Face |
| 62 | yes | yes | shipped | My People, relationship matrix |
| 63 | — | yes | shipped | Discord fix + Back to Map |
| 64 | — | yes | shipped | DB test isolation; `TEST_DATABASE_URL` required |
| 65 | yes | yes | shipped | Third Place / Headshot Commons — **source of K76-H02** |
| 66 | yes | yes | shipped | Show Run, Roster, Audience Program |
| 67 | yes | yes | shipped | Show instance model |
| 68 | yes | yes | shipped | Venue visibility; established the Trailer Face gate that K76-H02 restores |
| 69 | yes | yes | shipped | Scene Library |
| 70 / 70A | yes | yes | deployed | 70's migrations were initially never applied to production |
| 71 | — | yes | deployed | two-punch ticket system |
| 72 | yes | yes | deployed | migrations moved into `backend/migrations/`, auto-applied at boot with ledger + backup; `SameSite=Lax`; credential rate limiting |
| 73 | — | yes | deployed | Equip Mode, 150-item catalogue |
| 74 | — | yes | deployed | locked door, Ra, participant-local projection |
| 75 | yes | yes | deployed | tutorial completion, Story So Far, Aftercare |
| **76** | this kernel | this kernel | — | audit + bounded repair |

**Discrepancies found:**

- Two files claim Kernel 75: `kernel-75-tutorial-completion-aftercare-continuation-v0.1.md`
  and `kernel-75-tutorial-completion-story-so-far-aftercare-mvp-proof-v0.1.md`. The second
  matches what shipped.
- Kernels 63, 64, 71, 73, and 74 have reportbacks but no spec file in `Construction/Kernels/`.
- `Security-notes.md` was titled "Kernel 2 — Security Notes" and self-described as "partly
  historical" while still being the only security document. Its disclosure that reset tokens
  were temporarily logged remained accurate for 74 kernels (K76-C01).

---

## 3. Runtime

| Fact | Value |
|---|---|
| Serving path | Caddy (`bread-caddy`, shared container) → `backend:8081` |
| Backend | `victory-backend` container on `edge_net` + `victory_internal` |
| Database | `victory-postgres` (`postgres:16-alpine`), bound `127.0.0.1:5432` |
| Static frontend | `/opt/victory/frontend` → `/srv/web2` |
| Proxied prefixes | `/api/*`, `/ws/*`, `/auth/*` — nothing else |
| Firewall | UFW: 22, 80, 443, 53. Not 5432, not 8081 |
| Stale process | host `go run ./cmd/victory` on `:18081` from 2026-07-28, sharing the live database — stopped (K76-L02) |

---

## 4. Schema

**92 tables** before the rebuild; **81 migrations** in `backend/migrations/`, latest
`080_kernel75_hide_handoff_card_for_players.sql`.

Migrations are auto-applied at boot under `MIGRATE_ON_BOOT` with a ledger (`schema_migrations`)
and an automatic pre-migration dump. Proven complete: an empty database reached the current
schema from migrations alone, twice, with no dependency on pre-existing rows — which answers
the §5.1 question about migrations not represented in a fresh install.

Seeds reproduced on a clean database: 2 Locations, 16 Venues, 150 equipment items, 1 merchant
packet, 1 dialogue packet with 6 topics, 2 Scenes.

**Not reproduced by migrations** (hand-authored through the application, and destroyed by the
Kernel 76 rebuild with Grant's explicit confirmation): 108 stage `elements`, 30 `assets`,
the active venue map and 2 grid configs, 34 character cards, 3 headshots, 105 profile
workbooks.

---

## 5. Surfaces

- **221 HTTP route registrations** — see `kernel-76-route-matrix.md`.
- **4 WebSocket endpoints** — see `kernel-76-websocket-matrix.md`.
- **16 Venues**: catharsis, directors-chair, first-theater, grants-cabin, greenroom,
  info-booth, library, middle-school-stage, producers-office, show-runs, soil-experts,
  the-cave, third-place, trailers, warehouse, workshop.
- **Authentication methods:** Discord OAuth2 (open for account establishment) and
  email/password login (retained; signup closed by K76-H01).
- **Roles:** operator (handle match), producer, director, cast, crew, audience.

---

## 6. Documentation that no longer described reality

| Document | Problem | Action |
|---|---|---|
| `Security-notes.md` | described Kernel 2 posture as current through Kernel 75; disclosed the reset-token logging as temporary | replaced with a current-state document |
| `dev-workflow.md`, `kernel-maker-field-guide.md` | embedded the literal database password | now reference `${POSTGRES_PASSWORD}` |
| `.env.example` | same | same |
| roadmap §"Kernel 76" | anticipated the audit | consistent with what was done |

Historical reportbacks retain the old credential string deliberately: they are a record of
what was true when written, and rewriting them would falsify the archive. The credential is
rotated, so they disclose nothing live.

---

## 7. Post-rebuild state

| Table | Count |
|---|---|
| `users` | 2 (`discord_bridge` system account, `straturli`) |
| `auth.sessions` | 0 at rebuild |
| `locations` | 2 |
| `venues` | 16 |
| `equipment_items` | 150 |

`straturli` holds `producer` at `amurray-family` and operator authority by handle, verified
through a live browser-equivalent session.
