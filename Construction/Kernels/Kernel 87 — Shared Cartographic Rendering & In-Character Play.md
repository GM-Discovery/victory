# Kernel 87 — Shared Cartographic Rendering & In-Character Play

**Status:** READY FOR IMPLEMENTATION  
**Type:** Shared stage authoring / collaborative cartography / play-surface expansion  
**Sequence position:** After Kernel 86A  
**Primary proof:** Cartograph-style drawing-as-play

## 0. Kernel contract

Kernel 87 establishes persistent collaborative drawing on the canonical shared stage, suitable for drawing-as-play.

It also adds a bounded **In Character** chat tab using the player's canonical Show-selected Character as speaker identity.

Core vertical:

```text
authorized player/director draws
→ server validates authority + scope
→ canonical drawing object persists
→ authorized viewers render it live
→ player can edit it
→ region can open large in Detail View
→ map can be measured using physical scale
→ drawings survive reconnect
→ current map exports to PNG
```

This is not Photoshop, a scripting language, a full asset system, or a CRPG rules engine.

---

# 1. Locked product decisions

## Drawing authority
- Director+ can always draw.
- Current Turn holder / Group Leader can draw when drawing is enabled.
- Optional Freeform mode may allow all eligible participants.

## Editing ownership
- Creator edits/deletes their own marks by default.
- Director+ may edit/delete/unlock/reassign.
- Ordinary Players do not edit another Player's marks by default.

## Scope
Support:
- **Cohort**
- **Show**

Default for Cartograph-style play: **Cohort**.

Do not leak Cohort drawings through live delivery, reconnect, snapshot, history, or export.

## Detail View
Support:
- one grid cell;
- rectangular selected region.

This is first-class **Draw Big, Return Small** behavior.

Do not rasterize the selected region. It is a camera/editing transform over the same canonical drawing objects.

## Live stroke
Preferred: viewers see a stroke while it is being drawn.

If streaming live stroke points materially complicates networking/persistence, completed-stroke sync is acceptable for Kernel 87. Record live streaming as bounded polish rather than new architecture.

## Stamps
Include a small production-provided stamp palette.

Do not build a generalized asset marketplace/library.

## Visibility seam
Drawing objects should be compatible with future generalized participant visibility, but Kernel 87 does not block on that system.

Current drawing scopes are Cohort and Show only.

## Export
Support **Export Current Map as PNG**.

Native drawing state remains separately editable.

## In Character chat
Add an **In Character** tab. Server resolves the speaker Character; client may not impersonate arbitrary Characters.

---

# 2. Canonical drawing object model

Do not bake every mark destructively into the map bitmap.

Support these object families:

```text
freehand stroke
straight line
polyline/path
rectangle
ellipse/circle
polygon
text/label
stamp
```

Conceptual common state:

```text
DrawingObject
- id
- show_id
- cohort_id optional
- scene/show spatial context
- creator_user_id
- creator_character_id optional
- type
- geometry
- stroke_color
- fill_color optional
- stroke_width
- opacity
- line_style optional
- rotation
- z_order
- locked
- created_at
- updated_at
```

Exact schema follows repository architecture.

Persist map/stage-relative coordinates, not browser pixels.

Pan/zoom/fullscreen/theatrical fit must not change object placement.

---

# 3. Required drawing tools

## Select
Support:
- single select;
- multi-select where practical;
- move;
- resize;
- rotate;
- recolor;
- duplicate;
- delete;
- lock/unlock;
- bring forward;
- send backward;
- bring to front;
- send to back.

No node-by-node Bezier editing in 87.

## Free Draw
- smooth freehand;
- width;
- color;
- opacity;
- one completed stroke = one selectable object.

No pressure sensitivity requirement.

## Straight Line
- point-to-point;
- color;
- width;
- opacity;
- optional simple dash style if inexpensive.

## Polyline / Path
- successive points;
- finish path;
- selectable/movable as one object.

Use cases: roads, rivers, borders, routes.

## Rectangle
Stroke, fill, opacity, resize, rotate.

## Ellipse / Circle
Same styling/editing model.

## Polygon
Vertices, close polygon, stroke, fill/no fill, opacity, move/resize/rotate where practical.

## Text / Label
- click to place;
- enter/edit text;
- font size;
- color;
- move;
- rotate.

No rich-text editor.

## Stamp
- select from palette;
- click to place;
- move;
- scale;
- rotate;
- delete;
- tint only if current format makes it cheap/safe.

Stamp remains a reusable symbol reference, not destructive pixels.

## Eraser
Prefer object-aware erase/delete.

Partial freehand-stroke erasing may be deferred.

## Undo / Redo
Primarily per-user.

Do not globally undo another user's later work when a Player undoes their own stroke.

Director+ may have broader correction authority.

