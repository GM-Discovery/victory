package identity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"victory/backend/internal/sessions"
)

func TestDiscordAudioStatusRequiresAuthentication(t *testing.T) {
	pool := openDiscordTestPool(t)

	req := httptest.NewRequest(http.MethodGet, "/api/discord/audio/status?venue_slug=first-theater", nil)
	rec := httptest.NewRecorder()

	HandleDiscordAudioStatus(pool, DiscordServerLinkConfig{}, nil).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status %d", rec.Code)
	}
}

func TestDiscordAudioStatusTracksMappedVoiceChannels(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()
	t.Setenv("OPERATOR_HANDLE", "audio_operator")

	ensureDiscordServerTestSchema(t, pool)

	userID := insertDiscordServerTestUser(t, pool, "audio_operator", "Audio Operator")
	locationID := resolveDiscordServerTestLocationID(t, pool, "producers-office")
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_link_settings WHERE location_id = $1`, locationID)
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_links WHERE location_id = $1`, locationID)
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_channel_mappings WHERE location_id = $1`, locationID)
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
		t.Fatalf("unexpected repair status %d body=%s", repairRec.Code, repairRec.Body.String())
	}

	var audioChannelID string
	if err := pool.QueryRow(ctx, `
		SELECT discord_channel_id
		FROM auth.discord_channel_mappings
		WHERE location_id = $1
		  AND mapping_kind = $2
		  AND victory_scope_slug = $3
		LIMIT 1
	`, locationID, discordChannelMappingKindVenueAudio, "first-theater").Scan(&audioChannelID); err != nil {
		t.Fatalf("load audio channel id: %v", err)
	}

	for _, venueSlug := range []string{"first-theater", "the-cave", "catharsis"} {
		req := httptest.NewRequest(http.MethodGet, "/api/discord/audio/status?venue_slug="+venueSlug, nil)
		req.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: rawSession})
		rec := httptest.NewRecorder()
		HandleDiscordAudioStatus(pool, cfg, nil).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status for %s: %d body=%s", venueSlug, rec.Code, rec.Body.String())
		}

		var payload struct {
			Ok   bool `json:"ok"`
			Data struct {
				Configured          bool   `json:"configured"`
				DiscordServerLinked bool   `json:"discord_server_linked"`
				CanOpen             bool   `json:"can_open"`
				Status              string `json:"status"`
				AudioChannel        struct {
					Name    string `json:"name"`
					Type    string `json:"type"`
					OpenURL string `json:"open_url"`
				} `json:"audio_channel"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode audio status for %s: %v", venueSlug, err)
		}
		if !payload.Ok || !payload.Data.Configured || !payload.Data.DiscordServerLinked || !payload.Data.CanOpen || payload.Data.Status != "ready" {
			t.Fatalf("unexpected audio status for %s: %+v", venueSlug, payload.Data)
		}
		if !strings.Contains(payload.Data.AudioChannel.OpenURL, "/guild-1/") || !strings.HasPrefix(payload.Data.AudioChannel.OpenURL, "https://discord.com/channels/") {
			t.Fatalf("expected open url for %s, got %s", venueSlug, payload.Data.AudioChannel.OpenURL)
		}
		if payload.Data.AudioChannel.Type != "voice" {
			t.Fatalf("unexpected audio channel type for %s: %+v", venueSlug, payload.Data.AudioChannel)
		}
	}
}

