# Kernel 86A — Spatial Dice Landing & Explosion-Safe Projection

**Status:** READY FOR IMPLEMENTATION
**Type:** Continuation / repair kernel
**Parent kernel:** Kernel 86 — Theatrical Roll Projection
**Primary tracks:** V4 — Theatrical Control, V5 — Stage Runtime, V7 — Rules Integration Boundary
**Secondary tracks:** A — Cartograph readiness, O — Operational Integrity
**Sequence position:** Immediately after Kernel 86, before Kernel 87
**Planning authority:** Canonical Roadmap/current-state after Kernel 86
**Implementation authority:** current stage renderer, current dice projection code, current dice engine, Kernel 86 reportback, operator decisions

---

## 0. Kernel contract

Kernel 86A repairs one specific product mismatch in Kernel 86:

> Dice are currently rendered as a grouped announcement overlay, but they do not individually roll onto and land on the active map in spatially meaningful positions.

The existing announcement group is **preserved**.

The required corrected sequence is:

```text
server resolves canonical roll
↓ individual dice animate onto the visible active map
↓ each die lands at its own map-relative coordinate
↓ all dice finish landing
↓ only then does the announcement group appear
↓ transient dice remain for normal duration
↓ static/pinned dice remain in place until cleared
```

The announcement is informational.

The dice themselves are the spatial projection.

This kernel is required before Cartograph-style drawing depends on dice positions.

---

# 1. Locked product decisions

## 1.1 Keep the existing announcement group

Do not remove the current result announcement UI.

Preserve its current useful information, including:

- actor/Character name where available;
- roll label;
- die results;
- modifier;
- total.

The announcement remains a valid theatrical summary.

It does **not** satisfy the spatial dice requirement by itself.

## 1.2 Announcement timing

The announcement must not appear before the dice have finished landing.

Required order:

```text
ROLL START
↓ dice enter map
↓ dice animate/tumble
↓ each die settles
↓ all dice settled
↓ announcement appears
```

No result banner should spoil the landing sequence.

If the announcement uses the same canonical data that already exists, delay only its presentation, not the server result.

## 1.3 Dice land on the active map

Each die must have its own final coordinate on the currently visible active map/stage.

Requirements:

- each die lands independently;
- landing coordinates must be inside visible map space;
- dice must not remain grouped in the announcement region;
- dice should read as objects resting on the map;
- final positions must be spatially meaningful for drawing/placement workflows.

## 1.4 Map-relative, not HUD-relative

Once landed, dice must behave as map/stage objects.

If the user pans or zooms after landing:

- the dice should move/scale with the map appropriately;
- they must not remain glued to the browser viewport like HUD widgets.

Screen-space math may be used during entry animation.

Final landed coordinates must be converted into the appropriate stage/world coordinate space.

## 1.5 Visible-space constraint

Dice may not land outside the currently visible active map region.

The renderer must determine a valid visible map polygon/rectangle under the current fit/pan/zoom state and choose positions inside it.

Avoid:

- off-map landing;
- clipped dice;
- landing under non-map chrome;
- landing outside the visible viewport.

## 1.6 Dice spacing

Multiple dice must not all land on exactly the same coordinate.

Use reasonable spacing/collision avoidance so individual dice remain distinguishable.

Do not require perfect physics packing.

A lightweight spacing pass is sufficient.

## 1.7 Landed size

Each die, once settled, should be approximately the size of a default token.

Preserve the Kernel 86 size decision.

## 1.8 Transient duration

For normal rolls:

- dice land;
- announcement appears after landing;
- dice remain visible for approximately 3–4 seconds;
- then fade/remove.

Timing may begin from landing completion rather than initial roll start.

## 1.9 Static / pinned mode

For static/pinned rolls:

- dice land spatially;
- announcement appears after all dice land;
- announcement may fade normally;
- dice remain in their exact landed map-relative positions;
- authorized user can later clear/unpin them;
- subsequent transient rolls can still occur.

This is the explicit Cartograph-readiness behavior.

## 1.10 Announcement after pinning

The announcement does not need to remain pinned with the dice.

Preferred behavior:

```text
static dice stay
announcement fades normally
```

This keeps the map readable during drawing.

---

# 2. Explosion directive — required and explicit

Kernel 86A must not introduce automatic explosion behavior into spatial dice landing.

