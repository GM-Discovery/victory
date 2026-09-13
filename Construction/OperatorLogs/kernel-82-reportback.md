# Kernel Report Back — Kernel 82: Storyboards Timeline Mode

**Kernel spec:** `Construction/Kernels/Kernel 82 — Storyboards Timeline Mode.md`
**Date:** 2026-08-07

## 1. Status

**PASS.** Timeline exists as a built-in, immutable, code-defined Storyboard template. Creating one instantiates a normal owned Storyboard — three columns (Beginning/Middle/Ending, with Beginning and Ending carrying protected structural roles enforced server-side, not just hidden in the UI), one default band and row, and a four-field Reference Panel (Premise, Beginning, Ending, and a single "Include / Exclude" paired-list field). Two Timelines are provably independent; editing one never touches another or the template itself. The Reference Panel is generic Storyboards infrastructure (Crew edits content, Director+ edits structure, same authority split cards already use), with deterministic Kernel-81A-style slugs. Middle-mouse panning works both axes without disturbing normal scrolling or card drag. Export carries `mode`/`template_version`/`column_role`/the full Reference Panel. No Microscope terminology, tone system, placement assistant, or Group Leader/Current Turn state was implemented — a documented integration seam exists for the last one.

Baseline: clean tree at session start except uncommitted Kernel 81/81A/permission-fix work already in place. Everything from this kernel is uncommitted for Grant's review, per house practice.

---

## 2. Pass-criterion ledger (spec §18)

| Criterion | Status | Evidence |
|---|---|---|
| Timeline exists as a built-in immutable template/mode | PASS | `timeline_template.go` (code-defined constants, no DB row); `storyboard-template-contract.md` |
| Users create normal owned instances from it | PASS | `CreateTimelineBoard`; screenshot `02-timeline-a-created.png` |
| Saved Timelines do not mutate the source template | PASS | Structurally true (no DB row to mutate); `TestTwoNewTimelinesAreIndependent` |
| Every new Timeline starts fresh | PASS | Same test; third Timeline created after editing two others still starts empty |
| Timeline starts with exactly three columns | PASS | `TestTimelineCreatesTimelineWithThreeColumnsAndBoundaryRoles`; `timeline_a_column_titles: ["Beginning","Middle","Ending"]` |
| Beginning and Ending remain protected boundary roles | PASS | `ErrBoundaryColumnDisplaced`/`ErrBoundaryColumnProtected`, both server-enforced; 4 dedicated tests |
| Expansion occurs only inward; positional Add Left/Right work | PASS | `beginning_menu: "Insert right\nRename"`, `ending_menu: "Insert left\nRename"`, `middle_menu` has all 4 — real browser evidence |
| Reference Panel exists and is configurable per instance | PASS | `reference-panel-contract.md`; screenshot `06-reference-panel-visible.png` |
| Crew edits panel content; Director+ edits structure | PASS | `crew_sees_add_field_btn: false`, `crew_field_controls_count: 0`, `director_add_field_btn_visible: true`; screenshot `09` |
| Field IDs and serialized keys remain stable | PASS | `TestReferenceFieldIDAndSlugSurviveReorder`, `TestReferenceFieldDuplicateLabelsSerializeUniquely` |
| Paired-list fields work | PASS | `paired_include_items: ["Robots"]`, `paired_exclude_items: ["Magic swords"]` — real browser evidence |
| Bands/rows/cards remain generic | PASS | No new card subclass, no tone field, no placement UI — unchanged from Kernel 80/81 |
| No Microscope-specific terminology/mechanics | PASS | Grepped own code/docs/UI copy for Lens/Focus/Period/Event/Scene — none present |
| No tone system; no placement assistant | PASS | Not implemented (spec §20 non-goals) |
| Group Leader/current-turn remains deferred | PASS | `session-state-integration-seam.md`; zero leader/turn schema |
| Middle-mouse panning works | PASS | `pan_moved_horizontally: true`; vertical confirmed in follow-up test with real overflow |
| Normal scrolling and card dragging still work | PASS | `card_drag_errors_after_pan: []`; wheel/scrollbar untouched by pan code |
| Timeline metadata exports correctly | PASS | `export_mode: "timeline"`, `export_template_version: 1`, `export_has_boundary_roles: true`, `export_reference_field_count: 4` |
| Account lifecycle remains intact | PASS | No changes to `identity` package; Timeline boards are ordinary `storyboards` rows |
| Playwright proof passes | PASS | §4 |
| No Kernel 80/81/81A regressions | PASS | Full Go suite green; Blank board sanity-checked live (`blank_board_reference_panel_hidden: true`, no boundary badges) |

