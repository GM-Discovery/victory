# Kernel Report Back — Kernel 81: Storyboards Presentation Rework

**Kernel spec:** `Construction/Kernels/Kernel 81 — Storyboards Presentation Rework.md`
**Date:** 2026-08-06

## 1. Status

**PASS.** The plain HTML-table presentation is replaced with a CSS Grid board, cards match The Cave's compact typed-card visual language, `color_token` renders through a controlled whitelist, drag-and-drop is the primary card-movement interaction with the modal fallback retained, occupied-cell drops force a deliberate Swap/Move-existing/Cancel choice, column insert-left/right works, the red/rose/black palette replaced the wrong Writer's Room amber/gold, and one pinned card image with a real lightbox and a genuine gravestone-on-deletion behavior is new and working. The Kernel 80 backend — ownership, sharing, authority, hidden-card filtering, locks, live sync, export, account lifecycle — was preserved untouched except the two narrow, spec-pre-approved additions this kernel needed (`SetCardImage`, `SwapCards`).

Two real bugs were found and fixed during the real-browser proof pass, not shipped: a CSS Grid line-number collision between a band's "+ row" spacer and the next band's bar, and a genuine authorization gap where a Storyboard grant holder with no location membership was 403'd loading a card image they were fully entitled to see. Both are covered by regression tests now (§4, §8).

Baseline: clean tree at session start (Kernel 80 already merged to `main` per `git log`). Everything from this kernel is uncommitted for Grant's review, per house practice.

---

## 2. Pass-criterion ledger (spec §17)

| Criterion | Status | Evidence |
|---|---|---|
| Plain-table presentation replaced by CSS Grid | PASS | `#board-grid` is `display: grid`, zero `<table>` elements (`no_table_element: true` in proof output); `computeGridLayout` in `grid-model.js` |
| Red/rose/black palette | PASS | `--bg:#080607`/`--accent:#d84b4f` tokens on both `board.html` and `index.html`; screenshot `01-owner-venue-landing-palette.png` |
| Cards match The Cave's compact typed-card language | PASS | `.sb-card`: rounded corners, layered shadow, colorable face, flip button — screenshot `08-owner-after-drag-to-empty-cell.png` |
| `color_token` renders | PASS | `card_has_color_class: 1` after selecting "teal"; visible in every card screenshot from `07` onward |
| One card shown per cell in ordinary UI | PASS | `visibleCardForCell` (unit-tested); backend multi-card capability untouched |
| Occupied-cell drops offer swap/move-existing/cancel | PASS | Screenshots `09`-`10` (swap), `22`-`23` (move-existing); `occupied_dialog_shown: true`, `card_one_and_two_in_different_cells: true`, `no_cards_lost: true` |
| Drag-and-drop is the primary interaction | PASS | Pointer Events drag proven live, zero errors (`drag_errors: []`) |
| Keyboard/modal fallback remains | PASS | `fallback_move_visible: true`; screenshot `11` |
| Column insert-left/right works | PASS | `column_count_before: 2` → `column_count_after_insert_left: 3`, `"Inserted Left"` correctly ordered first |
| Sticky labels and bands remain legible during scrolling | PASS | Screenshot `14-owner-scrolled-sticky-row-labels.png` |
| One pinned thumbnail per card works | PASS | Upload → `image_upload_status: "Image attached."`, thumbnail + original variants both verified 200 via curl |
| Lightbox works | PASS | `lightbox_visible: true`, `lightbox_closed_on_escape: true` |
| Missing images produce gravestones | PASS | Real DB tombstone + reload: card face "Image unavailable", modal "Image unavailable (removed)", `image_asset_id` unchanged — screenshots `20`-`21` |
| Live updates remain server-authoritative | PASS | `audience_saw_live_card: 1` (remote creation), remote move verified via a second script |
| Hidden-card filtering remains intact | PASS | `audience_sees_hidden_card: false`, `audience_total_cards_visible: 1` while the board had 2+ non-hidden cards plus one hidden one |
| Established role affordances remain correct | PASS | Crew: no structure toolbar/column menu, sees hidden card, drags; Audience: none of the above, view-only |
| Backend permissions/ownership/export/lifecycle intact | PASS | Full Go suite green, zero changes to `boards.go`/`grants.go`/`authority.go`/`snapshot.go`/`export.go`/`identity`'s lifecycle code |
| Real Playwright browser proof passes | PASS | §4 |
| Grant's live visual review considers the surface usable | **PENDING** | This is explicitly a human judgment call the spec reserves for Grant, not something automatable — see §12 |

---

## 3. What Was Built

### Backend (bounded, per spec §12's change budget)

