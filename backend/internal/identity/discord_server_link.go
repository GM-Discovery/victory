package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"victory/backend/internal/access"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	discordServerLinkProvider     = "discord_server_install"
	discordServerLinkDefaultGuild = "victory-system"
	discordServerLinkAuthorizeURL = "https://discord.com/oauth2/authorize"
	discordServerLinkAPIBaseURL   = "https://discord.com/api/v10"
)

type DiscordServerLinkConfig struct {
	ApplicationID string
	BotToken      string
	RedirectURL   string
	Permissions   string
	PublicKey     string
	Enabled       bool
	AuthorizeURL  string
	APIBaseURL    string
	HTTPClient    *http.Client
}

type DiscordServerLinkStatus struct {
	Linked         bool                      `json:"linked"`
	DiscordServer  *DiscordServerLinkServer  `json:"discord_server,omitempty"`
	Bot            DiscordServerLinkBot      `json:"bot"`
	SystemChannel  *DiscordServerLinkChannel `json:"system_channel,omitempty"`
	CanManage      bool                      `json:"can_manage"`
	InstallURL     string                    `json:"install_url,omitempty"`
	UnlinkURL      string                    `json:"unlink_url,omitempty"`
	SetupAvailable bool                      `json:"setup_available"`
}

type DiscordServerLinkServer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DiscordServerLinkChannel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DiscordServerLinkBot struct {
	Installed bool `json:"installed"`
	Verified  bool `json:"verified"`
}

type discordServerLinkRecord struct {
	LocationID        string
	DiscordGuildID    string
	DiscordGuildName  string
	SystemChannelID   string
	SystemChannelName string
	BotVerified       bool
	Active            bool
}

type discordGuild struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type discordChannel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     int    `json:"type"`
	ParentID string `json:"parent_id,omitempty"`
}

type discordServerLinkUpsert struct {
	LocationID        string
	DiscordGuildID    string
	DiscordGuildName  string
	SystemChannelID   string
	SystemChannelName string
	BotVerified       bool
	LinkedByUserID    string
}

func DiscordServerLinkConfigured(cfg DiscordServerLinkConfig) bool {
	return cfg.Enabled &&
		strings.TrimSpace(cfg.ApplicationID) != "" &&
		strings.TrimSpace(cfg.BotToken) != "" &&
		strings.TrimSpace(cfg.RedirectURL) != ""
}

func DiscordInteractionsConfigured(cfg DiscordServerLinkConfig) bool {
	return cfg.Enabled && strings.TrimSpace(cfg.PublicKey) != ""
}

func DiscordMicCommandConfigured(cfg DiscordServerLinkConfig) bool {
	return DiscordServerLinkConfigured(cfg) && DiscordInteractionsConfigured(cfg)
}

func discordServerLinkHTTPClient(cfg DiscordServerLinkConfig) *http.Client {
	if cfg.HTTPClient != nil {
		return cfg.HTTPClient
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func HandleDiscordServerLinkStatus(pool *pgxpool.Pool, cfg DiscordServerLinkConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID, err := currentUserID(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		location, err := resolveProducerOfficeLocation(ctx, pool)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "location_lookup_failed", "detail": err.Error()})
			return
		}

		runtimeCfg, err := resolveDiscordServerLinkRuntimeConfig(ctx, pool, cfg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "config_lookup_failed", "detail": err.Error()})
			return
		}

		record, err := loadDiscordServerLinkRecord(ctx, pool, location.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "link_lookup_failed"})
			return
		}

		allowed, err := access.IsOperatorUser(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "access_check_failed"})
			return
		}

		status := DiscordServerLinkStatus{
			Linked:         record.Active && strings.TrimSpace(record.DiscordGuildID) != "",
			Bot:            DiscordServerLinkBot{Installed: record.Active && strings.TrimSpace(record.DiscordGuildID) != "", Verified: record.BotVerified && record.Active},
			CanManage:      allowed,
			InstallURL:     "/auth/discord/server/install?return_to=/venues/producers-office/",
			UnlinkURL:      "/api/discord/server/unlink",
			SetupAvailable: DiscordServerLinkConfigured(runtimeCfg),
		}
		if status.Linked {
			status.DiscordServer = &DiscordServerLinkServer{
				ID:   record.DiscordGuildID,
				Name: fallbackString(record.DiscordGuildName, record.DiscordGuildID),
			}
		}
		if strings.TrimSpace(record.SystemChannelID) != "" {
			status.SystemChannel = &DiscordServerLinkChannel{
				ID:   record.SystemChannelID,
				Name: fallbackString(record.SystemChannelName, discordServerLinkDefaultGuild),
			}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"data": status,
		})
	}
}

