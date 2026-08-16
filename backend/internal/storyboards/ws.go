package storyboards

// ServeStoryboardWS is a deliberately lightweight websocket endpoint,
// modeled on network.ServeProfileWS rather than network.ServeVenueWS:
// Storyboards has no location/session/presence concept, many independent
// boards, and a user may watch two boards in two tabs -- none of which
// fits the venue socket's session-scoped design. It lives in this package
// (not network) because the one inbound message it accepts, watch_board,
// must be authorized with CanViewBoard before the hub starts delivering
// events for that board, and network must never import a feature package.
//
// Kernel 83 layers a live venue-session concept on top of exactly this
// watch_board mechanism, without contradicting the paragraph above: a
// Storyboards "collaborative venue session" is defined as the period
// during which at least one distinct user is watching a given board (see
// coordination.go's StoryboardVenueSessionID and
// Construction/Venues/venue-session-state-lifecycle.md for the full
// rationale). The 0-watcher -> 1-watcher and 1-watcher -> 0-watcher
// transitions detected in this file are the only session start/end
// triggers that exist for this venue.
//
// Kernel 84 fixed a context-lifetime bug Kernel 83 flagged but only
// worked around for its own disconnect path: the pre-upgrade handshake
// context used to be reused for every watch_board message for the whole
// connection's life, so any DB-backed operation more than 5 seconds into
// a connection (a board switch, a late reconnect-style re-watch) silently
// failed with a expired-context error. This file now matches the
// convention network/ws.go's ServeVenueWS already established elsewhere
// in this package family: a short setup-only context for the pre-upgrade
// handshake, a cancel-only (no timer) context for the connection's
// lifetime, and a fresh 5-second bounded context per inbound message,
// derived from the connection context so it's cancelled early if the
// connection itself closes mid-query. See
// Construction/Operations/websocket-context-lifecycle.md.

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
	"victory/backend/internal/network"
	"victory/backend/internal/ratelimit"
	"victory/backend/internal/venuecoordination"
)

// storyboardWSOperationTimeout bounds each individual database-backed
// operation the read pump performs, matching the per-message-type budget
// network/ws.go's ServeVenueWS already uses for the same reason: a
// long-lived connection's later messages get the same query budget as its
// first, rather than inheriting however much time happens to be left on a
// context created once at handshake time.
const storyboardWSOperationTimeout = 5 * time.Second

func ServeStoryboardWS(hub *network.Hub, pool *pgxpool.Pool, reg *venuecoordination.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !network.AllowNewConnection(ratelimit.ClientIP(r)) {
			http.Error(w, "rate_limited", http.StatusTooManyRequests)
			return
		}

		sessionCookie := ""
		if c, err := r.Cookie("victory_session"); err == nil {
			sessionCookie = c.Value
		}

		// Setup-only context: bounded strictly to the pre-upgrade
		// handshake/auth check, never passed into the pump.
		setupCtx, setupCancel := context.WithTimeout(r.Context(), storyboardWSOperationTimeout)
		userID, err := access.CurrentUserIDFromRequest(setupCtx, pool, sessionCookie)
		setupCancel()
		if err != nil || strings.TrimSpace(userID) == "" {
			http.Error(w, "not_authenticated", http.StatusUnauthorized)
			return
		}

		conn, err := network.Upgrade(w, r)
		if err != nil {
			log.Printf("storyboard ws upgrade failed: %v", err)
			return
		}

		// Connection-lifetime context: cancelled (not timed out) when this
		// handler returns, which only happens once the blocking pump call
		// below returns -- i.e. exactly when the connection ends. Every
		// discrete DB operation the pump performs derives its own fresh
		// storyboardWSOperationTimeout-bounded context from this one, so
		// it inherits early cancellation on disconnect without inheriting
		// a stale deadline.
		connCtx, connCancel := context.WithCancel(context.Background())
		defer connCancel()

		client := &network.Client{
			Conn:   conn,
			// See the venue socket's buffer comment (network/ws.go): Kernel
			// 88B moved this socket's replies onto the same channel too.
			Send:   make(chan []byte, 64),
			UserID: userID,
		}
		hub.Add(client)

		go network.WritePump(client)
		readStoryboardPump(connCtx, hub, pool, reg, client)
	}
}

// maxStoryboardMessageBytes caps a single inbound frame -- the only
// message this socket ever accepts is a small watch_board envelope.
const maxStoryboardMessageBytes = 4 << 10

