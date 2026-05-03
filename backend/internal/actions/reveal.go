package actions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RevealRequest struct {
	SessionID   string `json:"session_id"`
	ActorID     string `json:"actor_id"`
	ElementID   string `json:"element_id"`
	ElementSlug string `json:"element_slug"`
	Layer       string `json:"layer"`   // "actor" or "audience"
	Visible     bool   `json:"visible"` // true = reveal, false = hide
}

func StoreReveal(ctx context.Context, pool *pgxpool.Pool, req RevealRequest) (*StoredAction, error) {
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
	if req.Layer != "actor" && req.Layer != "audience" {
		return nil, errors.New("invalid layer")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var joined bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM session_participants
			WHERE session_id = $1 AND user_id = $2
		)
	`, req.SessionID, req.ActorID).Scan(&joined)
	if err != nil {
		return nil, err
	}
	if !joined {
		return nil, errors.New("actor is not joined to session")
	}

	resolvedID, resolvedSlug, resolvedType, resolvedSurface, err := resolveRevealTarget(ctx, tx, req.SessionID, req.ElementID, req.ElementSlug)
	if err != nil {
		return nil, err
	}

	if !isRevealableElement(resolvedType, resolvedSurface, resolvedSlug) {
		return nil, errors.New("element is not revealable")
	}

	decision, err := CanAct(ctx, tx, req.ActorID, actionTypeForReveal(req.Visible), req.SessionID, ActionTarget{
		Kind:        "element",
		ElementID:   resolvedID,
		ElementSlug: resolvedSlug,
		Layer:       req.Layer,
	})
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
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(moment_id), 0) + 1
		FROM actions
		WHERE session_id = $1
	`, req.SessionID).Scan(&nextMoment)
	if err != nil {
		return nil, err
	}

	target := map[string]any{
		"element_id":   resolvedID,
		"element_slug": resolvedSlug,
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
	out.Target = target
	out.Payload = payload
	out.Scope = scope
	out.Visibility = visibility
	out.Timestamp = ts.UTC().Format(time.RFC3339)

	return &out, nil
}

func actionTypeForReveal(visible bool) string {
	if visible {
		return "act/reveal_element"
	}

	return "act/hide_element"
}

func resolveRevealTarget(ctx context.Context, tx pgx.Tx, sessionID, elementID, elementSlug string) (resolvedID, resolvedSlug, elementType, surface string, err error) {
	base := `
		SELECT
			e.id,
			e.slug,
			e.element_type,
			vle.surface
		FROM sessions s
		JOIN venues v ON v.id = s.venue_id
		JOIN venue_layout_elements vle ON vle.venue_id = v.id
		JOIN elements e ON e.id = vle.element_id
		WHERE s.id = $1
	`

	switch {
	case elementID != "":
		err = tx.QueryRow(ctx, base+` AND e.id = $2 LIMIT 1`, sessionID, elementID).
			Scan(&resolvedID, &resolvedSlug, &elementType, &surface)
	case elementSlug != "":
		err = tx.QueryRow(ctx, base+` AND e.slug = $2 LIMIT 1`, sessionID, elementSlug).
			Scan(&resolvedID, &resolvedSlug, &elementType, &surface)
	default:
		return "", "", "", "", errors.New("element_id or element_slug is required")
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", "", "", errors.New("target element not found in session venue")
		}
		return "", "", "", "", err
	}

	return resolvedID, resolvedSlug, elementType, surface, nil
}

func isRevealableElement(elementType, surface, slug string) bool {
	elementType = strings.TrimSpace(strings.ToLower(elementType))
	surface = strings.TrimSpace(strings.ToLower(surface))
	_ = strings.TrimSpace(strings.ToLower(slug))

	if surface == "" {
		return false
	}

	switch elementType {
	case "image", "prop", "set_piece", "overlay", "html", "text", "panel", "index_card":
		return true
	default:
		return surface == "stage" || surface == "overlay"
	}
}
