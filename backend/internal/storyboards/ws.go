package storyboards

// ServeStoryboardWS is a deliberately lightweight websocket endpoint,
// modeled on network.ServeProfileWS rather than network.ServeVenueWS:
// Storyboards has no location/session/presence concept, many independent
// boards, and a user may watch two boards in two tabs -- none of which
// fits the venue socket's session-scoped design. It lives in this package
// (not network) because the one inbound message it accepts, watch_board,
// must be authorized with CanViewBoard before the hub starts delivering
// events for that board, and network must never import a feature package.

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
)

func ServeStoryboardWS(hub *network.Hub, pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !network.AllowNewConnection(ratelimit.ClientIP(r)) {
			http.Error(w, "rate_limited", http.StatusTooManyRequests)
			return
		}

		sessionCookie := ""
		if c, err := r.Cookie("victory_session"); err == nil {
			sessionCookie = c.Value
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
		if err != nil || strings.TrimSpace(userID) == "" {
			http.Error(w, "not_authenticated", http.StatusUnauthorized)
			return
		}

		conn, err := network.Upgrade(w, r)
		if err != nil {
			log.Printf("storyboard ws upgrade failed: %v", err)
			return
		}

		client := &network.Client{
			Conn:   conn,
			Send:   make(chan []byte, 16),
			UserID: userID,
		}
		hub.Add(client)

		go network.WritePump(client)
		readStoryboardPump(ctx, hub, pool, client)
	}
}

// maxStoryboardMessageBytes caps a single inbound frame -- the only
// message this socket ever accepts is a small watch_board envelope.
const maxStoryboardMessageBytes = 4 << 10

func readStoryboardPump(ctx context.Context, hub *network.Hub, pool *pgxpool.Pool, c *network.Client) {
	c.Conn.SetReadLimit(maxStoryboardMessageBytes)
	defer func() {
		hub.Remove(c)
		_ = c.Conn.Close()
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

		board, err := LoadBoard(ctx, pool, boardID)
		if err != nil {
			_ = c.Conn.WriteJSON(map[string]any{"type": "error", "error": "storyboard_not_found"})
			continue
		}
		allowed, err := CanViewBoard(ctx, pool, c.UserID, board)
		if err != nil || !allowed {
			// Reject before setting the watch so a client cannot fish for
			// board existence/access by watch-spamming IDs -- the error
			// response is identical whether the board doesn't exist or
			// the caller just isn't authorized for it.
			_ = c.Conn.WriteJSON(map[string]any{"type": "error", "error": "not_authorized"})
			continue
		}

		hub.SetClientWatchBoard(c, boardID)

		snap, err := ProjectBoardSnapshot(ctx, pool, c.UserID, board)
		if err != nil {
			_ = c.Conn.WriteJSON(map[string]any{"type": "error", "error": "snapshot_failed"})
			continue
		}
		_ = c.Conn.WriteJSON(map[string]any{"type": "snapshot", "data": snap})
	}
}
