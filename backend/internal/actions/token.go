package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showings"
)

type TokenPlacementRequest struct {
	SessionID  string  `json:"session_id"`
	ActorID    string  `json:"actor_id"`
	AssetID    string  `json:"asset_id"`
	VenueSlug  string  `json:"venue_slug"`
	Layer      string  `json:"layer"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Order      int     `json:"order"`
	SnapMode   string  `json:"snap_mode"`
	TokenLayer string  `json:"token_layer"`
	Scale      float64 `json:"scale"`
}

type TokenUpdateRequest struct {
	SessionID   string  `json:"session_id"`
	ActorID     string  `json:"actor_id"`
	ElementID   string  `json:"element_id"`
	ElementSlug string  `json:"element_slug"`
	AssetID     string  `json:"asset_id"`
	VenueSlug   string  `json:"venue_slug"`
	Layer       string  `json:"layer"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Order       int     `json:"order"`
	SnapMode    string  `json:"snap_mode"`
	TokenLayer  string  `json:"token_layer"`
	Scale       float64 `json:"scale"`
}

type warehouseTokenAssetRef struct {
	ID                string
	Name              string
	Shape             string
	DefaultGridWidth  int
	DefaultGridHeight int
	RetainOriginal    bool
	AssetType         string
	Status            string
	LocationID        string
	OriginalFilename  string
	SourceMime        string
	SniffedMime       string
	StoredBytes       int64
}

