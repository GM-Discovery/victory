# Kernel 85 — Socio Sustained Play: Cohorts, Scene Progression & Game Status

**Status:** IN IMPLEMENTATION
**Type:** Product-facing sustained-play kernel
**Primary tracks:** S6 — Sustained Adventure, V3 — Scenes/Storyboarding, V6 — Shows/Admission, V7 — Rules Integration Boundary
**Secondary tracks:** V1 — Character Truth, V8 — Player Identity, O — Operational Integrity
**Sequence position:** After Kernel 84
**Primary production:** Catharsis
**Primary authority:** Director+
**Planning authority:** Canonical Roadmap as reconciled by Kernel 84
**Implementation authority:** current repository behavior, current Show/Scene/Character state, kernel reportbacks, operator decisions

---

## 0. Kernel contract

Kernel 85 must prove that Socio can move beyond one guided tutorial path into **sustained, Director-run play**.

The core vertical is:

```text
Catharsis tutorial participants begin ungrouped
→ Director+ creates Cohort 1 / Cohort 2 / ...
→ participants are explicitly assigned to one cohort
→ each cohort may occupy a different current Scene at the same time
→ Director+ right-clicks the current stage/map
→ opens Scene Configuration
→ selects a Scene attached to the current Show
→ applies that Scene to the selected cohort only
→ other cohorts remain on their existing Scene
→ saved Scene placements restore when defined
→ Director may update the current Scene arrangement or save it as a new Scene
→ Director opens movable Game Status tool
→ Game Status shows canonical Socio status for Characters in the selected cohort
→ Director may adjust supported HP/status state through canonical Socio domain operations
→ participant access is granted through a proper People Picker for the current Show
→ the group can leave, return, and continue without manually reconstructing the cohort's Scene or Character mechanical state
```

Kernel 85 is not another tutorial kernel. Victory should assist the Director in running open play. It should not hardcode the story path, automatically choose consequences, or become the Narrator.

---

# 1. Locked product decisions

## 1.1 Catharsis begins ungrouped

Participants in the starting Catharsis Show are **not** automatically placed into Cohort 1.

Initial state:

```text
Ungrouped
- Alice
- Bob
- Kyle
- Grant
```

Participants may complete the existing tutorial while ungrouped.

To progress past the tutorial-finish Scene into sustained play, a participant must be assigned to a cohort.

Do not silently create cohorts for everyone.

## 1.2 Cohort naming

Cohorts auto-name:

```text
Cohort 1
Cohort 2
Cohort 3
...
```

Use deterministic numeric serialization.

Requirements:

- stable UUID identity;
- safe serialized key/slug;
- visible auto-name;
- deleting Cohort 2 does not cause a future cohort to collide with its historical identity;
- next generated number must be deterministic and collision-safe.

Do not key identity off visible name.

## 1.3 One active cohort per participant

A participant may belong to **at most one active Scene-progression cohort per Show**.

Director+ must be able to:

- move participant Ungrouped → Cohort;
- move participant Cohort A → Cohort B;
- move participant Cohort → Ungrouped.

No participant may silently end up in two active Scene-progression cohorts simultaneously.

## 1.4 Cohort scope

Cohorts belong to the **current Show**.

They are not global account groups, My People categories, Production-wide permanent teams, venue roles, or Victory permission roles.

## 1.5 Per-cohort current Scene

Each cohort may have its own current Scene within the current Show.

Changing Cohort 2's Scene must not move Cohort 1 or Ungrouped participants.

## 1.6 Scene source

Scene Configuration initially browses **Scenes attached to the current Show**.

Do not build broad Production-level Scene browsing in Kernel 85.

## 1.7 Scene placement behavior

When a Scene contains saved placements:

- those placements restore for the cohort when that Scene is activated;
- Character/token placements restore only if actually defined;
- Victory does not invent Character positions.

## 1.8 Scene authoring actions

