package identity

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"victory/backend/internal/access"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DiscordServerBootstrapSummary struct {
	ApplicationID string `json:"application_id"`
	RedirectURL   string `json:"redirect_url"`
	Permissions   string `json:"permissions"`
	PublicKey     string `json:"public_key"`
	Enabled       bool   `json:"enabled"`
	BotTokenSet   bool   `json:"bot_token_set"`
	PublicKeySet  bool   `json:"public_key_set"`
	Source        string `json:"source"`
}

type DiscordServerBootstrapRequest struct {
	ApplicationID string `json:"application_id"`
	BotToken      string `json:"bot_token"`
	RedirectURL   string `json:"redirect_url"`
	Permissions   string `json:"permissions"`
	PublicKey     string `json:"public_key"`
	Enabled       bool   `json:"enabled"`
}

type discordServerLinkSettingsRow struct {
	LocationID      string
	ApplicationID   string
	BotToken        string
	RedirectURL     string
	Permissions     string
	PublicKey       string
	Enabled         bool
	UpdatedByUserID string
}

func HandleDiscordServerBootstrap(pool *pgxpool.Pool, cfg DiscordServerLinkConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()

		userID, err := currentUserID(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		allowed, err := access.IsOperatorUser(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "access_check_failed", "detail": err.Error()})
			return
		}
		if !allowed {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
			return
		}

		switch r.Method {
		case http.MethodGet:
			summary, err := loadDiscordServerBootstrapSummary(ctx, pool, cfg)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "bootstrap_lookup_failed", "detail": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": summary})
		case http.MethodPost:
			var req DiscordServerBootstrapRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_json"})
				return
			}

			req.ApplicationID = strings.TrimSpace(req.ApplicationID)
			req.BotToken = strings.TrimSpace(req.BotToken)
			req.RedirectURL = strings.TrimSpace(req.RedirectURL)
			req.Permissions = strings.TrimSpace(req.Permissions)
			req.PublicKey = strings.TrimSpace(req.PublicKey)
			if req.Permissions == "" {
				req.Permissions = "16"
			}

			location, err := resolveProducerOfficeLocation(ctx, pool)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "location_lookup_failed", "detail": err.Error()})
				return
			}

			if err := upsertDiscordServerBootstrap(ctx, pool, location.ID, userID, req); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "bootstrap_save_failed", "detail": err.Error()})
				return
			}

			summary, err := loadDiscordServerBootstrapSummary(ctx, pool, cfg)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "bootstrap_lookup_failed", "detail": err.Error()})
				return
			}
			if linkRecord, err := loadDiscordServerLinkRecord(ctx, pool, location.ID); err == nil && linkRecord.Active && strings.TrimSpace(linkRecord.DiscordGuildID) != "" {
				scheduleDiscordBootstrapReconcile(pool, cfg)
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": summary})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
		}
	}
}

func loadDiscordServerBootstrapSummary(ctx context.Context, pool *pgxpool.Pool, fallback DiscordServerLinkConfig) (DiscordServerBootstrapSummary, error) {
	cfg := fallback
	row, err := loadDiscordServerLinkSettings(ctx, pool)
	if err != nil {
		return DiscordServerBootstrapSummary{}, err
	}
	source := "env"
	if row != nil {
		source = "database"
		if strings.TrimSpace(row.ApplicationID) != "" {
			cfg.ApplicationID = row.ApplicationID
		}
		if strings.TrimSpace(cfg.BotToken) == "" && strings.TrimSpace(row.BotToken) != "" {
			cfg.BotToken = row.BotToken
		}
		if strings.TrimSpace(row.RedirectURL) != "" {
			cfg.RedirectURL = row.RedirectURL
		}
		if strings.TrimSpace(row.Permissions) != "" {
			cfg.Permissions = row.Permissions
		}
		cfg.Enabled = row.Enabled
	}

	return DiscordServerBootstrapSummary{
		ApplicationID: cfg.ApplicationID,
		RedirectURL:   cfg.RedirectURL,
		Permissions:   cfg.Permissions,
		PublicKey:     cfg.PublicKey,
		Enabled:       cfg.Enabled,
		BotTokenSet:   strings.TrimSpace(cfg.BotToken) != "",
		PublicKeySet:  strings.TrimSpace(cfg.PublicKey) != "",
		Source:        source,
	}, nil
}