The operator's directive is:

> **Exploding dice should never be automatically re-rolled by the projection system. Explosions should only occur when a macro/game operation explicitly requests explosion behavior.**

However, Kernel 86A is a continuation of 86 and should not silently rewrite the current canonical dice engine if that would exceed this repair scope.

Therefore apply these rules:

## 2.1 Projection layer

The renderer:

- never decides that a die should explode;
- never creates a reroll;
- never adds child dice;
- never invokes dice-engine explosion behavior;
- only renders the canonical result it is given.

## 2.2 Existing canonical explosion chains

If the current `roll/dice` Action already contains explosion-chain results because the existing expression explicitly requested exploding dice, render them faithfully.

Do not suppress valid explicitly-requested explosion results.

## 2.3 Default roll behavior audit

Before implementation, verify whether the current normal/default roll syntax automatically requests explosions.

If ordinary dice expressions are auto-exploding without an explicit operator/macro request, record that as an immediate product defect.

If the fix is narrowly limited to default expression construction or one compatibility flag, repair it in 86A.

If changing canonical explosion semantics would require broader engine/compatibility work, do **not** bury that rewrite inside this continuation. Instead:

- preserve current engine behavior temporarily;
- ensure the renderer adds no explosion behavior of its own;
- file a required 86B / pre-87 continuation;
- Cartograph may not proceed until default explosion semantics are explicit.

The acceptance outcome before Kernel 87 is:

> A normal Cartograph-style die roll cannot unexpectedly explode unless the roll/macro explicitly requests an exploding die.

---

# 3. Current failure mode to repair

The observed behavior is approximately:

```text
[ 3 ] [ 10 ] [ 6 ]
Grant A. Murray rolled 3d12
19
```

rendered as a grouped fixed announcement over the map.

This is useful and remains.

What is missing:

```text
die A rolls across map → lands here
die B rolls across map → lands elsewhere
die C rolls across map → lands elsewhere
THEN announcement appears
```

Kernel 86A must add the spatial phase without regressing the existing announcement.

---

# 4. Spatial projection model

## 4.1 Per-die presentation state

For each atomic die result, the renderer needs temporary presentation state such as:

```text
diePresentation
- source action id
- group index
- die index
- final value
- start screen/stage position
- control/path points optional
- final stage/world x
- final stage/world y
- rotation
- animation duration
- settled boolean
- static/pinned boolean
```

This is presentation state, not canonical roll truth.

## 4.2 Server vs client responsibility

Server remains authoritative for:

- dice expression;
- canonical roll values;
- action identity;
- visibility/audience;
- static/transient request where applicable.

Client renderer may choose:

- entry trajectory;
- visual rotation;
- final map-relative landing coordinates;
- decorative intermediate faces.

Landing coordinates are presentation state unless a future Cartograph rule explicitly makes them game state.

For static dice used as drawing references, the final coordinates must persist at least for the life of the pinned projection.

## 4.3 Reconnect behavior

If pinned/static dice already persist across reconnect under Kernel 86:

- preserve exact landing coordinates.

If Kernel 86 stores only "this roll is pinned" but not where its dice landed, extend pinned presentation state to include final coordinates.

A reconnect must not randomly relocate pinned dice.

---

# 5. Visible map bounds

Before choosing landing positions, determine the visible active-map region.

Use current stage-runtime transforms.

Required:

- map-local visible bounds under current zoom/pan;
- safe inset at least half a die size from the edge;
- account for theatrical/full-screen fit;
- avoid UI-only top/bottom regions where map is not actually visible.

Do not hardcode one resolution.

---

# 6. Landing algorithm

No physics engine required.

A simple deterministic or pseudo-random presentation algorithm is acceptable if it satisfies:

- all dice land inside visible map bounds;
- dice are separated enough to read;
- every die reaches a unique/usable final coordinate;
- trajectories look like rolling/tumbling rather than teleporting;
- final canonical face/value is correct.

## 6.1 Suggested strategy

For each die:

1. choose an entry point near a visible-map edge;
2. choose a final point within safe visible-map bounds;
3. reject/retry final points too close to previously allocated dice;
4. animate with easing + rotation + face cycling;
5. convert/store final coordinate in stage/world space;
6. mark die settled.

Do not over-engineer rigid-body simulation.

