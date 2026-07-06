package network

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/characters"
	"victory/backend/internal/commands"
)

// BroadcastCharacterProjectionInvalidation emits a server-authored refetch
// notice after a committed character projection mutation. The payload is an
// invalidation only; clients must refetch the canonical venue sheet.
func BroadcastCharacterProjectionInvalidation(ctx context.Context, hub *Hub, pool *pgxpool.Pool, cardID string, changedDimensions []string, sourceEventID string) {
	if hub == nil || pool == nil {
		return
	}
	cardID = strings.TrimSpace(cardID)
	if cardID == "" {
		return
	}
	version, err := characters.CharacterProjectionVersion(ctx, pool, cardID)
	if err != nil {
		log.Printf("projection invalidation version lookup failed: card=%s err=%v", cardID, err)
		return
	}
	if len(changedDimensions) == 0 {
		changedDimensions = []string{"face"}
	}
	payload := map[string]any{
		"type":               "character/projection_updated",
		"character_id":       cardID,
		"projection_version": version,
		"changed_dimensions": changedDimensions,
		"source_event_id":    strings.TrimSpace(sourceEventID),
		"ts":                 time.Now().UTC().Format(time.RFC3339Nano),
	}
	msg, err := json.Marshal(payload)
	if err != nil {
		log.Printf("projection invalidation marshal failed: %v", err)
		return
	}

	hub.mu.RLock()
	clients := make([]*Client, 0, len(hub.clients))
	for c := range hub.clients {
		clients = append(clients, c)
	}
	hub.mu.RUnlock()

	for _, c := range clients {
		if c == nil || strings.TrimSpace(c.UserID) == "" {
			continue
		}
		if !clientMayReceiveProjectionInvalidation(ctx, pool, c, cardID) {
			continue
		}
		select {
		case c.Send <- msg:
		default:
			log.Printf("dropping slow websocket client")
		}
	}
}

func clientMayReceiveProjectionInvalidation(ctx context.Context, pool *pgxpool.Pool, c *Client, cardID string) bool {
	persona, _, err := commands.ResolveActiveCharacter(ctx, pool, c.UserID, c.SessionID)
	if err == nil && strings.TrimSpace(stringFromMap(persona, "character_card_id")) == cardID {
		return true
	}
	allowed, err := characters.CanEditCard(ctx, pool, c.UserID, cardID)
	return err == nil && allowed
}

func stringFromMap(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	out, _ := values[key].(string)
	return strings.TrimSpace(out)
}
