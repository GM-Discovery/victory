package network

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"victory/backend/internal/db"
	"victory/backend/internal/identity"
	"victory/backend/internal/showings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDiscordGatewayMessageUpdateCreatesEditedAction(t *testing.T) {
	pool := openDiscordGatewayTestPool(t)
	ctx := context.Background()

	if err := identity.EnsureKernel39DiscordGatewaySurface(ctx, pool); err != nil {
		t.Fatalf("ensure gateway surface: %v", err)
	}

	locationID, venueID, lotID, productionID, userID, sessionID, showingID := setupDiscordGatewayEditFixture(t, pool)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_chat_imports WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_session_threads WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM session_participants WHERE session_id = $1`, sessionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM actions WHERE session_id = $1`, sessionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM showings WHERE id = $1`, showingID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, sessionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id = $1`, productionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM venues WHERE id = $1`, venueID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM lots WHERE id = $1`, lotID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM locations WHERE id = $1`, locationID)
	})

	hub := NewHub()
	linkRecord := discordGatewayLinkRecord{LocationID: locationID, DiscordGuildID: "guild-1", Active: true}
	baseMsg := discordGatewayMessage{
		ID:        "msg-1",
		ChannelID: "thread-1",
		GuildID:   "guild-1",
		Content:   "hello world",
		Type:      0,
		Author: &discordGatewayUser{
			ID:       "discord-user-1",
			Username: "grant",
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	if err := handleDiscordGatewayMessageCreate(ctx, pool, hub, identity.DiscordServerLinkConfig{}, locationID, linkRecord, identity.DiscordGatewayConfig{}, "", baseMsg); err != nil {
		t.Fatalf("create import: %v", err)
	}

	editedMsg := baseMsg
	editedMsg.Content = "hello edited"
	editedMsg.EditedTimestamp = time.Now().UTC().Format(time.RFC3339)
	if err := handleDiscordGatewayMessageUpdate(ctx, pool, hub, identity.DiscordServerLinkConfig{}, locationID, linkRecord, identity.DiscordGatewayConfig{}, editedMsg); err != nil {
		t.Fatalf("update import: %v", err)
	}

	var importStatus string
	var editActionID sql.NullString
	var messageEditedAt sql.NullTime
	if err := pool.QueryRow(ctx, `
		SELECT import_status, edit_action_id, message_edited_at
		FROM auth.discord_chat_imports
		WHERE discord_message_id = $1
	`, baseMsg.ID).Scan(&importStatus, &editActionID, &messageEditedAt); err != nil {
		t.Fatalf("load import row: %v", err)
	}
	if importStatus != "edited" || !editActionID.Valid || !messageEditedAt.Valid {
		t.Fatalf("unexpected import edit state: status=%s edit_action=%v edited_at=%v", importStatus, editActionID.Valid, messageEditedAt.Valid)
	}

	var originalText string
	if err := pool.QueryRow(ctx, `SELECT payload ->> 'text' FROM actions WHERE session_id = $1 ORDER BY moment_id ASC LIMIT 1`, sessionID).Scan(&originalText); err != nil {
		t.Fatalf("load original action: %v", err)
	}
	if originalText != "hello world" {
		t.Fatalf("unexpected original action text %q", originalText)
	}

	var editText string
	var edited bool
	if err := pool.QueryRow(ctx, `SELECT payload ->> 'text', COALESCE((payload ->> 'edited')::boolean, FALSE) FROM actions WHERE id = $1::uuid`, editActionID.String).Scan(&editText, &edited); err != nil {
		t.Fatalf("load edit action: %v", err)
	}
	if editText != "hello edited" || !edited {
		t.Fatalf("unexpected edit action payload text=%q edited=%v", editText, edited)
	}
}

func TestDiscordGatewayMessageCreateBackfillsSparseGatewayPayload(t *testing.T) {
	pool := openDiscordGatewayTestPool(t)
	ctx := context.Background()

	if err := identity.EnsureKernel39DiscordGatewaySurface(ctx, pool); err != nil {
		t.Fatalf("ensure gateway surface: %v", err)
	}

	locationID, venueID, lotID, productionID, userID, sessionID, showingID := setupDiscordGatewayEditFixture(t, pool)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_chat_imports WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_session_threads WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM session_participants WHERE session_id = $1`, sessionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM actions WHERE session_id = $1`, sessionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM showings WHERE id = $1`, showingID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, sessionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id = $1`, productionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM venues WHERE id = $1`, venueID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM lots WHERE id = $1`, lotID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM locations WHERE id = $1`, locationID)
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method %s", r.Method)
			http.Error(w, "bad method", http.StatusBadRequest)
			return
		}
		if r.URL.Path != "/channels/thread-1/messages/msg-1" {
			t.Errorf("unexpected fetch path %s", r.URL.Path)
			http.Error(w, "bad path", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":         "msg-1",
			"channel_id": "thread-1",
			"guild_id":   "guild-1",
			"content":    "pong from discord",
			"type":       0,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
			"author": map[string]any{
				"id":       "discord-user-1",
				"username": "grant",
			},
		})
	}))
	defer server.Close()

	hub := NewHub()
	linkRecord := discordGatewayLinkRecord{LocationID: locationID, DiscordGuildID: "guild-1", Active: true}
	msg := discordGatewayMessage{
		ID:        "msg-1",
		ChannelID: "thread-1",
		GuildID:   "",
		Content:   "",
		Type:      0,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	cfg := identity.DiscordServerLinkConfig{BotToken: "bot-token", APIBaseURL: server.URL}
	if err := handleDiscordGatewayMessageCreate(ctx, pool, hub, cfg, locationID, linkRecord, identity.DiscordGatewayConfig{}, "", msg); err != nil {
		t.Fatalf("create import: %v", err)
	}

	var text string
	if err := pool.QueryRow(ctx, `
		SELECT payload ->> 'text'
		FROM actions
		WHERE session_id = $1
		ORDER BY moment_id DESC
		LIMIT 1
	`, sessionID).Scan(&text); err != nil {
		t.Fatalf("load imported action: %v", err)
	}
	if text != "pong from discord" {
		t.Fatalf("unexpected imported text %q", text)
	}
}

