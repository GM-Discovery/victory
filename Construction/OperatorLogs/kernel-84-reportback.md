# Kernel Report Back — Kernel 84: Canonical Reconciliation & Runtime Cleanup

**Kernel spec:** `Construction/Kernels/Kernel 84 — Canonical Reconciliation & Runtime Cleanup.md`
**Date:** 2026-08-08

## 1. Status

**PASS.** Both required responsibilities are complete: the known Storyboards WebSocket context-lifetime bug is repaired and regression-tested, and the single Canonical Roadmap — plus `current-state.md`, the old Master Guide, and the accessibility/security operator docs — are reconciled through Kernel 83. A bounded adjacent audit found no second instance of the WS bug anywhere else in the repository. A real, independent test-isolation bug (`TestEnsureCanonicalSocioManuscriptSeedsAndIsIdempotent`, flagged as a mystery flake across three prior kernels' reportbacks) was found and fixed as part of investigating a full-suite failure, rather than dismissed. Kernel 84's own required browser regression pass surfaced one new, real, narrowly-scoped Storyboards bug (a Timeline boundary gap in the generic "add column" action) — documented in detail, deliberately not fixed, per the kernel's own scope guardrail.

Baseline: at session start, `git status --short` showed Kernel 83's own uncommitted work in progress (per house practice). **Mid-session, a commit landed on `main`** (`90754da`, author `Grant A. Murray`, message "Implement Kernel 83 venue coordination...") that captured Kernel 83's full working tree *and* whatever Kernel 84 work existed in the tree at that moment — this was not an action I took (I never ran `git commit` in this session); see §6 for detail. Nothing was lost; all Kernel 84 work continued and is reflected in the current tree regardless of which side of that commit boundary it landed on.

---

## 2. Pass-criterion ledger (spec §17)

| Criterion | Status | Evidence |
|---|---|---|
| WS context-lifetime bug repaired | PASS | `ws.go` three-tier context scope; `Construction/Operations/websocket-context-lifecycle.md` |
| Bounded adjacent same-root-cause issues repaired/split | PASS | §3 audit — no second instance found; `network/ws.go` already correct, `profile_ws.go` has no DB work in its loop |
| Late board-switch/watch works beyond old timeout | PASS | `TestWSLateBoardSwitchSurvivesOldSetupTimeout` (6s sleep, then switch) |
| Disconnect/session cleanup leaves no stale state | PASS | Same test's final assertions; pre-existing Kernel 83 tests still green |
| Active testing docs no longer rely on closed signup | PASS | `kernel-maker-field-guide.md`, `dev-workflow.md` updated |
| Disposable test residue verified clean | PASS | DB queries returned zero stray accounts/boards/sessions; 2 "Test"-titled boards confirmed as Grant's own real content, untouched |
| Clean DB migration/bootstrap succeeds | PASS | `reset-test-database.sh` from empty → 95 migrations → Go bootstrap, twice |
| Full Go suite passes on fresh isolated DB | PASS | Green on a fresh reset; green again on a second consecutive run without reset (proves the ewrite fix, not just database luck) |
| Storyboards/Timeline/coordination browser smoke passes | PASS | `scripts/smoke/kernel84-regression-browser.js`, 17/17, live production |
| At least one negative authority proof | PASS | Same script — forged API call from an ungranted account rejected `403` |
| Accessibility debt reconciled into one backlog | PASS | `storyboards-accessibility.md`'s new canonical table |
| Active security/operator docs match current behavior | PASS | `Security-notes.md` Brevo status updated |
| Post-75 kernel history recovered through 83 | PASS | `kernel-history-reconciliation-through-83.md`, 12 reportbacks read in full |
| Canonical Roadmap baseline updated through 83 | PASS | §5.7 new band; full change-log entry |
| V3/V9/A1/C/O statuses reflect actual implementation | PASS | Each rewritten from "required"/flat lists to Established/Open |
| Obsolete requirements moved to skip/defer | PASS | 9 new ledger entries, categorized |
| Old Master Guide clearly superseded | PASS | Banner added to both `victory-master-actual-implementation-guide-v1.md` and `victory-track-roadmaps-v1.md` |
| current-state documentation brought current | PASS | Header, surfaces, API families, flows, gaps, next-direction all updated |
| Committed horizon → Socio then Cartograph | PASS | §12 replaced; no blocker found; third slot explicitly left decision-gated, not invented |
| No new competing roadmap created | PASS | One file (`Victory_Canonical_Roadmap_v2.md`) remains canonical; both old files clearly marked historical, not deleted |
| No unrelated product redesign | PASS | Code changes scoped to `ws.go`/`ewrite` test fix only; the Timeline boundary bug found was documented, not fixed |

---

## 3. What Was Built / Fixed / Reconciled

### Backend (bounded code repair — spec §21's guardrail)

- `backend/internal/storyboards/ws.go` — the WS context-lifetime fix: `setupCtx` (handshake-only), `connCtx` (cancel-only, connection-lifetime), and a fresh `storyboardWSOperationTimeout`-bounded context per `watch_board` message (`handleWatchBoard`, extracted from the read loop so `defer cancel()` doesn't leak across loop iterations). Matches the pre-existing correct convention in `internal/network/ws.go`.
- `backend/internal/storyboards/kernel84_ws_context_dbtest_test.go` (new) — `TestWSLateBoardSwitchSurvivesOldSetupTimeout`: real dialed WS connection, 6-second hold past the old timeout, board switch, coordination-session correctness, disconnect cleanup.
- `backend/internal/ewrite/seed_dbtest_test.go` — fixed the real root cause of the long-flagged `TestEnsureCanonicalSocioManuscriptSeedsAndIsIdempotent` flake: a hardcoded absolute revision-number assertion that only held on a freshly-reset database. Now asserts relative to the revision actually loaded. Verified by running the full suite twice consecutively without a reset (previously failed every second run; now passes both).

### Documentation reconciliation

- `Construction/roadmaps/Victory_Canonical_Roadmap_v2.md` — new §5.7 baseline band (Kernels 76–84); V3/V6/V9/A1/C4/O4 rewritten Established/Open; new §V11 (Live venue coordination); 9 new skip/defer entries; 1 open decision resolved, 1 added; §12 committed horizon replaced (Kernel 85 — Socio Sustained Play, Kernel 86 — Cartograph-Style Drawing Foundation, third slot explicitly decision-gated); §13 provisional horizon updated; full 2026-08-08 change-log entry; §18/§4.3 stale "next sequence" pointers corrected.
- `Construction/roadmaps/victory-master-actual-implementation-guide-v1.md`, `victory-track-roadmaps-v1.md` — superseded/historical banners added, content preserved.
- `Construction/current-state.md` — header brought to Kernel 84; new Major Surfaces for eWrite, Storyboards, and Venue Coordination; new API route families; 3 new proven end-to-end flows; 2 new Known Gaps (the found-not-fixed Timeline boundary bug, the Presence Tray keyboard gap); Next Recommended Direction repointed to Kernel 85/86.
- `Construction/OperatorLogs/kernel-history-reconciliation-through-83.md` (new) — the full per-kernel evidence ledger (title/date/status/reportback path/capabilities/open findings) for Kernels 76–83, plus a documented discrepancies section (the migration-088 gap, the unattributed migration 093, `operator-log.md`'s missing 75–77A entries).
- `Construction/Operations/websocket-context-lifecycle.md`, `cleanup-ledger-kernel-84.md` (new) — the WS fix's design rationale and the full repaired/deferred/investigated-benign cleanup ledger.
- `Construction/Storyboards/storyboards-accessibility.md` — new canonical backlog table (blocker/convenience/general, workaround column), consolidating gaps previously scattered across three kernels' own reportbacks.
- `Construction/OperatorLogs/Security-notes.md` — Brevo reminder status updated (still blocked, no further recheck scheduled on a kernel-number cadence).
- `Construction/kernel-maker-field-guide.md`, `Construction/workflow/dev-workflow.md` — disposable-account technique rewritten to lead with fixture-row insertion (signup is closed in production).
- `Construction/Kernels/Kernel 84 — Canonical Reconciliation & Runtime Cleanup.md` — filed, mojibake cleaned.

### Investigated, confirmed benign, deliberately not touched

Migration 088's numbering gap (harmless — runner sorts by filename, not contiguity); migration 093's unattributed provenance; a pre-existing gofmt issue in `internal/ewrite/sections.go` (untouched file, not in scope); Kernel 81A's own already-documented `sort_order` race (unrelated to this kernel).

---

## 4. Evidence (MANDATORY)

### Baseline

```
$ git status --short   (at session start)
 M Construction/Storyboards/session-state-integration-seam.md
 M Construction/Storyboards/storyboards-accessibility.md
 M backend/cmd/victory/main.go
 ... (Kernel 83's own uncommitted work)
$ git rev-parse HEAD
07859c8873fe870c64c7cafa62dc1ff4bd7de060
$ git branch --show-current
main
```

### History recovery

12 reportback files read in full (Kernels 76, 77, 77A, 78, 79-Phase1, 79A, 79-GoalF, 79-GoalCE, 80, 81, 82, 83), cross-checked against `operator-log.md`, `Construction/Kernels/*.md`, `backend/migrations/*.sql` (95 files, `000`–`095`, one numbering gap at `088`), and `git log`. Full ledger: `kernel-history-reconciliation-through-83.md`.

### WebSocket repair

```
$ TEST_DATABASE_URL=... go test -run TestWSLateBoardSwitchSurvivesOldSetupTimeout ./internal/storyboards/... -v
--- PASS: TestWSLateBoardSwitchSurvivesOldSetupTimeout (6.35s)
```

Adjacent audit: `internal/network/ws.go` (main venue socket) already uses fresh per-message-type contexts — confirmed the correct pattern this fix now matches; `profile_ws.go`'s pump does no DB work at all (nothing to fix); Discord bridge/gateway already correct.

### Backend tests

```
$ TEST_DATABASE_URL=... CONFIRM_TEST_DB_RESET=1 scripts/test/reset-test-database.sh
PASS: victory_test reset, migrated, and Go-side bootstrapped from empty. (95 migrations)

$ TEST_DATABASE_URL=... GOCACHE=/tmp/victory-gocache go test -count=1 -timeout=600s ./...
ok  	victory/backend/internal/ewrite	14.999s
ok  	victory/backend/internal/storyboards	41.997s
ok  	victory/backend/internal/venuecoordination	0.004s
... (full repo, zero failures)

$ (immediately, same database, no reset) go test -count=1 -timeout=600s ./...
ok  	victory/backend/internal/ewrite	13.630s
... (zero failures — proves the ewrite fix, not database luck)
```

### Fresh install

```
$ docker exec victory-postgres psql ... 'SELECT count(*) FROM schema_migrations'   → 95
$ ... 'SELECT count(*) FROM venues'                                                → 19
$ ... 'SELECT count(*) FROM ewrite_collections'                                    → 4
$ ... 'SELECT count(*) FROM ewrite_directories'                                    → 1
```

### Static checks

```
$ git diff --check          (clean)
$ gofmt -l <touched files>  (clean)
$ go vet ./...              (clean)
$ node --check <inline script>  (clean, unrelated to this kernel's frontend footprint — none this kernel)
```

### Live deploy

```
$ docker compose build backend && docker compose up -d backend
2026/08/08 19:58:31 migrate: schema current (95 migrations recorded)
2026/08/08 19:58:31 victory backend listening on :8081
$ docker exec victory-backend wget -qO- http://localhost:8081/health
{"ok":true,...}
```

### Browser regression (Playwright, live production, `scripts/smoke/kernel84-regression-browser.js`)

17/17 assertions passed:

```
blank_board_opens, blank_card_created_via_ui, blank_card_drag_no_js_errors,
timeline_boundary_columns_present, timeline_ending_still_last_after_filler_columns,
timeline_reference_panel_visible, timeline_insert_inward_column,
timeline_middle_pan_moved_scroll, timeline_export_succeeds,
coord_owner_leader_on_start, coord_assign_and_live_update, coord_give_turn_live_update,
coord_disconnect_no_auto_reassign, ewrite_reader_opens_real_content,
ewrite_skill_directory_has_resolvable_links, ewrite_directory_link_resolves_to_real_section,
negative_auth_protected_action_rejected
```

Two fixture accounts + 3 test boards created and fully cleaned up afterward; verified zero residue (`SELECT count(*) FROM users WHERE handle LIKE 'k84%'` → 0, same for boards and sessions).

**One real gap was found during this proof, investigated rather than worked around silently**: the script's own first draft generated "filler" columns via the raw `POST /columns` endpoint to force horizontal overflow for the middle-pan test, and the resulting board no longer had `Ending` as its last column. Rather than assume the test script was simply wrong, the actual product code (`AddColumn` in `columns.go`) was inspected — confirmed it has no `column_role` awareness at all, and only the UI's client-side "Insert left/right" create-then-reorder dance keeps the boundary correct in normal use. This is now a documented, real, bounded finding for a future kernel (§9 below), not silently patched over in the test script and forgotten.

---

## 5. How to Run (Operator Steps)

Already deployed live. No user-facing change (this kernel is cleanup/reconciliation) except: Storyboards connections that stay open and switch/re-watch boards more than 5 seconds in now work correctly, where they previously failed silently.

---

## 6. Operator Notes (CRITICAL)

- **A commit landed on `main` mid-session that I did not make.** `90754da` ("Implement Kernel 83 venue coordination...", author `Grant A. Murray`) captured the entire working tree at that moment — Kernel 83's finished work *and* whatever Kernel 84 files existed at that instant (a partial, in-progress set). I never ran `git commit` in this session; per my own operating rules I only commit when explicitly asked. Nothing was lost — every file this kernel touches is present and correct in the current tree regardless of which side of that commit boundary it landed on — but the commit message only describes Kernel 83, so `git log` for that commit alone will undersell what it actually contains. Flagging this rather than silently working around it or attempting to rewrite history (which I won't do without being asked).
- **The Timeline `AddColumn` boundary gap (§4 above) is real and worth prioritizing** if Storyboards gets touched again before a dedicated fix — it's a one-function, low-risk repair (make `AddColumn` insert before any `column_role='ending'` row when one exists) but was correctly out of this kernel's own bounded scope.
- **Migration 093's provenance is genuinely unknown** — it exists, is applied, is presumably correct (nothing in the full test suite or fresh-bootstrap proof suggests otherwise), but no reportback among the 12 read claims it. If this surfaces again, check commit history around Kernels 81–81A directly rather than re-deriving from reportbacks a second time.
- **Password signup is closed in production, permanently, by design (Kernel 76's own decision)** — this is now reflected everywhere it matters (field guide, dev-workflow, current-state.md). Do not treat a future "signup fails" observation as a new bug.

---

## 7. Blockers & Workarounds

None that blocked this kernel. The mid-session external commit (§6) was noted, not worked around, since it didn't block any of this kernel's own work — everything just kept accumulating in the working tree as normal.

---

## 8. Deviations from Kernel

None from the locked decisions (§1). One scope judgment call worth naming: the Timeline `AddColumn` boundary bug (found via this kernel's own required §7.3 regression pass) was investigated fully enough to understand and document precisely, but the actual fix was withheld per §1.1's own instruction ("if a repair... risks user-visible behavior... record it and split it into a follow-up") — touching Storyboards' structural sort_order/reorder mechanism, which already has one documented unrelated race (Kernel 81A), was judged to exceed this kernel's narrow WS/reconciliation code-repair guardrail (§21).

---

## 9. Known Issues

- Storyboards `AddColumn` has no Timeline boundary awareness (found this kernel, documented in `current-state.md`, the roadmap's V3 section, and the cleanup ledger — not fixed).
- Presence Tray context menu has no keyboard entry point (Kernel 83, reconciled into the canonical accessibility backlog, not newly introduced or newly fixed).
- Kernel 81A's `sort_order` concurrent-creation race remains unfixed (pre-existing, unrelated to this kernel).
- `internal/ewrite`'s cosmetic raw-Postgres-error leak on malformed `object_id` (Kernel 79 Goal C&E) remains unfixed (pre-existing, unrelated).

---

## 10. Files Changed or Created

**Backend:**
- `backend/internal/storyboards/ws.go` (modified — context-lifetime fix)
- `backend/internal/storyboards/kernel84_ws_context_dbtest_test.go` (new)
- `backend/internal/ewrite/seed_dbtest_test.go` (modified — test-isolation fix)

**Scripts:**
- `scripts/smoke/kernel84-regression-browser.js` (new)

**Construction/docs:**
- `Construction/Kernels/Kernel 84 — Canonical Reconciliation & Runtime Cleanup.md` (filed, mojibake cleaned)
- `Construction/OperatorLogs/kernel-84-reportback.md` (this file)
- `Construction/OperatorLogs/kernel-history-reconciliation-through-83.md` (new)
- `Construction/Operations/websocket-context-lifecycle.md`, `cleanup-ledger-kernel-84.md` (new)
- `Construction/roadmaps/Victory_Canonical_Roadmap_v2.md` (extensively updated — see §3)
- `Construction/roadmaps/victory-master-actual-implementation-guide-v1.md`, `victory-track-roadmaps-v1.md` (superseded banners)
- `Construction/current-state.md` (updated through Kernel 84)
- `Construction/Storyboards/storyboards-accessibility.md` (canonical backlog table)
- `Construction/OperatorLogs/Security-notes.md` (Brevo status)
- `Construction/kernel-maker-field-guide.md`, `Construction/workflow/dev-workflow.md` (disposable-account technique)

---

## 11. Next Recommended Step

Per the reconciled Canonical Roadmap: **Kernel 85 — Socio Sustained Play**, then **Kernel 86 — Cartograph-Style Drawing Foundation**. Neither was fully specified inside this kernel, per its own instruction — each needs its own kernel-maker audit of current state first. Smaller, genuinely optional items surfaced here (not blocking either): the Timeline `AddColumn` boundary fix, a Presence Tray keyboard entry point, and amending or re-splitting the mid-session commit noted in §6 if Grant wants the history cleaner.

---

## 12. Required project-memory updates completed

- [x] Kernel spec filed (`Construction/Kernels/Kernel 84 — Canonical Reconciliation & Runtime Cleanup.md`)
- [x] Reportback saved in repository (this file)
- [x] `operator-log.md` — Kernel 84 entry appended
- [x] `operator-notes.md` — 2 durable lessons appended (flake misdiagnosis, regression-pass value)
- [x] `kernel-maker-field-guide.md` — updated (disposable-account technique)
- [x] `dev-workflow.md` — updated (signup-closed note)
- [x] `Security-notes.md` — updated (Brevo status)
- [x] Canonical Roadmap — fully reconciled through Kernel 83/84
- [x] `current-state.md` — updated through Kernel 84
- [x] Master Actual Implementation Guide — marked superseded
- [x] Fresh-install/bootstrap migration list — verified, no new migration this kernel
- [ ] Help/command documentation — not applicable