func HandleDiscordServerInstall(pool *pgxpool.Pool, cfg DiscordServerLinkConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID, err := currentUserID(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		allowed, err := access.IsOperatorUser(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "access_check_failed"})
			return
		}
		if !allowed {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
			return
		}

		runtimeCfg, err := resolveDiscordServerLinkRuntimeConfig(ctx, pool, cfg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "config_lookup_failed"})
			return
		}
		if !DiscordServerLinkConfigured(runtimeCfg) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": "discord_server_link_unavailable"})
			return
		}

		rawState, stateHash, err := newOAuthStateToken()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "state_generation_failed"})
			return
		}

		returnTo := safeReturnTo(r.URL.Query().Get("return_to"))
		if returnTo == "/" {
			returnTo = "/venues/producers-office/"
		}
		expiresAt := time.Now().UTC().Add(discordOAuthStateTTL)

		_, _ = pool.Exec(ctx, `
			DELETE FROM auth.oauth_states
			WHERE provider = $1
			  AND expires_at < NOW()
		`, discordServerLinkProvider)

		_, err = pool.Exec(ctx, `
			INSERT INTO auth.oauth_states (provider, state_hash, return_to, expires_at)
			VALUES ($1, $2, $3, $4)
		`, discordServerLinkProvider, stateHash, returnTo, expiresAt)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "state_store_failed"})
			return
		}

		authURL, err := buildDiscordServerAuthorizeURL(runtimeCfg, rawState)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "install_url_failed"})
			return
		}

		http.Redirect(w, r, authURL, http.StatusFound)
	}
}

func HandleDiscordServerCallback(pool *pgxpool.Pool, cfg DiscordServerLinkConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		userID, err := currentUserID(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		allowed, err := access.IsOperatorUser(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "access_check_failed"})
			return
		}
		if !allowed {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
			return
		}

		runtimeCfg, err := resolveDiscordServerLinkRuntimeConfig(ctx, pool, cfg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "config_lookup_failed"})
			return
		}
		if !DiscordServerLinkConfigured(runtimeCfg) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": "discord_server_link_unavailable"})
			return
		}

		state := strings.TrimSpace(r.URL.Query().Get("state"))
		guildID := strings.TrimSpace(r.URL.Query().Get("guild_id"))
		if state == "" || guildID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "missing_state_or_guild"})
			return
		}

		var returnTo string
		err = pool.QueryRow(ctx, `
			UPDATE auth.oauth_states
			SET consumed_at = NOW()
			WHERE provider = $1
			  AND state_hash = $2
			  AND consumed_at IS NULL
			  AND expires_at > NOW()
			RETURNING COALESCE(NULLIF(return_to, ''), '/venues/producers-office/')
		`, discordServerLinkProvider, hashOAuthToken(state)).Scan(&returnTo)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_or_expired_state"})
			return
		}

		location, err := resolveProducerOfficeLocation(ctx, pool)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "location_lookup_failed", "detail": err.Error()})
			return
		}

		guild, err := fetchDiscordGuild(ctx, runtimeCfg, guildID)
		if err != nil {
			log.Printf("discord server callback: guild lookup failed for guild_id=%s: %v", guildID, err)
			writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": "discord_guild_lookup_failed", "detail": err.Error()})
			return
		}

		channel, err := ensureDiscordSystemChannel(ctx, runtimeCfg, guildID)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": "discord_system_channel_failed", "detail": err.Error()})
			return
		}

		if err := saveDiscordServerLink(ctx, pool, discordServerLinkUpsert{
			LocationID:        location.ID,
			DiscordGuildID:    guild.ID,
			DiscordGuildName:  fallbackString(guild.Name, guild.ID),
			SystemChannelID:   channel.ID,
			SystemChannelName: fallbackString(channel.Name, discordServerLinkDefaultGuild),
			BotVerified:       true,
			LinkedByUserID:    userID,
		}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "link_store_failed", "detail": err.Error()})
			return
		}

		scheduleDiscordBootstrapReconcile(pool, cfg)

		http.Redirect(w, r, safeReturnTo(returnTo), http.StatusFound)
	}
}

