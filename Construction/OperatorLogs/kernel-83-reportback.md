# Kernel Report Back — Kernel 83: Venue Leadership & Turn State

**Kernel spec:** `Construction/Kernels/Kernel 83 — Venue Leadership & Turn State.md`
**Date:** 2026-08-08

## 1. Status

**PASS.** Group Leader and Current Turn now exist as generic, ephemeral, per-venue-session coordination signals (`backend/internal/venuecoordination`), with Storyboards wired as the first consumer for its whole venue family (Blank and Timeline both). A live Storyboards session's Group Leader defaults to the owner only when the owner is literally the user who starts the session; Current Turn always begins unset. Right-clicking a Presence Tray chip — a UI that did not exist in Storyboards before this kernel — exposes **Make Group Leader**/**Give Turn** only to authorized actors (Director+/owner/Operator, the current Group Leader, or, for turn, the current Current Turn holder), server-verified independent of what the menu shows. State updates broadcast live to every connected watcher of that board; disconnecting never auto-reassigns or auto-advances anything; ending the session (last watcher leaves) clears both fields; a later session starts completely fresh. Neither state ever touches `storyboard_grants` or any role/tier. Timeline's Kernel 82 Reference Panel seam now renders a small read-only "Group Leader: X / Current Turn: Y" slot when a session is active, and Timeline JSON export was verified — at both the Go test level and via a live browser fetch — to contain none of it.

Baseline: clean tree at session start (Kernel 82 already merged/deployed per prior session). Everything from this kernel is uncommitted for Grant's review, per house practice.

---

## 2. Pass-criterion ledger (spec §15)

| Criterion | Status | Evidence |
|---|---|---|
| Collaborative venues can opt into generic leadership/turn state | PASS | `venuecoordination.Registry` — venue-agnostic, no Storyboards/role imports; contract in `collaborative-venue-coordination-contract.md` |
| Storyboards wired as an initial consumer | PASS | `backend/internal/storyboards/coordination.go`, `coordination_http.go` |
| Group Leader / Current Turn exist per active session | PASS | `TestCoordinationInitLeaderDefaultsToOwnerWhenFirstWatcher` et al. |
| Owner initializes as Group Leader when applicable | PASS | Same test; browser proof scenario 2 |
| Current Turn starts unset | PASS | `TestCoordinationInitLeaderDefaultsToOwnerWhenFirstWatcher`; browser scenario 3 |
| Director+ may assign Group Leader | PASS | `TestCoordinationDirectorAssignsLeader`; browser scenario 5/6 |
| Current Group Leader may pass leadership | PASS | `TestCoordinationLeaderPassesLeadershipWithoutBeingDirector`; browser scenario 9 |
| Director+ may assign Current Turn | PASS | `TestCoordinationCurrentTurnAuthority`; browser scenario 11/12 |
| Group Leader may assign Current Turn | PASS | Same test |
| Current Turn holder may pass turn | PASS | Same test; browser scenario 13 |
| Turn handoff is explicit only; no participant order/rotation | PASS | `TestCoordinationDisconnectDoesNotAutoAdvanceTurn` |
| Disconnect does not auto-reassign either state | PASS | `TestCoordinationLeaderDisconnectLeavesAssignmentIntact`, `TestCoordinationCurrentTurnHolderDisconnectLeavesAssignmentIntact`; browser scenario 18/19 |
| Session end clears both states; later sessions start fresh | PASS | `TestCoordinationSessionEndClearsStateAndLaterSessionStartsFresh`; browser scenario 20/21 |
| Presence Tray right-click provides authorized actions | PASS | `presence-tray-coordination-actions.md`; browser scenario 5/9/11/13 |
| Presence Tray visibly indicates both states | PASS | Browser scenario 4/15/16, screenshots |
| Presence Tray ordering is unaffected | PASS | Server sorts by handle (never by coordination fields); browser scenario 17 |
| Live changes synchronize to connected clients | PASS | `TestCoordinationChangeBroadcastsLiveAndIsIdempotent`; browser scenario 7 |
| Server enforces authority (not just hidden menu items) | PASS | `TestCoordinationUnauthorizedTiersCannotSeizeLeader`; browser scenario 10b/14b (forged API calls from unauthorized accounts) |
| Neither state changes Victory permissions | PASS | `TestCoordinationAssignmentDoesNotTouchGrantsOrPermissions`; browser scenario 22/23 |
| Timeline consumes live state through the Kernel 82 seam, no persistence | PASS | `buildReferenceCoordinationSlot`; browser scenario 24, screenshot `03-reference-panel-seam.png` |
| Timeline export excludes live coordination state | PASS | `TestCoordinationStateExcludedFromExport`; browser scenario 25 (live fetch of the real export endpoint) |
| No unrelated venue regression | PASS | Full Go suite green (§4) |
| Playwright proof passes | PASS | §4, all 25 required scenarios plus setup checks — 38/38 assertions |

