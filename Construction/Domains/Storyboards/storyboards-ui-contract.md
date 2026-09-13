# Storyboards UI Contract

Implementation: `frontend/venues/storyboards/index.html`, `board.html`,
`grid-model.js`, `socket.js`. No build step (static files, per repo
convention) — plain `<script>` blocks, `fetch(..., {credentials:"include"})`
for REST, `WebSocket` for live events.

## Kernel 80 vs Kernel 81 — what changed and why

Kernel 80 shipped a functionally-correct but deliberately plain
presentation: an HTML `<table>` grid, unstyled bordered-`<div>` cards, the
Writer's Room amber/gold palette (copied as a template, not a deliberate
choice), append-only columns, and no pointer-drag interaction. A post-deploy
design review with Grant found this did not match his actual product
intent, captured in `Construction/Kernels/Kernel 81 — Storyboards
Presentation Rework.md` and `kernel-80-reportback.md` §8.

Kernel 81 replaced the presentation layer only. **Nothing below this
section describes Kernel 80's table/plain-card implementation — that code
no longer exists.** The backend (`backend/internal/storyboards/`) is
unchanged in its ownership/sharing/authority/hidden-filtering/WebSocket/
export/lifecycle behavior; only `card_image.go` and `cards.go`'s
`SetCardImage`/`SwapCards` were added, per Kernel 81's explicitly bounded
backend-change budget.

## Surface map

- `index.html` — Venue landing: owned/shared tabs, archived-boards toggle,
  create-board form. Now uses the red/rose/black palette (`--bg`,
  `--panel`, `--accent`, etc.) shared with Trailers/Third Place/Audition
  Hall/Catharsis, replacing Kernel 80's amber/gold. Structure and JS
  unchanged from Kernel 80 — only the `<style>` block was rewritten.
- `board.html?id=<board_id>` — a single CSS Grid (`display: grid` on
  `#board-grid`, no `<table>` anywhere), with plain `<div>`s placed via
  inline `grid-row`/`grid-column` computed by `grid-model.js`'s
  `computeGridLayout`. Sticky corner/column-headers/row-labels/band-bar
  labels via `position: sticky`. Compact Cave-inspired cards
  (`.sb-card`) render inside cells.
- `grid-model.js` (new) — pure presentation logic with no DOM/network
  dependency, dual Node/browser export (same UMD pattern as
  `frontend/lib/stage-runtime/geometry.js`): grid line-number layout,
  cell/cards grouping, the one-visible-card-per-cell rule, role
  affordances, the color-token whitelist, gravestone detection, and the
  drop-resolution-kind decision (move/swap/noop). Tested under
  `node:test` at `tests/storyboards/grid-model.test.js` — see that file
  for why (a real layout collision bug was caught here, not by
  inspection; see "Known layout gotcha" below).
- `socket.js` — unchanged from Kernel 80. On any `storyboard/*` event
  (including the new `storyboard/card_swapped`), `board.html` still just
  re-fetches the full snapshot (`loadSnapshot()`) rather than patching
  state incrementally — correct by construction, and Kernel 81's spec
  explicitly said "correctness is more important than micro-optimization"
  for live sync (§10).

## The one-visible-card-per-cell rule (spec 2.2)

The backend still allows multiple cards per cell (Kernel 80 capability,
deliberately untouched). The frontend enforces a **visual** one-card rule:
`grid-model.js`'s `visibleCardForCell` picks the lowest
`sort_order_in_cell` card in a cell; any others are invisible in ordinary
board view (they still exist server-side and would surface again if this
rule were ever relaxed). Dragging or moving a card onto an occupied cell
never silently stacks — it always opens the occupied-cell resolution
dialog (see below).

## Drag-and-drop (spec 7) — see `storyboards-drag-and-drop.md` for detail

Pointer Events (`pointerdown`/`pointermove`/`pointerup`/`pointercancel`),
not HTML5 `dragstart`/`drop` — matches the existing `show-runs/show.html`
pointer-drag idiom elsewhere in this codebase. A 6px movement threshold
distinguishes a drag from a click-to-open-editor tap. A cloned
`.drag-ghost` element tracks the pointer; the cell under the pointer
(`document.elementFromPoint`) gets `.is-drop-target` or
`.is-drop-occupied` styling live during the drag. Auto-scroll near
`#board-scroll`'s edges. Only Crew+ (and only Director+ if the card or its
row's band is locked — mirrors `canMutateCard`'s server-side rule exactly)
ever get a `pointerdown` handler attached to a card at all; Audience/Cast
cards are inert.

The pre-Kernel-81 non-drag fallback (row `<select>` + column `<select>` +
"Move" button in the card editor modal) is retained unchanged and reachable
without ever touching a pointer — required by spec 7.5, and doubles as the
touch-safe path on devices where a careful drag is impractical. Each card
also carries a small "⠿" move-handle button that simply opens the same
editor modal, for discoverability.

## Occupied-cell resolution (spec 7.4)