Director+ must have both:

- **Update Current Scene**
- **Save as New Scene**

These are explicit separate actions.

## 1.9 Scene Configuration entry point

Director+ right-clicking the current stage/map should expose:

> **Scene Configuration**

Do not replace all right-click behavior globally; add the Director+ tool action to the current context menu.

## 1.10 Game Status panel

Game Status is a **movable tool panel**.

It should:

- open/close on demand;
- move around the screen;
- display the currently selected cohort;
- display one compact Character block per Character in that cohort;
- read canonical Socio Character/game state;
- update when canonical state changes.

Victory owns the generic movable tool-panel shell. Socio defines the game-specific status content.

## 1.11 Game Status cohort scope

The panel shows the **current cohort**, not every Character in the venue.

Director+ must be able to switch which cohort it displays.

Ungrouped may be selectable if useful.

## 1.12 People access scope

People added through My People / Third Place are granted for the **current Show**.

The presented action is:

> Add this person to the current Show.

---

# 2. Socio Game Status content

The supplied Socio mechanics define eight attribute-based HP pools:

| Attribute | HP Pool |
|---|---|
| Might | Health |
| Intellect | Psyche |
| Grace | Motion |
| Presence | Will |
| Spirit | Essence |
| Resolve | Focus |
| Awareness | Perception |
| Empathy | Heart |

Lore and Crafting are progress-stage systems rather than HP pools.

Kernel 85 should expose these eight HP pools.

## 2.1 Required per-Character status block

At minimum:

- Character name / identifying Face;
- current cohort;
- current Scene if readily available;
- Health current/max;
- Psyche current/max;
- Motion current/max;
- Will current/max;
- Essence current/max;
- Focus current/max;
- Perception current/max;
- Heart current/max;
- current Socio status conditions;
- current Social Stance if canonical state already exists;
- Fate/EP/resources only where canonical state already exists.

## 2.2 HP interaction

Director+ should be able to adjust supported HP values from Game Status.

The panel must call the canonical Socio mechanics/value operation. It must not write directly to a second status store.

If one or more HP pools lack canonical writable operations, audit first and add only the narrow missing canonical operation if bounded.

## 2.3 Status conditions

The supplied Socio source includes statuses such as Raw, Winded, Wounded 1–3, Limping, Burning, Haunted, Numb, Obsessed, Fragmented, Overstimulated, Unmoored, Inspired, Shadowbound, Sanctified, and Radiant.

Do not assume this list is exhaustive or final.

Use a ruleset/package-defined status registry or current canonical Socio representation if one exists.

Required Director actions:

- apply status;
- clear status;
- adjust stack/intensity when supported.

Do not hardcode narrative consequences into the tool.

## 2.4 Collapse/recovery rules

The panel may display depleted-HP/status warnings.

Do not automatically enforce collapse, death, therapy, or recovery consequences unless the canonical Socio operation already does so.

## 2.5 Social Stance

If canonical Social Stance state exists, display it and allow Director correction only through an existing canonical operation.

If it does not exist, do not create a new Stance subsystem merely to decorate Game Status.

---

# 3. Cohort domain model

Repository architecture determines exact schema. Conceptually:

```text
show_cohorts
- id UUID
- show_id
- serial_number
- name
- slug/key
- current_scene_placement_id or current_scene_id
- created_by
- created_at
- updated_at

show_cohort_members
- cohort_id
- user_id and/or show participant identity
- joined_at
```

Use the existing Show participant/Character identity model rather than inventing a parallel identity.

## 3.1 Ungrouped

Ungrouped should preferably be computed as Show participants with no active cohort membership.

Do not create a fake Cohort 0 unless needed.

## 3.2 Serial allocation

Creating a cohort must allocate the next safe serial number transactionally.

Example:

```text
existing: Cohort 1, Cohort 3
next created: Cohort 4
```

Do not reuse deleted numbers merely to fill gaps unless repository conventions require it.

