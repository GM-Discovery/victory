# WebSocket Context Lifecycle (Kernel 84)

## The bug

`backend/internal/storyboards/ws.go`'s `ServeStoryboardWS` (Kernel 80) created one `context.WithTimeout(r.Context(), 5*time.Second)` at handshake time and reused that same context for every subsequent `watch_board` message the connection ever sent, for the connection's entire life:

```go
// BEFORE (Kernel 80–83)
ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
defer cancel()
userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
// ... upgrade ...
readStoryboardPump(ctx, hub, pool, reg, client) // same ctx, whole connection life
```

Once 5 seconds elapsed, `ctx.Done()` fired. Any DB-backed operation `readStoryboardPump` performed after that point — loading a board, checking `CanViewBoard`, projecting a snapshot, a board switch, Kernel 83's coordination-session bookkeeping — silently failed with a context-deadline error. In practice this meant: a client that connects and sends one `watch_board` within the first 5 seconds works fine (the common case, since `board.html` calls `watchStoryboard` immediately on load); a client that switches to a different board, or reconnects to re-watch, more than 5 seconds into the same connection, would fail.

Kernel 83 found this and worked around it only for its own new disconnect-cleanup path (using a fresh `context.Background()`-derived context there specifically). It did not touch the pump's main per-message path, and flagged that explicitly as out of its own scope.

## The fix (Kernel 84)

Three context scopes now exist, matching the convention `internal/network/ws.go`'s `ServeVenueWS` already used correctly (Storyboards' `ws.go` just hadn't followed it):

```go
// 1. Setup-only: bounded strictly to the pre-upgrade handshake/auth check.
setupCtx, setupCancel := context.WithTimeout(r.Context(), storyboardWSOperationTimeout)
userID, err := access.CurrentUserIDFromRequest(setupCtx, pool, sessionCookie)
setupCancel()
// ... never passed into the pump ...

// 2. Connection-lifetime: cancelled (not timed out) when the connection
//    actually ends -- the deferred cancel fires once the blocking pump
//    call below returns, which only happens on disconnect.
connCtx, connCancel := context.WithCancel(context.Background())
defer connCancel()
readStoryboardPump(connCtx, hub, pool, reg, client)

// 3. Per-operation: a fresh, storyboardWSOperationTimeout-bounded child of
//    connCtx, created inside handleWatchBoard for each inbound message.
//    Inherits early cancellation if the connection itself closes
//    mid-operation, but never inherits a stale deadline from handshake time.
func handleWatchBoard(connCtx context.Context, ...) {
    ctx, cancel := context.WithTimeout(connCtx, storyboardWSOperationTimeout)
    defer cancel()
    // ... all DB work for this one watch_board uses ctx ...
}
```

`storyboardWSOperationTimeout` is 5 seconds, matching `network/ws.go`'s own per-message-type budget (that file creates a fresh `context.WithTimeout(context.Background(), 5*time.Second)` inside every one of its ~17 message-type switch cases — the same pattern, just not previously followed here).

Disconnect cleanup (Kernel 83's own fix) is unchanged: it still uses its own independent `context.WithTimeout(context.Background(), storyboardWSOperationTimeout)` rather than a child of `connCtx`, because by the time cleanup runs, the caller's deferred `connCancel()` may already be racing toward firing — cleanup needs a context guaranteed *not* to be mid-cancellation, not one degrees-removed from the connection's own lifecycle.

## Why this shape and not simpler alternatives

- **Not "make the context effectively infinite"** (e.g. `context.Background()` with no timeout at all for per-message work): would remove the query-budget safety net entirely — a hung DB call would then block that connection's read loop forever instead of failing fast after 5 seconds. The spec explicitly ruled this out (§2.2: "Do not solve this by making every context effectively infinite").
- **Not one fresh context per whole connection re-derived from `r.Context()`**: `r.Context()`'s behavior after a WebSocket upgrade (which hijacks the underlying connection) is not reliably tied to the connection's real lifetime once the HTTP server no longer owns the socket — relying on it post-upgrade would be building on undocumented behavior. `context.WithCancel(context.Background())`, explicitly cancelled by this file's own code on disconnect, has no such ambiguity.
- **Not a single shared per-message context reused across the whole switch/loop body via `defer`**: `defer` inside a `for` loop accumulates until the function returns, leaking every prior iteration's cancel func until disconnect. `handleWatchBoard` being its own function (rather than an inline loop body) is what makes `defer cancel()` safe to use per message.

## Adjacent audit (spec §3)

Every other long-lived read pump in the repository was checked for the same anti-pattern:

| File | Pump | Result |
|---|---|---|
| `internal/network/ws.go` (`ServeVenueWS`) | The Cave/Catharsis/First Theater venue socket | **Already correct** — this is the file the fix above now matches. Setup context bounded to handshake; every one of its ~17 message-type switch cases creates its own fresh `context.WithTimeout(context.Background(), 5*time.Second)`. |
| `internal/network/profile_ws.go` (`ServeProfileWS`) | Player Workbook/Trailer Face invalidation socket (Kernel 61A) | **Not applicable** — `readProfilePump`'s loop does zero database work (`hub.SetClientWatchProfile` is a pure in-memory call); there is no context-lifetime hazard to have. |
| `internal/network/discord_chat_bridge.go` (`mirrorVictoryChatToDiscord`) | Not a pump — a per-action handler called with `context.Background()` and its own fresh `context.WithTimeout(ctx, 20*time.Second)` | **Already correct**. |
| `internal/network/session_control.go` (`HandleSessionControl`) | An ordinary per-request HTTP handler, not a long-lived socket | **Not applicable** — a fresh bounded context per request is inherently correct here; there is no multi-message reuse to go stale. |
| `internal/network/discord_gateway.go` | The outbound connection to Discord's own gateway | **Already correct** — no handshake-scoped context is reused across its read loop. |

Presence/watcher cleanup on disconnect was checked at the same time (spec §3's "watcher/presence entries not removed on disconnect" and "session end tied incorrectly to one browser tab" concerns): `ServeVenueWS`'s pump calls `hub.Presence().Disconnect(...)` and `hub.Remove(client)` on every exit path (normal loop exit and each early-return branch), and Storyboards' own disconnect cleanup (`onBoardWatcherLeft`, Kernel 83) already correctly derives session-end from the *last distinct watcher leaving*, not from a single tab closing. No repair was needed in either area.

**Conclusion: Storyboards' `ws.go` was the only offender.** No second same-root-cause fix was required.

## Regression proof

`backend/internal/storyboards/kernel84_ws_context_dbtest_test.go`'s `TestWSLateBoardSwitchSurvivesOldSetupTimeout`: dials a real WebSocket connection, watches board A, sleeps 6 seconds (strictly longer than the old 5-second setup-context timeout), then sends a `watch_board` for board B and asserts a fresh DB-backed snapshot arrives, the old board's coordination session correctly ended, the new board's correctly started with the owner as leader, and disconnect cleanup afterward leaves no leaked watcher/session state. This exercises exactly the failure window the bug occupied (any DB-backed operation more than 5 seconds into the connection); by inspection of the diff, the pre-fix code would have failed this test at the `watch_board` for board B, since `ctx` there was the single handshake-scoped context passed all the way into `readStoryboardPump`. The test was written and run only against the already-fixed code, not separately confirmed red against the old code.