func StoreCreateToken(ctx context.Context, pool *pgxpool.Pool, req TokenPlacementRequest) (*StoredAction, error) {
	req = sanitizeTokenPlacementRequest(req)
	if req.SessionID == "" {
		return nil, errors.New("session_id is required")
	}
	if req.ActorID == "" {
		return nil, errors.New("actor_id is required")
	}
	if req.AssetID == "" {
		return nil, errors.New("asset_id is required")
	}
	if req.VenueSlug == "" {
		return nil, errors.New("venue_slug is required")
	}
	if req.Layer == "" {
		req.Layer = "stage"
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	decision, err := CanAct(ctx, tx, req.ActorID, "create/token", req.SessionID, ActionTarget{
		Kind:      "warehouse_asset",
		ElementID: req.AssetID,
		VenueSlug: req.VenueSlug,
		Layer:     req.Layer,
	})
	if err != nil {
		return nil, err
	}
	if !decision.Allowed {
		return nil, &ActionDeniedError{Reason: decision.Reason}
	}

	showing, err := showings.EnsureForSession(ctx, tx, req.SessionID, req.ActorID)
	if err != nil {
		return nil, err
	}

	asset, err := loadWarehouseTokenAsset(ctx, tx, req.AssetID)
	if err != nil {
		return nil, err
	}

	libraryID, err := resolveIndexCardLibrary(ctx, tx, asset.LocationID)
	if err != nil {
		return nil, err
	}

	displayName, handle, role, persona, err := loadActorIdentity(ctx, tx, req.SessionID, req.ActorID)
	if err != nil {
		return nil, err
	}

	var nextMoment int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(moment_id), 0) + 1
		FROM actions
		WHERE session_id = $1
	`, req.SessionID).Scan(&nextMoment); err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	snapMode := normalizeTokenSnapMode(req.SnapMode)
	tokenLayer := normalizeTokenLayer(req.TokenLayer)
	if tokenLayer == "" {
		tokenLayer = "public"
	}
	scale := clampTokenScale(req.Scale)
	tokenName := strings.TrimSpace(asset.Name)
	if tokenName == "" {
		tokenName = asset.OriginalFilename
	}
	if tokenName == "" {
		tokenName = asset.ID
	}
	tokenSlug := uniqueTokenSlug(tokenName, req.ActorID)
	visible := tokenLayer != "director"

	data := map[string]any{
		"type":                    "token",
		"context_class":           "token",
		"asset_id":                asset.ID,
		"asset_name":              tokenName,
		"asset_shape":             asset.Shape,
		"asset_type":              asset.AssetType,
		"asset_status":            asset.Status,
		"asset_content_url":       "/api/assets/" + asset.ID + "/content?variant=stage",
		"asset_thumbnail_url":     "/api/assets/" + asset.ID + "/content?variant=thumbnail",
		"default_grid_width":      maxIntLocal(asset.DefaultGridWidth, 1),
		"default_grid_height":     maxIntLocal(asset.DefaultGridHeight, 1),
		"retain_original":         asset.RetainOriginal,
		"snap_mode":               snapMode,
		"grid_relative":           snapMode == "grid",
		"token_layer":             tokenLayer,
		"scale":                   scale,
		"created_by":              req.ActorID,
		"created_by_display_name": displayName,
		"created_by_handle":       handle,
		"created_by_role":         role,
		"created_at":              now,
		"updated_at":              now,
		"venue_slug":              req.VenueSlug,
		"session_id":              req.SessionID,
	}
	dataJSON, _ := json.Marshal(data)

	elementID := ""
	if err := tx.QueryRow(ctx, `
		INSERT INTO elements (
			library_id,
			name,
			slug,
			element_type,
			context_class,
			state,
			data
		)
		VALUES ($1, $2, $3, 'token', 'token', 'library', $4)
		RETURNING id::text
	`, libraryID, tokenName, tokenSlug, dataJSON).Scan(&elementID); err != nil {
		return nil, err
	}

	venueID, enabled, err := resolvePlacementVenue(ctx, tx, req.VenueSlug)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, &ActionDeniedError{Reason: "policy_denied"}
	}

	positionJSON, _ := json.Marshal(map[string]any{
		"anchor": "stage",
		"x":      req.X,
		"y":      req.Y,
		"z":      0,
		"order":  req.Order,
		"frame":  "center",
	})
	visibilityJSON := marshalVisibilityState(map[string]any{
		"toRoles":           []string{"director", "producer"},
		"privateTo":         []string{},
		"visible":           visible,
		"nameplate_visible": true,
		"locked":            false,
	})

	if _, err := tx.Exec(ctx, `
		INSERT INTO venue_layout_elements (
			venue_id,
			element_id,
			surface,
			position,
			visibility,
			is_default
		)
		VALUES ($1, $2, 'stage', $3, $4, FALSE)
		ON CONFLICT (venue_id, element_id, surface) DO UPDATE
		SET position = EXCLUDED.position,
		    visibility = EXCLUDED.visibility,
		    is_default = FALSE
	`, venueID, elementID, positionJSON, visibilityJSON); err != nil {
		return nil, err
	}

	target := map[string]any{
		"kind":         "element",
		"element_id":   elementID,
		"element_slug": tokenSlug,
		"asset_id":     asset.ID,
		"venue_slug":   req.VenueSlug,
		"layer":        req.Layer,
	}
	payload := map[string]any{
		"asset_id":      asset.ID,
		"asset_name":    tokenName,
		"asset_shape":   asset.Shape,
		"scale":         scale,
		"snap_mode":     snapMode,
		"token_layer":   tokenLayer,
		"x":             req.X,
		"y":             req.Y,
		"order":         req.Order,
		"actor_persona": persona,
	}
	scope := map[string]any{
		"surfaces":         []string{req.Layer},
		"audienceSegments": []string{"director", "producer"},
	}
	visibility := map[string]any{
		"toRoles":   []string{"director", "producer"},
		"privateTo": []string{},
	}

	targetJSON, _ := json.Marshal(target)
	payloadJSON, _ := json.Marshal(payload)
	scopeJSON, _ := json.Marshal(scope)
	visibilityJSON2, _ := json.Marshal(visibility)

	var out StoredAction
	var ts time.Time
	if err := tx.QueryRow(ctx, `
		INSERT INTO actions (
			session_id,
			moment_id,
			actor_id,
			type,
			target,
			payload,
			scope,
			visibility,
			showing_id,
			recorded
		)
		VALUES ($1, $2, $3, 'create/token', $4, $5, $6, $7, $8, TRUE)
		RETURNING id, ts
	`, req.SessionID, nextMoment, req.ActorID, targetJSON, payloadJSON, scopeJSON, visibilityJSON2, showing.ID).Scan(&out.ID, &ts); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	out.SessionID = req.SessionID
	out.ShowingID = showing.ID
	out.MomentID = nextMoment
	out.ActorID = req.ActorID
	out.ActorDisplayName = displayName
	out.ActorHandle = handle
	out.ActorRole = role
	out.Actor = map[string]any{
		"user_id":      req.ActorID,
		"handle":       handle,
		"display_name": displayName,
		"role":         role,
		"persona":      persona,
	}
	out.Persona = persona
	out.Type = "create/token"
	out.Target = target
	out.Payload = payload
	out.Scope = scope
	out.Visibility = visibility
	out.Timestamp = ts.UTC().Format(time.RFC3339)

	return &out, nil
}

func StoreUpdateToken(ctx context.Context, pool *pgxpool.Pool, req TokenUpdateRequest) (*StoredAction, error) {
	req = sanitizeTokenUpdateRequest(req)
	if req.SessionID == "" {
		return nil, errors.New("session_id is required")
	}
	if req.ActorID == "" {
		return nil, errors.New("actor_id is required")
	}
	if req.ElementID == "" && req.ElementSlug == "" {
		return nil, errors.New("element_id or element_slug is required")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	decision, err := CanAct(ctx, tx, req.ActorID, "update/token", req.SessionID, ActionTarget{
		Kind:        "element",
		ElementID:   req.ElementID,
		ElementSlug: req.ElementSlug,
		VenueSlug:   req.VenueSlug,
		Layer:       req.Layer,
	})
	if err != nil {
		return nil, err
	}
	if !decision.Allowed {
		return nil, &ActionDeniedError{Reason: decision.Reason}
	}

	showing, err := showings.EnsureForSession(ctx, tx, req.SessionID, req.ActorID)
	if err != nil {
		return nil, err
	}

	state, err := resolveVenueLayoutElementState(ctx, tx, req.SessionID, req.ElementID, req.ElementSlug)
	if err != nil {
		return nil, err
	}
	if state.VenueSlug != "the-cave" || strings.ToLower(strings.TrimSpace(state.Surface)) != "stage" {
		return nil, &ActionDeniedError{Reason: "unknown_target"}
	}

	currentData, currentName, err := loadTokenElementData(ctx, tx, state.ElementID)
	if err != nil {
		return nil, err
	}
	if strings.ToLower(strings.TrimSpace(stringValue(currentData["context_class"]))) != "token" && strings.ToLower(strings.TrimSpace(state.ContextClass)) != "token" {
		return nil, &ActionDeniedError{Reason: "unknown_target"}
	}

	updatedData := cloneStringMap(currentData)
	if updatedData == nil {
		updatedData = map[string]any{}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	scale := clampTokenScale(req.Scale)
	if req.Scale > 0 {
		updatedData["scale"] = scale
	}

	if snapMode := normalizeTokenSnapMode(req.SnapMode); snapMode != "" {
		updatedData["snap_mode"] = snapMode
		updatedData["grid_relative"] = snapMode == "grid"
	}

	if layer := normalizeTokenLayer(req.TokenLayer); layer != "" {
		updatedData["token_layer"] = layer
		visibility := mergeVisibilityState(state.Visibility, map[string]any{
			"visible": layer != "director",
		})
		if _, ok := visibility["nameplate_visible"]; !ok {
			visibility["nameplate_visible"] = true
		}
		if _, ok := visibility["locked"]; !ok {
			visibility["locked"] = false
		}
		if _, err := tx.Exec(ctx, `
			UPDATE venue_layout_elements
			SET visibility = $4
			WHERE venue_id = $1
			  AND element_id = $2
			  AND surface = $3
		`, state.VenueID, state.ElementID, state.Surface, marshalVisibilityState(visibility)); err != nil {
			return nil, err
		}
		state.Visibility = visibility
	}

	positionFrame := "center"
	if snapMode := normalizeTokenSnapMode(req.SnapMode); snapMode == "free" {
		positionFrame = "top-left"
	}
	if _, err := tx.Exec(ctx, `
		UPDATE venue_layout_elements
		SET position = jsonb_build_object(
			'anchor', surface,
			'x', $4::double precision,
			'y', $5::double precision,
			'z', 0,
			'order', $6::integer,
			'frame', $7::text
		)
		WHERE venue_id = $1
		  AND element_id = $2
		  AND surface = $3
	`, state.VenueID, state.ElementID, state.Surface, req.X, req.Y, req.Order, positionFrame); err != nil {
		return nil, err
	}

	var asset warehouseTokenAssetRef
	if req.AssetID != "" && !strings.EqualFold(strings.TrimSpace(req.AssetID), strings.TrimSpace(stringValue(updatedData["asset_id"]))) {
		asset, err = loadWarehouseTokenAsset(ctx, tx, req.AssetID)
		if err != nil {
			return nil, err
		}
		updatedData["asset_id"] = asset.ID
		updatedData["asset_name"] = asset.Name
		updatedData["asset_shape"] = asset.Shape
		updatedData["asset_type"] = asset.AssetType
		updatedData["asset_status"] = asset.Status
		updatedData["asset_content_url"] = "/api/assets/" + asset.ID + "/content?variant=stage"
		updatedData["asset_thumbnail_url"] = "/api/assets/" + asset.ID + "/content?variant=thumbnail"
		updatedData["default_grid_width"] = maxIntLocal(asset.DefaultGridWidth, 1)
		updatedData["default_grid_height"] = maxIntLocal(asset.DefaultGridHeight, 1)
		updatedData["retain_original"] = asset.RetainOriginal
		currentName = asset.Name
	}

	updatedData["updated_at"] = now
	if req.Scale > 0 {
		updatedData["scale"] = scale
	}

	dataJSON, _ := json.Marshal(updatedData)
	if _, err := tx.Exec(ctx, `
		UPDATE elements
		SET name = $2,
		    data = $3
		WHERE id = $1
		  AND COALESCE(state::text, '') <> 'deleted'
	`, state.ElementID, currentName, dataJSON); err != nil {
		return nil, err
	}

	displayName, handle, role, persona, err := loadActorIdentity(ctx, tx, req.SessionID, req.ActorID)
	if err != nil {
		return nil, err
	}

	var nextMoment int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(moment_id), 0) + 1
		FROM actions
		WHERE session_id = $1
	`, req.SessionID).Scan(&nextMoment); err != nil {
		return nil, err
	}

	target := map[string]any{
		"kind":         "element",
		"element_id":   state.ElementID,
		"element_slug": state.ElementSlug,
		"venue_slug":   state.VenueSlug,
		"surface":      state.Surface,
	}
	payload := map[string]any{
		"asset_id":      updatedData["asset_id"],
		"scale":         updatedData["scale"],
		"snap_mode":     updatedData["snap_mode"],
		"token_layer":   updatedData["token_layer"],
		"x":             req.X,
		"y":             req.Y,
		"order":         req.Order,
		"actor_persona": persona,
	}
	scope := map[string]any{
		"surfaces":         []string{state.Surface},
		"audienceSegments": []string{"director", "producer"},
	}
	visibility := map[string]any{
		"toRoles":   []string{"director", "producer"},
		"privateTo": []string{},
	}
	targetJSON, _ := json.Marshal(target)
	payloadJSON, _ := json.Marshal(payload)
	scopeJSON, _ := json.Marshal(scope)
	visibilityJSON, _ := json.Marshal(visibility)

	var out StoredAction
	var ts time.Time
	if err := tx.QueryRow(ctx, `
		INSERT INTO actions (
			session_id,
			moment_id,
			actor_id,
			type,
			target,
			payload,
			scope,
			visibility,
			showing_id,
			recorded
		)
		VALUES ($1, $2, $3, 'update/token', $4, $5, $6, $7, $8, TRUE)
		RETURNING id, ts
	`, req.SessionID, nextMoment, req.ActorID, targetJSON, payloadJSON, scopeJSON, visibilityJSON, showing.ID).Scan(&out.ID, &ts); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	out.SessionID = req.SessionID
	out.ShowingID = showing.ID
	out.MomentID = nextMoment
	out.ActorID = req.ActorID
	out.ActorDisplayName = displayName
	out.ActorHandle = handle
	out.ActorRole = role
	out.Actor = map[string]any{
		"user_id":      req.ActorID,
		"handle":       handle,
		"display_name": displayName,
		"role":         role,
		"persona":      persona,
	}
	out.Persona = persona
	out.Type = "update/token"
	out.Target = target
	out.Payload = payload
	out.Scope = scope
	out.Visibility = visibility
	out.Timestamp = ts.UTC().Format(time.RFC3339)

	return &out, nil
}

