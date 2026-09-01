# Kernel 94 — Storyboards Vue Rebuild Ledger

**Kernel spec:** `Construction/Kernels/Kernel 94 — Storyboards Vue Rebuild + Major Beautification.md`
**Status:** Passes 1–5 complete and live-reviewed by Grant, who reacted positively to each in turn ("that admittedly feels better," "looks better," "Well that's nothing functional add[ed] but did add a lot of good stupid clean movement fun," "It looks great"). PDF export (Grant's follow-up request, 2026-08-31) is built and verified. **Not yet formally closed** — spec §32/34 requires Grant to explicitly accept all three board states (Empty/Working/Full) as a discrete gate, and that hasn't been asked for or given as its own verdict yet, only pass-by-pass reactions during the build.
**Test target:** dev-local backend/Caddy (`Caddyfile.local`, port 8080) against the dev Postgres container, not production — three persistent test boards left in place for Grant's own review (see §7).

---

## 0. Repository audit (spec §26)

Done before writing any code:

- **No Vue, no npm, no build step anywhere in this repo.** `frontend/` is served as plain static files by Caddy (`root * /opt/victory/frontend; file_server`). The one precedent for a vendored third-party frontend library is `frontend/lib/pixi.min.js` — a single minified file, no bundler. Vue 3 was vendored the same way (`frontend/lib/vue.esm-browser.prod.js`, pinned 3.5.42), loaded via native `<script type="module">` and ES `import`, no build step introduced.
- **Domain model is Column (chronological axis) / Band (era/phase) / Row (concurrent lane) / Card (moment)** — confirmed against `backend/internal/storyboards/types.go` and Kernel 82's own terminology table. Kernel 94's "Time Frame/Event/Moment" language is descriptive, not a literal renaming target; the actual model was preserved unchanged (spec §3).
- **WS sync is coarse by design:** any mutation triggers a full snapshot re-fetch and (pre-rebuild) a full DOM rebuild — `frontend/venues/storyboards/socket.js` (unchanged) pushes either a fresh snapshot or a bare change-type event that the client always resolves by reloading. Preserved as-is; Vue's reactivity replaces the manual DOM rebuild, not the sync model itself (spec §19: don't re-solve networking that already works).
- **Two defects matching spec §14/§15 were found by reading the code, before any browser testing:**
  - §14: `AddColumn` (`backend/internal/storyboards/columns.go`) always appends after every existing column with no position parameter. The old board.html's generic "+ Column (end)" toolbar button called it directly, so on a Timeline board it would insert a new column *after* the protected Ending column. `insertColumnFlow`'s append-then-reorder trick already avoided this for the column menu's Insert left/right items, just not for the generic toolbar button.
  - §15: the Presence Tray's Leader/Turn assignment menu only opened via `contextmenu` (right-click) on a plain `<div>` chip — no keyboard path existed to it at all.

---

## 1. Pass 1 — Vue foundation + parity

Rebuilt `frontend/venues/storyboards/board.html` (2,150-line single file, no components) as ~13 Vue 3 components under `frontend/venues/storyboards/app/` (`store.js` centralizing all REST calls and mutations; `AppRoot`, `Toolbar`, `PresenceTray`, `BoardGrid`, `CardTile`, `CardModal`, `OccupiedModal`, `PickBanner`, `Lightbox`, `ShareModal`, `ReferencePanel`, `ReferenceField`, `ReferenceFieldModal`). `grid-model.js` and `socket.js` kept unchanged and still relied on directly. Visual design was deliberately left near-identical to the original (CSS moved to `board.css`, ~verbatim) — parity first, redesign in later passes, per spec §35.

Bugs fixed in this pass:
- **§14 protected-Ending fix:** the generic "+ Column" action now routes through the same append-then-reorder path as the column menu whenever a Timeline board's Ending column exists, so a new column always lands before it. Label changed from "+ Column (end)" to "+ Column" to match the new behavior.
- **§15 keyboard-access fix:** the Presence Tray chip is now a real `<button>` (focusable, native Enter/Space activation); both click and right-click open the same Leader/Turn menu.

Also wired `tests/storyboards` into `scripts/test/alpha-gate.sh`'s node test invocation — it existed (`grid-model.test.js`, 9 tests) but wasn't part of the gate.

