# Detail View Contract — Kernel 87

Canon for "Draw Big, Return Small" — opening a map region large enough to draw fine detail into, without ever creating a second document. Frontend only: `frontend/lib/stage-runtime/drawing.js`'s `openDetailViewFromSelection`/`closeDetailView`.

## 1. What Detail View is

**A camera transform over the same canonical drawing objects, not a raster copy and not a second scene graph.** Opening Detail View does not create any new PIXI container, does not clone any `PIXI.Graphics` object, and does not write any new/different geometry to the backend. The exact same `layerNode` (child of the shared `drawingWorldLayer`, itself a child of the shared `worldLayer` every other stage object pans/zooms with) that renders the object at normal map scale is what renders it at Detail View scale — only the camera's zoom/pan changes.

This satisfies kernel §6's hard requirement directly: *"Do not rasterize the selected region. It is a camera/editing transform over the same canonical drawing objects."*

## 2. Open

`openDetailViewFromSelection()`:

1. Resolves a region — either the bounding box of the currently-selected drawing object (`objectBounds(obj)`), or (if nothing is selected) the full playable map bounds as a fallback rectangle.
2. Records `state.detailView = {previousView}` (the camera's pan/zoom *before* opening, so Close can restore it exactly — local UI state only, nothing persisted server-side; Detail View is not a durable mode any object is "in").
3. Calls the shared camera's own `setView({zoomRelativeToFit, panX, panY})` (`frontend/lib/victory-stage-camera.js`, the same API the runtime's own pan/zoom controls use) to actually zoom/center on the region, computed with a small 15% margin so the region isn't flush against the viewport edge.

This is a real camera move, not a stub: `drawing.mount()` receives a `getCamera` accessor (`runtime.js` passes `() => stageCamera`, the same camera instance every other pan/zoom/fullscreen interaction already uses), so Detail View and ordinary map navigation are provably the same camera, not two independent view models that could drift.

One grid cell is supported by selecting a drawing object sized to one cell (or, for a gridless map, any object) and opening Detail View on its bounds; rectangular region selection is the same mechanism with an arbitrary bounding rectangle. There is no separate "cell-select" vs "region-select" code path — both resolve to the same `{x,y,width,height}` region shape.

## 3. Edit

All drawing tools work identically inside Detail View, because nothing about the tool code changed — `finishShape`, `onPointerDown`/`onPointerMove`/`onPointerUp`, and every CRUD call still write through the exact same `POST/PATCH/DELETE /api/sessions/{id}/drawing-objects` endpoints, using the exact same world coordinate space. The only thing Detail View changes is what fraction of that world coordinate space is currently mapped onto the visible screen pixels (the camera's zoom level) — a new stroke drawn while "inside" Detail View lands at the correct absolute world coordinates the same way a stroke drawn at normal zoom does, because `stagePointFromClient` (the existing `runtime.js` screen→world conversion every tool already goes through) is zoom-aware by construction, not something Detail View has to special-case.

## 4. Return

`closeDetailView()`:

1. Restores the camera to `state.detailView.previousView` via the same `setView` call (or falls back to `camera.fit()` if no camera was available when Detail View opened).
2. Clears `state.detailView`.
3. **The drawing objects themselves are untouched** — there is no "merge back" step, no re-projection, no thumbnail generation, because they were never in a different coordinate space or a different document to begin with. Objects "remain in correct map-relative positions" and "scale naturally back into normal map view" as a structural consequence of never having moved, not as a feature that had to be separately built and could fail.

Text/stamps/shapes remain full object/vector state throughout — they were never flattened to pixels at any point, matching kernel §6's "no fuzzy thumbnail replacement."

## 5. Shared view

**Only the active editor's own camera moves.** `cartograph:detail-view-open`/`close` events are local to that browser tab; no WS message is broadcast, and no other connected viewer's camera is affected. This satisfies the kernel's minimum bar ("At minimum, active editor can use Detail View locally") but **not** the preferred-if-inexpensive stretch goal ("invite/follow other authorized viewers into the same region") — that would need a new targeted WS event type and was judged not inexpensive enough to include this kernel without risking the "do not force everyone's camera unless Director authority explicitly supports it" boundary. Recorded as a known gap, not a silent omission.

## 6. Gridless

For gridless maps, Detail View's rectangular-region path is exactly the same code as the grid-cell path (a region is a region, gridded or not) — kernel §6's "allow rectangular region selection" for gridless maps is satisfied by construction, not a separate feature. Arbitrary freeform (non-rectangular) region selection was not built, per the kernel's own explicit permission not to block on it.

## 7. What would make this claim false (kernel §21/23 self-check)

Per the kernel's own fail criteria, Detail View would FAIL this kernel if it "creates a second document/state" or "destroys editability." Neither is true here: `git grep` for any Detail-View-specific object table, duplicate geometry field, or raster-export-on-open code path in `backend/internal/drawing/` returns nothing, because none exists — Detail View is purely a frontend camera-event emission with no backend surface at all.
