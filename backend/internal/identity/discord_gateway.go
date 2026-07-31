package identity

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"victory/backend/internal/access"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultDiscordGatewayIntents int64 = (1 << 0) | (1 << 9)

type DiscordGatewayConfig struct {
	Enabled           bool
	BotToken          string
	Intents           int64
	GatewayAPIBaseURL string
	GatewayURL        string
	HTTPClient        *http.Client
}

type DiscordGatewayStatusResponse struct {
	Configured               bool   `json:"configured"`
	Enabled                  bool   `json:"enabled"`
	Running                  bool   `json:"running"`
	Connected                bool   `json:"connected"`
	DebugEnabled             bool   `json:"debug_enabled"`
	SessionID                string `json:"session_id,omitempty"`
	BotUserID                string `json:"bot_user_id,omitempty"`
	Intents                  int64  `json:"intents"`
	MessageContentIntent     bool   `json:"message_content_intent"`
	ActiveThreadCount        int    `json:"active_thread_count"`
	LastConnectedAt          string `json:"last_connected_at,omitempty"`
	LastEventAt              string `json:"last_event_at,omitempty"`
	LastError                string `json:"last_error,omitempty"`
	MessageContentIntentNote string `json:"message_content_intent_note,omitempty"`
	UpdatedAt                string `json:"updated_at,omitempty"`
}

type DiscordGatewayStateRow struct {
	LocationID           string
	Enabled              bool
	Configured           bool
	Running              bool
	Connected            bool
	SessionID            string
	BotUserID            string
	Intents              int64
	MessageContentIntent bool
	ActiveThreadCount    int
	LastConnectedAt      *time.Time
	LastEventAt          *time.Time
	LastError            string
	UpdatedAt            time.Time
}

type DiscordGatewayDebugSettingsRow struct {
	LocationID      string
	DebugEnabled    bool
	UpdatedByUserID string
	UpdatedAt       time.Time
}

// Schema DDL lives in backend/migrations/054_kernel72_discord_gateway_surface_ddl.sql;
// this bootstrap only seeds the discord_bridge user (tests call it directly too).
func EnsureKernel39DiscordGatewaySurface(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO users (handle, display_name)
		SELECT 'discord_bridge', 'Discord Bridge'
		WHERE NOT EXISTS (
		  SELECT 1 FROM users WHERE handle = 'discord_bridge'
		);
	`)
	return err
}

func DiscordGatewayConfigured(cfg DiscordGatewayConfig) bool {
	return cfg.Enabled && strings.TrimSpace(cfg.BotToken) != ""
}

func loadDiscordGatewayState(ctx context.Context, pool *pgxpool.Pool, locationID string) (*DiscordGatewayStateRow, error) {
	var row DiscordGatewayStateRow
	err := pool.QueryRow(ctx, `
		SELECT
			location_id::text,
			COALESCE(enabled, FALSE),
			COALESCE(configured, FALSE),
			COALESCE(running, FALSE),
			COALESCE(connected, FALSE),
			COALESCE(NULLIF(session_id, ''), ''),
			COALESCE(NULLIF(bot_user_id, ''), ''),
			COALESCE(intents, 0),
			COALESCE(message_content_intent, FALSE),
			COALESCE(active_thread_count, 0),
			last_connected_at,
			last_event_at,
			COALESCE(NULLIF(last_error, ''), ''),
			updated_at
		FROM auth.discord_gateway_state
		WHERE location_id = $1::uuid
		LIMIT 1
	`, locationID).Scan(
		&row.LocationID,
		&row.Enabled,
		&row.Configured,
		&row.Running,
		&row.Connected,
		&row.SessionID,
		&row.BotUserID,
		&row.Intents,
		&row.MessageContentIntent,
		&row.ActiveThreadCount,
		&row.LastConnectedAt,
		&row.LastEventAt,
		&row.LastError,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func loadDiscordGatewayDebugSettings(ctx context.Context, pool *pgxpool.Pool, locationID string) (*DiscordGatewayDebugSettingsRow, error) {
	var row DiscordGatewayDebugSettingsRow
	err := pool.QueryRow(ctx, `
		SELECT
			location_id::text,
			COALESCE(debug_enabled, FALSE),
			COALESCE(updated_by_user_id::text, ''),
			updated_at
		FROM auth.discord_gateway_settings
		WHERE location_id = $1::uuid
		LIMIT 1
	`, locationID).Scan(&row.LocationID, &row.DebugEnabled, &row.UpdatedByUserID, &row.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func loadDiscordGatewayDebugEnabled(ctx context.Context, pool *pgxpool.Pool, locationID string) (bool, error) {
	row, err := loadDiscordGatewayDebugSettings(ctx, pool, locationID)
	if err != nil || row == nil {
		return false, err
	}
	return row.DebugEnabled, nil
}

func LoadDiscordGatewayDebugEnabled(ctx context.Context, pool *pgxpool.Pool, locationID string) (bool, error) {
	return loadDiscordGatewayDebugEnabled(ctx, pool, locationID)
}

func upsertDiscordGatewayDebugEnabled(ctx context.Context, pool *pgxpool.Pool, locationID, updatedByUserID string, enabled bool) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO auth.discord_gateway_settings (
			location_id,
			debug_enabled,
			updated_by_user_id,
			updated_at
		)
		VALUES ($1::uuid, $2, NULLIF($3, '')::uuid, NOW())
		ON CONFLICT (location_id) DO UPDATE
		SET debug_enabled = EXCLUDED.debug_enabled,
			updated_by_user_id = EXCLUDED.updated_by_user_id,
			updated_at = NOW()
	`, locationID, enabled, updatedByUserID)
	return err
}