## 3.3 Concurrent creation

Cohort serial allocation must be safe under concurrent Director requests.

Add a unique constraint or transactional allocator as appropriate.

---

# 4. Cohort management UI

Director+ can:

- create cohort;
- see all cohorts in current Show;
- see Ungrouped participants;
- assign participant to cohort;
- move participant between cohorts;
- remove participant back to Ungrouped;
- select active cohort for Scene Configuration;
- select active cohort for Game Status.

Ordinary participants do not gain cohort-management controls.

Prefer a compact tool/control integrated with the Director venue experience rather than a giant admin page.

Presence Tray cohort actions may be added if bounded, but are not required if another compact control is clearer.

---

# 5. Scene Configuration tool

## 5.1 Entry

```text
Director+ right-click stage/map
→ Scene Configuration
```

## 5.2 Required display

Show:

- selected cohort;
- current Scene for that cohort;
- Scenes attached to current Show;
- scene title;
- enough thumbnail/metadata to distinguish Scenes if already available;
- Activate;
- Update Current Scene;
- Save as New Scene.

## 5.3 Activate Scene for cohort

Server validates:

- Director+;
- current Show;
- cohort belongs to Show;
- Scene is attached/eligible for Show.

Then:

- update only that cohort's current Scene;
- project the selected Scene only to members of that cohort;
- leave other cohort Scene projections untouched.

## 5.4 Projection

Two users in the same Show may simultaneously receive different Scene configurations because they are in different cohorts.

No cross-cohort Scene leakage.

## 5.5 Reconnect

On reconnect:

- resolve Show participant;
- resolve active cohort;
- resolve cohort current Scene;
- restore correct Scene projection;
- restore canonical Character status independently.

## 5.6 Ungrouped progression boundary

Ungrouped Catharsis participants may remain on tutorial/default Scene flow.

A participant may not move beyond the tutorial-finish Scene into sustained-play Scene progression until assigned to a cohort.

Enforce server-side using current Catharsis progression authority where possible.

---

# 6. Scene capture and update

## 6.1 Update Current Scene

Director+ can persist the current supported arrangement back to the active Scene.

Capture only canonical supported Scene state: Elements, placements, transforms, visibility, labels, locks/context where supported, and Character/token placements where those are real Scene placement concepts.

Do not capture Presence or transient UI state.

## 6.2 Save as New Scene

Director+ can capture the current arrangement as a distinct new Scene.

Required:

- stable Scene identity;
- editable name;
- current arrangement captured;
- original Scene unchanged;
- new Scene available in Scene Configuration.

## 6.3 Placement ease

Character/token positions should be defined by placing them and saving the Scene. Do not require manual coordinate entry if the stage already knows positions.

---

# 7. People Picker / identity resolution

Kernel 85 must repair the identity mismatch exposed during Storyboard sharing.

Observed behavior:

- a person such as Kyle appears selectable;
- entering the visible handle is rejected as an invalid identifier.

Users should not need to know which opaque identifier a sharing endpoint expects.

## 7.1 Reusable People Picker

Sources:

- My People;
- Third Place / Headshot Commons or current equivalent.

## 7.2 Required person display

Show enough to distinguish the person:

- display name/Face where available;
- handle;
- canonical stable user ID visible/copyable where useful.

Do not expose unrelated private profile data.

## 7.3 Current Show grant

Authorized Director+/Owner can:

```text
select Kyle
→ Add to Current Show
```

Use the current canonical Show admission/access model.

Do not create an alternate membership universe.

## 7.4 Storyboard sharing repair

Storyboard sharing should consume the same canonical selected identity.

At minimum:

- selecting Kyle succeeds;
- no manual opaque-ID transcription required;
- if manual handle entry remains, resolve it canonically or clearly label an ID-only field.

Prefer selection over free-text identity entry.

## 7.5 Individual venue access

