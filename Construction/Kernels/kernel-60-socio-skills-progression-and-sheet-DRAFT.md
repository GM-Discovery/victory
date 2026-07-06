# Kernel 60 — Socio- Skills, Progression, and the Living Character Sheet

**Revision:** 0.3 — FINALIZED, IMPLEMENTATION STARTING (§11 items 1–7 resolved; verified against current codebase 2026-07-04: dice ladder means, `CanAct` allowlist pattern, `DiceRollRequest` payload extensibility, and Chapter4 capacity pattern all check out)
**Kernel type:** Game-rules domain layer (skills, dice progression, advancement), venue Game Events, right-tray character sheet UI
**Canonical rules source:** `/opt/victory/frontend/assets/rulesets/Sociov1_1.md` (Socio- v1.1, full text — builders must read Chapters 5–8 before implementing)
**Depends on:** Kernel 59 command registry/execute surface (shipped); compound dice pools in `backend/internal/dice` (shipped alongside this draft — `2d20+d12!+3` parses and rolls today); Chapter 2–4 onboarding facts (`workbook_context` chapters, attribute totals, first skill)
**Deliberately builds what Kernel 59 deferred:** the mechanical skill registry, the `game/event` venue mirror, and the first real rules-facing mutations

---

## 1. Objective

Give a completed character a living mechanical life inside a venue:

1. players add skills to their own active character with `/char add skill <name>`, surfacing at **d4**;
2. skills advance along the canonical Socio- dice ladder by the roll-under-10 rule, verified against actual session use;
3. the character sheet (Face) is visible and scrollable in the venue's right tray;
4. clicking a skill on that sheet rolls it (attribute + skill dice vs TV) through the existing server-authoritative `roll/dice` pipeline;
5. skill additions and advancements broadcast as structured **Game Events** into the venue's Game Events chat lane (the tab shipped empty in Kernel 59 — this kernel gives it producers);
6. all of it is server-authoritative, append-only, and idempotent, reusing the Kernel 59 command execute surface.

## 2. Why this is one kernel (and what the lattice looks like)

The owner's design is a lattice: dice engine → skill ladder → advancement → sheet UI → Face priority → History events all interlock. This kernel deliberately takes the load-bearing middle of that lattice in one pass because splitting it leaves dangling halves (a sheet with nothing to roll; a ladder with no UI to see it). But the lattice property cuts both ways — **single-point failures cascade**. The risk register (§12) names each joint and the seam that isolates it.

Explicitly **out** of this kernel (deferred, not forgotten):
- Face priority becoming game-aware (ranking History items by mechanical significance) and pretty-rendering History entries — later kernel; this kernel must only *emit clean typed events* so that later kernel has good data.
- Helper cards, stances, statuses, HP pools, Fate Points, TV challenge tables as enforced mechanics — the rules define them; the app does not yet. `/roll` + a visible sheet is the minimum playable loop.
- Milestone levels / reward cycles (every 5 skill improvements) — track the counter now, apply benefits later.
- `/value`, `/override`, Director locks — still deferred from Kernel 59.

## 3. Honesty section — what exists today (verified, not assumed)

A previous kernel (59) was specced against a phantom domain layer. Do not repeat that. Verified current state:

- **Dice**: `backend/internal/dice` parses and rolls compound pools with per-group exploding dice (`2d20+d12!`, `5d20`, `d10+d4!-1`). `MaxDiceGroups=10`, total dice ≤100. Rolls resolve server-side via `actions.StoreDiceRoll` (`roll/dice` actions, rate-limited, broadcast over the hub).
- **Skills**: only the Chapter-4 one-time "first trained skill" exists (`CommitChapter4FirstSkill`, `chapter4_select.go`). The 100-skill catalogue lives in `chapter4_skills.go` (`Chapter4SkillByID/ByName`, `Chapter4SkillsForAttribute`). Attribute capacity = raw attribute score from Chapter 2 (`loadChapter2Attributes`). There is **no** post-onboarding skill list, no dice-step storage, no advancement path.
- **Commands**: Kernel 59 shipped the registry (`backend/internal/commands`), `/api/commands/{available,preview,execute}`, idempotent execution (`command_execution_receipts`), and active-character resolution (`commands.ResolveActiveCharacter`). `/char add skill` was explicitly descoped and returns nothing today.
- **Chat lanes**: All/Roleplay/Game Events/OOC tabs exist in the venues; the Game Events lane filters on `type` prefix `game/event` — **no producer emits one yet**.
- **History**: `character_workbook_entries` is the append-only per-character ledger (`RecordWorkbookEvents`), used by all chapter commits.
- **Right tray**: venues have a right drawer with character summary + "Open workbook" link; the workbook/Face renders only in Greenroom (`buildWorkbookPages`). No in-venue sheet.

