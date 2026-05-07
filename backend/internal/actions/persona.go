package actions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/characters"
	"victory/backend/internal/showings"
)

type PersonaRequest struct {
	SessionID       string `json:"session_id"`
	ActorID         string `json:"actor_id"`
	CharacterCardID string `json:"character_card_id"`
}

func StorePersonaEquip(ctx context.Context, pool *pgxpool.Pool, req PersonaRequest) (*StoredAction, error) {
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ActorID = strings.TrimSpace(req.ActorID)
	req.CharacterCardID = strings.TrimSpace(req.CharacterCardID)
	if req.SessionID == "" {
		return nil, errors.New("session_id is required")
	}
	if req.ActorID == "" {
		return nil, errors.New("actor_id is required")
	}
	if req.CharacterCardID == "" {
		return nil, errors.New("character_card_id is required")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	decision, err := CanAct(ctx, tx, req.ActorID, "persona/equip", req.SessionID, ActionTarget{
		Kind:      "persona",
		ElementID: req.CharacterCardID,
	})
	if err != nil {
		return nil, err
	}
	if !decision.Allowed {
		return nil, &ActionDeniedError{Reason: decision.Reason}
	}

	persona, err := characters.PersonaForCard(ctx, tx, req.ActorID, req.CharacterCardID)
	if err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO current_session_personas (session_id, user_id, character_card_id, equipped_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (session_id, user_id) DO UPDATE
		SET character_card_id = EXCLUDED.character_card_id,
		    equipped_at = NOW()
	`, req.SessionID, req.ActorID, req.CharacterCardID); err != nil {
		return nil, err
	}

	return storePersonaAction(ctx, tx, req.SessionID, req.ActorID, "persona/equip", persona)
}

func StorePersonaUnequip(ctx context.Context, pool *pgxpool.Pool, req PersonaRequest) (*StoredAction, error) {
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ActorID = strings.TrimSpace(req.ActorID)
	if req.SessionID == "" {
		return nil, errors.New("session_id is required")
	}
	if req.ActorID == "" {
		return nil, errors.New("actor_id is required")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	decision, err := CanAct(ctx, tx, req.ActorID, "persona/unequip", req.SessionID, ActionTarget{Kind: "persona"})
	if err != nil {
		return nil, err
	}
	if !decision.Allowed {
		return nil, &ActionDeniedError{Reason: decision.Reason}
	}

	var persona any
	var characterCardID string
	if err := tx.QueryRow(ctx, `
		SELECT character_card_id::text
		FROM current_session_personas
		WHERE session_id = $1
		  AND user_id = $2
		LIMIT 1
	`, req.SessionID, req.ActorID).Scan(&characterCardID); err == nil {
		persona, _ = characters.PersonaForCard(ctx, tx, req.ActorID, characterCardID)
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM current_session_personas
		WHERE session_id = $1
		  AND user_id = $2
	`, req.SessionID, req.ActorID); err != nil {
		return nil, err
	}

	return storePersonaAction(ctx, tx, req.SessionID, req.ActorID, "persona/unequip", persona)
}

func storePersonaAction(ctx context.Context, tx pgx.Tx, sessionID, actorID, actionType string, persona any) (*StoredAction, error) {
	showing, err := showings.EnsureForSession(ctx, tx, sessionID, actorID)
	if err != nil {
		return nil, err
	}

	displayName, handle, role, _, err := loadActorIdentity(ctx, tx, sessionID, actorID)
	if err != nil {
		return nil, err
	}

	var nextMoment int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(moment_id), 0) + 1
		FROM actions
		WHERE session_id = $1
	`, sessionID).Scan(&nextMoment); err != nil {
		return nil, err
	}

	target := map[string]any{
		"kind": "persona",
	}
	payload := map[string]any{
		"persona":       persona,
		"actor_persona": persona,
	}
	scope := map[string]any{
		"surfaces":         []string{"presence", "identity"},
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
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, TRUE)
		RETURNING id, ts
	`, sessionID, nextMoment, actorID, actionType, targetJSON, payloadJSON, scopeJSON, visibilityJSON, showing.ID).
		Scan(&out.ID, &ts); err != nil {
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
	out.Type = actionType
	out.Target = target
	out.Payload = payload
	out.Scope = scope
	out.Visibility = visibility
	out.Timestamp = ts.UTC().Format(time.RFC3339)

	return &out, nil
}
