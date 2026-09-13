# Kernel 94 Visual Language

Candidate patterns from the Storyboards rebuild, for Kernel 95 to evaluate before propagating outward. None of this was applied anywhere outside Storyboards — per Kernel 94 §21, that pass belongs to K95. Written after Passes 1–5 and the PDF export were built and live-reviewed.

Nothing here replaces the existing token language (`--bg`, `--panel`, `--panel-strong`, `--accent`, `--accent-strong`, `--muted`, `--accent-soft`, all defined in `board.css`'s `:root`) — those are unchanged from the rest of Victory. This is about what got *built with* them, not a new palette.

---

## Hierarchy through spacing, not boxes

The single biggest visual shift: cells, column headers, and row labels lost their hard 1px borders. What used to be a fully-ruled spreadsheet grid now reads as open space, with structure implied by:
- generous padding inside cells rather than lines between them;
- a `box-shadow` hairline only where two regions genuinely need separating (under sticky column headers, right of sticky row labels) — not a border on every edge;
- background color differences (the pastel cards) doing the work borders used to do.

If a surface elsewhere in Victory is currently a dense grid of bordered boxes and doesn't have a specific reason to be one (a real spreadsheet-like editing surface, say), this is the pattern to try first.

## The "chip on a hairline" grouping pattern

A structural grouping (Storyboards' Band) is a small rounded pill — `background: var(--accent-soft); border-radius: 999px; padding: 4px 10px;` — sitting on a single `border-top` hairline that spans the full row, rather than a full-width tinted bar. Reads as a heading over the content, not a container around it. Candidate for any "this group of items belongs together" UI that currently uses a filled section header bar.

## Secondary controls: invisible until hover or focus

Non-primary actions (a band's Edit/Lock/Remove, a row's rename/remove icons) are `opacity: 0` at rest and `opacity: 1` on `:hover` or `:focus-within` — never `display: none`, so they stay in the layout and reachable by keyboard tab order the whole time; they just don't visually compete with content until you're actually interacting with that region. This is different from the *primary* creation affordance (the empty-cell "+ card"), which follows a related but distinct rule below.

**Caution for K95:** apply this only to controls that have some other reasonable discovery path (hovering the region they belong to is intuitive). Don't hide a control that has no other way to be found.

## Primary affordance visibility is state-dependent, not fixed

The empty-cell "+ card" button is invisible at rest *once the board has any real content*, but stays visibly present (dim, not hidden) on a genuinely empty board. The rule: an affordance's visibility should track how much the user still needs to be told where to start, not be a single fixed CSS state. Concretely: gate a "quiet by default" treatment behind a computed "is this surface still empty/new" condition, not just "is this a secondary control."

This came from a real correction — the first version made "+ card" moderately visible everywhere, and it read as noisy on a populated board even though it looked fine on an empty one.

## Cards as objects, not form containers

- Two-layer shadow (`0 2px 5px ... , 0 10px 22px ...`, plus a 1px inset highlight) reads as more physically "lifted" than one flat drop-shadow.
- A subtle hover lift (`translateY(-2px)` + a deeper shadow) gives a "grabbable" read before any drag starts.
- Content is centered vertically within the card (flex column, `justify-content: center`) rather than pinned to the top — a sparse card (just a title) doesn't look like it's missing something.
- Size floors and ceilings (a min-height and a max-height, with `overflow: hidden` plus text `-webkit-line-clamp` as the hard backstop) let a card's height genuinely reflect how much content it holds, without ever becoming either a sliver or a wall.

## Motion: FLIP for cross-container repositioning, not a component refactor

Where an item visually moves to a different DOM parent (a Storyboards card changing cells) and a native `<TransitionGroup>` isn't feasible without restructuring how the item is rendered, the lower-risk answer is the classic FLIP technique applied imperatively: measure on-screen position before the data changes (`flush: 'pre'` watcher), invert the newly-placed element back to its old position with no transition immediately after the DOM patches (`flush: 'post'` watcher), then release it into a transitioning one. Reads as one continuous slide even though it's two different elements underneath. See `BoardGrid.js`'s `cardPositionSignature` watcher pair for the reference implementation.

For a flip/reveal effect (Storyboards' card flip), a 2D squash (`scaleX` toward ~0, content swapped at the animation's midpoint) is a safer default than a true 3D `backface-visibility` flip, which fights `overflow: hidden` and box-shadows on the flipping element.

All of it goes through one shared `prefersReducedMotion()` check (`app/motion.js`) for anything JS-driven, and every pure-CSS transition/animation has its own `@media (prefers-reduced-motion: reduce)` block disabling it. Both are required, not just the CSS side — anything timed in JS (the flip's content-swap delay, the FLIP slide) needs the JS-side check too, or reduced-motion users still get the timing-dependent behavior with the visual smoothing stripped out from under it.

## Modal/popup entrance

A shared `<Transition name="modal">` (opacity + `scale(0.96) translateY(6px)` on the inner panel, ~160ms) wraps every modal-backdrop-style overlay. A lighter `<Transition name="popup">` (opacity + `scale(0.96)`, ~120ms, no translate) is for small contextual menus. Both are two CSS classes reused across every instance rather than bespoke per-component — worth lifting as shared classes if K95 finds more modals to wrap.

## Print/export as a first-class view, not an afterthought

The PDF export reuses the live interactive view with one `@media print` stylesheet rather than a second rendered path — but that meant explicitly re-deciding, for print, everything that's normally sticky, hover-revealed, or collapsed-by-default: sticky positioning doesn't mean anything on paper (forced `position: static`), hover-revealed controls are simply irrelevant chrome in an exported artifact (hidden outright, not left at their resting opacity), and anything that can be collapsed (a band, the Reference Panel) needs an explicit "expand for the duration of export only" escape hatch or its content silently vanishes from the export. If any other venue grows an export/print surface, budget for this category of decision, not just color/layout.

Also worth remembering for any print stylesheet: CSS custom properties tuned for a dark theme (light text, near-black fills) need their own light-theme override inside the print media query — `color-scheme`/theme tokens do not invert themselves automatically, and it's easy to only remember the obvious ones (background) and miss text-color tokens like `--accent-strong` that read fine on black and disappear on white.
