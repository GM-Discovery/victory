package actions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/identity"
	"victory/backend/internal/showings"
)

var allowedReactions = map[string]struct{}{
	"clap":          {},
	"boo":           {},
	"cry":           {},
	"smile":         {},
	"laugh":         {},
	"startle":       {},
	"heart":         {},
	"thumbs_up":     {},
	"thumbs_down":   {},
	"cheer":         {},
	"standing_clap": {},
}

type ReactRequest struct {
	SessionID string `json:"session_id"`
	ActorID   string `json:"actor_id"`
	Kind      string `json:"kind"`
}

type StoredAction struct {
	ID               string         `json:"id"`
	SessionID        string         `json:"session_id"`
	ShowingID        string         `json:"showing_id,omitempty"`
	MomentID         int64          `json:"moment_id"`
	ActorID          string         `json:"actor_id"`
	ActorDisplayName string         `json:"actor_display_name,omitempty"`
	ActorHandle      string         `json:"actor_handle,omitempty"`
	ActorRole        string         `json:"actor_role,omitempty"`
	Actor            map[string]any `json:"actor,omitempty"`
	Persona          any            `json:"persona"`
	Type             string         `json:"type"`
	Target           map[string]any `json:"target"`
	Payload          map[string]any `json:"payload"`
	Scope            map[string]any `json:"scope"`
	Visibility       map[string]any `json:"visibility"`
	Timestamp        string         `json:"ts"`
}

func StoreReaction(ctx context.Context, pool *pgxpool.Pool, req ReactRequest) (*StoredAction, error) {
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ActorID = strings.TrimSpace(req.ActorID)
	req.Kind = strings.TrimSpace(req.Kind)

	if req.SessionID == "" {
		return nil, errors.New("session_id is required")
	}
	if req.ActorID == "" {
		return nil, errors.New("actor_id is required")
	}
	if _, ok := allowedReactions[req.Kind]; !ok {
		return nil, errors.New("invalid reaction kind")
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

	showing, err := showings.EnsureForSession(ctx, tx, req.SessionID, req.ActorID)
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
		"kind": "session",
		"id":   req.SessionID,
	}
	displayName, handle, role, persona, err := loadActorIdentity(ctx, tx, req.SessionID, req.ActorID)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"kind":          req.Kind,
		"actor_persona": persona,
	}
	scope := map[string]any{
		"surfaces":         []string{"overlay"},
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
			showing_id,
			recorded
		)
		VALUES ($1, $2, $3, 'react/emote', $4, $5, $6, $7, $8, TRUE)
		RETURNING id, ts
	`, req.SessionID, nextMoment, req.ActorID, targetJSON, payloadJSON, scopeJSON, visibilityJSON, showing.ID).
		Scan(&out.ID, &ts)
	if err != nil {
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
	out.Type = "react/emote"
	out.Target = target
	out.Payload = payload
	out.Scope = scope
	out.Visibility = visibility
	out.Timestamp = ts.UTC().Format(time.RFC3339)

	return &out, nil
}

func StoreSpeak(ctx context.Context, pool *pgxpool.Pool, sessionID, actorID, text string) (*StoredAction, error) {
	sessionID = strings.TrimSpace(sessionID)
	actorID = strings.TrimSpace(actorID)
	text = strings.TrimSpace(text)

	if sessionID == "" {
		return nil, errors.New("session_id is required")
	}
	if actorID == "" {
		return nil, errors.New("actor_id is required")
	}
	if text == "" {
		return nil, errors.New("text is required")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var exists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM session_participants
			WHERE session_id = $1 AND user_id = $2
		)
	`, sessionID, actorID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("actor is not joined to session")
	}

	showing, err := showings.EnsureForSession(ctx, tx, sessionID, actorID)
	if err != nil {
		return nil, err
	}

	var nextMoment int64
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(moment_id), 0) + 1
		FROM actions
		WHERE session_id = $1
	`, sessionID).Scan(&nextMoment)
	if err != nil {
		return nil, err
	}

	displayName, handle, role, persona, err := loadActorIdentity(ctx, tx, sessionID, actorID)
	if err != nil {
		return nil, err
	}

	target := map[string]any{
		"kind": "session",
		"id":   sessionID,
	}
	payload := map[string]any{
		"text":          text,
		"actor_persona": persona,
	}
	scope := map[string]any{
		"surfaces":         []string{"overlay"},
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
			showing_id,
			recorded
		)
		VALUES ($1, $2, $3, 'perform/speak', $4, $5, $6, $7, $8, TRUE)
		RETURNING id, ts
	`, sessionID, nextMoment, actorID, targetJSON, payloadJSON, scopeJSON, visibilityJSON, showing.ID).
		Scan(&out.ID, &ts)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	out.SessionID = sessionID
	out.ShowingID = showing.ID
	out.MomentID = nextMoment
	out.ActorID = actorID
	out.ActorDisplayName = displayName
	out.ActorHandle = handle
	out.ActorRole = role
	out.Actor = map[string]any{
		"user_id":      actorID,
		"handle":       handle,
		"display_name": displayName,
		"role":         role,
		"persona":      persona,
	}
	out.Persona = persona
	out.Type = "perform/speak"
	out.Target = target
	out.Payload = payload
	out.Scope = scope
	out.Visibility = visibility
	out.Timestamp = ts.UTC().Format(time.RFC3339)

	return &out, nil
}

func loadActorIdentity(ctx context.Context, q actionQuerier, sessionID, actorID string) (displayName, handle, role string, persona any, err error) {
	ident, err := identity.ResolveSessionIdentity(ctx, q, sessionID, actorID)
	if err != nil {
		return "", "", "", nil, err
	}

	return ident.DisplayName, ident.Handle, ident.Role, ident.Persona, nil
}
