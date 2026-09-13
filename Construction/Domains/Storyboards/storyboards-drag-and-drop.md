# Storyboards Drag-and-Drop Contract (Kernel 81)

## Why Pointer Events, not HTML5 drag-and-drop

Pointer Events (`pointerdown`/`pointermove`/`pointerup`/`pointercancel`)
unify mouse, touch, and pen under one event model and give full control
over the drag-threshold/ghost/auto-scroll behavior the spec requires
(§7.2). This matches the existing precedent in
`frontend/venues/show-runs/show.html`'s composer surface
(`pointerdown`/`pointermove`/`pointerup` token dragging) rather than
introducing a second drag idiom into the codebase. No drag-and-drop
library was added (spec §7.2: "Do not introduce a large frontend
framework solely for drag-and-drop").

## Who gets a drag handler at all

`attachCardDragHandlers` is only ever called from `buildCardEl` when
`canDragCard(card)` is true:

```js
function canDragCard(card) {
  if (!affordances.canDragCards) return false;   // Audience/Cast: never
  if (!card.is_locked) return true;               // Crew+: unlocked cards
  return affordances.canEditStructure;             // locked cards: Director+/owner only
}
```

This exactly mirrors `cards.go`'s `canMutateCard` server-side rule. An
Audience/Cast card never receives a `pointerdown` listener at all — not
just visually non-draggable, structurally inert. The server re-checks
the same rule on every mutation regardless; this is a UX optimization
(no failed-drag flicker for a user who was never going to be allowed to
drag), not the security boundary.

## The drag lifecycle

1. **`pointerdown`** on a card records `{cardId, originRowId,
   originColumnId, startX, startY, pointerId, dragging: false}`. No
   visual change yet, and no pointer capture yet — this matters because a
   plain tap/click must still work.
2. **`pointermove`** — once movement exceeds a 6px threshold
   (`Math.hypot(dx, dy) < 6`), the drag "really" starts:
   `setPointerCapture`, a cloned `.drag-ghost` element is appended to
   `document.body` and tracks the pointer, the origin card gets
   `.is-dragging` (32% opacity). Every subsequent move updates the ghost
   position, checks `document.elementFromPoint` for the `.cell` under the
   pointer, and toggles `.is-drop-target` (empty) or `.is-drop-occupied`
   (has a visible card) on it. `autoScrollNearEdges` nudges
   `#board-scroll`'s `scrollLeft`/`scrollTop` when the pointer is within
   36px of the scroll container's edge.
3. **`pointerup`** — if the drag threshold was never exceeded, this was a
   click: `cardEl._suppressClick` is left `false` and the existing
   `click` handler opens the card editor normally (no move attempted). If
   dragging did happen, `_suppressClick` is set `true` (so the browser's
   trailing synthetic `click` event, if any, is swallowed) and
   `finishDrag` runs.
4. **`pointercancel`** (e.g. an interrupted gesture) tears down the ghost
   and drop-target styling without sending any request.

## Click vs. drag disambiguation

Standard idiom: track whether the 6px threshold was crossed
(`dragState.dragging`); if the pointer released without crossing it,
treat it as a click. `cardEl._suppressClick` additionally guards against
the browser firing a `click` event after a real `pointerup`-ending-a-drag
sequence on some platforms — the `click` listener checks and clears this
flag before deciding whether to open the editor. The flip button (`⟲`)
and move-handle button (`⠿`) both call `e.stopPropagation()` so clicking
them never triggers the card's own click-to-open handler.

## Server authority (spec §7.3) — never optimistic

`finishDrag` never mutates local grid state directly. It determines the
target cell, looks up the target's current visible card via
`grid-model.js`'s `cardsByCell`/`visibleCardForCell` (computed fresh from
the last-known `snapshot`, not from anything the drag itself touched),
decides `move`/`noop`/`occupied` via `dropResolutionKind`, and only then
issues the real HTTP mutation (`MoveCard`, or opens the occupied dialog
which itself issues `SwapCards`/two `MoveCard` calls). `await
loadSnapshot()` always follows — win, lose, or error — so the rendered
board is reconciled from the server's actual state, never left showing an
assumed outcome. A version conflict or lock/authority rejection surfaces
as an `alert()` and a fresh snapshot load; the card visually returns to
wherever the server says it actually is.

## Occupied-cell resolution

See `storyboards-ui-contract.md`'s "Occupied-cell resolution" section and
`storyboards-presentation-contract.md`. Summary: Swap uses the new atomic
`SwapCards` backend function; Move-existing is a guided two-step
`MoveCard`/`MoveCard` sequence via an empty-cell picker overlay; Cancel
sends nothing.

## What was proven in the real browser proof (not just unit-tested)

- drag onto an empty cell (`08-owner-after-drag-to-empty-cell.png`)
- drag onto an occupied cell → dialog → Swap (`09`/`10`)
- drag onto an occupied cell → dialog → Move existing → empty-cell picker
  (`22-owner-move-existing-picker-active.png`,
  `23-owner-after-move-existing.png`) — confirmed both cards survive in
  distinct cells afterward, no card lost
- the modal/select fallback path, unaffected by any of the above
  (`11-owner-fallback-move-ui.png`)
- zero console/page errors during any drag sequence