---

## 3. What Was Built

### Backend

- `backend/internal/venuecoordination/registry.go` (new) — the generic, venue-agnostic in-memory `Registry`: `EnsureSession`/`EndSession`/`Get`/`SetGroupLeader`/`SetCurrentTurn`. No DB, no imports beyond the standard library. `registry_test.go` (new) — 6 pure-Go unit tests, no `TEST_DATABASE_URL` needed.
- `backend/internal/storyboards/coordination.go` (new) — Storyboards' authority-checked wrapper: `StoryboardVenueSessionID` (= `board_id`), session start/end hooks, `AssignGroupLeader`/`AssignCurrentTurn` (authority + target-membership checks, then delegates to `Registry`), `BuildPresenceRoster`/`BuildCoordinationView` (wire shaping, including resolved identity for absent holders).
- `backend/internal/storyboards/coordination_http.go` (new) — `HandleCoordinationGroupLeader`/`HandleCoordinationCurrentTurn` HTTP handlers; `attachLiveCoordination` (shared enrichment used by both the WS snapshot and the REST GET); `broadcastCoordinationChanged`/`broadcastPresenceChanged`.
- `backend/internal/storyboards/ws.go` (modified) — session-lifecycle detection: 0→1 distinct-watcher transition starts a session (leader = owner only if the owner is the one connecting first), 1→0 ends it. Board-switch and disconnect both correctly leave/notify the old board. Disconnect cleanup uses its own fresh context rather than the pump's already-short-lived one (see `venue-session-state-lifecycle.md`).
- `backend/internal/network/hub.go` (modified) — added `BroadcastBoardWatchers` (unfiltered broadcast to every current watcher of a board, mirroring the existing `BroadcastProfileWatchers`).
- `backend/internal/storyboards/snapshot.go` (modified) — `BoardSnapshot` gained `ViewerUserID`, `Presence`, `Coordination` — all attached by callers after `ProjectBoardSnapshot` returns, never part of the persisted-content projection itself, and never touched by `export.go`.
- `backend/internal/storyboards/events.go` (modified) — two new event type constants, `storyboard/coordination_changed` and `storyboard/presence_changed`.
- `backend/internal/storyboards/http.go` (modified) — `HandleBoardItem`'s GET path now calls `attachLiveCoordination`.
- `backend/cmd/victory/main.go` (modified) — constructs one `venuecoordination.NewRegistry()`, threads it through the Storyboards HTTP/WS wiring, registers the two new coordination routes.
- `backend/internal/storyboards/ws_dbtest_test.go` (modified) — updated 4 existing call sites for `ServeStoryboardWS`'s new `reg` parameter; no behavioral change.

### Frontend

- `frontend/venues/storyboards/board.html` — new Presence Tray (`#presence-tray`, one `.presence-chip` per live watcher, green presence dot, Leader/Turn badges), right-click context menu (`#presence-context-menu`) with server-authority-informed client-side item filtering, and the Reference Panel's read-only coordination slot (`buildReferenceCoordinationSlot`). No new script files — this venue's existing "any `storyboard/*` event → reload snapshot" WS handling needed zero changes to pick up the two new event types live.

### Tests

- `backend/internal/venuecoordination/registry_test.go` (new) — 6 tests.
- `backend/internal/storyboards/kernel83_coordination_dbtest_test.go` (new) — 16 tests covering spec §11.1–§11.8 plus the export-exclusion proof, all against real dialed WebSocket connections and real HTTP handlers (not mocked), all passing on first full run.

---

## 4. Evidence (MANDATORY)

### Baseline

```
$ git status --short
(clean at session start)
$ git rev-parse HEAD
07859c8873fe870c64c7cafa62dc1ff4bd7de060
$ git branch --show-current
main
```

### Backend tests