func loadWarehouseTokenAsset(ctx context.Context, q actionQuerier, assetID string) (warehouseTokenAssetRef, error) {
	assetID = strings.TrimSpace(assetID)
	if assetID == "" {
		return warehouseTokenAssetRef{}, errors.New("asset_id is required")
	}

	var rec warehouseTokenAssetRef
	err := q.QueryRow(ctx, `
		SELECT
			a.id::text,
			COALESCE(NULLIF(a.name, ''), COALESCE(NULLIF(a.original_filename, ''), a.id::text)) AS name,
			COALESCE(NULLIF(a.shape, ''), 'circle') AS shape,
			COALESCE(a.default_grid_width, 1),
			COALESCE(a.default_grid_height, 1),
			COALESCE(a.retain_original, FALSE),
			COALESCE(NULLIF(a.asset_type, ''), 'generic') AS asset_type,
			COALESCE(NULLIF(a.status, ''), 'active') AS status,
			a.location_id::text,
			COALESCE(NULLIF(a.original_filename, ''), ''),
			COALESCE(a.source_mime, ''),
			COALESCE(a.sniffed_mime, ''),
			COALESCE(a.stored_bytes, 0)
		FROM assets a
		WHERE a.id = $1
		  AND COALESCE(a.is_deleted, FALSE) = FALSE
		  AND a.deleted_at IS NULL
		  AND COALESCE(NULLIF(a.asset_type, ''), 'generic') = 'token'
		  AND COALESCE(NULLIF(a.status, ''), 'active') = 'active'
		LIMIT 1
	`, assetID).Scan(
		&rec.ID,
		&rec.Name,
		&rec.Shape,
		&rec.DefaultGridWidth,
		&rec.DefaultGridHeight,
		&rec.RetainOriginal,
		&rec.AssetType,
		&rec.Status,
		&rec.LocationID,
		&rec.OriginalFilename,
		&rec.SourceMime,
		&rec.SniffedMime,
		&rec.StoredBytes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return warehouseTokenAssetRef{}, errors.New("token_asset_not_found")
		}
		return warehouseTokenAssetRef{}, err
	}
	return rec, nil
}

