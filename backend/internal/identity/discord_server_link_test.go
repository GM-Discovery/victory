package identity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"victory/backend/internal/sessions"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestHandleDiscordServerLinkStatusRequiresAuthentication(t *testing.T) {
	pool := openDiscordTestPool(t)

	req := httptest.NewRequest(http.MethodGet, "/api/discord/server-link/status", nil)
	rec := httptest.NewRecorder()

	HandleDiscordServerLinkStatus(pool, DiscordServerLinkConfig{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status %d", rec.Code)
	}
}

func TestDiscordServerInstallAndCallbackPersistLink(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()
	operatorHandle := "serverlink_operator"
	t.Setenv("OPERATOR_HANDLE", operatorHandle)

	ensureDiscordServerTestSchema(t, pool)

	userID := insertDiscordServerTestUser(t, pool, operatorHandle, "Server Link Operator")
	locationID := resolveDiscordServerTestLocationID(t, pool, "amurray-family")
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_link_settings WHERE location_id = $1`, locationID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.sessions WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_link_settings WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_links WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.oauth_states WHERE provider = $1`, discordServerLinkProvider)
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

	cfg := DiscordServerLinkConfig{
		ApplicationID: "app-123",
		BotToken:      "bot-token",
		RedirectURL:   "https://victory.example/auth/discord/server/callback",
		Permissions:   "16",
		Enabled:       true,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/guilds/123"):
				return jsonResponse(`{"id":"123","name":"Example Server"}`), nil
			case req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/guilds/123/channels"):
				return jsonResponse(`[]`), nil
			case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/guilds/123/channels"):
				return jsonResponse(`{"id":"456","name":"victory-system","type":0}`), nil
			default:
				t.Fatalf("unexpected discord request %s %s", req.Method, req.URL.Path)
				return nil, nil
			}
		})},
	}

	installReq := httptest.NewRequest(http.MethodGet, "/auth/discord/server/install?return_to=/venues/producers-office/", nil)
	installReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: rawSession})
	installRec := httptest.NewRecorder()
	HandleDiscordServerInstall(pool, cfg).ServeHTTP(installRec, installReq)
	if installRec.Code != http.StatusFound {
		t.Fatalf("unexpected install status %d body=%s", installRec.Code, installRec.Body.String())
	}

	redirectURL, err := url.Parse(installRec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse install redirect: %v", err)
	}
	state := redirectURL.Query().Get("state")
	if state == "" {
		t.Fatalf("missing install state")
	}

	callbackReq := httptest.NewRequest(http.MethodGet, "/auth/discord/server/callback?state="+url.QueryEscape(state)+"&guild_id=123&code=code-1", nil)
	callbackReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: rawSession})
	callbackRec := httptest.NewRecorder()
	HandleDiscordServerCallback(pool, cfg).ServeHTTP(callbackRec, callbackReq)
	if callbackRec.Code != http.StatusFound {
		t.Fatalf("unexpected callback status %d", callbackRec.Code)
	}

	var linked struct {
		DiscordGuildID    string `json:"discord_guild_id"`
		DiscordGuildName  string `json:"discord_guild_name"`
		SystemChannelID   string `json:"system_channel_id"`
		SystemChannelName string `json:"system_channel_name"`
		BotVerified       bool   `json:"bot_verified"`
		Active            bool   `json:"active"`
	}
	if err := pool.QueryRow(ctx, `
		SELECT discord_guild_id, COALESCE(discord_guild_name, ''), COALESCE(system_channel_id, ''), COALESCE(system_channel_name, ''), bot_verified, active
		FROM auth.discord_server_links
		WHERE location_id = $1
		LIMIT 1
	`, locationID).Scan(&linked.DiscordGuildID, &linked.DiscordGuildName, &linked.SystemChannelID, &linked.SystemChannelName, &linked.BotVerified, &linked.Active); err != nil {
		t.Fatalf("load linked server: %v", err)
	}
	if !linked.Active || !linked.BotVerified {
		t.Fatalf("expected active verified link, got %+v", linked)
	}
	if linked.DiscordGuildID != "123" || linked.SystemChannelID != "456" {
		t.Fatalf("unexpected linked server state: %+v", linked)
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/api/discord/server-link/status", nil)
	statusReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: rawSession})
	statusRec := httptest.NewRecorder()
	HandleDiscordServerLinkStatus(pool, cfg).ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("unexpected status code %d", statusRec.Code)
	}

	var payload struct {
		Ok   bool `json:"ok"`
		Data struct {
			Linked        bool `json:"linked"`
			DiscordServer struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"discord_server"`
			Bot struct {
				Installed bool `json:"installed"`
				Verified  bool `json:"verified"`
			} `json:"bot"`
			SystemChannel struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"system_channel"`
		} `json:"data"`
	}
	if err := json.Unmarshal(statusRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if !payload.Ok || !payload.Data.Linked || !payload.Data.Bot.Verified {
		t.Fatalf("unexpected status payload: %+v", payload)
	}
	if payload.Data.DiscordServer.ID != "123" || payload.Data.SystemChannel.ID != "456" {
		t.Fatalf("unexpected status payload: %+v", payload)
	}

	unlinkReq := httptest.NewRequest(http.MethodPost, "/api/discord/server/unlink", nil)
	unlinkReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: rawSession})
	unlinkRec := httptest.NewRecorder()
	HandleDiscordServerUnlink(pool, cfg).ServeHTTP(unlinkRec, unlinkReq)
	if unlinkRec.Code != http.StatusOK {
		t.Fatalf("unexpected unlink status %d", unlinkRec.Code)
	}
}

