package network

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
	"victory/backend/internal/actions"
	"victory/backend/internal/world"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func ServeCaveWS(hub *Hub, pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("ws upgrade failed: %v", err)
			return
		}

		client := &Client{
			Conn: conn,
			Send: make(chan []byte, 16),
		}
		hub.Add(client)

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)

		sessionCookie := ""
		if c, err := r.Cookie("victory_session"); err == nil {
			sessionCookie = c.Value
		}

		userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
		if err != nil {
			cancel()
			log.Printf("ws current user failed: %v", err)
			_ = conn.WriteJSON(map[string]any{
				"type":  "error",
				"error": "not_authenticated",
			})
			_ = conn.Close()
			hub.Remove(client)
			return
		}

		viewerRole, err := lookupVenueRole(ctx, pool, userID, "the-cave")
		if err != nil {
			cancel()
			log.Printf("WS DEBUG role lookup failed user=%s err=%v", userID, err)
			_ = conn.WriteJSON(map[string]any{
				"type":  "error",
				"error": "viewer_role_lookup_failed",
			})
			_ = conn.Close()
			hub.Remove(client)
			return
		}

		log.Printf("WS DEBUG connected user=%s role=%s", userID, viewerRole)

		snapshot, err := world.LoadCaveSnapshot(ctx, pool, viewerRole)
		cancel()
		if err != nil {
			log.Printf("ws snapshot failed: %v", err)
			_ = conn.WriteJSON(map[string]any{
				"type":  "error",
				"error": "failed_to_load_world",
			})
			_ = conn.Close()
			hub.Remove(client)
			return
		}

		if err := conn.WriteJSON(map[string]any{
			"type": "snapshot",
			"data": snapshot,
		}); err != nil {
			log.Printf("ws initial write failed: %v", err)
			_ = conn.Close()
			hub.Remove(client)
			return
		}

		go writePump(client)
		readPump(hub, pool, client)
	}
}

func writePump(c *Client) {
	defer c.Conn.Close()

	for msg := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func readPump(hub *Hub, pool *pgxpool.Pool, c *Client) {
	defer func() {
		hub.Remove(c)
		_ = c.Conn.Close()
	}()

	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			return
		}

		var payload map[string]any
		if err := json.Unmarshal(msg, &payload); err != nil {
			continue
		}

		switch payload["type"] {
		case "ping":
			_ = c.Conn.WriteJSON(map[string]any{
				"type": "pong",
				"ts":   time.Now().UTC().Format(time.RFC3339),
			})

		case "react/emote":
			sessionID, _ := payload["session_id"].(string)
			actorID, _ := payload["actor_id"].(string)
			kind, _ := payload["kind"].(string)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			stored, err := actions.StoreReaction(ctx, pool, actions.ReactRequest{
				SessionID: sessionID,
				ActorID:   actorID,
				Kind:      kind,
			})
			cancel()

			if err != nil {
				_ = c.Conn.WriteJSON(map[string]any{
					"type":  "error",
					"error": err.Error(),
				})
				continue
			}

			out, _ := json.Marshal(map[string]any{
				"type": "action",
				"data": stored,
			})
			hub.Broadcast(out)

		case "perform/speak":
			sessionID, _ := payload["session_id"].(string)
			actorID, _ := payload["actor_id"].(string)
			text, _ := payload["text"].(string)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			stored, err := actions.StoreSpeak(ctx, pool, sessionID, actorID, text)
			cancel()

			if err != nil {
				_ = c.Conn.WriteJSON(map[string]any{
					"type":  "error",
					"error": err.Error(),
				})
				continue
			}

			out, _ := json.Marshal(map[string]any{
				"type": "action",
				"data": stored,
			})
			hub.Broadcast(out)
		}
	}
}

func lookupVenueRole(ctx context.Context, pool *pgxpool.Pool, userID string, venueSlug string) (string, error) {
	var role string

	err := pool.QueryRow(ctx, `
		SELECT lower(role_text) FROM (
			-- exact venue membership first
			SELECT m.role::text AS role_text, 1 AS priority
			FROM memberships m
			JOIN venues v ON v.id = m.venue_id
			WHERE m.user_id = $1
			  AND v.slug = $2

			UNION ALL

			-- fallback: global producer membership
			SELECT m.role::text AS role_text, 2 AS priority
			FROM memberships m
			WHERE m.user_id = $1
			  AND m.venue_id IS NULL
			  AND m.role::text = 'producer'
		) ranked
		ORDER BY priority
		LIMIT 1
	`, userID, venueSlug).Scan(&role)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "none", nil
		}
		return "", err
	}

	return role, nil
}