func TestDiscordGatewayMessageCreateFallsBackWhenLinkedUserNotInSession(t *testing.T) {
	pool := openDiscordGatewayTestPool(t)
	ctx := context.Background()

	if err := identity.EnsureKernel39DiscordGatewaySurface(ctx, pool); err != nil {
		t.Fatalf("ensure gateway surface: %v", err)
	}

	locationID, venueID, lotID, productionID, userID, sessionID, showingID := setupDiscordGatewayEditFixture(t, pool)
	var linkedUserID string
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_chat_imports WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_session_threads WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_identities WHERE discord_user_id = $1`, "linked-discord-user-1")
		_, _ = pool.Exec(context.Background(), `DELETE FROM session_participants WHERE session_id = $1`, sessionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM actions WHERE session_id = $1`, sessionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM showings WHERE id = $1`, showingID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, sessionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id = $1`, productionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM venues WHERE id = $1`, venueID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM lots WHERE id = $1`, lotID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1 OR id = $2`, userID, linkedUserID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM locations WHERE id = $1`, locationID)
	})

	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name)
		VALUES ($1, $2)
		RETURNING id::text
	`, "linked_discord_user", "Linked Discord User").Scan(&linkedUserID); err != nil {
		t.Fatalf("insert linked user: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO auth.discord_identities (user_id, discord_user_id)
		VALUES ($1::uuid, $2)
	`, linkedUserID, "linked-discord-user-1"); err != nil {
		t.Fatalf("insert discord identity: %v", err)
	}

	hub := NewHub()
	linkRecord := discordGatewayLinkRecord{LocationID: locationID, DiscordGuildID: "guild-1", Active: true}
	msg := discordGatewayMessage{
		ID:        "msg-linked-user",
		ChannelID: "thread-1",
		GuildID:   "guild-1",
		Content:   "pong",
		Type:      0,
		Author: &discordGatewayUser{
			ID:       "linked-discord-user-1",
			Username: "linked_discord_user",
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	if err := handleDiscordGatewayMessageCreate(ctx, pool, hub, identity.DiscordServerLinkConfig{}, locationID, linkRecord, identity.DiscordGatewayConfig{}, "", msg); err != nil {
		t.Fatalf("create import with fallback: %v", err)
	}

	var importStatus, linkedUserIDInImport string
	if err := pool.QueryRow(ctx, `
		SELECT import_status, COALESCE(linked_user_id::text, '')
		FROM auth.discord_chat_imports
		WHERE discord_message_id = $1
	`, msg.ID).Scan(&importStatus, &linkedUserIDInImport); err != nil {
		t.Fatalf("load import row: %v", err)
	}
	if importStatus != "imported" {
		t.Fatalf("unexpected import status %q", importStatus)
	}
	if linkedUserIDInImport != "" {
		t.Fatalf("expected fallback import to clear linked user id, got %q", linkedUserIDInImport)
	}

	var text string
	if err := pool.QueryRow(ctx, `
		SELECT payload ->> 'text'
		FROM actions
		WHERE session_id = $1
		ORDER BY moment_id DESC
		LIMIT 1
	`, sessionID).Scan(&text); err != nil {
		t.Fatalf("load imported action: %v", err)
	}
	if text != "pong" {
		t.Fatalf("unexpected imported text %q", text)
	}
}

func TestDiscordGatewayMessageCreateSkipsWrongLocationThread(t *testing.T) {
	pool := openDiscordGatewayTestPool(t)
	ctx := context.Background()

	if err := identity.EnsureKernel39DiscordGatewaySurface(ctx, pool); err != nil {
		t.Fatalf("ensure gateway surface: %v", err)
	}

	locationA, venueA, lotA, productionA, userA, sessionA, showingA := setupDiscordGatewayEditFixture(t, pool)
	locationB, venueB, lotB, productionB, userB, sessionB, showingB := setupDiscordGatewayEditFixture(t, pool)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_chat_imports WHERE location_id IN ($1, $2)`, locationA, locationB)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_session_threads WHERE location_id IN ($1, $2)`, locationA, locationB)
		_, _ = pool.Exec(context.Background(), `DELETE FROM session_participants WHERE session_id IN ($1, $2)`, sessionA, sessionB)
		_, _ = pool.Exec(context.Background(), `DELETE FROM actions WHERE session_id IN ($1, $2)`, sessionA, sessionB)
		_, _ = pool.Exec(context.Background(), `DELETE FROM showings WHERE id IN ($1, $2)`, showingA, showingB)
		_, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id IN ($1, $2)`, sessionA, sessionB)
		_, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id IN ($1, $2)`, productionA, productionB)
		_, _ = pool.Exec(context.Background(), `DELETE FROM venues WHERE id IN ($1, $2)`, venueA, venueB)
		_, _ = pool.Exec(context.Background(), `DELETE FROM lots WHERE id IN ($1, $2)`, lotA, lotB)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id IN ($1, $2)`, userA, userB)
		_, _ = pool.Exec(context.Background(), `DELETE FROM locations WHERE id IN ($1, $2)`, locationA, locationB)
	})

	if _, err := pool.Exec(ctx, `
		INSERT INTO auth.discord_session_threads (
			location_id,
			venue_id,
			venue_slug,
			session_id,
			showing_id,
			discord_server_id,
			parent_channel_id,
			thread_id,
			thread_name,
			started_by_user_id,
			started_by_discord_user_id,
			started_at,
			showtime_at,
			status
		)
		VALUES (
			$1::uuid,
			$2::uuid,
			'gateway-edit-fixture',
			$3::uuid,
			$4::uuid,
			'guild-1',
			'parent-2',
			'thread-2',
			'Wrong Location Thread',
			$5::uuid,
			'discord-user-2',
			NOW(),
			NOW(),
			'active'
		)
	`, locationB, venueB, sessionB, showingB, userB); err != nil {
		t.Fatalf("insert secondary thread: %v", err)
	}

	hub := NewHub()
	linkRecord := discordGatewayLinkRecord{LocationID: locationA, DiscordGuildID: "guild-1", Active: true}
	msg := discordGatewayMessage{
		ID:        "msg-wrong-location",
		ChannelID: "thread-2",
		GuildID:   "guild-1",
		Content:   "pong",
		Type:      0,
		Author: &discordGatewayUser{
			ID:       "discord-user-2",
			Username: "wrong_location",
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	if err := handleDiscordGatewayMessageCreate(ctx, pool, hub, identity.DiscordServerLinkConfig{}, locationA, linkRecord, identity.DiscordGatewayConfig{}, "", msg); err != nil {
		t.Fatalf("create import for wrong location: %v", err)
	}

	var importCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM auth.discord_chat_imports
		WHERE discord_message_id = $1
	`, msg.ID).Scan(&importCount); err != nil {
		t.Fatalf("load import count: %v", err)
	}
	if importCount != 0 {
		t.Fatalf("expected wrong-location message to be skipped, got %d imports", importCount)
	}
}

