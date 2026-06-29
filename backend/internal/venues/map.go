package venues

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"victory/backend/internal/access"
	"victory/backend/internal/network"
	"victory/backend/internal/sessions"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	firstTheaterSlug = "first-theater"
	catharsisSlug    = "catharsis"
)

func isSupportedTheaterVenueSlug(slug string) bool {
	switch strings.ToLower(strings.TrimSpace(slug)) {
	case firstTheaterSlug, catharsisSlug:
		return true
	default:
		return false
	}
}

type venueMapState struct {
	AssetID         string         `json:"asset_id"`
	Fit             string         `json:"fit"`
	CropX           float64        `json:"crop_x"`
	CropY           float64        `json:"crop_y"`
	Scale           float64        `json:"scale"`
	SafeMargin      int            `json:"safe_margin"`
	DisplayMode     string         `json:"display_mode"`
	UpdatedAt       string         `json:"updated_at,omitempty"`
	CreatedByUserID string         `json:"created_by_user_id,omitempty"`
	UpdatedByUserID string         `json:"updated_by_user_id,omitempty"`
	Asset           *venueMapAsset `json:"asset,omitempty"`
}

type venueMapAsset struct {
	AssetID          string   `json:"asset_id"`
	OriginalFilename string   `json:"original_filename"`
	SourceMime       string   `json:"source_mime"`
	SniffedMime      string   `json:"sniffed_mime"`
	AssetType        string   `json:"asset_type"`
	Tags             []string `json:"tags"`
	ContentURL       string   `json:"content_url"`
	Width            int      `json:"width"`
	Height           int      `json:"height"`
	ByteSize         int      `json:"byte_size"`
}

type venueMapRequest struct {
	AssetID     string  `json:"asset_id"`
	Fit         string  `json:"fit"`
	CropX       float64 `json:"crop_x"`
	CropY       float64 `json:"crop_y"`
	Scale       float64 `json:"scale"`
	SafeMargin  int     `json:"safe_margin"`
	DisplayMode string  `json:"display_mode"`
}

func HandleVenueMap(hub *network.Hub, pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		venueSlug := resolveVenueMapSlug(r)
		if !isSupportedTheaterVenueSlug(venueSlug) {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"ok":    false,
				"error": "venue_map_not_supported",
			})
			return
		}

		switch r.Method {
		case http.MethodGet:
			handleVenueMapGet(w, r, pool, venueSlug)
		case http.MethodPost:
			handleVenueMapSave(w, r, hub, pool, venueSlug)
		case http.MethodDelete:
			handleVenueMapDelete(w, r, hub, pool, venueSlug)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"ok":    false,
				"error": "method_not_allowed",
			})
		}
	}
}

func resolveVenueMapSlug(r *http.Request) string {
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
	if parts[0] != "api" || parts[1] != "venues" || parts[len(parts)-1] != "map" {
		return ""
	}

	return strings.ToLower(strings.TrimSpace(strings.Join(parts[2:len(parts)-1], "/")))
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func handleVenueMapGet(w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool, venueSlug string) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	allowed, err := canViewVenueMap(ctx, pool, r, venueSlug)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "map_access_check_failed",
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

	state, err := loadVenueMapState(ctx, pool, venueSlug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusOK, map[string]any{
				"ok":   true,
				"data": nil,
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "venue_map_lookup_failed",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"data": state,
	})
}

