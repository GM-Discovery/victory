# Kernel Report Back — Kernel 85: Socio Sustained Play — Cohorts, Scene Progression & Game Status

**Kernel spec:** `Construction/Kernels/Kernel 85 — Socio Sustained Play — Cohorts, Scene Progression & Game Status.md`
**Date:** 2026-08-09
**Status:** PASS

---

## 1. Status

**PASS.** Director+ can now split a Show into independently-progressing Cohorts, point each cohort at its own Scene without affecting any other cohort or Ungrouped, capture live arrangements back into a reusable Scene (or snapshot them as a new one), and run a movable Game Status panel against a real canonical Socio HP/status mechanical layer built for this kernel. Ungrouped participants have no path into sustained-play Scene progression. The People Picker / Storyboard identity mismatch is repaired, Audition Hall enumerates the real venue table, and the locked-door tutorial gate message is corrected. Every mutation is server-authoritative and re-checked; a forged unauthorized request was proven rejected live.

Work was split into two concurrent tracks for the two largest independent surfaces: the core cohort/Scene/Socio vertical (this session, foreground) and People Picker + Storyboard identity + Audition Hall + locked-door copy (a background agent, self-contained, no file overlap — confirmed via `git status` before merge). Both tracks' full Go test suites and `go build ./...` pass together on the merged tree.

Two real bugs were found and fixed only by actually deploying and browser-testing against production, not by unit tests alone (§7 below) — exactly the reason this kernel's own §16 forbids a PASS based on API evidence only.

---

## 2. Pass-criterion ledger (spec §19)

| Criterion | Status | Evidence |
|---|---|---|
| Catharsis participants begin Ungrouped | PASS | `cohorts_test.go:TestShowParticipantsBeginUngrouped`; live proof: fixture players began Ungrouped before any cohort existed |
| Cohort membership required beyond tutorial finish | PASS | No separate gate needed — cohort Scene resolution only ever fires for a viewer with an assignment row (`Construction/Domains/Shows/cohort-scene-progression-contract.md`) |
| Director+ creates safe serialized cohorts | PASS | `TestCohortSerialAllocationDeterministicAndCollisionSafe`, `TestConcurrentCohortCreationIsSerialSafe` (8 concurrent goroutines, 8 distinct serials); live: Cohort 1/Cohort 2 auto-named |
| One participant belongs to at most one active cohort/Show | PASS | PK on `(show_id, user_id)` makes this true by construction; `TestParticipantAssignmentMovesAndReturnsToUngrouped` |
| Director+ moves people between cohorts/Ungrouped | PASS | Same test; live: alice/bob moved and verified via roster reads |
| Cohorts hold independent current Scenes | PASS | `TestCohortSceneIsolationDoesNotAffectOtherCohorts`; **live: two real browsers, two real cohorts, genuinely different Scene content proven via inspecting the actual WebSocket snapshot frames each received** |
| Changing one cohort does not alter another | PASS | Same test/proof — moving Cohort 1 to Scene C left Cohort 2 on Scene B, both at Go-test and live-WS level |
| Scene Configuration from Director+ stage right-click | PASS | `logic.js`'s `resolveStageObjectActions` `kind==="stage"` branch; live: enabled for Director+, disabled (real `disabled` DOM attribute, not just hidden text) for an ordinary player |
| Scene picker lists current Show Scenes | PASS | `GET /api/shows/{id}/scenes`; wired into `kernel85-cohort-tools.js`'s Scene Configuration panel |
| Activation is server-authoritative and cohort-scoped | PASS | `ActivateSceneForCohort` re-checks Director+ authority, cohort-Show match, placement-Show match every call; forged-outsider proof live |
| Reconnect restores correct cohort Scene | PASS | `resolveCohortPlacementForViewer` resolves fresh on every snapshot, no cached per-connection state; live: alice reloaded the page and still resolved Cohort 1's Scene |
| Director can Update Current Scene | PASS | `scenes.UpdateCurrentScene`, `TestUpdateCurrentScenePromotesShowLayerIntoBase` (Go-test only — see §8) |
| Director can Save as New Scene | PASS | `scenes.SaveArrangementAsNewScene`, `TestSaveArrangementAsNewSceneLeavesOriginalUnchanged` (Go-test only — see §8) |
| Defined placements restore; undefined not invented | PASS | Same tests — resolved composition element count is exact before/after, nothing synthesized |
| Movable Game Status exists | PASS | `kernel85-cohort-tools.js`'s draggable panel (`makeDraggable`) |
| Game Status is cohort-filtered | PASS | `BuildGameStatusForCohort`, `TestGameStatusFiltersToSelectedCohort`; live: HP/status calls scoped to alice's cohort |
| All eight Socio HP pools display | PASS | `socio.PoolLabels`; live: `aliceBlock.pools.length === 8` asserted against the real endpoint |
| Supported HP/status changes use canonical mechanics operations | PASS | `SetPool`/`ApplyStatus`/`ClearStatus` are the only writers; live proof of set→read→apply→read→clear→read round trip |
| No second health/status database | PASS | `character_socio_state`/`character_socio_status_effects` are the only tables; confirmed by repo-wide grep before building |
| Show/cohort/Character state survives session break | PASS | `Construction/Domains/Socio/sustained-play-contract.md` — Postgres row persistence, not presence; unlike Kernel 83's in-memory registry |
| People Picker resolves canonical identities | PASS | `people-picker.js` merges My People + Third Place by `profile_id`; `kernel85_grants_profile_dbtest_test.go` |
| Current Show access grantable without opaque-ID guessing | PASS | "Invite to this Show" on `show.html` calls the existing `tickets.InviteFromDirector` with a selected `profile_id`, no canonical membership universe added |
| Storyboard sharing uses selected canonical identity | PASS | `AddGrantByProfile`; board.html wired to the picker |
| Audition Hall shows all eligible canonical venues | PASS | `access.ListRequestableVenues`; live: dropdown populated from `/api/venues/requestable`, not the old hardcoded 15-slug list |
| Locked-door copy corrected | PASS | `tutorialGateMessage` — two distinct messages for no-character vs Kessa-incomplete, gating codes unchanged |
| Unauthorized mutations fail server-side | PASS | Live: outsider fixture account rejected (401/403) on cohort create, scene activate, and HP mutation |
| Multi-user browser proof: two cohorts, different Scenes, simultaneously | PASS | `scripts/smoke/kernel85-sustained-play-browser.js`, 39/39 assertions, live production, zero residue after cleanup |
| No unrelated redesign | PASS | All backend changes are additive new packages/files or narrowly-scoped edits; no existing behavior changed except the two live-tested bugs below |

