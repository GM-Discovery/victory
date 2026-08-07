# Storyboards Presentation Contract (Kernel 81)

What the CSS Grid board guarantees, independent of drag-and-drop or card
visuals specifically (those have their own documents). This is the
structural/layout contract.

## Rendering technology

`board.html`'s `#board-grid` is a single `display: grid` element — not a
`<table>`, not a nested-grid-per-band. `frontend/venues/storyboards/
grid-model.js`'s `computeGridLayout(snapshot)` is the one function that
decides every element's `grid-row`/`grid-column` line numbers; `board.html`
never computes a line number itself (see `storyboards-ui-contract.md`'s
"Known layout gotcha" for what went wrong the one time it tried to).

This was a locked decision (Kernel 81 spec §2.1), explicitly *not* the
`stage-runtime` PixiJS canvas: Storyboards is a bounded, labeled,
two-axis-scrollable spreadsheet-shaped structure with sticky headers and
text-heavy cells — a fundamentally different rendering problem from
`stage-runtime`'s tactical battle-map canvas (uniform infinite grid,
snap-to-cell-center token placement, no row/column labels or bands at
all). The two share no code and are not expected to.

## Grid line numbering (`computeGridLayout`)

```text
column 1        = corner / row-label / band-label column
columns 2..N+1  = the board's columns, in sort_order

row 1           = header row (corner + column headers)
for each band, in sort_order:
  one line       = band bar
  one line each  = the band's member rows (skipped entirely if collapsed)
  one line       = reserved "add row" spacer (skipped if collapsed)
```

The "add row" spacer line is reserved **unconditionally**, whether or not
the current viewer's role affords an actual button there — this is the
detail that keeps two independently-computed line numbers from ever
colliding (see the dedicated regression test in
`tests/storyboards/grid-model.test.js`).

## Sticky elements

| Element | Sticky axis | z-index | Notes |
|---|---|---|---|
| `.corner-cell` | top + left | 6 | highest — sits above both header row and row-label column |
| `.col-header` | top | 4 | |
| `.row-label` | left | 2 | |
| `.band-bar-inner` | left | (default, above band-bar background) | only the label/controls stick; the bar's background scrolls normally, which is fine — it's already full-width |

`.board-scroll` is the one scrolling container (`overflow: auto`,
`max-height: 76vh`), both axes. `#board-grid` itself is `width:
max-content` so it grows naturally as columns are added, up to the
existing 200-column technical safeguard (`storyboards.MaxColumns`,
unchanged from Kernel 80 — this kernel's job was to prove the interface
stays *navigable* at scale, not to render 200 columns beautifully at
once, per spec §5.5).

## Band collapse (spec 5.3)

`SetBandCollapsed`/`is_collapsed` already existed in the Kernel 80
backend (`bands.go`, `HandleBandCollapse`) but had no UI. Kernel 81 adds
the "▾"/"▸" toggle button on `.band-bar-inner`. A collapsed band's rows
and cards are removed from grid layout entirely (`computeGridLayout`
returns no `rowLine`/`addRowLine` entries for it) — they are not merely
`display: none`'d, they're never placed. Their data is untouched
server-side and reappears immediately on expand; a live update inside a
collapsed band (e.g. another viewer editing one of its cards) still
arrives over the WebSocket and updates `snapshot`, it just isn't rendered
until the band is expanded again.

## Cell sizing

Fixed-width columns (`132px` corner, `172px` per board column) rather
than content-driven sizing — this keeps the grid legible and predictable
at any column count, and keeps cards (max `152×176px`) from ever forcing
a cell wider than its neighbors. `min-height: 118px` on `.cell` gives a
card room without excess whitespace around short cards (spec §5.4's
"avoid excessive whitespace").

## Role-driven rendering, never role-driven authority

Every conditional render (`+ column`, `+ row`, `+ card`, column ⋮ menu,
band Edit/Lock/Remove, card drag handlers, card hidden/lock checkboxes,
Sharing/Export/Archive buttons) reads `grid-model.js`'s
`roleAffordances(snapshot.viewer_tier)` — `viewer_tier` is always the
server-resolved tier from the last `GET`/WS snapshot, never a value the
client computes or could forge. `roleAffordances`'s boundaries are unit
tested against the exact backend tier ladder
(`tests/storyboards/grid-model.test.js`) so a future backend authority
change that isn't mirrored here fails a fast local test instead of
silently drifting.
