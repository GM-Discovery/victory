package identity

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"victory/backend/internal/sessions"
)

func TestDiscordMicInteractionsVerifyPingAndRejectBadSignature(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	pool := openDiscordTestPool(t)
	locationID := resolveDiscordServerTestLocationID(t, pool, "producers-office")
	ensureDiscordServerTestSchema(t, pool)
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_link_settings WHERE location_id = $1`, locationID)
	_, _ = pool.Exec(context.Background(), `
		INSERT INTO auth.discord_server_link_settings (
			location_id,
			public_key,
			enabled,
			updated_at
		)
		VALUES ($1, $2, TRUE, NOW())
	`, locationID, hex.EncodeToString(pub))
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_link_settings WHERE location_id = $1`, locationID)
	})

	body := []byte(`{"type":1}`)
	timestamp := "1710000000"
	signature := ed25519.Sign(priv, append([]byte(timestamp), body...))

	req := httptest.NewRequest(http.MethodPost, "/api/discord/interactions", strings.NewReader(string(body)))
	req.Header.Set("X-Signature-Timestamp", timestamp)
	req.Header.Set("X-Signature-Ed25519", hex.EncodeToString(signature))
	rec := httptest.NewRecorder()

	HandleDiscordInteractions(pool, DiscordServerLinkConfig{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected ping status %d", rec.Code)
	}

	var payload struct {
		Type int `json:"type"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode ping response: %v", err)
	}
	if payload.Type != discordInteractionResponseTypePong {
		t.Fatalf("unexpected ping response type %d", payload.Type)
	}

	badReq := httptest.NewRequest(http.MethodPost, "/api/discord/interactions", strings.NewReader(string(body)))
	badReq.Header.Set("X-Signature-Timestamp", timestamp)
	badReq.Header.Set("X-Signature-Ed25519", strings.Repeat("0", len(hex.EncodeToString(signature))))
	badRec := httptest.NewRecorder()
	HandleDiscordInteractions(pool, DiscordServerLinkConfig{PublicKey: hex.EncodeToString(pub)}).ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected bad signature status %d", badRec.Code)
	}
}

func TestDiscordMicRegisterAndStatusDispatch(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()
	t.Setenv("OPERATOR_HANDLE", "mic_operator")

	ensureDiscordServerTestSchema(t, pool)

	userID := insertDiscordServerTestUser(t, pool, "mic_operator", "Mic Operator")
	locationID := resolveDiscordServerTestLocationID(t, pool, "producers-office")
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_link_settings WHERE location_id = $1`, locationID)
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_links WHERE location_id = $1`, locationID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.sessions WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_link_settings WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_links WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_channel_mappings WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_identities WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, 'producer', TRUE)
	`, locationID, userID); err != nil {
		t.Fatalf("insert membership: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO auth.discord_identities (
			user_id,
			discord_user_id,
			username,
			global_name
		)
		VALUES ($1, $2, $3, $4)
	`, userID, "discord-user-1", "micbird", "Mic Bird"); err != nil {
		t.Fatalf("insert discord identity: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO auth.discord_server_links (
			location_id,
			discord_guild_id,
			discord_guild_name,
			system_channel_id,
			system_channel_name,
			bot_verified,
			active,
			linked_by_user_id,
			linked_at,
			updated_at
		)
		VALUES ($1, 'guild-1', 'Example Server', 'sys-1', 'victory-system', TRUE, TRUE, $2, NOW(), NOW())
	`, locationID, userID); err != nil {
		t.Fatalf("insert linked server: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO auth.discord_server_link_settings (
			location_id,
			application_id,
			bot_token,
			redirect_url,
			permissions,
			public_key,
			enabled,
			updated_at
		)
		VALUES ($1, 'app-1', 'bot-1', 'https://victory.example/auth/discord/server/callback', '16', $2, TRUE, NOW())
	`, locationID, hex.EncodeToString(mustDiscordMicPublicKey(t))); err != nil {
		t.Fatalf("insert link settings: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO auth.discord_channel_mappings (
			location_id,
			discord_server_id,
			discord_channel_id,
			discord_channel_name,
			discord_channel_type,
			mapping_kind,
			victory_scope_kind,
			victory_scope_slug,
			expected_name,
			created_by_user_id
		)
		VALUES ($1, 'guild-1', 'chat-1', 'the-cave-chat', 'text', 'venue_chat_channel', 'venue_chat_channel', 'the-cave', 'the-cave-chat', $2)
	`, locationID, userID); err != nil {
		t.Fatalf("insert channel mapping: %v", err)
	}

	rawSession, _, err := sessions.CreateSession(ctx, pool, userID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	cfg := DiscordServerLinkConfig{
		ApplicationID: "app-1",
		BotToken:      "bot-1",
		RedirectURL:   "https://victory.example/auth/discord/server/callback",
		PublicKey:     "public-key-1",
		Enabled:       true,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case req.Method == http.MethodGet && strings.Contains(req.URL.Path, "/applications/app-1/guilds/guild-1/commands"):
				return jsonResponse(`[]`), nil
			case req.Method == http.MethodPost && strings.Contains(req.URL.Path, "/applications/app-1/guilds/guild-1/commands"):
				return jsonResponse(`{"id":"mic-1","name":"mic"}`), nil
			default:
				t.Fatalf("unexpected mic request %s %s", req.Method, req.URL.Path)
				return nil, nil
			}
		})},
	}

	registerReq := httptest.NewRequest(http.MethodPost, "/api/discord/mic/register", nil)
	registerReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: rawSession})
	registerRec := httptest.NewRecorder()
	HandleDiscordMicRegister(pool, cfg).ServeHTTP(registerRec, registerReq)
	if registerRec.Code != http.StatusOK {
		t.Fatalf("unexpected register status %d", registerRec.Code)
	}

	body := []byte(`{"type":2,"data":{"name":"mic","options":[{"type":1,"name":"status"}]},"guild_id":"guild-1","channel_id":"chat-1","member":{"user":{"id":"discord-user-1","username":"micbird"}}}`)
	timestamp := "1710000000"
	signature := ed25519.Sign(mustDiscordMicPrivateKey(t), append([]byte(timestamp), body...))

	interactionReq := httptest.NewRequest(http.MethodPost, "/api/discord/interactions", strings.NewReader(string(body)))
	interactionReq.Header.Set("X-Signature-Timestamp", timestamp)
	interactionReq.Header.Set("X-Signature-Ed25519", hex.EncodeToString(signature))
	interactionRec := httptest.NewRecorder()
	HandleDiscordInteractions(pool, DiscordServerLinkConfig{
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case req.Method == http.MethodGet && strings.Contains(req.URL.Path, "/applications/app-1/guilds/guild-1/commands"):
				return jsonResponse(`[{"id":"mic-1","name":"mic"}]`), nil
			default:
				t.Fatalf("unexpected interaction request %s %s", req.Method, req.URL.Path)
				return nil, nil
			}
		})},
	}).ServeHTTP(interactionRec, interactionReq)
	if interactionRec.Code != http.StatusOK {
		t.Fatalf("unexpected interaction status %d", interactionRec.Code)
	}

	var response struct {
		Type int `json:"type"`
		Data struct {
			Content string `json:"content"`
			Flags   int    `json:"flags"`
		} `json:"data"`
	}
	if err := json.Unmarshal(interactionRec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode interaction response: %v", err)
	}
	if response.Type != discordInteractionResponseTypeDeferredChannelMessageWithSource {
		t.Fatalf("unexpected interaction response type %d", response.Type)
	}
	if response.Data.Flags != 1<<6 {
		t.Fatalf("unexpected interaction flags %d", response.Data.Flags)
	}
}