## 6.2 All-settled barrier

Announcement presentation waits on:

```text
all dice in this roll effect have settled
```

Only after that barrier resolves may the announcement enter.

This must be explicit in code/tests.

---

# 7. Announcement sequencing

## 7.1 Current renderer preservation

Preserve current result renderer.

Do not duplicate it.

Add sequencing so it listens for the spatial roll completion event/promise/state.

## 7.2 Timing

Preferred:

```text
t=0       roll effect begins
t=0-N     dice animate
t=N       final die settles
t=N+small announcement enters
t=N+3-4s  transient dice/announcement fade according to their own timers
```

Announcement timing may overlap with the landed hold time.

## 7.3 Queue interaction

For queued rolls:

- roll A dice fully land;
- roll A announcement appears;
- roll A presentation completes according to transient/static rules;
- roll B then begins according to existing queue semantics.

If static roll A remains pinned, roll B may still begin after A's normal active presentation window.

Do not require clearing pinned dice before the next roll.

---

# 8. Static dice as Cartograph reference objects

Pinned/static dice must remain in exact spatial positions on the active map.

Required:

- map-relative x/y retained;
- pan/zoom preserves relative placement;
- drawing can occur around them;
- later transient dice can roll over/around the map;
- authorized clear removes only intended static roll/dice;
- no accidental conversion into permanent Scene elements.

These are presentation/reference objects, not Scene-authoring elements.

A later Cartograph kernel may choose to snapshot/reference their positions as part of a game procedure.

Kernel 86A only makes the positions stable and usable.

---

# 9. Authority and privacy

Preserve Kernel 86 audience rules:

- Show;
- Cohort;
- Director;
- Private.

Spatial landing and announcement must go to exactly the same authorized audience.

Do not create a situation where:

- private dice are spatially rendered only privately but announcement broadcasts globally;
- or vice versa.

The effect is one audience-scoped presentation sequence.

---

# 10. Required repository audit before implementation

Before changing code, inspect:

1. current Kernel 86 dice projection renderer;
2. current announcement component;
3. current queue implementation;
4. current static/pin implementation;
5. current stage/runtime layer hierarchy;
6. map-to-screen and screen-to-map transforms;
7. active viewport bounds helpers;
8. default token dimensions;
9. current pinned-effect persistence;
10. current explosion-expression defaults;
11. which callers explicitly request exploding dice;
12. whether ordinary roll UI silently adds explosion syntax.

Do not rebuild anything already working.

---

# 11. Tests — spatial behavior

Add automated/frontend tests where practical for:

- one die receives one final coordinate;
- three dice receive distinguishable coordinates;
- all coordinates lie inside safe visible bounds;
- final coordinates convert correctly to map/stage space;
- pan/zoom after landing preserves map-relative position;
- pinned reconnect restores exact coordinates;
- transient cleanup removes spatial dice;
- clearing pinned roll removes only its dice.

---

# 12. Tests — sequencing

Required:

- announcement hidden while any die is still animating;
- first die landing does not trigger announcement if other dice still move;
- announcement appears only after final die settles;
- canonical totals unchanged;
- queue waits appropriately;
- pinned dice do not block later queued rolls.

---

# 13. Tests — explosion safety

Required:

- renderer never generates explosion rerolls;
- renderer never mutates explosion chains;
- explicitly exploding canonical roll renders its supplied chain;
- ordinary/default Cartograph-compatible roll does not unexpectedly explode before Kernel 87.

If that last test cannot pass without broader canonical-engine work:

> Kernel 86A must be PARTIAL and issue a mandatory 86B before Kernel 87.

Do not waive this requirement merely because the old dice engine historically auto-exploded.

---

# 14. Required browser proof

Playwright/manual browser evidence should prove:

1. roll `3d12` or equivalent from existing UI;
2. announcement is initially absent;
3. three dice visibly enter/animate on active map;
4. each die lands in a different visible map position;
5. final values match canonical server Action;
6. only after all three settle does announcement appear;
7. dice are about default-token size;
8. dice sit on top spatial layer;
9. pan stage after landing and dice move with map, not HUD;
10. zoom stage and dice remain attached to map coordinates;
11. transient dice disappear after normal hold;
12. pinned/static dice remain;
13. announcement may fade while pinned dice remain;
14. subsequent roll still projects with pinned dice present;
15. unpin/clear removes pinned dice;
16. cohort/private audience still scopes both spatial dice and announcement correctly;
17. ordinary/default non-exploding roll does not unexpectedly explode;
18. an explicitly requested exploding expression still renders the canonical explosion chain correctly.

