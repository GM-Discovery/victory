# Kernel 81 — Storyboards Presentation Rework

**Status:** READY FOR IMPLEMENTATION
**Type:** Frontend continuation and product-correction kernel
**Primary track:** Victory Core
**Secondary tracks:** Socio, Anthology, Education
**Sequence position:** After Kernel 80
**Product surface:** Storyboards
**Implementation posture:** Preserve the proven Kernel 80 backend; replace the Storyboard board presentation layer

---

## 0. Kernel contract

Kernel 81 replaces the current Storyboard board presentation with the intended Victory interaction and visual language.

Kernel 80 successfully delivered ownership, explicit sharing, role-based authority, columns, rows, bands, cells, cards, server-authoritative live synchronization, hidden-card filtering, locks, eWrite links, JSON export, account lifecycle integration, and real browser automation.

Those systems are not to be rewritten.

Kernel 81 exists because the deployed presentation does not match the intended product:

- plain HTML-table appearance;
- weak card identity;
- no primary drag-and-drop interaction;
- append-only column creation;
- stored card colors not rendered;
- no card thumbnail attachment;
- wrong venue palette;
- presentation optimized around implementation rather than the Storyboard.

This kernel must deliver one coherent vertical:

```text
User opens an existing Storyboard
→ sees a polished bounded CSS Grid
→ bands, rows, columns, and cells are immediately legible
→ compact Cave-style cards fill occupied cells
→ user drags a card to an empty cell
→ occupied-cell drops offer deliberate resolution
→ live users receive the authoritative update
→ card color and pinned thumbnail render correctly
→ missing image produces a gravestone
→ keyboard/modal fallback remains available
```

The result must feel like a Victory feature, not an administrative table.

---

## 1. Agent assignment

Kernel 81 should be executed by the same agent that completed Kernel 80, provided it follows this specification as a fresh presentation implementation.

Reasons:

- it already knows the Storyboards backend;
- it already has working Playwright and authenticated-session browser proof;
- it knows the WebSocket event model;
- it knows the exact permission boundaries;
- it discovered the current presentation defects directly;
- replacing the agent would introduce rediscovery cost and regression risk.

However, continuity does not make the current frontend authoritative.

The implementation instruction is:

> Preserve the Kernel 80 backend and contracts. Discard the current `board.html` rendering approach where necessary. Rebuild the presentation intentionally from the product decisions in Kernel 81.

Do not defend the existing table merely because the same agent built it.

---

## 2. Locked product decisions

### 2.1 Rendering approach

Use **DOM-based CSS Grid**.

Do not use PixiJS for the primary Storyboard grid.

Reason:

- Storyboards are bounded and semantically structured;
- columns and rows require labels;
- bands contain adjacent rows;
- sticky labels and two-axis scrolling matter;
- keyboard navigation and text editing matter;
- DOM drag interactions are appropriate;
- the stage runtime's tactical canvas is not a spreadsheet-style board.

The implementation may borrow visual ideas from existing Victory cards, but it must not force Storyboards into the tactical-map rendering model.

### 2.2 Visual card occupancy

The ordinary UI displays one card per cell.

The Kernel 80 backend's multiple-card capability remains intact.

The frontend must not display multiple stacked cards in one cell during ordinary use.

When an operation targets an occupied cell, offer:

- **Swap** — exchange the two cards' cells;
- **Move existing** — move the existing card to another chosen empty cell, then place the incoming card;
- **Cancel**.

Do not silently stack.

Do not immediately rewrite the database to enforce one card per cell.

### 2.3 Card visual reference

Use the compact index-card language already present in **The Cave** as the primary visual reference.

Do not use literal physical index-card proportions.

These are typed digital cards, not handwritten paper cards.

Cards should be:

- compact;
- horizontally efficient;
- shorter than the stage-runtime 192×132 card node where appropriate;
- readable at normal board zoom;
- clearly card-like;
- visually distinct from cells;
- able to display typed content without becoming posters.

Required visual characteristics:

- rounded corners;
- subtle drop shadow;
- colorable face;
- title;
- short front text;
- optional pinned thumbnail;
- flip affordance;
- hidden indicator where authorized;
- locked indicator;
- selected/dragging state;
- clear focus state.

### 2.4 Palette

