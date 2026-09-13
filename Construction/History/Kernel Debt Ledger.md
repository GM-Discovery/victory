# Kernel Debt Ledger

**Produced by:** Kernel 99, 2026-09-12, per spec §20. Consolidates every unresolved item any kernel intentionally handed forward, so a future kernel doesn't have to re-derive this from 100 individual reportbacks. Already-fixed debt is not listed here as though still open — check the "current status" column, which reflects state as of tonight, not as of the originating kernel's own report.

| Originating kernel | Description | Current status | Fixed by | Classification |
|---|---|---|---|---|
| 42 (migration baseline cleanup) | Baseline migration `000_kernel42_productions_baseline.sql` sorts before `001_init.sql` — non-chronological ordering | Still true, confirmed harmless (runner sorts by filename + checksum ledger, not chronology) | — | ACCEPTED, not a defect |
| 51 | Multi-client live verification pending | Unknown whether ever separately verified | — | UNKNOWN — check before relying on multi-client First Theater behavior |
| 52 | Live multi-client verification pending for canonical dice | Unknown whether ever separately verified | — | UNKNOWN |
| 53 | Browser-level evidence pending (Character Workbook) | Unknown whether ever separately verified | — | UNKNOWN |
| 60 | Browser-level verification pending (Socio skills/sheet) | Unknown whether ever separately verified | — | UNKNOWN |
| 71 | Phase A committed; rest reportedly uncommitted at report time | Superseded — full participation resolver is live and was directly audited by Kernel 97 | Kernel 97 (confirmed live, correct) | CLOSED |
| 75 | Criteria 25 (art) and 27 (operator walkthrough) outstanding | Still open — these require Grant's own review, not more engineering | — | OPEN, requires Grant |
| 77 | Recovery-email delivery blocked by a Brevo `535 Authentication failed` (account-approval gate, not config) | Still blocked as of Kernel 84's own security notes; not re-checked since | — | OPEN — external dependency, not code |
| 79 Goal C&E | Cosmetic raw-Postgres-error leak on malformed `object_id` | Confirmed still real and much broader than originally scoped — Kernel 98 found ~21 sites across `ws.go`/`indexcard_http.go`/`director_console.go` echoing raw `err.Error()` to clients | Not yet fixed | ROUTED to K101 |
| 81A | `sort_order` race for concurrent structural creation, worked around not fixed | Not re-verified this pass | — | OPEN, low-severity |
| 83 | Presence Tray right-click has no keyboard entry point | Not re-verified this pass | — | OPEN, accessibility debt |
| 84 | Timeline boundary gap in `AddColumn`, documented not fixed | Not re-verified this pass | — | OPEN |
| 86 | Explosion semantics deliberately left unchanged (86B ruled unnecessary) | Confirmed intentional, not debt | — | ACCEPTED |
| 87 | Live-stroke streaming not built; full 50-item browser matrix not exhaustively driven | Not re-verified this pass | — | OPEN, scoped as future cartography work |
| 88 | Player-initiated roll path untested; health bars not click-to-reveal; Story So Far half-wired | Not re-verified this pass | — | OPEN |
| 91 | Scope reduction vs. full spec was a deliberate agreement | Confirmed intentional, not debt | — | ACCEPTED |
| 92 | True-Showtime/End-Showtime paths Go-tested, not browser-scripted | Not re-verified this pass | — | OPEN — low priority, functionally proven at the API layer |
| 93 | Season tickets remain unbuilt (per prior session memory) | Still unbuilt as of tonight | — | OPEN, deferred by product decision |
| 94 | Formal combined Empty/Working/Full acceptance gate (§32/34) never separately invoked, despite positive pass-by-pass live review | Still open — requires Grant's explicit acceptance call, not more engineering | — | OPEN, requires Grant |
| 95 | Entire kernel (Unified Victory Shell, sitewide visual unification) never executed | Still fully unbuilt | — | OPEN — real, named, unbuilt scope |
| 96 | CSP/HSTS headers, release cleanup, git history scrub — explicitly named deferrals | Not re-verified this pass | — | ROUTED, named not hidden |
| 97 | `actions.isOperatorDiceRoller` duplicates `IsOperatorUser` independently (would silently drift if the canonical check ever changes) | Still present | — | OPEN, low priority |
| 97 | `CanCrewPerformNonDestructiveEdit`'s 5 call sites not individually re-verified against their own "must be non-destructive" contract | Still unverified | — | OPEN, maintainability risk not confirmed bug |
| 97 | No general-purpose "view Victory as Role X" tool exists (only narrow Scene-composition preview) | Still true | — | RECORDED as legitimate product gap, out of scope by design |
| 98 | Raw `err.Error()` leak (see 79 Goal C&E above — same finding, confirmed broader) | Open | — | ROUTED to K101 |
| 98 | `CreateShowing` has no duplicate-submission guard | Open | — | ROUTED to K101 |
| 98 | Two console-only silent client failures (`tour-engine.js`, `victory-pixi-stage.js`) | Open | — | ROUTED to K101 |
| 98 | Two stale Scene-capture tests exercise a pre-Kernel-93 model | Open | — | ROUTED to K101 |
| 98 | Dice-roll test's nil-pool mock doesn't cover a second code path it triggers | Open | — | ROUTED to K101 |
| 98 | Storyboards board-content loaders have no pagination (future perf cliff, not urgent) | Open | — | ROUTED to K101 |
| 98 | eWrite skill-directory seed test's fictional entry never appears in read-back — root cause not found | Open | — | ROUTED to K102+ (or targeted follow-up) |
| 98 | Deeper §4/§19 dead-code/duplicate-truth sweep (~19 files flagged by a shallow marker search, not individually triaged) | Open | — | ROUTED to K102+ |
| 100 | No code-signing certificate obtained | Open, requires Grant's own action | — | OPEN, non-engineering |
| 100 | No genuinely-external-network remote-access proof performed | Open, requires Grant's own testing | — | OPEN, non-engineering |
| 100 | No explicit A→B data-integrity upgrade proof performed | Open, requires Grant's own testing | — | OPEN, non-engineering |

---

## Debt this kernel (99) itself is creating

- **Part II (fresh-install proof) and remaining Part VII validation** are being executed as part of this same kernel pass — see the reportback for final disposition, not carried as debt if completed tonight.
- Kernels **14, 17, 19, 20, 26** remain permanently UNKNOWN (see `Historical Uncertainty Report.md`) — not fixable debt, an accepted historical limit.