func handleVenueMapSave(w http.ResponseWriter, r *http.Request, hub *network.Hub, pool *pgxpool.Pool, venueSlug string) {
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()

	userID, allowed, err := canEditVenueMap(ctx, pool, r, venueSlug)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "map_access_check_failed",
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

	var req venueMapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": "invalid_json",
		})
		return
	}

	req.AssetID = strings.TrimSpace(req.AssetID)
	if req.AssetID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": "asset_id_required",
		})
		return
	}

	req.Fit = normalizeMapFit(req.Fit)
	req.CropX = clampFloat(req.CropX, 0, 1, 0.5)
	req.CropY = clampFloat(req.CropY, 0, 1, 0.5)
	req.Scale = clampFloat(req.Scale, 0.25, 4, 1)
	req.SafeMargin = clampInt(req.SafeMargin, 0, 128, 24)

	venueID, locationID, err := resolveVenueLocation(ctx, pool, venueSlug)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "venue_lookup_failed",
		})
		return
	}

	asset, err := loadVenueMapAsset(ctx, pool, req.AssetID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"ok":    false,
				"error": "asset_not_found",
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "asset_lookup_failed",
		})
		return
	}

	if asset.AssetType != "map" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": "asset_type_must_be_map",
		})
		return
	}

	var assetLocationID string
	if err := pool.QueryRow(ctx, `
		SELECT location_id::text
		FROM assets
		WHERE id = $1
		  AND is_deleted = FALSE
		LIMIT 1
	`, req.AssetID).Scan(&assetLocationID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"ok":    false,
				"error": "asset_not_found",
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "asset_lookup_failed",
		})
		return
	}

	if assetLocationID != locationID {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"ok":    false,
			"error": "asset_scope_mismatch",
		})
		return
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "tx_begin_failed",
		})
		return
	}
	defer tx.Rollback(ctx)

	var createdByUserID string
	err = tx.QueryRow(ctx, `
		SELECT created_by_user_id::text
		FROM venue_active_maps
		WHERE venue_id = $1
		LIMIT 1
	`, venueID).Scan(&createdByUserID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "venue_map_lookup_failed",
		})
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		createdByUserID = userID
	}

	displayMode := strings.TrimSpace(req.DisplayMode)
	if displayMode != "fullscreen" {
		displayMode = "theater"
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO venue_active_maps (
			venue_id,
			asset_id,
			fit,
			crop_x,
			crop_y,
			scale,
			safe_margin,
			display_mode,
			created_by_user_id,
			updated_by_user_id,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		ON CONFLICT (venue_id) DO UPDATE
		SET asset_id = EXCLUDED.asset_id,
		    fit = EXCLUDED.fit,
		    crop_x = EXCLUDED.crop_x,
		    crop_y = EXCLUDED.crop_y,
		    scale = EXCLUDED.scale,
		    safe_margin = EXCLUDED.safe_margin,
		    display_mode = EXCLUDED.display_mode,
		    updated_by_user_id = EXCLUDED.updated_by_user_id,
		    updated_at = NOW()
	`, venueID, req.AssetID, req.Fit, req.CropX, req.CropY, req.Scale, req.SafeMargin, displayMode, createdByUserID, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "venue_map_save_failed",
		})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "tx_commit_failed",
		})
		return
	}

	state, err := loadVenueMapState(ctx, pool, venueSlug)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "venue_map_lookup_failed",
		})
		return
	}

	broadcastVenueMapUpdate(hub, venueSlug, state)

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"data": state,
	})
}

func handleVenueMapDelete(w http.ResponseWriter, r *http.Request, hub *network.Hub, pool *pgxpool.Pool, venueSlug string) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	_, allowed, err := canEditVenueMap(ctx, pool, r, venueSlug)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "map_access_check_failed",
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

	venueID, _, err := resolveVenueLocation(ctx, pool, venueSlug)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "venue_lookup_failed",
		})
		return
	}

	if _, err := pool.Exec(ctx, `DELETE FROM venue_active_maps WHERE venue_id = $1`, venueID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "venue_map_delete_failed",
		})
		return
	}

	broadcastVenueMapUpdate(hub, venueSlug, nil)

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"data": nil,
	})
}

func broadcastVenueMapUpdate(hub *network.Hub, venueSlug string, state any) {
	if hub == nil {
		return
	}

	msgOut, err := json.Marshal(map[string]any{
		"type":       "venue/update",
		"venue_slug": strings.ToLower(strings.TrimSpace(venueSlug)),
		"map_state":  state,
	})
	if err != nil {
		return
	}

	hub.Broadcast(msgOut)
}

func resolveVenueMapAccess(ctx context.Context, pool *pgxpool.Pool, r *http.Request) (string, bool, error) {
	rawSession, err := sessions.ReadSessionCookie(r)
	if err != nil || strings.TrimSpace(rawSession) == "" {
		return "", false, nil
	}

	rec, err := sessions.GetSessionByRawToken(ctx, pool, rawSession)
	if err != nil || strings.TrimSpace(rec.UserID) == "" {
		return "", false, nil
	}

	userID := rec.UserID
	if ok, err := access.IsOperatorUser(ctx, pool, userID); err == nil && ok {
		return userID, true, nil
	}

	role, err := access.CurrentLocationRole(ctx, pool, userID)
	if err != nil {
		return "", false, err
	}
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "producer", "director":
		return userID, true, nil
	default:
		return userID, false, nil
	}
}

func canViewVenueMap(ctx context.Context, pool *pgxpool.Pool, r *http.Request, venueSlug string) (bool, error) {
	rawSession, err := sessions.ReadSessionCookie(r)
	userID := ""
	if err == nil && strings.TrimSpace(rawSession) != "" {
		if rec, sessionErr := sessions.GetSessionByRawToken(ctx, pool, rawSession); sessionErr == nil {
			userID = rec.UserID
		}
	}
	return access.UserCanAccessVenueSlug(ctx, pool, userID, venueSlug)
}

func canEditVenueMap(ctx context.Context, pool *pgxpool.Pool, r *http.Request, venueSlug string) (string, bool, error) {
	userID, allowed, err := resolveVenueMapAccess(ctx, pool, r)
	if err != nil || !allowed {
		return userID, allowed, err
	}
	if ok, err := access.UserCanAccessVenueSlug(ctx, pool, userID, venueSlug); err != nil {
		return userID, false, err
	} else if !ok {
		return userID, false, nil
	}
	return userID, true, nil
}

func loadVenueLocation(ctx context.Context, pool *pgxpool.Pool, venueSlug string) (venueID string, locationID string, err error) {
	err = pool.QueryRow(ctx, `
		SELECT v.id::text, l.id::text
		FROM venues v
		JOIN lots lo ON lo.id = v.lot_id
		JOIN locations l ON l.id = lo.location_id
		WHERE v.slug = $1
		LIMIT 1
	`, venueSlug).Scan(&venueID, &locationID)
	return venueID, locationID, err
}

func resolveVenueLocation(ctx context.Context, pool *pgxpool.Pool, venueSlug string) (venueID string, locationID string, err error) {
	return loadVenueLocation(ctx, pool, venueSlug)
}

func loadVenueMapState(ctx context.Context, pool *pgxpool.Pool, venueSlug string) (venueMapState, error) {
	var state venueMapState
	var asset venueMapAsset
	var assetID, fit, displayMode, updatedAt, createdByUserID, updatedByUserID string
	var cropX, cropY, scale float64
	var safeMargin int
	var assetOriginalFilename, assetSourceMime, assetSniffedMime, assetType string
	var assetTags []string
	var assetWidth, assetHeight, assetByteSize int
	var assetContentPath string

	err := pool.QueryRow(ctx, `
		SELECT
			vm.asset_id::text,
			COALESCE(vm.fit, 'cover'),
			COALESCE(vm.crop_x, 0.5),
			COALESCE(vm.crop_y, 0.5),
			COALESCE(vm.scale, 1),
			COALESCE(vm.safe_margin, 24),
			COALESCE(vm.display_mode, 'theater'),
			COALESCE(vm.updated_at::text, ''),
			COALESCE(vm.created_by_user_id::text, ''),
			COALESCE(vm.updated_by_user_id::text, ''),
			COALESCE(a.id::text, ''),
			COALESCE(NULLIF(a.original_filename, ''), a.id::text),
			COALESCE(a.source_mime, ''),
			COALESCE(a.sniffed_mime, ''),
			COALESCE(NULLIF(a.asset_type, ''), 'generic'),
			COALESCE(a.tags, '{}'::text[]),
			COALESCE(a.width, 0),
			COALESCE(a.height, 0),
			COALESCE(a.byte_size, 0),
			COALESCE(a.original_path, '')
		FROM venues v
		JOIN venue_active_maps vm ON vm.venue_id = v.id
		LEFT JOIN assets a ON a.id = vm.asset_id AND a.is_deleted = FALSE
		WHERE v.slug = $1
		LIMIT 1
	`, venueSlug).Scan(
		&assetID,
		&fit,
		&cropX,
		&cropY,
		&scale,
		&safeMargin,
		&displayMode,
		&updatedAt,
		&createdByUserID,
		&updatedByUserID,
		&asset.AssetID,
		&assetOriginalFilename,
		&assetSourceMime,
		&assetSniffedMime,
		&assetType,
		&assetTags,
		&assetWidth,
		&assetHeight,
		&assetByteSize,
		&assetContentPath,
	)
	if err != nil {
		return venueMapState{}, err
	}

	state = venueMapState{
		AssetID:         assetID,
		Fit:             fit,
		CropX:           cropX,
		CropY:           cropY,
		Scale:           scale,
		SafeMargin:      safeMargin,
		DisplayMode:     displayMode,
		UpdatedAt:       strings.TrimSpace(updatedAt),
		CreatedByUserID: strings.TrimSpace(createdByUserID),
		UpdatedByUserID: strings.TrimSpace(updatedByUserID),
	}
	if strings.TrimSpace(asset.AssetID) != "" {
		asset.OriginalFilename = assetOriginalFilename
		asset.SourceMime = assetSourceMime
		asset.SniffedMime = assetSniffedMime
		asset.AssetType = assetType
		asset.Tags = assetTags
		asset.ContentURL = "/api/assets/" + asset.AssetID + "/content"
		asset.Width = assetWidth
		asset.Height = assetHeight
		asset.ByteSize = assetByteSize
		state.Asset = &asset
	}

	_ = assetContentPath
	return state, nil
}

func loadVenueMapAsset(ctx context.Context, pool *pgxpool.Pool, assetID string) (venueMapAsset, error) {
	var asset venueMapAsset
	err := pool.QueryRow(ctx, `
		SELECT
			id::text,
			COALESCE(NULLIF(original_filename, ''), id::text),
			COALESCE(source_mime, ''),
			COALESCE(sniffed_mime, ''),
			COALESCE(NULLIF(asset_type, ''), 'generic'),
			COALESCE(tags, '{}'::text[]),
			COALESCE(width, 0),
			COALESCE(height, 0),
			COALESCE(byte_size, 0)
		FROM assets
		WHERE id = $1
		  AND is_deleted = FALSE
		LIMIT 1
	`, assetID).Scan(
		&asset.AssetID,
		&asset.OriginalFilename,
		&asset.SourceMime,
		&asset.SniffedMime,
		&asset.AssetType,
		&asset.Tags,
		&asset.Width,
		&asset.Height,
		&asset.ByteSize,
	)
	return asset, err
}

func normalizeMapFit(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "contain":
		return "contain"
	default:
		return "cover"
	}
}

func clampFloat(value, min, max, fallback float64) float64 {
	if value != value {
		return fallback
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func clampInt(value, min, max, fallback int) int {
	if value == 0 && fallback != 0 {
		return fallback
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