Screenshots/video frames should include:

- dice mid-roll before announcement;
- dice landed with announcement visible;
- static dice remaining after announcement is gone.

---

# 15. Required artifacts

At minimum:

```text
Construction/Kernels/Kernel 86A — Spatial Dice Landing & Explosion-Safe Projection.md
Construction/Domains/Dice/spatial-dice-projection-contract.md
Construction/OperatorLogs/kernel-86A-reportback.md
```

Update:

- current-state.md;
- canonical roadmap/open-work ledger;
- dice projection contract;
- explosion-semantics follow-up.

Do not create another roadmap.

---

# 16. Pass criteria

Kernel 86A passes when:

- existing announcement group is preserved;
- announcement never appears before all dice finish landing;
- each die visibly animates onto the active map;
- each die has its own final map-relative coordinate;
- dice land inside visible map space;
- dice remain distinguishable;
- landed size remains approximately default-token size;
- pan/zoom treats landed dice as map objects rather than HUD;
- transient dice disappear after normal duration;
- static/pinned dice remain at exact coordinates;
- pinned dice do not block future rolls;
- announcement can fade independently of pinned dice;
- final displayed values equal canonical server result;
- projection layer performs no explosion rerolls;
- ordinary Cartograph-compatible rolls cannot unexpectedly auto-explode;
- explicitly requested explosion chains remain renderable;
- audience/privacy behavior from Kernel 86 remains intact;
- existing dice tray/skill-click/`/roll` behavior remains intact.

---

# 17. Partial criteria

Mark PARTIAL if:

- announcement sequencing is fixed but dice remain HUD-relative;
- dice land spatially but pinned reconnect relocates them;
- dice positions are map-relative but can land off-screen;
- static dice block later rolls;
- ordinary rolls still auto-explode and require a separate canonical fix.

If explosion semantics remain unresolved, issue **Kernel 86B — Explicit Explosion Semantics** and block Cartograph Kernel 87 until it passes.

---

# 18. Fail criteria

Mark FAIL if:

- announcement reveals result before dice finish landing;
- browser changes canonical roll values;
- dice landing positions alter numerical results;
- spatial dice leak to unauthorized audiences;
- pinned dice are written as permanent Scene elements by default;
- explosion behavior is silently changed without compatibility consideration;
- ordinary Cartograph rolls can unexpectedly explode and the kernel still claims PASS;
- existing roll pathways regress;
- Grant is locked out.

---

# 19. Non-goals

Kernel 86A does not:

- remove the announcement renderer;
- build full rigid-body dice physics;
- build 3D dice;
- redesign dice syntax broadly;
- redesign Cartograph;
- build vector drawing;
- create permanent Scene dice;
- add universal success/failure semantics;
- rewrite all WebSocket broadcasting;
- change canonical randomness.

---

# 20. Immediate operator outcome

At completion, Grant should be able to answer:

1. Do the dice themselves actually roll onto the visible map?
2. Does each die land somewhere distinct?
3. Are they spatially attached to the map after landing?
4. Can I pan/zoom and have them stay where they landed relative to the map?
5. Does the announcement wait until the last die finishes landing?
6. Is the existing announcement still there afterward?
7. Are the landed dice about token size?
8. Do normal dice disappear after a few seconds?
9. Can I pin them for Cartograph?
10. Can the announcement disappear while pinned dice remain?
11. Can another roll occur while pinned dice are still present?
12. Can I clear the pinned dice later?
13. Do the dice values still exactly match the server result?
14. Does the projection layer avoid creating explosion rerolls?
15. Can a normal Cartograph-compatible roll avoid unexpected explosions?
16. Can an explicitly exploding macro/expression still show its canonical chain?
17. Do private/cohort rolls remain correctly private/cohort-scoped?
18. Is Kernel 87 now safe to build drawing around the dice that actually landed?

Kernel 86A succeeds when Victory's dice stop merely **announcing a roll over the map** and instead **physically perform on the map before announcing the result**.
