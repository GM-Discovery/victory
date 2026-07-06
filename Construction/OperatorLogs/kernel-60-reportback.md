# Kernel Report Back - Kernel 60

## 1. Status
PASS WITH PENDING BROWSER-LEVEL VERIFICATION

Kernel 60 gives a completed character a living mechanical life inside a venue: players add skills at d4 with `/char add skill`, skills advance along the canonical Socio- dice ladder by the roll-under-10 rule, the character sheet is visible and clickable in the venue right tray, clicking a skill rolls it through the existing server-authoritative dice pipeline, and skill additions/advancements broadcast as structured Game Events. All mutation paths were exercised end-to-end against a live isolated database via the real HTTP command surface. Sheet rendering and click-to-roll in the DOM were code-reviewed and syntax-checked but not visually driven in a browser.

## 2. What Was Built

- Canonical 36-step skill dice ladder (`backend/internal/characters/skill_ladder.go`), reordered by mean roll value per owner ruling (`4d12` at step 19, `5d12` at step 23), with both an exploding form (normal skill checks) and a plain form (advancement rolls).
- `character_skills` table (migration `033_kernel60_character_skills.sql`, added to `scripts/smoke/fresh-install.sh`) and its model (`backend/internal/characters/character_skills.go`): catalogue lookup by exact-or-unambiguous-prefix name, idempotent Chapter-4 first-skill backfill, half-slot-unit capacity enforcement (full skill = 2 units, attribute capacity = attribute score × 2) serialized per character+attribute with a Postgres advisory transaction lock.
- Skill advancement (`RecordSkillAdvancement`, `FindCharacterSkillByName`): roll-under-10 resolution, ladder-cap detection, History entries for both success and failure.
- `skill_id` threaded through `DiceRollRequest` (`backend/internal/actions/dice.go`), the websocket dice handler (`backend/internal/network/ws.go`), and both venues' dice-tray modules (`runtime/dice.js` in catharsis and first-theater) — this is what advancement-verification reads.
- New `game/event` action type (`backend/internal/actions/game_event.go`), added to `CanAct`'s allowlist reusing `roll/dice`'s authority shape (`backend/internal/actions/authority.go`), with three producers: `character_skill_added`, `character_skill_advanced`, `character_skill_advance_failed`.
- Command surface: `/char add skill <name>`, `/char add skill --custom --name --description --attribute`, `/char skills`, `/char advance <skill>` (`backend/internal/commands/char_skills.go`, `registry.go`, `network/commands_http.go`). Advancement idempotency is hardcoded to a skill+session-scoped key (`char.advance.<skill_id>.<session_id>`, fixed `"attempt"` token) so "once per skill per session" is server-enforced regardless of client behavior.
- Right-tray venue sheet endpoint (`backend/internal/characters/venue_sheet.go`, `network/venue_sheet_http.go`) — a trimmed, no-private-fields projection (name/pronouns/portrait/aura + attributes + skills-with-dice).
- Right-tray sheet UI in both catharsis and first-theater: identity block, attributes grid, clickable skill rows that call the existing dice-tray roll with the skill's exploding expression and `skill_id`.
- New **Game Events** tab added to both venues' Chat panel (they previously only had Chat/OOC/Dice/Guide — no Game Events lane existed outside the-cave), wired to the new `game/event` websocket action type.
- Command-palette parser fix (`frontend/lib/victory-command-palette.js`) so `/char add skill <name>` and `/char advance <skill>` actually split into the right args instead of dropping the tail — needed for any of the above to be reachable by typing.

## 3. Evidence

Checks run:

- `cd backend && go build ./...`
- `cd backend && go vet ./...`
- `cd backend && go test ./internal/characters/... ./internal/actions/... ./internal/commands/...`
- `cd backend && go test ./internal/network/... -run TestParseCharAddSkillArgs` (scoped: the broader network package has a pre-existing stateful Discord-bridge test that touches live-style data and is excluded per standing project guidance)
- `node --check` on all touched frontend JS (`runtime.js`, `runtime/dice.js`, `runtime/session-sync.js` in both venues; `victory-command-palette.js`)
- `scripts/smoke/fresh-install.sh --local` (full clean-DB migration + backend boot + smoke suite)
- A second, separate throwaway database + scratch backend, driven directly with `curl` through the real `/api/commands/execute` and `/api/characters/venue-sheet` HTTP surface

Result summary:

- Backend build, vet, and all new/touched package tests passed.
- `fresh-install.sh --local` passed clean, including migration `033` applying without error from an empty database.
- All new frontend JS parses cleanly.
- Isolated functional verification (see below) confirmed the full mutation chain end-to-end.

## 4. Verified Kernel Points

