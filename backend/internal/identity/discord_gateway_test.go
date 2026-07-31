package identity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/sessions"
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

	req := newGatewayStatusRequest(t, pool, "gateway_status_runtime_op")
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

	req := newGatewayStatusRequest(t, pool, "gateway_status_bootstrap_op")
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

	req := newGatewayStatusRequest(t, pool, "gateway_status_debug_op")
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

func TestDiscordGatewayDebugToggleEndpoint(t *testing.T) {
	pool := openDiscordTestPool(t)
	ensureDiscordServerTestSchema(t, pool)
	if err := EnsureKernel39DiscordGatewaySurface(context.Background(), pool); err != nil {
		t.Fatalf("ensure gateway surface: %v", err)
	}

	t.Setenv("OPERATOR_HANDLE", "gateway_debug_operator")
	locationID := resolveDiscordServerTestLocationID(t, pool, "amurray-family")
	userID := insertDiscordServerTestUser(t, pool, "gateway_debug_operator", "Gateway Debug Operator")
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_gateway_settings WHERE location_id = $1`, locationID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.sessions WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_gateway_settings WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	if _, err := pool.Exec(context.Background(), `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, 'producer', TRUE)
	`, locationID, userID); err != nil {
		t.Fatalf("insert membership: %v", err)
	}

	rawSession, _, err := sessions.CreateSession(context.Background(), pool, userID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	toggleReq := httptest.NewRequest(http.MethodPost, "/api/discord/gateway/debug", nil)
	toggleReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: rawSession})
	toggleRec := httptest.NewRecorder()
	HandleDiscordGatewayDebug(pool).ServeHTTP(toggleRec, toggleReq)
	if toggleRec.Code != http.StatusOK {
		t.Fatalf("unexpected toggle status %d body=%s", toggleRec.Code, toggleRec.Body.String())
	}

	var first struct {
		Ok   bool `json:"ok"`
		Data struct {
			DebugEnabled bool `json:"debug_enabled"`
		} `json:"data"`
	}
	if err := json.Unmarshal(toggleRec.Body.Bytes(), &first); err != nil {
		t.Fatalf("decode toggle response: %v", err)
	}
	if !first.Ok || !first.Data.DebugEnabled {
		t.Fatalf("expected toggle on, got %+v", first.Data)
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/api/discord/gateway/debug", nil)
	secondReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: rawSession})
	secondRec := httptest.NewRecorder()
	HandleDiscordGatewayDebug(pool).ServeHTTP(secondRec, secondReq)
	if secondRec.Code != http.StatusOK {
		t.Fatalf("unexpected second toggle status %d body=%s", secondRec.Code, secondRec.Body.String())
	}

	var second struct {
		Ok   bool `json:"ok"`
		Data struct {
			DebugEnabled bool `json:"debug_enabled"`
		} `json:"data"`
	}
	if err := json.Unmarshal(secondRec.Body.Bytes(), &second); err != nil {
		t.Fatalf("decode second toggle response: %v", err)
	}
	if !second.Ok || second.Data.DebugEnabled {
		t.Fatalf("expected toggle off, got %+v", second.Data)
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

// newGatewayStatusRequest builds an authenticated operator request for
// /api/discord/gateway/status. Kernel 76 (K76-M02) closed that route to
// anonymous callers, so the status tests now have to arrive as the operator.
func newGatewayStatusRequest(t *testing.T, pool *pgxpool.Pool, handle string) *http.Request {
	t.Helper()

	t.Setenv("OPERATOR_HANDLE", handle)
	locationID := resolveDiscordServerTestLocationID(t, pool, "amurray-family")
	userID := insertDiscordServerTestUser(t, pool, handle, handle)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.sessions WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	if _, err := pool.Exec(context.Background(), `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, 'producer', TRUE)
		ON CONFLICT (location_id, user_id, role) DO UPDATE SET active = TRUE
	`, locationID, userID); err != nil {
		t.Fatalf("insert membership: %v", err)
	}

	rawSession, _, err := sessions.CreateSession(context.Background(), pool, userID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/discord/gateway/status", nil)
	req.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: rawSession})
	return req
}

func TestDiscordGatewayStatusRejectsAnonymousCallers(t *testing.T) {
	pool := openDiscordTestPool(t)
	ensureDiscordServerTestSchema(t, pool)
	if err := EnsureKernel39DiscordGatewaySurface(context.Background(), pool); err != nil {
		t.Fatalf("ensure gateway surface: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/discord/gateway/status", nil)
	rec := httptest.NewRecorder()
	HandleDiscordGatewayStatus(pool, DiscordGatewayConfig{Enabled: true}).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous gateway status should be 401, got %d body=%s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); strings.Contains(body, "last_error") || strings.Contains(body, "intents") {
		t.Fatalf("anonymous rejection leaked gateway state: %s", body)
	}
}

func TestDiscordGatewayStatusRejectsNonOperator(t *testing.T) {
	pool := openDiscordTestPool(t)
	ensureDiscordServerTestSchema(t, pool)
	if err := EnsureKernel39DiscordGatewaySurface(context.Background(), pool); err != nil {
		t.Fatalf("ensure gateway surface: %v", err)
	}

	t.Setenv("OPERATOR_HANDLE", "gateway_status_real_operator")
	userID := insertDiscordServerTestUser(t, pool, "gateway_status_bystander", "Bystander")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.sessions WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	rawSession, _, err := sessions.CreateSession(context.Background(), pool, userID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/discord/gateway/status", nil)
	req.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: rawSession})
	rec := httptest.NewRecorder()
	HandleDiscordGatewayStatus(pool, DiscordGatewayConfig{Enabled: true}).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-operator gateway status should be 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}