---

# 4. Appearance controls

For relevant tools:
- stroke color;
- fill color;
- no fill;
- width;
- opacity;
- simple solid/dashed/dotted line if inexpensive.

Provide a small recent/saved palette for the current drawing context.

No gradients, texture brushes, or professional color system.

---

# 5. Z-order and locking

Do not build Photoshop-style layer management.

Required:
- bring forward;
- send backward;
- bring to front;
- send to back;
- lock;
- unlock.

Pinned spatial dice from Kernel 86A should normally render above drawings.

---

# 6. Detail View — Draw Big, Return Small

## Open
Select:
- one grid cell; or
- rectangular region.

Choose **Open Detail View**.

Victory enlarges that exact map region into a focused drawing workspace.

## Edit
All drawing tools work there.

The view uses the same canonical drawing objects and coordinate space.

It is not a separate document and not a raster copy.

## Return
When Detail View closes:
- objects remain in correct map-relative positions;
- they scale naturally back into normal map view;
- they remain individually editable;
- text/stamps/shapes remain object/vector state;
- no fuzzy thumbnail replacement.

## Shared view
At minimum, active editor can use Detail View locally.

Preferred if inexpensive:
- invite/follow other authorized viewers into the same region.

Do not force everyone's camera unless current Director authority explicitly supports it.

## Gridless
For gridless maps, allow rectangular region selection.

Do not block 87 on arbitrary freeform region selection.

---

# 7. Measured tabletop

Kernel 87 should turn the existing grid/runtime into a measured tabletop unless repository audit reveals a concrete dependency that makes it unsafe.

Required chain:

```text
grid/map
→ physical scale
→ unit
→ measurement
```

## Physical scale
Director can define examples such as:
- `1 square = 5 ft`
- `1 hex = 2 miles`

Do not hardcode feet only.

## Gridless scale
Allow calibration against map coordinates using the simplest current-runtime-compatible model.

## Measure tool
Support:
- straight point-to-point;
- multi-segment/path;
- displayed distance;
- configured unit.

## Diagonal policy
For square grids, define/configure a diagonal policy.

Do not silently assume one tabletop convention if a simple setting is feasible.

At minimum document the initial policy.

## Hex
Respect current hex orientation/config.

## Ephemeral default
Measurement is ephemeral by default.

Optional **Pin Measurement** is acceptable if easy, but pinned measurement is reference/presentation state, not drawing truth.

## Explicit exclusions
No:
- movement cost;
- pathfinding;
- movement enforcement;
- initiative;
- attack-range automation.

---

# 8. Collaborative synchronization

Drawing creation/edit/delete is server-authoritative.

Server validates:
- identity;
- Show;
- Cohort;
- drawing mode;
- object ownership or Director override;
- payload/geometry bounds;
- object existence/version where appropriate.

Authorized clients receive updates.

If live stroke streaming is implemented:
- batch/limit points;
- validate scope;
- persist final stroke;
- handle mid-stroke disconnect safely;
- rate-limit.

If not:
- local preview during stroke;
- canonical broadcast on completion.

Reconnect restores all persisted drawing state.

---

# 9. Drawing authority modes

Provide a small mode model:

```text
Director Only
Turn/Leader
Freeform
```

### Director Only
Director+ only.

### Turn/Leader
Director+ plus authorized Current Turn holder / Group Leader.

### Freeform
All eligible participants in current drawing scope.

Use existing Group Leader/Current Turn infrastructure.

Do not build initiative.

---

# 10. In Character chat

Add distinct tab:

> **In Character**

Existing OOC/venue chat remains separate.

On send:

```text
authenticated user
→ server resolves current Show/participation Character
→ stores accountable user identity
→ stores/presents Character speaker identity
→ broadcasts message
```

Client may not provide arbitrary Character authority.

Preferred canonical Character source:
- Show Run roster-selected Character for current participation.

Use site-wide active Character only as validated fallback/default if current architecture requires it.

Do not revive session persona as authority.

If no valid Character:
- disable IC send or reject clearly with "choose a Character first."

Changing selected Character changes future IC messages immediately.

Old messages preserve the Character identity used when sent.

Display Character name and Face/avatar where current chat supports it.

---

# 11. Cartograph proof flow

Kernel 87 must prove:

```text
Director enables Turn/Leader drawing
→ Cohort rolls and pins spatial dice
→ current player opens drawing tools
→ measures between reference points
→ opens one grid cell in Detail View
→ free-draws a feature
→ adds a filled shape
→ adds a settlement stamp
→ adds a label
→ closes Detail View
→ detail returns correctly to map
→ player hands off turn
→ second player adds to map
→ Director corrects one object
→ reconnect
→ drawing persists
→ export current map PNG
→ IC chat message appears from selected Character
```

