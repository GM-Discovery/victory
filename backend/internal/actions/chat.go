package actions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showings"
)

type ChatMessageRequest struct {
	SessionID string `json:"session_id"`
	ActorID   string `json:"actor_id"`
	Text      string `json:"text"`
}

func StoreChatMessage(ctx context.Context, pool *pgxpool.Pool, req ChatMessageRequest) (*StoredAction, error) {
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ActorID = strings.TrimSpace(req.ActorID)
	req.Text = strings.TrimSpace(req.Text)

	if req.SessionID == "" {
		return nil, errors.New("session_id is required")
	}
	if req.ActorID == "" {
		return nil, errors.New("actor_id is required")
	}
	if req.Text == "" {
		return nil, errors.New("text is required")
	}
	if utf8.RuneCountInString(req.Text) > 250 {
		return nil, errors.New("message_too_long")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	decision, err := CanAct(ctx, tx, req.ActorID, "chat/message", req.SessionID, ActionTarget{Kind: "session"})
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
	payload := map[string]any{
		"text":          req.Text,
		"actor_persona": persona,
	}
	scope := map[string]any{
		"surfaces":         []string{"chat"},
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
		VALUES ($1, $2, $3, 'chat/message', $4, $5, $6, $7, $8, TRUE)
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
	out.Type = "chat/message"
	out.Target = target
	out.Payload = payload
	out.Scope = scope
	out.Visibility = visibility
	out.Timestamp = ts.UTC().Format(time.RFC3339)

	return &out, nil
}

func canActChatMessage(ctx context.Context, q actionQuerier, userID, sessionID string) (Decision, error) {
	var (
		participantRole string
		chatEnabled     bool
		talkingEnabled  bool
	)

	err := q.QueryRow(ctx, `
		SELECT
			sp.role::text,
			COALESCE((v.config ->> 'chat_enabled')::boolean, FALSE),
			COALESCE((v.config ->> 'talking_enabled')::boolean, FALSE)
		FROM sessions s
		JOIN venues v ON v.id = s.venue_id
		JOIN session_participants sp ON sp.session_id = s.id
		WHERE s.id = $1
		  AND sp.user_id = $2
		LIMIT 1
	`, sessionID, userID).Scan(&participantRole, &chatEnabled, &talkingEnabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Decision{Allowed: false, Reason: "not_session_participant"}, nil
		}
		return Decision{}, err
	}

	if !chatEnabled {
		return Decision{Allowed: false, Reason: "policy_denied"}, nil
	}

	if talkingEnabled {
		return Decision{Allowed: true, Reason: "allowed"}, nil
	}

	switch normalizeActionRole(participantRole) {
	case "producer", "director", "cast", "crew":
		return Decision{Allowed: true, Reason: "allowed"}, nil
	default:
		return Decision{Allowed: false, Reason: "insufficient_role"}, nil
	}
}
