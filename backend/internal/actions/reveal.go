package actions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RevealRequest struct {
	SessionID   string `json:"session_id"`
	ActorID     string `json:"actor_id"`
	ElementSlug string `json:"element_slug"`
	Layer       string `json:"layer"`   // "actor" or "audience"
	Visible     bool   `json:"visible"` // true = reveal, false = hide
}

func StoreReveal(ctx context.Context, pool *pgxpool.Pool, req RevealRequest) (*StoredAction, error) {
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ActorID = strings.TrimSpace(req.ActorID)
	req.ElementSlug = strings.TrimSpace(req.ElementSlug)
	req.Layer = strings.TrimSpace(req.Layer)

	if req.SessionID == "" {
		return nil, errors.New("session_id is required")
	}
	if req.ActorID == "" {
		return nil, errors.New("actor_id is required")
	}
	if req.ElementSlug != "first-fire" {
		return nil, errors.New("unsupported element")
	}
	if req.Layer != "actor" && req.Layer != "audience" {
		return nil, errors.New("invalid layer")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var exists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM session_participants
			WHERE session_id = $1 AND user_id = $2
		)
	`, req.SessionID, req.ActorID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("actor is not joined to session")
	}

	var nextMoment int64
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(moment_id), 0) + 1
		FROM actions
		WHERE session_id = $1
	`, req.SessionID).Scan(&nextMoment)
	if err != nil {
		return nil, err
	}

	target := map[string]any{
		"element_slug": req.ElementSlug,
		"layer":        req.Layer,
	}

	payload := map[string]any{
		"visible": req.Visible,
	}

	scope := map[string]any{
		"surfaces":         []string{"stage"},
		"audienceSegments": []string{"all"},
	}

	visibility := map[string]any{
		"toRoles":   []string{"audience", "cast", "crew", "director", "producer"},
		"privateTo": []string{},
	}

	targetJSON, _ := json.Marshal(target)
	payloadJSON, _ := json.Marshal(payload)
	scopeJSON, _ := json.Marshal(scope)
	visibilityJSON, _ := json.Marshal(visibility)

	actionType := "act/hide_element"
	if req.Visible {
		actionType = "act/reveal_element"
	}

	var out StoredAction
	var ts time.Time

	err = tx.QueryRow(ctx, `
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
		Scan(&out.ID, &ts)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	out.SessionID = req.SessionID
	out.MomentID = nextMoment
	out.ActorID = req.ActorID
	out.Type = actionType
	out.Target = target
	out.Payload = payload
	out.Scope = scope
	out.Visibility = visibility
	out.Timestamp = ts.UTC().Format(time.RFC3339)

	return &out, nil
}
