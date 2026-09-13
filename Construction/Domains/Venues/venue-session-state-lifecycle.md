# Venue Session-State Lifecycle (Kernel 83)

Spec §5.4 required naming this decision explicitly: "If Victory currently lacks an explicit collaborative session lifecycle, investigate the narrowest existing lifecycle primitive and document the chosen mapping." This document is that mapping, for Storyboards specifically, plus the general shape a future venue should follow.

## The problem

Group Leader / Current Turn are scoped to "a live collaborative venue session." Some Victory venues (The Cave) already have an explicit session concept with real start/end moments (`internal/network/session_control.go`, Kernel 70A's "Start Show Session"). Storyboards does not — Kernel 80's `ws.go` says so directly: boards are watched persistently by any number of users at any time, with no "start/end" lifecycle of their own. A board simply exists, and zero or more users happen to be looking at it via `watch_board` right now.

## The mapping actually used

For Storyboards, **a live coordination session is defined as the period during which at least one distinct user is watching a given board.** Concretely:

```text
session start = the 0-watcher -> 1-watcher transition for a board_id
session end   = the 1-watcher -> 0-watcher transition for the same board_id
venue_session_id = the board_id itself
```

This reuses the *only* existing session-shaped primitive Storyboards actually has: `network.Hub.BoardWatcherUserIDs(boardID)`, which Kernel 80 built for a different purpose (per-viewer event fan-out) but which already answers "who is currently, live, looking at this board" — exactly the question a coordination session needs answered. No new session table, no new session concept, and no reuse of The Cave's unrelated (and semantically different — Production/Show-scoped, not board-scoped) session machinery.

### Where the transition is detected

`backend/internal/storyboards/ws.go`:

- On an inbound `watch_board` message, before calling `hub.SetClientWatchBoard`, the handler checks `len(hub.BoardWatcherUserIDs(boardID)) == 0` to know whether this connection is about to become the *first* watcher. If so, `startCoordinationSessionIfFirstWatcher` runs.
- If the same connection was already watching a *different* board (switching boards in one tab), `onBoardWatcherLeft` runs for the old board immediately after the switch.
- On disconnect (the pump's deferred cleanup), `onBoardWatcherLeft` runs for whatever board that connection was last watching.
- `onBoardWatcherLeft` re-checks `hub.BoardWatcherUserIDs(boardID)`; if it's now empty, `endCoordinationSessionIfLastWatcherLeft` (→ `Registry.EndSession`) runs. Otherwise the remaining watchers get a fresh presence broadcast.

### A subtlety: the pump's own context can't be reused for cleanup

`ServeStoryboardWS` builds its context with `context.WithTimeout(r.Context(), 5*time.Second)` — a pre-existing 5-second timeout meant for the initial auth check, then (as a pre-existing quirk of this file, not something Kernel 83 introduced or fixed) reused for the entire connection's lifetime. A real disconnect routinely happens well past 5 seconds. Kernel 83's cleanup path (`onBoardWatcherLeft` on disconnect) therefore uses its own fresh, short-lived `context.Background()`-derived context rather than the pump's expired one — otherwise every disconnect-triggered "end the session" or "notify remaining watchers" database read would silently fail once the connection had been open more than 5 seconds. This is scoped narrowly to Kernel 83's own new cleanup code; the pre-existing dormant issue with the pump's own mid-loop queries (which only manifests if a client sends a *second* `watch_board` more than 5 seconds into a connection) was left alone as out of this kernel's scope.

## Why not board CRUD lifecycle (create/delete/archive)?

Considered and rejected: a board's create/archive lifecycle is unrelated to who's currently looking at it — a board can exist archived for months with zero watchers, or be actively co-edited by three people for an hour. Session-shaped meaning only exists at the "who's currently connected" layer, which is exactly the watcher-count transition above.

## What a future venue should do

A venue with its own explicit session/production/show concept (The Cave) should use *that* as its `venue_session_id`, not reinvent a watcher-counting scheme — the whole point of `venuecoordination.Registry` being keyed by an opaque `venue_session_id` string is that each venue supplies whatever session identity is natural for it. A venue with no existing session concept (like Storyboards before this kernel) should look for the narrowest real "who's currently live here" primitive it has, the same way this document did, rather than building a new one from scratch or borrowing an unrelated venue's session machinery.

## Ephemeral, never durable

`venuecoordination.Registry` is a plain in-memory `map[string]*State` (see `backend/internal/venuecoordination/registry.go`) — no table, no migration, no persistence layer at all. A backend restart clears every active session's Group Leader/Current Turn along with it, which is consistent with the product rule (spec §1.5): this state was never meant to survive past the live session anyway, and a restart is the platform ending every session that happened to be active at that moment.