Scope through the current Show. Do not create permanent global venue access when the intent is Show participation.

---

# 8. Catharsis bounded repairs

## 8.1 Locked-door milestone copy

Replace the gate message with wording equivalent to:

> **You need a character before you can interact with the locked door. If you already have a character, speak with Kessa first so you leave her stall with more than the clothes on your back. You cannot proceed until you have finished with Kessa.**

Preserve the actual progression rule.

## 8.2 Audition Hall venue completeness

Audition Hall must enumerate **all canonical Venues** visible under current authority rules.

Do not maintain another stale hand-curated subset if a canonical venue registry/query exists.

---

# 9. Sustained-play persistence

Cohort membership and per-cohort current Scene are Show-level persistent state.

Unlike Kernel 83's Group Leader/Current Turn:

```text
end live session
→ reopen Show later
→ cohort membership remains
→ cohort current Scene remains
→ Character HP/status remains
→ Director continues play
```

Presence remains ephemeral.

If current Show already has Story So Far/history support, preserve and reuse it. Do not build an automatic narrative summarizer.

Important Director mutations should produce append-forward history/audit where current conventions support it.

---

# 10. Authority

Director+:

- creates/manages cohorts;
- sets cohort Scene;
- captures/updates Scenes;
- uses Game Status mutation controls.

People Picker grants use existing Show admission authority.

Every mutation validates actor, Show, target participant/Character, cohort, Scene eligibility, and canonical Socio operation server-side.

No client-selected privilege.

---

# 11. Live sync

Required live updates:

- cohort created;
- participant moved;
- cohort Scene changed;
- Scene arrangement update where supported;
- HP/status adjustment;
- Show admission change where current live model supports it.

Reuse current WebSocket/event infrastructure. Do not regress Kernel 84's context-lifecycle repair.

---

# 12. Storyboards boundary

Kernel 85 is not a Storyboards kernel.

Do not pull in the Timeline generic AddColumn boundary bug found in Kernel 84 unless implementation unexpectedly makes it inseparable.

---

# 13. Required repository audit before implementation

Determine:

1. canonical Show participant/admission model;
2. canonical current Scene/Show Scene Placement model;
3. whether Show stage projection can vary by participant today;
4. Catharsis tutorial-finish progression authority;
5. current Character/token placement storage;
6. existing Scene capture/save behavior;
7. which Socio HP/resource values have canonical operations;
8. whether a status/tag/condition registry exists;
9. whether Social Stance is canonical state;
10. how My People and Third Place identify users;
11. identifier Storyboard sharing expects;
12. why visible handle entry failed;
13. how Audition Hall enumerates venues;
14. canonical venue registry/query.

Do not ask Grant to restate repository-answerable facts.

**Audit findings (2026-08-09), see Construction/OperatorLogs/kernel-85-reportback.md for full detail:**

1. No per-Show roster table — a Show inherits its parent Show Run's roster wholesale. `show_run_roster_members` + `location_memberships`, resolved via `participation.ResolveParticipationContext`.
2. `shows.current_show_scene_placement_id` is a single pointer to `show_scene_placements`; Scene content lives in `scene_stage_elements`.
3. Yes — `world.LoadVenueSnapshot` already varies per viewer (role filtering + Kernel 74 `participant_local_projections`). The WS push is a uniform "go refetch" signal; the per-viewer difference happens on the subsequent GET.
4. `backend/internal/tutorial` package, milestone enum + `participant_tutorial_progress` table, keyed `(user_id, character_card_id, show_id, milestone_key)`.
5. `scene_stage_elements` rows of `kind = 'token'` carry normalized `Position`/`Width`/`Height`. No separate Character-to-token binding table.
6. Missing entirely — no bulk "update current Scene" or "save as new Scene" action exists today, only per-element CRUD.
7. None exist — `CharacterCard` has zero attribute/HP fields. Net-new, built inline in this kernel per operator decision (bounded: 8 pools + status table, no character-sheet UI wiring required).
8. Does not exist — net-new, built inline alongside the HP pools.
9. Does not exist as game state (only as manuscript prose in eWrite fixtures) — per §2.5, not built.
10. Both key off a stable account UUID internally but expose only opaque `profile_id` (`player_profile_workbooks.id`) + `stage_name` to the client, never raw UUID or `handle`.
11. `POST /api/storyboards/{board_id}/grants` requires `user_handle`, exact match against `users.handle`.
12. My People's sharing suggestions render `stage_name`; the grant endpoint requires `handle` — two unrelated columns with no bridge.
13. Hardcoded 15-slug `<select>` in `frontend/venues/audition-hall/index.html`, not validated against the `venues` table.
14. `access.ResolveVisibleVenues`'s Operator branch (`SELECT slug, name, kind FROM venues ORDER BY slug`, filtered by `isHiddenMainMapVenueSlug`) is the right base query, generalized for any authenticated user for a request-access context.

