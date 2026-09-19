# Kernel 101 — Live Bug Burn & Update Discipline

**Status:** DRAFT — ready for implementation
**Type:** Live bug-fix kernel (append-only ledger) + update/deploy mechanism refactor
**Sequence position:** After Kernel 100 (Windows Consumer Installer) and Kernel 99 (Archive/Docs); the first kernel executed against a genuinely live install with real players
**Primary proof:** Every bug on the ledger below is fixed, routed, or explicitly deferred with a reason; the update mechanism no longer forces a choice between "leave the show running" and "get the fix installed"
**Core doctrine:** Victory is live now. A bug found today may already be shaping tonight's game. Fix what's real, verify it before it leaves this machine, and never let "shipping the fix" become more disruptive than the bug itself.

---

## 0. Kernel mode

Kernel 101 is not a rewrite kernel. It is not another audit kernel (96–98 already did that broad work). It is not a feature kernel, with one explicitly named exception (§4).

It is a **live bug burn**: a running ledger of real, reported-or-found defects, worked in priority order, verified locally before anything touches production or a Windows release.

This kernel is **append-only by design**. Grant is actively finding bugs while playing on real hardware — new entries get added to §3 as they surface, at any point before or during execution, without needing a new kernel number. Do not close this kernel while known, unaddressed entries remain open; mark them deferred with a reason instead (§6).

---

## 1. Deploy discipline (read this before fixing anything)

Victory is live. This is the operating rule for the rest of this kernel, and for every kernel after it until Grant says otherwise:

- **Do not deploy a fix to murray-vserver, and do not push a branch that triggers a Windows release build, as a reflex the moment a fix is written.** Batch fixes. Grant deploys roughly once a day now, on his own schedule.
- **Verify locally first.** Use a local Linux test environment — not production — to reproduce and confirm a fix before it's considered done. (Grant: confirm exactly what this environment is — this machine's own local dev/podman stack, as used for tonight's Cast-rename testing, or something else you have in mind — before execution starts in earnest.)
- **A deploy is its own decision, not a side effect of a fix being ready.** Ask, or wait to be told, before pushing/deploying — every single time, not just the first time in a session.
- Exception: a fix that is purely local-file/documentation/spec work (like this document) never needed a deploy in the first place — this rule is about anything that touches murray-vserver or triggers `windows-installer.yml`.

---

## 2. Fix versus route

Use the same policy Kernel 98 established:

### Fix now
Confirmed bugs with a bounded, low-risk fix — most of §3's Tier 0–2 entries.

### Route
A finding that's real but broad enough to be its own project (e.g., a full accessibility pass, a full error-message redesign) gets classified and pointed at a future kernel rather than expanding this one indefinitely.

Do not let this kernel become an open-ended rewrite because one bug's real fix touches a lot of surface area.

---

## 3. Bug ledger

Append new entries here as they're found. Each entry needs: a description, evidence/repro if known, and a tier. Do not remove an entry when it's fixed — mark it `FIXED` with the commit, so the ledger stays a true history.

### Tier 0 — Fixed already (pre-kernel or same-session)

| # | Bug | Status | Evidence |
|---|---|---|---|
| 101-01 | Add/Replace Map context-menu unclickable — `openMapEditor()` called the undefined `hideGridEditor()` instead of `hideGridEditorController()`, throwing before the panel could show | **FIXED**, commit `87fc923` | `frontend/lib/stage-runtime/editors.js:553` |

### Tier 1 — Launch blockers (silent failures, no visible error to the user)

| # | Bug | Status | Notes |
|---|---|---|---|
| 101-02 | Fresh installs never go live at Catharsis — `firstrun.BootstrapFirstOperator` creates the Show/Session but never sets the Show's own `status` to `"live"`; stays `"draft"` forever without manual Director intervention | **FIXED**, not yet deployed | `backend/internal/firstrun/firstrun.go` now explicitly sets `status`/`actual_start_at` via `shows.UpdateShow` after `showtime.Start`. New test `firstrun_dbtest_test.go` (package had zero prior coverage) — confirmed it fails without the fix (`got "draft"`), passes with it, in an isolated run. Data on murray-vserver was already patched separately. |
| 101-03 | Audition Hall's Cast-permission request may not actually submit — real confirmation email + in-app note, but zero rows in `show_run_tickets` for the show run in question | DEFERRED — retest after the rest of Tier 1/2 ship, in case it isn't actually reproducible | Grant's call 2026-09-17: hold this one until other fixes are live |
| 101-04 | Windows updater sends "dozens of annoying notifications" for a single available update, instead of one clear, stateful prompt | **FIXED, not yet CI-verified** (branch `windows-native`, commit `9ea09c5`) | Grant chose persistent/dismiss-once. `UpdateChecker.ReportPendingOnce` dedupes the "waiting" balloon to once per version (was re-firing every 15 min once pending); `TrayApplicationContext`'s new "Update Ready: vX.Y.Z" menu item is the persistent half, hidden by default, stays until dismissed via `UpdateChecker.DismissPendingVersion`/`IsPendingVersionDismissed`, self-resets on a genuinely new version. No local C# toolchain/CI available in this session (see §5) — self-reviewed (brace/paren balance, manual read-through against existing patterns), not yet compiled. `packaging/windows/` only exists on `windows-native`, not this branch. |
| 101-05 | No way to force a backend update through while a Show is live — the updater only applies backend updates during a quiet window or after an explicit session-close; Grant needs an override with an explicit warning dialog | **FIXED, not yet CI-verified** (branch `windows-native`, commit `9ea09c5`) | New `UpdateChecker.ForceApplyAsync` (public) skips the live-session block entirely; new tray menu item "Force Update Now...", warns via `MessageBox` only if a Show is actually live (confirmed via the existing `/api/system/live-sessions` check before the warning ever shows, never unconditionally). Same CI-verification caveat as 101-04. |

