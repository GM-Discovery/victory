package network

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// BroadcastShowStageInvalidation emits a server-authored refetch notice
// after a committed Cue execution or current-Scene-pointer change,
// following BroadcastCharacterProjectionInvalidation's lighter-weight
// "invalidation only, client refetches" pattern rather than a full
// snapshot rebroadcast (Kernel 70). It is sent to every connected client
// whose active session is currently linked to showID -- resolved fresh
// against sessions.show_id each call rather than a cached per-client
// field, so there is nothing to keep in sync when a session links/unlinks
// from a Show elsewhere.
func BroadcastShowStageInvalidation(ctx context.Context, hub *Hub, pool *pgxpool.Pool, showID, reason string) {
	if hub == nil || pool == nil {
		return
	}
	showID = strings.TrimSpace(showID)
	if showID == "" {
		return
	}

	sessionIDs, err := sessionIDsLinkedToShow(ctx, pool, showID)
	if err != nil {
		log.Printf("show stage invalidation session lookup failed: show=%s err=%v", showID, err)
		return
	}
	if len(sessionIDs) == 0 {
		return
	}

	payload := map[string]any{
		"type":    "show/stage_updated",
		"show_id": showID,
		"reason":  strings.TrimSpace(reason),
		"ts":      time.Now().UTC().Format(time.RFC3339Nano),
	}
	msg, err := json.Marshal(payload)
	if err != nil {
		log.Printf("show stage invalidation marshal failed: %v", err)
		return
	}

	hub.mu.RLock()
	clients := make([]*Client, 0, len(hub.clients))
	for c := range hub.clients {
		clients = append(clients, c)
	}
	hub.mu.RUnlock()

	for _, c := range clients {
		if c == nil || !sessionIDs[c.SessionID] {
			continue
		}
		select {
		case c.Send <- msg:
		default:
			log.Printf("dropping slow websocket client")
		}
	}
}

func sessionIDsLinkedToShow(ctx context.Context, pool *pgxpool.Pool, showID string) (map[string]bool, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text FROM sessions WHERE show_id = $1 AND status IN ('rehearsal', 'live')
	`, showID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}