---

# 14. Schema guidance

Likely additions may include:

```text
show_cohorts
show_cohort_members
show_cohort_scene_state
```

or equivalent.

Prefer additive migration, UUID identity, Show foreign keys, one-active-cohort-per-participant constraint, safe serial allocation, explicit current Scene reference, and timestamps/audit metadata.

Do not create duplicate Character-health tables.

**Actual schema implemented:** see Construction/Shows/cohort-scene-progression-contract.md and Construction/Socio/game-status-tool-contract.md.

---

# 15. Required backend tests

## Cohorts

- Show participants begin Ungrouped;
- create Cohort 1 and 2;
- serialization unique/stable;
- deletion does not create collision;
- concurrent create safe;
- move Ungrouped → Cohort;
- move Cohort A → B;
- move Cohort → Ungrouped;
- no dual active membership.

## Scene progression

- Cohort 1 Scene A;
- Cohort 2 Scene B;
- changing Cohort 1 → Scene C leaves Cohort 2 unchanged;
- ungrouped tutorial participant remains on tutorial/default projection;
- ungrouped cannot cross sustained-play boundary;
- reconnect restores cohort Scene;
- Scene must belong/be attached to current Show;
- unauthorized assignment rejected.

## Scene capture

- update current Scene persists supported placements;
- save as new Scene creates distinct Scene;
- original unchanged;
- defined Character/token placements restore;
- undefined placement not invented.

## Game Status

- all eight HP pools read correctly;
- Director HP change uses canonical operation;
- status apply/clear uses canonical operation;
- unsupported direct field mutation rejected;
- data filtered to selected cohort;
- state persists across session end/re-entry.

## People Picker

- My People resolves canonical user ID;
- Third Place resolves canonical user ID;
- selected person added to current Show;
- Storyboard share succeeds with selected identity;
- malformed arbitrary ID rejected;
- unauthorized Show grant rejected.

## Audition Hall

- canonical query returns all eligible venues;
- Audition Hall renders full eligible set;
- hidden/private venues remain correctly filtered.

## Catharsis milestone

- no Character → correct gate;
- Character but Kessa incomplete → correct message;
- Kessa completed → progression allowed.

---

# 16. Required browser proof

Playwright is mandatory.

Prove:

