# Scene Configuration Tool (Kernel 85)

## What existed before

`backend/internal/scenes/composition.go` already had per-element CRUD (`CreateSceneStageElement` for the Scene's reusable Base layer, `CreatePlacementStageElement` for a Show-specific override layer scoped to one placement) and `LoadResolvedComposition` to merge them for display. There was no bulk "commit this arrangement" or "snapshot this arrangement as a new Scene" action — both were genuinely missing, confirmed by repository audit before this kernel began.

## Entry point

Director+ right-clicking the stage/map opens the shared Pixi stage engine's existing context menu (`backend/internal/... ` — frontend side: `frontend/lib/stage-runtime/logic.js`'s `resolveStageObjectActions`, `kind === "stage"` branch). Kernel 85 adds one menu item there, gated on the same `canManageIndexCards` authority (producer/director/operator) every other item in that branch already uses:

```js
push("open-scene-configuration", "Scene Configuration", "director", { disabled: !canManageIndexCards });
```

The click is routed in `action-router.js` to `window.VictoryKernel85Tools.openSceneConfiguration()` — a self-contained module (`frontend/lib/stage-runtime/kernel85-cohort-tools.js`) that the generic stage engine never imports anything from. It reads the current Show id and Director+ gate through one narrow read-only bridge the engine exposes (`window.VictoryStageKernel85Bridge`, added in `runtime.js`), keeping cohorts/Scenes/Socio knowledge entirely out of the generic engine — the same "engine is generic, venue/game layer reads it" convention `VictoryStageVenue` already established.

## Update Current Scene: promotion, not delete-and-recreate

The naive implementation — delete every Base-layer element and Show-layer element for this placement, then re-insert the resolved merge as new Base rows — would silently drop `stage_element_bindings` (interaction hotspot bindings cascade-delete with their element). `scenes.UpdateCurrentScene` instead does the minimum real mutation:

```sql
UPDATE scene_stage_elements SET show_scene_placement_id = NULL, updated_at = NOW()
WHERE show_scene_placement_id = $1
```

Every Show-layer element scoped to this placement is *promoted* into the Scene's Base layer in place — same id, same bindings, same everything, just no longer placement-scoped. Base-layer elements that already existed are untouched. `LoadResolvedComposition` for this placement returns the identical element set before and after (proven in `scenes/capture_test.go`'s `TestUpdateCurrentScenePromotesShowLayerIntoBase`) — nothing is invented, nothing is duplicated, and a bound interaction hotspot survives being "made permanent."

## Save as New Scene: an explicit snapshot

`scenes.SaveArrangementAsNewScene` resolves the placement's current Base+Show composition and copies each element into a brand-new `scenes` row's Base layer with fresh ids. The original Scene and placement are never touched. This deliberately does **not** copy `stage_element_bindings` — bindings reference the original element ids, and re-binding an interaction on a freshly authored Scene is the same authoring step as binding one on any newly created Scene. This mirrors Kernel 82's Storyboard-template convention ("boards are snapshots/instances after creation, they do not dynamically inherit later template changes") applied to Scenes: a save-as-new is a point-in-time copy, not a live link.

## Authority: narrower than general composition editing

Both actions require `showruns.CanManageShowRun` (Director+/Producer/Operator) via a new `requireDirectorPlus` helper in `scenes/capture.go` — deliberately **not** `canManageComposer`, which also grants Crew a non-destructive-edit right for ordinary element CRUD. Kernel 85 §10 scopes Scene capture to Director+ specifically; Crew's existing composer rights are unchanged for individual element edits.
