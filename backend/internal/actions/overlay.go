package actions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type OverlayRequest struct {
	SessionID   string `json:"session_id"`
	ActorID     string `json:"actor_id"`
	ElementID   string `json:"element_id"`
	ElementSlug string `json:"element_slug"`
	OverlayType string `json:"overlay_type"`
}

func StoreOverlayShow(ctx context.Context, pool *pgxpool.Pool, req OverlayRequest) (*StoredAction, error) {
	return storeOverlayAction(ctx, pool, req, "act/show_overlay")
}

func StoreOverlayHide(ctx context.Context, pool *pgxpool.Pool, req OverlayRequest) (*StoredAction, error) {
	return storeOverlayAction(ctx, pool, req, "act/hide_overlay")
}

func storeOverlayAction(ctx context.Context, pool *pgxpool.Pool, req OverlayRequest, actionType string) (*StoredAction, error) {
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ActorID = strings.TrimSpace(req.ActorID)
	req.ElementID = strings.TrimSpace(req.ElementID)
	req.ElementSlug = strings.TrimSpace(req.ElementSlug)
	req.OverlayType = strings.TrimSpace(strings.ToLower(req.OverlayType))

	if req.SessionID == "" {
		return nil, errors.New("session_id is required")
	}
	if req.ActorID == "" {
		return nil, errors.New("actor_id is required")
	}
	if req.ElementID == "" && req.ElementSlug == "" {
		return nil, errors.New("element_id or element_slug is required")
	}
	if actionType == "act/show_overlay" && req.OverlayType != "text" && req.OverlayType != "image" {
		return nil, errors.New("invalid overlay type")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	resolvedID, resolvedSlug, _, resolvedSurface, err := resolveRevealTarget(ctx, tx, req.SessionID, req.ElementID, req.ElementSlug)
	if err != nil {
		return nil, err
	}
	if strings.ToLower(strings.TrimSpace(resolvedSurface)) != "stage" {
		return nil, errors.New("element is not revealable")
	}

	target := ActionTarget{
		Kind:        "element",
		ElementID:   resolvedID,
		ElementSlug: resolvedSlug,
		Layer:       "stage",
	}
	decision, err := CanAct(ctx, tx, req.ActorID, actionType, req.SessionID, target)
	if err != nil {
		return nil, err
	}
	if !decision.Allowed {
		return nil, &ActionDeniedError{Reason: decision.Reason}
	}

	displayName, handle, role, err := loadActorIdentity(ctx, tx, req.SessionID, req.ActorID)
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

	payload := map[string]any{
		"overlay_type": req.OverlayType,
	}
	targetPayload := map[string]any{
		"element_id":   resolvedID,
		"element_slug": resolvedSlug,
	}
	scope := map[string]any{
		"surfaces":         []string{"overlay"},
		"audienceSegments": []string{"all"},
	}
	visibility := map[string]any{
		"toRoles":   []string{"audience", "cast", "crew", "director", "producer"},
		"privateTo": []string{},
	}

	targetJSON, _ := json.Marshal(targetPayload)
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
			recorded
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, TRUE)
		RETURNING id, ts
	`, req.SessionID, nextMoment, req.ActorID, actionType, targetJSON, payloadJSON, scopeJSON, visibilityJSON).
		Scan(&out.ID, &ts); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	out.SessionID = req.SessionID
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
		"persona":      nil,
	}
	out.Persona = nil
	out.Type = actionType
	out.Target = targetPayload
	out.Payload = payload
	out.Scope = scope
	out.Visibility = visibility
	out.Timestamp = ts.UTC().Format(time.RFC3339)

	return &out, nil
}