**Browser proof:** live-tested with two real simultaneous logged-in sessions (not just one) — created a Timeline board, exercised column insert (confirmed Ending stayed last), card create/edit/delete, drag-and-drop between cells, reload persistence, Reference Panel field editing, and the Leader/Turn menu with a genuine second user present (confirmed both "Make Group Leader" and "Give Turn" appear correctly, via a real DOM click — Playwright's synthetic `.click()` had a timing quirk in single-user testing that made it look broken when it wasn't; a raw `dispatchEvent` proved the logic itself was correct). Checked against Vue's dev build (unminified, warns on prop/key issues) with zero console output beyond the expected "dev build" notice.

Grant caught a real gap here that wasn't a bug in the work itself: the first hand-off gave a `127.0.0.1:8080` URL without confirming Grant's environment could actually reach sandbox-local ports, and he'd gone to production (which was never touched) first. No code issue — a review-process gap, resolved by confirming his access and giving the exact URL/login.

---

## 2. Pass 2 — Spatial composition, empty state

Scope: board structure, hierarchy, whitespace, empty state (spec §35 Pass 2), explicitly not cards or motion yet.

- Removed hard borders on every cell/column-header/row-label (previously every cell was fully ruled, spreadsheet-style). Kept a single outer container border.
- Band headers changed from a full-width tinted bar to a small rounded label chip riding a hairline, with Edit/Lock/Remove revealed only on hover/focus (not shown at rest).
- Row labels: same hover/focus-reveal treatment for their edit/remove icons.
- "+ Column"/"+ Band" toolbar restyled as ghost buttons (ownership de-emphasized, ownership is Director+ only anyway).
- Empty-cell "+ card" affordance redesigned (bigger glyph, smaller label, low opacity) — **then Grant flagged it as "a little much" when every empty cell on a working/populated board showed it simultaneously.** Fixed by making it invisible by default (`opacity: 0`) once the board has any real content anywhere, revealed only on hover/keyboard-focus of that specific cell; a genuinely untouched board (`.is-sparse`, zero cards) keeps it visible at rest, since spec §10 requires an obvious first action when there's nothing else to point at.
- Signature element: a single static (non-animated, to stay in Pass 2's scope) warm radial-gradient wash over the top-left of the board, present only while `.is-sparse`, gone the instant any card exists — plus a one-line italic invitation ("A blank stage. Place your first card to begin.").

**Browser proof:** screenshotted sparse vs. working states before and after the empty-cell-affordance fix; confirmed hover-reveal on both band actions and row-label icons via Playwright `.hover()`.

---

## 3. Pass 3 — Card system

Scope: the card face itself (spec §6), not the editing modal's layout.

- Dimensions grown modestly (152→168px max-width, 96→104px min-height, 176→196px max-height) — still bounded, not "enormous" per spec §6's explicit warning.
- Typography: title now clamps at 2 lines with real line-height instead of one unclamped line; front text clamps at 4 lines (was 3).
- **Category is now shown on the card face** — it existed in the data model (`StoryboardCard.Category`) but was never rendered anywhere except inside the edit modal.
- Depth: two-layer shadow instead of one flat drop-shadow; a hover lift (`translateY(-2px)` + deeper shadow).
- Back face gets a small "BACK" label so a flipped card is unambiguous at a glance, especially with several flipped at once.
- Grant asked why only one card visibly "grew" after this pass — verified via actual `getComputedStyle`/`getBoundingClientRect` measurement (not assumption) that all cards *did* get the same wider footprint; height is legitimately content-driven (min-height floor, max-height ceiling with `overflow:hidden` + line-clamp as the hard backstop) — a title-only card is supposed to look different from one with a category and two sentences. Separately fixed a real rough edge this surfaced: a title-only card had its content pinned to the top with dead space below it instead of centering — `.sb-card-front`/`.sb-card-back` are now flex columns with `justify-content: center`.

**Browser proof:** measured actual computed styles/bounding rects for multiple cards side by side to distinguish "content-driven sizing" from "the CSS only applied to one card" before answering Grant's question, rather than guessing.

---

## 4. Pass 4 — Interaction + motion

Scope: spec §9's named-appropriate motions, explicitly avoiding "fragile cleverness."

- **Card relocation now visibly slides instead of teleporting.** Cards are nested inside per-cell containers (kept, for the drop-target hit-testing CardTile's drag logic depends on), so a card changing cells is, to Vue, an unrelated unmount in the old cell and a fresh mount in the new one — no native shared-element identity for `<TransitionGroup>` to animate between. Implemented the classic FLIP technique instead in `BoardGrid.js`: a `watch()` pair (default `flush: 'pre'` to capture every `[data-card-id]` element's `getBoundingClientRect()` before the DOM patches, `flush: 'post'` to invert the newly-placed element back to its old screen position with `transition: none` and immediately release it into an animated one) keyed off a computed signature of every card's row/column id plus every band's collapsed state (so band collapse/expand reflow animates too, for free). Verified this isn't just theoretical — listened for the browser's actual `transitionrun` event during a real drag and confirmed a genuine `translate(-535px, ...)` invert fired.
- Card flip changed from an instant `display` swap to a squash-to-a-sliver-and-back animation, with the front/back content swap timed to the animation's midpoint (when the card is edge-on and effectively invisible) — deliberately not a true 3D flip, which would fight the card's own `overflow: hidden`/shadow.
- New cards pop in (scale + fade) instead of just appearing.
- All five modals (Card/Occupied/Share/ReferenceField/Lightbox) plus the Presence menu now fade/scale via Vue `<Transition>` instead of snapping.
- Drag ghost gets a slight scale-up on top of its existing rotation, for a "picked up" read.
- `prefers-reduced-motion` explicitly tested (Playwright `reducedMotion: "reduce"` context), not just coded to a media query and assumed correct: confirmed flip becomes instant with no `is-flipping` class ever applied, and confirmed no errors during a drag under that setting.

---

## 5. Pass 5 — Dense-board exhibition pass

Scope: no new features — load a genuinely populated board (seeded via the API rather than clicking through the UI: 8 columns, 4 bands, 9 rows, 46 cards with varied color/category/text length) and find what only shows up at real scale. It found two real, independent bugs:

- **Boundary column headers (Beginning/Ending) had a translucent background** (`var(--accent-soft)`, ~16% opacity) — fine at rest, but on a scrolled, populated board, whatever card was scrolling underneath the sticky header visibly bled straight through it. **Pre-existing since Kernel 82** (present in the original board.html's CSS, verified by reading it), never surfaced before because no earlier test board had enough content to scroll under a boundary column. Fixed by layering the same tint over an opaque base instead of relying on alpha transparency.
- **Horizontal scrolling was silently broken for any board wide enough to exceed the viewport.** Pass 2's empty-state invite line required wrapping `.board-scroll` in a new `.board-column` div; the flex rule that let the board shrink to fit its container (`flex: 1 1 auto; min-width: 0`) was still targeting `.board-scroll` by descendant selector, but `.board-scroll` was no longer a *direct* flex child of `#board-root` — `min-width`/`flex` on a non-flex-item element does nothing. The board silently overflowed the whole page instead of scrolling internally, shoving the Reference Panel off-screen. Only visible with enough columns to actually exceed 1050px of available width — every earlier test board (3–4 columns) was too narrow to trigger it. Fixed by moving the rule onto the actual flex item (`.board-column`) and adding `min-width: 0` to `.board-scroll` itself for its own (nested, column-direction) flex context.

Both confirmed fixed by direct measurement (`scrollWidth`/`clientWidth`/`scrollLeft` before and after), not just a screenshot.

---

## 6. Export PDF (Grant, 2026-08-31: "a JSON feels like an ugly way to export art")

Clarified scope with Grant before building (three real options — print-to-PDF / rendered PNG / public read-only link — are materially different builds). **Decision: print-optimized view → browser's native Save-as-PDF now; PNG image export explicitly deferred** (see §8).

Built as a `@media print` stylesheet applied to the existing live board view (Toolbar's new "Export PDF" button just calls `window.print()`) — no backend change, no new dependency, per spec §25. Found and fixed three more real bugs while building it:

| Bug | Found how | Fix |
|---|---|---|
| `@click="window.print()"` inline template expression threw `Cannot read properties of undefined (reading 'print')` | Real Playwright click, not just visual inspection | Vue's compiler resolves a bare *call-shaped* global reference differently from an assignment (the adjacent `window.location.href = ...` worked fine inline) — moved both Export buttons to real methods in `setup()` |
| Band-chip labels and the Reference Panel heading were invisible (light pink `--accent-strong`, meant for a near-black page, on a white one) | Screenshot review of the actual print output | Added a darker `--accent-strong` override inside the print media query |
| Reference Panel textareas printed as solid black boxes (inherited the dark theme's `#0d0a0b` fill) | Same screenshot review | Forced light input/textarea/select colors in print |
| A collapsed band or a collapsed Reference Panel would silently drop its content from the export (collapsed content isn't in the DOM at all, `v-if`-gated) | Reasoned through before it could ship, then verified the underlying mechanism fires correctly | `beforeprint`/`afterprint` window listeners flip a local-only "force expand" flag for the duration of the print — no server write, doesn't touch the real persisted collapse state or broadcast to other viewers |
| Stray "+ row in {band}" buttons were still visible in print output | Screenshot review | Added a class and hid it |

**Testing note, disclosed rather than glossed over:** headless Chromium's `window.print()` fires `beforeprint`→`afterprint` back-to-back almost instantly (no real dialog to hold it open), which reverts the force-expand flag before Vue's re-render can land — so the collapsed-band-during-print case could not be fully verified end-to-end via browser automation the way everything else in this kernel was. What *was* verified directly: the flag does flip correctly on `beforeprint` (confirmed via direct console instrumentation), and the rest of the print stylesheet (typography, color, layout, page breaks, the two Vue components' JS-side force-expand logic in isolation) was confirmed via `page.emulateMedia({media:"print"})` screenshots and an actual generated PDF (`page.pdf()`, verified as a real 3-page landscape PDF via file inspection) with bands left in their normal expanded state. This one specific interaction (collapsed band + real user's `window.print()` dialog blocking) rests on documented browser behavior rather than an automated end-to-end proof, and is worth a quick manual double-check from Grant.

---

## 7. Boards left for review

Three persistent test boards on the dev-local instance, all created via the `k94tester`/`k94tester2` throwaway accounts (password signup temporarily enabled on this dev backend only — production untouched):

- **Empty:** `77e9e4fa-863e-42ed-9965-19fee26eef36` ("Pass 2 Sparse Preview")
- **Working:** `fcd247fb-7610-4022-8eae-ba00dd770e52` ("Kernel 94 Pass 1 Review")
- **Full:** `8c222da4-77b6-4d9c-9386-97e22fe4ad65` ("Full Production Board", 46 cards)

---

## 8. Decisions recorded (not to be re-asked)

- Vue vendored as a single unbuilt file (`frontend/lib/vue.esm-browser.prod.js`), same pattern as `pixi.min.js` — no bundler, no npm, matching the rest of the repo.
- CSS Grid + snapping placement kept (not a freeform canvas) — matches spec §4's explicit preference for the simplest interaction model that still feels spatial.
- Card size bounds (104–196px height) are deliberate, not a bug — content-driven within a floor/ceiling, per spec §6.
- Motion implemented via a measure-before/after (FLIP) technique rather than restructuring card rendering into a flat `<TransitionGroup>`-friendly list — lower risk, didn't touch the proven per-cell drop-target hit-testing from Pass 1.
- Card flip is a 2D squash animation, not a 3D perspective flip — avoids fighting the card's own `overflow: hidden`/shadow.
- Export: PDF via native browser print now; rendered PNG image export explicitly deferred (see below), a public unauthenticated read-only share link was raised as a third option and **not** built (bigger access-control change, Grant didn't choose it).

## 9. Deferred / candidates for K95 or a later K94 polish pass

- **PNG rendered-image export** — Grant, 2026-08-31: build later. Needs a real decision on approach (client-side canvas capture library vs. something else) before starting; not scoped here.
- Card editing stayed in the modal rather than moving toward inline/contextual editing on the card face itself (spec §11's "avoid modal dialogs for routine edits unless there is a compelling reason") — the modal covers image upload, permission-gated fields, and the keyboard-accessible Move fallback, which is a defensible reason to keep it, but a lighter inline path for just title/color was considered and not built, to keep Pass 3 bounded.
- Column/row reorder (as opposed to card move or band collapse) doesn't get the FLIP slide animation — scoped out of Pass 4 as a less-frequent, more deliberate action; only card-cell-membership and band-collapse changes are covered.
- Very long row labels still truncate with an ellipsis (136px fixed column) — has a `title` tooltip fallback, not fixed further.
- The `.is-sparse` spotlight glow still shows on an empty board's print export — harmless, arguably still coherent ("waiting to be started"), left as-is rather than suppressed.

---

## 10. What Grant should still confirm

1. All three states (Empty/Working/Full) accepted as their own explicit gate — reactions so far were per-pass, not a final combined verdict (spec §32/34 wants this named explicitly before PASS is called).
2. The one export edge case noted in §6 — printing a board with a band or Reference Panel left collapsed, in a real (non-automated) browser.
3. Whether PNG export should be scheduled now or genuinely left for K95.
