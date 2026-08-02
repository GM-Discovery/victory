package access

import (
	"context"
	"testing"
)

// TestResolveVisibleVenuesAuditionHallOpenToAnyAuthenticatedUser proves
// Kernel 77A: the Audition Hall, restored by migration
// 083_kernel77a_audition_hall_seed_repair.sql, is visible to any
// authenticated user regardless of role or venue admission -- matching
// ResolveVisibleVenues' existing (previously silently unsatisfiable)
// `v.slug IN ('audition-hall', 'trailers')` authenticated_surface clause --
// while a genuinely restricted venue (show-runs, Producer/Director/crew
// only) stays hidden for the same plain-audience account.
func TestResolveVisibleVenuesAuditionHallOpenToAnyAuthenticatedUser(t *testing.T) {
	pool := openVisibilityTestPool(t)
	locationID := loadAmurrayFamilyLocationID(t, pool)
	audience := insertVisibilityTestUser(t, pool, "vis_audition_audience")
	grantVisibilityLocationRole(t, pool, locationID, audience, "audience")

	venues, err := ResolveVisibleVenues(context.Background(), pool, audience)
	if err != nil {
		t.Fatalf("ResolveVisibleVenues: %v", err)
	}
	slugs := visibleSlugSet(t, venues)
	if !slugs["audition-hall"] {
		t.Fatal("expected a plain audience-role account to see Audition Hall")
	}
	if !slugs["trailers"] {
		t.Fatal("expected Trailers to remain visible alongside Audition Hall (same visibility class)")
	}
	if slugs["show-runs"] {
		t.Fatal("expected a plain audience-role account to NOT see Stage Management (show-runs) -- Audition Hall's broad visibility must not leak into unrelated restricted venues")
	}

	for _, v := range venues {
		if v.Slug == "audition-hall" {
			if v.VisibleBecause != "authenticated_surface" {
				t.Fatalf("expected audition-hall visible_because=authenticated_surface, got %q", v.VisibleBecause)
			}
			if v.Name != "Audition Hall" {
				t.Fatalf("expected display name %q, got %q", "Audition Hall", v.Name)
			}
		}
	}
}

// TestAuditionHallExistsExactlyOnceAndUnderCanonicalLot proves the
// migration's idempotency and correct placement directly against the
// schema, independent of the visibility resolver.
func TestAuditionHallExistsExactlyOnceAndUnderCanonicalLot(t *testing.T) {
	pool := openVisibilityTestPool(t)
	ctx := context.Background()

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM venues WHERE slug = 'audition-hall'`).Scan(&count); err != nil {
		t.Fatalf("count audition-hall rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one audition-hall venue row, got %d", count)
	}

	var lotSlug, locationSlug string
	if err := pool.QueryRow(ctx, `
		SELECT lo.slug, l.slug
		FROM venues v
		JOIN lots lo ON lo.id = v.lot_id
		JOIN locations l ON l.id = lo.location_id
		WHERE v.slug = 'audition-hall'
	`).Scan(&lotSlug, &locationSlug); err != nil {
		t.Fatalf("load audition-hall lot/location: %v", err)
	}
	if lotSlug != "main-lot" || locationSlug != "amurray-family" {
		t.Fatalf("expected audition-hall under amurray-family/main-lot, got %s/%s", locationSlug, lotSlug)
	}
}