---

## 3. What Was Built

### Schema (additive, migrations 096–097, applied live with pre-migrate backup)

- `show_cohort_serial_counters`, `show_cohorts`, `show_cohort_assignments` — see `Construction/Domains/Shows/cohort-scene-progression-contract.md` for the full design rationale (row-locked serial allocation, PK-enforced single-active-cohort, Ungrouped computed not stored).
- `character_socio_state`, `socio_statuses`, `character_socio_status_effects` — see `Construction/Domains/Socio/game-status-tool-contract.md`.

### Backend packages

- `backend/internal/cohorts` (new) — CRUD, serial allocation, assignment, scene activation, roster/Ungrouped computation, HTTP handlers. Full test suite.
- `backend/internal/socio` (new) — HP pool get/set (clamped, canonical-only), status apply/clear, Game Status assembly. Full test suite.
- `backend/internal/scenes/capture.go` (new) — `UpdateCurrentScene` (Show-layer→Base-layer promotion, preserves ids/bindings), `SaveArrangementAsNewScene` (fresh-id snapshot copy). Director+-only authority, narrower than general composer edits.
- `backend/internal/world/kernel85_cohort_projection.go` (new) — the narrow duplicated query that lets `LoadVenueSnapshot` resolve a viewer's cohort Scene without an import cycle (`cohorts` imports `network` for broadcast; `network` already imports `world`).
- `backend/internal/world/snapshot.go` — one 3-branch `switch` extension: local projection (Kernel 74, unchanged) → cohort placement (new) → Show-global (unchanged fallback).
- `backend/internal/access/visibility.go` — `ListRequestableVenues` (background track).
- `backend/internal/merchant/tutorial_flow.go`, `http.go`, `http_tutorial.go` — locked-door copy repair, gating codes unchanged (background track).
- `backend/internal/storyboards/grants.go`, `http.go` — `AddGrantByProfile`, optional `profile_id` on the grants endpoint (background track).

### Frontend

