# Storyboards Live Events (Kernel 80)

Implementation: `backend/internal/storyboards/events.go` + `ws.go`, plus
board-watch primitives added to `backend/internal/network/hub.go`.

## Transport: a lightweight socket, not the venue-session socket

`ServeStoryboardWS` is modeled on `network.ServeProfileWS`
(`profile_ws.go`), not `network.ServeVenueWS` (`ws.go`): Storyboards has
no location/session/presence concept, many independent boards, and a
user may watch two boards in two tabs — none of which fits the venue
socket's session-scoped design. It lives in `storyboards/ws.go`, not
`network/`, because its one inbound message (`watch_board`) must be
authorized with `CanViewBoard` before the hub starts delivering events,
and `network` must never import a feature package. Three small exported
wrappers in `network/lightweight_ws_support.go` (`Upgrade`, `WritePump`,
`AllowNewConnection`, `AllowMessage`) expose the package's existing
upgrader/write-pump/rate-limiters to this external caller without
changing behavior for the venue or profile sockets.

## `Hub` extension

- `Client.WatchingBoardID` — one board watched per connection at a time,
  same "smallest reusable delivery mechanism" idiom as
  `WatchingProfileID` (Kernel 61A). A client wanting a different board
  sends a new `watch_board` to re-point the same connection.
- `SetClientWatchBoard`, `BoardWatcherUserIDs` (distinct user IDs
  currently watching a board — multi-tab safe), `SendToBoardWatcher`
  (deliver to one user's connections watching one board) — the same
  shape as the existing `BroadcastToSessionUser`, keyed on
  `(board, user)` instead of `(session, user)`.

The `Hub` itself does no role filtering — a standing invariant of that
package (`network.Hub.Broadcast` sends raw bytes to whoever's connected;
the caller must have already shaped the payload per recipient). All
shaping for Storyboards lives in `storyboards/events.go`.

## Per-viewer fan-out (the hidden-card leak boundary)

`emitBoardEvent(ctx, pool, hub, boardID, eventType, build func(tier
string) any)` is the single fan-out primitive: it resolves every current
watcher of `boardID`, calls `build` once per watcher with that watcher's
own server-resolved tier, and delivers the result only to that watcher.
**`build` returning `nil` means "send this watcher nothing at all"** —
not a redacted stub, not a content-free "something changed" ping — since
even that could leak a hidden card's existence via a count delta (spec
1.11, "moving a hidden card does not broadcast content to viewers").

- `broadcastGenericBoardEvent` — same payload to every tier. Used for
  board metadata, columns, bands, and rows, none of which are ever
  hidden-from-audience content in Kernel 80.
- `broadcastCardEvent` — the one category that can differ per viewer:
  if `card.HiddenFromAudience` and the watcher's tier can't see hidden
  cards, `build` returns `nil`.
- `broadcastCellReorder` — a subtler case: the `ordered_card_ids` list
  itself would leak a hidden card's ID and position to Audience/Cast even
  though the event never carries card content, so Audience/Cast watchers
  receive the same list with any hidden card IDs filtered out (positions
  closing up around the gap) rather than the raw list.
- `broadcastGrantChanged` — delivered to the **owner's own connections
  only**. Grant content is a sharing-management detail; the affected
  user simply finds their own authority different on their next action
  or reconnect, rather than receiving a live event about their own
  grant.

`TestWSCardEventDeliveredLiveAndHiddenFiltered` proves this end-to-end
over a real WebSocket connection: a hidden card's creation is not
delivered to an Audience-tier watcher, while a subsequent non-hidden
card's creation is.

## Event catalogue

All server-authored only — a client-sent copy of any of these types is
rejected, matching the existing `character/projection_updated` precedent
in `network/ws.go`.

`storyboard/metadata_changed`, `storyboard/grant_changed`,
`storyboard/column_added|updated|reordered|removed`,
`storyboard/band_added|updated|reordered|removed|lock_changed`,
`storyboard/row_added|updated|reordered|moved|removed`,
`storyboard/card_added|updated|moved|reordered|removed|lock_changed`,
`storyboard/archived`.

Envelope: `{"type": "...", "board_id": "...", "data": {...}}`.

## Reconnect contract

`watch_board` — on first connect or any reconnect — always yields exactly
one `{"type":"snapshot","data": <ProjectBoardSnapshot>}`. No replay, no
buffered backlog, matching `ws.go`'s existing venue-connect contract.
`TestWSSnapshotOnWatchAndReconnect` proves a fresh connection watching
the same board after a close gets a new snapshot, not stale state.

## Kernel 81 addition: `storyboard/card_swapped`

`EventCardSwapped` broadcasts twice per swap (once per card, each still
per-viewer-filtered through `broadcastCardEvent`'s existing
hidden-from-audience gate) rather than introducing a combined
two-card-payload event type. `board.html` treats it exactly like every
other `storyboard/*` event — a signal to re-fetch the full snapshot, not
something it patches incrementally — so no new client-side event-handling
code was needed. Card image attach/replace/remove reuses the existing
`storyboard/card_updated` event unchanged (an image reference is just
another card field, from the event model's point of view).

## Recorded scope boundary: no live-kick on revocation

Grant revocation stops the next mutation attempt (HTTP 403) and the next
`watch_board`/reconnect (`TestWSRevokedGrantRejectsNextWatch`) — an
already-open watch is not proactively evicted mid-session. This matches
the repo's existing posture that live session revocation is a
ticker-based concern (`Hub.RevalidateSessions`), not a per-feature
instant-kick, and satisfies the spec's literal requirement ("stops
mutation authority immediately," which per-request HTTP checks already
guarantee) without building new eviction machinery for this kernel.