- **Ladder math**: every one of the 36 steps parses and rolls in both exploding and plain form; means are strictly increasing (unit-tested).
- **Chapter-4 backfill**: seeded a character with a confirmed `chapter4` workbook fact (no `character_skills` row yet) and called the venue-sheet endpoint — the skill was backfilled at step 1 (d6), `source=chapter4_first`, idempotently (verified no duplicate row on repeat calls).
- **`/char add skill`**: added a catalogue skill at step 0 (d4); confirmed as its own `character_skills` row via `/char skills`.
- **Capacity enforcement**: with attribute score 2 (Awareness, capacity 4 units) and two full skills already occupying 4 units, a third Awareness skill was correctly rejected with `skill_capacity_reached`. A Craft skill (score 3, capacity 6, only 2 units used) was correctly accepted.
- **Advancement gating**: `/char advance <skill>` correctly returned `skill_not_used_this_session` before any qualifying roll existed for that skill in that session.
- **Advancement resolution**: after seeding a `roll/dice` action tagged with the skill's `skill_id` for the session/actor, `/char advance` correctly rolled the skill's **plain** (non-exploding) expression, applied roll-under-10, incremented `ladder_step` and `improvement_count`, and recorded a `character_skill_advanced` History entry with the real character name.
- **Game Event mirror**: both `character_skill_added` and `character_skill_advanced` produced a `game/event` action row, correctly linked to its canonical History entry via `source_character_event_id`, with the expected `event_kind`/`detail` payload shape.
- **Idempotency**: calling `/char advance` twice for the same skill/session returned the identical stored action (same `id`/`ts`) on the second call, and the database confirmed only one ladder-step increment and one Game Event row — the fixed-key idempotency wrapper works as designed.
- **`/r` alias**: confirmed already registered as an alias of `/roll` prior to this kernel; no work was needed there (doc corrected accordingly).

## 5. How to Run

1. Onboard a character through Chapter 4 (or seed `workbook_context.chapter4` directly) so a first skill exists.
2. Open Catharsis or First Theater; the right tray should show the sheet with the backfilled skill at d6.
3. `/char add skill <name>` to add a second skill at d4; it should appear on the sheet.
4. Click the skill on the sheet — it should roll through the Dice tab and post the tagged roll.
5. `/char advance <skill>` — should fail with a friendly message before the click-to-roll above; after it, should resolve and post to the new **Game Events** tab.
6. Repeat `/char advance` for the same skill in the same session — should return the same result without re-rolling or double-advancing.

## 6. Operator Notes

- Every row this kernel writes to `character_skills` is a full skill (`is_helper = FALSE`, 2 units). The half-unit/`is_helper` column and formula exist now so a later kernel can populate real helper rows without a migration — there is currently no catalogue lookup path for helper cards (they live nested inside each skill's `Helpers` array with no top-level index), so helper selection was deliberately left out of scope this pass.
- Skill checks explode by default (matching `/roll`'s existing behavior); only `/char advance` rolls the plain, non-exploding form. This was an open question in the original draft, resolved with the owner before implementation.
- The advancement idempotency key intentionally ignores any client-supplied idempotency key and uses a fixed `"attempt"` token under a skill+session-scoped command path, because "once per skill per session" is a hard server rule, not a client-optional dedupe courtesy.
- `game/event` reuses `roll/dice`'s `CanAct` authority function (`canActDiceRoll`) rather than a bespoke check, since both are always server-produced, never directly user-authored, and share the same "any session participant" shape.
- Discovered during isolated verification (not a kernel-60 defect, but worth recording): the live session/action pipeline (`ResolveSessionIdentity`) currently hardcodes `v.slug = 'the-cave'`, and any action (chat, dice, and now game/event) requires a `showings` row that only gets created lazily by the first successful action against an already-passing `CanAct` check — which itself requires a showing to already exist. In practice this means a session only becomes actionable once a showing has been started through the normal producer/session-start flow; a from-scratch session assembled purely at the database layer (as in this verification) needs a showing inserted by hand first.

## 7. Deviations from Kernel

- Helper-skill selection (`is_helper = TRUE` rows) is not implemented — see Operator Notes. The original draft's §2 already deferred "helper cards ... as enforced mechanics," but §5/§11 as originally written implied helper accounting was active in this pass; the draft was corrected to say the column is reserved only.
- The Game Events tab did not previously exist in catharsis or first-theater (only the-cave has it) — the original draft's honesty section assumed otherwise. This kernel adds the tab to both venues from scratch rather than "filling an existing empty tab" as originally described.
- Custom skill creation needs an explicit `--attribute <name>` flag (`/char add skill --custom --name <n> --description <d> --attribute <a>`) that the original draft's command syntax didn't list — without it there is no way to know which attribute's capacity the custom skill should count against.

## 8. Known Issues

- Sheet rendering, click-to-roll, and the Game Events tab were verified by code review and `node --check` syntax validation only — no browser was driven to visually confirm the right-tray sheet paints correctly or that a real DOM click triggers a roll.
- TV enforcement, helper cards, milestone-level benefits, `/value`, `/override`, and Director locks remain deferred, consistent with the kernel's stated scope.

## 9. Next Recommended Step

- Drive an actual browser session (throwaway DB + scratch backend/Caddy) through: onboard → `/char add skill` → sheet shows it at d4 → click it → roll lands in Dice and Game Events lanes with the skill label → `/char advance` before/after use → advancement Game Event visible to a second connected user.
- Decide when a later kernel should build real helper-card selection, now that the schema has room for it.