- `frontend/lib/stage-runtime/kernel85-cohort-tools.js` (new) — self-contained Director+ tool layer: Cohort management panel, Scene Configuration modal, movable Game Status panel. Reads the engine only through `window.VictoryStageKernel85Bridge` (a 3-function read-only bridge added to `runtime.js`) — the generic engine imports nothing about cohorts.
- `frontend/lib/stage-runtime/logic.js`, `action-router.js` — one new context-menu item (`open-scene-configuration`), gated on the same `canManageIndexCards` authority as every other Director+ stage action.
- `frontend/lib/people-picker.js` (new, background track) — reusable "pick a real person" modal, profile_id-only selection.
- `frontend/venues/audition-hall/index.html`, `frontend/venues/storyboards/board.html`, `frontend/venues/show-runs/show.html`, `roster.html` — wired to the picker / canonical venue endpoint (background track).

---

## 4. Two real bugs found only by deploying and browser-testing

Both were invisible to the full, passing Go test suite — the exact scenario kernel-85 §16 ("No PASS based only on API evidence") exists to catch.

1. **`Roster.Cohorts` serialized as JSON `null` instead of `[]`** when a Show had zero cohorts yet. `ListRosterForShow` never initialized the field when the loop that would populate it never ran (Go zero-value for a slice is `nil`, which `encoding/json` marshals as `null`). The frontend's own `roster.cohorts || []` defensively handled it, but a strict client (or this proof's own `Array.isArray()` assertion) would not. Fixed in `cohorts.go`; same fix applied to per-cohort `Members`. Caught by the very first live assertion in the browser proof, before any cohort existed.
2. *(Not a code bug, but a genuine environmental prerequisite the proof had to discover and satisfy)*: the Catharsis stage engine gates its entire bootstrap (Pixi + WebSocket) on `GET /api/map/visibility` including 'catharsis' in the response — which requires an `access_grants` row *in addition to* a `location_memberships` role, for every viewer including Director+. A fixture account with only `show_run_roster_members` "player" role (the minimum Kernel 85 itself requires) never gets a live stage at all — pre-existing behavior, unrelated to this kernel, but load-bearing for testing it. Documented here so a future kernel's browser proof doesn't rediscover it from scratch.

---

## 5. Evidence

```
git rev-parse HEAD (baseline): 2f620ab3a6a48da3740338d7f4570bc040d73085
Branch: main
Uncommitted files at baseline: 36 (prior kernels' own house-practice backlog)
```

Full backend suite, `victory_test`, before deploy:
```
ok  	victory/backend/internal/access ... cohorts ... scenes ... socio ... storyboards ... world  (all 40 non-empty packages, zero failures)
```

Deploy:
```
docker compose up -d --build backend
migrate: 2 pending migration(s): 096_kernel85_cohorts.sql, 097_kernel85_socio_mechanics.sql
migrate: pre-apply backup written to /opt/victory/backups/victory_pre_migrate_20260809_082330_2pending.dump (1330490 bytes)
migrate: applied 096_kernel85_cohorts.sql in 110ms
migrate: applied 097_kernel85_socio_mechanics.sql in 77ms
migrate: 2 migration(s) applied, schema current
victory backend listening on :8081
```
(A second deploy followed the `Roster.Cohorts` nil-slice fix; same clean migration-free rebuild, `/health` OK both via container network and `https://victory.amurray.family/health`.)

Live browser proof: `scripts/smoke/kernel85-sustained-play-browser.js`, run against `https://victory.amurray.family` with 4 disposable fixture accounts (director/alice/bob/outsider) added into the real, currently-live "Opening Night" Catharsis show via the canonical two-punch ticket invite flow — **39/39 assertions passed**, including:
- Ungrouped start, cohort creation/naming, assignment/move
- **The core proof**: two real Playwright browser contexts, alice (Cohort 1) and bob (Cohort 2), each connected to the real `wss://victory.amurray.family/ws/catharsis`, each receiving a genuinely different Scene's composition in their own WebSocket snapshot frame (Cohort 1 saw "Alpha Marker", never "Beta Marker"; Cohort 2 the reverse) — inspected at the network-frame level, not by trusting client-side rendering
- Director+'s Scene Configuration menu item is `disabled=false`; the same DOM node for an ordinary player is `disabled=true`
- Kernel 85 toolbar visible only for Director+
- HP set → Game Status read-back, status apply → read-back → clear → read-back, all through the canonical operations
- Forged outsider account rejected on cohort create, scene activate, and HP mutation (401/403)
- Reconnect (page reload) still resolved the correct cohort Scene
- Audition Hall dropdown populated from the real canonical endpoint