func TestDiscordServerBootstrapPersistsSettingsAndUnblocksInstall(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()
	operatorHandle := "serverlink_operator_bootstrap"
	t.Setenv("OPERATOR_HANDLE", operatorHandle)

	ensureDiscordServerTestSchema(t, pool)

	userID := insertDiscordServerTestUser(t, pool, operatorHandle, "Server Link Operator")
	locationID := resolveDiscordServerTestLocationID(t, pool, "amurray-family")
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_link_settings WHERE location_id = $1`, locationID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.sessions WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_link_settings WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.oauth_states WHERE provider = $1`, discordServerLinkProvider)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	rawSession, _, err := sessions.CreateSession(ctx, pool, userID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	bootstrapReq := httptest.NewRequest(http.MethodPost, "/api/discord/server/bootstrap", strings.NewReader(`{"application_id":"app-456","bot_token":"bot-456","redirect_url":"https://victory.example/auth/discord/server/callback","permissions":"16","public_key":"public-key-456","enabled":true}`))
	bootstrapReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: rawSession})
	bootstrapReq.Header.Set("Content-Type", "application/json")
	bootstrapRec := httptest.NewRecorder()
	HandleDiscordServerBootstrap(pool, DiscordServerLinkConfig{}).ServeHTTP(bootstrapRec, bootstrapReq)
	if bootstrapRec.Code != http.StatusOK {
		t.Fatalf("unexpected bootstrap status %d body=%s", bootstrapRec.Code, bootstrapRec.Body.String())
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/api/discord/server/bootstrap", nil)
	statusReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: rawSession})
	statusRec := httptest.NewRecorder()
	HandleDiscordServerBootstrap(pool, DiscordServerLinkConfig{}).ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("unexpected bootstrap GET status %d", statusRec.Code)
	}

	var summary struct {
		Ok   bool `json:"ok"`
		Data struct {
			ApplicationID string `json:"application_id"`
			RedirectURL   string `json:"redirect_url"`
			Permissions   string `json:"permissions"`
			Enabled       bool   `json:"enabled"`
			BotTokenSet   bool   `json:"bot_token_set"`
			Source        string `json:"source"`
		} `json:"data"`
	}
	if err := json.Unmarshal(statusRec.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode bootstrap summary: %v", err)
	}
	if !summary.Ok || !summary.Data.BotTokenSet || summary.Data.Source != "database" {
		t.Fatalf("unexpected bootstrap summary: %+v", summary)
	}

	cfg := DiscordServerLinkConfig{
		Enabled: true,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/guilds/789"):
				return jsonResponse(`{"id":"789","name":"Example Server"}`), nil
			case req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/guilds/789/channels"):
				return jsonResponse(`[]`), nil
			case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/guilds/789/channels"):
				return jsonResponse(`{"id":"987","name":"victory-system","type":0}`), nil
			default:
				t.Fatalf("unexpected discord request %s %s", req.Method, req.URL.Path)
				return nil, nil
			}
		})},
	}

	installReq := httptest.NewRequest(http.MethodGet, "/auth/discord/server/install?return_to=/venues/producers-office/", nil)
	installReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: rawSession})
	installRec := httptest.NewRecorder()
	HandleDiscordServerInstall(pool, cfg).ServeHTTP(installRec, installReq)
	if installRec.Code != http.StatusFound {
		t.Fatalf("unexpected install status %d body=%s", installRec.Code, installRec.Body.String())
	}
}

