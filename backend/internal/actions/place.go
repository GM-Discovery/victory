package actions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
	"victory/backend/internal/showings"
)

type PlaceElementRequest struct {
	SessionID   string  `json:"session_id"`
	ActorID     string  `json:"actor_id"`
	ElementID   string  `json:"element_id"`
	ElementSlug string  `json:"element_slug"`
	VenueSlug   string  `json:"venue_slug"`
	Layer       string  `json:"layer"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Order       int     `json:"order"`
}

func StorePlaceElement(ctx context.Context, pool *pgxpool.Pool, req PlaceElementRequest) (*StoredAction, error) {
	req = sanitizePlaceElementRequest(req)
	if req.SessionID == "" {
		return nil, errors.New("session_id is required")
	}
	if req.ActorID == "" {
		return nil, errors.New("actor_id is required")
	}
	if req.ElementID == "" && req.ElementSlug == "" {
		return nil, errors.New("element_id or element_slug is required")
	}
	if req.VenueSlug == "" {
		return nil, errors.New("venue_slug is required")
	}
	if req.Layer != "tray" && req.Layer != "stage" {
		return nil, errors.New("invalid_layer")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	decision, err := CanAct(ctx, tx, req.ActorID, "act/place_element", req.SessionID, ActionTarget{
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

	elementID, elementSlug, err := resolvePlaceableElement(ctx, tx, req.ElementID, req.ElementSlug)
	if err != nil {
		return nil, err
	}

	elementType := ""
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(NULLIF(e.element_type, ''), '')
		FROM elements e
		WHERE e.id = $1
		LIMIT 1
	`, elementID).Scan(&elementType); err != nil {
		return nil, err
	}

	venueID, enabled, err := resolvePlacementVenue(ctx, tx, req.VenueSlug)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, &ActionDeniedError{Reason: "policy_denied"}
	}

	if ok, err := access.UserCanAccessVenueSlug(ctx, pool, req.ActorID, req.VenueSlug); err != nil {
		return nil, err
	} else if !ok {
		return nil, &ActionDeniedError{Reason: "unknown_target"}
	}

	existingVisibility := map[string]any{}
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(vle.visibility, '{}'::jsonb)
		FROM venue_layout_elements vle
		WHERE vle.venue_id = $1
		  AND vle.element_id = $2
		LIMIT 1
	`, venueID, elementID).Scan(&existingVisibility); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	positionFrame := "top-left"
	if strings.EqualFold(strings.TrimSpace(elementType), "token") {
		positionFrame = "center"
	}
	placementPosition := map[string]any{
		"anchor": req.Layer,
		"x":      req.X,
		"y":      req.Y,
		"z":      0,
		"order":  req.Order,
		"frame":  positionFrame,
	}
	placementVisibility := mergeVisibilityState(existingVisibility, map[string]any{
		"toRoles":           []string{"director", "producer"},
		"privateTo":         []string{},
		"visible":           true,
		"nameplate_visible": visibilityBool(existingVisibility, "nameplate_visible", true),
		"locked":            visibilityBool(existingVisibility, "locked", false),
	})
	positionJSON, _ := json.Marshal(placementPosition)
	visibilityJSON := marshalVisibilityState(placementVisibility)

	if _, err := tx.Exec(ctx, `
		INSERT INTO venue_layout_elements (
			venue_id,
			element_id,
			surface,
			position,
			visibility,
			is_default
		)
		VALUES ($1, $2, $3, $4, $5, FALSE)
		ON CONFLICT (venue_id, element_id, surface) DO UPDATE
		SET position = EXCLUDED.position,
		    visibility = EXCLUDED.visibility,
		    is_default = FALSE
	`, venueID, elementID, req.Layer, positionJSON, visibilityJSON); err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM venue_layout_elements
		WHERE venue_id = $1
		  AND element_id = $2
		  AND surface <> $3
		  AND surface IN ('tray', 'stage')
	`, venueID, elementID, req.Layer); err != nil {
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
		"element_id":   elementID,
		"element_slug": elementSlug,
		"venue_slug":   req.VenueSlug,
		"layer":        req.Layer,
		"x":            req.X,
		"y":            req.Y,
	}
	payload := map[string]any{
		"venue_slug":    req.VenueSlug,
		"layer":         req.Layer,
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
		VALUES ($1, $2, $3, 'act/place_element', $4, $5, $6, $7, $8, TRUE)
		RETURNING id, ts
	`, req.SessionID, nextMoment, req.ActorID, targetJSON, payloadJSON, scopeJSON, visibilityJSON2, showing.ID).
		Scan(&out.ID, &ts); err != nil {
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
	out.Type = "act/place_element"
	out.Target = target
	out.Payload = payload
	out.Scope = scope
	out.Visibility = visibility
	out.Timestamp = ts.UTC().Format(time.RFC3339)

	return &out, nil
}

func sanitizePlaceElementRequest(req PlaceElementRequest) PlaceElementRequest {
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ActorID = strings.TrimSpace(req.ActorID)
	req.ElementID = strings.TrimSpace(req.ElementID)
	req.ElementSlug = strings.TrimSpace(req.ElementSlug)
	req.VenueSlug = strings.ToLower(strings.TrimSpace(req.VenueSlug))
	req.Layer = strings.ToLower(strings.TrimSpace(req.Layer))
	if req.Layer == "" {
		req.Layer = "tray"
	}
	return req
}

// resolvePlacementVenue gates placing/creating/duplicating elements onto a
// venue's stage. Kernel 72A: this checks stage_elements_enabled — the
// element-surface capability — not index_cards_enabled, which it borrowed
// historically (migration 010 only ever set that flag on the-cave, so token
// placement on first-theater/catharsis died here with policy_denied even
// after the venue passed every authority check). index_cards_enabled remains
// the index-card-specific policy consulted by the CanAct authority path.
func resolvePlacementVenue(ctx context.Context, tx pgx.Tx, venueSlug string) (venueID string, enabled bool, err error) {
	err = tx.QueryRow(ctx, `
		SELECT
			v.id::text,
			COALESCE((v.config ->> 'stage_elements_enabled')::boolean, FALSE)
		FROM venues v
		WHERE v.slug = $1
		LIMIT 1
	`, venueSlug).Scan(&venueID, &enabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, errors.New("unknown_target")
		}
		return "", false, err
	}

	return venueID, enabled, nil
}
