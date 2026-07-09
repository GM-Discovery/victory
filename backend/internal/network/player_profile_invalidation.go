package network

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/playerprofile"
)

// BroadcastPlayerProfileProjectionInvalidation emits a server-authored
// refetch notice after a committed Player Workbook / Trailer Face mutation
// (Kernel 61A §9.2-9.3). The payload carries only an opaque profile_id
// (the workbook ID) and a comparable version -- never facts, email,
// handle, or any other private data -- so it's safe to deliver to every
// client watching that profile_id, whether that's the owner's own other
// tabs or another authenticated user's Trailer-viewer tab (Kernel 61A
// §9.1: both watch by the same ID, there is no separate "owner channel").
func BroadcastPlayerProfileProjectionInvalidation(ctx context.Context, hub *Hub, pool *pgxpool.Pool, userID string, changedDimensions []string) {
	if hub == nil || pool == nil {
		return
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return
	}

	wb, err := playerprofile.EnsureWorkbook(ctx, pool, userID)
	if err != nil {
		log.Printf("player profile invalidation workbook lookup failed: user=%s err=%v", userID, err)
		return
	}

	version, err := playerprofile.ProjectionVersion(ctx, pool, wb.ID)
	if err != nil {
		log.Printf("player profile invalidation version lookup failed: workbook=%s err=%v", wb.ID, err)
		return
	}

	if len(changedDimensions) == 0 {
		changedDimensions = []string{"profile"}
	}

	payload := map[string]any{
		"type":               "player_profile/projection_updated",
		"profile_id":         wb.ID,
		"projection_version": version,
		"changed_dimensions": changedDimensions,
		"ts":                 time.Now().UTC().Format(time.RFC3339Nano),
	}
	msg, err := json.Marshal(payload)
	if err != nil {
		log.Printf("player profile invalidation marshal failed: %v", err)
		return
	}

	hub.BroadcastProfileWatchers(wb.ID, msg)
}
