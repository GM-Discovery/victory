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

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestHandleAccountMeRequiresAuthentication(t *testing.T) {
	pool := openDiscordTestPool(t)

	req := httptest.NewRequest(http.MethodGet, "/api/account/me", nil)
	rec := httptest.NewRecorder()

	HandleAccountMe(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "not_authenticated") {
		t.Fatalf("unexpected body %q", body)
	}
}

func TestHandleAccountMeReturnsDiscordMembershipAuthority(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()

	ensureAccountTestSchema(t, pool)

	userID := insertAccountTestUser(t, pool, "account_producer_"+time.Now().UTC().Format("150405.000000"), "Account Producer")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.sessions WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_identities WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	locationID := resolveAccountTestLocationID(t, pool, "amurray-family")

	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, 'producer', TRUE)
	`, locationID, userID); err != nil {
		t.Fatalf("insert membership: %v", err)
	}

	discordUserID := "887766554433221100"
	if _, err := pool.Exec(ctx, `
		INSERT INTO auth.discord_identities (
			user_id,
			discord_user_id,
			username,
			global_name,
			email,
			email_verified
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, userID, discordUserID, "grant", "Grant", "grant@example.com", true); err != nil {
		t.Fatalf("insert discord identity: %v", err)
	}

	raw, _, err := sessions.CreateSession(ctx, pool, userID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/account/me", nil)
	req.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: raw})
	rec := httptest.NewRecorder()

	HandleAccountMe(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "access_token") || strings.Contains(body, "refresh_token") || strings.Contains(body, "client_secret") {
		t.Fatalf("response leaked a secret-like field: %s", body)
	}

	var payload struct {
		Ok   bool `json:"ok"`
		Data struct {
			User struct {
				ID          string `json:"id"`
				Handle      string `json:"handle"`
				DisplayName string `json:"display_name"`
			} `json:"user"`
			Auth struct {
				DiscordLinked bool `json:"discord_linked"`
				Discord       *struct {
					DiscordUserID string `json:"discord_user_id"`
					Username      string `json:"username"`
					GlobalName    string `json:"global_name"`
				} `json:"discord"`
			} `json:"auth"`
			Memberships []struct {
				LocationID   string `json:"location_id"`
				LocationSlug string `json:"location_slug"`
				LocationName string `json:"location_name"`
				Role         string `json:"role"`
			} `json:"memberships"`
			Authority struct {
				IsProducer        bool     `json:"is_producer"`
				ProducerLocations []string `json:"producer_locations"`
				CurrentRole       string   `json:"current_role"`
			} `json:"authority"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.Ok {
		t.Fatalf("expected ok response")
	}
	if payload.Data.User.ID != userID {
		t.Fatalf("unexpected user id %q", payload.Data.User.ID)
	}
	if !payload.Data.Auth.DiscordLinked {
		t.Fatalf("expected discord link to be reported")
	}
	if payload.Data.Auth.Discord == nil || payload.Data.Auth.Discord.Username != "grant" || payload.Data.Auth.Discord.GlobalName != "Grant" {
		t.Fatalf("unexpected discord payload: %+v", payload.Data.Auth.Discord)
	}
	if !payload.Data.Authority.IsProducer {
		t.Fatalf("expected producer status to be true")
	}
	if payload.Data.Authority.CurrentRole != "producer" {
		t.Fatalf("unexpected current role %q", payload.Data.Authority.CurrentRole)
	}
	if len(payload.Data.Authority.ProducerLocations) != 1 || payload.Data.Authority.ProducerLocations[0] != "amurray-family" {
		t.Fatalf("unexpected producer locations: %+v", payload.Data.Authority.ProducerLocations)
	}
	if len(payload.Data.Memberships) != 1 || payload.Data.Memberships[0].Role != "producer" {
		t.Fatalf("unexpected memberships: %+v", payload.Data.Memberships)
	}
}

func TestHandleAccountMeReportsUnlinkedAccount(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()

	userID := insertAccountTestUser(t, pool, "account_guest_"+time.Now().UTC().Format("150405.000000"), "Account Guest")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.sessions WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	raw, _, err := sessions.CreateSession(ctx, pool, userID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/account/me", nil)
	req.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: raw})
	rec := httptest.NewRecorder()

	HandleAccountMe(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d", rec.Code)
	}

	var payload struct {
		Ok   bool `json:"ok"`
		Data struct {
			Auth struct {
				DiscordLinked bool `json:"discord_linked"`
				Discord       any  `json:"discord"`
			} `json:"auth"`
			Authority struct {
				IsProducer        bool     `json:"is_producer"`
				ProducerLocations []string `json:"producer_locations"`
			} `json:"authority"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.Ok {
		t.Fatalf("expected ok response")
	}
	if payload.Data.Auth.DiscordLinked {
		t.Fatalf("expected discord to be unlinked")
	}
	if payload.Data.Auth.Discord != nil {
		t.Fatalf("expected discord payload to be omitted")
	}
	if payload.Data.Authority.IsProducer {
		t.Fatalf("expected non-producer account")
	}
	if len(payload.Data.Authority.ProducerLocations) != 0 {
		t.Fatalf("expected no producer locations, got %+v", payload.Data.Authority.ProducerLocations)
	}
}

func ensureAccountTestSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()
	_, _ = pool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS auth`)
	_, _ = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS auth.discord_identities (
			id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			discord_user_id text NOT NULL UNIQUE,
			username text,
			global_name text,
			discriminator text,
			avatar text,
			email text,
			email_verified boolean,
			locale text,
			last_login_at timestamptz,
			created_at timestamptz NOT NULL DEFAULT now(),
			updated_at timestamptz NOT NULL DEFAULT now()
		)
	`)
}

func insertAccountTestUser(t *testing.T, pool *pgxpool.Pool, handle, displayName string) string {
	t.Helper()

	var userID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO users (handle, display_name)
		VALUES ($1, $2)
		RETURNING id::text
	`, handle, displayName).Scan(&userID); err != nil {
		t.Fatalf("insert account test user: %v", err)
	}
	return userID
}

func resolveAccountTestLocationID(t *testing.T, pool *pgxpool.Pool, slug string) string {
	t.Helper()

	var locationID string
	if err := pool.QueryRow(context.Background(), `
		SELECT id::text
		FROM locations
		WHERE slug = $1
		LIMIT 1
	`, slug).Scan(&locationID); err != nil {
		t.Fatalf("resolve location %q: %v", slug, err)
	}
	return locationID
}
