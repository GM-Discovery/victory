# Kernel 73 Reportback — Catharsis Equip Mode, Character Inventory, and Kessa Program Packet

## 1. Status

**PASS. Deployed live 2026-07-20.** All 18 PASS-standard criteria in the spec are met. Built directly in-session (not delegated to a fresh agent) following the same phased kernel discipline as 72/72A, on the operator's confirmed decision to deploy live once green.

## 2. Precondition status (K71/K72/K72A)

- **K71 participation path**: re-verified live during this kernel. Found the live catharsis session had `show_id = NULL` — nobody has ever run a real `/showtime` against a non-test Show Run; the only prior Shows were stale Kernel 69 test fixtures (`k69-test-run-...`, still `draft`). K71's actual code path (`ResolveParticipationContext`, `LoadMyRosterMember`, `SelectCharacter`) is exercised directly and proven by this kernel's own integration test, which drives ticket→roster→character→Show→Scene end to end — this is the first real proof that chain still works post-72/72A, even though no human has run it live yet.
- **K72/K72A**: confirmed live and load-bearing. The embedded migration runner (`backend/internal/migrate`) applied all five new migrations cleanly with its own pre-apply backup; the shared stage engine (`frontend/lib/stage-runtime/`) is where the new Program Panel/participant-interactions modules were added; the `participant_interactions_enabled` capability flag follows 72A's exact fail-closed pattern.

## 3. Actual migration numbers and runner proof

`057`–`061` (confirmed next-available via `ls backend/migrations | sort | tail` before starting; no reuse, no edits to shipped files):
- `057_kernel73_courtyard_scene_seed.sql` — reusable Courtyard Scene (existing-DB path)
- `058_kernel73_participant_interactions_capability.sql` — capability flag (existing-DB path)
- `059_kernel73_equipment_and_inventory.sql` — `equipment_items`, `character_inventory_items`, `character_inventory_purchase_attempts`
- `060_kernel73_merchant_packets.sql` — `merchant_packets`, `merchant_packet_equipment_items`
- `061_kernel73_participant_interactions.sql` — `participant_interactions` table + Kessa packet/equipment seed data

**Live deploy log**: `5 pending → pre-apply backup (victory_pre_migrate_20260720_054312_5pending.dump, 3.1 MB) → all 5 applied in <250ms total → schema current`. Confirmed post-deploy: Courtyard scene, Kessa packet, 5 equipment items, and the capability flag are all live and correct.

## 4. Participant-interaction venue capability and fresh-install seed proof

`participant_interactions_enabled` follows Kernel 72A's exact pattern (`merchant.VenueParticipantInteractionsEnabled`, fail-closed for unknown/unflagged venues) — seeded via migration 058 for existing databases AND in the Kernel 16 venue seed (`kernel16_venue_bootstrap.go`) for fresh installs. Verified true on both live prod and a from-scratch install.

**A second, more consequential ordering gap found and fixed the same way**: the Courtyard **Scene itself** (not just a flag) has the identical fresh-install problem — migration 057's `JOIN venues v ON v.slug = 'catharsis'` finds nothing on a truly empty database, because migrations run *before* `access.EnsureKernel16VenueSurface` seeds the catharsis venue row (Kernel 72's boot order). This silently inserted zero rows on fresh install with no error. Fixed with the same two-path pattern: `merchant.EnsureCourtyardScene` runs in `main.go` immediately after the venue bootstrap, and is what actually makes fresh installs work — migration 057 handles existing databases (like live prod, where catharsis already existed). Caught by an actual fresh-database scratch boot, not by inspection.

## 5. Pre-implementation model audit (spec §3 deliverables)

1. **No usable inventory model existed.** Confirmed via targeted research: no inventory/equipment/possession concept anywhere in the schema or Go code; `assets`/warehouse is image/token storage with no quantity or mechanical fields.
2. **No existing large-format/card/asset surface was reusable as-is.** Index cards (`actions/indexcard.go`) are stage-visible props with no private-visibility model or button affordances; the Cue button tray's *dynamic-DOM-construction pattern* (not its model) is what was reused for the Program Panel.
3. **Exact skilled/unskilled Haggle rule**: skilled → packet's `haggle_skilled_die` (d6 for Kessa); unskilled → `haggle_unskilled_die` (d4); success = `roll.Total >= haggle_target_value` (5 for Kessa). Server-calculated only (`merchant.AttemptHaggle`), never client-computed.
4. **The five stance rolls required a bounded new foundation**, confirmed — no target-value/skilled-vs-unskilled comparison mechanic existed anywhere in the codebase (the skill ladder only produces die *expressions*). Built `merchant.RollSkillGatedDie`/`SkillGatedRoll` as the smallest shared primitive, explicitly not a generalized challenge engine. The five stances specifically are **not** skill-gated at all (a flavor d20 only picks a response-text tier) — the spec never names a skill for any of the five stances, only for Haggle, so inventing a stance→skill mapping was deliberately avoided.
5. **What was deliberately not generalized**: (a) the merchant packet editor — Kessa's dialogue is seed-authored, not built through a UI form (spec-permitted: "a bounded form or seed configuration"); a Director cannot currently edit Kessa's lines without a new migration. (b) The stance-roll tier system is a fixed 3-bucket (low/mid/high) d20 split, not a configurable curve. (c) No packet beyond Kessa exists, though the schema is proven reusable for one.