- `backend/migrations/092_kernel81_storyboard_card_image.sql` — `storyboard_cards.image_asset_id UUID`, deliberately no foreign key (comment explains why: two real asset-lifecycle paths would otherwise erase the "an image was pinned here" state the gravestone requirement depends on).
- `backend/internal/storyboards/types.go` — `ImageAssetID` field.
- `backend/internal/storyboards/cards.go` — `SetCardImage` (Crew+-unless-locked, same authority as any other card-content field) and `SwapCards` (new atomic transaction, both cards version-checked, both directions' locked-band override applied).
- `backend/internal/storyboards/card_image.go` (new) — `resolveCardImageStorageScope`: scopes upload storage to the *board owner's* producer membership (or the single-instance fallback location), never the uploading Crew member's own membership, since Storyboards has no Production/location of its own.
- `backend/internal/assets/card_image.go` (new) — `CreateReferencedImageAsset`, a leaner sibling of `HandleWorkshopUpload`'s pipeline with no membership gate of its own (the caller does its own authority check) — the existing upload endpoints are Producer-only and would have locked out ordinary Crew editors.
- `backend/internal/assets/read.go` — `storyboardCardImageViewerAllowed`, a raw-SQL inline authority check (not an import of `storyboards`, which would close a two-package cycle) fixing a real bug found via Playwright: a Storyboard grant holder with no location membership was 403'd loading a card image they were authorized to see.
- `backend/internal/storyboards/http.go` — `HandleCardSwap`, `HandleCardImage` (multipart POST to attach/replace, DELETE to clear).
- `backend/internal/storyboards/events.go` — `EventCardSwapped`.
- `backend/cmd/victory/main.go` — two new routes (`POST .../cards/swap`, `POST`/`DELETE .../cards/{id}/image`).

### Frontend (the primary work)

- `frontend/venues/storyboards/grid-model.js` (new) — pure presentation logic, dual Node/browser (same UMD pattern as `stage-runtime/geometry.js`): `computeGridLayout`, `cardsByCell`/`visibleCardForCell`, `roleAffordances`, `colorTokenClass` (fixed whitelist, no CSS injection path), `isImageGravestone`, `dropResolutionKind`.
- `frontend/venues/storyboards/board.html` — full rewrite: CSS Grid layout, Cave-inspired `.sb-card`, red/rose/black palette, Pointer-Events drag-and-drop with a 6px threshold and click/drag disambiguation, occupied-cell resolution modal + empty-cell picker for "move existing", column insert-left/right menu, card image attach/replace/remove/lightbox/gravestone UI, band collapse/expand toggle (backend capability already existed, had no UI until now).
- `frontend/venues/storyboards/index.html` — palette-only rewrite (structure/JS untouched).

### Tests

- `tests/storyboards/grid-model.test.js` (new) — 9 `node:test` cases, including a dedicated regression test for the grid-line-collision bug found live.
- `backend/internal/storyboards/kernel81_dbtest_test.go` (new) — 7 tests: image round-trip/clear, no-FK gravestone-preserving reference, authority/lock enforcement on image set, swap atomicity, swap version-conflict all-or-nothing, swap lock enforcement, swap same-card rejection.
- `backend/internal/assets/kernel81_storyboard_card_image_dbtest_test.go` (new) — 1 test proving the asset-authority fix: a grant holder with zero location memberships can read the image, a stranger with neither cannot, the owner always can.

---

## 4. Evidence (MANDATORY)

### Backend tests

```
$ TEST_DATABASE_URL=postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable \
  CONFIRM_TEST_DB_RESET=1 scripts/test/reset-test-database.sh
PASS: victory_test reset, migrated, and Go-side bootstrapped from empty. (92 migrations)

$ TEST_DATABASE_URL=... go test -count=1 -timeout=600s ./...
ok  	victory/backend/cmd/victory	0.013s
ok  	victory/backend/internal/access	2.492s
ok  	victory/backend/internal/assets	2.414s
...
ok  	victory/backend/internal/ewrite	14.891s
ok  	victory/backend/internal/identity	15.784s
ok  	victory/backend/internal/network	4.865s
ok  	victory/backend/internal/storyboards	14.310s
...
(full repo, zero failures, against a freshly reset victory_test — run twice, both clean)
```

### Frontend/node tests

```
$ node --test tests/
...
# tests 135
# pass 126
# fail 9   <- all 9 are tests/stage-runtime/dice.test.js, confirmed pre-existing
             (reproduced identically on `git stash`'d clean main, unrelated to
             this kernel: dice.js:477:19)
```

```
$ node --test tests/storyboards/grid-model.test.js
# tests 9
# pass 9
# fail 0
```

### Static checks

```
$ node --check frontend/venues/storyboards/grid-model.js frontend/venues/storyboards/socket.js
(clean)
$ node -e "... extract and new Function() every inline <script> block ..."
board.html inline scripts OK, 1 block(s)
index.html inline scripts OK, 1 block(s)
$ git diff --check
(clean)
```

### Live deploy (twice — initial, then the asset-authority fix)

```
$ docker compose build backend && docker compose up -d backend
 Container victory-backend  Started
$ docker logs victory-backend --tail 20
migrate: 1 pending migration(s): 092_kernel81_storyboard_card_image.sql
migrate: pre-apply backup written to /opt/victory/backups/victory_pre_migrate_...dump
migrate: applied 092_kernel81_storyboard_card_image.sql in 9ms
victory backend listening on :8081
$ docker exec victory-backend wget -qO- http://localhost:8081/health
{"ok":true,...}
(second deploy, after the read.go fix: no new migration, schema already current, health OK, zero errors in logs)
```

### Real browser proof (Playwright, headless Chromium, against the live production instance)

Three throwaway DB-seeded accounts (owner/crew/audience — no location memberships at all, deliberately, to catch exactly the class of bug that was found), sessions minted directly via the same `sessions.CreateSession` hashing scheme used in Kernel 80's proof, full cleanup afterward (boards, grants, sessions, users, orphaned test assets, and their storage directory all removed; verified zero residue).

**21 numbered screenshots** covering every required item from spec §14's 20-point list plus the gravestone follow-up: palette, CSS Grid, Cave-style card with color+image, lightbox open/closed-on-Escape, drag to empty cell, occupied-cell dialog → Swap, occupied-cell dialog → Move-existing (separate run), fallback Move UI, column-menu → Insert-left, sticky-label scrolling, band collapse, sharing panel, crew view (structure hidden, hidden-card visible, keyboard Enter opens modal), audience view (everything hidden/absent, zero drag/edit affordances), live remote card creation over WS, live remote card *move* over WS (separate run), and the tombstoned-asset gravestone on both the card face and the modal.

Two real bugs were caught here, not by inspection, and both fixed with regression coverage:

1. **Grid-line collision.** `board.html` originally computed the "+ row" button's `grid-row` line independently of `computeGridLayout`, and the two computations collided whenever a band had exactly one row — Playwright's error was literally `<span>Act Two</span> ... subtree intercepts pointer events` on a button click. Fixed by making `computeGridLayout` reserve the line itself (`addRowLine`), with a dedicated `node:test` regression test asserting no `addRowLine` value ever equals any `bandHeaderLine`/`rowLine` value.
2. **Card-image 403 for a legitimately authorized viewer.** A crew account with a real Storyboard grant but zero location memberships got 403 loading the card's hidden-card image (`console.error` × 3 in the proof output). Root cause: asset read authorization gates on location membership, unaware that this asset's *real* authority is the Storyboard grant. Fixed in `assets/read.go`; re-ran the full proof afterward with `crew_errors: []`.

---

## 5. How to Run (Operator Steps)

Already deployed live. No user-facing setup — Storyboards is on the map for any authenticated user, unchanged navigation from Kernel 80.

---

## 6. Operator Notes (CRITICAL)

- `TEST_DATABASE_URL` still needs exporting per session (standing note since Kernel 64).
- `storyboards` now imports `assets` directly (for the card-image upload helper) — this is a new one-way edge in the dependency graph. `assets` must never import `storyboards` back (would close the cycle); the asset-authority fix in `read.go` uses raw SQL specifically to avoid needing to.
- The Playwright + session-seeding procedure (raw session token minted directly in `auth.sessions`, matching `sessions.HashToken`'s exact `sha256(raw)` scheme) was reused unchanged from Kernel 80's session, including the same scratchpad `node_modules`. **This is now the second kernel in a row to use it successfully** — still worth capturing as a documented project procedure or skill; not done this kernel either (flagging again, not silently repeating the gap).
- `computeGridLayout` in `grid-model.js` is the single source of truth for every grid line number `board.html` uses. Any future change to what gets rendered between a band's rows and the next band's bar **must** go through it, not a local computation in `board.html` — see the collision bug in §4 for exactly what goes wrong otherwise.

---

## 7. Blockers & Workarounds

None new. Playwright + Chromium were already cached in this environment's scratchpad from Kernel 80's session (same conversation, same scratchpad path) — no reinstall needed. One test-PNG generation snag (an inline base64 blob wasn't a valid enough PNG for Go's `image.Decode`) was resolved by generating a real minimal PNG via raw `zlib`/`struct` in Python instead; not a product issue, purely a test-fixture one.

---

## 8. Deviations from Kernel

None from the locked product decisions (§2 of the spec). Two implementation details worth naming as deliberate choices rather than literal spec text:

- **Swap uses a new atomic backend endpoint rather than two sequential `MoveCard` calls.** The spec explicitly permitted this ("add a bounded backend swap operation" if two requests would produce unsafe transient state) — judged that a cell briefly holding two cards (visible to a third concurrent watcher) crossed that bar, given it's one of the kernel's own named required proof scenarios.
- **Card image storage is scoped to the board owner, not the uploading Crew member.** The existing asset-storage system has no concept of a Production-less upload; requiring the *uploader* to be a Producer somewhere (matching the existing `HandleWorkshopUpload` gate) would have silently excluded most real Crew editors. Scoping to the owner (whose board it is) instead keeps the feature usable for the role the spec explicitly names ("Crew: ...attach image") without inventing new authority machinery.

---

## 9. Known Issues

- **Keyboard-only "Move existing" resolution has no non-pointer path.** If a keyboard-only user reaches the occupied-cell dialog and picks "Move existing…", the empty-cell picker itself requires a pointer click. Named explicitly in `storyboards-accessibility.md` as the concrete next accessibility gap, not silently left undocumented.
- No screen-reader pass was performed (no tooling available in this environment); contrast ratios were not formally audited. See `storyboards-accessibility.md`.
- The Playwright/session-seeding procedure is still not captured as a reusable skill, despite being used successfully twice now (Kernel 80, Kernel 81).

---

## 10. Files Changed or Created

**Backend:**
- `backend/migrations/092_kernel81_storyboard_card_image.sql` (new)
- `backend/internal/storyboards/types.go`, `cards.go`, `events.go`, `http.go` (modified)
- `backend/internal/storyboards/card_image.go` (new)
- `backend/internal/storyboards/kernel81_dbtest_test.go` (new)
- `backend/internal/assets/card_image.go` (new)
- `backend/internal/assets/read.go` (modified — asset-authority fix)
- `backend/internal/assets/kernel81_storyboard_card_image_dbtest_test.go` (new)
- `backend/cmd/victory/main.go` (modified — 2 new routes)

**Frontend:**
- `frontend/venues/storyboards/board.html` (full rewrite)
- `frontend/venues/storyboards/index.html` (palette rewrite)
- `frontend/venues/storyboards/grid-model.js` (new)

**Tests:**
- `tests/storyboards/grid-model.test.js` (new)

**Construction/docs:**
- `Construction/Kernels/Kernel 81 — Storyboards Presentation Rework.md` (filed, mojibake cleaned)
- `Construction/Domains/Storyboards/storyboards-presentation-contract.md`, `storyboards-drag-and-drop.md`, `storyboards-card-image-contract.md`, `storyboards-accessibility.md` (new)
- `Construction/Domains/Storyboards/storyboards-ui-contract.md` (rewritten to describe Kernel 81's presentation, with Kernel 80's superseded table/plain-card design explicitly marked historical)
- `Construction/Domains/Storyboards/storyboards-domain-model.md`, `storyboards-permissions.md`, `storyboards-live-events.md` (small Kernel 81 addenda)
- `Construction/OperatorLogs/kernel-81-reportback.md` (this file)

---

## 11. Next Recommended Step

Per spec §21, the one criterion this reportback cannot itself satisfy is Grant's own live visual review. Recommend: Grant opens the live board UI, exercises drag-and-drop, color, image attach, and the occupied-cell dialog directly, and confirms the surface answers "yes" to the 16 questions in spec §21. If yes, per the spec's stop-loss rule (§20), pause further Storyboard feature work and let it sit as a stable base rather than immediately scheduling another cosmetic continuation. If the keyboard-only "Move existing" gap (§9) matters for Grant's actual usage, that's the next concrete, scoped piece of work — not a general accessibility restart.

---

## 12. Required project-memory updates completed

- [x] Kernel spec filed (`Construction/Kernels/Kernel 81 — Storyboards Presentation Rework.md`)
- [x] Reportback saved in repository (this file)
- [ ] `operator-log.md` — pending, next step in this session
- [ ] `operator-notes.md` — pending, next step in this session
- [ ] `kernel-maker-field-guide.md` — not needed, no repo-layout/test-command/runtime-mode change
- [ ] `dev-workflow.md` — not needed, no startup/port/service/migration-process change
- [ ] Master Actual Implementation Guide (`current-state.md`) — still lagging (now Kernels 79/79A/80/81 behind), same flagged-not-actioned status as Kernel 80's reportback
- [x] Fresh-install/bootstrap migration list — migration 092 is embedded automatically (`backend/migrations/embed.go`'s `//go:embed *.sql`)
- [ ] Help/command documentation — not applicable
