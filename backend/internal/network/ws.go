package network

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"

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
		snapshot, err := world.LoadCaveSnapshot(ctx, pool)
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
