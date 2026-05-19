package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"victory/backend/internal/sessions"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	discordOAuthProvider      = "discord"
	discordOAuthDefaultScopes = "identify email"
	discordOAuthStateTTL      = 10 * time.Minute
	discordOAuthSessionTTL    = 24 * time.Hour
	discordOAuthAuthorizeURL  = "https://discord.com/oauth2/authorize"
	discordOAuthTokenURL      = "https://discord.com/api/oauth2/token"
	discordOAuthUserURL       = "https://discord.com/api/v10/users/@me"
)

type DiscordOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	Enabled      bool
	AuthorizeURL string
	TokenURL     string
	UserURL      string
	HTTPClient   *http.Client
}

type DiscordUserProfile struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	GlobalName    string `json:"global_name"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"avatar"`
	Email         string `json:"email"`
	Verified      bool   `json:"verified"`
	Locale        string `json:"locale"`
}

type discordTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

func DiscordOAuthConfigured(cfg DiscordOAuthConfig) bool {
	return cfg.Enabled &&
		strings.TrimSpace(cfg.ClientID) != "" &&
		strings.TrimSpace(cfg.ClientSecret) != "" &&
		strings.TrimSpace(cfg.RedirectURL) != ""
}

func discordScopeString(scopes []string) string {
	parts := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		parts = append(parts, scope)
	}
	if len(parts) == 0 {
		return discordOAuthDefaultScopes
	}
	return strings.Join(parts, " ")
}

func discordAuthorizeURL(cfg DiscordOAuthConfig, state string) (string, error) {
	endpoint := strings.TrimSpace(cfg.AuthorizeURL)
	if endpoint == "" {
		endpoint = discordOAuthAuthorizeURL
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}

	q := parsed.Query()
	q.Set("response_type", "code")
	q.Set("client_id", strings.TrimSpace(cfg.ClientID))
	q.Set("redirect_uri", strings.TrimSpace(cfg.RedirectURL))
	q.Set("scope", discordScopeString(cfg.Scopes))
	q.Set("state", strings.TrimSpace(state))
	parsed.RawQuery = q.Encode()
	return parsed.String(), nil
}

func newOAuthStateToken() (string, []byte, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", nil, err
	}
	raw := base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(raw))
	return raw, sum[:], nil
}

func hashOAuthToken(raw string) []byte {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return sum[:]
}

func safeReturnTo(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "/"
	}
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.Contains(raw, "://") {
		return "/"
	}
	return raw
}

func discordHTTPClient(cfg DiscordOAuthConfig) *http.Client {
	if cfg.HTTPClient != nil {
		return cfg.HTTPClient
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func HandleDiscordOAuthProviders(cfg DiscordOAuthConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true,
			"data": map[string]any{
				"discord_enabled":   DiscordOAuthConfigured(cfg),
				"discord_login_url": "/auth/discord/start",
			},
		})
	}
}

func HandleDiscordOAuthStart(pool *pgxpool.Pool, cfg DiscordOAuthConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		if !DiscordOAuthConfigured(cfg) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": "discord_oauth_not_configured"})
			return
		}

		if pool == nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "database_unavailable"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		rawState, stateHash, err := newOAuthStateToken()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "state_generation_failed"})
			return
		}

		returnTo := safeReturnTo(r.URL.Query().Get("return_to"))
		expiresAt := time.Now().UTC().Add(discordOAuthStateTTL)

		_, _ = pool.Exec(ctx, `
			DELETE FROM auth.oauth_states
			WHERE provider = $1
			  AND expires_at < NOW()
		`, discordOAuthProvider)

		_, err = pool.Exec(ctx, `
			INSERT INTO auth.oauth_states (provider, state_hash, return_to, expires_at)
			VALUES ($1, $2, $3, $4)
		`, discordOAuthProvider, stateHash, returnTo, expiresAt)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "state_store_failed"})
			return
		}

		authURL, err := discordAuthorizeURL(cfg, rawState)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "auth_url_failed"})
			return
		}

		http.Redirect(w, r, authURL, http.StatusFound)
	}
}