func loadTokenElementData(ctx context.Context, q actionQuerier, elementID string) (map[string]any, string, error) {
	var (
		name    string
		dataRaw []byte
	)
	if err := q.QueryRow(ctx, `
		SELECT
			e.name,
			e.data
		FROM elements e
		WHERE e.id = $1
		  AND COALESCE(e.state::text, '') <> 'deleted'
		LIMIT 1
	`, elementID).Scan(&name, &dataRaw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", errors.New("token_not_found")
		}
		return nil, "", err
	}

	data := decodeStageJSONMap(dataRaw)
	return data, name, nil
}

func sanitizeTokenPlacementRequest(req TokenPlacementRequest) TokenPlacementRequest {
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ActorID = strings.TrimSpace(req.ActorID)
	req.AssetID = strings.TrimSpace(req.AssetID)
	req.VenueSlug = strings.ToLower(strings.TrimSpace(req.VenueSlug))
	req.Layer = strings.ToLower(strings.TrimSpace(req.Layer))
	req.SnapMode = normalizeTokenSnapMode(req.SnapMode)
	req.TokenLayer = normalizeTokenLayer(req.TokenLayer)
	if req.TokenLayer == "" {
		req.TokenLayer = "public"
	}
	req.Scale = clampTokenScale(req.Scale)
	if req.Layer == "" {
		req.Layer = "stage"
	}
	return req
}