func TestDiscordBootstrapReconcileRestoresMappingsAndMicCommand(t *testing.T) {
	pool := openDiscordTestPool(t)
	ensureDiscordServerTestSchema(t, pool)

	locationID := resolveDiscordServerTestLocationID(t, pool, "amurray-family")
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_link_settings WHERE location_id = $1`, locationID)
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_links WHERE location_id = $1`, locationID)
	_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_channel_mappings WHERE location_id = $1`, locationID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_link_settings WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_server_links WHERE location_id = $1`, locationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_channel_mappings WHERE location_id = $1`, locationID)
	})

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
		VALUES ($1, 'app-1', 'bot-1', 'https://victory.example/auth/discord/server/callback', '16', 'public-key-1', TRUE, NOW())
	`, locationID)

	_, _ = pool.Exec(context.Background(), `
		INSERT INTO auth.discord_server_links (
			location_id,
			discord_guild_id,
			discord_guild_name,
			system_channel_id,
			system_channel_name,
			bot_verified,
			active,
			linked_at,
			updated_at
		)
		VALUES ($1, 'guild-1', 'Example Server', 'sys-1', 'victory-system', TRUE, TRUE, NOW(), NOW())
	`, locationID)

	channelsJSON := `[
		{"id":"cat-core","name":"Victory Theater","type":4},
		{"id":"sys-1","name":"victory-system","type":0},
		{"id":"ann-1","name":"victory-announcements","type":0},
		{"id":"lobby-1","name":"victory-lobby","type":0},
		{"id":"support-1","name":"victory-support","type":0},
		{"id":"the-cave-cat","name":"The Cave","type":4},
		{"id":"first-theater-cat","name":"First Theater","type":4},
		{"id":"middle-school-stage-cat","name":"Middle School Stage","type":4},
		{"id":"producers-office-cat","name":"Producer's Office","type":4},
		{"id":"directors-chair-cat","name":"The Director's Chair","type":4},
		{"id":"audition-hall-cat","name":"Audition Hall","type":4},
		{"id":"greenroom-cat","name":"The Greenroom","type":4},
		{"id":"trailers-cat","name":"Trailers","type":4},
		{"id":"workshop-cat","name":"Workshop","type":4},
		{"id":"the-cave-chat","name":"the-cave-chat","type":0,"parent_id":"the-cave-cat"},
		{"id":"first-theater-chat","name":"first-theater-chat","type":0,"parent_id":"first-theater-cat"},
		{"id":"middle-school-stage-chat","name":"middle-school-stage-chat","type":0,"parent_id":"middle-school-stage-cat"}
	]`

	cfg := DiscordServerLinkConfig{
		ApplicationID: "app-1",
		BotToken:      "bot-1",
		RedirectURL:   "https://victory.example/auth/discord/server/callback",
		PublicKey:     "public-key-1",
		Enabled:       true,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/guilds/guild-1/channels"):
				return jsonResponse(channelsJSON), nil
			case req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/applications/app-1/guilds/guild-1/commands"):
				return jsonResponse(`[]`), nil
			case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/applications/app-1/guilds/guild-1/commands"):
				return jsonResponse(`{"id":"mic-1","name":"mic"}`), nil
			default:
				t.Fatalf("unexpected discord request %s %s", req.Method, req.URL.Path)
				return nil, nil
			}
		})},
	}

	if err := ReconcileDiscordBootstrap(context.Background(), pool, cfg); err != nil {
		t.Fatalf("reconcile bootstrap: %v", err)
	}

	var mappingCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM auth.discord_channel_mappings
		WHERE location_id = $1
	`, locationID).Scan(&mappingCount); err != nil {
		t.Fatalf("count mappings: %v", err)
	}
	if mappingCount < 16 {
		t.Fatalf("expected reconciled mappings, got %d", mappingCount)
	}
}

func ensureDiscordServerTestSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()
	_, _ = pool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS auth`)
	_, _ = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS auth.discord_server_links (
			location_id uuid PRIMARY KEY REFERENCES locations(id) ON DELETE CASCADE,
			discord_guild_id text NOT NULL,
			discord_guild_name text,
			system_channel_id text,
			system_channel_name text,
			bot_verified boolean NOT NULL DEFAULT FALSE,
			active boolean NOT NULL DEFAULT TRUE,
			linked_by_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
			linked_at timestamptz,
			removed_at timestamptz,
			updated_at timestamptz NOT NULL DEFAULT now()
		)
	`)
	_, _ = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS auth.discord_server_link_settings (
			location_id uuid PRIMARY KEY REFERENCES locations(id) ON DELETE CASCADE,
			application_id text,
			bot_token text,
			redirect_url text,
			permissions text,
			public_key text NOT NULL DEFAULT '',
			enabled boolean NOT NULL DEFAULT FALSE,
			updated_by_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
			updated_at timestamptz NOT NULL DEFAULT now()
		)
	`)
	_, _ = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS auth.discord_channel_mappings (
			id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
			location_id uuid NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
			discord_server_id text NOT NULL,
			discord_channel_id text NOT NULL,
			discord_channel_name text NOT NULL,
			discord_channel_type text NOT NULL,
			mapping_kind text NOT NULL,
			victory_scope_kind text NOT NULL,
			victory_scope_id uuid,
			victory_scope_slug text,
			expected_name text NOT NULL,
			parent_discord_channel_id text,
			created_by_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
			last_verified_at timestamptz,
			created_at timestamptz NOT NULL DEFAULT now(),
			updated_at timestamptz NOT NULL DEFAULT now(),
			UNIQUE (location_id, mapping_kind, victory_scope_kind, victory_scope_slug)
		)
	`)
	_, _ = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS auth.oauth_states (
			id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
			provider text NOT NULL,
			state_hash bytea NOT NULL UNIQUE,
			return_to text,
			expires_at timestamptz NOT NULL,
			consumed_at timestamptz,
			created_at timestamptz NOT NULL DEFAULT now()
		)
	`)
	_, _ = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS auth.discord_session_threads (
			id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
			location_id uuid NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
			venue_id uuid,
			venue_slug text NOT NULL,
			session_id uuid,
			showing_id uuid,
			discord_server_id text NOT NULL,
			parent_channel_id text NOT NULL,
			thread_id text NOT NULL,
			thread_name text NOT NULL,
			started_by_user_id uuid,
			started_by_discord_user_id text,
			started_at timestamptz NOT NULL DEFAULT now(),
			showtime_at timestamptz NOT NULL,
			ended_at timestamptz,
			status text NOT NULL DEFAULT 'active',
			created_at timestamptz NOT NULL DEFAULT now(),
			updated_at timestamptz NOT NULL DEFAULT now(),
			UNIQUE (location_id, venue_slug)
		)
	`)
	_, _ = pool.Exec(ctx, `
		WITH location_row AS (
			INSERT INTO locations (name, slug)
			VALUES ('amurray.family', 'amurray-family')
			ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
			RETURNING id
		),
		location_pick AS (
			SELECT id FROM location_row
			UNION
			SELECT id FROM locations WHERE slug = 'amurray-family'
			LIMIT 1
		),
		lot_row AS (
			INSERT INTO lots (location_id, name, slug)
			SELECT id, 'main-lot', 'main-lot'
			FROM location_pick
			ON CONFLICT (location_id, slug) DO UPDATE SET name = EXCLUDED.name
			RETURNING id, location_id
		),
		lot_pick AS (
			SELECT id, location_id FROM lot_row
			UNION
			SELECT id, location_id FROM lots
			WHERE slug = 'main-lot'
			  AND location_id = (SELECT id FROM location_pick)
			LIMIT 1
		)
		INSERT INTO venues (lot_id, name, slug, kind, config, is_public, is_workshop)
		SELECT
			(SELECT id FROM lot_pick),
			'Producer''s Office',
			'producers-office',
			'office',
			'{ "surface": "permissions", "office": "producer", "requests": true, "permissions": true }'::jsonb,
			FALSE,
			FALSE
		ON CONFLICT (lot_id, slug) DO UPDATE
			SET name = EXCLUDED.name,
				kind = EXCLUDED.kind,
				config = EXCLUDED.config,
				is_public = EXCLUDED.is_public,
				is_workshop = EXCLUDED.is_workshop
	`)
	_, _ = pool.Exec(ctx, `
		WITH location_row AS (
			SELECT id
			FROM locations
			WHERE slug = 'amurray-family'
			LIMIT 1
		),
		lot_row AS (
			SELECT id
			FROM lots
			WHERE location_id = (SELECT id FROM location_row)
			  AND slug = 'main-lot'
			LIMIT 1
		)
		INSERT INTO venues (lot_id, name, slug, kind, config, is_public, is_workshop)
		SELECT
			(SELECT id FROM lot_row),
			'The Director''s Chair',
			'directors-chair',
			'plaza',
			'{ "surface": "permissions", "office": "director", "requests": true, "permissions": true }'::jsonb,
			FALSE,
			FALSE
		WHERE EXISTS (SELECT 1 FROM lot_row)
		ON CONFLICT (lot_id, slug) DO UPDATE
			SET name = EXCLUDED.name,
				kind = EXCLUDED.kind,
				config = EXCLUDED.config,
				is_public = EXCLUDED.is_public,
				is_workshop = EXCLUDED.is_workshop
	`)
}

func insertDiscordServerTestUser(t *testing.T, pool *pgxpool.Pool, handle, displayName string) string {
	t.Helper()

	var userID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO users (handle, display_name)
		VALUES ($1, $2)
		ON CONFLICT (handle) DO UPDATE
		SET display_name = EXCLUDED.display_name
		RETURNING id::text
	`, handle, displayName).Scan(&userID); err != nil {
		t.Fatalf("insert discord server test user: %v", err)
	}
	return userID
}

func resolveDiscordServerTestLocationID(t *testing.T, pool *pgxpool.Pool, slug string) string {
	t.Helper()

	var locationID string
	err := pool.QueryRow(context.Background(), `
		SELECT id::text
		FROM locations
		WHERE slug = $1
		LIMIT 1
	`, slug).Scan(&locationID)
	if err != nil {
		err = pool.QueryRow(context.Background(), `
			SELECT l.id::text
			FROM venues v
			JOIN lots lo ON lo.id = v.lot_id
			JOIN locations l ON l.id = lo.location_id
			WHERE v.slug = $1
			LIMIT 1
		`, slug).Scan(&locationID)
	}
	if err != nil {
		t.Fatalf("resolve location %q: %v", slug, err)
	}
	return locationID
}