## 4. Canonical skill dice ladder

One ordered ladder, server-authoritative, stored as an integer **step index** per skill. Derived from the Level Progression Chart (Sociov1_1.md, Chapter 7), **reordered by mean roll value per owner ruling** — the chart's original placement of `4d12` and `5d12` was a math error; they now sit where their averages put them:

```
step  pool       mean
 0    d4          2.5   (untrained surface — new skills start here)
 1    d6          3.5   (trained)
 2    d8          4.5
 3    d10         5.5
 4    d12         6.5
 5    d10+d4      8
 6    d10+d6      9
 7    d10+d8     10
 8    2d10       11
 9    d10+d12    12
10    d20+d4     13
11    d20+d6     14
12    d20+d8     15
13    d20+d10    16
14    d20+d12    17
15    3d12       19.5
16    2d20+d4    23.5
17    2d20+d6    24.5
18    2d20+d8    25.5
19    4d12       26     (moved here from the chart's level-29 slot)
20    2d20+d10   26.5
21    2d20+d12   27.5
22    3d20       31.5
23    5d12       32.5   (moved here from the chart's level-37 slot)
24    3d20+d4    34
25    3d20+d6    35
26    3d20+d8    36
27    3d20+d10   37
28    3d20+d12   38
29    4d20       42
30    4d20+d4    44.5
31    4d20+d6    45.5
32    4d20+d8    46.5
33    4d20+d10   47.5
34    4d20+d12   48.5
35    5d20       52.5
```

(Note for the table: steps 18→19→20 swap dice sets — `2d20+d8` → `4d12` → `2d20+d10` — because `4d12`'s mean lands between them. Mathematically ordered per ruling; flag to players in the sheet UI so it doesn't read as a typo.)

Store the ladder in one Go table (`backend/internal/characters/skill_ladder.go` or a new `rules` package) with `StepExpression(step int) string` returning the dice expression; the dice engine already rolls every entry. Per §11 item 7, expose both an exploding form (each group suffixed `!`, for normal skill checks) and a plain form (for advancement rolls) — e.g. `StepExpression(step)` returns the exploding form and `StepExpressionPlain(step)` the advancement form, or a single function with an `explode bool` parameter.

**Skill dice caps by character level**: **track, don't enforce** (owner ruling). The server records `improvement_count`; the sheet shows each skill's improvement pips plus **player-toggleable checkboxes** the player can mark and unmark when they level (self-tracked, no mechanical enforcement — stored on the skill row so they persist, but the rules engine never reads them).

## 5. Skill model and storage

New table (migration `033_kernel60_character_skills.sql`, plus fresh-install.sh array entry):

```sql
CREATE TABLE IF NOT EXISTS character_skills (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  character_card_id UUID NOT NULL REFERENCES character_cards(id) ON DELETE CASCADE,
  skill_id TEXT NOT NULL,             -- catalogue stable ID or SKILL_xxxx custom ID
  skill_name TEXT NOT NULL,
  attribute_name TEXT NOT NULL,       -- governing attribute (capacity bucket)
  ladder_step INT NOT NULL DEFAULT 0, -- index into the canonical ladder (0 = d4)
  is_helper BOOLEAN NOT NULL DEFAULT FALSE,
  source TEXT NOT NULL DEFAULT 'player_added',  -- player_added | chapter4_first | custom
  improvement_count INT NOT NULL DEFAULT 0,     -- feeds milestone levels later
  level_checkboxes JSONB NOT NULL DEFAULT '[]'::jsonb, -- player-toggled self-tracking pips (no rules effect)
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (character_card_id, skill_id)
);
```