func HandleDiscordOAuthCallback(pool *pgxpool.Pool, cfg DiscordOAuthConfig, secureCookie bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		if !DiscordOAuthConfigured(cfg) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": "discord_oauth_not_configured"})
			return
		}

		if pool == nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "database_unavailable"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		code := strings.TrimSpace(r.URL.Query().Get("code"))
		state := strings.TrimSpace(r.URL.Query().Get("state"))
		if code == "" || state == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "missing_code_or_state"})
			return
		}

		var returnTo string
		err := pool.QueryRow(ctx, `
			UPDATE auth.oauth_states
			SET consumed_at = NOW()
			WHERE provider = $1
			  AND state_hash = $2
			  AND consumed_at IS NULL
			  AND expires_at > NOW()
			RETURNING COALESCE(NULLIF(return_to, ''), '/')
		`, discordOAuthProvider, hashOAuthToken(state)).Scan(&returnTo)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_or_expired_state"})
			return
		}

		accessToken, err := exchangeDiscordCode(ctx, discordHTTPClient(cfg), cfg, code)
		if err != nil {
			log.Printf("discord token exchange failed: %v", err)
			writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": "discord_token_exchange_failed"})
			return
		}

		discordUser, err := fetchDiscordUser(ctx, discordHTTPClient(cfg), cfg, accessToken)
		if err != nil {
			log.Printf("discord user fetch failed: %v", err)
			writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": "discord_user_fetch_failed"})
			return
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "tx_begin_failed"})
			return
		}

		userID, err := linkDiscordUser(ctx, tx, discordUser)
		if err != nil {
			_ = tx.Rollback(ctx)
			log.Printf("discord user link failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "discord_user_link_failed"})
			return
		}

		if err := tx.Commit(ctx); err != nil {
			_ = tx.Rollback(ctx)
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "tx_commit_failed"})
			return
		}

		rawSession, expiresAt, err := sessions.CreateSession(ctx, pool, userID, discordOAuthSessionTTL, r)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "session_create_failed"})
			return
		}

		sessions.SetSessionCookie(w, rawSession, expiresAt, secureCookie)
		http.Redirect(w, r, safeReturnTo(returnTo), http.StatusFound)
	}
}

func exchangeDiscordCode(ctx context.Context, client *http.Client, cfg DiscordOAuthConfig, code string) (string, error) {
	endpoint := strings.TrimSpace(cfg.TokenURL)
	if endpoint == "" {
		endpoint = discordOAuthTokenURL
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", strings.TrimSpace(code))
	form.Set("redirect_uri", strings.TrimSpace(cfg.RedirectURL))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(strings.TrimSpace(cfg.ClientID), strings.TrimSpace(cfg.ClientSecret))

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("discord token exchange http %d", resp.StatusCode)
	}

	var payload discordTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return "", errors.New("discord access token missing")
	}

	return payload.AccessToken, nil
}

func fetchDiscordUser(ctx context.Context, client *http.Client, cfg DiscordOAuthConfig, accessToken string) (DiscordUserProfile, error) {
	endpoint := strings.TrimSpace(cfg.UserURL)
	if endpoint == "" {
		endpoint = discordOAuthUserURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return DiscordUserProfile{}, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return DiscordUserProfile{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return DiscordUserProfile{}, fmt.Errorf("discord user fetch http %d", resp.StatusCode)
	}

	var profile DiscordUserProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return DiscordUserProfile{}, err
	}
	profile.ID = strings.TrimSpace(profile.ID)
	profile.Username = strings.TrimSpace(profile.Username)
	profile.GlobalName = strings.TrimSpace(profile.GlobalName)
	profile.Discriminator = strings.TrimSpace(profile.Discriminator)
	profile.Avatar = strings.TrimSpace(profile.Avatar)
	profile.Email = strings.TrimSpace(strings.ToLower(profile.Email))
	profile.Locale = strings.TrimSpace(profile.Locale)
	return profile, nil
}

