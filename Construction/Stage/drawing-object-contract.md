# Drawing Object Contract — Kernel 87

Canon for Cartograph's shared collaborative drawing objects: schema, authority, scope, z-order/lock, and the library decision. Backend: `backend/internal/drawing/`. Frontend: `frontend/lib/stage-runtime/drawing.js`.

## 1. Object model

One row in `drawing_objects` (migration `098_kernel87_cartograph_drawing.sql`) per canonical mark:

```text
id, show_id, cohort_id (nullable), creator_user_id, creator_character_id (nullable),
object_type, geometry (jsonb), stroke_color, fill_color (nullable), stroke_width,
opacity, line_style, rotation, z_order, locked, text_content, stamp_key,
deleted_at (soft delete), created_at, updated_at
```

`object_type` is one of `freehand | line | polyline | rectangle | ellipse | polygon | text | stamp`, CHECK-constrained.

**Geometry is always expressed in the same map-relative "world" coordinate space** tokens and index cards already use (`world_x`/`world_y` in `frontend/lib/stage-runtime/geometry.js` — resolution-independent stage pixels, not browser/viewport pixels). Pan/zoom/fullscreen/Detail View never rewrite a stored point; they only change the camera transform reading it. This is enforced structurally, not by convention: the same PIXI `worldLayer` container that carries tokens and pinned dice also carries `drawingWorldLayer` (`runtime.js`), so drawings inherit the shared camera for free.

Per-type geometry shapes (all point/vertex objects use `{x,y}` pairs):

- `freehand` / `polyline`: `{points: [{x,y}, ...]}` (≥2 points, ≤2000 — `MaxPointsPerStroke`)
- `line`: `{points: [{x,y},{x,y}]}` (exactly 2)
- `polygon`: `{points: [...]}` (≥3, ≤500 — `MaxVertices`)
- `rectangle` / `ellipse`: `{x, y, width, height}`
- `text`: `{x, y}` + `text_content` (≤500 chars — `MaxTextLength`)
- `stamp`: `{x, y, scale}` + `stamp_key` (must be in the fixed palette, `drawing.ValidStamps`)

## 2. Authority modes (kernel §9)

Three modes, stored per-Show in `stage_drawing_settings.drawing_mode`:

- `director_only` — Director+ only (default).
- `turn_leader` — Director+, plus whoever currently holds Group Leader or Current Turn.
- `freeform` — any non-Audience session participant.

Director+ (`session_participants.role IN ('director','producer')`, resolved via `rollaudience.IsDirectorPlus`) can **always** draw regardless of mode.

**Turn/Leader reuses Kernel 83's `venuecoordination.Registry` directly** — no new initiative/turn infrastructure was built. The main stage's coordination "venue session" key is the live Session ID (`drawing.StageVenueSessionID`), mirroring exactly how Storyboards keys by board ID (`storyboards.StoryboardVenueSessionID`). The same process-wide `venuecoordination.Registry` instance (`main.go`'s `venueCoordination`) now serves both Storyboards and the main stage — different UUID namespaces, no collision risk. Assignment (`AssignGroupLeader`/`AssignCurrentTurn` in `backend/internal/drawing/coordination.go`) requires the target to currently be a `session_participants` row (not a live-watcher set, since the main stage tracks participation as durable rows, unlike Storyboards' pure live-watcher model).

## 3. Editing ownership (kernel §1)

- Creator may edit/delete/lock/unlock their own **unlocked** object.
- Director+ may edit/delete/unlock/reassign **any** object, locked or not.
- An ordinary Player can never edit another Player's object, in any mode — checked in `drawing.CanEditObject`, independent of `CanDraw` (drawing-mode authority governs *creating*; ownership governs *editing*, and the two are never conflated).

This is what makes per-user undo safe without any explicit undo endpoint: the client re-issues a normal delete/update call for its own last action, and the server's ownership check makes it structurally impossible for that call to revert another user's later work — there is no separate "undo log" to get out of sync.

## 4. Scope (Cohort vs Show)

Reuses `backend/internal/rollaudience` (Kernel 86) rather than re-deriving Cohort membership — the same package that already resolves Show/Cohort audience for dice-roll projection. `rollaudience.Resolve(ctx, pool, sessionID, actorID, ModeCohort)` narrows to the actor's current Cohort (or falls back to Show scope if ungrouped or the session has no Show). Default scope for Cartograph-style play is Cohort per kernel §1.

**No-leak guarantee**: `drawing.List` (the one read path used for the initial GET, every reconnect, and every "something changed, refetch" live update) filters Cohort-scoped objects out for any viewer who is neither Director+ nor a member of that same Cohort. There is exactly one filtering code path — no separate "snapshot" or "history" object model to independently get right or wrong.

