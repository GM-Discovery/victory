package sessions

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const CookieName = "victory_session"

type SessionRecord struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

func NewRawToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func HashToken(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}

func CreateSession(ctx context.Context, pool *pgxpool.Pool, userID string, ttl time.Duration, r *http.Request) (string, time.Time, error) {
	raw, err := NewRawToken()
	if err != nil {
		return "", time.Time{}, err
	}

	expiresAt := time.Now().UTC().Add(ttl)
	tokenHash := HashToken(raw)

	ip := clientIP(r)
	userAgent := r.UserAgent()

	_, err = pool.Exec(ctx, `
		INSERT INTO auth.sessions (user_id, token_hash, expires_at, ip, user_agent)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, tokenHash, expiresAt, ip, userAgent)
	if err != nil {
		return "", time.Time{}, err
	}

	return raw, expiresAt, nil
}

func GetSessionByRawToken(ctx context.Context, pool *pgxpool.Pool, raw string) (*SessionRecord, error) {
	tokenHash := HashToken(raw)

	row := pool.QueryRow(ctx, `
		SELECT id, user_id, expires_at, revoked_at
		FROM auth.sessions
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
		LIMIT 1
	`, tokenHash)

	var rec SessionRecord
	err := row.Scan(&rec.ID, &rec.UserID, &rec.ExpiresAt, &rec.RevokedAt)
	if err != nil {
		return nil, err
	}

	_, _ = pool.Exec(ctx, `
		UPDATE auth.sessions
		SET last_seen_at = NOW()
		WHERE id = $1
	`, rec.ID)

	return &rec, nil
}

func RevokeSessionByRawToken(ctx context.Context, pool *pgxpool.Pool, raw string) error {
	tokenHash := HashToken(raw)

	_, err := pool.Exec(ctx, `
		UPDATE auth.sessions
		SET revoked_at = NOW()
		WHERE token_hash = $1
		  AND revoked_at IS NULL
	`, tokenHash)

	return err
}

func RevokeAllUserSessions(ctx context.Context, pool *pgxpool.Pool, userID string) error {
	_, err := pool.Exec(ctx, `
		UPDATE auth.sessions
		SET revoked_at = NOW()
		WHERE user_id = $1
		  AND revoked_at IS NULL
	`, userID)

	return err
}

func SetSessionCookie(w http.ResponseWriter, raw string, expiresAt time.Time, secure bool) {
	sameSite := http.SameSiteLaxMode
	if secure {
		sameSite = http.SameSiteNoneMode
	}

	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    raw,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		Expires:  expiresAt,
	})
}

func ClearSessionCookie(w http.ResponseWriter, secure bool) {
	sameSite := http.SameSiteLaxMode
	if secure {
		sameSite = http.SameSiteNoneMode
	}

	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

func ReadSessionCookie(r *http.Request) (string, error) {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return "", err
	}
	return c.Value, nil
}

func clientIP(r *http.Request) string {
	xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}

	return strings.TrimSpace(r.RemoteAddr)
}
