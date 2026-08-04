package ewrite

// Named editor grants (spec 4.5): publication-scoped, resolved
// server-side. Grant management requires CanManageEditors.

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var validGrantKinds = map[string]bool{"edit": true, "publish": true, "manage_editors": true}

// ListEditors returns a publication's grants with handles for display.
func ListEditors(ctx context.Context, pool *pgxpool.Pool, publicationID string) ([]EditorGrant, error) {
	rows, err := pool.Query(ctx, `
		SELECT g.id::text, g.publication_id::text, g.user_id::text, COALESCE(u.handle, ''),
		       g.grant_kind, COALESCE(g.granted_by::text, ''), g.created_at
		FROM ewrite_editors g
		LEFT JOIN users u ON u.id = g.user_id
		WHERE g.publication_id = $1
		ORDER BY g.created_at
	`, publicationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []EditorGrant{}
	for rows.Next() {
		var g EditorGrant
		if err := rows.Scan(&g.ID, &g.PublicationID, &g.UserID, &g.UserHandle, &g.GrantKind, &g.GrantedBy, &g.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// AddEditor grants a kind to a user identified by handle.
func AddEditor(ctx context.Context, pool *pgxpool.Pool, grantorID, publicationID, userHandle, grantKind string) (*EditorGrant, error) {
	grantKind = strings.ToLower(strings.TrimSpace(grantKind))
	if !validGrantKinds[grantKind] {
		return nil, errors.New("invalid_grant_kind")
	}
	p, err := LoadPublication(ctx, pool, publicationID)
	if err != nil {
		return nil, err
	}
	if ok, err := CanManageEditors(ctx, pool, grantorID, p); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.New("not_authorized")
	}

	var userID string
	err = pool.QueryRow(ctx, `SELECT id::text FROM users WHERE handle = $1`, strings.TrimSpace(userHandle)).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("user_not_found")
	}
	if err != nil {
		return nil, err
	}

	var g EditorGrant
	err = pool.QueryRow(ctx, `
		INSERT INTO ewrite_editors (publication_id, user_id, grant_kind, granted_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (publication_id, user_id, grant_kind) DO UPDATE SET granted_by = EXCLUDED.granted_by
		RETURNING id::text, created_at
	`, publicationID, userID, grantKind, grantorID).Scan(&g.ID, &g.CreatedAt)
	if err != nil {
		return nil, err
	}
	g.PublicationID = publicationID
	g.UserID = userID
	g.UserHandle = strings.TrimSpace(userHandle)
	g.GrantKind = grantKind
	g.GrantedBy = grantorID
	return &g, nil
}

// RemoveEditor revokes one grant by id.
func RemoveEditor(ctx context.Context, pool *pgxpool.Pool, userID, grantID string) error {
	var publicationID string
	err := pool.QueryRow(ctx, `SELECT publication_id::text FROM ewrite_editors WHERE id = $1`, grantID).Scan(&publicationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("grant_not_found")
	}
	if err != nil {
		return err
	}
	p, err := LoadPublication(ctx, pool, publicationID)
	if err != nil {
		return err
	}
	if ok, err := CanManageEditors(ctx, pool, userID, p); err != nil {
		return err
	} else if !ok {
		return errors.New("not_authorized")
	}
	_, err = pool.Exec(ctx, `DELETE FROM ewrite_editors WHERE id = $1`, grantID)
	return err
}