1. Catharsis participants begin Ungrouped.
2. Ungrouped participant can use tutorial path.
3. Ungrouped participant cannot move beyond tutorial-finish sustained-play boundary.
4. Director+ creates Cohort 1.
5. Director+ creates Cohort 2.
6. Cohort names/serials correct.
7. Move Player A to Cohort 1.
8. Move Player B to Cohort 2.
9. Move Player A Cohort 1 → 2.
10. Return Player A to Ungrouped.
11. Right-click stage/map exposes Scene Configuration to Director+.
12. Ordinary player lacks Director mutation controls.
13. Scene Configuration lists current Show Scenes.
14. Set Cohort 1 to Scene A.
15. Set Cohort 2 to Scene B.
16. Both users simultaneously see different correct Scenes.
17. Move Cohort 1 to Scene C; Cohort 2 unchanged.
18. Reconnect restores correct cohort Scene.
19. Update Current Scene persists arranged placements.
20. Save as New Scene creates new selectable Scene.
21. Defined placements restore.
22. Game Status tool opens.
23. Game Status panel moves.
24. Game Status switches cohorts.
25. It shows Characters only from selected cohort.
26. All eight HP pools render for a real Socio Character.
27. Director adjusts at least one HP pool canonically.
28. Relevant second client sees canonical update.
29. Director applies and clears at least one status.
30. Session closes/reopens and cohort + Scene + health/status persist.
31. My People picker selects a person.
32. Third Place picker selects a person.
33. Handle and canonical ID display appropriately.
34. Add to Current Show succeeds without opaque-ID guessing.
35. Selected identity can be shared to Storyboard.
36. Prior handle/ID mismatch is resolved or clearly replaced.
37. Audition Hall shows every eligible canonical venue.
38. Locked-door milestone shows corrected copy.
39. Forged unauthorized cohort/Scene/status mutation is rejected.
40. No cross-cohort Scene leakage occurs.

No PASS based only on API evidence.

---

# 17. Required artifacts

```text
Construction/Kernels/Kernel 85 — Socio Sustained Play — Cohorts, Scene Progression & Game Status.md
Construction/Socio/sustained-play-contract.md
Construction/Shows/cohort-scene-progression-contract.md
Construction/Scenes/scene-configuration-tool.md
Construction/Socio/game-status-tool-contract.md
Construction/Identity/people-picker-contract.md
Construction/OperatorLogs/kernel-85-reportback.md
```

Update existing canonical docs rather than duplicating them.

---

# 18. Required evidence

Baseline:

```bash
cd /opt/victory
git status --short
git rev-parse HEAD
git branch --show-current
docker ps --format '{{.Names}}	{{.Status}}	{{.Ports}}'
```

Backend:

```bash
TEST_DATABASE_URL=... GOCACHE=/tmp/victory-gocache go test -count=1 -timeout=600s ./...
```

Prove additive migration/fresh bootstrap, Playwright sustained-play browser journey, and forged negative-authority rejection.

---

# 19. Pass criteria

Kernel 85 passes when:

- Catharsis participants begin Ungrouped;
- cohort membership is required beyond tutorial finish;
- Director+ creates safe serialized cohorts;
- one participant belongs to at most one active cohort per Show;
- Director+ moves people between cohorts/Ungrouped;
- cohorts hold independent current Scenes;
- changing one cohort does not alter another;
- Scene Configuration is available from Director+ stage/map right-click;
- Scene picker lists current Show Scenes;
- activation is server-authoritative and cohort-scoped;
- reconnect restores correct cohort Scene;
- Director can Update Current Scene;
- Director can Save as New Scene;
- defined placements restore;
- undefined Character placement is not invented;
- movable Game Status exists;
- Game Status is cohort-filtered;
- all eight Socio HP pools display;
- supported HP/status changes use canonical mechanics operations;
- no second health/status database exists;
- Show/cohort/Character state survives session break appropriately;
- People Picker resolves canonical identities from My People/Third Place;
- current Show access can be granted without opaque-ID guessing;
- Storyboard sharing uses selected canonical identity;
- Audition Hall shows all eligible canonical venues;
- locked-door copy is corrected;
- unauthorized mutations fail server-side;
- multi-user browser proof shows two cohorts on different Scenes simultaneously;
- no unrelated redesign occurs.

---

# 20. Partial rules

Mark PARTIAL if:

