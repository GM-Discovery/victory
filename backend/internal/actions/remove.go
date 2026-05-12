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

type RemoveElementRequest struct {
	SessionID   string `json:"session_id"`
	ActorID     string `json:"actor_id"`
	ElementID   string `json:"element_id"`
	ElementSlug string `json:"element_slug"`
}

func StoreRemoveElement(ctx context.Context, pool *pgxpool.Pool, req RemoveElementRequest) (*StoredAction, error) {
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

	decision, err := CanAct(ctx, tx, req.ActorID, "act/remove_element", req.SessionID, ActionTarget{
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
	if state.VenueSlug != "the-cave" {
		return nil, errors.New("unknown_target")
	}
	if strings.ToLower(strings.TrimSpace(state.Surface)) != "stage" {
		return nil, errors.New("unknown_target")
	}

	displayName, handle, role, persona, err := loadActorIdentity(ctx, tx, req.SessionID, req.ActorID)
	if err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM venue_layout_elements
		WHERE venue_id = $1
		  AND element_id = $2
	`, state.VenueID, state.ElementID); err != nil {
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
		"removed_at":    time.Now().UTC().Format(time.RFC3339),
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
			showing_id,
			moment_id,
			actor_id,
			type,
			target,
			payload,
			scope,
			visibility,
			recorded
		)
		VALUES ($1, $2, $3, $4, 'act/remove_element', $5, $6, $7, $8, TRUE)
		RETURNING id, ts
	`, req.SessionID, showing.ID, nextMoment, req.ActorID, targetJSON, payloadJSON, scopeJSON, visibilityJSON).
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
	out.Type = "act/remove_element"
	out.Target = target
	out.Payload = payload
	out.Scope = scope
	out.Visibility = visibility
	out.Timestamp = ts.UTC().Format(time.RFC3339)

	return &out, nil
}