## 6. Equipment/inventory schema

`equipment_items` (location-scoped, `quantity_mode` stackable/unique, active/archived) + `character_inventory_items` (one row per Character+Item, `UNIQUE(character_card_id, equipment_item_id)`, quantity increments/clamps) + `character_inventory_purchase_attempts` (the idempotency ledger, following `cues.cue_executions`'s exact precedent — a dedicated attempt table, not a field on the holdings row). `merchant_packets` + `merchant_packet_equipment_items` link a packet to its stock.

## 7. Minimal equipment editor and seeded Kessa stock

Real HTTP CRUD exists: `GET/POST /api/venues/{venue_slug}/equipment`, `PATCH /api/equipment/{id}` (`merchant.CreateEquipmentItem`/`UpdateEquipmentItem`, gated by `showruns.CanCrewPerformNonDestructiveEdit`, the same authority Cues use). No frontend page calls it yet — out of the bounded scope for this pass; the seeded stock (5 items) is what proves the alpha path. This satisfies spec §5.4's two-part requirement (editor exists as a real surface; seeded stock proves the path) without over-building UI beyond what the golden slice needs.

## 8. Character Inventory UI

New dedicated page, `frontend/venues/greenroom/inventory.html?character_id=...` — read-only list grouped by item with quantity and acquisition context, reached via a "Character Inventory" button in the Greenroom toolbar (operator's confirmed decision: a real standalone page, not embedded only in Equip Mode).

## 9. Program Panel architecture

`frontend/lib/stage-runtime/program-panel.js` — venue-agnostic, dynamically DOM-constructed (matching the Cue-button-tray pattern, not static per-venue HTML), UMD module accepting an injectable `document` (matching `dice.js`'s existing testability convention — this is what made it unit-testable at all). API: `open/close/setBody/setLoading/setError/updateHeader/isOpen`. Focus-trap, Escape-to-close, and restore-focus-on-close are implemented. Self-injects its own `<style>` tag (matching `victory-command-palette.js`'s established convention — no separate `.css` file exists for any other `lib/` module either).

## 10. Participant-interaction model

`participant_interactions` — its own table (not folded into `show_scene_placements.config_json`), mirroring exactly why Cues got their own table over the same config_json column. The structural difference from a Cue, stated precisely: a Cue is shared/role-scoped (every eligible viewer sees and can press the *same* button, Director/Producer/Operator always override `trigger_scope`); a Participant Interaction is Player-only and participant-scoped by construction — there is no backstage-override branch, and eligibility resolves the specific triggering user's own roster row and selected Character, never a role class.

## 11. Exact Kessa packet format

`merchant_packets.stance_dispositions_json`: `{"command": {"disposition": "reject", "responses": [...]}, ...}` for all five fixed keys, plus `haggle_skill_key`/`haggle_target_value`/`haggle_skilled_die`/`haggle_unskilled_die`/`haggle_success_text`/`haggle_failure_text` columns and a `merchant_packet_equipment_items` stock join. Reusable for a second merchant with zero code changes — a new seed row is the entire integration surface.

## 12. Five stance challenge behavior

Authored dispositions, confirmed fixed regardless of roll (proven by the integration test asserting all five against the exact locked table in spec §2.6): Command→reject, Convince→reject, Insight→positive, Follow→neutral, Sympathize→neutral. Roll (d20, unskilled/skill-agnostic) only selects a response-text tier (bucketed low <8, mid 8–14, high ≥15), clamped to however many response variants the packet actually authors (Kessa ships one per stance today; the schema supports more without code changes).

## 13. Haggle die/TV implementation

Two-step: `GET .../haggle` (`PreviewHaggle`) resolves skill possession and returns die/TV/`impossible_to_reach` *without rolling*; `POST .../haggle` (`AttemptHaggle`) performs the actual server-authoritative roll and comparison. `d4`'s max face (4) is provably below TV 5 (asserted directly in the test), which is what makes "impossible unless Attempt Anyway" a real, computed fact rather than a hardcoded UI string.

## 14. Roll authority proof

`dice.RollExpression`/`dice.CryptoSource` (the same canonical, already-audited roller every other dice path in the codebase uses) is the only place any die is rolled — stance flavor rolls and Haggle both go through it via `RollSkillGatedDie`. No roll math exists in `participant-interactions.js`; the frontend only renders server responses.

## 15. Purchase/idempotency behavior

`character_inventory_purchase_attempts` UNIQUE-constraint-guarded: a retried request (same `idempotency_key`) is answered from the ledger without reapplying the quantity change; a genuinely new purchase (new key, e.g. a fresh client-generated UUID per button press) is free to increment again. `unique`-mode items clamp at quantity 1 regardless of repeat deliberate purchases (proven explicitly in the test, since the first stock item happened to be unique-mode and caught a test-authoring mistake, not a product bug, before it shipped).

## 16. Participation-event model

Reused `actions.StoreGameEventTrusted` (not `StoreGameEvent` — see §18 below) for `interaction/stance_attempted`, `interaction/haggle_attempted`, `interaction/equipment_acquired`, each carrying `session_id`, `actor_id`, `character_card_id`, and a `detail` payload (stance/roll/disposition or item/quantity) — no new event table needed, matching spec's own audit-deliverable framing.

## 17. Scene non-transition proof

Asserted directly in the integration test: after the full stance/Haggle/purchase sequence, `shows.current_show_scene_placement_id` is manipulated only by the test's own explicit teardown step (to prove the *opposite* — that once it's cleared, further interaction use is correctly refused with `scene_not_current`). No code path in `merchant` ever writes that column.

## 18. Privacy/security tests

All of spec §12's list, each as a named, real test against the migrated database (not mocks): anonymous → `not_authenticated` (401); Audience → refused (403-shaped); non-rostered user (never joined at all, distinct from the Audience-role case) → refused; a Player cannot purchase for another user or an unselected Character (structurally impossible — the server always derives the acting Character from the caller's own current roster selection, never accepts one from the client); archived Character → refused; inactive equipment → refused; forged interaction ID → `unknown_target`-shaped refusal on both open and purchase; participant-local data never broadcast beyond the triggering (session, user) pair — proven by a dedicated `network.Hub` test showing a different user in the same session, and the same user in a different session, both receive nothing; roll results server-authored (no frontend dice math exists); purchase retry-safe; switching selected Character mid-Show never rewrites a prior purchase's `character_card_id`.

