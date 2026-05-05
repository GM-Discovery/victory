package identity

import (
	"context"
	"errors"
	"strings"

	"victory/backend/internal/sessions"
	"victory/backend/internal/showings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type JoinRequest struct {
	Handle      string `json:"handle"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

type JoinResponse struct {
	UserID      string `json:"user_id"`
	Handle      string `json:"handle"`
	DisplayName string `json:"display_name"`
	SessionID   string `json:"session_id"`
	Role        string `json:"role"`
}

func JoinTheCave(ctx context.Context, pool *pgxpool.Pool, req JoinRequest, sessionCookie string) (*JoinResponse, error) {
	displayName := strings.TrimSpace(req.DisplayName)

	userID, handle, resolvedRole, err := resolveJoiningUser(ctx, pool, req, sessionCookie)
	if err != nil {
		return nil, err
	}

	if displayName == "" {
		displayName = handle
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var sessionID string
	err = tx.QueryRow(ctx, `
		SELECT s.id
		FROM sessions s
		JOIN venues v ON v.id = s.venue_id
		WHERE v.slug = 'the-cave'
		  AND s.status IN ('rehearsal', 'live')
		ORDER BY s.started_at DESC
		LIMIT 1
	`).Scan(&sessionID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO session_participants (session_id, user_id, role)
		VALUES ($1, $2, $3::location_role)
		ON CONFLICT (session_id, user_id) DO UPDATE
		SET role = EXCLUDED.role,
		    left_at = NULL
	`, sessionID, userID, resolvedRole)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE users
		SET display_name = $2,
		    last_seen_at = NOW()
		WHERE id = $1
	`, userID, displayName)
	if err != nil {
		return nil, err
	}

	if _, err := showings.EnsureForSession(ctx, tx, sessionID, userID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &JoinResponse{
		UserID:      userID,
		Handle:      handle,
		DisplayName: displayName,
		SessionID:   sessionID,
		Role:        resolvedRole,
	}, nil
}

func resolveJoiningUser(ctx context.Context, pool *pgxpool.Pool, req JoinRequest, sessionCookie string) (string, string, string, error) {
	if strings.TrimSpace(sessionCookie) != "" {
		rec, err := sessions.GetSessionByRawToken(ctx, pool, sessionCookie)
		if err == nil {
			var handle string
			err = pool.QueryRow(ctx, `
				SELECT handle
				FROM users
				WHERE id = $1
				LIMIT 1
			`, rec.UserID).Scan(&handle)
			if err != nil {
				return "", "", "", err
			}

			role, err := resolveRoleForUser(ctx, pool, rec.UserID)
			if err != nil {
				return "", "", "", err
			}

			return rec.UserID, handle, role, nil
		}
	}

	handle := normalizeHandle(req.Handle)
	if handle == "" {
		return "", "", "", errors.New("handle is required")
	}

	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = handle
	}

	var userID string
	err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name)
		VALUES ($1, $2)
		ON CONFLICT (handle) DO UPDATE
		SET display_name = EXCLUDED.display_name,
		    last_seen_at = NOW()
		RETURNING id
	`, handle, displayName).Scan(&userID)
	if err != nil {
		return "", "", "", err
	}

	return userID, handle, "audience", nil
}

func resolveRoleForUser(ctx context.Context, pool *pgxpool.Pool, userID string) (string, error) {
	var role string

	err := pool.QueryRow(ctx, `
		SELECT m.role::text
		FROM memberships m
		WHERE m.user_id = $1
		  AND m.active = TRUE
		ORDER BY
		  CASE m.role
		    WHEN 'producer' THEN 1
		    WHEN 'director' THEN 2
		    WHEN 'cast' THEN 3
		    WHEN 'crew' THEN 4
		    WHEN 'audience' THEN 5
		    ELSE 99
		  END,
		  m.created_at ASC
		LIMIT 1
	`, userID).Scan(&role)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "audience", nil
		}
		return "", err
	}

	switch role {
	case "producer", "director", "cast", "crew", "audience":
		return role, nil
	default:
		return "audience", nil
	}
}

func normalizeHandle(in string) string {
	in = strings.TrimSpace(strings.ToLower(in))
	in = strings.ReplaceAll(in, " ", "_")
	return in
}