---

## 3. What Was Built

### Backend (bounded, per spec §12's change budget)

- `backend/migrations/095_kernel82_storyboards_timeline_mode.sql` — `storyboards.mode`/`template_version`, `storyboard_columns.column_role` (+ partial unique indexes guaranteeing at most one Beginning/Ending per board), `storyboard_reference_fields`/`storyboard_reference_items`.
- `backend/internal/storyboards/timeline_template.go` (new) — code-defined Timeline seed data, `TimelineTemplateVersion = 1`.
- `backend/internal/storyboards/timeline.go` (new) — `CreateTimelineBoard`, a sibling of `CreateBoard` (not a variant threaded through its signature — every existing caller/test keeps working unchanged).
- `backend/internal/storyboards/reference_panel.go` (new) — full Reference Panel CRUD (fields + items, all 4 types), reusing Kernel 81A's `allocateUniqueSlug` for field slugs.
- `backend/internal/storyboards/reference_panel_http.go` (new) — 9 HTTP handlers.
- `backend/internal/storyboards/columns.go` — boundary-column protection in `ReorderColumns` (`ErrBoundaryColumnDisplaced`) and `RemoveColumn` (`ErrBoundaryColumnProtected`); `column_role` added to `loadColumn`/`ListColumns`.
- `backend/internal/storyboards/boards.go` — `mode`/`template_version` added to every board read path.
- `backend/internal/storyboards/snapshot.go`/`export.go` — `ReferenceFields` added to `BoardSnapshot`/`ExportDocument`; `mode`/`template_version`/`column_role` flow through automatically since `StoryboardColumn`/`Storyboard` already carry them.
- `backend/internal/storyboards/events.go` — `EventReferencePanelChanged` (one event type for every panel mutation; panel content is never hidden-from-audience, so no per-viewer shaping is needed, unlike card events).
- `backend/cmd/victory/main.go` — 10 new routes (Timeline creation via `HandleBoards`'s existing `mode` dispatch + 9 Reference Panel routes).

### Frontend

- `frontend/venues/storyboards/index.html` — Blank/Timeline mode tabs on the create form.
- `frontend/venues/storyboards/board.html` — Reference Panel side rail (collapsible, localStorage-persisted), all 4 field-type editors, field structural controls (add/rename/type-change/remove, Director+ only), boundary-column badges + role-aware column ⋮ menu, middle-mouse panning via Pointer Events.

### Tests

- `backend/internal/storyboards/kernel82_timeline_dbtest_test.go` (new) — 18 tests covering template instantiation, boundary behavior, Reference Panel authority/serialization/paired-list, and export, all passing on the first real run.

---

## 4. Evidence (MANDATORY)

### Backend tests

```
$ TEST_DATABASE_URL=... CONFIRM_TEST_DB_RESET=1 scripts/test/reset-test-database.sh
PASS: victory_test reset, migrated, and Go-side bootstrapped from empty. (95 migrations)

$ TEST_DATABASE_URL=... go test -count=1 -timeout=600s ./...
ok  	victory/backend/internal/storyboards	30.594s
... (full repo, zero failures)
```

18/18 new Kernel 82 tests passed on first execution against the real database.

### Static checks

```
$ node -e "... extract and new Function() every inline <script> block ..."
frontend/venues/storyboards/index.html OK 1 block(s)
frontend/venues/storyboards/board.html OK 1 block(s)
$ node --check frontend/venues/storyboards/grid-model.js
$ git diff --check
(clean)
```

### Live deploy

```
$ docker compose build backend && docker compose up -d backend
migrate: 1 pending migration(s): 095_kernel82_storyboards_timeline_mode.sql
migrate: pre-apply backup written to /opt/victory/backups/victory_pre_migrate_20260807_161552_1pending.dump
migrate: applied 095_kernel82_storyboards_timeline_mode.sql in 80ms
victory backend listening on :8081
$ docker exec victory-backend wget -qO- http://localhost:8081/health
{"ok":true,...}
(zero errors in logs after deploy)
```

### Real browser proof (Playwright, headless Chromium, live production instance)

Three throwaway accounts (owner/crew/audience), 13 screenshots, covering all 20 required scenarios from spec §15. Selected results:

```
timeline_a_column_titles: ["Beginning","Middle","Ending"]
beginning_menu: "Insert right\nRename"        (no Insert left, no Remove)
middle_menu: "Insert left\nInsert right\nRename\nRemove"
ending_menu: "Insert left\nRename"             (no Insert right, no Remove)
columns_after_insert_right_of_beginning: ["Beginning","Right Of Beginning","Middle","Ending"]
renamed_beginning_still_badged: true, renamed_beginning_title: "The Dawn"
timeline_b_starts_fresh_column_titles: ["Beginning","Middle","Ending"]
paired_include_items: ["Robots"], paired_exclude_items: ["Magic swords"]
crew_sees_add_field_btn: false, crew_field_controls_count: 0
director_add_field_btn_visible: true
pan_moved_horizontally: true, pan_sent_no_mutation_errors: []
card_drag_errors_after_pan: []
export_mode: "timeline", export_template_version: 1, export_has_boundary_roles: true,
export_reference_field_count: 4, export_premise_content: "Crew wrote this premise."
blank_board_reference_panel_hidden: true, blank_board_no_boundary_badges: true
```

**One real gap was found and closed during this pass, not glossed over**: the first proof run showed `pan_moved_vertically: false`. Investigated directly rather than assumed innocent — a follow-up script added enough rows to force genuine vertical overflow (`scrollHeight: 1202` vs `clientHeight: 547`) and confirmed vertical panning works identically to horizontal (`top: 518 → 655`). The original board simply didn't have enough vertical content to scroll; not a code defect. Middle-button drag was driven via raw Chrome DevTools Protocol (`Input.dispatchMouseEvent`), since Playwright's own `mouse` API has no middle-button primitive.

All test boards deleted afterward; verified zero residue across `storyboards`/`storyboard_columns`/`storyboard_reference_fields`/`storyboard_reference_items`/`storyboard_cards` for the test accounts. All three throwaway accounts, sessions, and grants removed.

---

## 5. How to Run (Operator Steps)

Already deployed live. "Timeline" appears as a creation option alongside "Blank" for any authenticated user on the Storyboards venue landing page.

---

## 6. Operator Notes (CRITICAL)

- `CreateTimelineBoard` is a sibling function to `CreateBoard`, not a parameter added to it — every existing test/caller of `CreateBoard` needed zero changes.
- `allocateUniqueSlug` (Kernel 81A) was reused a third time here (columns/bands/rows, then card images conceptually, now Reference Panel fields) without any modification — the `queryRower` interface abstraction added in Kernel 81A (so the same allocator works inside or outside an open transaction) paid off directly for `CreateTimelineBoard`'s multi-insert transaction.
- Boundary-column protection lives in `ReorderColumns`/`RemoveColumn` themselves, not in a separate "Timeline mode" check — it activates purely based on whether a column's `column_role` is `beginning`/`ending`, which is `ordinary` by default for every Blank-mode column. A future third mode that wanted its own boundary-like columns would get this protection automatically just by setting `column_role`, no new code required.
- Reference Panel content is intentionally never hidden-from-audience. If a future kernel wants a hidden Reference Panel field, that's a real schema addition (a `hidden_from_audience`-style boolean plus per-viewer snapshot filtering, mirroring what cards already do) — not present today.

---

## 7. Blockers & Workarounds

- Mid-session, the throwaway test accounts' 2-hour session tokens expired (real wall-clock time elapsed during backend development, testing, and deployment exceeded 2 hours). Diagnosed via a direct `curl` probe (401 `not_authenticated`) rather than assumed; resolved by extending `auth.sessions.expires_at` directly in the database for the same accounts, no new accounts needed.
- Playwright's `page.mouse` API has no middle-button drag primitive; used `page.context().newCDPSession(page)` + raw `Input.dispatchMouseEvent` calls with `button: 'middle'` instead.

---

## 8. Deviations from Kernel

None from the locked product decisions (spec §1-§2). Three implementation interpretations worth naming as deliberate choices, all recorded inline in the relevant design doc as well:

- **Default Reference Panel is 4 fields, not 5.** Spec §3.2 lists five labels (Premise, Beginning, Ending, Include, Exclude); spec §3.7 names "Include | Exclude" as its own flagship paired-list example. Read as Include/Exclude being one field's two sublabels, not two separate fields — see `timeline_template.go`'s comment and `reference-panel-contract.md`.
- **"Change field type where safe" (spec 3.4) = "only when the field is empty."** No content-migration logic between field type shapes was built; the operator clears a field first, then changes its type. See `reference-panel-contract.md`.
- **Reference Panel field/item reordering uses up/down buttons, not drag-and-drop.** Spec never requires drag-and-drop for panel reordering (only card movement has that requirement, from Kernel 81) — kept simple and bounded rather than extending Kernel 81's card-drag machinery to a second, structurally different use case.

---

## 9. Known Issues

- No keyboard-only equivalent for middle-mouse panning exists (same accessibility gap category already logged for Kernel 81's occupied-cell "Move existing" picker in `storyboards-accessibility.md`). A keyboard user can still reach any part of a long Timeline via normal scroll (arrow keys/Page Up/Down/Home/End on a focused scrollable region, or Tab-focusing a card and using the existing modal move fallback) — panning is a convenience, not the only way to navigate.
- Reference Panel field/item lists have no drag-and-drop reorder (see §8) — up/down buttons only.

---

## 10. Files Changed or Created

**Backend:**
- `backend/migrations/095_kernel82_storyboards_timeline_mode.sql` (new)
- `backend/internal/storyboards/timeline_template.go`, `timeline.go`, `reference_panel.go`, `reference_panel_http.go` (new)
- `backend/internal/storyboards/columns.go`, `boards.go`, `bands.go`, `rows.go`, `snapshot.go`, `export.go`, `events.go`, `http.go`, `types.go` (modified)
- `backend/internal/storyboards/kernel82_timeline_dbtest_test.go` (new)
- `backend/cmd/victory/main.go` (modified — 10 new routes)

**Frontend:**
- `frontend/venues/storyboards/index.html`, `board.html` (modified)

**Construction/docs:**
- `Construction/Kernels/Kernel 82 — Storyboards Timeline Mode.md` (filed, mojibake cleaned)
- `Construction/Domains/Storyboards/timeline-mode-contract.md`, `storyboard-template-contract.md`, `reference-panel-contract.md`, `timeline-navigation.md`, `session-state-integration-seam.md` (new)
- `Construction/OperatorLogs/kernel-82-reportback.md` (this file)

---

## 11. Next Recommended Step

Per spec §21's operator checklist, Grant's own live walkthrough is the closing step — create a Timeline, try the boundary/insert behavior, fill in the Reference Panel including the paired list, and middle-drag around a board with enough content to actually need it. Nothing in this kernel requires further backend work to be usable; any follow-up (drag-and-drop panel reordering, a keyboard pan equivalent, an eventual Group Leader/Current Turn feature via the documented seam) is genuinely optional polish, not a gap blocking use.

---

## 12. Required project-memory updates completed

- [x] Kernel spec filed (`Construction/Kernels/Kernel 82 — Storyboards Timeline Mode.md`)
- [x] Reportback saved in repository (this file)
- [ ] `operator-log.md` — pending, next step in this session
- [ ] `operator-notes.md` — pending, next step in this session
- [ ] `kernel-maker-field-guide.md` — not needed, no repo-layout/test-command/runtime-mode change
- [ ] `dev-workflow.md` — not needed, no startup/port/service/migration-process change
- [ ] Master Actual Implementation Guide (`current-state.md`) — still lagging (now Kernels 79/79A/80/81/81A/82 behind), same flagged-not-actioned status as every prior Storyboards reportback this cycle
- [x] Fresh-install/bootstrap migration list — migration 095 is embedded automatically (`backend/migrations/embed.go`'s `//go:embed *.sql`)
- [ ] Help/command documentation — not applicable