Use the red/rose/black Victory palette associated with Trailers, Third Place, Audition Hall, and Catharsis.

Do not copy the Writer's Room amber/gold palette.

Reuse current design tokens rather than approximating independently.

### 2.5 Card colors

`card.color_token` must affect the rendered card.

Requirements:

- approved color tokens map to controlled CSS variables/classes;
- no arbitrary CSS injection;
- color remains readable against text;
- hidden/locked/selected states remain distinguishable;
- cards without a color use a deliberate default.

### 2.6 Card images

Each card supports at most one pinned image.

The interaction should resemble the existing token-image attachment pattern:

- attach one Victory asset;
- show it as a thumbnail;
- click to enlarge;
- replace;
- remove;
- no gallery;
- no arbitrary external embed.

If the referenced asset disappears, the card retains the reference state and displays a gravestone placeholder.

Do not silently remove the thumbnail area or discard the missing reference.

### 2.7 Drag-and-drop

Drag-and-drop is the primary card-movement interaction.

The existing modal/select movement path remains as a keyboard-accessible and touch-safe fallback.

Drag-and-drop must not be the only movement path.

### 2.8 Column insertion

Director+ must be able to insert a column:

- to the left of an existing column;
- to the right of an existing column;
- at the end.

Use the existing reorder endpoint if it can perform the operation safely.

A small backend addition for atomic insertion is allowed if the current endpoint would create visible race conditions or transient incorrect state.

### 2.9 Existing backend

Do not rewrite ownership, sharing grants, authority tiers, hidden filtering, locks, WebSockets, snapshot composition, account lifecycle, JSON export, or eWrite links.

Only change backend code where required for:

- card asset reference;
- gravestone-preserving asset behavior;
- atomic column insertion;
- occupied-cell resolution if existing endpoints cannot express it safely.

---

## 3. Kernel goals

1. Replace the plain table with a polished CSS Grid.
2. Render compact Cave-inspired typed cards.
3. Make drag-and-drop the primary movement path.
4. Handle occupied cells deliberately.
5. Support arbitrary column insertion and usable labels, sizing, scrolling, and bands.
6. Add one pinned card image with lightbox and gravestone behavior.
7. Prove the presentation in real browsers.

---

## 4. Required preflight

Before implementation:

1. Read the Kernel 80 reportback and Storyboards construction documents.
2. Inspect the live `storyboards/board.html`.
3. Inspect The Cave's card styling and interaction.
4. Inspect stage-runtime card visuals only for transferable ideas.
5. Inspect the token-image attachment flow.
6. Inspect asset deletion and gravestone behavior elsewhere in Victory.
7. Inspect Storyboard WebSocket payloads and mutation endpoints.
8. Inspect current Storyboards role rendering.
9. Capture baseline screenshots.
10. Record the browser-automation setup as a reusable project procedure if not already documented.

Do not begin by editing the backend.

First determine what can be completed entirely in the frontend.

---

## 5. CSS Grid board model

### 5.1 Grid structure

The rendered board should use one bounded CSS Grid or a small coordinated set of CSS Grids.

It must represent:

```text
top-left corner
column headers
band labels
row labels
cells
cards
```

The DOM structure must preserve semantic relationships.

### 5.2 Sticky labels

Required:

- column headers remain visible during vertical scrolling;
- row labels remain visible during horizontal scrolling;
- band identity remains understandable while scrolling;
- top-left intersection does not overlap controls.

Handle the previously discovered floating-widget collision.

### 5.3 Bands

Bands must be more than colored row labels.

Visually show:

- beginning and end of band;
- band label;
- rows contained;
- collapsed state;
- locked state;
- band-level controls for authorized users.

When collapsed:

- contained rows and cards are hidden from ordinary board view;
- band summary remains visible;
- hidden content does not lose state;
- live updates still reconcile correctly.

### 5.4 Cells

Cells must:

- have visible but restrained boundaries;
- show empty drop-target state during drag;
- show occupied state;
- preserve one visible card;
- remain addressable by row and column;
- avoid excessive whitespace.

### 5.5 Scaling

The board must remain usable with many columns, many rows, compact cards, long labels, collapsed bands, and two-axis scrolling.

The 200-column technical safeguard remains.

Do not attempt to render all 200 columns beautifully at once. Do prove the interface remains stable and navigable.