This is the minimum meaningful drawing-as-play proof.

---

# 12. Export

Add **Export Current Map as PNG**.

Preferred export content:
- base map;
- persistent drawing objects;
- labels;
- stamps.

Exclude:
- ordinary UI chrome;
- transient measurement.

Pinned dice may be excluded by default or included via a simple existing option if easy.

Do not build a broad export-settings product.

Native drawing state remains canonical/editable.

---

# 13. Story So Far boundary

Do not write every stroke into Story So Far.

Drawing is not biography.

Only a later game-level milestone may emit one meaningful narrative event.

---

# 14. Repository audit before implementation

Inspect:
1. shared Pixi runtime;
2. map/grid implementation;
3. coordinate transforms;
4. stage object/state model;
5. Scene composition persistence;
6. Show/Cohort authority;
7. Group Leader/Current Turn APIs;
8. targeted WebSocket path from Kernel 86;
9. stage effect z-order;
10. token default size;
11. export/screenshot utilities;
12. chat/OOC implementation;
13. Character participation authority;
14. active Character signals;
15. existing Pixi/vector/graphics support;
16. compatible open-source drawing/vector libraries.

Do not ask Grant repository-answerable questions.

---

# 15. Open-source/library rule

Before hand-building complex vector editing behavior, audit mature open-source browser drawing/vector libraries.

Evaluate:
- license;
- compatibility with current frontend;
- serializable geometry;
- map-relative transforms;
- selection/resize/rotate quality;
- path smoothing;
- export;
- ability to keep server authoritative;
- whether it would force a conflicting second scene graph.

Do not import a huge dependency merely to get a line tool.

Use a library where it materially improves drawing quality or editing ergonomics without compromising Victory architecture.

Record license and rationale.

---

# 16. Security and abuse bounds

Enforce:
- max points per stroke;
- max drawing payload;
- max text length;
- sane object count/rate limits;
- valid colors/styles;
- valid stamp references;
- server-side authority;
- no arbitrary user/Character/Cohort impersonation.

Do not let live stroke streaming become an unbounded WS flood.

---

# 17. Required backend tests

### Authority
- Director can draw.
- unauthorized user cannot draw in Director Only.
- Current Turn/Leader can draw in Turn/Leader mode.
- Player cannot edit another Player's object by default.
- Director can correct/delete.
- Freeform allows eligible participants.

### Scope
- Cohort drawing reaches only proper Cohort.
- Show drawing reaches proper Show.
- reconnect does not leak Cohort drawing.
- restricted export does not leak drawings.

### Persistence
- create/edit/delete persist.
- z-order persists.
- lock persists.
- reconnect restores geometry.

### Geometry
- freehand payload bounded.
- shapes validated.
- coordinates remain map-relative.
- Detail View does not alter canonical geometry.

### Measurement
- scale persists.
- straight distance correct.
- path distance correct.
- diagonal policy tested.
- hex behavior tested where applicable.
- gridless scale tested if implemented.

### IC chat
- Character server-resolved.
- forged Character rejected/ignored.
- accountable user stored.
- Character switch affects future messages.
- no Character refuses cleanly.
- old messages preserve speaker identity.

---

# 18. Required browser proof

Prove at minimum:

1. Director enables Turn/Leader drawing.
2. authorized player sees tools.
3. unauthorized participant cannot draw.
4. freehand works.
5. line works.
6. polyline/path works.
7. rectangle works.
8. ellipse works.
9. polygon works.
10. text works.
11. stamp works.
12. stroke color works.
13. fill works.
14. no-fill works.
15. width works.
16. opacity works.
17. selection works.
18. move works.
19. resize works.
20. rotate works.
21. duplicate works.
22. delete works.
23. lock works.
24. z-order controls work.
25. undo does not reverse another Player's later mark.
26. Cohort drawing does not appear to another Cohort.
27. Show drawing does.
28. live stroke works if implemented.
29. completed-stroke sync works if live streaming deferred.
30. one-cell Detail View opens.
31. drawing there remains editable after return.
32. rectangular Detail View works.
33. returned objects align correctly.
34. pan/zoom preserves placement.
35. grid scale can be set.
36. straight measurement works.
37. path measurement works.
38. diagonal rule behaves as configured.
39. pinned dice remain above drawings.
40. drawing around pinned dice works.
41. reconnect restores drawing.
42. second player can add after handoff.
43. Director can correct another Player's mark.
44. PNG export works without UI chrome.
45. IC tab exists.
46. IC message uses selected Character.
47. another user sees Character as speaker.
48. forged Character impersonation fails.
49. changing Character changes future IC speaker.
50. no Character prevents IC send cleanly.

Capture screenshots of:
- collaborative map;
- Detail View;
- returned detail;
- measurement;
- pinned dice + drawing;
- PNG export;
- IC chat.