func resolveDiscordServerLinkRuntimeConfig(ctx context.Context, pool *pgxpool.Pool, fallback DiscordServerLinkConfig) (DiscordServerLinkConfig, error) {
	cfg := fallback
	row, err := loadDiscordServerLinkSettings(ctx, pool)
	if err != nil {
		return DiscordServerLinkConfig{}, err
	}
	if row == nil {
		return cfg, nil
	}
	if strings.TrimSpace(row.ApplicationID) != "" {
		cfg.ApplicationID = row.ApplicationID
	}
	if strings.TrimSpace(cfg.BotToken) == "" && strings.TrimSpace(row.BotToken) != "" {
		cfg.BotToken = row.BotToken
	}
	if strings.TrimSpace(row.RedirectURL) != "" {
		cfg.RedirectURL = row.RedirectURL
	}
	if strings.TrimSpace(row.Permissions) != "" {
		cfg.Permissions = row.Permissions
	}
	if strings.TrimSpace(row.PublicKey) != "" {
		cfg.PublicKey = row.PublicKey
	}
	cfg.Enabled = row.Enabled
	return cfg, nil
}

func ResolveDiscordServerLinkRuntimeConfig(ctx context.Context, pool *pgxpool.Pool, fallback DiscordServerLinkConfig) (DiscordServerLinkConfig, error) {
	return resolveDiscordServerLinkRuntimeConfig(ctx, pool, fallback)
}

func loadDiscordServerLinkSettings(ctx context.Context, pool *pgxpool.Pool) (*discordServerLinkSettingsRow, error) {
	location, err := resolveProducerOfficeLocation(ctx, pool)
	if err != nil {
		return nil, err
	}

	var row discordServerLinkSettingsRow
	err = pool.QueryRow(ctx, `
			SELECT
				location_id::text,
				COALESCE(NULLIF(application_id, ''), ''),
				COALESCE(NULLIF(bot_token, ''), ''),
				COALESCE(NULLIF(redirect_url, ''), ''),
				COALESCE(NULLIF(permissions, ''), ''),
				COALESCE(NULLIF(public_key, ''), ''),
				COALESCE(enabled, FALSE),
				COALESCE(NULLIF(updated_by_user_id::text, ''), '')
			FROM auth.discord_server_link_settings
			WHERE location_id = $1::uuid
			LIMIT 1
		`, location.ID).Scan(&row.LocationID, &row.ApplicationID, &row.BotToken, &row.RedirectURL, &row.Permissions, &row.PublicKey, &row.Enabled, &row.UpdatedByUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &row, nil
}

func upsertDiscordServerBootstrap(ctx context.Context, pool *pgxpool.Pool, locationID, userID string, req DiscordServerBootstrapRequest) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO auth.discord_server_link_settings (
			location_id,
			application_id,
			bot_token,
			redirect_url,
			permissions,
			public_key,
			enabled,
			updated_by_user_id,
			updated_at
		)
		VALUES ($1::uuid, NULLIF($2, ''), NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), $7, $8::uuid, NOW())
		ON CONFLICT (location_id) DO UPDATE
		SET application_id = EXCLUDED.application_id,
			bot_token = EXCLUDED.bot_token,
			redirect_url = EXCLUDED.redirect_url,
			permissions = EXCLUDED.permissions,
			public_key = EXCLUDED.public_key,
			enabled = EXCLUDED.enabled,
			updated_by_user_id = EXCLUDED.updated_by_user_id,
			updated_at = NOW()
	`, locationID, req.ApplicationID, req.BotToken, req.RedirectURL, req.Permissions, req.PublicKey, req.Enabled, userID)
	return err
}