---

## 6. Card presentation

### 6.1 Front

The front may display:

- pinned thumbnail;
- title;
- short front text;
- category/label;
- color;
- hidden/locked indicators;
- eWrite-link indicator;
- flip button.

### 6.2 Back

The back may display:

- longer back text;
- eWrite link;
- intentionally minimal metadata;
- flip-back affordance.

The back should not require navigating to a separate page.

### 6.3 Sizing

Cards must be compact enough to support board overview.

Do not size cards to mimic literal 3×5 index cards.

Use typed-text density.

Define controlled width and minimum/maximum height behavior.

Avoid cells expanding without bound because one card contains long text.

### 6.4 States

Provide visible states for:

- normal;
- hover;
- keyboard focus;
- selected;
- dragging;
- drop target;
- hidden;
- locked;
- image missing.

Avoid using color alone.

---

## 7. Drag-and-drop

### 7.1 Primary interaction

Crew+ users may drag unlocked cards.

Audience/Cast do not receive draggable affordances.

### 7.2 Pointer behavior

Support:

- mouse;
- touch/pointer events where practical;
- clear drag threshold;
- no accidental drag when clicking flip/edit/link controls;
- auto-scroll near board edges;
- visible source and destination states.

Use Pointer Events or a maintained dependency already compatible with the project.

Do not introduce a large frontend framework solely for drag-and-drop.

### 7.3 Server authority

On drop:

1. determine target cell;
2. inspect current snapshot;
3. submit authoritative mutation;
4. wait for success/event confirmation;
5. reconcile from server state;
6. recover gracefully from conflict or denial.

Do not treat DOM movement as committed state before the server accepts it.

### 7.4 Occupied-cell drop

When target cell is occupied, show a compact resolution dialog:

- Swap;
- Move existing;
- Cancel.

#### Swap

Atomically exchange cell assignments where possible.

If no atomic endpoint exists and two requests would produce unsafe transient state, add a bounded backend swap operation.

#### Move existing

Let the user select an empty destination for the existing card, then place the incoming card into the target.

This must not lose either card.

#### Cancel

Return the card to its authoritative origin.

### 7.5 Fallback

Retain modal/select movement.

Keyboard users must be able to choose row, choose column, resolve occupied target, submit, and receive status.

---

## 8. Column insertion and structure controls

### 8.1 Column menu

Each column header should offer authorized controls:

- insert left;
- insert right;
- rename;
- move left/right or reorder;
- remove.

Do not expose these controls to Crew or viewer roles.

### 8.2 Row and band controls

Director+ controls should remain available but visually subordinate to the board.

Crew may edit unlocked band labels as established.

Do not make the board look like a database-administration screen.

### 8.3 Atomicity

If insert-left/right is implemented as add-then-reorder, prove:

- no duplicate sort orders;
- no visible incorrect order broadcast;
- no race under two connected editors;
- no card movement.

Otherwise add a small atomic backend operation.

---

## 9. Card image attachment

### 9.1 Data model

Add one optional card asset reference.

Preferred conceptual field:

```text
storyboard_cards.image_asset_id NULL
```

Use a real FK only if existing asset-deletion behavior permits gravestones.

If `ON DELETE SET NULL` would erase the knowledge that an image once existed, use a reference/attachment model that preserves a missing-asset state.

The required product result is:

> Deleted or unavailable asset — visible gravestone, not silent disappearance.

### 9.2 Upload/selection

Reuse the token-image attachment approach where practical.

Authorized card editors may choose/upload, pin, replace, and remove deliberately.

Do not expose private assets beyond their authority.

### 9.3 Thumbnail

Thumbnail behavior:

- compact;
- deliberately cropped;
- does not dominate the card;
- has alt text or accessible label;
- does not distort card sizing;
- lazy-loaded where useful.

### 9.4 Lightbox

Clicking the thumbnail opens a simple full-size lightbox.

Required:

- close button;
- Escape closes;
- click outside closes where appropriate;
- keyboard focus management;
- no gallery navigation;
- no remote embed.

### 9.5 Gravestone

When an asset is missing or unreadable:

- preserve layout;
- show missing-image icon/label;
- indicate the attachment is unavailable;
- authorized editor may replace/remove;
- do not expose private filename or path;
- exports preserve the missing-reference state where appropriate.

