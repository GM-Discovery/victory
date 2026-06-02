package identity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"victory/backend/internal/sessions"
)

func TestDiscordChannelMappingRepairCreatesAndReusesSkeleton(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()
	t.Setenv("OPERATOR_HANDLE", "mapping_operator")

	ensureDiscordServerTestSchema(t, pool)

	userID := insertDiscordServerTestUser(t, pool, "mapping_operator", "Mapping Operator")
	locationID := resolveDiscordServerTestLocationID(t, pool, "producers-office")
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_link_settings WHERE location_id = $1`, locationID)
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_links WHERE location_id = $1`, locationID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.sessions WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_link_settings WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_links WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_channel_mappings WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	if _, err := pool.Exec(ctx, `
		INSERT INTO auth.discord_server_links (
			location_id, discord_guild_id, discord_guild_name, system_channel_id, system_channel_name,
			bot_verified, active, linked_by_user_id, linked_at, updated_at
		)
		VALUES ($1, 'guild-1', 'Example Server', '', '', TRUE, TRUE, $2, NOW(), NOW())
	`, locationID, userID); err != nil {
		t.Fatalf("insert link: %v", err)
	}

	rawSession, _, err := sessions.CreateSession(ctx, pool, userID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	transport := &discordMappingTransport{}
	cfg := DiscordServerLinkConfig{
		ApplicationID: "app-1",
		BotToken:      "bot-1",
		RedirectURL:   "https://victory.example/auth/discord/server/callback",
		Permissions:   "16",
		Enabled:       true,
		HTTPClient:    &http.Client{Transport: transport},
	}

	repairReq := httptest.NewRequest(http.MethodPost, "/api/discord/channel-mapping/repair", nil)
	repairReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: rawSession})
	repairRec := httptest.NewRecorder()
	HandleDiscordChannelMappingRepair(pool, cfg).ServeHTTP(repairRec, repairReq)
	if repairRec.Code != http.StatusOK {
		t.Fatalf("unexpected repair status %d", repairRec.Code)
	}

	var repairPayload struct {
		Ok   bool `json:"ok"`
		Data struct {
			Created []string `json:"created"`
			Found   []string `json:"found"`
			Updated []string `json:"updated"`
			Failed  []string `json:"failed"`
		} `json:"data"`
	}
	if err := json.Unmarshal(repairRec.Body.Bytes(), &repairPayload); err != nil {
		t.Fatalf("decode repair response: %v", err)
	}
	if !repairPayload.Ok || len(repairPayload.Data.Created) == 0 || len(repairPayload.Data.Failed) != 0 {
		t.Fatalf("unexpected repair payload: %+v", repairPayload)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM auth.discord_channel_mappings WHERE location_id = $1`, locationID).Scan(&count); err != nil {
		t.Fatalf("count mappings: %v", err)
	}
	if count != 17 {
		t.Fatalf("expected 17 mappings, got %d", count)
	}

	repeatReq := httptest.NewRequest(http.MethodPost, "/api/discord/channel-mapping/repair", nil)
	repeatReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: rawSession})
	repeatRec := httptest.NewRecorder()
	HandleDiscordChannelMappingRepair(pool, cfg).ServeHTTP(repeatRec, repeatReq)
	if repeatRec.Code != http.StatusOK {
		t.Fatalf("unexpected repeat repair status %d", repeatRec.Code)
	}

	if err := json.Unmarshal(repeatRec.Body.Bytes(), &repairPayload); err != nil {
		t.Fatalf("decode repeat repair response: %v", err)
	}
	if len(repairPayload.Data.Created) != 0 || len(repairPayload.Data.Failed) != 0 {
		t.Fatalf("expected repeat repair to reuse skeleton, got %+v", repairPayload)
	}

	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM auth.discord_channel_mappings WHERE location_id = $1`, locationID).Scan(&count); err != nil {
		t.Fatalf("count mappings after repeat: %v", err)
	}
	if count != 17 {
		t.Fatalf("repeat repair changed mapping count to %d", count)
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/api/discord/channel-mapping/status", nil)
	statusReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: rawSession})
	statusRec := httptest.NewRecorder()
	HandleDiscordChannelMappingStatus(pool, cfg).ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("unexpected status code %d", statusRec.Code)
	}

	var statusPayload struct {
		Ok   bool `json:"ok"`
		Data struct {
			Linked bool `json:"linked"`
			Core   struct {
				Category DiscordChannelMappingItem   `json:"category"`
				Channels []DiscordChannelMappingItem `json:"channels"`
			} `json:"core"`
			Venues      []DiscordChannelMappingItem `json:"venues"`
			ChatParents []DiscordChannelMappingItem `json:"chat_parents"`
		} `json:"data"`
	}
	if err := json.Unmarshal(statusRec.Body.Bytes(), &statusPayload); err != nil {
		t.Fatalf("decode status payload: %v", err)
	}
	if !statusPayload.Ok || !statusPayload.Data.Linked {
		t.Fatalf("unexpected status payload: %+v", statusPayload)
	}
	if statusPayload.Data.Core.Category.Status != "found" {
		t.Fatalf("unexpected core category status: %+v", statusPayload.Data.Core.Category)
	}
	if len(statusPayload.Data.Core.Channels) != 4 || len(statusPayload.Data.Venues) != 9 || len(statusPayload.Data.ChatParents) != 3 {
		t.Fatalf("unexpected mapping counts: %+v", statusPayload.Data)
	}
}

func TestDiscordChannelMappingStatusRequiresAuthentication(t *testing.T) {
	pool := openDiscordTestPool(t)

	req := httptest.NewRequest(http.MethodGet, "/api/discord/channel-mapping/status", nil)
	rec := httptest.NewRecorder()

	HandleDiscordChannelMappingStatus(pool, DiscordServerLinkConfig{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status %d", rec.Code)
	}
}

type discordMappingTransport struct {
	mu       sync.Mutex
	nextID   int
	channels []discordChannel
}

func (t *discordMappingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch {
	case req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/guilds/guild-1/channels"):
		return jsonResponse(mustJSON(t.channels)), nil
	case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/guilds/guild-1/channels"):
		var payload struct {
			Name     string `json:"name"`
			Type     int    `json:"type"`
			ParentID string `json:"parent_id"`
		}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			return jsonResponse(`{"error":"bad json"}`), nil
		}
		t.nextID++
		ch := discordChannel{
			ID:       "chan-" + strconv.Itoa(t.nextID),
			Name:     payload.Name,
			Type:     payload.Type,
			ParentID: payload.ParentID,
		}
		t.channels = append(t.channels, ch)
		return jsonResponse(mustJSON(ch)), nil
	default:
		return jsonResponse(`{"error":"unexpected request"}`), nil
	}
}

func mustJSON(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}
