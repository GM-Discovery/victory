# Storyboards Accessibility Notes (Kernel 81)

Kernel 81's spec required "keyboard/modal fallback remains available" and
a specific "keyboard focus and Escape behavior" browser-proof item — not a
full accessibility audit. This document is scoped honestly to what was
actually built and verified, and flags what wasn't.

## What's actually in place

- **Every card is keyboard-reachable and operable without a pointer.**
  `.sb-card` has `tabIndex = 0` and `role="button"`; `Enter`/`Space` opens
  the card editor, identical to clicking. This was true in Kernel 80 and
  is unchanged.
- **Card movement never requires a pointer.** The modal's "Move to" row
  `<select>` + column `<select>` + "Move" button (spec 7.5's required
  fallback) is fully Tab/arrow-key/Enter operable and is the *only* path
  on a card whose role doesn't afford dragging (Audience/Cast never get a
  drag handler at all, so this is their sole path even for cards they
  could otherwise interact with — though in practice they can't move
  cards regardless of input method).
- **Escape closes both the lightbox and the "move existing" cell-picker
  overlay.** A single `document`-level `keydown` listener handles both;
  verified live (`lightbox_closed_on_escape: true` in the Kernel 81
  browser-proof output).
- **Focus states are visible**, not just implied by hover:
  `.sb-card:focus-visible` gets a 2px `--accent-strong` outline with
  offset, distinct from the plain drop-shadow hover state.
  `input`/`textarea`/`select` get a 1px `--accent` outline on focus.
- **Card images have `alt` text** ("Card image for {title}"), not empty
  or missing.
- **State is never color-only.** Locked/hidden cards carry text badges
  ("LOCKED"/"HIDDEN"), not just a border-style or background-color
  change; the occupied-cell drop state pairs a background tint with an
  `inset box-shadow` outline, not color alone.
- **The occupied-cell resolution dialog and gravestone labels are plain
  text**, readable by a screen reader the same way any other modal text
  would be — no custom widgets, no canvas-rendered text anywhere in
  Storyboards (a deliberate consequence of the CSS Grid vs. PixiJS
  decision in `storyboards-presentation-contract.md`).

## What was verified in the real browser proof

- Tab-focusing a card and pressing Enter opens the editor (tested as the
  Crew account specifically, screenshot-free but assertion-backed:
  `crew_keyboard_open_modal_ok: true`).
- Escape closes the lightbox (`lightbox_closed_on_escape: true`).
- The fallback Move UI renders and is reachable via the same modal every
  keyboard-only flow already goes through
  (`11-owner-fallback-move-ui.png`).

## What was not done — flagged, not silently skipped

- **No screen-reader testing was performed** (no VoiceOver/NVDA/JAWS pass;
  this environment has no such tooling available). The `role="button"`/
  `tabIndex`/`alt` attributes above are correct by inspection and by
  matching this repo's existing conventions elsewhere, not verified by
  ear.
- **No formal WCAG contrast audit.** The card color palette (light
  pastels with dark ink text, matching The Cave's own card language) was
  chosen for visual consistency with the existing reference, not measured
  against a contrast ratio target. Worth a dedicated pass if Storyboards
  becomes a primary surface rather than a GM/Director tool.
- **Drag-and-drop itself has no non-pointer equivalent beyond the
  fallback.** There is no keyboard-driven "pick up card, arrow to target
  cell, drop" interaction layered on top of the pointer-drag path — the
  select/select/Move modal is the accessible path, full stop, not a
  degraded version of a richer keyboard-drag experience. This matches
  spec 7.5's literal requirement but is worth naming as a gap rather than
  implying keyboard drag exists.
- **The occupied-cell "Move existing" empty-cell picker is mouse-first**
  in its current form (click a highlighted cell) — it has no keyboard
  equivalent of its own. A keyboard-only user reaching this state (via
  the fallback Move UI landing on an occupied cell) currently has no way
  to resolve it without a pointer. This is a real, named gap for the next
  kernel to close, not something this kernel claims to have solved.
- **Auto-scroll during drag** has no keyboard equivalent either, but this
  only matters during pointer-drag, which is already pointer-only by
  definition.

## Recommendation for a future kernel

If Storyboards accessibility becomes a priority (e.g. before wider rollout
beyond Grant's own instance), the two named gaps above — keyboard
resolution of an occupied-cell drop, and any real screen-reader pass — are
the concrete next steps, not a general "improve accessibility" restart.