func discordDisplayName(profile DiscordUserProfile) string {
	if name := strings.TrimSpace(profile.GlobalName); name != "" {
		return name
	}
	if name := strings.TrimSpace(profile.Username); name != "" {
		return name
	}
	if id := strings.TrimSpace(profile.ID); id != "" {
		suffix := id
		if len(suffix) > 8 {
			suffix = suffix[len(suffix)-8:]
		}
		return "discord_" + suffix
	}
	return "Discord User"
}

func discordHandle(profile DiscordUserProfile) string {
	if id := strings.TrimSpace(profile.ID); id != "" {
		return normalizeHandle("discord_" + id)
	}
	return "discord_user"
}

func normalizeDiscordEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

func resolveSafeEmailCandidate(ctx context.Context, tx pgx.Tx, email, excludeUserID string) (string, error) {
	email = normalizeDiscordEmail(email)
	if email == "" {
		return "", nil
	}

	query := `
		SELECT id::text
		FROM users
		WHERE lower(email) = $1
	`
	args := []any{email}
	excludeUserID = strings.TrimSpace(excludeUserID)
	if excludeUserID != "" {
		query += " AND id <> $2::uuid"
		args = append(args, excludeUserID)
	}
	query += " LIMIT 1"

	var existingUserID string
	err := tx.QueryRow(ctx, query, args...).Scan(&existingUserID)
	if err == nil {
		return "", nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	return email, nil
}

func linkDiscordUser(ctx context.Context, tx pgx.Tx, profile DiscordUserProfile) (string, error) {
	discordUserID := strings.TrimSpace(profile.ID)
	if discordUserID == "" {
		return "", errors.New("discord_user_id_required")
	}

	displayName := discordDisplayName(profile)
	handle := discordHandle(profile)
	emailCandidate, err := resolveSafeEmailCandidate(ctx, tx, profile.Email, "")
	if err != nil {
		return "", err
	}

	var userID string
	err = tx.QueryRow(ctx, `
		SELECT user_id::text
		FROM auth.discord_identities
		WHERE discord_user_id = $1
		LIMIT 1
	`, discordUserID).Scan(&userID)
	switch {
	case err == nil:
		safeEmail, err := resolveSafeEmailCandidate(ctx, tx, profile.Email, userID)
		if err != nil {
			return "", err
		}
		_, err = tx.Exec(ctx, `
			UPDATE users
			SET display_name = $2,
			    email = CASE
			      WHEN email IS NULL OR email = '' THEN COALESCE(NULLIF($3, ''), email)
			      ELSE email
			    END,
			    updated_at = NOW(),
			    last_seen_at = NOW()
			WHERE id = $1
		`, userID, displayName, safeEmail)
		if err != nil {
			return "", err
		}

		_, err = tx.Exec(ctx, `
			UPDATE auth.discord_identities
			SET username = $2,
			    global_name = $3,
			    discriminator = $4,
			    avatar = $5,
			    email = $6,
			    email_verified = $7,
			    locale = $8,
			    last_login_at = NOW(),
			    updated_at = NOW()
			WHERE discord_user_id = $1
		`, discordUserID, profile.Username, profile.GlobalName, profile.Discriminator, profile.Avatar, profile.Email, profile.Verified, profile.Locale)
		if err != nil {
			return "", err
		}
		return userID, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return "", err
	}

	insertEmail := emailCandidate
	err = tx.QueryRow(ctx, `
		INSERT INTO users (handle, display_name, email)
		VALUES ($1, $2, NULLIF($3, ''))
		RETURNING id::text
	`, handle, displayName, insertEmail).Scan(&userID)
	if err != nil {
		if insertEmail != "" {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				err = tx.QueryRow(ctx, `
					INSERT INTO users (handle, display_name)
					VALUES ($1, $2)
					RETURNING id::text
				`, handle, displayName).Scan(&userID)
			}
		}
		if err != nil {
			return "", err
		}
	}

	_, err = tx.Exec(ctx, `
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
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
	`, userID, discordUserID, profile.Username, profile.GlobalName, profile.Discriminator, profile.Avatar, profile.Email, profile.Verified, profile.Locale)
	if err != nil {
		return "", err
	}

	return userID, nil
}