## 5. Live delivery: invalidate, don't broadcast payload

Every mutation (create/update/delete/lock/z-order/settings/coordination) calls `network.BroadcastShowStageInvalidation(..., "drawing_changed")` — the same content-free `show/stage_updated` WS event Kernel 70's Cue/Scene changes already use. The frontend's existing `show_stage_updated` handler (`session-sync.js`) now also calls `window.VictoryStageDrawing.refresh()`, which re-fetches the scope-filtered object list.

This was a deliberate simplicity/safety trade against a fine-grained payload-push design: a payload broadcast would need its own per-recipient Cohort filtering logic duplicated from `List`, with a second chance to leak. An invalidate-and-refetch round trip is not truly per-stroke live streaming (kernel §3's "preferred: viewers see a stroke while it is being drawn" is **not** implemented — see §7 below), but every completed object is authoritative and reaches every authorized viewer promptly, and Cohort scoping can only ever be wrong in one place (`List`), not two.

## 6. Z-order and locking (kernel §5)

Z-order is a plain integer per Show (all objects on one Show's map share one axis, regardless of Cohort — a Director+ correcting stacking order always sees consistent results). Four operations, all in `drawing.Reorder`:

- **front**: set to `max(z_order)+1` in the Show.
- **back**: set to `min(z_order)-1`.
- **forward/backward**: swap with the nearest neighbor above/below, in one transaction.

Pinned spatial dice from Kernel 86A render **above** drawings by PIXI insertion order, not zIndex sorting: `worldLayer.addChild(mapLayer, gridLayer, pinnedObjectLayer, drawingWorldLayer, diceWorldLayer)` in `runtime.js` — `drawingWorldLayer` is deliberately inserted between `pinnedObjectLayer` and `diceWorldLayer`.

## 7. Library decision (kernel §15)

**No third-party drawing/vector library was added.** Evaluated and rejected:

- The frontend has **no build step and no package.json** (repo-wide constraint, confirmed by audit) — most mature browser vector libraries (Fabric.js, Konva, paper.js, Perfect Freehand as an npm dependency) assume a bundler or at minimum a package manager to pin/audit versions. Vendoring a raw UMD/IIFE build file is possible but adds an unaudited third-party surface with its own licensing/update burden for a kernel whose non-goals explicitly exclude Bezier node editing, pressure sensitivity, and texture brushes — exactly the capabilities those libraries exist to provide.
- PIXI.Graphics (already the sole rendering dependency, already vendored and understood by every other stage-runtime module) is sufficient for every required tool: freehand (polyline stroke), line, polyline, rectangle, ellipse, polygon (fill + stroke path), text (`PIXI.Text`), and stamp (a small fixed vector-glyph set, drawn with `PIXI.Graphics`, never uploaded pixels).
- A second scene graph (e.g. an SVG overlay library) would violate the kernel's own constraint ("whether it would force a conflicting second scene graph") — drawings must live in the *same* PIXI world container as tokens/dice for the shared pan/zoom/Detail-View camera to carry them for free.
- Freehand smoothing is a simple client-side point-distance decimation (drop points closer than 2px to the last kept point, cap at `MaxPointsPerStroke`), not a spline-fit smoothing algorithm — adequate for drawing-as-play sketching, not claimed to be pen-quality.

**Conclusion**: hand-rolling on top of the existing PIXI dependency was the correct call for Kernel 87's actual scope. Revisit only if a future kernel needs true Bezier/node editing (explicitly out of scope here).

## 8. Security/abuse bounds (kernel §16)

Enforced server-side in `drawing.Create`/`Update` (never trusted from the client beyond validation):

- `MaxPointsPerStroke = 2000`, `MaxVertices = 500`, `MaxTextLength = 500`, `MaxStampKeyLength = 64`
- `MaxObjectsPerShow = 5000` (a hard ceiling per Show, checked before insert)
- Every point/vertex must be a finite `{x,y}` number pair within a sane absolute bound (excludes NaN/Infinity and absurd coordinates)
- Colors validated against a hex-color regex; line style against a fixed enum; stamp key against the fixed palette
- Object type validated against the fixed CHECK-constrained enum
- Live-stroke point streaming was **not implemented** (see §7 of the measured-tabletop/detail-view work — deferred as bounded polish per kernel §0/§22, not required for PASS), so there is no unbounded WS flood surface from mid-stroke point spam to begin with.

## 9. What is deliberately NOT built

Per kernel §24 non-goals, confirmed absent: Bezier node editing, pressure sensitivity, texture brushes, gradients, a full layer palette, an asset marketplace, procedural/AI map generation.
