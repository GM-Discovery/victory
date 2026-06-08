package actions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showings"
)

type DiscordChatMetadata struct {
	MessageID        string `json:"message_id"`
	ThreadID         string `json:"thread_id"`
	ChannelID        string `json:"channel_id"`
	ServerID         string `json:"server_id"`
	AuthorID         string `json:"author_id"`
	AuthorUsername   string `json:"author_username,omitempty"`
	AuthorGlobalName string `json:"author_global_name,omitempty"`
	LinkedUserID     string `json:"linked_user_id,omitempty"`
	Edited           bool   `json:"edited,omitempty"`
	EditOfMessageID  string `json:"edit_of_discord_message_id,omitempty"`
}

type DiscordChatMessageRequest struct {
	SessionID        string
	ActorID          string
	ActorDisplayName string
	ActorHandle      string
	ActorRole        string
	ActorPersona     any
	Text             string
	Discord          DiscordChatMetadata
	EditedAt         *time.Time
}

func StoreDiscordChatMessage(ctx context.Context, pool *pgxpool.Pool, req DiscordChatMessageRequest) (*StoredAction, error) {
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ActorID = strings.TrimSpace(req.ActorID)
	req.ActorDisplayName = strings.TrimSpace(req.ActorDisplayName)
	req.ActorHandle = strings.TrimSpace(req.ActorHandle)
	req.ActorRole = strings.TrimSpace(req.ActorRole)
	req.Text = strings.TrimSpace(req.Text)
	req.Discord.MessageID = strings.TrimSpace(req.Discord.MessageID)
	req.Discord.ThreadID = strings.TrimSpace(req.Discord.ThreadID)
	req.Discord.ChannelID = strings.TrimSpace(req.Discord.ChannelID)
	req.Discord.ServerID = strings.TrimSpace(req.Discord.ServerID)
	req.Discord.AuthorID = strings.TrimSpace(req.Discord.AuthorID)
	req.Discord.AuthorUsername = strings.TrimSpace(req.Discord.AuthorUsername)
	req.Discord.AuthorGlobalName = strings.TrimSpace(req.Discord.AuthorGlobalName)
	req.Discord.LinkedUserID = strings.TrimSpace(req.Discord.LinkedUserID)
	req.Discord.EditOfMessageID = strings.TrimSpace(req.Discord.EditOfMessageID)

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

	showing, err := showings.EnsureForSession(ctx, tx, req.SessionID, req.ActorID)
	if err != nil {
		var fallbackCreatedBy string
		if fallbackErr := tx.QueryRow(ctx, `
			SELECT sp.user_id::text
			FROM session_participants sp
			WHERE sp.session_id = $1
			ORDER BY sp.joined_at ASC
			LIMIT 1
		`, req.SessionID).Scan(&fallbackCreatedBy); fallbackErr == nil && strings.TrimSpace(fallbackCreatedBy) != "" {
			showing, err = showings.EnsureForSession(ctx, tx, req.SessionID, fallbackCreatedBy)
		}
		if err != nil {
			return nil, err
		}
	}

	var nextMoment int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(moment_id), 0) + 1
		FROM actions
		WHERE session_id = $1::uuid
	`, req.SessionID).Scan(&nextMoment); err != nil {
		return nil, err
	}

	target := map[string]any{
		"kind": "session",
		"id":   req.SessionID,
	}
	payload := map[string]any{
		"text":   req.Text,
		"source": "discord",
		"discord": map[string]any{
			"message_id":         req.Discord.MessageID,
			"thread_id":          req.Discord.ThreadID,
			"channel_id":         req.Discord.ChannelID,
			"server_id":          req.Discord.ServerID,
			"author_id":          req.Discord.AuthorID,
			"author_username":    req.Discord.AuthorUsername,
			"author_global_name": req.Discord.AuthorGlobalName,
			"linked_user_id":     req.Discord.LinkedUserID,
			"edited":             req.Discord.Edited,
			"edit_of_message_id": req.Discord.EditOfMessageID,
		},
		"actor_persona": req.ActorPersona,
	}
	if req.Discord.Edited {
		payload["edited"] = true
		if req.EditedAt != nil {
			payload["edited_at"] = req.EditedAt.UTC().Format(time.RFC3339)
		}
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
		VALUES ($1::uuid, $2, $3::uuid, 'chat/message', $4::jsonb, $5::jsonb, $6::jsonb, $7::jsonb, $8::uuid, TRUE)
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
	out.ActorDisplayName = req.ActorDisplayName
	out.ActorHandle = req.ActorHandle
	out.ActorRole = req.ActorRole
	out.Actor = map[string]any{
		"user_id":      req.ActorID,
		"handle":       req.ActorHandle,
		"display_name": req.ActorDisplayName,
		"role":         req.ActorRole,
		"persona":      req.ActorPersona,
		"source":       "discord",
		"discord":      req.Discord,
	}
	out.Persona = req.ActorPersona
	out.Type = "chat/message"
	out.Target = target
	out.Payload = payload
	out.Scope = scope
	out.Visibility = visibility
	out.Timestamp = ts.UTC().Format(time.RFC3339)

	return &out, nil
}
