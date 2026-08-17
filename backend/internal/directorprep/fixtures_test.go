package directorprep

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
)

// The fixture shape here is a trimmed copy of backend/internal/socio's
// (insert user -> grant Director at the seeded location -> production ->
// Show Run -> Show), reduced to what these tests need: no Player roster, no
// Characters, because this package has no Player-facing surface to test
// against.
func openTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return dbtest.OpenTestPool(t)
}

func testSuffix(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000000")
}

func insertTestUser(t *testing.T, pool *pgxpool.Pool, handlePrefix string) string {
	t.Helper()
	ctx := context.Background()
	handle := handlePrefix + "_" + testSuffix(t)

	var userID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name) VALUES ($1, $2) RETURNING id::text
	`, handle, handlePrefix).Scan(&userID); err != nil {
		t.Fatalf("insert user %q: %v", handle, err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `DELETE FROM director_preparations WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM shows WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM show_runs WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM users WHERE id = $1`, userID)
	})
	return userID
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

func showFixture(t *testing.T, pool *pgxpool.Pool, directorUserID string) (showID, locationID string) {
	t.Helper()
	ctx := context.Background()
	suffix := testSuffix(t)

	if err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&locationID); err != nil {
		t.Fatalf("load amurray-family location: %v", err)
	}
	grantLocationRole(t, pool, locationID, directorUserID, "director")

	var productionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug) VALUES ($1, $2, $3) RETURNING id::text
	`, locationID, "K89 Production "+suffix, "k89-production-"+suffix).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id = $1`, productionID) })

	sr, err := showruns.CreateShowRun(ctx, pool, directorUserID, productionID, showruns.CreateShowRunInput{
		Title: "K89 Show Run " + suffix, Slug: "k89-show-run-" + suffix,
	})
	if err != nil {
		t.Fatalf("create show run fixture: %v", err)
	}
	s, err := shows.CreateShow(ctx, pool, directorUserID, sr.ID, shows.CreateShowInput{
		Title: "K89 Show " + suffix, Slug: "k89-show-" + suffix,
	})
	if err != nil {
		t.Fatalf("create show fixture: %v", err)
	}
	return s.ID, locationID
}