```
$ TEST_DATABASE_URL=... CONFIRM_TEST_DB_RESET=1 scripts/test/reset-test-database.sh
PASS: victory_test reset, migrated, and Go-side bootstrapped from empty.

$ TEST_DATABASE_URL=... GOCACHE=/tmp/victory-gocache go test -count=1 -timeout=600s ./...
ok  	victory/backend/internal/storyboards	36.351s
ok  	victory/backend/internal/venuecoordination	0.006s
ok  	victory/backend/internal/network	4.397s
... (full repo, zero failures on a freshly reset test database)
```

22/22 new Kernel 83 tests (6 registry unit tests + 16 storyboards dbtests, including the export-exclusion regression) passed on first full execution against a real database and real dialed WebSocket connections.

One transient failure (`internal/ewrite`'s `TestEnsureCanonicalSocioManuscriptSeedsAndIsIdempotent`) was seen on an un-reset, long-lived `victory_test` database and diagnosed as pre-existing accumulated test-DB state (revision number climbing 2→3→4 across repeated whole-suite runs over many prior sessions), unrelated to any file this kernel touched (`ewrite` was never edited). Confirmed clean on a full reset — see above.

### Static checks

```
$ tmp=/tmp/storyboards-board-inline.js
$ sed -n '/<script>/,/<\/script>/p' frontend/venues/storyboards/board.html | sed '1d;$d' > "$tmp"
$ node --check "$tmp"
(clean)
$ git diff --check
(clean)
$ gofmt -l backend/internal/venuecoordination backend/internal/storyboards backend/cmd/victory/main.go
(clean)
```

### Live deploy

```
$ docker compose build backend && docker compose up -d backend
(build succeeded, 95 migrations unchanged — this kernel adds no schema)
$ docker logs --tail 60 victory-backend
2026/08/08 04:27:22 migrate: schema current (95 migrations recorded)
2026/08/08 04:27:23 victory backend listening on :8081
2026/08/08 04:27:23 discord gateway link active=true ...
$ docker exec victory-backend wget -qO- http://localhost:8081/health
{"ok":true,"service":"victory-backend","time":"2026-08-08T04:27:29.129578052Z"}
```

### Real browser proof (Playwright, headless Chromium, live production instance)

Five disposable accounts (owner/director/crew/cast/audience) driven as five simultaneous browser contexts against `https://victory.amurray.family`, a real Timeline-mode Storyboards board, all 25 required spec §12 scenarios plus 13 setup/plumbing assertions — **38/38 passed**:

```
scenario1_coordination_active = true
scenario2_leader_is_owner = true
scenario3_turn_unset = true
scenario4_presence_tray_marks_leader = true
scenario5_director_plus_sees_make_leader = true
scenario6_director_assigns_new_leader = true
scenario7_live_update_no_reload = true
scenario8_previous_leader_unmarked = true
scenario9_leader_sees_make_leader_without_director_tier = true
scenario10a_unauthorized_no_menu = true
scenario10b_unauthorized_api_rejected = true
scenario11_director_sees_give_turn = true
scenario12_turn_given_by_director = true
scenario13a_turn_holder_sees_give_turn = true
scenario13b_turn_holder_passes_turn = true
scenario14a_unrelated_participant_no_give_turn_item = true
scenario14b_unrelated_participant_api_rejected = true
scenario15_presence_tray_marks_turn = true
scenario16_same_participant_holds_both = true
scenario17_tray_order_unchanged = true
scenario18a_turn_assignment_survives_disconnect = true
scenario18b_presence_truthfully_shows_absent = true
scenario19_leader_assignment_survives_disconnect = true
scenario22_group_leader_no_director_controls = true
scenario23a_group_leader_no_structural_controls = true
scenario23b_role_badge_unchanged = true
scenario24_timeline_seam_displays_values = true
scenario25_export_reachable = true
scenario25_export_excludes_coordination = true
scenario20_session_ended_then = true
scenario21a_fresh_session_no_restored_leader = true
scenario21b_fresh_session_no_restored_turn = true
```

Screenshots in `Construction/OperatorLogs/evidence/kernel-83/`:
- `01-owner-leader.png` — owner's chip badged Leader immediately on session start.
- `02-audience-leader-and-turn.png` — the audience account visibly holding **both** Leader and Turn badges simultaneously, plus the Reference Panel's live "Group Leader: k83audience / Current Turn: k83audience" slot, on a real Timeline board.
- `03-reference-panel-seam.png` — the wired Kernel 82 seam.

Scenarios 10b and 14b specifically forged a direct API POST from an unauthorized account's own session cookie (not just checking the menu was hidden) and confirmed a real `403`, matching spec §12's "No PASS based only on API tests" by using it as a *supplement* to, not a substitute for, the browser-driven proof.

All five accounts, the board, and all their sessions/grants were deleted afterward; verified zero residue (`SELECT count(*) FROM users WHERE handle LIKE 'k83%'` → 0, same for the board and fixture sessions).

---

## 5. How to Run (Operator Steps)

Already deployed live. Open any Storyboard (Blank or Timeline) with at least one other participant — the Presence Tray appears automatically above the toolbar once the WebSocket connects. Right-click any participant's chip to see available actions.

---

## 6. Operator Notes (CRITICAL)

- **Password signup is now closed in production** (`password_signup_closed` — "Victory accounts are created by signing in with Discord."). This is a change from the state documented in earlier kernel notes (Kernel 61/62's Two-Browser technique doc, which assumed `POST /api/auth/signup` worked). The Kernel 83 browser proof used the technique doc's *second* documented method instead — a direct `users` row plus a matching `auth.sessions` row (raw token + `SHA256(raw token string)` as `token_hash`, exactly matching `sessions.HashToken`) — inserted via `docker exec victory-postgres psql`, for all five disposable accounts. Worth updating `kernel-maker-field-guide.md`'s Two-Browser technique section to lead with this method now that signup is closed.
- **Storyboards had zero presence concept before this kernel** — Kernel 80's own `ws.go` comment said so explicitly ("Storyboards has no location/session/presence concept"). The Presence Tray built here is genuinely new UI, not a reskin of something pre-existing; it reuses `network.Hub.BoardWatcherUserIDs`, a primitive that already existed for a different purpose (per-viewer card-event fan-out).
- **"Venue session" for Storyboards = watcher-count transitions on a board**, not any CRUD lifecycle. A board can exist archived for months with zero watchers; a coordination session only exists while ≥1 distinct user is actively watching it via `watch_board`. Full rationale in `Construction/Venues/venue-session-state-lifecycle.md` — read this before wiring a second venue, since a venue with its own real session concept (The Cave) should use that instead of reinventing watcher-counting.
- **The pump's WS context is only good for 5 seconds** (`ServeStoryboardWS`'s `context.WithTimeout(r.Context(), 5*time.Second)`, pre-existing since Kernel 80, reused for the whole connection's lifetime). This kernel's disconnect-cleanup path deliberately does *not* reuse that context — it builds its own fresh one — because a disconnect routinely happens well past 5 seconds and every DB-touching cleanup step would otherwise silently fail. The underlying dormant issue in the pump's own mid-loop queries (only reachable if a client sends a second `watch_board` more than 5s into a connection) was left alone as out of scope; flagging it here since it's now directly adjacent to code this kernel added.
- **`venuecoordination.Registry` is a second, independent in-memory map from `network.PresenceRegistry`** — they look similar (both `map[string]something` keyed by a session-like string) but serve different purposes and were kept deliberately separate: `PresenceRegistry` is Cave-specific connection/persona presence; `venuecoordination.Registry` is the new generic Group Leader/Current Turn store any venue can use. Do not try to merge them.
- **No migration was needed.** This is the first Storyboards kernel since Kernel 80 with zero new tables/columns — everything is in-memory and cleared on backend restart by design (spec §1.5).

---

## 7. Blockers & Workarounds

- Password signup being closed (see §6) blocked the originally-planned disposable-account creation method; worked around via direct fixture-row insertion, per the field guide's own documented fallback.
- The shared package-level `wsConnectionLimiter` (`network/ws.go`, burst 10/refill 20 per minute, keyed by client IP) is tuned for abuse detection, not for a test suite opening 20+ real WebSocket connections from the same loopback address in a few seconds. Worked around in `kernel83_coordination_dbtest_test.go` by giving each simulated test user its own `X-Forwarded-For` value (`dialWSFrom`), which the limiter already keys on — this doesn't weaken the limiter for real traffic, it just gives the test suite the same "many distinct real IPs" shape the limiter is actually meant to allow.

---

## 8. Deviations from Kernel

None from the locked product decisions (spec §1–§2). Implementation interpretations worth naming as deliberate choices:

- **Storyboards opted in for its whole venue family (Blank + Timeline), not just Timeline.** Spec §6.2 explicitly permits this ("Blank Storyboards may also receive the same venue-session capability if Storyboards is treated as one collaborative venue family"). `board.html` is shared between both modes, so this added no extra surface and avoided a mode-conditional branch that would only exist to withhold the feature from Blank boards for no product reason.
- **No dedicated `GET .../coordination` read route was added.** `coordination` and `presence` are already embedded in the existing board snapshot (both the REST `GET /api/storyboards/{board_id}` and the WS `snapshot` message), which every client already fetches/receives. Spec §8 explicitly says the exact route names/shapes are not mandatory ("Do not treat these exact route names as mandatory if Victory already has a better venue-session API pattern"); adding a redundant second read of state the client already has would have been extra surface with no product benefit.
- **Absence is shown two ways, deliberately different in scope.** The Presence Tray simply omits a chip for a disconnected holder (they're not in the "who's currently connected" roster at all — itself a truthful absence signal). The Reference Panel's coordination slot additionally shows an explicit "(absent)" note using the server's `*_present` booleans, since that slot is meant to answer "who's in charge" even when the Presence Tray's live roster has scrolled that person out of view. Both read from the same server-computed truth; neither invents client-side state.

---

## 9. Known Issues

- **Presence Tray right-click has no keyboard entry point** (spec §10.3 explicitly scoped this out of Kernel 83 itself). Logged in `Construction/Storyboards/storyboards-accessibility.md` alongside the two pre-existing gaps from Kernel 81/82.
- **The pump's WS context reuse issue** described in §6 is pre-existing (Kernel 80) and unfixed — flagged, not addressed, since it's outside this kernel's bounded scope and doesn't manifest on the paths Kernel 83 added (which use their own fresh context).

---

## 10. Files Changed or Created

**Backend:**
- `backend/internal/venuecoordination/registry.go`, `registry_test.go` (new)
- `backend/internal/storyboards/coordination.go`, `coordination_http.go`, `kernel83_coordination_dbtest_test.go` (new)
- `backend/internal/storyboards/ws.go`, `events.go`, `http.go`, `snapshot.go`, `ws_dbtest_test.go` (modified)
- `backend/internal/network/hub.go` (modified — `BroadcastBoardWatchers`)
- `backend/cmd/victory/main.go` (modified — registry construction, 2 new routes)

**Frontend:**
- `frontend/venues/storyboards/board.html` (modified — Presence Tray, context menu, Reference Panel coordination slot)

**Construction/docs:**
- `Construction/Kernels/Kernel 83 — Venue Leadership & Turn State.md` (filed, mojibake cleaned)
- `Construction/Venues/collaborative-venue-coordination-contract.md`, `presence-tray-coordination-actions.md`, `venue-session-state-lifecycle.md` (new)
- `Construction/Storyboards/session-state-integration-seam.md` (updated — marked wired)
- `Construction/Storyboards/storyboards-accessibility.md` (updated — Kernel 83 gap logged)
- `Construction/OperatorLogs/kernel-83-reportback.md` (this file)
- `Construction/OperatorLogs/evidence/kernel-83/*.png` (3 screenshots)

---

## 11. Next Recommended Step

Per spec §19's operator checklist, Grant's own live walkthrough is the closing step: open a Storyboard with a second account, confirm the Presence Tray, right-click to assign leadership/turn, and confirm the Reference Panel slot on a Timeline board. Nothing here requires further backend work to be usable. Genuinely optional follow-ups: a keyboard entry point for the context menu (§9), and updating `kernel-maker-field-guide.md`'s Two-Browser technique section to lead with fixture-row account creation now that password signup is closed (§6).

---

## 12. Required project-memory updates completed

- [x] Kernel spec filed (`Construction/Kernels/Kernel 83 — Venue Leadership & Turn State.md`)
- [x] Reportback saved in repository (this file)
- [x] `operator-log.md` — Kernel 83 entry appended
- [x] `operator-notes.md` — 3 durable lessons appended (generic/specific split, session-primitive discovery, stale-technique-doc)
- [ ] `kernel-maker-field-guide.md` — recommended (password-signup-closed note, §6/§7) but not yet applied
- [ ] `dev-workflow.md` — not needed, no startup/port/service/migration-process change
- [ ] Master Actual Implementation Guide (`current-state.md`) — still lagging, same flagged-not-actioned status as every prior Storyboards reportback this cycle
- [x] Fresh-install/bootstrap migration list — not applicable, this kernel adds no migration
- [ ] Help/command documentation — not applicable