func readStoryboardPump(connCtx context.Context, hub *network.Hub, pool *pgxpool.Pool, reg *venuecoordination.Registry, c *network.Client) {
	c.Conn.SetReadLimit(maxStoryboardMessageBytes)
	defer func() {
		// Disconnect can happen arbitrarily long after connCtx was
		// created, and connCtx is about to be cancelled by the caller's
		// deferred connCancel() the moment this function returns -- so
		// cleanup gets its own independent, always-fresh context rather
		// than a child of connCtx (which could already be racing toward
		// cancellation) or the pump's original setup context (long since
		// expired).
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), storyboardWSOperationTimeout)
		defer cleanupCancel()

		leftBoardID := c.WatchingBoardID
		hub.Remove(c)
		_ = c.Conn.Close()

		if leftBoardID == "" {
			return
		}
		onBoardWatcherLeft(cleanupCtx, hub, pool, reg, leftBoardID)
	}()

	for {
		_, raw, err := c.Conn.ReadMessage()
		if err != nil {
			return
		}
		if !network.AllowMessage(c.UserID) {
			continue
		}

		var msg map[string]any
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		msgType, _ := msg["type"].(string)
		if msgType != "watch_board" {
			// No other inbound type is recognized -- in particular, every
			// storyboard/* event type is server-authored only (spec 6);
			// nothing here relays or echoes an inbound message back out to
			// other clients, so a client cannot get one delivered to
			// anyone by sending one itself.
			continue
		}

		boardIDVal, _ := msg["board_id"].(string)
		boardID := strings.TrimSpace(boardIDVal)
		if boardID == "" {
			continue
		}

		handleWatchBoard(connCtx, hub, pool, reg, c, boardID)
	}
}

// handleWatchBoard performs one watch_board request's worth of
// database-backed work under its own fresh, bounded context (a child of
// the connection-lifetime connCtx, so it's also cancelled early if the
// connection itself closes mid-operation) -- never the pump's original
// setup context, which regression-tested to expire well before a
// long-lived connection's later messages (Kernel 84).
func handleWatchBoard(connCtx context.Context, hub *network.Hub, pool *pgxpool.Pool, reg *venuecoordination.Registry, c *network.Client, boardID string) {
	ctx, cancel := context.WithTimeout(connCtx, storyboardWSOperationTimeout)
	defer cancel()

	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		_ = c.SendJSON(map[string]any{"type": "error", "error": "storyboard_not_found"})
		return
	}
	allowed, err := CanViewBoard(ctx, pool, c.UserID, board)
	if err != nil || !allowed {
		// Reject before setting the watch so a client cannot fish for
		// board existence/access by watch-spamming IDs -- the error
		// response is identical whether the board doesn't exist or
		// the caller just isn't authorized for it.
		_ = c.SendJSON(map[string]any{"type": "error", "error": "not_authorized"})
		return
	}

	previousBoardID := c.WatchingBoardID
	isFirstOnNewBoard := len(hub.BoardWatcherUserIDs(boardID)) == 0

	hub.SetClientWatchBoard(c, boardID)

	if previousBoardID != "" && previousBoardID != boardID {
		onBoardWatcherLeft(ctx, hub, pool, reg, previousBoardID)
	}

	if isFirstOnNewBoard {
		startCoordinationSessionIfFirstWatcher(reg, board, c.UserID)
	}

	snap, err := ProjectBoardSnapshot(ctx, pool, c.UserID, board)
	if err != nil {
		_ = c.SendJSON(map[string]any{"type": "error", "error": "snapshot_failed"})
		return
	}
	attachLiveCoordination(ctx, pool, hub, reg, board, snap)
	_ = c.SendJSON(map[string]any{"type": "snapshot", "data": snap})

	if !isFirstOnNewBoard {
		// Someone joined a board that already had watchers -- tell
		// the ones already there so their Presence Tray updates live
		// without waiting on an unrelated event to trigger a reload.
		notifyBoardPresenceChanged(ctx, hub, pool, boardID)
	}
}

// onBoardWatcherLeft is called once a connection is no longer watching
// boardID (switched to a different board, or disconnected entirely). If
// that was the last distinct-user watcher, the live coordination session
// ends (spec 5.4); otherwise the remaining watchers get a fresh presence
// roster so a departure is reflected live.
func onBoardWatcherLeft(ctx context.Context, hub *network.Hub, pool *pgxpool.Pool, reg *venuecoordination.Registry, boardID string) {
	if len(hub.BoardWatcherUserIDs(boardID)) == 0 {
		endCoordinationSessionIfLastWatcherLeft(reg, boardID)
		return
	}
	notifyBoardPresenceChanged(ctx, hub, pool, boardID)
}

func notifyBoardPresenceChanged(ctx context.Context, hub *network.Hub, pool *pgxpool.Pool, boardID string) {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		log.Printf("storyboards: notifyBoardPresenceChanged: load board %s: %v", boardID, err)
		return
	}
	roster, err := BuildPresenceRoster(ctx, pool, hub, board)
	if err != nil {
		log.Printf("storyboards: notifyBoardPresenceChanged: roster for %s: %v", boardID, err)
		return
	}
	broadcastPresenceChanged(hub, boardID, roster)
}