func HandleDiscordServerUnlink(pool *pgxpool.Pool, cfg DiscordServerLinkConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID, err := currentUserID(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		allowed, err := access.IsOperatorUser(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "access_check_failed"})
			return
		}
		if !allowed {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
			return
		}

		location, err := resolveProducerOfficeLocation(ctx, pool)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "location_lookup_failed", "detail": err.Error()})
			return
		}

		if err := unlinkDiscordServer(ctx, pool, location.ID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "unlink_failed", "detail": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func canManageDiscordServerLink(ctx context.Context, pool *pgxpool.Pool, userID string) (bool, error) {
	return access.IsOperatorUser(ctx, pool, userID)
}

func resolveProducerOfficeLocation(ctx context.Context, pool *pgxpool.Pool) (struct {
	ID   string
	Slug string
	Name string
}, error) {
	var location struct {
		ID   string
		Slug string
		Name string
	}
	err := pool.QueryRow(ctx, `
		SELECT l.id::text, l.slug, l.name
		FROM venues v
		JOIN lots lo ON lo.id = v.lot_id
		JOIN locations l ON l.id = lo.location_id
		WHERE v.slug = 'producers-office'
		LIMIT 1
	`).Scan(&location.ID, &location.Slug, &location.Name)
	if err == nil {
		return location, nil
	}
	err = pool.QueryRow(ctx, `
		SELECT id::text, slug, name
		FROM locations
		WHERE slug = $1
		LIMIT 1
	`, access.DefaultLocationSlug()).Scan(&location.ID, &location.Slug, &location.Name)
	return location, err
}

func loadDiscordServerLinkRecord(ctx context.Context, pool *pgxpool.Pool, locationID string) (discordServerLinkRecord, error) {
	var record discordServerLinkRecord
	err := pool.QueryRow(ctx, `
		SELECT
			location_id::text,
			COALESCE(NULLIF(discord_guild_id, ''), ''),
			COALESCE(NULLIF(discord_guild_name, ''), ''),
			COALESCE(NULLIF(system_channel_id, ''), ''),
			COALESCE(NULLIF(system_channel_name, ''), ''),
			COALESCE(bot_verified, FALSE),
			COALESCE(active, FALSE)
		FROM auth.discord_server_links
		WHERE location_id = $1::uuid
		LIMIT 1
	`, locationID).Scan(&record.LocationID, &record.DiscordGuildID, &record.DiscordGuildName, &record.SystemChannelID, &record.SystemChannelName, &record.BotVerified, &record.Active)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return discordServerLinkRecord{}, nil
		}
		return discordServerLinkRecord{}, err
	}
	return record, nil
}

func saveDiscordServerLink(ctx context.Context, pool *pgxpool.Pool, input discordServerLinkUpsert) error {
	_, err := pool.Exec(ctx, `
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
			updated_at,
			removed_at
		)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, TRUE, $7::uuid, NOW(), NOW(), NULL)
		ON CONFLICT (location_id) DO UPDATE
		SET discord_guild_id = EXCLUDED.discord_guild_id,
			discord_guild_name = EXCLUDED.discord_guild_name,
			system_channel_id = EXCLUDED.system_channel_id,
			system_channel_name = EXCLUDED.system_channel_name,
			bot_verified = EXCLUDED.bot_verified,
			active = TRUE,
			linked_by_user_id = EXCLUDED.linked_by_user_id,
			linked_at = NOW(),
			updated_at = NOW(),
			removed_at = NULL
	`, input.LocationID, input.DiscordGuildID, input.DiscordGuildName, input.SystemChannelID, input.SystemChannelName, input.BotVerified, input.LinkedByUserID)
	return err
}