---

## 10. Live synchronization

Preserve Kernel 80 WebSocket behavior.

Prove:

- remote card creation appears;
- remote card move updates cells;
- remote swap updates both cards;
- remote color change appears;
- remote thumbnail attachment appears;
- remote asset deletion produces gravestone;
- hidden cards remain filtered;
- revoked access still stops updates;
- reconnect restores current layout.

Correctness is more important than micro-optimization.

---

## 11. Permissions and UI affordances

### Audience / Cast

- view only;
- no drag;
- no card edit controls;
- no structure controls;
- hidden cards absent.

### Crew

- drag cards;
- create/edit cards;
- change card color;
- attach image;
- edit unlocked band labels;
- no structural column/row/band changes.

### Director+

- Crew capabilities;
- insert/reorder/remove columns;
- add/reorder/remove rows and bands;
- lock/unlock.

### Owner

- Director+ capabilities;
- sharing/permission controls;
- archive/delete/export.

The UI must not merely hide controls. The server remains authoritative.

---

## 12. Backend change budget

Kernel 81 is primarily frontend work.

Allowed backend changes:

1. one-card image reference/attachment;
2. missing-image gravestone projection;
3. atomic card swap if required;
4. atomic column insertion if required;
5. snapshot fields needed by the renderer;
6. tests for those bounded additions.

Not allowed:

- redesign Storyboard permissions;
- redesign WebSockets;
- redesign ownership;
- replace rows/bands/columns schema;
- remove multi-card backend capability;
- rebuild export;
- rework account lifecycle.

If implementation begins expanding beyond this list, stop and document why.

---

## 13. Required tests

### 13.1 Presentation logic

Extract and test:

- grid model construction;
- band-to-row grouping;
- card-to-cell placement;
- one-visible-card rule;
- occupied-cell resolution;
- color-token mapping;
- image gravestone projection;
- role affordance calculation.

Do not leave all behavior buried in one inline script.

### 13.2 Drag behavior

Test:

- empty-cell move;
- occupied-cell swap;
- occupied-cell move-existing;
- cancel;
- locked card denial;
- locked band denial;
- Crew allowed;
- Audience/Cast denied;
- stale-version recovery;
- live remote update during drag.

### 13.3 Column insertion

Test insert-left, insert-right, insertion between occupied columns, stable ordering, preserved card positions, and two-editor conflict safety.

### 13.4 Image behavior

Test attach, replace, remove, thumbnail rendering, lightbox, unauthorized asset denial, deletion gravestone, and export/account-lifecycle safety.

### 13.5 Regression

Run the full Go suite, frontend/node suite, fresh-install smoke, existing Storyboards tests, WebSocket tests, hidden-card tests, export tests, account lifecycle tests, and Kernel 79 eWrite-link tests.

---

## 14. Required browser automation

Playwright proof is mandatory.

Use disposable owner, Crew, and Audience/Cast accounts.

Capture screenshots or traces for:

1. red/rose/black Storyboard board;
2. CSS Grid labels and bands;
3. Cave-style compact card;
4. rendered `color_token`;
5. drag to empty cell;
6. occupied-cell swap;
7. occupied-cell move-existing;
8. fallback movement path;
9. insert column left/right;
10. thumbnail attachment;
11. lightbox;
12. missing-asset gravestone;
13. remote WebSocket move on second tab;
14. Audience/Cast no drag/edit controls;
15. hidden card absent;
16. Crew controls present;
17. Director structure controls present;
18. owner sharing controls present;
19. keyboard focus and Escape behavior;
20. horizontal and vertical scrolling.

No PASS based only on API evidence for this presentation kernel.

---

## 15. Required artifacts

```text
Construction/Kernels/Kernel 81 — Storyboards Presentation Rework.md
Construction/Storyboards/storyboards-presentation-contract.md
Construction/Storyboards/storyboards-drag-and-drop.md
Construction/Storyboards/storyboards-card-image-contract.md
Construction/Storyboards/storyboards-accessibility.md
Construction/OperatorLogs/kernel-81-reportback.md
```

Update the Storyboards UI contract to distinguish Kernel 80's placeholder implementation from Kernel 81's intended presentation.

Capture the Playwright/session-seeding procedure as a reusable project skill or documented test procedure.

