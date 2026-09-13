# Targeted Live Delivery Contract (Kernel 86)

## The problem the audit found

`backend/internal/network/hub.go`'s `Hub.Broadcast(msg)` sends to **every connected client on the whole server** — not this session's, not this Show's, every socket the process has open, across every venue and every session. Before Kernel 86, the `roll/dice` WS handler (`network/ws.go`) called exactly this for every roll, regardless of its (effectively-ignored) visibility field: a roll in one Show was pushed to literally any other user connected to a completely unrelated session at that moment. This was invisible in practice mainly because nothing sensitive rode on it yet — Kernel 86 is the first feature where it mattered.

## What already existed to build on

The Hub already had two narrower precedents, both following the same standing invariant stated repeatedly in that file's own comments — **the Hub does no role/visibility filtering itself; the caller decides who's authorized before ever calling in**:

- `BroadcastToSessionUser(sessionID, userID, msg)` (Kernel 73, Program Panel privacy) — filters to one `(session, user)` pair, multi-tab safe.
- `SendToBoardWatcher(boardID, userID, msg)` / `BoardWatcherUserIDs(boardID)` (Kernel 80, Storyboards) — the "enumerate watchers, then decide per-viewer what to send, then call the narrow send" idiom `storyboards/events.go` already uses.

Kernel 86 needed the general form: a caller-resolved **set** of user IDs within one session, not a single user.

## `Hub.SendToUsers(sessionID string, userIDs []string, msg []byte)`

One pass over the client set, filtering on `SessionID` match **and** `UserID` membership in a caller-provided set (built once as a `map[string]struct{}` before the loop, not an O(n·m) nested scan). Empty/all-empty `userIDs` is a documented no-op — a caller resolving zero recipients (e.g. a malformed Private roll with no actor) must never fall through to sending anyone anything. Proven by `hub_test.go`'s `TestSendToUsersTargetsExactSetWithinSession` (multi-tab, cross-session, and unrelated-third-party isolation in one test) and `TestSendToUsersEmptySetIsANoOp`.

## `Show` mode: `BroadcastSession`, not global `Broadcast`

The default/legacy `show` audience mode doesn't enumerate a recipient set at all — `rollaudience.LiveRecipients` returns `useSessionBroadcast=true` for it, and the caller uses the pre-existing `Hub.BroadcastSession(sessionID, msg)` instead. This is itself the fix for the audit's global-leak finding: even the *unrestricted* default case is now scoped to "everyone on this session," not "everyone on the server." Every roll, regardless of mode, is strictly narrower after Kernel 86 than it was before.

## `deliverStageMessage` — the single choke point

`network/ws.go`'s `deliverStageMessage(ctx, hub, pool, sessionID, actorID, audienceMode, cohortID, msg)` is the one function that turns a resolved `rollaudience.Decision`-shaped input into an actual send: it calls `rollaudience.LiveRecipients`, then either `hub.BroadcastSession` or `hub.SendToUsers`. It's used for all three live message families a roll can produce — the canonical `"action"` push, the `"stage_effect"` push, and the `"stage_effect_pinned"`/`"stage_effect_dismissed"` pushes — so there is exactly one place that decides "who gets this," not three independently-maintained copies. `resolvedStoredMode` (a deliberately different function from `rollaudience.NormalizeMode`, with its own doc comment explaining why) defaults an absent/malformed stored `audienceMode` to `show` here, matching the identical fail-open default `world.LoadVenueSnapshot`'s read-side filter uses for the same class of legacy/malformed data.

## Guardrail respected: no broadcast-site refactor

Per spec §4's explicit guardrail, this kernel did not touch any of the codebase's other `hub.Broadcast`/`BroadcastSession`/`BroadcastBoardWatchers` call sites (chat, index cards, tokens, presence, Storyboards, etc.) — only the `roll/dice` case and the two new `stage_effect/*` cases route through `deliverStageMessage`. Whether any of those other call sites have their own version of the same global-fan-out issue is unaudited debt, not resolved by this kernel, and not claimed to be.
