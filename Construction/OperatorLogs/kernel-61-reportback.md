# Kernel Report Back — Kernel 61: Trailer Player Workbook, Face Compiler, and Legacy Profile Migration

**Kernel spec:** (pasted directly into chat this session; no committed spec file path exists in-repo)
**Commit(s):** none yet — all work is uncommitted on `main` (see `git status` in §4). Ask before committing; this repo's convention is to only commit when the operator asks.
**Date:** 2026-07-06

---

## 1. Status

**SUPERSEDED — see Kernel 61A.** This kernel (61) reached PARTIAL on its own (schema, domain layer, HTTP API, legacy-data migration, and a first read-only "My Face" page, all live-verified). Everything this reportback originally listed as not done — the Workbook/History UI, Face compiler, stage-name UI, websocket live-invalidation, secure email flow, cross-user viewing, `fresh-install.sh --local`, and legacy `/api/profiles/*` closure — was completed across the Kernel 61A closure sessions. **Kernel 61A reached PASS on 2026-07-09.** See `Construction/OperatorLogs/kernel-61A-reportback.md` for the final acceptance-criterion ledger and evidence; this file is kept as the historical record of the initial backend-foundation pass.

<details>
<summary>Original Kernel 61 status text (2026-07-06, kept for history)</summary>

**PARTIAL**

