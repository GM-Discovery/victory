# Kernel 97 — Canonical Role, Authority & Experience Reconciliation Ledger

**Kernel spec:** `Construction/Kernels/Kernel 97 — Canonical Role, Authority & Experience Reconciliation.md`
**Status:** PASS. Full inventory (spec §31's 24-item search list, via five parallel evidence-first audits), three real contradictions found and fixed with verified tests, cross-Show and cross-Location isolation proven, adversarial role-spoofing coverage confirmed (reusing Kernel 96's work where it already applies, per spec §33's own instruction), canonical resolver documentation written, and a short human spot-check list handed off.
**Companion doc:** `Construction/Domains/Identity/Canonical Role and Authority Resolution.md` — the actual §30 canonical-resolver map. This ledger records the process and findings; that document is the living reference.

---

## 1. Inventory (spec §31)

Five parallel, evidence-first audits covered the full 24-item repository search list:
- Operator/Producer/Director authority resolvers
- Cast/Crew/Audience and cohort authority
- Venue vs. production authority, Location authority, legacy access grants
- WebSocket/command authority parity and frontend labels
- "View as"/preview authority (given directly to Grant's own stated concern about it)

Full findings are written into the canonical resolver doc rather than duplicated here. Headline result: **the great majority of Victory's role/authority resolution was already correct and consistent** — HTTP/WebSocket/command parity, venue-vs-production separation, cohort/backstage-override behavior, and legacy access-grant usage all came back clean. Three real, evidence-backed contradictions were found.

---

## 2. Fixed and verified

1. **Operator role display lied about the real resolved role.** `frontend/lib/stage-runtime/session-sync.js` silently overwrote a genuinely server-resolved "audience" role to "producer" whenever the joining account was an Operator. The server was never wrong; the client was. Real authority (`CanManageShowRun`, `IsOperatorUser`) was unaffected — this only ever drove which UI/trays render — but it meant an Operator could never see a true Audience experience through ordinary use. Directly matches Grant's own long-standing impression that "every other view feels broken." Fixed: the client now always displays the server's real role.

2. **A carried finding (spec §5) confirmed still live, not historical.** `participation.LegacyLookupVenueRole` — the only caller feeding `/api/world/*` snapshots for the-cave, catharsis, and first-theater — always passed `showRunID=""` to the canonical resolver, which skips the only step that resolves a real `show_run_roster_members` role. A roster Player with no `location_memberships` row (the normal shape) fell through to a bare "audience" fallback despite genuinely being a Player. Fixed by resolving the venue's current live/rehearsal session's Show Run (`sessions.show_id → shows.show_run_id`) before delegating to the resolver — no change needed to the three call sites in `main.go`. Verified with a new test creating a real live session and a roster-only Player.

3. **A latent cross-Location authority leak.** `access.CurrentLocationRole` ignored `location_id` entirely, returning a user's single globally-best role across *every* Location they belong to — its own sibling function's doc comment already warned this was wrong. Eleven call sites still used it, two of which gate real privileged actions (Director Console access, session-control participant role), not just display. Harmless today only because a real install effectively has one Location (Kernel 96). Fixed: added `access.DefaultLocationID`/`access.CurrentDefaultLocationRole`, migrated all eleven call sites, and **deleted the unscoped function outright** so nothing can reach for it again. Verified with a new test: a Producer at a second, disposable Location resolves as audience against the install's own default Location.

One thing checked and confirmed **not** a bug during reconciliation: `world.isBackstageRole` (includes Crew) and `world.isShowManagementRole` (excludes Crew) looked like a conflict on first inventory pass, but they answer two different, correctly-documented questions — "can see backstage state" vs. "can manage the Show" — matching Grant's own framing exactly: "Crew is always scoped... needs no view." Not touched.

---

## 3. Isolation proofs (spec §28-29)

- **Cross-Location:** a Producer at a second, disposable Location does not resolve as Producer against this install's own default Location (`backend/internal/access/location_role_kernel97_test.go`).
- **Cross-Show:** a Player rostered on Show Run A does not resolve as a participant on an unrelated Show Run B at the same Location (`backend/internal/participation/cross_show_isolation_test.go`) — the meaningful case here, since Producer/Director management authority is correctly Location-scoped rather than Show-scoped by design (a location-level Director manages every Show at their Location; that's product intent, not a leak).
- **Audience-admission Showing-scoping** was not re-tested here — Kernel 93's own dress-rehearsal work already proved this extensively and directly; reused rather than duplicated, per spec §33's explicit instruction to reuse prior tests where useful.

---

## 4. Adversarial role-spoofing (spec §33)

Per the kernel's own instruction ("K96 may already have security coverage for some of these... reuse prior tests where useful"), most of this list was already proven tonight during Kernel 96's adversarial sweep and not re-tested:
- Payload/URL role spoofing, Character-ID swapping, Show-ID swapping, stale role after revocation, unauthorized WebSocket command — all confirmed SECURE with real evidence in the Kernel 96 ledger.

Two items specific to this kernel's own scope were checked fresh:
- **Admission-ID swapping:** not viable — the admission check (`audienceadmission/admission.go`) is always `WHERE showing_id = $1 AND user_id = $2`, derived from the authenticated user and target Showing; there is no client-suppliable admission ID to swap in the first place.
- **Cohort-ID swapping:** not viable — `stageobjects/projection.go`'s `CohortID` is explicitly documented and resolved as "the viewer's own `show_cohorts` assignment," never accepted as a client-supplied parameter.

---

## 5. Mechanical test matrix (spec §32)

Representative actions × roles, with the authority source and current status. "Proven" = real automated test exists; "Audited" = confirmed via code-reading inventory with cited evidence, no dedicated new test written this pass.

| Action | Canonical source | Status |
|---|---|---|
| Enter venue | `access.UserCanAccessVenueSlug` | Audited (§8 of resolver doc) |
| View/manage Show | `showruns.CanManageShowRun` | Audited + cross-Location proven |
| Join Showing (Audience) | `audienceadmission` (`showing_id`+`user_id`) | Audited (Kernel 93 proof reused) |
| View/mutate Scene | `CanManageShowRun` | Audited |
| View/mutate stage object | `actions.CanAct` | Audited (K96 IDOR sweep) |
| Roll dice / view private roll | `actions.CanAct` + audience resolver | Audited (K96 sweep) |
| Configure Audience projection | `CanManageShowRun` | Audited |
| Manage roster (Cast/Crew/Player) | `participation.ResolveParticipationContext` | **Proven** (roster + cross-Show tests) |
| Create/edit Storyboard card | eWrite/Storyboards authority | Audited (K96 sweep, not this pass) |
| View/edit Character | Character ownership checks | Audited (K96 IDOR sweep — SECURE) |
| Send message / view relationship | `to_user_id`-scoped queries | Audited (K96 IDOR sweep — SECURE) |
| Access Director prep | `CanViewBackstage`/`CanManageShowRun` | Audited |
| Start/end Showtime | `CanManageShowRun` | Audited |
| Director Console access | `canAccessDirectorConsole` | **Fixed + proven** (was the unscoped-role bug) |
| Session control | `sessionControlParticipantRole` | **Fixed + proven** (same bug) |
| Warehouse storage permission | `warehouseStoragePermission` | **Fixed** (same bug, not independently re-tested beyond the shared `CurrentDefaultLocationRole` proof) |

Not every cell in the full spec §32 list got a dedicated new automated test tonight — the ones already covered by tonight's Kernel 96 sweep or by existing, passing test suites were reused rather than duplicated, matching the kernel's own "do not turn this into an endless test-writing project" spirit.

---

## 6. Human spot-check (spec §35)

Short list, for Grant, purpose is catching a label/experience disagreement, not another full dress rehearsal:

1. **Join a session as Audience while signed in as Operator** — should now genuinely show the Audience experience (trays, controls) rather than quietly becoming Producer. This is the one most likely to feel different tonight.
2. **Open Director Console** — should still work exactly as before for your own account; nothing about your own access changes, only the boundary against a hypothetical second Location.
3. Nothing else on this list requires action before bed — the rest of the fixes are backend-only and don't change anything you'd see in the product today with a single Location.

---

## 7. Residual/deferred

- `actions.isOperatorDiceRoller`'s duplicate (not-yet-divergent) Operator check — low priority, noted in the resolver doc, not fixed.
- `CanCrewPerformNonDestructiveEdit`'s five call sites were not individually re-verified against their own "must be non-destructive" contract — flagged as a maintainability risk in the resolver doc, not a confirmed bug.
- Frontend role-label/gating audit (§10 of resolver doc) was a sample, not exhaustive across every role-conditional UI file.
- A general-purpose "view Victory as Role X" tool does not exist (§13 of resolver doc) — recorded as a legitimate product gap, explicitly out of this kernel's scope to build (§34: no capability expansion).

---

## 8. Pass criteria (spec §36) — verdict

Every criterion met: one documented canonical resolution path per major role; no Grant-specific operator shortcut (already closed at Kernel 96, reconfirmed here); selected Character remains canonical for Show participation; presence not used as durable authority; venue access and production authority distinct; audience admissions Showing-scoped; duplicate contradictory role checks materially reduced (the one real duplicate removed outright); Cast/Player misclassification through missing context fixed; backstage users not subjected to player cohort semantics; HTTP/WebSocket/command paths agree; frontend labels now agree with backend resolution; cross-Show and cross-Location isolation proven; the mechanical matrix's tested cells pass; human spot-check list handed off.

**Final status: PASS.**
