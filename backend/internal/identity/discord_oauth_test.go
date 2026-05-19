package identity

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"victory/backend/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDiscordOAuthConfigured(t *testing.T) {
	if DiscordOAuthConfigured(DiscordOAuthConfig{}) {
		t.Fatalf("expected empty config to be disabled")
	}

	cfg := DiscordOAuthConfig{
		ClientID:     "client",
		ClientSecret: "secret",
		RedirectURL:  "https://example.com/auth/discord/callback",
		Enabled:      true,
	}
	if !DiscordOAuthConfigured(cfg) {
		t.Fatalf("expected populated config to be enabled")
	}
}

func TestDiscordOAuthRoutesFailSafelyWhenDisabled(t *testing.T) {
	startReq := httptest.NewRequest(http.MethodGet, "/auth/discord/start", nil)
	startRec := httptest.NewRecorder()
	HandleDiscordOAuthStart(nil, DiscordOAuthConfig{}).ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected start status %d", startRec.Code)
	}

	callbackReq := httptest.NewRequest(http.MethodGet, "/auth/discord/callback?code=x&state=y", nil)
	callbackRec := httptest.NewRecorder()
	HandleDiscordOAuthCallback(nil, DiscordOAuthConfig{}, false).ServeHTTP(callbackRec, callbackReq)
	if callbackRec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected callback status %d", callbackRec.Code)
	}
}

func TestDiscordOAuthCallbackConsumesAndRejectsReusedState(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS auth`); err != nil {
		t.Fatalf("create auth schema: %v", err)
	}
	if _, err := pool.Exec(ctx, `
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
	`); err != nil {
		t.Fatalf("ensure discord identities: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS auth.oauth_states (
			id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
			provider text NOT NULL,
			state_hash bytea NOT NULL UNIQUE,
			return_to text,
			expires_at timestamptz NOT NULL,
			consumed_at timestamptz,
			created_at timestamptz NOT NULL DEFAULT now()
		)
	`); err != nil {
		t.Fatalf("ensure oauth states: %v", err)
	}

	userHandle := "discord_test_" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000")
	var userID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (display_name, handle)
		VALUES ($1, $2)
		RETURNING id::text
	`, "Existing Discord User", userHandle).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.sessions WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_identities WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.oauth_states WHERE provider = 'discord'`)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

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
	`, userID, "123456789012345678", "bluebird", "Blue Bird", "bird@example.com", true); err != nil {
		t.Fatalf("insert discord identity: %v", err)
	}

	rawState, stateHash, err := newOAuthStateToken()
	if err != nil {
		t.Fatalf("state token: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO auth.oauth_states (provider, state_hash, return_to, expires_at)
		VALUES ($1, $2, $3, NOW() + INTERVAL '10 minutes')
	`, discordOAuthProvider, stateHash, "/account/"); err != nil {
		t.Fatalf("insert oauth state: %v", err)
	}

	cfg := DiscordOAuthConfig{
		ClientID:     "client",
		ClientSecret: "secret",
		RedirectURL:  "https://victory.example/auth/discord/callback",
		Enabled:      true,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case strings.Contains(req.URL.Path, "/oauth2/token"):
				return jsonResponse(`{"access_token":"token-1","token_type":"Bearer"}`), nil
			case strings.Contains(req.URL.Path, "/users/@me"):
				return jsonResponse(`{"id":"123456789012345678","username":"bluebird","global_name":"Blue Bird","verified":true,"email":"bird@example.com","locale":"en-US"}`), nil
			default:
				t.Fatalf("unexpected request path %q", req.URL.Path)
				return nil, nil
			}
		})},
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/discord/callback?code=code-1&state="+url.QueryEscape(rawState), nil)
	rec := httptest.NewRecorder()
	HandleDiscordOAuthCallback(pool, cfg, false).ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("unexpected callback status %d", rec.Code)
	}
	if got := rec.Header().Get("Set-Cookie"); !strings.Contains(got, "victory_session=") {
		t.Fatalf("missing session cookie: %q", got)
	}

	var consumedAt time.Time
	if err := pool.QueryRow(ctx, `
		SELECT consumed_at
		FROM auth.oauth_states
		WHERE provider = $1 AND state_hash = $2
	`, discordOAuthProvider, hashOAuthToken(rawState)).Scan(&consumedAt); err != nil {
		t.Fatalf("check consumed state: %v", err)
	}
	if consumedAt.IsZero() {
		t.Fatalf("expected state to be consumed")
	}

	reuseReq := httptest.NewRequest(http.MethodGet, "/auth/discord/callback?code=code-1&state="+url.QueryEscape(rawState), nil)
	reuseRec := httptest.NewRecorder()
	HandleDiscordOAuthCallback(pool, cfg, false).ServeHTTP(reuseRec, reuseReq)
	if reuseRec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected reused-state status %d", reuseRec.Code)
	}
}

