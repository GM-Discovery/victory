package identity

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type JoinRequest struct {
	Handle      string `json:"handle"`
	DisplayName string `json:"display_name"`
}

type JoinResponse struct {
	UserID      string `json:"user_id"`
	Handle      string `json:"handle"`
	DisplayName string `json:"display_name"`
	SessionID   string `json:"session_id"`
	Role        string `json:"role"`
}

func JoinTheCave(ctx context.Context, pool *pgxpool.Pool, req JoinRequest) (*JoinResponse, error) {
	handle := normalizeHandle(req.Handle)
	displayName := strings.TrimSpace(req.DisplayName)

	if handle == "" {
		return nil, errors.New("handle is required")
	}
	if displayName == "" {
		return nil, errors.New("display_name is required")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var userID string
	err = tx.QueryRow(ctx, `
		INSERT INTO users (handle, display_name)
		VALUES ($1, $2)
		ON CONFLICT (handle) DO UPDATE
		SET display_name = EXCLUDED.display_name,
		    last_seen_at = NOW()
		RETURNING id
	`, handle, displayName).Scan(&userID)
	if err != nil {
		return nil, err
	}

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

	const role = "audience"

	_, err = tx.Exec(ctx, `
		INSERT INTO session_participants (session_id, user_id, role)
		VALUES ($1, $2, $3::location_role)
		ON CONFLICT (session_id, user_id) DO UPDATE
		SET role = EXCLUDED.role,
		    left_at = NULL
	`, sessionID, userID, role)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE users
		SET last_seen_at = NOW()
		WHERE id = $1
	`, userID)
	if err != nil {
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
		Role:        role,
	}, nil
}

func normalizeHandle(in string) string {
	in = strings.TrimSpace(strings.ToLower(in))
	in = strings.ReplaceAll(in, " ", "_")
	return in
}