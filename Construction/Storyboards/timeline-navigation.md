# Timeline Navigation Contract (Kernel 82)

Covers two things: boundary-column-aware column insertion, and middle-mouse panning.

## Boundary column insertion (spec 2.3/2.4)

No new backend endpoint. Insertion reuses Kernel 81's exact frontend-only append-then-reorder flow (`AddColumn` always appends at the end, then `ReorderColumns` places it) — the only change is that `ReorderColumns` (`columns.go`) now rejects, server-side, any ordering that would move a `beginning`-role column away from index 0 or an `ending`-role column away from the last index (`ErrBoundaryColumnDisplaced`). This is a no-op check for every Blank-mode board (no column there ever carries a boundary role), and it means the boundary guarantee holds even if a client sent a forged reorder request directly — the column-menu only *offering* "Insert left" on Beginning or "Insert right" on Ending was never itself the guarantee (this codebase's standing rule: hiding a control is never a security boundary).

`RemoveColumn` similarly rejects removing a `beginning`/`ending` column outright (`ErrBoundaryColumnProtected`), with no resolution flow — unlike an ordinary occupied-column removal, there is no "confirm and it happens anyway" path for a boundary column. Deleting one would be the literal case of "Beginning/Ending can be displaced outside the timeline" the spec's own FAIL criteria (§19) names.

The frontend's column ⋮ menu (`buildColumnMenu`, `board.html`) reads `col.column_role` and only renders the options that could ever succeed: Beginning gets "Insert right" + "Rename" (no "Insert left", no "Remove"); Ending gets "Insert left" + "Rename"; ordinary columns get all four. This matches spec §2.4/§13 exactly and was verified directly in the Kernel 82 browser proof (`beginning_menu`/`middle_menu`/`ending_menu` results).

## Middle-mouse panning (spec 6)

Implemented with Pointer Events on `#board-scroll` (`initMiddleMousePan`, `board.html`) — the same event model Kernel 81's card drag-and-drop already uses, not a second interaction library.

- `pointerdown` with `event.button === 1` (middle button) starts a pan: records the starting pointer position and the scroller's current `scrollLeft`/`scrollTop`, captures the pointer, and sets a `grabbing` cursor.
- `pointermove` while panning sets `scrollLeft`/`scrollTop` directly from the pointer delta — pure client-side scroll-position manipulation, no server request of any kind (spec's explicit "no Storyboard mutation is sent").
- `pointerup`/`pointercancel` end the pan and restore the cursor.
- `mousedown`/`auxclick` for button 1 call `preventDefault()` to suppress the browser/OS's own middle-click affordances (autoscroll-mode cursor, Linux X11 middle-click paste) that would otherwise fire alongside the custom pan.

## Why this doesn't conflict with card drag or normal scrolling (spec 6.3)

Card drag (`attachCardDragHandlers`, Kernel 81) already checks `event.button !== 0` and returns early for any non-primary-button pointer event on a card — a middle-button `pointerdown` on a card never starts a card drag, and (having done nothing, not called `stopPropagation`) bubbles up to `#board-scroll`'s own listener, which panning correctly picks up. Left-button drag, click, and keyboard interaction on cards are completely untouched by this addition. Mouse wheel scroll and the scrollbar are native browser behavior on `#board-scroll`'s `overflow: auto` — panning never calls `preventDefault()` on `wheel`, so they're unaffected.

## Verified live, not just by inspection

The Kernel 82 Playwright proof drove a real middle-button drag via Chrome DevTools Protocol's `Input.dispatchMouseEvent` (`button: 'middle'`) — Playwright's own high-level `mouse` API has no middle-button drag primitive, so raw CDP was necessary to exercise this specific interaction. Confirmed horizontal pan on a board with columns wide enough to overflow; a first attempt at vertical pan showed no scroll change purely because that board didn't yet have enough rows to overflow vertically (`scrollHeight` ≤ `clientHeight`) — adding rows until real vertical overflow existed (`scrollHeight: 1202` vs `clientHeight: 547`) confirmed vertical panning works identically to horizontal. Also confirmed a card drag performed immediately after a pan gesture completes with zero errors, and that panning itself produces zero console/page errors and zero network requests.