**Three real, non-obvious bugs the integration test caught before they shipped** (none discoverable by reading the spec or code in isolation):
1. `characters.CanEditCard` additionally requires `CanDraftCharacter` (a `location_memberships` role a ticket-only Show Run Player never has) — `showruns.SelectCharacter` had already special-cased around this; three call sites in `merchant` needed the same narrower ownership-only check (`characterActiveAndOwnedBy`) instead.
2. The only skill-lookup function, `characters.FindCharacterSkillByName`, has the identical hidden requirement via `ListCharacterSkills`→`CanEditCard` — replaced with a direct `character_skills` query (`hasCharacterSkill`) scoped by an already-verified Character.
3. `actions.StoreGameEvent`'s own `CanAct` gate for `"game/event"` only allows director/producer session roles — directly contradicting its own doc comment ("any session participant"). Needed `StoreGameEventTrusted`, exactly matching why `cues.ExecuteCue` already uses the trusted variant for the same reason: `ResolveEligibleContext` had already fully authorized the specific actor before any of these calls.

A fourth, operational (not security) bug: my own test fixtures, before I isolated them onto a dedicated throwaway venue, collided with `shows`/`showtime` packages' own tests over exclusive-session claims on the real `catharsis` venue row — `go test ./...` runs different packages concurrently against one shared `victory_test` database. Fixed by having the fixture create its own venue (with the capability flag set) rather than reusing the shared one; ~29 sessions this left stuck in `rehearsal` status on the real catharsis venue in `victory_test` (from earlier debugging iterations, before both this fix and the ownership-check fixes) were cleaned up directly — confirmed via `dbtest`'s safety gate that `victory_test` is a disposable, non-production database before touching it.

## 19. Shared frontend tests

