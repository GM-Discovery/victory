package network

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
	"victory/backend/internal/ratelimit"
)

// ServeProfileWS is a deliberately lightweight websocket endpoint for Player
// Workbook / Trailer Face invalidations (Kernel 61A §9.1). Unlike
// ServeVenueWS, it requires only an authenticated session -- no active venue
// session, no venue role, no presence -- because viewing or editing a
// Trailer isn't a "joining a session" activity the way Cave/Catharsis are.
// The only inbound message a client may send is `watch_profile`; anything
// else is silently dropped, and nothing is ever relayed from one client to
// another, so a client cannot forge a `player_profile/projection_updated`
// event (Kernel 61A §9.5) -- that type is only ever server-authored, from
// BroadcastPlayerProfileProjectionInvalidation.
func ServeProfileWS(hub *Hub, pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !wsConnectionLimiter.Allow(ratelimit.ClientIP(r)) {
			http.Error(w, "rate_limited", http.StatusTooManyRequests)
			return
		}

		// Kernel 77 K77-08: authenticate before completing the handshake,
		// matching the same fix in ServeVenueWS.
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

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("profile ws upgrade failed: %v", err)
			return
		}

		client := &Client{
			Conn:   conn,
			Send:   make(chan []byte, 16),
			UserID: userID,
		}
		hub.Add(client)

		go writePump(client)
		readProfilePump(hub, client)
	}
}

func readProfilePump(hub *Hub, c *Client) {
	defer func() {
		hub.Remove(c)
		_ = c.Conn.Close()
	}()

	for {
		_, raw, err := c.Conn.ReadMessage()
		if err != nil {
			return
		}

		if !checkMessageRate(c.UserID) {
			continue
		}

		var msg map[string]any
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		msgType, _ := msg["type"].(string)
		if msgType != "watch_profile" {
			// No other inbound type is recognized. In particular, nothing
			// here relays or echoes an inbound message back out to other
			// clients, so a client cannot get a
			// `player_profile/projection_updated` message delivered to
			// anyone by sending one itself.
			continue
		}

		profileID, _ := msg["profile_id"].(string)
		hub.SetClientWatchProfile(c, strings.TrimSpace(profileID))
	}
}