Rules:
- **Catalogue lookup** reuses `Chapter4SkillByID/ByName` (exact or unambiguous-prefix match; ambiguous → `ambiguous_skill_name`). Custom skills reuse the existing custom-skill contract (`customStableID("skill")`) where the ruleset permits.
- **Capacity** (owner ruling): counted in **half-slot units** so a player isn't overloaded with skills a character could invoke at once. A full skill costs 2 units, a **helper skill costs 1 unit (half capacity)**; total units per governing attribute ≤ attribute score × 2. Same transactional check pattern as `CommitChapter4FirstSkill`'s `attribute_capacity_full`.
  - **This kernel does not add a helper-selection path** (§2 already defers helper cards as enforced mechanics, and there's no top-level helper lookup today — `Chapter4SkillByID/ByName` only indexes the 100 full skills; helper cards live nested inside each skill's `Helpers []Chapter4SkillHelper` with no by-ID/by-name index, and the custom-skill contract has no helper concept). `is_helper` is always `FALSE` for every row this kernel writes; the column and the half-unit formula exist now so a later kernel can populate real helper rows without a migration. Every `/char add skill` addition costs 2 units.
- The Chapter-4 first skill is **backfilled** into `character_skills` on first read (step 1 = d6, since it is "trained"; source `chapter4_first`), so the sheet shows one unified list. Backfill is idempotent and never duplicates.
- Concurrency: capacity check inside the insert transaction (same pattern as chapter4's transactional capacity check).

## 6. Commands

Extend the Kernel 59 registry (all server-side, idempotent via `ExecuteIdempotent`, active-character resolved server-side):

```
/char add skill <name>          — add catalogue skill at d4 (preview → confirm; capacity + duplicate checks)
/char add skill --custom --name <n> --description <d>   — custom skill path (existing contract)
/char skills                    — list your active character's skills with current dice (private reply)
/char advance <skill>           — the roll-under-10 advancement attempt (see §7)
```

Errors reuse/extend the Kernel 59 contract: `no_active_character`, `skill_already_known`, `skill_capacity_reached`, `unknown_skill`, `ambiguous_skill_name`, `skill_not_used_this_session`, `skill_at_ladder_cap`.

## 7. Advancement — backward golf

Owner ruling: the player rolls their skill's current dice hoping to roll **low**; the system automates the outcome messaging.

Flow for `/char advance <skill>`:
1. Server verifies the skill was **used this session**: at least one `roll/dice` action in the current session by this user whose payload carries this skill's ID. **Owner ruling: untagged rolls do not count.** Players roll in many ways (face-sheet click, dice buttons, `/r`, `/roll`, custom roller UIs) and a bare `2d8` can't be attributed to a skill — only the skill-tagged paths (sheet click-to-roll, and any future roller that passes `skill_id`) count as verifiable use.
2. Server rolls the skill's current pool via the dice engine. **No explosion on advancement rolls** (owner ruling: a max face is automatically a disqualifying result anyway — it's well over 10; exploding it would be pointless).
3. `total < 10` → step up one ladder step, `improvement_count++`; else no change. Either way append a typed History entry and mirror a Game Event: *"Mara Venn tried to advance Blacksmithing (rolled 14 on d10+d4) — no improvement."* / *"…rolled 7 — Blacksmithing improves to d12!"*
4. No tier-cap enforcement (§4 — track only). At the top of the ladder return `skill_at_ladder_cap`.
5. Once per skill per session (unique receipt key `char.advance.<skill>.<session>`).

## 8. Click-to-roll and the right-tray sheet

- **Right tray sheet**: a scrollable Face panel inside the venue right drawer (both catharsis and first-theater; the-cave later). Server provides a compact sheet endpoint (name/pronouns/aura/portrait + attributes + skills with current dice) — a trimmed projection of `buildWorkbookPages`, **not** a second source of truth. No private fields (journal, private notes) in the venue payload.
- **Click a skill** → client sends the existing dice-tray roll with `expression = ladder expression (exploding form, per §11 item 7)`, `label = skill name`, and a new `skill_id` payload field threaded through `DiceRollRequest` → stored in the `roll/dice` payload (this is what advancement verification reads). Attribute contribution (owner ruling): **the attribute is NOT auto-added**. A skill's own description says when an attribute value is actually invoked, and it's rare — the full 100-skill catalogue with those descriptions already lives in `backend/internal/characters/chapter4_skills.go` (not just the picker cards). Click-to-roll rolls the skill's ladder dice only; when a skill's text invokes an attribute, that's table adjudication for now (a future per-skill metadata flag can automate it if it earns its keep).
- TV is not enforced by the app in this kernel (Narrator judgment per Chapter 6); the roll simply reports totals. TV tables become app-visible reference content in the Guide tab (cheap, optional).

## 9. Game Events (the deferred mirror, built now)

