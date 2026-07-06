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

// GameEventRequest is the venue mirror of a canonical character event
// (Kernel 60 §9). The canonical History row (character_workbook_entries)
// is always written first; this mirror never rolls back the underlying
// skill mutation if it fails (Kernel 59 §16.4 semantics, Kernel 60 §12).
type GameEventRequest struct {
	SessionID              string         `json:"session_id"`
	ActorID                string         `json:"actor_id"`
	EventKind              string         `json:"event_kind"`
	SourceCharacterEventID string         `json:"source_character_event_id"`
	CharacterCardID        string         `json:"character_card_id"`
	CharacterName          string         `json:"character_name"`
	Detail                 map[string]any `json:"detail"`
}

func StoreGameEvent(ctx context.Context, pool *pgxpool.Pool, req GameEventRequest) (*StoredAction, error) {
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ActorID = strings.TrimSpace(req.ActorID)
	req.EventKind = strings.TrimSpace(req.EventKind)
	req.SourceCharacterEventID = strings.TrimSpace(req.SourceCharacterEventID)
	req.CharacterCardID = strings.TrimSpace(req.CharacterCardID)
	req.CharacterName = strings.TrimSpace(req.CharacterName)

	if req.SessionID == "" {
		return nil, errors.New("session_id is required")
	}
	if req.ActorID == "" {
		return nil, errors.New("actor_id is required")
	}
	if req.EventKind == "" {
		return nil, errors.New("event_kind is required")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	decision, err := CanAct(ctx, tx, req.ActorID, "game/event", req.SessionID, ActionTarget{Kind: "session"})
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
		"kind": "session",
		"id":   req.SessionID,
	}
	detail := req.Detail
	if detail == nil {
		detail = map[string]any{}
	}
	payload := map[string]any{
		"event_kind":                req.EventKind,
		"source_character_event_id": req.SourceCharacterEventID,
		"character_card_id":         req.CharacterCardID,
		"character_name":            req.CharacterName,
		"actor_persona":             persona,
		"detail":                    detail,
	}
	scope := map[string]any{
		"surfaces":         []string{"game_events"},
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
		VALUES ($1, $2, $3, 'game/event', $4, $5, $6, $7, $8, TRUE)
		RETURNING id, ts
	`, req.SessionID, nextMoment, req.ActorID, targetJSON, payloadJSON, scopeJSON, visibilityJSON, showing.ID).
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
	out.Type = "game/event"
	out.Target = target
	out.Payload = payload
	out.Scope = scope
	out.Visibility = visibility
	out.Timestamp = ts.UTC().Format(time.RFC3339)

	return &out, nil
}