func TestDiscordAudioStatusIncludesVoiceParticipantsAndFeatureNotes(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()
	t.Setenv("OPERATOR_HANDLE", "audio_operator_presence")

	ensureDiscordServerTestSchema(t, pool)

	userID := insertDiscordServerTestUser(t, pool, "audio_operator_presence", "Audio Operator Presence")
	locationID := resolveDiscordServerTestLocationID(t, pool, "producers-office")
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_link_settings WHERE location_id = $1`, locationID)
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_links WHERE location_id = $1`, locationID)
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_channel_mappings WHERE location_id = $1`, locationID)
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_gateway_state WHERE location_id = $1`, locationID)
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_identities WHERE user_id = $1`, userID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.sessions WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_link_settings WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_links WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_channel_mappings WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_gateway_state WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_identities WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	if _, err := pool.Exec(ctx, `
		INSERT INTO auth.discord_server_links (
			location_id, discord_guild_id, discord_guild_name, system_channel_id, system_channel_name,
			bot_verified, active, linked_by_user_id, linked_at, updated_at
		)
		VALUES ($1, 'guild-voice-1', 'Voice Example Server', '', '', TRUE, TRUE, $2, NOW(), NOW())
	`, locationID, userID); err != nil {
		t.Fatalf("insert link: %v", err)
	}

	discordLinkedUserID := insertDiscordServerTestUser(t, pool, "grant_voice", "Grant")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_identities WHERE user_id = $1`, discordLinkedUserID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, discordLinkedUserID)
	})
	if _, err := pool.Exec(ctx, `
		INSERT INTO auth.discord_identities (
			user_id,
			discord_user_id,
			username,
			global_name,
			discriminator,
			avatar,
			email,
			email_verified,
			locale,
			last_login_at
		)
		VALUES ($1, 'discord-linked-1', 'grantvoice', 'Grant Murray', '0', 'avatarhash-1', '', FALSE, 'en-US', NOW())
	`, discordLinkedUserID); err != nil {
		t.Fatalf("insert discord identity: %v", err)
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
		t.Fatalf("unexpected repair status %d body=%s", repairRec.Code, repairRec.Body.String())
	}

	var audioChannelID string
	if err := pool.QueryRow(ctx, `
		SELECT discord_channel_id
		FROM auth.discord_channel_mappings
		WHERE location_id = $1
		  AND mapping_kind = $2
		  AND victory_scope_slug = $3
		LIMIT 1
	`, locationID, discordChannelMappingKindVenueAudio, "first-theater").Scan(&audioChannelID); err != nil {
		t.Fatalf("load audio channel id: %v", err)
	}

	presenceStore := NewDiscordAudioPresenceStore()
	presenceStore.ReplaceGuildVoiceStates("guild-voice-1", []DiscordAudioPresenceState{
		{
			GuildID:       "guild-voice-1",
			ChannelID:     audioChannelID,
			UserID:        "discord-linked-1",
			Username:      "grantvoice",
			GlobalName:    "Grant Murray",
			Nick:          "Grant",
			AvatarHash:    "avatarhash-1",
			Discriminator: "0",
			LastSeenAt:    time.Now().UTC(),
		},
		{
			GuildID:       "guild-voice-1",
			ChannelID:     audioChannelID,
			UserID:        "discord-unlinked-1",
			Username:      "buddy",
			GlobalName:    "Buddy",
			AvatarHash:    "",
			Discriminator: "1234",
			LastSeenAt:    time.Now().UTC(),
			SelfMute:      true,
		},
	})

	if err := UpsertDiscordGatewayState(ctx, pool, DiscordGatewayStateRow{
		LocationID:        locationID,
		Enabled:           true,
		Configured:        true,
		Running:           true,
		Connected:         true,
		SessionID:         "gateway-session-1",
		BotUserID:         "bot-1",
		Intents:           (1 << 0) | (1 << 7) | (1 << 9),
		ActiveThreadCount: 1,
		LastConnectedAt:   ptrTime(time.Now().UTC()),
		LastEventAt:       ptrTime(time.Now().UTC()),
	}); err != nil {
		t.Fatalf("upsert gateway state: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/discord/audio/status?venue_slug=first-theater", nil)
	req.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: rawSession})
	rec := httptest.NewRecorder()
	HandleDiscordAudioStatus(pool, cfg, presenceStore).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Ok   bool `json:"ok"`
		Data struct {
			Configured          bool   `json:"configured"`
			DiscordServerLinked bool   `json:"discord_server_linked"`
			CanOpen             bool   `json:"can_open"`
			Status              string `json:"status"`
			VoiceStateTracking  struct {
				Available bool   `json:"available"`
				Reason    string `json:"reason"`
			} `json:"voice_state_tracking"`
			SpeakerIndicator struct {
				Available bool   `json:"available"`
				Reason    string `json:"reason"`
			} `json:"speaker_indicator"`
			VolumeControls struct {
				Available bool   `json:"available"`
				Reason    string `json:"reason"`
			} `json:"volume_controls"`
			Participants []struct {
				DiscordUserID      string `json:"discord_user_id"`
				DiscordDisplayName string `json:"discord_display_name"`
				DisplayName        string `json:"display_name"`
				AvatarURL          string `json:"avatar_url"`
				LinkedUserID       string `json:"linked_user_id"`
				VictoryDisplayName string `json:"victory_display_name"`
				SourceLabel        string `json:"source_label"`
				Status             string `json:"status"`
			} `json:"participants"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode audio presence payload: %v", err)
	}
	if !payload.Ok || !payload.Data.Configured || !payload.Data.DiscordServerLinked || !payload.Data.CanOpen || payload.Data.Status != "ready" {
		t.Fatalf("unexpected ready payload: %+v", payload.Data)
	}
	if !payload.Data.VoiceStateTracking.Available {
		t.Fatalf("expected voice-state tracking available, got %+v", payload.Data.VoiceStateTracking)
	}
	if payload.Data.SpeakerIndicator.Available || payload.Data.VolumeControls.Available {
		t.Fatalf("expected speaker/volume controls to be deferred, got speaker=%+v volume=%+v", payload.Data.SpeakerIndicator, payload.Data.VolumeControls)
	}
	if len(payload.Data.Participants) != 2 {
		t.Fatalf("expected 2 participants, got %+v", payload.Data.Participants)
	}
	// Snapshot() sorts participants alphabetically by display name (a
	// deliberate, deterministic UI ordering -- see discord_audio_presence.go),
	// so "Buddy" legitimately sorts before "Grant" regardless of insertion
	// order.
	if payload.Data.Participants[0].SourceLabel != "via Discord" || payload.Data.Participants[0].DisplayName != "Buddy" || payload.Data.Participants[0].LinkedUserID != "" {
		t.Fatalf("expected unlinked participant label, got %+v", payload.Data.Participants[0])
	}
	if payload.Data.Participants[1].DisplayName != "Grant" || payload.Data.Participants[1].LinkedUserID == "" || payload.Data.Participants[1].VictoryDisplayName != "Grant" {
		t.Fatalf("expected linked participant label, got %+v", payload.Data.Participants[1])
	}
}

func ptrTime(v time.Time) *time.Time {
	return &v
}