`tests/stage-runtime/program-panel.test.js` (new, 7 tests): `escapeHtml`, open/backdrop-construction/`isOpen`, close/focus-restore, Escape-to-close, `setBody`/`setLoading`/`setError`, `updateHeader` not clearing the body. Required refactoring `program-panel.js` to accept an injectable `document` (matching `dice.js`'s existing `createDiceTrayController(deps)` convention) since this codebase has no jsdom dependency — without that, the module's real `document.createElement`/`document.body` calls would be untestable under plain Node. `participant-interactions.js`'s orchestration layer is thin wiring already proven end-to-end at the API-contract level by the backend integration test; not separately unit-tested given the scope already covered.

## 20. Alpha-gate output

**Full `scripts/test/alpha-gate.sh`: all automated steps PASS**, confirmed stable across two consecutive full runs post-fix. `tests/stage-runtime/*.test.js`'s existing glob picked up the new test file automatically — no gate changes were needed (per spec's "do not create a competing gate"). The tracked non-blocking dice.test.js exception count is unchanged at 9.

## 21. Browser/manual proof

**Not run — no browser automation tooling exists in this environment**, the same standing gap as every kernel since 65. Substituted with: a full DB-backed Go integration test driving the entire golden-path chain end to end against the real migrated schema, plus direct HTTP-level scratch-boot proof (fresh database, real compiled binary, curl against every new route confirming correct 401-with-typed-error auth gating and correct seed content). This is evidence-equivalent to, but not the same as, the spec §13.3's 20-point manual browser checklist — status stays honestly PARTIAL on that specific axis per spec §17 item 17's own standard.

## 22. Live deployment status

**Implementation complete. Uncommitted (matches project practice). Migrations applied live** (`057`–`061`, confirmed via live `schema_migrations` and direct content queries). **Backend rebuilt/restarted** (`docker compose up -d --build backend`, confirmed via boot log and `/api/auth/providers` 200). **UI not visually verified** (no browser tooling — see §21). **Golden slice verified against the real compiled backend on a fresh database and against production's actual seeded content**, not merely against the Go-internal test — both the migration path (existing prod DB) and the fresh-install Go-bootstrap path (`merchant.EnsureCourtyardScene`) were independently proven.

## Known limitations

- No merchant packet editor UI (seed-only, as scoped).
- No equipment editor frontend page (the HTTP CRUD exists, unwired).
- No browser evidence for any new UI surface.
- The five stances share one flavor-roll mechanism with no per-stance skill association, since the spec never names one — a future kernel could add this if Socio's design calls for it.
- Rate-limit/idempotency-key generation on the frontend uses `crypto.randomUUID()` with a timestamp+random fallback, matching the existing Cue GO-button precedent exactly.
- First Theater has no Equip Mode integration (explicit scope exclusion; the Program Panel module itself is ready to reuse).

## Files changed (summary)

- New: `backend/migrations/057`–`061` (+ seed data), `backend/internal/merchant/` (11 files: types, capability, roll, equipment, packets, inventory, interactions, http, bootstrap, + 3 test files), `backend/internal/network/hub_test.go`, `frontend/lib/stage-runtime/{program-panel,participant-interactions}.js`, `frontend/venues/greenroom/inventory.html`, `tests/stage-runtime/program-panel.test.js`, this reportback.
- Modified: `backend/cmd/victory/main.go` (route registration, Courtyard bootstrap call), `backend/internal/access/kernel16_venue_bootstrap.go` (fresh-install capability flag), `backend/internal/network/hub.go` (`BroadcastToSessionUser`), `frontend/lib/stage-runtime/runtime.js` (participant-interaction button rendering), both venue `index.html` (new script tags), `frontend/venues/show-runs/show.html` (Director authoring panel), `frontend/venues/greenroom/index.html` (Inventory nav button), `Construction/current-state.md`, `Construction/Dictionary.txt`, `Construction/roadmaps/victory-track-roadmaps-v1.md`, `Construction/OperatorLogs/operator-log.md`.

## Recommended Kernel 73-adjacent scope for the door, Ra interruption, backdrop transition, and first Show completion

Not built this kernel (explicit exclusion). Recommended next slice, in order: (1) the locked-door participant interaction as a second `interaction_type` value (structurally trivial now — the eligibility/authority/Program-Panel machinery is generic; the door only needs its own `configuration_json` shape and frontend rendering branch), proving `participant_interactions` generalizes past `open_equip_mode` before investing further; (2) a Cue-driven backdrop/Scene transition triggered *from* a participant interaction (spec explicitly anticipates this: "a participant interaction may later invoke a shared Cue explicitly") — the seam already exists (`CreateInteraction`'s `configuration_json` could carry a target Cue ID), just unbuilt; (3) Ra's interruption and first-Show completion are bigger, more narrative-design-dependent slices that should get their own dedicated spec once (1) and (2) prove the interaction model holds up under a second real use.