func TestDiscordGatewayMessageCreateSkipsDuplicateAndBotEcho(t *testing.T) {
	pool := openDiscordGatewayTestPool(t)
	ctx := context.Background()

	if err := identity.EnsureKernel39DiscordGatewaySurface(ctx, pool); err != nil {
		t.Fatalf("ensure gateway surface: %v", err)
	}

	locationID, venueID, lotID, productionID, userID, sessionID, showingID := setupDiscordGatewayEditFixture(t, pool)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_chat_imports WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_session_threads WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM session_participants WHERE session_id = $1`, sessionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM actions WHERE session_id = $1`, sessionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM showings WHERE id = $1`, showingID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, sessionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id = $1`, productionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM venues WHERE id = $1`, venueID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM lots WHERE id = $1`, lotID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM locations WHERE id = $1`, locationID)
	})

	hub := NewHub()
	linkRecord := discordGatewayLinkRecord{LocationID: locationID, DiscordGuildID: "guild-1", Active: true}
	msg := discordGatewayMessage{
		ID:        "msg-duplicate",
		ChannelID: "thread-1",
		GuildID:   "guild-1",
		Content:   "pong",
		Type:      0,
		Author: &discordGatewayUser{
			ID:       "discord-user-1",
			Username: "grant",
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	if err := handleDiscordGatewayMessageCreate(ctx, pool, hub, identity.DiscordServerLinkConfig{}, locationID, linkRecord, identity.DiscordGatewayConfig{}, "", msg); err != nil {
		t.Fatalf("first import: %v", err)
	}
	if err := handleDiscordGatewayMessageCreate(ctx, pool, hub, identity.DiscordServerLinkConfig{}, locationID, linkRecord, identity.DiscordGatewayConfig{}, "", msg); err != nil {
		t.Fatalf("duplicate import: %v", err)
	}

	var actionCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM actions
		WHERE session_id = $1::uuid
	`, sessionID).Scan(&actionCount); err != nil {
		t.Fatalf("load action count: %v", err)
	}
	if actionCount != 1 {
		t.Fatalf("expected duplicate import to stay at 1 action, got %d", actionCount)
	}

	var botMsgCount int
	botMsg := msg
	botMsg.ID = "msg-bot-echo"
	botMsg.Author = &discordGatewayUser{
		ID:       "discord-bot-1",
		Username: "victory-bot",
		Bot:      true,
	}
	if err := handleDiscordGatewayMessageCreate(ctx, pool, hub, identity.DiscordServerLinkConfig{}, locationID, linkRecord, identity.DiscordGatewayConfig{}, "discord-bot-1", botMsg); err != nil {
		t.Fatalf("bot echo import: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM auth.discord_chat_imports
		WHERE discord_message_id = $1
	`, botMsg.ID).Scan(&botMsgCount); err != nil {
		t.Fatalf("load bot import count: %v", err)
	}
	if botMsgCount != 0 {
		t.Fatalf("expected bot echo to be dropped, got %d imports", botMsgCount)
	}
}

func TestDiscordGatewayMessageCreateHonorsDebugToggle(t *testing.T) {
	pool := openDiscordGatewayTestPool(t)
	ctx := context.Background()

	if err := identity.EnsureKernel39DiscordGatewaySurface(ctx, pool); err != nil {
		t.Fatalf("ensure gateway surface: %v", err)
	}

	locationID, venueID, lotID, productionID, userID, sessionID, showingID := setupDiscordGatewayEditFixture(t, pool)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_chat_imports WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_gateway_settings WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_session_threads WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM session_participants WHERE session_id = $1`, sessionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM actions WHERE session_id = $1`, sessionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM showings WHERE id = $1`, showingID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, sessionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id = $1`, productionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM venues WHERE id = $1`, venueID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM lots WHERE id = $1`, lotID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM locations WHERE id = $1`, locationID)
	})

	if _, err := pool.Exec(ctx, `
		INSERT INTO auth.discord_gateway_settings (
			location_id,
			debug_enabled,
			updated_by_user_id,
			updated_at
		)
		VALUES ($1::uuid, TRUE, $2::uuid, NOW())
		ON CONFLICT (location_id) DO UPDATE
		SET debug_enabled = EXCLUDED.debug_enabled,
			updated_by_user_id = EXCLUDED.updated_by_user_id,
			updated_at = NOW()
	`, locationID, userID); err != nil {
		t.Fatalf("enable debug tracing: %v", err)
	}

	hub := NewHub()
	linkRecord := discordGatewayLinkRecord{LocationID: locationID, DiscordGuildID: "guild-1", Active: true}
	msg := discordGatewayMessage{
		ID:        "msg-debug-enabled",
		ChannelID: "thread-1",
		GuildID:   "guild-1",
		Content:   "pong with debug",
		Type:      0,
		Author: &discordGatewayUser{
			ID:       "discord-user-1",
			Username: "grant",
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	if err := handleDiscordGatewayMessageCreate(ctx, pool, hub, identity.DiscordServerLinkConfig{}, locationID, linkRecord, identity.DiscordGatewayConfig{}, "", msg); err != nil {
		t.Fatalf("debug-enabled import: %v", err)
	}

	var importStatus string
	if err := pool.QueryRow(ctx, `
		SELECT import_status
		FROM auth.discord_chat_imports
		WHERE discord_message_id = $1
	`, msg.ID).Scan(&importStatus); err != nil {
		t.Fatalf("load debug import status: %v", err)
	}
	if importStatus != "imported" {
		t.Fatalf("expected debug-enabled import to succeed, got %s", importStatus)
	}
}

func TestDiscordGatewayMessageDeleteMarksImportDeleted(t *testing.T) {
	pool := openDiscordGatewayTestPool(t)
	ctx := context.Background()

	if err := identity.EnsureKernel39DiscordGatewaySurface(ctx, pool); err != nil {
		t.Fatalf("ensure gateway surface: %v", err)
	}

	locationID, venueID, lotID, productionID, userID, sessionID, showingID := setupDiscordGatewayEditFixture(t, pool)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_chat_imports WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_session_threads WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM session_participants WHERE session_id = $1`, sessionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM actions WHERE session_id = $1`, sessionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM showings WHERE id = $1`, showingID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, sessionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id = $1`, productionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM venues WHERE id = $1`, venueID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM lots WHERE id = $1`, lotID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM locations WHERE id = $1`, locationID)
	})

	hub := NewHub()
	linkRecord := discordGatewayLinkRecord{LocationID: locationID, DiscordGuildID: "guild-1", Active: true}
	msg := discordGatewayMessage{
		ID:        "msg-delete-me",
		ChannelID: "thread-1",
		GuildID:   "guild-1",
		Content:   "delete me",
		Type:      0,
		Author: &discordGatewayUser{
			ID:       "discord-user-1",
			Username: "grant",
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	if err := handleDiscordGatewayMessageCreate(ctx, pool, hub, identity.DiscordServerLinkConfig{}, locationID, linkRecord, identity.DiscordGatewayConfig{}, "", msg); err != nil {
		t.Fatalf("seed import before delete: %v", err)
	}
	if err := handleDiscordGatewayMessageDelete(ctx, pool, locationID, msg.ID, msg.ChannelID, msg.GuildID); err != nil {
		t.Fatalf("delete handling: %v", err)
	}

	var importStatus string
	if err := pool.QueryRow(ctx, `
		SELECT import_status
		FROM auth.discord_chat_imports
		WHERE discord_message_id = $1
	`, msg.ID).Scan(&importStatus); err != nil {
		t.Fatalf("load deleted import status: %v", err)
	}
	if importStatus != "deleted" {
		t.Fatalf("expected deleted import status, got %s", importStatus)
	}
}

func openDiscordGatewayTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	pool, err := db.NewPool(context.Background(), "postgres://victory:REDACTED@127.0.0.1:5432/victory?sslmode=disable")
	if err != nil {
		t.Skipf("postgres unavailable for integration test: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Skipf("postgres unavailable for integration test: %v", err)
	}
	return pool
}

func setupDiscordGatewayEditFixture(t *testing.T, pool *pgxpool.Pool) (locationID, venueID, lotID, productionID, userID, sessionID, showingID string) {
	t.Helper()

	ctx := context.Background()
	venueSlug := "gateway-edit-fixture"
	locationSlug := "gateway-edit-location"
	userHandle := "gateway_editor"

	if err := pool.QueryRow(ctx, `
		INSERT INTO locations (name, slug)
		VALUES ($1, $2)
		RETURNING id::text
	`, "Gateway Edit Location", locationSlug).Scan(&locationID); err != nil {
		t.Fatalf("insert location: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO lots (location_id, name, slug)
		VALUES ($1::uuid, $2, $3)
		RETURNING id::text
	`, locationID, "Gateway Edit Lot", "gateway-edit-lot").Scan(&lotID); err != nil {
		t.Fatalf("insert lot: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO venues (lot_id, name, slug, kind, config)
		VALUES ($1::uuid, $2, $3, 'presentation', jsonb_build_object('chat_enabled', TRUE))
		RETURNING id::text
	`, lotID, "Gateway Edit Venue", venueSlug).Scan(&venueID); err != nil {
		t.Fatalf("insert venue: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug)
		VALUES ($1::uuid, $2, $3)
		RETURNING id::text
	`, locationID, "Gateway Edit Production", "gateway-edit-production").Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name)
		VALUES ($1, $2)
		RETURNING id::text
	`, userHandle, "Gateway Editor").Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO sessions (venue_id, status, started_at, ended_at)
		VALUES ($1::uuid, 'rehearsal', NOW(), NULL)
	`, venueID); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT id::text
		FROM sessions
		WHERE venue_id = $1::uuid
		ORDER BY started_at DESC
		LIMIT 1
	`, venueID).Scan(&sessionID); err != nil {
		t.Fatalf("load session id: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO session_participants (session_id, user_id, role)
		VALUES ($1::uuid, $2::uuid, 'producer')
	`, sessionID, userID); err != nil {
		t.Fatalf("insert participant: %v", err)
	}

	showing, err := showings.EnsureForSession(ctx, pool, sessionID, userID)
	if err != nil {
		t.Fatalf("ensure showing: %v", err)
	}
	showingID = showing.ID

	if _, err := pool.Exec(ctx, `
		INSERT INTO auth.discord_session_threads (
			location_id,
			venue_id,
			venue_slug,
			session_id,
			showing_id,
			discord_server_id,
			parent_channel_id,
			thread_id,
			thread_name,
			started_by_user_id,
			started_by_discord_user_id,
			started_at,
			showtime_at,
			status
		)
		VALUES (
			$1::uuid,
			$2::uuid,
			$3,
			$4::uuid,
			$5::uuid,
			'guild-1',
			'parent-1',
			'thread-1',
			'Gateway Edit Thread',
			$6::uuid,
			'discord-user-1',
			NOW(),
			NOW(),
			'active'
		)
	`, locationID, venueID, venueSlug, sessionID, showingID, userID); err != nil {
		t.Fatalf("insert thread: %v", err)
	}

	return locationID, venueID, lotID, productionID, userID, sessionID, showingID
}
