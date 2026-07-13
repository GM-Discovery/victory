package access

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
)

func openVisibilityTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return dbtest.OpenTestPool(t)
}

func visibilityTestSuffix(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000")
}

func insertVisibilityTestUser(t *testing.T, pool *pgxpool.Pool, handlePrefix string) string {
	t.Helper()
	ctx := context.Background()
	handle := handlePrefix + "_" + visibilityTestSuffix(t)

	var userID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name)
		VALUES ($1, $2)
		RETURNING id::text
	`, handle, handlePrefix).Scan(&userID); err != nil {
		t.Fatalf("insert user %q: %v", handle, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM show_run_roster_members WHERE user_id = $1 OR added_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM show_runs WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})
	return userID
}

func grantVisibilityLocationRole(t *testing.T, pool *pgxpool.Pool, locationID, userID, role string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, $3::location_role, TRUE)
		ON CONFLICT (location_id, user_id, role) DO UPDATE SET active = TRUE
	`, locationID, userID, role); err != nil {
		t.Fatalf("grant location role %q: %v", role, err)
	}
}

func loadAmurrayFamilyLocationID(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var locationID string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&locationID); err != nil {
		t.Fatalf("load amurray-family location: %v", err)
	}
	return locationID
}

func visibleSlugSet(t *testing.T, venues []VisibleVenue) map[string]bool {
	t.Helper()
	out := make(map[string]bool, len(venues))
	for _, v := range venues {
		out[v.Slug] = true
	}
	return out
}

func TestResolveVisibleVenuesTrailersOpenToAnyAuthenticatedUser(t *testing.T) {
	pool := openVisibilityTestPool(t)
	locationID := loadAmurrayFamilyLocationID(t, pool)
	audience := insertVisibilityTestUser(t, pool, "vis_audience")
	grantVisibilityLocationRole(t, pool, locationID, audience, "audience")

	venues, err := ResolveVisibleVenues(context.Background(), pool, audience)
	if err != nil {
		t.Fatalf("ResolveVisibleVenues: %v", err)
	}
	slugs := visibleSlugSet(t, venues)
	if !slugs["trailers"] {
		t.Fatal("expected a plain audience-role account to see Trailers")
	}
}

func TestResolveVisibleVenuesThirdPlaceGatedOnReadiness(t *testing.T) {
	pool := openVisibilityTestPool(t)
	locationID := loadAmurrayFamilyLocationID(t, pool)
	user := insertVisibilityTestUser(t, pool, "vis_thirdplace")
	grantVisibilityLocationRole(t, pool, locationID, user, "audience")

	prev := thirdPlaceReadinessChecker
	t.Cleanup(func() { thirdPlaceReadinessChecker = prev })

	SetThirdPlaceReadinessChecker(func(ctx context.Context, pool *pgxpool.Pool, userID string) (bool, error) {
		return false, nil
	})
	venues, err := ResolveVisibleVenues(context.Background(), pool, user)
	if err != nil {
		t.Fatalf("ResolveVisibleVenues: %v", err)
	}
	if visibleSlugSet(t, venues)["third-place"] {
		t.Fatal("expected third-place hidden when readiness checker reports false")
	}

	SetThirdPlaceReadinessChecker(func(ctx context.Context, pool *pgxpool.Pool, userID string) (bool, error) {
		return true, nil
	})
	venues, err = ResolveVisibleVenues(context.Background(), pool, user)
	if err != nil {
		t.Fatalf("ResolveVisibleVenues: %v", err)
	}
	if !visibleSlugSet(t, venues)["third-place"] {
		t.Fatal("expected third-place visible when readiness checker reports true")
	}
}

func TestResolveVisibleVenuesShowRunsBackstageOnly(t *testing.T) {
	pool := openVisibilityTestPool(t)
	locationID := loadAmurrayFamilyLocationID(t, pool)

	audience := insertVisibilityTestUser(t, pool, "vis_sr_audience")
	grantVisibilityLocationRole(t, pool, locationID, audience, "audience")

	producer := insertVisibilityTestUser(t, pool, "vis_sr_producer")
	grantVisibilityLocationRole(t, pool, locationID, producer, "producer")

	venues, err := ResolveVisibleVenues(context.Background(), pool, audience)
	if err != nil {
		t.Fatalf("ResolveVisibleVenues(audience): %v", err)
	}
	if visibleSlugSet(t, venues)["show-runs"] {
		t.Fatal("expected Audience-only membership to not see Stage Management (show-runs)")
	}

	venues, err = ResolveVisibleVenues(context.Background(), pool, producer)
	if err != nil {
		t.Fatalf("ResolveVisibleVenues(producer): %v", err)
	}
	if !visibleSlugSet(t, venues)["show-runs"] {
		t.Fatal("expected Producer to see Stage Management (show-runs)")
	}
}

func TestResolveVisibleVenuesShowRunsCrewException(t *testing.T) {
	pool := openVisibilityTestPool(t)
	locationID := loadAmurrayFamilyLocationID(t, pool)
	ctx := context.Background()

	producer := insertVisibilityTestUser(t, pool, "vis_crew_producer")
	grantVisibilityLocationRole(t, pool, locationID, producer, "producer")

	crewUser := insertVisibilityTestUser(t, pool, "vis_crew_member")
	grantVisibilityLocationRole(t, pool, locationID, crewUser, "audience")

	suffix := visibilityTestSuffix(t)
	var productionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug)
		VALUES ($1, $2, $3)
		RETURNING id::text
	`, locationID, "Vis Test Production "+suffix, "vis-test-production-"+suffix).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id = $1`, productionID) })

	var showRunID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_runs (location_id, production_id, title, slug, status, created_by_user_id)
		VALUES ($1, $2, $3, $4, 'active', $5)
		RETURNING id::text
	`, locationID, productionID, "Vis Test Run "+suffix, "vis-test-run-"+suffix, producer).Scan(&showRunID); err != nil {
		t.Fatalf("insert show run: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM show_runs WHERE id = $1`, showRunID) })

	if _, err := pool.Exec(ctx, `
		INSERT INTO show_run_roster_members (show_run_id, user_id, role, added_by_user_id, program_visible)
		VALUES ($1, $2, 'crew', $3, TRUE)
	`, showRunID, crewUser, producer); err != nil {
		t.Fatalf("insert crew roster row: %v", err)
	}

	venues, err := ResolveVisibleVenues(ctx, pool, crewUser)
	if err != nil {
		t.Fatalf("ResolveVisibleVenues(crew): %v", err)
	}
	if !visibleSlugSet(t, venues)["show-runs"] {
		t.Fatal("expected a Show Run crew roster row to unlock the Stage Management (show-runs) tile")
	}
}
