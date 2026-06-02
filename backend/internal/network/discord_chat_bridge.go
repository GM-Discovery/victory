package network

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"victory/backend/internal/actions"
	"victory/backend/internal/identity"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const discordChatBridgeMaxContentRunes = 2000

var discordChatBridgeEligibleVenues = map[string]string{
	"the-cave":            "The Cave",
	"first-theater":       "First Theater",
	"middle-school-stage": "Middle School Stage",
}

type discordSessionThreadMirrorRow struct {
	DiscordServerID string
	ThreadID        string
	Status          string
}

type discordChatMessageBridgeRow struct {
	ActionID         string
	LocationID       string
	SessionID        string
	ShowingID        string
	VenueSlug        string
	DiscordServerID  string
	DiscordThreadID  string
	DiscordMessageID string
	BridgeStatus     string
	ErrorCode        string
	ErrorMessage     string
	AttemptedAt      time.Time
	MirroredAt       *time.Time
}

type discordCreatedMessage struct {
	ID string `json:"id"`
}

func mirrorVictoryChatToDiscord(ctx context.Context, pool *pgxpool.Pool, cfg identity.DiscordServerLinkConfig, action *actions.StoredAction) {
	if pool == nil || action == nil || !strings.EqualFold(strings.TrimSpace(action.Type), "chat/message") {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	runtimeCfg, err := identity.ResolveDiscordServerLinkRuntimeConfig(ctx, pool, cfg)
	if err != nil || !identity.DiscordServerLinkConfigured(runtimeCfg) {
		return
	}

	sessionVenue, locationID, err := loadChatBridgeVenueContext(ctx, pool, action.SessionID)
	if err != nil {
		return
	}
	if _, ok := discordChatBridgeEligibleVenues[sessionVenue]; !ok {
		return
	}

	threadRow, err := loadActiveDiscordSessionThread(ctx, pool, locationID, sessionVenue)
	if err != nil || threadRow == nil || strings.TrimSpace(threadRow.ThreadID) == "" || !strings.EqualFold(strings.TrimSpace(threadRow.Status), "active") {
		return
	}

	content, truncated := formatDiscordChatMirrorContent(action)
	if content == "" {
		return
	}

	now := time.Now().UTC()
	_ = upsertDiscordChatMessageBridge(ctx, pool, discordChatMessageBridgeRow{
		ActionID:        action.ID,
		LocationID:      locationID,
		SessionID:       action.SessionID,
		ShowingID:       action.ShowingID,
		VenueSlug:       sessionVenue,
		DiscordServerID: threadRow.DiscordServerID,
		DiscordThreadID: threadRow.ThreadID,
		BridgeStatus:    "pending",
		AttemptedAt:     now,
	})

	messageID, err := postDiscordThreadMessage(ctx, runtimeCfg, threadRow.ThreadID, content)
	if err != nil {
		updateDiscordChatMessageBridge(ctx, pool, discordChatMessageBridgeRow{
			ActionID:        action.ID,
			LocationID:      locationID,
			SessionID:       action.SessionID,
			ShowingID:       action.ShowingID,
			VenueSlug:       sessionVenue,
			DiscordServerID: threadRow.DiscordServerID,
			DiscordThreadID: threadRow.ThreadID,
			BridgeStatus:    "failed",
			ErrorCode:       discordBridgeErrorCode(err),
			ErrorMessage:    err.Error(),
			AttemptedAt:     now,
			MirroredAt:      &now,
		})
		log.Printf("discord chat bridge failed: action=%s session=%s venue=%s err=%v", action.ID, action.SessionID, sessionVenue, err)
		return
	}

	status := "sent"
	errorCode := ""
	if truncated {
		status = "sent_truncated"
		errorCode = "content_truncated"
	}
	updateDiscordChatMessageBridge(ctx, pool, discordChatMessageBridgeRow{
		ActionID:         action.ID,
		LocationID:       locationID,
		SessionID:        action.SessionID,
		ShowingID:        action.ShowingID,
		VenueSlug:        sessionVenue,
		DiscordServerID:  threadRow.DiscordServerID,
		DiscordThreadID:  threadRow.ThreadID,
		DiscordMessageID: messageID,
		BridgeStatus:     status,
		ErrorCode:        errorCode,
		AttemptedAt:      now,
		MirroredAt:       &now,
	})
}

func loadChatBridgeVenueContext(ctx context.Context, pool *pgxpool.Pool, sessionID string) (venueSlug, locationID string, err error) {
	err = pool.QueryRow(ctx, `
		SELECT
			v.slug,
			l.id::text
		FROM sessions s
		JOIN venues v ON v.id = s.venue_id
		JOIN lots lo ON lo.id = v.lot_id
		JOIN locations l ON l.id = lo.location_id
		WHERE s.id = $1::uuid
		LIMIT 1
	`, strings.TrimSpace(sessionID)).Scan(&venueSlug, &locationID)
	if err != nil {
		return "", "", err
	}
	return strings.ToLower(strings.TrimSpace(venueSlug)), strings.TrimSpace(locationID), nil
}

func loadActiveDiscordSessionThread(ctx context.Context, pool *pgxpool.Pool, locationID, venueSlug string) (*discordSessionThreadMirrorRow, error) {
	var row discordSessionThreadMirrorRow
	err := pool.QueryRow(ctx, `
		SELECT discord_server_id, thread_id, status
		FROM auth.discord_session_threads
		WHERE location_id = $1::uuid
		  AND venue_slug = $2
		  AND status = 'active'
		LIMIT 1
	`, strings.TrimSpace(locationID), strings.TrimSpace(venueSlug)).Scan(&row.DiscordServerID, &row.ThreadID, &row.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func postDiscordThreadMessage(ctx context.Context, cfg identity.DiscordServerLinkConfig, threadID, content string) (string, error) {
	var created discordCreatedMessage
	if err := identity.DiscordServerLinkRequest(ctx, cfg, http.MethodPost, "/channels/"+urlPathEscape(strings.TrimSpace(threadID))+"/messages", map[string]any{
		"content": content,
	}, &created); err != nil {
		return "", err
	}
	return strings.TrimSpace(created.ID), nil
}

func upsertDiscordChatMessageBridge(ctx context.Context, pool *pgxpool.Pool, row discordChatMessageBridgeRow) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO auth.discord_chat_message_bridges (
			action_id,
			location_id,
			session_id,
			showing_id,
			venue_slug,
			discord_server_id,
			discord_thread_id,
			discord_message_id,
			bridge_status,
			error_code,
			error_message,
			attempted_at,
			mirrored_at,
			updated_at
		)
		VALUES (
			$1::uuid,
			$2::uuid,
			$3::uuid,
			NULLIF($4, '')::uuid,
			$5,
			$6,
			$7,
			NULLIF($8, ''),
			$9,
			NULLIF($10, ''),
			NULLIF($11, ''),
			$12,
			$13,
			NOW()
		)
		ON CONFLICT (action_id) DO UPDATE
		SET location_id = EXCLUDED.location_id,
			session_id = EXCLUDED.session_id,
			showing_id = EXCLUDED.showing_id,
			venue_slug = EXCLUDED.venue_slug,
			discord_server_id = EXCLUDED.discord_server_id,
			discord_thread_id = EXCLUDED.discord_thread_id,
			discord_message_id = EXCLUDED.discord_message_id,
			bridge_status = EXCLUDED.bridge_status,
			error_code = EXCLUDED.error_code,
			error_message = EXCLUDED.error_message,
			attempted_at = EXCLUDED.attempted_at,
			mirrored_at = EXCLUDED.mirrored_at,
			updated_at = NOW()
	`, row.ActionID, row.LocationID, row.SessionID, row.ShowingID, row.VenueSlug, row.DiscordServerID, row.DiscordThreadID, row.DiscordMessageID, row.BridgeStatus, row.ErrorCode, row.ErrorMessage, row.AttemptedAt, row.MirroredAt)
	return err
}

func updateDiscordChatMessageBridge(ctx context.Context, pool *pgxpool.Pool, row discordChatMessageBridgeRow) {
	_ = upsertDiscordChatMessageBridge(ctx, pool, row)
}

func discordBridgeErrorCode(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "unauthorized"):
		return "discord_unauthorized"
	case strings.Contains(msg, "forbidden"):
		return "discord_forbidden"
	case strings.Contains(msg, "rate limit"):
		return "discord_rate_limited"
	default:
		return "discord_post_failed"
	}
}

func formatDiscordChatMirrorContent(action *actions.StoredAction) (string, bool) {
	if action == nil {
		return "", false
	}

	text := ""
	if action.Payload != nil {
		if raw, ok := action.Payload["text"].(string); ok {
			text = strings.TrimSpace(raw)
		}
	}
	if text == "" {
		return "", false
	}

	name := strings.TrimSpace(action.ActorDisplayName)
	if name == "" {
		name = strings.TrimSpace(action.ActorHandle)
	}
	if name == "" {
		name = shortChatBridgeUserID(action.ActorID)
	}

	persona := personaBridgeDisplayName(action.Persona)
	if persona != "" && !strings.EqualFold(persona, name) {
		name = fmt.Sprintf("%s / %s", name, persona)
	}

	if role := roleBridgeLabel(action.ActorRole); role != "" {
		name = fmt.Sprintf("%s [%s]", name, role)
	}

	prefix := name + ": "
	content := prefix + text
	if utf8.RuneCountInString(content) <= discordChatBridgeMaxContentRunes {
		return content, false
	}

	available := discordChatBridgeMaxContentRunes - utf8.RuneCountInString(prefix) - 3
	if available < 0 {
		return truncateRunes(prefix, discordChatBridgeMaxContentRunes), true
	}

	return prefix + truncateRunes(text, available) + "...", true
}

func personaBridgeDisplayName(persona any) string {
	m, ok := persona.(map[string]any)
	if !ok {
		return ""
	}
	for _, key := range []string{"display_name", "name", "handle"} {
		if raw, ok := m[key].(string); ok {
			if trimmed := strings.TrimSpace(raw); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func roleBridgeLabel(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "producer":
		return "Producer"
	case "director":
		return "Director"
	case "cast", "actor":
		return "Cast"
	case "crew":
		return "Crew"
	case "audience":
		return "Audience"
	default:
		return ""
	}
}

func shortChatBridgeUserID(userID string) string {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "Unknown"
	}
	if len(userID) <= 8 {
		return userID
	}
	return userID[:8]
}

func truncateRunes(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= limit {
		return s
	}

	var b strings.Builder
	b.Grow(limit)
	count := 0
	for _, r := range s {
		if count >= limit {
			break
		}
		b.WriteRune(r)
		count++
	}
	return b.String()
}

func urlPathEscape(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(value), "/", "%2F"), " ", "%20")
}
