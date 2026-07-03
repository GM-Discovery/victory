package network

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"victory/backend/internal/actions"
	"victory/backend/internal/identity"
	"victory/backend/internal/showings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMirrorVictoryChatToDiscordPostsMessageAndPersistsBridgeRow(t *testing.T) {
	pool := openDiscordGatewayTestPool(t)
	ctx := context.Background()

	if err := identity.EnsureKernel39DiscordGatewaySurface(ctx, pool); err != nil {
		t.Fatalf("ensure gateway surface: %v", err)
	}

	locationID, sessionID, showingID, userID := setupDiscordChatBridgeFixture(t, pool)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_chat_message_bridges WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_session_threads WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM session_participants WHERE session_id = $1`, sessionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM showings WHERE id = $1`, showingID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, sessionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_link_settings WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	var receivedMethod string
	var receivedPath string
	var receivedContent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		receivedContent = string(body)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "discord-message-1"})
	}))
	defer server.Close()

	cfg := identity.DiscordServerLinkConfig{
		ApplicationID: "app-bridge",
		BotToken:      "bot-bridge",
		RedirectURL:   "https://victory.example/auth/discord/server/callback",
		Enabled:       true,
		APIBaseURL:    server.URL,
		HTTPClient:    server.Client(),
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO auth.discord_server_link_settings (
			location_id,
			application_id,
			bot_token,
			redirect_url,
			enabled,
			updated_at
		)
		VALUES ($1::uuid, $2, $3, $4, TRUE, NOW())
		ON CONFLICT (location_id) DO UPDATE
		SET application_id = EXCLUDED.application_id,
			bot_token = EXCLUDED.bot_token,
			redirect_url = EXCLUDED.redirect_url,
			enabled = EXCLUDED.enabled,
			updated_at = NOW()
	`, locationID, cfg.ApplicationID, cfg.BotToken, cfg.RedirectURL); err != nil {
		t.Fatalf("seed discord runtime config: %v", err)
	}

	action := &actions.StoredAction{
		ID:               "action-mirror-1",
		SessionID:        sessionID,
		ShowingID:        showingID,
		ActorID:          userID,
		ActorDisplayName: "Grant A. Murray",
		ActorHandle:      "grant",
		ActorRole:        "producer",
		Type:             "chat/message",
		Payload: map[string]any{
			"text": "Hello from Victory",
		},
	}

	mirrorVictoryChatToDiscord(ctx, pool, cfg, action)

	if receivedMethod != http.MethodPost {
		t.Fatalf("unexpected discord mirror method %s", receivedMethod)
	}
	if receivedPath != "/channels/bridge-thread-1/messages" {
		t.Fatalf("unexpected discord mirror path %s", receivedPath)
	}
	if receivedContent == "" || !strings.Contains(receivedContent, "Hello from Victory") {
		t.Fatalf("unexpected discord mirror content %q", receivedContent)
	}

	var bridgeStatus, discordMessageID string
	if err := pool.QueryRow(ctx, `
		SELECT bridge_status, COALESCE(discord_message_id, '')
		FROM auth.discord_chat_message_bridges
		WHERE action_id = $1::uuid
		LIMIT 1
	`, action.ID).Scan(&bridgeStatus, &discordMessageID); err != nil {
		t.Fatalf("load bridge row: %v", err)
	}
	if bridgeStatus != "sent" || discordMessageID != "discord-message-1" {
		t.Fatalf("unexpected bridge row status=%s discord_message_id=%s", bridgeStatus, discordMessageID)
	}
}

func setupDiscordChatBridgeFixture(t *testing.T, pool *pgxpool.Pool) (locationID, sessionID, showingID, userID string) {
	t.Helper()

	ctx := context.Background()
	suffix := strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000")
	if err := ensureDiscordBridgeTestSchema(ctx, pool); err != nil {
		t.Fatalf("ensure bridge schema: %v", err)
	}

	if err := pool.QueryRow(ctx, `
		SELECT id::text
		FROM locations
		WHERE slug = 'amurray-family'
		LIMIT 1
	`).Scan(&locationID); err != nil {
		t.Fatalf("load location: %v", err)
	}
	var venueID string
	if err := pool.QueryRow(ctx, `
		SELECT v.id::text
		FROM venues v
		JOIN lots lo ON lo.id = v.lot_id
		WHERE lo.location_id = $1::uuid
		  AND v.slug = 'first-theater'
		LIMIT 1
	`, locationID).Scan(&venueID); err != nil {
		t.Fatalf("load venue: %v", err)
	}

	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name)
		VALUES ($1, $2)
		RETURNING id::text
	`, "bridge_operator_"+suffix, "Bridge Operator").Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1::uuid, $2::uuid, 'producer', TRUE)
	`, locationID, userID); err != nil {
		t.Fatalf("insert membership: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO sessions (venue_id, status, started_at)
		VALUES ($1::uuid, 'rehearsal', NOW())
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
		t.Fatalf("load session: %v", err)
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
			'first-theater',
			$3::uuid,
			$4::uuid,
			'guild-bridge',
			'parent-bridge',
			'bridge-thread-1',
			'First Theater — Showtime',
			$5::uuid,
			'discord-user-bridge',
			NOW(),
			NOW(),
			'active'
		)
	`, locationID, venueID, sessionID, showingID, userID); err != nil {
		t.Fatalf("insert thread: %v", err)
	}

	return locationID, sessionID, showingID, userID
}

func ensureDiscordBridgeTestSchema(ctx context.Context, pool *pgxpool.Pool) error {
	return identity.EnsureKernel39DiscordGatewaySurface(ctx, pool)
}
