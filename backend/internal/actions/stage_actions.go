package actions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showings"
)

type ElementLockRequest struct {
	SessionID   string `json:"session_id"`
	ActorID     string `json:"actor_id"`
	ElementID   string `json:"element_id"`
	ElementSlug string `json:"element_slug"`
	Locked      bool   `json:"locked"`
}

type NameplateVisibilityRequest struct {
	SessionID   string `json:"session_id"`
	ActorID     string `json:"actor_id"`
	ElementID   string `json:"element_id"`
	ElementSlug string `json:"element_slug"`
	Visible     bool   `json:"visible"`
	Layer       string `json:"layer"`
}

func StoreSetElementLock(ctx context.Context, pool *pgxpool.Pool, req ElementLockRequest) (*StoredAction, error) {
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ActorID = strings.TrimSpace(req.ActorID)
	req.ElementID = strings.TrimSpace(req.ElementID)
	req.ElementSlug = strings.TrimSpace(req.ElementSlug)
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

	decision, err := CanAct(ctx, tx, req.ActorID, "act/set_element_lock", req.SessionID, ActionTarget{
		Kind:        "element",
		ElementID:   req.ElementID,
		ElementSlug: req.ElementSlug,
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

	nextVisibility := mergeVisibilityState(state.Visibility, map[string]any{
		"locked": req.Locked,
	})
	if _, ok := nextVisibility["nameplate_visible"]; !ok {
		nextVisibility["nameplate_visible"] = true
	}
	if _, ok := nextVisibility["visible"]; !ok {
		nextVisibility["visible"] = true
	}

	if _, err := tx.Exec(ctx, `
		UPDATE venue_layout_elements
		SET visibility = $4
		WHERE venue_id = $1
		  AND element_id = $2
		  AND surface = $3
	`, state.VenueID, state.ElementID, state.Surface, marshalVisibilityState(nextVisibility)); err != nil {
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
		"locked":        req.Locked,
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
		VALUES ($1, $2, $3, 'act/set_element_lock', $4, $5, $6, $7, $8, TRUE)
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
	out.Type = "act/set_element_lock"
	out.Target = target
	out.Payload = payload
	out.Scope = scope
	out.Visibility = visibility
	out.Timestamp = ts.UTC().Format(time.RFC3339)

	return &out, nil
}

func StoreSetNameplateVisibility(ctx context.Context, pool *pgxpool.Pool, req NameplateVisibilityRequest) (*StoredAction, error) {
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ActorID = strings.TrimSpace(req.ActorID)
	req.ElementID = strings.TrimSpace(req.ElementID)
	req.ElementSlug = strings.TrimSpace(req.ElementSlug)
	req.Layer = strings.TrimSpace(req.Layer)
	if req.SessionID == "" {
		return nil, errors.New("session_id is required")
	}
	if req.ActorID == "" {
		return nil, errors.New("actor_id is required")
	}
	if req.ElementID == "" && req.ElementSlug == "" {
		return nil, errors.New("element_id or element_slug is required")
	}
	if req.Layer == "" {
		req.Layer = "audience"
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	decision, err := CanAct(ctx, tx, req.ActorID, "act/set_nameplate_visibility", req.SessionID, ActionTarget{
		Kind:        "element",
		ElementID:   req.ElementID,
		ElementSlug: req.ElementSlug,
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

	nextVisibility := mergeVisibilityState(state.Visibility, map[string]any{
		"nameplate_visible": req.Visible,
	})
	if _, ok := nextVisibility["locked"]; !ok {
		nextVisibility["locked"] = false
	}
	if _, ok := nextVisibility["visible"]; !ok {
		nextVisibility["visible"] = true
	}

	if _, err := tx.Exec(ctx, `
		UPDATE venue_layout_elements
		SET visibility = $4
		WHERE venue_id = $1
		  AND element_id = $2
		  AND surface = $3
	`, state.VenueID, state.ElementID, state.Surface, marshalVisibilityState(nextVisibility)); err != nil {
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
		"visible":       req.Visible,
		"layer":         req.Layer,
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
		VALUES ($1, $2, $3, 'act/set_nameplate_visibility', $4, $5, $6, $7, $8, TRUE)
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
	out.Type = "act/set_nameplate_visibility"
	out.Target = target
	out.Payload = payload
	out.Scope = scope
	out.Visibility = visibility
	out.Timestamp = ts.UTC().Format(time.RFC3339)

	return &out, nil
}