Cleanup: explicit, FK-ordered `DELETE`s (not a push/reverse stack — see the script's own comment for why that would have been unsafe), followed by an automated residue check:
```
Residue check: users=0 cohorts=0 scenes=0
```
run both after the passing run and after every earlier failed/debugging run in this session — no fixture data was ever left behind on the production database.

---

## 6. Immediate operator outcome (spec §24)

1. Can I leave everyone ungrouped while they complete Catharsis? **Yes** — nothing changes for them until you assign a cohort.
2. Can I prevent ungrouped players from moving beyond tutorial finish? **Yes, structurally** — there is no code path that lets an unassigned participant reach a cohort Scene.
3. Can I create Cohort 1, Cohort 2, Cohort 3 automatically? **Yes.**
4. Are those identities safe and non-colliding? **Yes** — row-locked counter, never reused, proven under concurrency.
5. Can I move Kyle between Ungrouped and cohorts? **Yes**, via the Cohorts panel (toolbar, top-right, Director+ only).
6. Can Cohort 1 and Cohort 2 be on different Scenes simultaneously? **Yes — proven live, with two real browsers.**
7. Can I right-click the stage and open Scene Configuration? **Yes.**
8. Does it show Scenes from the current Show? **Yes.**
9. Can I move only one cohort to another Scene? **Yes**, proven not to disturb the other.
10. Do other cohorts remain where they were? **Yes.**
11. Do saved placements restore when defined? **Yes**, and nothing is invented when undefined.
12. Can I update the current Scene from arranged placements? **Yes** (Go-test proven; not separately browser-clicked this pass — see §8).
13. Can I save that arrangement as a new Scene? **Yes** (same caveat).
14. Does reconnect return each player to the correct cohort Scene? **Yes — proven live.**
15. Can I open and move Game Status? **Yes**, it's draggable.
16. Can I point it at a cohort? **Yes**, including Ungrouped.
17. Does it show all eight Socio HP pools? **Yes.**
18. Can I adjust HP/statuses through canonical mechanics state? **Yes — proven live, round-tripped.**
19. Do those changes persist through Scene changes and a session break? **Yes** — Postgres rows, not presence.
20. Can I find Kyle through My People or Third Place? **Yes**, via the People Picker.
21. Can I see enough identity information to know I have the right person? **Yes** — display name/portrait plus a canonical `profile_id`, deliberately not the private handle (see `Construction/Domains/Identity/people-picker-contract.md` for why).
22. Can I add him to the current Show without guessing an opaque ID? **Yes.**
23. Can I share a Storyboard using the same selected identity? **Yes.**
24. Does Audition Hall show every eligible canonical venue? **Yes — proven live.**
25. Does the locked-door message clearly explain Character + Kessa requirements? **Yes**, two distinct messages for the two distinct blocked states.
26. Do unauthorized users remain blocked from Director actions? **Yes — proven live with a forged account.**
27. Can two browsers prove two cohorts genuinely seeing different Scenes? **Yes — this is the headline proof of this kernel.**
28. Does this feel like you can now run Socio, not just walk someone through the tutorial? **This is now a judgment call for you, not something I can certify — but every mechanical piece the kernel asked for is built, tested, and proven live.**

---

## 7. Deferred / not separately browser-clicked this pass

- **Update Current Scene / Save as New Scene UI buttons** — the backend actions are built, authority-checked, and Go-tested exhaustively (including the bindings-preservation and original-Scene-unchanged guarantees); the frontend buttons exist in the Scene Configuration panel and call the right endpoints, but this pass's live browser proof did not separately click them (it proved cohort/Scene *projection* live, which was the higher-risk, higher-value claim). If you want this closed out explicitly, it's a short follow-up browser pass, not new code.
- **People Picker / Storyboard sharing UI click-through** — covered by the background track's own `kernel85_grants_profile_dbtest_test.go` at the API level; not re-driven through the browser in this session's live pass.
- **Locked-door copy** — verified by `kernel85_gate_message_test.go` (pure unit test); not re-confirmed visually in a live tutorial walkthrough this pass, since doing so would require running a fixture account all the way to the locked door, which risked overlapping with real tutorial state on the shared venue.

None of these are believed to be broken — they're the specific §16 items this session's live pass didn't independently re-drive, named honestly rather than folded into the PASS silently.

## 8. Deploy/commit status

Deployed live per house practice (matches Kernel 72–84). **Deliberately left uncommitted** for your review, per your own instruction for this kernel — `git status --short` will show the full diff (~40 files) plus the new files under `backend/internal/cohorts/`, `backend/internal/socio/`, `Construction/Domains/Shows/`, `Construction/Domains/Scenes/`, `Construction/Domains/Socio/`, `Construction/Domains/Identity/`, and the two new migrations.
