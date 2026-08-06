package storyboards

// Shared dbtest helpers, following the pattern already duplicated per
// package (see ewrite/store_dbtest_test.go, showruns/showruns_test.go) --
// this repo does not centralize these beyond dbtest.OpenTestPool itself.

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func testSuffix(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000")
}

func insertTestUser(t *testing.T, pool *pgxpool.Pool, prefix string) string {
	t.Helper()
	ctx := context.Background()
	handle := prefix + "_" + testSuffix(t)
	var userID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name) VALUES ($1, $2) RETURNING id::text
	`, handle, prefix).Scan(&userID); err != nil {
		t.Fatalf("insert user %q: %v", handle, err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM storyboard_cards WHERE author_user_id = $1`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM storyboard_grants WHERE user_id = $1 OR granted_by = $1`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM storyboards WHERE owner_user_id = $1`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})
	return userID
}

func amurrayLocation(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&id); err != nil {
		t.Fatalf("load amurray-family: %v", err)
	}
	return id
}

func grantLocationRole(t *testing.T, pool *pgxpool.Pool, locationID, userID, role string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, $3::location_role, TRUE)
		ON CONFLICT (location_id, user_id, role) DO UPDATE SET active = TRUE
	`, locationID, userID, role); err != nil {
		t.Fatalf("grant location role %q: %v", role, err)
	}
}

func mustCreateBoard(t *testing.T, pool *pgxpool.Pool, ownerID, title string) *Storyboard {
	t.Helper()
	b, err := CreateBoard(context.Background(), pool, ownerID, title, "", nil, "", nil)
	if err != nil {
		t.Fatalf("create board %q: %v", title, err)
	}
	return b
}