Both the drag path and the fallback "Move" button funnel through the same
`grid-model.js` `dropResolutionKind(existingCard, draggedCardId)` decision
(`"move"` / `"noop"` / `"occupied"`). On `"occupied"`, `#occupied-modal`
offers:

- **Swap** — `POST /api/storyboards/{id}/cards/swap` (new in Kernel 81,
  `cards.go`'s `SwapCards`), a single atomic transaction exchanging both
  cards' `(row_id, column_id, sort_order_in_cell)` under both cards'
  version checks. Chosen over two sequential `MoveCard` calls specifically
  to avoid a transient state where both cards briefly share one cell and
  the origin cell is briefly empty, visible to a third concurrently-loaded
  watcher.
- **Move existing…** — closes the dialog, shows a `#pick-banner` and adds
  `.is-pick-target` + a click handler to every currently-empty cell; the
  chosen cell receives the existing card (`MoveCard`), then the originally
  dragged/moved card lands in the target cell (`MoveCard`). Escape or the
  banner's Cancel button aborts and reloads the snapshot.
- **Cancel** — reloads the snapshot, discarding the drag with no request
  sent.

## Card visuals (spec 2.3/2.5/6) — see `storyboards-card-image-contract.md`
## for the image/thumbnail/lightbox/gravestone half specifically

`.sb-card` is a compact (max 152×176px) Cave-inspired card: rounded
corners, layered drop shadow, a light-pastel `--card-bg` selected from a
**fixed whitelist** of named color tokens (`grid-model.js`'s
`COLOR_TOKENS`/`colorTokenClass`) — never the raw `color_token` string
interpolated into a style attribute, closing the "no arbitrary CSS
injection" requirement by construction. Front face shows an optional
thumbnail, title, and truncated front text (3-line clamp); a small "⟲"
button flips to the back face (`back_text`) in place, no page navigation.
Hidden/locked state render as small pill badges, not color alone.

## Column insert-left/insert-right (spec 2.8/8)

Each column header carries a "⋮" menu (Director+ only) with Insert
left/Insert right/Rename/Remove. Insertion is implemented entirely in the
frontend against Kernel 80's existing, unmodified endpoints — no new
backend route: `POST .../columns` (always appends), then
`POST .../columns/reorder` with the full ordered id list (old order plus
the new column spliced into the desired position). `ReorderColumns`
already validates the posted id list is exactly the board's current
column set inside one transaction, so a concurrent structural edit from
another Director between the two calls fails this second call cleanly
(`invalid_resolution`) rather than corrupting order — the user is told the
column landed at the end and to retry, never left with silently duplicate
or skipped `sort_order` values.

## Card image attach/replace/remove/lightbox/gravestone

See `storyboards-card-image-contract.md`.

## Occupied-removal resolution flow for columns/bands/rows (unchanged from Kernel 80)

Column/band/row "remove" controls still first attempt a plain `DELETE`;
on `column_occupied`/`band_occupied`/`row_occupied` the client `prompt()`s
for a resolution (type `delete`, or another sibling's exact title/label,
or blank to cancel) — unchanged from Kernel 80, out of Kernel 81's scope.

## eWrite link mount point (unchanged from Kernel 80)

The card editor calls `window.attachEwriteRuleLink(container, {objectType:
"storyboard_card", objectId: card.id})` verbatim.

## My People is a suggestion source, never a picker that grants anything (unchanged from Kernel 80)

The sharing panel's "People you know" list renders My People relationship
`stage_name`s as plain, non-interactive labels. Sharing always requires
the deliberate handle-entry step.

## Known layout gotcha found via real browser testing (Kernel 81)

`computeGridLayout` must reserve a grid row-line for the "+ row in Band X"
button between a band's last member row and the next band's bar, even
though only Director+ viewers ever render anything into that line
(`addRowLine` in the returned layout). Before this, `board.html` computed
that line independently (`lastRowLine + 1`), which is off by one whenever
a band has exactly one row — it silently lands on the exact same grid line
as the *next* band's bar. Playwright caught this as `<span>Act Two</span>
… subtree intercepts pointer events` when clicking "+ row in Band 1"; a
static check would not have (both elements have valid, individually
sensible-looking `grid-row` values — the collision only exists across the
two independent computations). `tests/storyboards/grid-model.test.js` has
a dedicated regression test (`addRowLine never collides with...`) that
would fail immediately if this ever reverts to a two-computation design.

## Layout gotcha: floating nav widgets collide with page-edge content (unchanged from Kernel 80)

`venue-account-badge.js` and `back-to-map.js` both float `position: fixed`
in a page corner whenever the page has no `<header>` element to dock into
— neither Storyboards page uses one. `board.html` reserves
`padding-right: 235px` on `.page-header`; `index.html` reserves
`padding: 0 210px`. Unchanged from Kernel 80's fix, still correct under
the Kernel 81 palette rewrite (verified in the Kernel 81 browser proof).
