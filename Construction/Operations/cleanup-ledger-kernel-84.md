# Cleanup Ledger — Kernel 84

Bounded runtime/documentation debt found and either repaired or explicitly deferred during Kernel 84 (Canonical Reconciliation & Runtime Cleanup, 2026-08-08). Per spec §0/§1.1: "Known discrepancies must be recorded rather than hidden" and "prefer repair over refactor."

## Repaired

| Item | What was wrong | Fix | Evidence |
|---|---|---|---|
| Storyboards WS context-lifetime bug | `ServeStoryboardWS` reused a 5-second handshake-scoped context for a connection's entire life; any `watch_board`/board-switch more than 5s into a connection silently failed | Three-tier context scope (setup / connection-lifetime / per-operation), matching `network/ws.go`'s existing correct convention | `Construction/Operations/websocket-context-lifecycle.md`; `TestWSLateBoardSwitchSurvivesOldSetupTimeout` |
| `TestEnsureCanonicalSocioManuscriptSeedsAndIsIdempotent` flake | Long-flagged across Kernels 81A/82/83 as "pre-existing test-DB state accumulation." Actually a test-isolation bug: the test asserted a hardcoded absolute revision number ("must be 2") for an edit it made itself, which only held on the very first run against a freshly reset database — any second invocation in the same session (proven by running the suite twice in a row without a reset) saw the revision the *first* run's own edit left behind and failed | Assert relative to the revision number actually loaded (`beforeRevisionNumber+1`), not a hardcoded constant | Ran the full suite twice consecutively without a reset in between — both green; previously the second run failed every time |
| Stale disposable-account testing guidance | `kernel-maker-field-guide.md` and `dev-workflow.md` both led with `POST /api/auth/signup`, which Kernel 83 discovered is closed in production (`password_signup_closed`) | Rewrote the Two-Browser technique's step 1 to lead with direct fixture-row insertion (the method that actually works now); added a note to `dev-workflow.md` | Both files' diffs; Kernel 84's own browser regression proof used the new method successfully |
| `Security-notes.md`'s stale Brevo reminder | "Remind me in kernel 78+" was satisfied (Kernel 79A) but the file wasn't updated to say so, and a future reader might re-trigger the same reminder indefinitely | Added a one-line status update: still blocked as of Kernel 84's own re-check, no further recheck scheduled on a kernel-number cadence | `Security-notes.md` §3 |
| Canonical Roadmap frozen at Kernel 75 | The single canonical roadmap's baseline, committed horizon, and several tracks (V3/V6/V9/A1/C4/O4) described Kernels 76–84's work as still-future requirements | Full reconciliation — see the roadmap's own 2026-08-08 change-log entry for the itemized list | `Victory_Canonical_Roadmap_v2.md` |
| `current-state.md` frozen at Kernel 78 | Missing eWrite's full scope, all of Storyboards, and Venue Coordination entirely | Header, Current Major Surfaces, API Families, Known Working Flows, Known Gaps, and Next Recommended Direction all updated | `current-state.md` diff |
| Old Master Actual Implementation Guide / Parallel Track Roadmaps read as if still active | No explicit "superseded" marker existed on either file — only implied by the Canonical Roadmap's own skip/defer ledger entry | Added an unmistakable status banner to both files' top | Both files' diffs |
| Accessibility gaps scattered across three kernels' own reportbacks | Kernel 82's middle-mouse-panning gap was named in its reportback's "Known Issues" but never actually added to `storyboards-accessibility.md`; no single canonical list existed | Added a "Canonical backlog" table at the top of `storyboards-accessibility.md`, categorizing each gap as blocker/convenience/general with its workaround | `storyboards-accessibility.md` diff |

## Found, documented, deliberately not fixed this kernel

| Item | Why it's real | Why not fixed here |
|---|---|---|
| Storyboards `AddColumn` has no Timeline boundary awareness | The generic "+ Column (end)" action appends past a Timeline's `Ending` column with no special-casing; only the UI's per-column "Insert left/right" menu items produce a correct result, and only because they perform a second client-computed reorder call after creation. A raw API caller (or this kernel's own first-draft regression script) can place an ordinary column after `Ending`. Discovered live, during Kernel 84's own required §7.3 browser regression pass, not invented | Touches Storyboards' structural sort_order/reorder mechanism, which Kernel 81A's own notes already flag as having a separate, unrelated, unfixed concurrent-creation race — this kernel's guardrail (§21) is a narrow WS/reconciliation code-repair surface, not a second Storyboards structural-integrity kernel. Documented in `current-state.md`'s Known Gaps and the roadmap's V3 section for a future bounded fix: make `AddColumn` insert immediately before any `column_role='ending'` row when one exists |
| Presence Tray context menu has no keyboard entry point | Spec-acknowledged at the time (Kernel 83 §10.3) | Explicitly out of Kernel 84's scope too (§20: "add an accessibility redesign" is a named non-goal) — reconciled into the canonical backlog instead of fixed |
| `sort_order` unique-constraint race under concurrent structural creation (Kernel 81A) | Real, previously found, previously left unfixed with the same reasoning | Same reasoning still applies — unrelated to this kernel's WS/reconciliation scope |
| `internal/ewrite`'s cosmetic raw-Postgres-error leak on malformed `object_id` (Kernel 79 Goal C&E) | Real, previously found, previously left unfixed | Cosmetic, not security; unrelated to this kernel's scope |

## Investigated, confirmed benign (no action needed)

| Item | Finding |
|---|---|
| Migration `088` missing from the sequence | Confirmed harmless — the migration runner (`internal/migrate/migrate.go`) sorts by filename and tracks a checksum ledger, not contiguous numbering. Gap's origin unexplained by any reportback; not invented an explanation |
| Migration `093` not attributed to any specific Kernel 76–83 reportback | Likely an out-of-band hotfix between Kernels 81 and 81A; not attributed further without more evidence |
| Disposable test/fixture residue in the live database | Queried directly: zero stray accounts, boards, or sessions matching any prior kernel's naming convention. Two boards titled "Test"/"Test 2" found and confirmed to belong to Grant's own real account (`straturli`) — real user content, not touched |
| Stray Docker containers / bound ports from old smoke runs | None found |
| Orphaned `location_memberships`/`access_grants` rows | None found |

## Evidence summary

- Full Go test suite: green on a freshly reset `victory_test` database, twice in a row without a reset in between (proving the ewrite fix holds).
- `git diff --check`: clean.
- `gofmt -l`: clean for every file this kernel touched (one pre-existing, untouched-by-this-kernel formatting issue in `internal/ewrite/sections.go` noted but not fixed — out of scope, not this kernel's file).
- Fresh-database bootstrap: 95 migrations, 19 venues, 4 eWrite collections, 1 directory, all from empty.
- Live browser regression (`scripts/smoke/kernel84-regression-browser.js`): 17/17 assertions passed against the real production instance — Blank Storyboard creation/card/drag, Timeline boundary columns/Reference Panel/inward-insert/middle-pan/export, 2-user live coordination (assign/live-update/disconnect-no-reassign), eWrite reader + Skill Directory link resolution, and one forged negative-authority API call rejected with `403`.
- Live deploy: WS context fix deployed to production (`docker compose build backend && up -d backend`), `/health` OK, migration ledger unchanged (95 — no schema change this kernel).
