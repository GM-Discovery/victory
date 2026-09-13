# Storyboards Accessibility Notes (Kernel 81, reconciled Kernel 84)

Kernel 81's spec required "keyboard/modal fallback remains available" and
a specific "keyboard focus and Escape behavior" browser-proof item — not a
full accessibility audit. This document is scoped honestly to what was
actually built and verified, and flags what wasn't. Kernel 84 (spec §8)
reconciled every accessibility gap named across Kernels 81/82/83's own
reportbacks into the single canonical backlog below — this file, not a
scattered "known issues" line in each kernel's own report, is where a
future kernel should look first.

## Canonical backlog (Kernel 84 reconciliation)

| Gap | Category | Workaround today | First named in |
|---|---|---|---|
| Occupied-cell "Move existing" picker has no keyboard path | **Blocker** for keyboard-only users who land in this specific state | None — a keyboard-only user reaching an occupied-cell resolution currently cannot complete the move | Kernel 81 |
| Presence Tray Group Leader/Current Turn menu opens only via right-click | **Blocker** for keyboard-only users performing this specific action | None — the action is simply unreachable without a pointer; menu contents are fully accessible once open | Kernel 83 |
| Middle-mouse panning has no keyboard/touch equivalent | Convenience | Yes — normal scroll, scrollbar drag, arrow/Page Up/Page Down/Home/End on a focused scrollable region, or Tab-focusing a card and using the Move modal fallback all still reach any part of the board | Kernel 82 |
| No real screen-reader pass (VoiceOver/NVDA/JAWS) performed | General, not interaction-specific | `role`/`tabIndex`/`alt` attributes are correct by inspection and repo convention, not verified by ear | Kernel 81 |
| No formal WCAG contrast audit | General, not interaction-specific | Palette chosen for visual consistency with The Cave's card language, not measured | Kernel 81 |
| Drag-and-drop has no keyboard-driven equivalent beyond the fallback | Mitigated | The select/select/Move modal is a complete, non-degraded accessible path (spec 7.5's literal requirement), not a stopgap | Kernel 81 |

Three real, unmitigated gaps exist as of Kernel 84: the two blockers above,
plus the general screen-reader/contrast gaps. None were fixed in Kernel 84
(deliberately — see `kernel-84-reportback.md` §1.1: accessibility
reconciliation was explicitly scoped to gathering the backlog, not solving
it, and a trivial low-risk fix wasn't available for either blocker without
expanding into the redesign work both are named as needing).

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

## Kernel 83: Presence Tray right-click is mouse-first

The Presence Tray's Group Leader/Current Turn context menu
(`Construction/Domains/Venues/presence-tray-coordination-actions.md`) opens only
via `contextmenu` (right-click). Per spec §10.3 this was the explicitly
required interaction for Kernel 83, not an oversight — but it is a real,
named gap in the same category as the occupied-cell picker above: a
keyboard-only user currently has no way to open this menu at all, so they
cannot assign Group Leader or Current Turn themselves even when otherwise
authorized. Once open, the menu itself is fully accessible (real
`<button>` elements, closes on `Escape` or outside click) — the gap is
specifically the *entry point*, not the menu contents. A future kernel
adding a focusable per-chip trigger (e.g. a "⋮" button) would not need to
touch the menu's authority logic, only give it a second way in.

## Recommendation for a future kernel

If Storyboards accessibility becomes a priority (e.g. before wider rollout
beyond Grant's own instance), the canonical backlog table at the top of
this file is the starting point, not a general "improve accessibility"
restart. Fix the two named blockers first (occupied-cell keyboard
resolution, a Presence Tray keyboard entry point); the general
screen-reader/contrast passes are lower-urgency since nothing currently
depends on them to complete a task.
