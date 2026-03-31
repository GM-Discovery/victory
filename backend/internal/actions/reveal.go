package actions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RevealRequest struct {
	SessionID   string `json:"session_id"`
	ActorID     string `json:"actor_id"`
	ElementSlug string `json:"element_slug"`
	Reveal      bool   `json:"reveal"`
}

func StoreRevealAction(ctx context.Context, pool *pgxpool.Pool, req RevealRequest, actorCanReveal bool) (*StoredAction, error) {
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ActorID = strings.TrimSpace(req.ActorID)
	req.ElementSlug = strings.TrimSpace(req.ElementSlug)

	if req.SessionID == "" {
		return nil, errors.New("session_id is required")
	}
	if req.ActorID == "" {
		return nil, errors.New("actor_id is required")
	}
	if req.ElementSlug == "" {
		return nil, errors.New("element_slug is required")
	}
	if req.ElementSlug != "first-fire" {
		return nil, errors.New("unsupported element_slug")
	}

	var joined bool
	var role string
	var nextMoment int64

	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM session_actors
			WHERE session_id = $1
			  AND actor_id = $2
		)
	`, req.SessionID, req.ActorID).Scan(&joined)
	if err != nil {
		return nil, err
	}
	if !joined {
		return nil, errors.New("actor is not joined to session")
	}

	err = pool.QueryRow(ctx, `
		SELECT COALESCE(
			(
				SELECT lower(m.role)
				FROM memberships m
				JOIN sessions s ON s.venue_id = m.venue_id
				JOIN session_actors sa ON sa.user_id = m.user_id
				WHERE s.id = $1
				  AND sa.session_id = $1
				  AND sa.actor_id = $2
				ORDER BY m.created_at DESC
				LIMIT 1
			),
			''
		)
	`, req.SessionID, req.ActorID).Scan(&role)
	if err != nil {
		return nil, err
	}

	switch role {
	case "producer", "director":
	case "actor":
		if !actorCanReveal {
			return nil, errors.New("actors_cannot_reveal")
		}
	default:
		return nil, errors.New("forbidden")
	}

	err = pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(moment_id), 0) + 1
		FROM actions
		WHERE session_id = $1
	`, req.SessionID).Scan(&nextMoment)
	if err != nil {
		return nil, err
	}

	target := map[string]any{
		"element_slug": req.ElementSlug,
	}
	payload := map[string]any{
		"visible_to_audience": req.Reveal,
	}
	scope := map[string]any{
		"audienceSegments": []string{"all"},
	}
	visibility := map[string]any{
		"toRoles": []string{"audience", "cast", "crew", "director", "producer"},
	}

	targetJSON, _ := json.Marshal(target)
	payloadJSON, _ := json.Marshal(payload)
	scopeJSON, _ := json.Marshal(scope)
	visibilityJSON, _ := json.Marshal(visibility)

	actionType := "direct/hide_element"
	if req.Reveal {
		actionType = "direct/reveal_element"
	}

	var out StoredAction
	err = pool.QueryRow(ctx, `
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
		RETURNING id, moment_id, ts
	`, req.SessionID, nextMoment, req.ActorID, actionType, targetJSON, payloadJSON, scopeJSON, visibilityJSON).
		Scan(&out.ID, &out.MomentID, &out.Timestamp)
	if err != nil {
		return nil, err
	}

	out.ActorID = req.ActorID
	out.Type = actionType
	out.Target = target
	out.Payload = payload
	out.Scope = scope
	out.Visibility = visibility

	return &out, nil
}