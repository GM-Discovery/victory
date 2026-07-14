# Kernel Report Back — Kernel 70: Persistent Show Stage, Rehearsal Workspace, and Go Cue Foundation

## Status

**PASS** — completed and committed as `920aeb7` on 2026-07-14.

This reportback was reconstructed during the post-Kernel-70 documentation audit from the committed diff, tests, migrations, Dictionary, operator log, and operator notes. It records implemented truth; it does not invent a missing pre-build specification.

## Delivered

- Corrected Scene ownership from Production-exclusive to Location-scoped. `scenes.location_id` is canonical; `source_production_id` is optional provenance.
- Added persistent Show stage ownership through `actions.show_id`, with world snapshots folding linked Show and current Session actions.
- Added a current-Scene pointer, backstage Show variables, and projection invalidation.
- Added Cues and Cue execution records with ordered actions, trigger scopes, idempotent GO, and recorded outcomes.
- Implemented `go_to_scene`, `emit_game_event`, and `set_show_variable`.
- Added curated player Cue listing and stage buttons in First Theater and Catharsis.
- Added Crew's non-destructive edit/GO boundary while preserving destructive/direct management authority.

## Important Corrections And Safety Findings

- Migration 043 guards its rename/backfill sequence, and migration 042's stale index reference was neutralized so full migration replay remains safe.
- Show variables/current placement initially risked leaking through the Audience Show Program. Both fields now use `json:"-"` and are exposed explicitly only by backstage responses.
- First Theater/Catharsis `runtime/` modules are live through the dynamically loaded sibling `runtime.js`; they are not orphaned files.

## Authority And Privacy Proof

- Audience cannot trigger any Cue and receives no Cue internals.
- Player Cue responses expose only `id` and display `label`, filtered through current authority.
- Crew can author non-destructive Cues and press GO, but cannot directly set current Scene, edit a Base Scene, archive a Show, or perform destructive management.
- Cross-Show and cross-Location targets are rejected.
- Backstage Show variables and current placement remain absent from audience program JSON.

## Validation

- `go build` and `go vet`: clean.
- Full Go suite with dedicated `TEST_DATABASE_URL`: green, including 40+ new tests.
- `scripts/smoke/fresh-install.sh --local`: PASS from an empty database with migrations 043–045.
- First Theater Node suite: unchanged at 46 passing and 9 pre-existing unrelated `dice.test.js` failures.
- Frontend syntax/code-path checks: PASS.
- Browser screenshot automation: not performed; this remains the evidence gap.

## Explicitly Deferred

- Visual Base Scene composition/capture and Show-specific visual overrides.
- Binding Scenes to `elements`, `venue_layout_elements`, and the existing stage-object model.
- `reveal_object`, `hide_object`, `enable_interaction`, and `disable_interaction` Cue actions.
- Browser screenshot proof of rehearsal messaging and Cue controls.

## Next Recommended Step

Implement visual Scene composition/capture on the existing stage-object authority model without creating a second object model. A smaller closure pass may add Kernel 70 browser evidence and repair the nine pre-existing frontend dice-test failures.