The backend schema, domain layer, HTTP API, and legacy-data migration are built and live-verified against the real database and the real (only) `performer_profiles` row (Straturli's). A first frontend surface — a read-only "My Face" page — is built and browser-verified, desktop and mobile, for both a populated and an empty profile. Two real bugs were found and fixed live during rollout (a JSON-encoding bug and a legacy-import self-heal gap), plus an unrelated but blocking CSS bug in four venues' hover-reveal headers (three fixed, one — Producer's Office — was already correct).

Not done: the Workbook (multi-page editing) and History (event list + delete) frontend modes, websocket live-invalidation, the secure email-change flow, a true two-browser cross-account viewing test, a from-scratch `fresh-install.sh` run, the legacy `/api/profiles/*` compatibility/deprecation work, and all operator-log/roadmap documentation updates. Kernel 61 is not PASS-eligible until those exist — see the ledger below for exactly what's proven and what isn't.

</details>

---

## 2. Acceptance-criterion ledger

Numbered criteria (kernel spec §12), then the summary ledger (kernel spec §17). "Evidence" cites what was actually run — anything without a live-test citation was, at best, unit-tested against pure logic, not exercised end-to-end.

| # | Criterion | Status | Evidence |
|---|---|---|---|
| AC-1 | One workbook per account | PASS | `player_profile_workbooks.user_id UNIQUE`; live: 28 users → 28 workbooks, 1:1, after bootstrap. |
| AC-2 | No account replacement | PASS | Migration is additive-only (no `ALTER`/`DROP` on `users`); live: user count 28→28, Straturli UUID unchanged across two container rebuilds. |
| AC-3 | Access preservation | PASS | Live counts unchanged pre/post migration: memberships 34, venue grants 7, character cards 11. (DB-level proof; Grant's Cabin was not re-clicked-through live this session — nothing in this kernel touches venue/operator tables, so risk is low, but it's not a fresh visual proof.) |
| AC-4 | Stage name required | PASS | `ValidateStageNameCandidate` rejects blank/whitespace (unit-tested: `TestValidateStageNameCandidateRejectsBlank`). No dedicated "complete identity setup" UI gate exists yet (that's Workbook UI, not built). |
| AC-5 | Stage-name ledger (close+open in one transaction) | **PARTIAL** | `ChangeStageName`'s transactional close/open logic was **not exercised live via HTTP** this session (deliberately avoided — didn't want to write a real, undeletable ledger entry to Straturli's account just to test it). Only the pure normalization/validation helpers are unit-tested. The ledger schema itself is proven live (see AC-28), but via the legacy-import path (`seedLegacyStageName`), not `ChangeStageName`. |
| AC-6 | Stage-name idempotency | **PARTIAL** | `NormalizeStageName` case/whitespace-insensitivity is unit-tested; the actual no-duplicate-row behavior in `ChangeStageName` was not live-tested (same reason as AC-5). |
| AC-7 | Stage-name isolation from other account data | **NOT DONE (live)** | True by construction (the function only writes to `player_stage_name_history`), but not exercised live this session. |
| AC-8 | Stage-name history cannot be deleted via ordinary endpoint | PASS | Live: `DELETE /api/player-profile/events/{stage_name_ledger_id}` → `404 event_not_found` (different table entirely; there is no code path that could reach it). |
| AC-9 | Catalogue validity | PASS | 8 unit tests directly exercise `ValidateCatalogue` (duplicate page/field keys, invalid type, face-eligible-without-region, wrong/missing favorite-TTRPG max, collection-without-max-items). The real embedded catalogue (`player-profile-v1.0.0.json`) passes validation at every backend boot (`LoadCatalogue` is called at startup and by every request handler; the process would `log.Fatalf` on a validation failure and it hasn't). |
| AC-10 | D&D class contract | PASS | `single_select_custom`, unit-tested (`TestValidatePageAnswersAllowsCustomDnDClass`). |
| AC-11 | Socio class/archetype contract | PASS | Same contract shape, fully separate from Character Workbook tables (no shared schema). |
| AC-12 | Favorite TTRPG max 3 | PASS | Unit-tested (accept 3, reject 4) and catalogue-validated (field must declare `max_items: 3`). |
| AC-13 | Typed page event on commit | PASS | Live: committed `favorite_book` on `favorites_and_personality`, got back a real `page_commit` event with a server-generated summary. |
| AC-14 | No-op commit creates no event | PASS | Live: re-submitting the identical value returned `{"changed": false}` with an empty event, no new row. |
| AC-15 | Ordinary event deletion | PASS | Live: deleted the event above, fact disappeared. |
| AC-16 | Deleting latest event reveals previous value | PASS (unit only) | `TestDeriveEffectiveFactsDeletingLatestRevealsPrevious`. Not separately exercised live via two sequential HTTP commits + delete-newest (the live test only covered the single-event case, which is AC-17). |
| AC-17 | Deleting only source event removes the fact | PASS | Live: same delete above — `favorite_book` fully absent from a subsequent `GET /me`. |
| AC-18 | Cross-account mutation rejected | **PARTIAL** | By design, no mutation endpoint accepts a target `user_id` at all — everything resolves the actor from session and acts only on themselves — so there's no request shape that even attempts a cross-account write. That's a stronger guarantee than a runtime check-and-reject, but it means there's no live negative test to point to (nothing to "try and get rejected"). |
| AC-19 | Social projection privacy | PASS | Live: `GET /{workbook_id}/face` response contains only `projection_version`, `stage_name`, `regions` — no email/handle/UUID/history/ledger, confirmed by inspecting the raw JSON. |
| AC-20 | Anonymous projection rejected | PASS (face route); untested (me route) | Live: unauthenticated `GET /{workbook_id}/face` → `401`. Same auth helper (`requireAuthenticatedUser`) gates `/me`, but that specific route wasn't separately hit anonymously this session. |
| AC-21 | Face show/hide/inferred | PASS | Live: hid `favorite_song` (disappeared from `/face`), reset to `inferred` (reappeared). |
| AC-22 | Manual integer priority | **PASS (unit only)** | `TestBuildProjectedFieldsManualPriorityWins`. The live `POST /face-priority` / reset-to-inferred endpoints were never called via HTTP this session. |
| AC-23 | Trusted projection-invalidation updates open viewers | **NOT DONE** | No websocket wiring exists — every `ProjectionChangeNotifier` parameter in `main.go` is `nil`. Mutations work; open tabs just don't get pushed an update (refetch-on-load only). |
| AC-24 | Forged invalidation rejected | **NOT DONE / N/A** | Nothing to forge against yet — no client-facing invalidation surface exists at all (which is itself safe, but doesn't satisfy the test as specified). |
| AC-25 | Secure email mutation | **NOT DONE** | No `PATCH /api/account/email` endpoint was built this session at all. |
| AC-26 | Legacy published import | PASS | Live: Straturli's real `performer_profiles` row (`is_published`) produced 18 current facts, visible on `/face`. |
| AC-27 | Legacy draft-only preserved privately | PASS (thin real-data coverage) | Straturli's only draft/published divergence was `display_name`, which imported as `legacy_draft__display_name` — present in owner facts, absent from `/face` (not a catalogue key, so never Face-eligible). The mechanism is real and correct; the real data only exercised one field of this path. |
| AC-28 | Idempotent migration | PASS | Live: `EnsureKernel61PlayerWorkbookSurface` ran on every one of 3 backend restarts this session; `player_profile_events` stayed at 2 rows and `player_stage_name_history` stayed at 1 row for Straturli throughout — no duplication. (This is also how the self-heal fix for the JSON-encoding bug was proven: facts went from 0→18 on a later restart without re-importing events.) |
| AC-29 | Clean install (`fresh-install.sh --local`) | **NOT DONE** | Migration `036` was added to the script's array but the script itself was never run this session. This is a real, not-yet-closed gap. |
| AC-30 | Compatibility / legacy endpoint adaptation | **NOT DONE** | `/api/profiles/*` routes are completely untouched — they still read/write `performer_profiles` directly via the original `profiles.go` code, with no adapter and no deprecation notice. |

### Summary ledger (kernel spec §17)

| Criterion | Status | Evidence |
|---|---|---|
| Existing user UUIDs preserved | PASS | Count + UUID check, live |
| Straturli UUID and account preserved | PASS | Live check |
| Operator status preserved | PASS | `handle` unchanged; not re-clicked-through live |
| Grant's Cabin access preserved | NOT RE-VERIFIED | Nothing in this kernel touches it; no fresh browser check this session |
| Buddy accounts preserved | PASS | `testflow1` loaded its own (empty) Face successfully |
| Memberships and venue grants preserved | PASS | Live counts unchanged |
| Character ownership preserved | PASS | Live counts unchanged |
| One Player Workbook per account | PASS | Live: 28/28 |
| Legacy published data migrated | PASS | Live, Straturli |
| Legacy draft-only data preserved privately | PASS | Live, Straturli (thin coverage) |
| Legacy migration is idempotent | PASS | Live, 3 restarts |
| Legacy table remains available/read-only | **PARTIAL** | Table untouched/not dropped (good), but old `/api/profiles/*` write endpoints are still live and still write to it — it is not actually "read-only" yet |
| Versioned catalogue drives backend and frontend | PASS | Backend fully catalogue-driven; frontend renders server-labeled projected fields (never redefines the field list) |
| Workbook contains multiple real pages | PASS | 6 pages / ~50 fields defined; no editing UI yet |
| Stage name required | PASS | Validation-level |
| Stage-name history append-only | PASS | Partial unique index + no delete path |
| Stage-name change preserves account/access | NOT LIVE-TESTED | See AC-7 |
| Stage-name history cannot be deleted | PASS | Live |
| Real name optional and Face-controlled | PASS | Catalogue field |
| Handle remains private/read-only | PASS | Never exposed in any new payload |
| Email remains private and securely editable | **PARTIAL** | Private: yes. Securely editable: no endpoint exists |
| D&D class selection works | PASS | Unit-tested |
| Socio class/archetype selection works | PASS | Unit-tested |
| Favorite TTRPGs enforce maximum three | PASS | Unit + catalogue tested |
| Page commit creates typed event and facts | PASS | Live |
| No-op commit creates no duplicate event | PASS | Live |
| Ordinary event deletion works | PASS | Live |
| Fact recomputation after deletion works | PASS | Live |
| Face show/hide/inferred works | PASS | Live (visibility only) |
| Manual integer priority works | **PARTIAL** | Unit-tested only |
| Stage name always appears on Face | PASS | Live, every payload observed |
| Owner Face preview matches social Face | PASS | Both derive from the same `BuildTrailerFace`; live-compared for Straturli |
| Other user sees only compiled Face | **PARTIAL** | Payload-shape privacy proven live; a true two-browser "A owns it, B views it" cross-user test was not run — both real accounts tested this session only viewed **their own** Face |
| Other user cannot edit owner workbook | PASS (by design) | No endpoint accepts a foreign target ID; not separately negative-tested live |
| Anonymous profile access rejected | PASS | Live, `/face` route |
| Account UUID absent from social UI/payload | PASS | Live payload inspection |
| Email and handle absent from social Face | PASS | Live payload inspection |
| Profile invalidation updates open viewer | **NOT DONE** | No websocket wiring |
| Forged profile invalidation rejected | **NOT DONE / N/A** | Nothing to forge against |
| Desktop visual acceptance attached | PASS (My Face only) | Screenshots below |
| Mobile visual acceptance attached | PASS (My Face only) | Screenshots below |
| Clean-install smoke passes | **NOT DONE** | Never run this session |
| Existing-database migration smoke passes | PASS | This *is* what was actually done live |
| Required automated checks recorded | PASS | See §4 |
| Operator and roadmap documents updated | **NOT DONE** | None of operator-log.md / operator-notes.md / kernel-maker-field-guide.md / dev-workflow.md / Master Implementation Guide / roadmaps were touched |

---

## 3. What was built

**Schema** — `database/migrations/036_kernel61_player_workbook_foundation.sql`: `player_profile_workbooks`, `player_profile_events`, `player_profile_facts`, `player_stage_name_history` (partial unique index enforcing one open ledger row per user), `player_profile_face_overrides`. Additive only. Registered in `scripts/smoke/fresh-install.sh`.

**Backend package** — `backend/internal/playerprofile/` (2,707 lines across 12 source files + 5 test files):
- `catalogue.go` + `catalogues/player-profile-v1.0.0.json` (go:embed'd) — versioned page/field catalogue: 6 pages (identity & presentation, TTRPG identity, table style, creative & production, availability & communication, favorites & personality), ~50 fields, including D&D class, Socio class/archetype, and favorite-3-TTRPGs.
- `types.go`, `facts.go`, `face.go`, `validation.go` — pure domain types and logic (fact derivation/recomputation, Face projection/region-grouping/sorting, page-answer validation), unit-tested without a database.
- `workbook.go`, `events.go`, `face_ops.go`, `stagename.go`, `projection.go`, `legacy_import.go` — DB-backed operations: `EnsureWorkbook`, `CommitPlayerProfilePage`, `DeletePlayerProfileEvent`, `RecomputePlayerFacts`, `ChangeStageName`, `LoadStageNameHistory`, `SetFaceVisibility`/`SetFacePriority` (+ inferred-reset variants), `ProjectOwnerWorkbook`, `ProjectTrailerFace`, `ProjectionVersion` (GREATEST()-over-timestamps, no separate counter table — same pattern as the Kernel 59A character projector), and `ImportLegacyProfiles`.
- `http.go` — 8 HTTP handlers, wired into `main.go`.
- 35 passing unit tests (`go test ./internal/playerprofile/...`).

**HTTP surface** (all wired in `main.go`):
```
GET    /api/player-profile/catalogue
GET    /api/player-profile/me
POST   /api/player-profile/pages/{page_key}/commit
DELETE /api/player-profile/events/{event_id}
POST   /api/player-profile/face-visibility
POST   /api/player-profile/face-priority
POST   /api/player-profile/stage-name
GET    /api/player-profile/{workbook_id}/face
```
`profile_id` in the social route is the **workbook ID**, not the raw account UUID, per the kernel's intent to keep account UUIDs out of ordinary Trailer URLs.

**Frontend** — `frontend/venues/trailers/face.html`: read-only "My Face" page. Identity header (portrait/stage name/pronouns) + At a Glance / Play & Create / About / Credits & Links cards, empty regions omitted, no "not set" wall. Linked from the legacy editor (`frontend/venues/trailers/index.html`) via a new "My Face (New)" button.

**Unrelated but blocking bug fix** — the "My Face (New)" button was unreachable due to a pre-existing hover-reveal header CSS pattern (shared across 4 venues) that collapsed mid-mouse-move and had no explicit `z-index`, letting page content underneath steal clicks. Fixed in `trailers`, `directors-chair`, `the-cave` (via a `.header-hover-zone` continuous-hover overlay + `z-index: 40`, matching `producers-office`'s already-correct implementation, which was left untouched).

**Live bug fixes found during rollout:**
1. `RecomputePlayerFacts` sent plain Go strings straight to a `jsonb` column; pgx treats a bare `string` as pre-encoded JSON text rather than marshaling it, so an unquoted value like `Straturli` failed Postgres's JSON parser. Fixed with an explicit `json.Marshal` (`jsonEncode`).
2. The legacy-import idempotency check returned early without recomputing facts, so a partial failure (event inserted, fact recompute then failed) would stay broken forever on every subsequent boot. Fixed: facts are now recomputed on every bootstrap run regardless of whether new events were imported.

---

## 4. Evidence

### Automated commands
```
cd backend && GOCACHE=/tmp/victory-gocache go build ./...      # PASS
cd backend && GOCACHE=/tmp/victory-gocache go vet ./...        # PASS
cd backend && GOCACHE=/tmp/victory-gocache go test ./...       # 1 pre-existing, unrelated failure:
  FAIL victory/backend/internal/assets
    TestLoadWarehouseStorageStatsKeepsDatabaseUsageWhenFilesystemLookupFails
    (invalid input syntax for type uuid — a filesystem-path test fixture bug, untouched by this kernel)
cd backend && GOCACHE=/tmp/victory-gocache go test ./internal/playerprofile/... -v   # 35/35 PASS
git diff --check                                                # clean
node --check (extracted inline script from face.html)           # clean
```

### Database and migration assertions
Pre-migration snapshot vs. current (three checkpoints across this session — after migration+first boot, after the JSON-bug-fix rebuild, and now):
| | Before | After (now) |
|---|---|---|
| `users` count | 28 | 28 |
| Straturli UUID | `9a51d20c-8646-4a54-ada1-4f7a9340dec6` | unchanged |
| `location_memberships` | 34 | 34 |
| `access_grants` | 7 | 7 |
| `character_cards` | 11 | 11 |
| `performer_profiles` | 1 | 1 (untouched, not dropped) |
| `player_profile_workbooks` | — | 28 |
| `player_profile_events` | — | 2 (stable across 3 restarts) |
| `player_profile_facts` | — | 18 |
| `player_stage_name_history` | — | 1 (stable across 3 restarts) |
| `player_profile_face_overrides` | — | 1 (from a live visibility-toggle test, reset to inferred but the row itself persists — expected behavior, not a bug) |

No leftover test artifacts: `auth.sessions` rows created for testing (`kernel61-smoke-test`, `kernel61-browser-smoke`, `kernel61-browser-smoke-empty`, `kernel61-hoverfix-smoke`, `kernel61-hoverfix-smoke2`) were all deleted after use; confirmed 0 remaining.

### Straturli/account/access preservation
Confirmed via the table above: UUID, handle, memberships, grants, character ownership all unchanged. Operator resolution is env/handle-based (`OPERATOR_HANDLE=straturli`) and untouched by this migration. Grant's Cabin itself was not re-clicked-through live this session (nothing in the migration or new code touches venue/operator tables, so risk is low, but this is not a fresh visual proof).

### Buddy-account preservation
`testflow1` (an existing real account, not a fixture) successfully loaded its own workbook (`GET /me`) and its own (empty) Face page in-browser, with no errors, confirming the new schema didn't disturb non-Straturli accounts either.

### Browser and two-user proof
- Trailers → My Face: real Playwright session (temporary `auth.sessions` row for Straturli, deleted after), realistic multi-step mouse movement from header to button, successful click-through to `face.html`, populated Face rendered correctly.
- Same for `testflow1`: empty-Face state rendered correctly (no "not set" wall, graceful "Unnamed Player" + "Your Face is empty so far" messaging).
- Director's Chair: same mouse-path-then-click test on its toolbar link, post hover-fix — works.
- **Gap:** no true two-browser cross-user test was performed (e.g., Browser A as Straturli, Browser B as `testflow1`, B viewing A's compiled Face). Both accounts tested this session only viewed their own Face.

### Desktop/mobile visual proof
Screenshots captured (not committed to the repo, in this session's scratchpad):
- My Face, desktop (1280px), populated (Straturli) — clean identity header, four populated region cards, no raw field keys, no "not set" wall.
- My Face, mobile (390px), populated — single-column responsive layout, no horizontal overflow.
- My Face, desktop, empty state (`testflow1`) — graceful empty state.
- The Cave and Producer's Office post-hover-fix — no regressions.

### Negative/security proof
- Anonymous `GET /{workbook_id}/face` → `401`.
- `GET /{unknown-uuid}/face` → `404 profile_not_found`.
- `DELETE /events/{stage_name_ledger_id}` → `404 event_not_found` (can't reach the ledger table through the ordinary-history endpoint).
- Re-deleting an already-deleted event → `404 event_not_found` (idempotent, no error/crash).
- No-op page commit → `changed: false`, no duplicate event.

### Production rebuild and health
`docker compose up -d --build backend` run twice (once for the initial rollout + bug discovery, once after the JSON-encoding/self-heal fixes). Container has been up and serving real traffic (visible in `docker logs`) continuously since the second rebuild. No health-check endpoint was queried directly, but ongoing successful request logs (`/api/discord/audio/status`, `/api/characters/venue-sheet`, etc.) confirm the process is healthy.

---

## 5. How to run from a clean state

1. `docker compose up -d postgres` (or use the existing running instance).
2. Apply migrations in order through `036_kernel61_player_workbook_foundation.sql` (see `scripts/smoke/fresh-install.sh`'s array for the exact ordered list) — **note: `fresh-install.sh --local` itself was not run end-to-end this session; do that before trusting a truly clean install.**
3. `docker compose up -d --build backend` — this runs `EnsureKernel61PlayerWorkbookSurface` at boot, which backfills one workbook per existing user and imports any `performer_profiles` row it finds (idempotent).
4. Sign in as any real account, visit `/venues/trailers/`, click "My Face (New)" (now reachable after the hover-header fix), or go directly to `/venues/trailers/face.html`.
5. To exercise the API directly: authenticate normally (real login, not a synthetic session — the synthetic-session technique used for testing this session required direct `auth.sessions` inserts and should not be treated as a normal workflow), then hit any of the 8 routes listed in §3.

---

## 6. Operator notes

- **Docker rebuild**: `docker compose up -d --build backend` from `/opt/victory`. **Correction (2026-07-07):** Caddy does *not* run outside Docker as originally stated here — it's a container named `bread-caddy` (image `caddy:2-alpine`), shared with an unrelated project on this host (`bread-exchange`). Its real config is `/opt/bread-exchange/Caddyfile` (not `/opt/victory/Caddyfile`, which appears to be stale/unused), and it mounts `/opt/victory/frontend` as a live volume (`/srv/web2`), proxying `victory.amurray.family`'s `/api/*`, `/ws/*`, `/auth/*` to the `backend` service and serving everything else as static files. The practical conclusion still holds: frontend HTML/CSS edits on disk are live immediately, no rebuild needed — just via a live-mounted volume rather than a host-level process.
- **Go build cache**: use `GOCACHE=/tmp/victory-gocache` for all `go build`/`vet`/`test` invocations in this environment.
- **Catalogue**: `backend/internal/playerprofile/catalogues/player-profile-v1.0.0.json`, go:embed'd — a new catalogue version means a new file + version bump, not editing v1.0.0 in place.
- **Migration image storage**: no new asset/image storage was introduced; `portrait_url`/`banner_url` are plain URL-string facts today (no upload pipeline built).
- **Email test**: not applicable — no email-mutation endpoint exists yet.
- **Projection invalidation**: `player_profile/projection_updated` is specified but **not implemented** — every `ProjectionChangeNotifier` in `main.go` is `nil`. Wiring this in is the same shape as `network.BroadcastCharacterProjectionInvalidation`, but that function broadcasts to clients with an active *character persona*; Trailer viewers have no persona concept, so it needs a new per-user-targeted broadcast path (the `Hub` currently only supports `Broadcast` (all) and `BroadcastSession` (by session ID), not by arbitrary `user_id` across sessions — this would need a small `Hub` addition first).
- **Known follow-up work, in rough priority order**: (1) run `fresh-install.sh --local` for real and fix whatever it finds; (2) Workbook editing UI (the multi-page form); (3) History UI (list + delete, the delete API already works); (4) a real two-browser cross-user Face-viewing test; (5) websocket invalidation; (6) email-change flow; (7) decide the fate of `/api/profiles/*` (adapt vs. deprecate) so `performer_profiles` can actually become read-only; (8) the operator-log/roadmap doc updates this reportback itself is partially fulfilling.
