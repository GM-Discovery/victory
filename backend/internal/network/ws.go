package network

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"

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
		readPump(hub, client)
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

func readPump(hub *Hub, c *Client) {
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

		if kind, _ := payload["type"].(string); kind == "ping" {
			_ = c.Conn.WriteJSON(map[string]any{
				"type": "pong",
				"ts":   time.Now().UTC().Format(time.RFC3339),
			})
		}
	}
}