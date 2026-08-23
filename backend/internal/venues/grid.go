package venues

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/network"
)

type venueGridConfig struct {
	GridType        string  `json:"grid_type"`
	HexOrientation  string  `json:"hex_orientation"`
	CellSize        float64 `json:"cell_size"`
	OffsetX         float64 `json:"offset_x"`
	OffsetY         float64 `json:"offset_y"`
	LineWidth       float64 `json:"line_width"`
	Opacity         float64 `json:"opacity"`
	LineStyle       string  `json:"line_style"`
	Visible         bool    `json:"visible"`
	UpdatedByUserID string  `json:"updated_by_user_id,omitempty"`
}

type venueGridRequest struct {
	GridType       string  `json:"grid_type"`
	HexOrientation string  `json:"hex_orientation"`
	CellSize       float64 `json:"cell_size"`
	OffsetX        float64 `json:"offset_x"`
	OffsetY        float64 `json:"offset_y"`
	LineWidth      float64 `json:"line_width"`
	Opacity        float64 `json:"opacity"`
	LineStyle      string  `json:"line_style"`
	Visible        bool    `json:"visible"`
}

func defaultVenueGridConfig() venueGridConfig {
	return venueGridConfig{
		GridType:       "none",
		HexOrientation: "flat-top",
		CellSize:       50,
		OffsetX:        0,
		OffsetY:        0,
		LineWidth:      1,
		Opacity:        0.45,
		LineStyle:      "neutral",
		Visible:        true,
	}
}

func HandleVenueGrid(hub *network.Hub, pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		venueSlug := resolveVenueGridSlug(r)
		if !isSupportedTheaterVenueSlug(venueSlug) {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"ok":    false,
				"error": "venue_grid_not_supported",
			})
			return
		}

		switch r.Method {
		case http.MethodGet:
			handleVenueGridGet(w, r, pool, venueSlug)
		case http.MethodPut:
			handleVenueGridSave(w, r, hub, pool, venueSlug)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"ok":    false,
				"error": "method_not_allowed",
			})
		}
	}
}

func resolveVenueGridSlug(r *http.Request) string {
	venueSlug := strings.ToLower(strings.TrimSpace(r.PathValue("slug")))
	if venueSlug != "" {
		return venueSlug
	}

	path := strings.Trim(strings.TrimSpace(r.URL.Path), "/")
	if path == "" {
		return ""
	}

	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		return ""
	}
	if parts[0] != "api" || parts[1] != "venues" || parts[len(parts)-1] != "grid" {
		return ""
	}

	return strings.ToLower(strings.TrimSpace(strings.Join(parts[2:len(parts)-1], "/")))
}

func handleVenueGridGet(w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool, venueSlug string) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	allowed, err := canViewVenueMap(ctx, pool, r, venueSlug)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "grid_access_check_failed",
		})
		return
	}
	if !allowed {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"ok":    false,
			"error": "forbidden",
		})
		return
	}

	config, err := loadVenueGridConfig(ctx, pool, venueSlug)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "venue_grid_lookup_failed",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"data": config,
	})
}

func handleVenueGridSave(w http.ResponseWriter, r *http.Request, hub *network.Hub, pool *pgxpool.Pool, venueSlug string) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	userID, allowed, err := canEditVenueMap(ctx, pool, r, venueSlug)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "grid_access_check_failed",
		})
		return
	}
	if !allowed {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"ok":    false,
			"error": "forbidden",
		})
		return
	}

	var req venueGridRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": "invalid_json",
		})
		return
	}

	req.GridType = strings.ToLower(strings.TrimSpace(req.GridType))
	if req.GridType != "square" && req.GridType != "hex" {
		req.GridType = "none"
	}

	req.HexOrientation = strings.ToLower(strings.TrimSpace(req.HexOrientation))
	if req.HexOrientation != "pointy-top" {
		req.HexOrientation = "flat-top"
	}

	req.LineStyle = strings.ToLower(strings.TrimSpace(req.LineStyle))
	if req.LineStyle != "light" && req.LineStyle != "dark" {
		req.LineStyle = "neutral"
	}

	req.CellSize = clampFloat(req.CellSize, 8, 500, 50)
	req.OffsetX = clampFloat(req.OffsetX, -2000, 2000, 0)
	req.OffsetY = clampFloat(req.OffsetY, -2000, 2000, 0)
	req.LineWidth = clampFloat(req.LineWidth, 0.5, 8, 1)
	req.Opacity = clampFloat(req.Opacity, 0, 1, 0.45)

	venueID, _, err := resolveVenueLocation(ctx, pool, venueSlug)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "venue_lookup_failed",
		})
		return
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO venue_grid_configs (
			venue_id,
			grid_type,
			hex_orientation,
			cell_size,
			offset_x,
			offset_y,
			line_width,
			opacity,
			line_style,
			visible,
			updated_by_user_id,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
		ON CONFLICT (venue_id) DO UPDATE
		SET grid_type = EXCLUDED.grid_type,
		    hex_orientation = EXCLUDED.hex_orientation,
		    cell_size = EXCLUDED.cell_size,
		    offset_x = EXCLUDED.offset_x,
		    offset_y = EXCLUDED.offset_y,
		    line_width = EXCLUDED.line_width,
		    opacity = EXCLUDED.opacity,
		    line_style = EXCLUDED.line_style,
		    visible = EXCLUDED.visible,
		    updated_by_user_id = EXCLUDED.updated_by_user_id,
		    updated_at = NOW()
	`, venueID, req.GridType, req.HexOrientation, req.CellSize, req.OffsetX, req.OffsetY, req.LineWidth, req.Opacity, req.LineStyle, req.Visible, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "venue_grid_save_failed",
		})
		return
	}

	config, err := loadVenueGridConfig(ctx, pool, venueSlug)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "venue_grid_lookup_failed",
		})
		return
	}

	// Grid changes previously only ever updated the saving client's own tab
	// -- reuse the same venue/update broadcast map.go's save/delete already
	// send (map_state is unused by the client's handler either way; it
	// always refetches both map and grid state on receipt) so every other
	// connected client, including Audience, picks up the change too.
	broadcastVenueMapUpdate(hub, venueSlug, nil)

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"data": config,
	})
}

func loadVenueGridConfig(ctx context.Context, pool *pgxpool.Pool, venueSlug string) (venueGridConfig, error) {
	config := defaultVenueGridConfig()
	var updatedByUserID string

	err := pool.QueryRow(ctx, `
		SELECT
			g.grid_type,
			g.hex_orientation,
			g.cell_size,
			g.offset_x,
			g.offset_y,
			g.line_width,
			g.opacity,
			g.line_style,
			g.visible,
			COALESCE(g.updated_by_user_id::text, '')
		FROM venues v
		JOIN venue_grid_configs g ON g.venue_id = v.id
		WHERE v.slug = $1
		LIMIT 1
	`, venueSlug).Scan(
		&config.GridType,
		&config.HexOrientation,
		&config.CellSize,
		&config.OffsetX,
		&config.OffsetY,
		&config.LineWidth,
		&config.Opacity,
		&config.LineStyle,
		&config.Visible,
		&updatedByUserID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return defaultVenueGridConfig(), nil
		}
		return venueGridConfig{}, err
	}

	config.UpdatedByUserID = strings.TrimSpace(updatedByUserID)
	return config, nil
}