### Tier 2 — Real, contained

| # | Bug | Status | Notes |
|---|---|---|---|
| 101-06 | "Cast" vs "Player" vocabulary mismatch | **Fixed, unreviewed** — branch `cast-canonical-rename` | Awaiting Grant's review/merge, not yet deployed |
| 101-07 | Raw internal error strings leak to the client (~21 sites in `ws.go`/`indexcard_http.go`/`director_console.go`) | **FIXED**, not yet deployed | New shared `clientSafeError(err)` helper (`indexcard_http.go`) passes through anything already shaped like this codebase's own established safe reason-code convention (lowercase snake_case, matching real examples confirmed in source: `unsupported_visibility_mode`, `cast_requires_ticket`, `not_authorized`) and masks anything else to a generic `internal_error` — closing a confirmed-real leak path (`rollaudience.resolveSessionShowID` returns a bare, unwrapped pgx error on any failure besides `ErrNoRows`). Every call site already logged the real error server-side before this change; that's untouched, only what crosses the wire to the client changed. All 24 sites converted (21 in `ws.go` + 3 in the other two files); 2 of the `ws.go` sites (`announcements`/`rollaudience` fail() calls) previously had *no* server-side log at all — added one, matching this file's own established pattern everywhere else. New test `client_safe_error_test.go` (8 cases). Full `internal/network` suite run clean (3 pre-existing, unrelated nil-pool panics skipped — see 101-12, now confirmed to affect 3 tests, not 1). |
| 101-08 | "Send Fanmail" button oversized in Catharsis — should be as small as the collapsed reaction icon | OPEN | Needs visual/browser inspection, not resolvable from code alone |
| 101-09 | `CreateShowing` has no duplicate-submission guard | OPEN | Double-click/double-POST creates two rows, not corruption |
| 101-10 | Two silent client-side failures (`tour-engine.js`, `victory-pixi-stage.js`) — console-only, no user feedback | OPEN | Low stakes |

### Tier 3 — Deferred / needs triage

| # | Bug | Status | Notes |
|---|---|---|---|
| 101-11 | Two Scene-capture tests exercise a stale pre-K93 model | OPEN | Tests are wrong, product is correct — rewrite the tests |
| 101-12 | Dice-roll test's nil-pool mock is incomplete | OPEN, scope confirmed wider | Found while running the full `internal/network` suite for 101-07: affects 3 tests, not just the 1 originally carried in from K98 — `TestKernel52RollDiceActionBroadcastsCanonicalResult`, `TestRollDiceProjectsStageEffectAfterAction`, `TestStageEffectSelfPinAndDismiss`, all panicking via the identical path (`deliverStageMessage` → `audienceDiceRollsHidden` → `showings.LoadBySession` with a nil pool). Still confirmed test-harness-only (production never passes a nil pool); the other 2 tests exercising the same broadcast path (`TestStageEffectPrivateRollNeverReachesAnotherSocket`, `TestStageEffectAuthorizedPrivateHasNoDirectorException`) pass, so it's specifically about which mock each test wires up, not a systemic gap in the path itself |
| 101-13 | Storyboards board-content loaders have no pagination | DEFERRED | Future perf cliff, no evidence of a real board near that size |
| 101-14 | "WebGL context was lost" seen once alongside 101-01's crash | NEEDS TRIAGE | Watch for recurrence independent of the (now-fixed) crash before treating as its own bug |
| 101-15 | "Book Antiqua" font repeatedly blocked at visibility level 2 (requires 3) | NEEDS TRIAGE | Unclear if an intentional gate or an over-restrictive default |
| 101-16 | Some test in `internal/shows`/`internal/showtime` leaves an active Catharsis session behind, colliding with any later test in the same `go test` batch that also needs Catharsis free | NEEDS TRIAGE | Test-infrastructure gap, not a product bug — found while verifying 101-02's fix; `internal/firstrun`'s own test passes cleanly in isolation |

### Tier 4 — Add new bugs here

*(append below this line as found — description, repro if known, suspected area)*

---

## 4. Named feature work: update-notification and forced-update override

