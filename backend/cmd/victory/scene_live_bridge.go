package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
	"victory/backend/internal/network"
	"victory/backend/internal/scenes"
	"victory/backend/internal/shows"
)

// handleShowCurrentSceneWithLiveBridge replaces shows.HandleShowCurrentScene
// for the one route Kernel 93's live-stage bridge needs to hook: setting a
// Show's current Scene must also apply that Scene's captured composition to
// the live venue (scenes.ApplySceneToLiveVenue), not just flip which
// placement is considered current. Lives here rather than in shows/http.go
// because scenes already imports shows (for showRunForShow) -- shows
// importing scenes back would cycle; main has no such constraint.
func handleShowCurrentSceneWithLiveBridge(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		sessionCookie := ""
		if c, err := r.Cookie("victory_session"); err == nil {
			sessionCookie = c.Value
		}
		userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		showID := strings.TrimSpace(r.PathValue("show_id"))
		var body struct {
			ShowScenePlacementID string `json:"show_scene_placement_id"`
		}
		if r.ContentLength != 0 {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_request_body"})
				return
			}
		}

		placementID := strings.TrimSpace(body.ShowScenePlacementID)
		var s shows.Show
		reason := "current_scene_cleared"
		if placementID == "" {
			s, err = shows.ClearCurrentScenePlacement(ctx, pool, userID, showID)
		} else {
			s, err = shows.SetCurrentScenePlacement(ctx, pool, userID, showID, placementID)
			reason = "current_scene_set"
		}
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
			return
		}

		if placementID != "" {
			// Grant, 2026-08-23: "no fancy transitions... new configurations
			// including the background map." A hard cut is correct here --
			// don't hold up the response trying to be clever about diffing.
			if err := scenes.ApplySceneToLiveVenue(ctx, pool, userID, "catharsis", placementID); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "scene_apply_failed", "detail": err.Error()})
				return
			}
			broadcastCatharsisVenueUpdate(hub)
		}

		network.BroadcastShowStageInvalidation(ctx, hub, pool, showID, reason)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": map[string]any{"show": s}})
	}
}

// broadcastCatharsisVenueUpdate tells every connected client to refetch the
// venue's map and grid state -- the same "venue/update" message venues/
// map.go and venues/grid.go already send on a manual map/grid edit, which
// the client already handles (session-sync.js's venue_update case calls
// both refreshVenueMapState and refreshVenueGridConfig). Element/token/card
// changes are covered separately by BroadcastShowStageInvalidation's
// refreshWorld().
func broadcastCatharsisVenueUpdate(hub *network.Hub) {
	if hub == nil {
		return
	}
	msgOut, err := json.Marshal(map[string]any{
		"type":       "venue/update",
		"venue_slug": "catharsis",
	})
	if err != nil {
		return
	}
	hub.Broadcast(msgOut)
}