func TestDiscordAuthorizeURLIncludesRequiredParams(t *testing.T) {
	cfg := DiscordOAuthConfig{
		ClientID:     "client-123",
		RedirectURL:  "https://victory.example/auth/discord/callback",
		Scopes:       []string{"identify", "email"},
		AuthorizeURL: "https://discord.example/oauth2/authorize",
	}

	raw, err := discordAuthorizeURL(cfg, "state-token")
	if err != nil {
		t.Fatalf("authorize url error: %v", err)
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse authorize url: %v", err)
	}

	if got := parsed.Scheme + "://" + parsed.Host + parsed.Path; got != "https://discord.example/oauth2/authorize" {
		t.Fatalf("unexpected authorize endpoint: %s", got)
	}

	q := parsed.Query()
	for _, key := range []string{"response_type", "client_id", "redirect_uri", "scope", "state"} {
		if q.Get(key) == "" {
			t.Fatalf("missing query parameter %q", key)
		}
	}

	if q.Get("response_type") != "code" {
		t.Fatalf("unexpected response_type %q", q.Get("response_type"))
	}
	if q.Get("client_id") != "client-123" {
		t.Fatalf("unexpected client_id %q", q.Get("client_id"))
	}
	if q.Get("redirect_uri") != "https://victory.example/auth/discord/callback" {
		t.Fatalf("unexpected redirect_uri %q", q.Get("redirect_uri"))
	}
	if q.Get("scope") != "identify email" {
		t.Fatalf("unexpected scope %q", q.Get("scope"))
	}
	if q.Get("state") != "state-token" {
		t.Fatalf("unexpected state %q", q.Get("state"))
	}
}

func TestSafeReturnTo(t *testing.T) {
	cases := map[string]string{
		"":                    "/",
		"/account/":           "/account/",
		"  /venues/greenroom": "/venues/greenroom",
		"https://evil.test/":  "/",
		"//evil.test/path":    "/",
		"account":             "/",
	}

	for input, want := range cases {
		if got := safeReturnTo(input); got != want {
			t.Fatalf("safeReturnTo(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestDiscordDisplayAndHandleFallbacks(t *testing.T) {
	if got := discordDisplayName(DiscordUserProfile{}); got != "Discord User" {
		t.Fatalf("unexpected empty display name fallback %q", got)
	}

	profile := DiscordUserProfile{
		ID:       "123456789012345678",
		Username: "bluebird",
	}

	if got := discordDisplayName(profile); got != "bluebird" {
		t.Fatalf("unexpected display name %q", got)
	}
	if got := discordHandle(profile); got != "discord_123456789012345678" {
		t.Fatalf("unexpected handle %q", got)
	}
}

func TestDiscordHTTPHelpers(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case strings.Contains(req.URL.Path, "/oauth2/token"):
				if got := req.Header.Get("Authorization"); !strings.HasPrefix(got, "Basic ") {
					t.Fatalf("expected basic auth header, got %q", got)
				}
				return jsonResponse(`{"access_token":"token-1","token_type":"Bearer"}`), nil
			case strings.Contains(req.URL.Path, "/users/@me"):
				if got := req.Header.Get("Authorization"); got != "Bearer token-1" {
					t.Fatalf("expected bearer token header, got %q", got)
				}
				return jsonResponse(`{"id":"1","username":"bird","global_name":"Bird","verified":true,"email":"bird@example.com","locale":"en-US"}`), nil
			default:
				t.Fatalf("unexpected request path %q", req.URL.Path)
				return nil, nil
			}
		}),
	}

	cfg := DiscordOAuthConfig{
		ClientID:     "client",
		ClientSecret: "secret",
		RedirectURL:  "https://victory.example/auth/discord/callback",
	}

	token, err := exchangeDiscordCode(testContext(t), client, cfg, "code-1")
	if err != nil {
		t.Fatalf("exchange code: %v", err)
	}
	if token != "token-1" {
		t.Fatalf("unexpected token %q", token)
	}

	profile, err := fetchDiscordUser(testContext(t), client, cfg, token)
	if err != nil {
		t.Fatalf("fetch discord user: %v", err)
	}
	if profile.ID != "1" || profile.Username != "bird" || profile.Email != "bird@example.com" || !profile.Verified {
		t.Fatalf("unexpected profile: %#v", profile)
	}
}

func TestHashOAuthTokenDeterministic(t *testing.T) {
	if got := hashOAuthToken(" state "); len(got) != 32 {
		t.Fatalf("unexpected hash length: %d", len(got))
	}
	if got1, got2 := hashOAuthToken("same"), hashOAuthToken("same"); !bytes.Equal(got1, got2) {
		t.Fatalf("expected stable hashes")
	}
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	return context.Background()
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func openDiscordTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	pool, err := db.NewPool(context.Background(), "postgres://victory:REDACTED@127.0.0.1:5432/victory?sslmode=disable")
	if err != nil {
		t.Skipf("postgres unavailable for integration test: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Skipf("postgres unavailable for integration test: %v", err)
	}
	return pool
}