---

## 16. Required evidence

### Baseline

```bash
cd /opt/victory
git status --short
git rev-parse HEAD
git branch --show-current
docker ps --format '{{.Names}}\t{{.Status}}\t{{.Ports}}'
```

### Backend

```bash
cd /opt/victory/backend
GOCACHE=/tmp/victory-gocache go test -count=1 -timeout=600s ./...
```

Use isolated `TEST_DATABASE_URL`.

### Frontend

Run `node --check`, extracted inline-script checks, frontend unit tests, and Playwright.

### Static

```bash
git diff --check
```

### Browser evidence

Report exact role, action, expected result, actual result, and screenshot/trace location.

Do not use production personal accounts for automation.

---

## 17. Pass criteria

Kernel 81 passes when:

- the plain-table presentation is replaced by a polished CSS Grid;
- Storyboards use the red/rose/black palette;
- cards visibly match The Cave's compact typed-card language;
- `color_token` renders;
- one card is shown per cell in ordinary UI;
- occupied-cell drops offer swap/move-existing/cancel;
- drag-and-drop is the primary interaction;
- keyboard/modal fallback remains;
- column insert-left/right works;
- sticky labels and bands remain legible during scrolling;
- one pinned thumbnail per card works;
- lightbox works;
- missing images produce gravestones;
- live updates remain server-authoritative;
- hidden-card filtering remains intact;
- established role affordances remain correct;
- backend permissions, ownership, export, and lifecycle remain intact;
- real Playwright browser proof passes;
- Grant's live visual review considers the surface usable enough to continue.

The last criterion matters.

Kernel 81 is a product-correction kernel. Automated correctness alone is not sufficient.

---

## 18. Partial and fail rules

### PARTIAL

Use when:

- CSS Grid ships but drag remains fallback-only;
- card visuals improve but do not match the compact intended language;
- occupied cells silently stack;
- image attachment lacks gravestone behavior;
- Playwright proof is incomplete;
- mobile/touch remains unusable;
- role controls are visually wrong but server authority remains safe.

### FAIL

Use when:

- Kernel 80 backend is unnecessarily rewritten;
- cards are lost during drag/swap;
- hidden cards leak;
- Audience/Cast can mutate;
- remote state diverges;
- asset deletion silently erases the attachment state;
- the interface remains an administrative table;
- visual review still shows the product concept was missed;
- Grant is locked out;
- Storyboards export or lifecycle regresses.

---

## 19. Non-goals

Kernel 81 does not:

- implement Microscope;
- add nested cards;
- add merged cells;
- allow cards to span cells;
- add image galleries;
- add visual/PDF export;
- add templates;
- add semantic analysis;
- redesign Storyboard sharing;
- add public links;
- remove backend multi-card support;
- redesign eWrite links;
- rewrite Storyboards as PixiJS.

---

## 20. Stop-loss rule

Kernel 81 is the test of whether the proven Storyboards backend can support the intended product.

If the surface still does not feel right after:

- CSS Grid;
- compact Cave-style cards;
- primary drag-and-drop;
- correct palette;
- color rendering;
- pinned thumbnails;
- deliberate occupied-cell behavior;

then stop.

Do not immediately schedule another cosmetic continuation.

Instead, perform a product and rendering reassessment before adding more Storyboard features.

This prevents continued investment merely because the backend already exists.

---

## 21. Immediate operator outcome

At completion, Grant must be able to answer:

1. Does this look like Victory rather than an admin table?
2. Can I understand columns, rows, bands, and cells immediately?
3. Do the cards resemble the compact cards in The Cave?
4. Do card colors actually render?
5. Can I drag a card naturally?
6. Can I still move it without dragging?
7. Does an occupied cell force a deliberate choice?
8. Can I insert a column exactly where I need it?
9. Can I pin one thumbnail to a card?
10. Can I enlarge it?
11. Does a deleted image leave a gravestone?
12. Do remote users see accepted changes?
13. Do Audience and Cast remain view-only?
14. Do hidden cards remain hidden?
15. Did the proven Kernel 80 backend remain intact?
16. Is this surface good enough to build Microscope-style play on top of?

Kernel 81 succeeds when Storyboards finally presents the strong Kernel 80 system as the product Grant intended.