func HandleDiscordGatewayDebug(pool *pgxpool.Pool) http.HandlerFunc {
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

		location, err := resolveProducerOfficeLocation(ctx, pool)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "location_lookup_failed", "detail": err.Error()})
			return
		}

		switch r.Method {
		case http.MethodGet:
			row, err := loadDiscordGatewayDebugSettings(ctx, pool, location.ID)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "debug_lookup_failed", "detail": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"ok": true,
				"data": map[string]any{
					"debug_enabled": row != nil && row.DebugEnabled,
				},
			})
		case http.MethodPost:
			row, err := loadDiscordGatewayDebugSettings(ctx, pool, location.ID)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "debug_lookup_failed", "detail": err.Error()})
				return
			}
			nextEnabled := true
			if row != nil {
				nextEnabled = !row.DebugEnabled
			}
			if err := upsertDiscordGatewayDebugEnabled(ctx, pool, location.ID, userID, nextEnabled); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "debug_update_failed", "detail": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"ok": true,
				"data": map[string]any{
					"debug_enabled": nextEnabled,
				},
			})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
		}
	}
}

func upsertDiscordGatewayState(ctx context.Context, pool *pgxpool.Pool, row DiscordGatewayStateRow) error {
	_, err := pool.Exec(ctx, `
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
		VALUES (
			$1::uuid,
			$2,
			$3,
			$4,
			$5,
			COALESCE(NULLIF($6, ''), ''),
			COALESCE(NULLIF($7, ''), ''),
			$8,
			$9,
			$10,
			$11,
			$12,
			COALESCE(NULLIF($13, ''), ''),
			NOW()
		)
		ON CONFLICT (location_id) DO UPDATE
		SET enabled = EXCLUDED.enabled,
			configured = EXCLUDED.configured,
			running = EXCLUDED.running,
			connected = EXCLUDED.connected,
			session_id = EXCLUDED.session_id,
			bot_user_id = EXCLUDED.bot_user_id,
			intents = EXCLUDED.intents,
			message_content_intent = EXCLUDED.message_content_intent,
			active_thread_count = EXCLUDED.active_thread_count,
			last_connected_at = EXCLUDED.last_connected_at,
			last_event_at = EXCLUDED.last_event_at,
			last_error = EXCLUDED.last_error,
			updated_at = NOW()
	`, row.LocationID, row.Enabled, row.Configured, row.Running, row.Connected, row.SessionID, row.BotUserID, row.Intents, row.MessageContentIntent, row.ActiveThreadCount, row.LastConnectedAt, row.LastEventAt, row.LastError)
	return err
}

func LoadDiscordGatewayState(ctx context.Context, pool *pgxpool.Pool, locationID string) (*DiscordGatewayStateRow, error) {
	return loadDiscordGatewayState(ctx, pool, locationID)
}

func UpsertDiscordGatewayState(ctx context.Context, pool *pgxpool.Pool, row DiscordGatewayStateRow) error {
	return upsertDiscordGatewayState(ctx, pool, row)
}

func HandleDiscordGatewayStatus(pool *pgxpool.Pool, cfg DiscordGatewayConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// Kernel 76 (K76-M02): this route answered anonymous callers with the
		// gateway's configured/enabled/connected flags, intent bitfield, thread
		// counts, and verbatim `last_error` from Discord -- an operations
		// readout of Grant's infrastructure served to the public internet.
		userID, err := currentUserID(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}
		if ok, err := access.IsOperatorUser(ctx, pool, userID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "operator_lookup_failed"})
			return
		} else if !ok {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "operator_required"})
			return
		}

		location, err := resolveProducerOfficeLocation(ctx, pool)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "location_lookup_failed", "detail": err.Error()})
			return
		}

		runtimeCfg := cfg
		if resolved, err := resolveDiscordServerLinkRuntimeConfig(ctx, pool, DiscordServerLinkConfig{
			BotToken: cfg.BotToken,
			Enabled:  cfg.Enabled,
		}); err == nil {
			if strings.TrimSpace(resolved.BotToken) != "" {
				runtimeCfg.BotToken = resolved.BotToken
			}
			runtimeCfg.Enabled = resolved.Enabled
		}

		state, err := loadDiscordGatewayState(ctx, pool, location.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "gateway_state_lookup_failed", "detail": err.Error()})
			return
		}

		runtime := DiscordGatewayStatusResponse{
			Configured:               DiscordGatewayConfigured(runtimeCfg),
			Enabled:                  runtimeCfg.Enabled,
			Running:                  false,
			Connected:                false,
			Intents:                  runtimeCfg.Intents,
			MessageContentIntent:     runtimeCfg.Intents&(1<<15) != 0,
			MessageContentIntentNote: "Discord chat intake now backfills message text over REST, so Message Content Intent is optional.",
		}
		if debugSettings, err := loadDiscordGatewayDebugSettings(ctx, pool, location.ID); err == nil && debugSettings != nil {
			runtime.DebugEnabled = debugSettings.DebugEnabled
		}
		if state != nil {
			runtime.Running = state.Running
			runtime.Connected = state.Connected
			runtime.SessionID = state.SessionID
			runtime.BotUserID = state.BotUserID
			runtime.Intents = state.Intents
			if state.Intents != 0 {
				runtime.MessageContentIntent = state.MessageContentIntent
			}
			runtime.ActiveThreadCount = state.ActiveThreadCount
			if state.LastConnectedAt != nil {
				runtime.LastConnectedAt = state.LastConnectedAt.UTC().Format(time.RFC3339)
			}
			if state.LastEventAt != nil {
				runtime.LastEventAt = state.LastEventAt.UTC().Format(time.RFC3339)
			}
			runtime.LastError = state.LastError
			runtime.UpdatedAt = state.UpdatedAt.UTC().Format(time.RFC3339)
		}

		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": runtime})
	}
}
