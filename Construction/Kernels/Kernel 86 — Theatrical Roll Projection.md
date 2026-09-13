# Kernel 86 — Theatrical Roll Projection

**Status:** PASS / DEPLOYED LIVE
**Type:** Shared stage presentation / dice projection
**Primary tracks:** V2 — Commands/Actions, V4 — Theatrical Control, V5 — Elements/Media/Stage, V7 — Rules Integration Boundary
**Secondary tracks:** S — Socio, A — Anthology, O — Operational Integrity
**Sequence position:** After Kernel 85
**Primary consumers:** Socio now; Cartograph-style drawing next
**Planning authority:** Canonical Roadmap as reconciled through Kernel 84/85
**Implementation authority:** repository behavior, current dice/action/runtime code, reportbacks, operator decisions

---

## 0. Kernel contract

Kernel 86 does **not** build a dice system.

The repository audit already established that Victory has:

- server-side dice resolution;
- crypto-backed randomness;
- canonical dice expression parsing;
- structured dice groups and explosion chains;
- skill-click integration;
- `/roll` and `/r`;
- Action persistence as `roll/dice`;
- live WebSocket delivery;
- a functional dice tray;
- rich structured roll payloads.

Preserve all of that.

The missing capability is:

> **Take an already-canonical roll result and project it theatrically onto the visible active stage/map for the correct audience.**

Core vertical:

```text
existing roll request
→ existing server-side dice engine resolves canonical result
→ existing roll/dice Action persists
→ server resolves intended audience
→ authorized sockets receive transient stage effect
→ visible active map renders dice on top presentation layer
→ dice animate toward already-determined result
→ landed dice ≈ default-token size
→ result holds ~3–4 seconds by default
→ multiple rolls queue and play in order
→ roll may instead remain static/pinned
→ canonical roll history remains the existing Action
```

Presentation never becomes authority.

---

# 1. Locked product decisions

## 1.1 Visible active map

Projected dice must land in visible space on the user's **currently visible active map/stage**.

Requirements:

- compatible with theatrical-fit view where practical;
- compatible with full-screen fit;
- guaranteed visible inside current viewport;
- top presentation layer above map/tokens/ordinary Scene elements;
- never placed off-screen merely because the Scene world is larger than the viewport.

If theatrical-fit geometry is reliable, use it. If not, full-screen visible-space placement is sufficient for Kernel 86 so long as projection is guaranteed visible.

## 1.2 Landed size

Each landed die should be approximately the size of a **default token**.

Do not reduce projected rolls to tiny HUD notifications.

Do not make each die full-screen.

## 1.3 Queue

If another roll occurs while one is projecting:

> **Queue it and play it afterward.**

Do not replace or silently drop a roll.

Do not stack an unreadable pile of simultaneous projections by default.

## 1.4 Duration

Transient default:

> **approximately 3–4 seconds after landing**

Then fade/remove.

## 1.5 Static / pinned mode

A projected roll may be static/pinned.

Explicit near-term use case:

> Cartograph-style play, where the dice need to stay visible while a player draws or acts on their result.

Required:

- transient projection;
- pin/make static;
- remain visible;
- authorized dismiss/unpin;
- later transient rolls still project while a static roll remains.

Pinning is presentation state linked to the canonical roll Action. It must not create a second canonical roll.

## 1.6 Display content

Projected roll should show, when present:

- actor / Character identity;
- skill or roll label;
- individual dice;
- modifier;
- total;
- existing canonical explosion-chain information.

## 1.7 Explosion semantics — deliberately unchanged here

Victory's current dice engine automatically resolves explosions.

**Kernel 86 does not change that behavior.**

The operator explicitly accepts this for Kernel 86 but requires a near-term revisit because it is unsuitable as an unquestioned default for Cartograph-style play.

Required follow-up ledger item:

> Before Cartograph depends on dice behavior, revisit explosion semantics so explosions occur only when a game operation/macro explicitly requests them, or preserve compatibility intentionally with a clearly chosen default.

Do not mark this resolved.

Do not compute new explosions in the renderer.

## 1.8 Animation

Preferred:

> Dice animate/tumble/tick toward the already-resolved server result.

No client-side canonical randomness.

No physics engine required.

Intermediate faces may be decorative. Final face/value must exactly match the server result.

## 1.9 Audience modes

Support:

- **Show**
- **Cohort**
- **Director**
- **Private**

Server resolves recipients.

## 1.10 Default visibility

Default ordinary roll:

> **Cohort**

If roller is Ungrouped, safely fall back to:

> **Show**

unless repository authority requires a narrower safe fallback.

## 1.11 Director default

Director rolls use the same default as ordinary rolls unless explicitly changed.

Do not make Director rolls invisible by surprise.

---

# 2. Preserve the existing dice stack

Do not rebuild or replace:

- `backend/internal/dice/dice.go`;
- `ParseExpression`;
- `Roll`;
- `RollExpression`;
- `SingleGroup`;
- crypto-backed RNG;
- existing compound-expression behavior;
- `actions.StoreDiceRoll`;
- `roll/dice` Action history;
- skill-click rolls;
- `/roll`;
- `/r`;
- merchant/haggle/stance callers;
- `/char advance` relationship to prior skill rolls;
- existing dice tray/history UI.

