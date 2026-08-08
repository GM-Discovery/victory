# Collaborative Venue Coordination Contract (Kernel 83)

Generic, reusable Group Leader / Current Turn state for any Victory venue where people work or play together live. This document is the contract a future venue reads before opting in — it does not itself specify Storyboards' particular wiring (see `Construction/Storyboards/session-state-integration-seam.md` for that).

## What this is, and is not

Group Leader and Current Turn are **live venue-session coordination signals**, not Victory roles, not permission grants, not durable state, and not a turn-order engine. Holding either state never changes what a user is allowed to read or write — see "Permission independence" below.

## The two layers

The implementation is deliberately split into a generic layer and a venue-specific layer, so no single venue ever owns the logic:

```text
backend/internal/venuecoordination   (generic)
  Registry: in-memory map[venue_session_id]State
  State{ VenueSessionID, VenueType, VenueInstanceID,
         GroupLeaderUserID, CurrentTurnUserID }
  EnsureSession / EndSession / Get / SetGroupLeader / SetCurrentTurn
  -- knows nothing about roles, tiers, grants, or what a "venue" is.

backend/internal/storyboards/coordination.go   (venue-specific, per consumer)
  StoryboardVenueSessionID(boardID) -> venue_session_id
  startCoordinationSessionIfFirstWatcher / endCoordinationSessionIfLastWatcherLeft
  AssignGroupLeader / AssignCurrentTurn -- authority checks, target
    membership checks, then delegates the actual mutation to Registry.
  BuildPresenceRoster / BuildCoordinationView -- wire shaping.
```

`venuecoordination.Registry` never imports a feature package and never resolves a viewer tier, an owner, or a grant. A future venue writes its own thin wrapper file (like `coordination.go`) rather than teaching the generic package about its authority model. This is why Storyboards opting in did not turn Storyboards into "the owner" of Group Leader/Current Turn — the generic Registry would work identically for a second, unrelated venue opting in tomorrow with its own authority rules.

## Opt-in shape

There is no boolean capability flag column anywhere. A venue "opts in" simply by:

1. Defining what its `venue_session_id` means (see `venue-session-state-lifecycle.md`).
2. Calling `Registry.EnsureSession` at whatever moment it defines as session start, `EndSession` at session end.
3. Writing its own authority-checked `AssignX` wrapper functions that call `Registry.SetGroupLeader`/`SetCurrentTurn` only after validating actor authority and target session-membership itself.
4. Wiring a Presence Tray (or reusing an existing one) with the right-click actions from `presence-tray-coordination-actions.md`.

A venue that never calls any of the above simply never has an active coordination session — `Registry.Get` on an ID that was never `EnsureSession`'d returns `(State{}, false)`, which is indistinguishable from "this venue doesn't support the capability." No separate "does this venue support coordination" flag was needed for that reason: non-participation and never-started both read the same way to a caller.

## Storyboards as the initial consumer

Storyboards (`backend/internal/storyboards/coordination.go`) opted in for its whole venue family — both Blank and Timeline-mode boards get live coordination state, not just Timeline. This was a deliberate scope choice (spec §6.2 explicitly permits it: "Blank Storyboards may also receive the same venue-session capability if Storyboards is treated as one collaborative venue family"). `board.html` is the same file for both modes, so this added no extra frontend surface.

## Server authority, always

Every mutation path re-derives everything server-side from the venue's own data:

- actor authentication (existing session cookie);
- actor/target venue-session membership (Storyboards: currently-connected board watchers, via `network.Hub.BoardWatcherUserIDs`);
- actor authority (Storyboards: `resolveViewerTier` + "are you the current Group Leader/Current Turn holder");
- session is active (`Registry.Get` returning `active`).

A client never supplies its own authority claim, and the Presence Tray hiding a menu item is a UX nicety, not the enforcement — `coordination_http.go`'s handlers reject an unauthorized POST with `403 not_authorized` regardless of what the client rendered.

## Permission independence (spec §9.5)

`venuecoordination.Registry` has no code path that touches `storyboard_grants`, `location_memberships`, or any other authority table. This is structural, not merely tested: the package doesn't import `access` or hold a database handle at all. See `backend/internal/storyboards/kernel83_coordination_dbtest_test.go`'s `TestCoordinationAssignmentDoesNotTouchGrantsOrPermissions` for the regression proof.

## Wire shape actually used

```json
POST /api/storyboards/{board_id}/coordination/group-leader
POST /api/storyboards/{board_id}/coordination/current-turn
{"target_user_id": "..."}

-> {"ok": true, "data": {"coordination": {
     "active": true,
     "group_leader_user_id": "...", "group_leader_handle": "...",
     "group_leader_display_name": "...", "group_leader_present": true,
     "current_turn_user_id": "...", "current_turn_handle": "...",
     "current_turn_display_name": "...", "current_turn_present": false
   }}}
```

No separate `GET .../coordination` read route was added — `coordination` and `presence` are already embedded in the board's existing snapshot (`GET /api/storyboards/{board_id}` and the WS `snapshot`/`storyboard/*` reload cycle), so a second read surface would have been a redundant duplicate of state the client already has. `CoordinationView` carries `*_present` booleans and resolved handle/display_name specifically so a client can render "Group Leader: Alice (absent)" for a disconnected holder without a second lookup.

Live sync reuses two new event types, `storyboard/coordination_changed` and `storyboard/presence_changed`, broadcast via `network.Hub.BroadcastBoardWatchers` (added for this kernel, mirroring the existing `BroadcastProfileWatchers`). Both are plain unfiltered broadcasts — unlike card events, neither is ever hidden-from-audience content, so there is no per-viewer shaping step. The frontend's pre-existing "any `storyboard/*` event reloads the snapshot" handling in `board.html` needed no new client code to pick these up live.