- cohorts work but Scene projection remains global;
- Scene picker works only through a separate admin page;
- Game Status is read-only because canonical mutation paths are missing;
- fewer than all eight canonical HP pools can be shown;
- statuses use a temporary parallel store;
- People Picker cannot grant current Show access;
- Storyboard identity mismatch remains;
- reconnect loses cohort Scene;
- Scene capture cannot restore saved placements.

A PARTIAL report must name the bounded continuation.

---

# 21. Fail rules

Mark FAIL if:

- moving one cohort changes everyone's Scene;
- participant can silently belong to multiple active cohorts;
- client chooses trusted cohort/Scene state without server validation;
- Game Status writes to duplicated health fields;
- Director controls leak to unauthorized users;
- Show grants become global/permanent contrary to scope;
- Storyboard sharing accepts ambiguous identity unsafely;
- real user content is deleted during tests;
- Catharsis tutorial breaks;
- Kernel 84 security/runtime repairs regress;
- Grant is locked out.

---

# 22. Non-goals

Kernel 85 does not:

- build a complete Socio rules engine;
- implement automatic initiative or turn rotation;
- automate Narrator judgment;
- implement all Stance mechanics;
- implement full action-economy UI;
- implement Lore/Crafting progress unless already canonical and nearly free to display;
- build Production-wide Scene browsing;
- build global reusable social groups;
- build permanent venue grants;
- build a new social network;
- build Cartograph drawing;
- fix Timeline AddColumn unless inseparable;
- redesign all Catharsis tutorial content;
- build a giant Director dashboard;
- rebuild Scene architecture from scratch.

---

# 23. Kernel-size guardrail

Kernel 85 is one coherent sustained-play vertical:

> **Director organizes players into cohorts and advances those cohorts independently through Show Scenes while retaining canonical Socio game state.**

People Picker and Catharsis/Audition Hall repairs belong only because they directly support that live loop.

If missing canonical Socio health operations become a substantial subsystem, split them instead of burying them.

**Operator decision (2026-08-09):** the HP pool + status registry subsystem, though entirely net-new, was scoped as bounded (8 current/max pools, no character-sheet wiring, simple canonical get/set) and built inline rather than split into a sub-kernel.

---

# 24. Immediate operator outcome

At completion, Grant should be able to answer:

1. Can I leave everyone ungrouped while they complete Catharsis?
2. Can I prevent ungrouped players from moving beyond tutorial finish?
3. Can I create Cohort 1, Cohort 2, Cohort 3 automatically?
4. Are those identities safe and non-colliding?
5. Can I move Kyle between Ungrouped and cohorts?
6. Can Cohort 1 and Cohort 2 be on different Scenes simultaneously?
7. Can I right-click the stage and open Scene Configuration?
8. Does it show Scenes from the current Show?
9. Can I move only one cohort to another Scene?
10. Do other cohorts remain where they were?
11. Do saved placements restore when defined?
12. Can I update the current Scene from arranged placements?
13. Can I save that arrangement as a new Scene?
14. Does reconnect return each player to the correct cohort Scene?
15. Can I open and move Game Status?
16. Can I point it at a cohort?
17. Does it show all eight Socio HP pools?
18. Can I adjust HP/statuses through canonical mechanics state?
19. Do those changes persist through Scene changes and a session break?
20. Can I find Kyle through My People or Third Place?
21. Can I see enough identity information to know I have the right Kyle?
22. Can I add him to the current Show without guessing an opaque ID?
23. Can I share a Storyboard using the same selected identity?
24. Does Audition Hall show every eligible canonical venue?
25. Does the locked-door message clearly explain Character + Kessa requirements?
26. Do unauthorized users remain blocked from Director actions?
27. Can two browsers prove two cohorts genuinely seeing different Scenes?
28. Does this feel like I can now **run Socio**, rather than merely walk somebody through the tutorial?

Kernel 85 succeeds when Victory supports **ongoing Socio play for multiple independently progressing groups inside one Show** while keeping Scene state, Character status, identity, and authority coherent.