The renderer consumes existing canonical result data.

---

# 3. Roll audience resolution

The current visibility field exists but is effectively public-only.

Activate it rather than inventing unrelated visibility metadata.

Normalize supported modes:

```text
show
cohort
director
private
```

Preserve backward compatibility with any current `"public"` value as needed.

## 3.1 Cohort

For `cohort`:

- resolve current Show;
- resolve roller's cohort;
- resolve roster server-side;
- build recipient set server-side;
- do not accept arbitrary client-provided recipient IDs.

If Ungrouped, use safe Show fallback.

## 3.2 Show

Resolve current Show membership/admission server-side.

## 3.3 Director

Resolve Director+ authority server-side.

## 3.4 Private

Roller only, including their appropriate connected sockets/tabs.

---

# 4. Targeted live delivery

Audit found current `hub.Broadcast` is global fan-out.

Kernel 86 must not implement privacy by globally sending the full roll and hiding it in unauthorized browsers.

Add a bounded reusable server-side recipient-targeted delivery path, conceptually:

```text
Hub.SendToRecipients(recipients, payload)
```

Exact API follows current architecture.

Requirements:

- recipient set server-resolved;
- unauthorized sockets never receive full private/cohort/director payload;
- authorized user's multiple sockets can receive it;
- disconnected users simply miss transient presentation;
- reconnect can recover permitted canonical history.

### Guardrail

Do **not** refactor all existing global broadcast call sites.

Use targeted delivery for dice/stage effects and record broader broadcast debt if discovered.

---

# 5. Canonical roll vs stage effect

`roll/dice` Action remains truth.

The stage projection is derived presentation.

Do **not** reuse the current singular persistent `Overlay`; audit established it is element-bound, persistent, singular, and not queue/expiry-oriented.

---

# 6. Stage Effect contract

Introduce a lightweight presentation contract, conceptually:

```text
StageEffect
- effect_id
- type = "dice_roll"
- source_action_id
- show_id
- cohort_id optional
- audience
- actor
- label
- payload
- created_at
- duration_ms
- static/pinned
```

Exact persistence follows repository architecture.

## 6.1 Persistence boundary

Preferred:

- transient queue state may remain live/in-memory;
- pinned/static state may persist at Show/cohort presentation scope if needed for reconnect;
- `roll/dice` Action remains canonical history.

Do not create a second durable dice-history universe.

---

# 7. Pixi stage renderer

Add a dedicated dice projection renderer in shared stage runtime.

Prefer a bounded module such as:

```text
frontend/lib/stage-runtime/dice-projection.js
```

rather than bloating `runtime.js` if modular architecture supports it.

## 7.1 Layering

Render above:

- active map;
- tokens;
- Scene elements;
- ordinary placement layers.

Respect critical UI/modal layering where necessary.

## 7.2 Visible placement

Calculate against current visible viewport/screen coordinates, not arbitrary world coordinates.

Projection must remain fully visible under pan/zoom.

Preferred region: upper/central visible map area without covering the entire scene.

## 7.3 Fit modes

Support full-screen fit.

Support theatrical fit where current runtime exposes reliable geometry.

---

# 8. Visual behavior

## 8.1 Entry

Use a short theatrical entry:

- tumble;
- flip;
- rapid face cycling;
- bounce/ease;
- scale/rotation;
- short motion arc.

No physics engine required.

## 8.2 Final result

Renderer already knows final canonical result.

No browser RNG determines it.

## 8.3 Landed state

Once settled:

- each die ≈ default-token size;
- final values readable;
- modifier readable;
- total prominent;
- explosion chain visually associated if canonical data contains one.

## 8.4 Multi-die pools

Keep pool readable and within viewport.

Wrap or scale sensibly when needed.

Do not allow wide pools to extend off-screen.

---

# 9. Queue behavior

Client maintains an ordered projection queue for authorized effects.

Transient:

```text
effect A
→ animate
→ land
→ hold 3–4s
→ fade
→ effect B
```

Pinned/static effect must not permanently block the queue.

Preferred:

- pinned result settles into stable pinned presentation position;
- later transient rolls continue.

Protect against unbounded queue growth/spam.

---

# 10. Static / pinned projection

Required operations:

- project transiently;
- pin/make static;
- dismiss/unpin.

Authority:

- roller may pin their own roll if current product authority permits;
- Director+ may pin/dismiss appropriate Show/cohort projections;
- unrelated users cannot dismiss another cohort's private/static projection.

If a pinned roll must support multi-minute authoring, preserve it across ordinary reconnect if bounded.

No permanent archival requirement beyond existing Action history.

---

# 11. Roll controls

Do not redesign the whole dice tray.

Add only required controls:

- audience/visibility;
- transient/static behavior;
- pin after landing;
- dismiss static roll.

Defaults:

```text
visibility = cohort
projection = transient
duration ≈ 3–4s
```

Ungrouped fallback:

```text
visibility = show
```

Skill-click and `/roll` must continue without forcing a configuration dialog.