- New action type `game/event` added to `CanAct`'s allowlist, stored via a `StoreGameEvent` modeled on `StoreChatMessage`, payload per Kernel 59 §16.3 (`event_kind`, `source_character_event_id`, character/actor identity, structured detail).
- Producers in this kernel: `character_skill_added`, `character_skill_advanced`, `character_skill_advance_failed` (the failed attempt is public too — it's a table moment).
- Visibility: standard role visibility (`toRoles` all); revealed-to-role filtering respects the action's `visibility` envelope exactly like chat.
- Canonical character event first (History `character_workbook_entries` row), venue mirror second, linked by `source_character_event_id`; mirror failure never rolls back the skill mutation (Kernel 59 §16.4 semantics).

## 10. Scope summary

**In:** skill ladder table; `character_skills` + migration 033; `/char add skill|skills|advance`; chapter-4 backfill; `skill_id` on dice rolls; right-tray scrollable sheet (catharsis + first-theater); click-to-roll with attribute modifier; `game/event` action type + three producers; Game Events lane rendering (real entries replace the empty tab); History entries for every mutation; Go tests for ladder/capacity/advancement/idempotency; isolated browser verification.

**Out:** Face priority game-awareness & pretty History rendering; helper cards/stances/statuses/HP/Fate as mechanics; milestone reward *benefits*; TV enforcement; `/value`; `/override`/locks; skill removal (Director-only, later); the-cave sheet UI.

## 11. Resolved owner decisions (2026-07-04 — canonical, do not substitute)

1. **Ladder order**: reordered by mean roll value; `4d12` sits at step 19 and `5d12` at step 23 (§4). The chart's original placement was a math error.
2. **Tier caps**: track only, never enforce. Sheet shows improvement pips plus player-toggleable checkboxes for self-tracked leveling (§4, §5).
3. **Attribute in rolls**: not auto-added. Skill descriptions (full catalogue in `chapter4_skills.go`) say when an attribute is invoked, and it's rare (§8).
4. **Advancement verification**: untagged rolls don't count — players roll through too many surfaces (`/r`, `/roll`, dice buttons, sheet clicks, custom rollers) to infer skill use; only `skill_id`-tagged rolls qualify (§7). `/r` is already registered as an alias of `/roll` (`registry.go` — verified, no work needed here).
5. **Advancement explosion**: none — a max face is already well over 10 and disqualifies itself (§7).
6. **Helper skills**: count at **half capacity** (1 unit vs 2 in half-slot accounting) so players aren't overloaded with simultaneous skills (§5) — **but this kernel reserves the column only**; no helper-selection path is built now (no top-level helper lookup exists, and §2 already deferred helper cards as enforced mechanics). Every row this kernel writes is a full skill (`is_helper = FALSE`, 2 units).
7. **Normal skill-roll explosion**: skill checks explode by default (each max face rolls a bonus die, matching `/roll`'s existing behavior); only `/char advance` suppresses it (§7 item 2 is a carve-out from this default, not a no-op). `StepExpression(step)` must therefore expose both forms: an exploding expression for click-to-roll (e.g. `d12!`, `d10!+d4!`) and a plain one for the advancement roll.

## 12. Lattice risk register

| Joint | Cascade if it breaks | Seam that contains it |
|---|---|---|
| Dice ladder table | wrong dice everywhere (rolls, advancement, sheet) | single Go table + exhaustive test asserting every step parses & rolls |
| `skill_id` on rolls | advancement verification silently never passes | payload field is additive; advancement error `skill_not_used_this_session` is explicit, never a silent no-op |
| `game/event` mirror | Game Events tab stays empty / duplicates | mirror is fire-after-commit + idempotent by source event ID; skill truth lives in `character_skills`+History, never in the mirror |
| Sheet projection | stale/duplicated character truth in venues | projection endpoint reads the same loaders as Greenroom; no venue-side writes |
| Capacity check | over-cap skill lists corrupt later milestone math | transactional check + unique index; backfill idempotent |
| Future Face priority | needs typed events | every mutation here emits `entry_type`-typed History rows now |

## 13. Validation

- `go test ./internal/dice/... ./internal/characters/... ./internal/commands/... ./internal/actions/...` (never the full suite; identity/network tests touch live data — standing rule).
- Fresh-install smoke (`scripts/smoke/fresh-install.sh --local`) with migration 033 in the array.
- Isolated browser pass (throwaway DB + scratch backend/Caddy, per the established recipe): onboard a character → `/char add skill` → skill appears at d4 on the right-tray sheet → click it → roll lands in Dice/Game Events lanes with skill label → `/char advance` before use fails, after use rolls golf → advancement Game Event visible to a second connected user.