func TestDiscordMicControlFallsBackWithoutDiscordBootstrap(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()
	t.Setenv("OPERATOR_HANDLE", "mic_operator_local")

	ensureDiscordServerTestSchema(t, pool)

	userID := insertDiscordServerTestUser(t, pool, "mic_operator_local", "Mic Operator Local")
	locationID := resolveDiscordServerTestLocationID(t, pool, "producers-office")
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_link_settings WHERE location_id = $1`, locationID)
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_links WHERE location_id = $1`, locationID)
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_session_threads WHERE location_id = $1 AND venue_slug = $2`, locationID, "first-theater")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.sessions WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_session_threads WHERE location_id = $1 AND venue_slug = $2`, locationID, "first-theater")
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_identities WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, 'producer', TRUE)
	`, locationID, userID); err != nil {
		t.Fatalf("insert membership: %v", err)
	}

	rawSession, _, err := sessions.CreateSession(ctx, pool, userID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if err := saveDiscordMicThread(ctx, pool, discordMicThreadRow{
		LocationID:      locationID,
		VenueSlug:       "first-theater",
		VenueName:       "First Theater",
		DiscordServerID: "guild-1",
		ParentChannelID: "chat-1",
		ThreadID:        "thread-1",
		ThreadName:      "First Theater — Showtime — June 6, 2026 12:00 PM",
		StartedByUserID: userID,
		StartedAt:       time.Now().UTC(),
		ShowtimeAt:      time.Now().UTC(),
		Status:          "active",
	}); err != nil {
		t.Fatalf("seed mic thread: %v", err)
	}

	statusReq := httptest.NewRequest(http.MethodPost, "/api/discord/mic/control", strings.NewReader(`{"venue_slug":"first-theater","command":"status"}`))
	statusReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: rawSession})
	statusReq.Header.Set("Content-Type", "application/json")
	statusRec := httptest.NewRecorder()
	HandleDiscordMicControl(pool, DiscordServerLinkConfig{}).ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("unexpected status control code %d body=%s", statusRec.Code, statusRec.Body.String())
	}
	if !strings.Contains(statusRec.Body.String(), "Mic: On") {
		t.Fatalf("expected local mic status, got %s", statusRec.Body.String())
	}

	offReq := httptest.NewRequest(http.MethodPost, "/api/discord/mic/control", strings.NewReader(`{"venue_slug":"first-theater","command":"off"}`))
	offReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: rawSession})
	offReq.Header.Set("Content-Type", "application/json")
	offRec := httptest.NewRecorder()
	HandleDiscordMicControl(pool, DiscordServerLinkConfig{}).ServeHTTP(offRec, offReq)
	if offRec.Code != http.StatusOK {
		t.Fatalf("unexpected off control code %d body=%s", offRec.Code, offRec.Body.String())
	}

	var rowStatus string
	if err := pool.QueryRow(ctx, `
		SELECT status
		FROM auth.discord_session_threads
		WHERE location_id = $1 AND venue_slug = $2
		LIMIT 1
	`, locationID, "first-theater").Scan(&rowStatus); err != nil {
		t.Fatalf("load mic thread status: %v", err)
	}
	if rowStatus != "inactive" {
		t.Fatalf("expected inactive status, got %s", rowStatus)
	}
}

var discordMicTestPubKey, discordMicTestPrivKey = func() (ed25519.PublicKey, ed25519.PrivateKey) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	return pub, priv
}()

func mustDiscordMicPublicKey(t *testing.T) ed25519.PublicKey {
	t.Helper()
	return discordMicTestPubKey
}

func mustDiscordMicPrivateKey(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	return discordMicTestPrivKey
}