Grant is explicit that this counts as part of Kernel 101 despite being a refactor, not a pure bug fix — both items live in `packaging/windows/VictoryLauncher` (the tray app / `UpdateChecker.cs`).

### 4.1 Update notification (101-04)

**Current behavior:** the tray icon fires a notification repeatedly for the same available update (matching the updater's own 4-hour/15-minute recheck cadence per the K100 ledger), producing "dozens" over time.

**Required behavior — pick one clear mode, do not build both:**
- **Persistent, dismiss-once mode (recommended default):** notify once when an update first becomes available; the notification (or a tray-icon badge state) persists until Grant dismisses it, and once dismissed, it does not reappear for that specific update version.
- **Pop-and-unpop mode:** notify once, briefly, and don't repeat it — Grant accepts he might miss it and will check the tray icon's own state manually if he suspects he missed something.

Either way: **never re-notify for the same update version the user has already dismissed or seen.** Track "last notified version" and "dismissed" state locally (the launcher already has a local data directory under `%LocalAppData%\Victory` — use it, don't introduce a new storage mechanism for one flag).

### 4.2 Forced-update override (101-05)

**Current behavior:** a backend-changing update only applies during the quiet window (2:30 AM local) or after the current Session is explicitly closed (`/showtime <code> end`) — per `UpdateChecker.ApplyPendingUpdateAsync`'s existing quiet-window/session-closed gating.

**Required behavior:**
1. Add an explicit "Update Now" override, reachable from the tray icon regardless of current Show/Session status.
2. If a Show/Session is currently live, selecting it must show a confirmation dialog stating plainly that this will end the live session, before proceeding — a real warning, not a checkbox buried in a settings page.
3. On confirmation: end the session the same way the existing quiet-window path does (stop Postgres/cloudflared cleanly first, per the K100 root-cause fix already in `ApplyPendingUpdateAsync`, so this doesn't reintroduce the original 60-failed-attempts bug), then apply the update and restart, without requiring Grant to separately run `/showtime end` himself first.
4. If no Show/Session is live, the override just applies the update immediately — no different from the existing quiet-window behavior, just on demand.

Do not silently fold this into the existing automatic gating — it must be an explicit, visible, opt-in action every time, given what it costs (an ended live session).

---

## 5. Testing expectations

- Every backend fix gets a real, run test (unit or `dbtest`-backed) where one is meaningful, following the existing test-database conventions (`scripts/test/setup-test-database.sh`/`reset-test-database.sh`, `dbtest.OpenTestPool`).
- Frontend fixes that can't be verified by reading code alone (like 101-08) are marked as needing visual/browser confirmation rather than guessed at blind.
- `packaging/windows/VictoryLauncher` changes (§4) get proven against the windows-installer CI build at minimum; real-hardware verification on Grant's Windows rig happens at his normal deploy cadence, not per-commit.
- Confirm §1's deploy discipline before ending each work session: nothing pushed/deployed that wasn't explicitly asked for.

---

## 6. Fix/route/defer classification

Every ledger entry ends in one of:

- **FIXED** — done, verified, cited with a commit.
- **ROUTED** — real but broad; named as a future kernel's scope.
- **DEFERRED** — real, understood, deliberately not fixed now, with a reason.
- **NEEDS TRIAGE** — not yet confirmed as a real bug worth fixing.

Do not close Kernel 101 with entries still marked plain "OPEN" — every entry needs one of the four dispositions above by the time this kernel reports back.

---

## 7. Human checkpoints

- Grant adds new bugs to §3 Tier 4 at any point — no permission needed, no separate kernel required.
- Before the §4 forced-update override design is finalized, confirm the exact wording/behavior of the confirmation dialog with Grant — this is a real, consequential product-facing decision (a person could end their own live game by mis-clicking), not a routine implementation detail.
- Before any deploy to murray-vserver or push that triggers a Windows release build — confirm per §1, every time.

---

## 8. Decision memory

- Victory is live; deploy cadence is Grant's decision, roughly daily, never automatic.
- This kernel is append-only — new bugs get added without a new kernel number.
- The update-notification and forced-update-override work (§4) is explicitly in scope despite being a refactor, at Grant's direction.
- Testing happens on a local Linux environment before anything reaches production, per §1 (exact environment TBD with Grant).
- Do not expand a contained bug fix into a broader rewrite — route it instead (§2, §6).

---

## 9. Reportback

Report, when this kernel closes (or at any natural pause point, given its append-only nature):

1. Full ledger with final disposition on every entry.
2. What was fixed vs. routed vs. deferred, and why for each deferral.
3. §4's actual implemented behavior for both the notification fix and the forced-update override.
4. Any new bug found during execution that wasn't already on the ledger when this kernel started.
5. Confirmation that deploy discipline (§1) was followed throughout — and if it wasn't, an honest account of when/why not.

Final status: **PASS** (every entry disposed) / **IN PROGRESS** (this kernel's normal resting state, given it's append-only) / **PARTIAL** (a real blocker stopped work, named explicitly).