func unlinkDiscordServer(ctx context.Context, pool *pgxpool.Pool, locationID string) error {
	_, err := pool.Exec(ctx, `
		UPDATE auth.discord_server_links
		SET active = FALSE,
			bot_verified = FALSE,
			removed_at = NOW(),
			updated_at = NOW()
		WHERE location_id = $1::uuid
	`, locationID)
	return err
}

func buildDiscordServerAuthorizeURL(cfg DiscordServerLinkConfig, state string) (string, error) {
	endpoint := strings.TrimSpace(cfg.AuthorizeURL)
	if endpoint == "" {
		endpoint = discordServerLinkAuthorizeURL
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}

	q := parsed.Query()
	q.Set("client_id", strings.TrimSpace(cfg.ApplicationID))
	q.Set("scope", "bot applications.commands identify")
	q.Set("permissions", strings.TrimSpace(cfg.Permissions))
	q.Set("response_type", "code")
	q.Set("redirect_uri", strings.TrimSpace(cfg.RedirectURL))
	q.Set("state", strings.TrimSpace(state))
	q.Set("integration_type", "0")
	parsed.RawQuery = q.Encode()
	return parsed.String(), nil
}

func fetchDiscordGuild(ctx context.Context, cfg DiscordServerLinkConfig, guildID string) (discordGuild, error) {
	var guild discordGuild
	if err := discordServerLinkRequest(ctx, cfg, http.MethodGet, "/guilds/"+url.PathEscape(strings.TrimSpace(guildID)), nil, &guild); err != nil {
		return discordGuild{}, err
	}
	return guild, nil
}

func ensureDiscordSystemChannel(ctx context.Context, cfg DiscordServerLinkConfig, guildID string) (discordChannel, error) {
	var channels []discordChannel
	if err := discordServerLinkRequest(ctx, cfg, http.MethodGet, "/guilds/"+url.PathEscape(strings.TrimSpace(guildID))+"/channels", nil, &channels); err != nil {
		return discordChannel{}, err
	}

	for _, ch := range channels {
		if strings.EqualFold(strings.TrimSpace(ch.Name), discordServerLinkDefaultGuild) && ch.Type == 0 {
			return ch, nil
		}
	}

	payload := map[string]any{
		"name": discordServerLinkDefaultGuild,
		"type": 0,
	}
	var created discordChannel
	if err := discordServerLinkRequest(ctx, cfg, http.MethodPost, "/guilds/"+url.PathEscape(strings.TrimSpace(guildID))+"/channels", payload, &created); err != nil {
		return discordChannel{}, err
	}
	return created, nil
}

func discordServerLinkRequest(ctx context.Context, cfg DiscordServerLinkConfig, method, path string, payload any, out any) error {
	baseURL := strings.TrimSpace(cfg.APIBaseURL)
	if baseURL == "" {
		baseURL = discordServerLinkAPIBaseURL
	}

	var bodyReader *bytes.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(raw)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(baseURL, "/")+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bot "+strings.TrimSpace(cfg.BotToken))
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := discordServerLinkHTTPClient(cfg).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		if resp.StatusCode == http.StatusUnauthorized {
			if message, ok := errBody["message"].(string); ok && strings.TrimSpace(message) != "" {
				return fmt.Errorf("discord api unauthorized: %s", message)
			}
			return fmt.Errorf("discord api unauthorized: check bot token")
		}
		if message, ok := errBody["message"].(string); ok && strings.TrimSpace(message) != "" {
			return fmt.Errorf("discord api http %d: %s", resp.StatusCode, message)
		}
		return fmt.Errorf("discord api http %d", resp.StatusCode)
	}

	if out == nil {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func DiscordServerLinkRequest(ctx context.Context, cfg DiscordServerLinkConfig, method, path string, payload any, out any) error {
	return discordServerLinkRequest(ctx, cfg, method, path, payload, out)
}

func fallbackString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
