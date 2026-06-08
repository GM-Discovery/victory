package identity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscordGatewayStatusReportsRuntimeState(t *testing.T) {
	pool := openDiscordTestPool(t)
	ensureDiscordServerTestSchema(t, pool)
	if err := EnsureKernel39DiscordGatewaySurface(context.Background(), pool); err != nil {
		t.Fatalf("ensure gateway surface: %v", err)
	}

	locationID := resolveDiscordServerTestLocationID(t, pool, "amurray-family")
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_gateway_state WHERE location_id = $1`, locationID)
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_gateway_settings WHERE location_id = $1`, locationID)
	_, _ = pool.Exec(context.Background(), `
		INSERT INTO auth.discord_gateway_state (
			location_id,
			enabled,
			configured,
			running,
			connected,
			session_id,
			bot_user_id,
			intents,
			message_content_intent,
			active_thread_count,
			last_connected_at,
			last_event_at,
			last_error,
			updated_at
		)
		VALUES ($1, TRUE, TRUE, TRUE, TRUE, 'session-1', 'bot-1', 513, FALSE, 4, NOW(), NOW(), '', NOW())
	`, locationID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_gateway_state WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_gateway_settings WHERE location_id = $1`, locationID)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/discord/gateway/status", nil)
	rec := httptest.NewRecorder()

	HandleDiscordGatewayStatus(pool, DiscordGatewayConfig{Enabled: true, BotToken: "bot-token", Intents: 513}).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d", rec.Code)
	}

	var response struct {
		Ok   bool `json:"ok"`
		Data struct {
			Configured           bool   `json:"configured"`
			Enabled              bool   `json:"enabled"`
			Running              bool   `json:"running"`
			Connected            bool   `json:"connected"`
			DebugEnabled         bool   `json:"debug_enabled"`
			SessionID            string `json:"session_id"`
			BotUserID            string `json:"bot_user_id"`
			Intents              int64  `json:"intents"`
			MessageContentIntent bool   `json:"message_content_intent"`
			ActiveThreadCount    int    `json:"active_thread_count"`
			LastError            string `json:"last_error"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Ok || !response.Data.Configured || !response.Data.Running || !response.Data.Connected {
		t.Fatalf("unexpected gateway status: %+v", response.Data)
	}
	if response.Data.DebugEnabled {
		t.Fatalf("expected debug disabled by default")
	}
	if response.Data.SessionID != "session-1" || response.Data.BotUserID != "bot-1" {
		t.Fatalf("unexpected gateway identifiers: %+v", response.Data)
	}
	if response.Data.ActiveThreadCount != 4 || response.Data.Intents != 513 || response.Data.MessageContentIntent {
		t.Fatalf("unexpected gateway counts/intents: %+v", response.Data)
	}
}

func TestDiscordGatewayStatusResolvesBootstrapSettingsFromDatabase(t *testing.T) {
	pool := openDiscordTestPool(t)
	ensureDiscordServerTestSchema(t, pool)
	if err := EnsureKernel39DiscordGatewaySurface(context.Background(), pool); err != nil {
		t.Fatalf("ensure gateway surface: %v", err)
	}

	locationID := resolveDiscordServerTestLocationID(t, pool, "amurray-family")
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_gateway_state WHERE location_id = $1`, locationID)
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_link_settings WHERE location_id = $1`, locationID)
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_gateway_settings WHERE location_id = $1`, locationID)
	_, _ = pool.Exec(context.Background(), `
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
		VALUES ($1, 'app-db', 'bot-db', 'https://victory.example/auth/discord/server/callback', '16', 'public-key-db', TRUE, NOW())
	`, locationID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_gateway_state WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_link_settings WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_gateway_settings WHERE location_id = $1`, locationID)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/discord/gateway/status", nil)
	rec := httptest.NewRecorder()

	HandleDiscordGatewayStatus(pool, DiscordGatewayConfig{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d", rec.Code)
	}

	var response struct {
		Ok   bool `json:"ok"`
		Data struct {
			Configured   bool `json:"configured"`
			Enabled      bool `json:"enabled"`
			DebugEnabled bool `json:"debug_enabled"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Ok || !response.Data.Configured || !response.Data.Enabled {
		t.Fatalf("expected database-backed gateway config, got %+v", response.Data)
	}
	if response.Data.DebugEnabled {
		t.Fatalf("expected debug disabled by default")
	}
}

func TestDiscordGatewayDebugEnabledPersistsInStatus(t *testing.T) {
	pool := openDiscordTestPool(t)
	ensureDiscordServerTestSchema(t, pool)
	if err := EnsureKernel39DiscordGatewaySurface(context.Background(), pool); err != nil {
		t.Fatalf("ensure gateway surface: %v", err)
	}

	locationID := resolveDiscordServerTestLocationID(t, pool, "amurray-family")
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_gateway_state WHERE location_id = $1`, locationID)
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_gateway_settings WHERE location_id = $1`, locationID)
	_, _ = pool.Exec(context.Background(), `
		INSERT INTO auth.discord_gateway_settings (
			location_id,
			debug_enabled,
			updated_at
		)
		VALUES ($1, TRUE, NOW())
	`, locationID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_gateway_state WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_gateway_settings WHERE location_id = $1`, locationID)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/discord/gateway/status", nil)
	rec := httptest.NewRecorder()

	HandleDiscordGatewayStatus(pool, DiscordGatewayConfig{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d", rec.Code)
	}

	var response struct {
		Ok   bool `json:"ok"`
		Data struct {
			DebugEnabled bool `json:"debug_enabled"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Ok || !response.Data.DebugEnabled {
		t.Fatalf("expected debug enabled in status, got %+v", response.Data)
	}
}

func TestDiscordGatewayStateUpsertHandlesEmptyIdentifiers(t *testing.T) {
	pool := openDiscordTestPool(t)
	ensureDiscordServerTestSchema(t, pool)
	if err := EnsureKernel39DiscordGatewaySurface(context.Background(), pool); err != nil {
		t.Fatalf("ensure gateway surface: %v", err)
	}

	locationID := resolveDiscordServerTestLocationID(t, pool, "amurray-family")
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_gateway_state WHERE location_id = $1`, locationID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_gateway_state WHERE location_id = $1`, locationID)
	})

	if err := upsertDiscordGatewayState(context.Background(), pool, DiscordGatewayStateRow{
		LocationID:           locationID,
		Enabled:              true,
		Configured:           true,
		Running:              true,
		Connected:            true,
		SessionID:            "",
		BotUserID:            "",
		Intents:              513,
		MessageContentIntent: false,
		ActiveThreadCount:    0,
	}); err != nil {
		t.Fatalf("upsert gateway state: %v", err)
	}

	row, err := LoadDiscordGatewayState(context.Background(), pool, locationID)
	if err != nil {
		t.Fatalf("load gateway state: %v", err)
	}
	if row == nil {
		t.Fatalf("expected gateway state row")
	}
	if row.SessionID != "" || row.BotUserID != "" {
		t.Fatalf("expected empty identifiers, got session=%q bot=%q", row.SessionID, row.BotUserID)
	}
	if !row.Enabled || !row.Configured || !row.Running || !row.Connected || row.Intents != 513 {
		t.Fatalf("unexpected gateway state row: %+v", row)
	}
}
