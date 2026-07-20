# Kernel 72A Reportback — Post-Deploy Bug Fixes and Venue Capability Flags

## Status

**PASS. Deployed live 2026-07-19** (three backend deploys the same day, each proving the Kernel 72 migration runner's behavior: full 55-migration adoption, `schema current` no-op, and a single-pending apply with auto-backup). Kernel 72A is the operator-report-driven follow-up to Kernel 72: five live bug fixes plus one structural refactor that removes the bug class entirely. Uncommitted alongside Kernel 72 for operator review.

## What the operator reported, and what each report actually was

| Report | Diagnosis | Class |
|---|---|---|
| First Theater: renderer fallback, "Snapshot refresh failed: no rows in result set", Ping → "socket unavailable" | FT has had **no Session since Kernel 70A** closed the legacy live stages; the stage bootstrap treated a failed join as fatal, killing snapshot/socket/Pixi/editors together. Pre-existing (pre-fork FT had identical code) | Bug — fixed |
| "Rehearsal available — open Stage Management" visible to everyone | Banner gated only on venue config, not viewer role | Bug — fixed |
| Map "saved but didn't mount"; token upload `warehouse_physical_reserve_exceeded` | One cause: the asset upload itself was refused — host free disk (6.6 GiB) had fallen below the 8 GiB warehouse physical reserve (Docker build cache). The editor preview is a local-file preview, so it looked like a save failure | Environment — fixed (disk reclaimed) |
| Map save "long delay" then appears | Upload image processing (~3–4 s per `POST /api/workshop/assets`) + Pixi texture decode; the map save itself is ~15 ms | Not a defect (candidate: async thumbnailing/progress UI) |
| "Cannot add tokens" (catharsis) | `create token denied … reason=unknown_target`: **catharsis was never in the hardcoded element-action venue allowlist**, so every element action there (tokens, index cards, reveal, remove, duplicate, lock, nameplate) had been denied since its stage shipped | Bug — fixed, then root-caused structurally (below) |

## Fixes in place

### 1. Session-less venues are a first-class idle state
- `identity.JoinVenue` returns typed **`no_active_session`** (`ErrNoActiveSession`) when a venue has no rehearsal/live Session, instead of leaking pgx's "no rows in result set".
- The shared stage engine (`frontend/lib/stage-runtime/`) tolerates exactly that typed refusal in both `startStageRuntime` and `refreshWorld`: socket skipped, status reads "No Show is on stage here yet — the stage is read-only until one starts.", world snapshot still fetched (its Kernel 70A `theater_context` message explains the state), Pixi and editors still initialize. Any other join failure still aborts loudly.
- Recovery path unchanged and now actually reachable: start a Show Session from Stage Management and the venue joins/sockets normally.

### 2. Rehearsal banner is backstage-only
`renderRehearsalBanner` additionally requires `theater_context.kind === "backstage"` — the server-computed Director/Producer/Operator/Crew classification (never re-derived client-side). Known nuance: a Director who is also a registered player classifies `participant` and won't see the banner.

### 3. Warehouse capacity restored
`docker builder prune -af` + dangling-image prune freed ~4.3 GB (~10 GB free → ~2 GB usable upload headroom above the reserve). Left for operator decision: unused `microscope-app` (1.63 GB) and `mysql:8.0.27` (684 MB) images belong to other projects; and `WarehousePhysicalReserveBytes` (8 GiB) is conservative for a 38 GB disk.

### 4. Structure: venue capability flags replace hardcoded slug allowlists

This is the piece the kernel maker should internalize — it changes how every future venue is wired.

**The bug class:** element actions were gated by `actions.isSingleVenueLegacySlug`, a hardcoded switch that grew one venue at a time ("the-cave" → Kernel 70A added "first-theater" → the Kernel 72 hotfix added "catharsis"). Catharsis shipped a complete stage whose every element action was denied `unknown_target` because nobody edited authority code. The planned 3–5 new game venues would each have repeated this.

**The replacement:** two `venues.config` boolean capabilities, following the existing `index_cards_enabled` / `actors_can_reveal` / `scene_rehearsal_enabled` precedent. Unknown venues and missing flags **fail closed**.

| Flag | Grants | Seeded true for |
|---|---|---|
| `stage_elements_enabled` | The whole element-action surface: tokens, index cards, reveal, remove, duplicate, lock, nameplate (`actions.stageElementsEnabled`, replaces `isSingleVenueLegacySlug` at all 12 call sites + the reveal-path SQL + both `indexcard.go` slug filters) | the-cave, first-theater, catharsis |
| `session_control_enabled` | `/session status\|start\|end` targeting (`network.VenueSessionControlEnabled`, replaces `normalizeSessionControlVenueSlug`) | the-cave, first-theater, catharsis, middle-school-stage (former list, membership unchanged) |

**Two seeding paths, both required** (the Kernel 70A/72 deploy lesson applied):
- Migration `055_kernel72a_stage_capability_flags.sql` — idempotent `jsonb_set`, covers every database whose venue rows already exist (live, victory_test).
- The Kernel 16 venue seed (`access/kernel16_venue_bootstrap.go`) now carries the flags in its INSERT configs — covers **fresh installs**, where Go-seeded venues are created *after* migrations run. This same ordering gap silently applied to Kernel 70's `scene_rehearsal_enabled` (fresh installs never got it); the seed now includes that flag too.

**For a new game venue**: add its venue row (seed or migration) with `"stage_elements_enabled": true, "session_control_enabled": true` in config. No authority code changes. `grep -rn "the-cave.*first-theater" backend/internal` should stay empty of new allowlists.

### 5. Second pass: the placement resolver's borrowed flag (operator re-reported "negative on token placement")

After the capability refactor deployed, token creation on catharsis advanced from `unknown_target` to **`policy_denied`** — a *third* venue gate nobody had mapped: `resolvePlacementVenue` (shared by token create, element place, and duplicate) checked **`index_cards_enabled`**, a flag migration 010 only ever set on **the-cave**. So even First Theater — despite Kernel 70A's "same authority surface" work — would have failed here the moment its stage went live. Two changes:

- `resolvePlacementVenue` now checks **`stage_elements_enabled`** — placing elements on a stage is exactly the element-surface capability, which it had only borrowed `index_cards_enabled` to approximate. the-cave's behavior is unchanged (it has both flags).
- Migration `056_kernel72a_index_cards_parity.sql` + the Kernel 16 seed give first-theater/catharsis `index_cards_enabled` (the genuinely index-card-specific policy the CanAct path still consults), completing toolset parity with the-cave.

**The regression test this saga was missing:** `actions.TestStoreCreateTokenOnCatharsisStage` now drives the *entire* create-token chain against the real test database — fixture user/session/participants/showing/warehouse-asset, then `StoreCreateToken` as a producer on catharsis (asserting the element AND its stage placement row exist) and as an audience participant (asserting refusal). No test had ever exercised `StoreCreateToken`, which is why three stacked venue gates could each fail silently in sequence.

## Verification evidence

- Full Go suite green after every step, including two new DB-backed tests asserting the flag matrix against the real migrated test database (`actions.TestStageElementsEnabled`, `network.TestVenueSessionControlEnabled` — both also prove fail-closed on unknown/blank venues). The obsolete pure-function allowlist test was converted rather than deleted.
- The `authority_test.go` fake querier gained a `stage_elements_enabled` dispatch case modeling an enabled stage venue.
- **Full `alpha-gate.sh`: all automated steps PASS** (with the standing 9-title dice exception), including the fresh-install smoke — which now exercises the seed-carried flags on a from-empty database.
- Scratch-boot proof of fix 1 (fresh DB, operator account): FT join → `{"error":"no_active_session"}`; FT world snapshot → `ok:true` with `theater_context.kind: backstage`.
- Live deploy proof: boot log `1 pending → pre-apply backup (victory_pre_migrate_20260719_200337_1pending.dump) → applied 055 in 21ms → schema current`; prod `venues.config` then queried showing exactly the intended flag matrix.
- Operator live-confirmed between fixes: map mounts on catharsis; grid works.
- Second-pass live deploy: boot log `1 pending → pre-apply backup (victory_pre_migrate_20260719_203621_1pending.dump) → applied 056 in 11ms → schema current`; prod shows `index_cards_enabled=true` for all three stage venues. Token placement on catharsis is proven end-to-end by the new DB-backed regression test; operator browser confirmation still the final word.

## Known limitations

- No browser evidence (standing gap; Playwright kernel still the top recommendation).
- The fake-querier unit tests model an enabled venue; the disabled-venue paths are covered by the DB-backed tests only.
- `workshop`'s long-lived session predates session-control and is untouched by the `session_control_enabled` flag (it was never in the old allowlist either).
- Map-appearance latency (upload processing + texture decode) is unaddressed by design; candidate future kernel: async asset processing with progress UI.

## Files changed

- New: `backend/migrations/055_kernel72a_stage_capability_flags.sql`, `backend/internal/actions/stage_elements_dbtest_test.go`, this reportback.
- Modified: `backend/internal/actions/{authority.go, authority_test.go, token.go, remove.go, duplicate.go, indexcard.go}` (capability gate), `backend/internal/network/{session_control.go, session_control_test.go, rehearsal_capability.go}`, `backend/internal/identity/join.go` (`ErrNoActiveSession`), `backend/internal/access/kernel16_venue_bootstrap.go` (seed flags incl. the latent `scene_rehearsal_enabled` fresh-install gap), `frontend/lib/stage-runtime/{runtime.js, session-sync.js}` (idle-state tolerance, backstage banner gate).
- The interim hardcoded-allowlist hotfix from the Kernel 72 triage (adding "catharsis" to the switch) is superseded by this refactor; the switch no longer exists.