func sanitizeTokenUpdateRequest(req TokenUpdateRequest) TokenUpdateRequest {
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ActorID = strings.TrimSpace(req.ActorID)
	req.ElementID = strings.TrimSpace(req.ElementID)
	req.ElementSlug = strings.TrimSpace(req.ElementSlug)
	req.AssetID = strings.TrimSpace(req.AssetID)
	req.VenueSlug = strings.ToLower(strings.TrimSpace(req.VenueSlug))
	req.Layer = strings.ToLower(strings.TrimSpace(req.Layer))
	req.SnapMode = normalizeTokenSnapMode(req.SnapMode)
	req.TokenLayer = normalizeTokenLayer(req.TokenLayer)
	if req.Scale > 0 {
		req.Scale = clampTokenScale(req.Scale)
	}
	return req
}

func normalizeTokenLayer(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "director":
		return "director"
	case "public":
		return "public"
	default:
		return ""
	}
}

func normalizeTokenSnapMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "grid":
		return "grid"
	case "free":
		return "free"
	default:
		return ""
	}
}

func clampTokenScale(value float64) float64 {
	if !isFiniteFloat(value) || value <= 0 {
		return 100
	}
	if value < 25 {
		return 25
	}
	if value > 500 {
		return 500
	}
	return mathRound2(value)
}

func isFiniteFloat(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func mathRound2(value float64) float64 {
	return math.Round(value*100) / 100
}

func maxIntLocal(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func uniqueTokenSlug(name, actorID string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "token"
	}
	name = strings.ToLower(name)
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-':
			return r
		default:
			return '-'
		}
	}, name)
	name = strings.Trim(name, "-")
	if name == "" {
		name = "token"
	}
	actorID = strings.ToLower(strings.TrimSpace(actorID))
	if len(actorID) > 8 {
		actorID = actorID[:8]
	}
	if actorID == "" {
		actorID = "user"
	}
	return fmt.Sprintf("%s-copy-%s-%d", name, actorID, time.Now().UTC().UnixNano())
}