---

# 19. Visual quality bar

This is a creative tool. Functional debug-looking drawing controls are not enough.

Required:
- clean tool palette;
- clear active-tool state;
- usable color/fill controls;
- unobtrusive selection handles;
- intentional-looking Detail View;
- smooth/anti-aliased drawing;
- clean text/stamps;
- map remains visually dominant.

Do not overdesign, but do not claim PASS if the surface looks disposable.

---

# 20. Required artifacts

At minimum:

```text
Construction/Kernels/Kernel 87 — Shared Cartographic Rendering & In-Character Play.md
Construction/Domains/Stage/drawing-object-contract.md
Construction/Domains/Stage/measured-tabletop-contract.md
Construction/Domains/Stage/detail-view-contract.md
Construction/Domains/Chat/in-character-chat-contract.md
Construction/OperatorLogs/kernel-87-reportback.md
```

Update current-state/canonical roadmap where required.

Do not create another roadmap.

---

# 21. Pass criteria

Kernel 87 passes when:
- persistent canonical drawing objects exist;
- Director/Turn/Leader/Freeform authority works as specified;
- creator ownership rules are enforced;
- Cohort/Show scopes do not leak;
- Select/Edit/Move/Resize/Rotate/Delete/Duplicate work;
- Free Draw, Line, Path, Rectangle, Ellipse, Polygon, Text, Stamp work;
- colors/fills/width/opacity work;
- z-order and lock work;
- per-user undo is safe;
- Detail View works for one cell and rectangle;
- Draw Big, Return Small preserves editable geometry;
- physical scale and units work;
- straight/path measurement works;
- diagonal policy is explicit/tested;
- drawings persist;
- pinned dice coexist above drawings;
- PNG export works;
- IC chat resolves Character server-side;
- Character impersonation is impossible;
- UI is usable enough for real drawing-as-play;
- Cartograph proof flow succeeds end-to-end.

---

# 22. Partial criteria

Mark PARTIAL if:
- drawing is local-only;
- Cohort scope leaks;
- Detail View rasterizes or changes coordinates;
- measurement is pixel-only;
- Player undo can revert other users' later work;
- Turn/Leader authority does not work;
- export is absent;
- IC chat trusts client Character identity;
- UI is technically functional but visibly unusable.

Live-stroke streaming being deferred alone does **not** require PARTIAL if completed-stroke sync is solid and all other requirements pass.

---

# 23. Fail criteria

Mark FAIL if:
- client is canonical authority for drawing;
- unauthorized users edit/delete others' work;
- Cohort drawings leak;
- coordinates drift under pan/zoom;
- Detail View creates a second document/state;
- Detail View destroys editability;
- measurement is based only on browser pixels;
- IC chat trusts arbitrary Character ID;
- pinned dice cannot coexist with drawing;
- PNG export destroys canonical working state;
- kernel expands into Photoshop/scripting/full asset management;
- Grant is locked out.

---

# 24. Non-goals

Kernel 87 does not build:
- Photoshop/Krita;
- pressure-sensitive painting;
- procedural terrain;
- AI map generation;
- Bezier node editing;
- texture brushes;
- gradients;
- full layer palette;
- animation;
- 3D drawing;
- pathfinding;
- movement-cost automation;
- initiative;
- generalized fog;
- asset marketplace;
- arbitrary scripting;
- card/deck engine;
- CRPG automation;
- broad Audience interaction.

---

# 25. Immediate operator outcome

Grant should be able to answer:

1. Can I draw directly on the shared map?
2. Can another authorized player see it?
3. Do Free Draw, lines, paths, shapes, text, and stamps work?
4. Can I change color, fill, width, and opacity?
5. Can I select/move/resize/rotate/delete?
6. Can Players avoid editing each other's marks?
7. Can a Director correct anything?
8. Can I lock/reorder finished work?
9. Can I undo my own work safely?
10. Can drawings be Cohort-scoped or Show-wide?
11. Can I open one grid cell large?
12. Can I draw detailed content there?
13. Can I return it to the correct place?
14. Is it still editable?
15. Can I open a rectangular region?
16. Can I set physical scale and units?
17. Can I measure straight distances and paths?
18. Is diagonal behavior explicit?
19. Can pinned dice remain on top while we draw?
20. Can drawing control pass to the next player?
21. Does it survive reconnect?
22. Can I export PNG?
23. Is it pleasant enough to actually play with?
24. Is there an In Character chat tab?
25. Does it speak as my canonical Character without impersonation?
26. Can two players and a Director actually play a Cartograph-style mapping procedure?

Kernel 87 succeeds when Victory makes **drawing on the shared stage itself a playable tabletop mechanic**, not merely a markup utility.