---

# 12. History

Existing Action history remains roll history.

Do not build another roll-history page.

If Game Events already displays `roll/dice`, preserve it.

No replay-from-history requirement in 86.

---

# 13. Target Value / success-failure deferred

Audit found target values and success/failure are not part of canonical roll record.

Do not add them in Kernel 86.

A roll records what was rolled. Resolution may later depend on Socio TV, oracle tables, Cartograph procedure, or Narrator judgment.

Do not prematurely universalize success semantics.

---

# 14. Mandatory explosion follow-up

Update canonical planning/current-state/reportback with the unresolved issue:

> Victory currently auto-resolves exploding dice. Kernel 86 preserves this for compatibility. Before Cartograph depends on dice, define explicit explosion semantics so explosion behavior is requested by a game operation/macro or otherwise deliberately configured.

This must remain visible near-term work.

---

# 15. Required audit before implementation

Confirmed (see `Construction/OperatorLogs/kernel-86-reportback.md` §2 for the recorded findings):

1. current `roll/dice` request/response shape;
2. current visibility field and normalization;
3. Action recipient fields;
4. hub/socket indexing by user/session;
5. any existing targeted-send primitive elsewhere;
6. current Show membership resolver;
7. Kernel 85 cohort roster APIs;
8. Director+ resolution;
9. dice tray architecture;
10. Pixi layer hierarchy;
11. viewport coordinate conversion;
12. default token rendered size;
13. top-layer z-order;
14. stage reconnect behavior;
15. whether generic snapshot Actions currently expose rolls to users outside intended audience.

The audit confirmed item 15 was a real gap (`world.LoadVenueSnapshot` returned every session/show-scoped `roll/dice` Action unfiltered regardless of its `visibility` field) — repaired as part of this kernel; see §6/§12 of the reportback.

---

# 16. Security/privacy

For Cohort/Director/Private:

- unauthorized sockets must not receive full live payload;
- unauthorized users must not receive the roll later through snapshot/history;
- client may select a supported visibility mode but not arbitrary recipients;
- preserve existing dice/action rate/bounds;
- queue must not enable unbounded client memory growth.

---

# 17. Backend tests

## Preserve existing behavior

- current roll still resolves server-side;
- skill-click still works;
- `/roll` still works;
- Action persistence unchanged;
- existing dice suite remains green.

## Visibility

- Show roll reaches only Show-authorized users;
- Cohort roll reaches only proper cohort recipients;
- Director roll reaches only Director+;
- Private reaches roller only;
- Ungrouped default safely falls back to Show;
- unauthorized user receives neither live payload nor restricted history.

## Targeted delivery

- authorized user with multiple sockets receives payload;
- unauthorized sockets do not;
- disconnect bookkeeping remains correct;
- unrelated global broadcast behavior unchanged.

## Stage effect

- references canonical Action;
- duration supported;
- static flag supported;
- pin/unpin authority enforced;
- transient expiry works;
- static persistence follows chosen bounded model;
- no duplicate roll Action.

## Queue

- ordering preserved;
- distinct effect IDs;
- deterministic enqueue sequence.

---

# 18. Browser proof

Playwright/live multi-user proof mandatory.

See `scripts/smoke/kernel86-dice-projection-browser.js` and the reportback for the executed proof (48/48 assertions, live production, two consecutive clean runs, zero residue).

---

# 19. Required artifacts

```text
Construction/Kernels/Kernel 86 — Theatrical Roll Projection.md
Construction/Domains/Dice/dice-projection-contract.md
Construction/Domains/Stage/transient-stage-effects.md
Construction/Domains/Network/targeted-live-delivery.md
Construction/OperatorLogs/kernel-86-reportback.md
```

---

# 20. Evidence

See `Construction/OperatorLogs/kernel-86-reportback.md` for full evidence (`go test`, deploy log, browser proof output).

---

# 21. Pass criteria

See reportback §2 for the criterion-by-criterion ledger.

---

# 22–23. Partial / Fail rules

Not triggered — see reportback for the PASS determination and the one item deliberately left as follow-up ledger work (§14 explosion semantics, unchanged by design).

---

# 24. Non-goals

Kernel 86 does not:

- rewrite dice math;
- replace crypto randomness;
- redesign dice syntax;
- redesign the dice tray;
- add 3D/physics dice;
- add universal success/TV resolution;
- build Cartograph drawing;
- change explosion semantics;
- refactor every global WebSocket broadcast;
- build a general animation engine;
- create permanent dice Scene objects;
- create a second roll-history database.

None of these were touched.

---

# 25. Kernel-size guardrail

The genuinely new primitives built:

1. server-resolved roll audiences (`backend/internal/rollaudience`);
2. targeted live delivery (`Hub.SendToUsers`, `Hub.BroadcastSession` reuse);
3. transient/static Stage Effect contract (`backend/internal/stageeffects`);
4. Pixi theatrical dice projection (`frontend/lib/stage-runtime/dice-projection.js`).

Everything upstream of display was reused unchanged.

---

# 26. Immediate operator outcome

See reportback §6 for the answered checklist.
