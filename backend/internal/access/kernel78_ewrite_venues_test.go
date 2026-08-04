package access

import (
	"context"
	"testing"
)

// TestResolveVisibleVenuesLibraryOpenToAnyAuthenticatedUser proves Kernel 78:
// the Library -- seeded since Kernel 16 but never surfaced by
// ResolveVisibleVenues for anyone but the Operator -- is now part of the
// authenticated_surface class alongside Audition Hall and Trailers. Any
// signed-in account sees it; readership authority for individual eWritings is
// enforced per publication by the /api/library routes, not by the map tile.
func TestResolveVisibleVenuesLibraryOpenToAnyAuthenticatedUser(t *testing.T) {
	pool := openVisibilityTestPool(t)
	locationID := loadAmurrayFamilyLocationID(t, pool)
	audience := insertVisibilityTestUser(t, pool, "vis_ewrite_audience")
	grantVisibilityLocationRole(t, pool, locationID, audience, "audience")

	venues, err := ResolveVisibleVenues(context.Background(), pool, audience)
	if err != nil {
		t.Fatalf("ResolveVisibleVenues: %v", err)
	}
	slugs := visibleSlugSet(t, venues)
	if !slugs["library"] {
		t.Fatal("expected a plain audience-role account to see the Library")
	}
	if slugs["writers-room"] {
		t.Fatal("expected a plain audience-role account to NOT see the Writer's Room -- authoring surface must stay Crew+ only")
	}

	for _, v := range venues {
		if v.Slug == "library" {
			if v.VisibleBecause != "authenticated_surface" {
				t.Fatalf("expected library visible_because=authenticated_surface, got %q", v.VisibleBecause)
			}
			if v.Name != "Library" {
				t.Fatalf("expected display name %q, got %q", "Library", v.Name)
			}
		}
	}
}

// TestResolveVisibleVenuesWritersRoomCrewPlusOnly proves the Writer's Room
// UNION arm role by role: producer/director/crew see the tile
// (ewrite_author_surface), cast do not. Cast is the sharpest negative --
// IsPerformerRole(cast) is true, so this catches any future refactor that
// swaps the explicit role list for the broader performer helper.
func TestResolveVisibleVenuesWritersRoomCrewPlusOnly(t *testing.T) {
	pool := openVisibilityTestPool(t)
	locationID := loadAmurrayFamilyLocationID(t, pool)

	seen := map[string]bool{"producer": true, "director": true, "crew": true, "cast": false}
	for role, wantVisible := range seen {
		user := insertVisibilityTestUser(t, pool, "vis_ewrite_"+role)
		grantVisibilityLocationRole(t, pool, locationID, user, role)

		venues, err := ResolveVisibleVenues(context.Background(), pool, user)
		if err != nil {
			t.Fatalf("ResolveVisibleVenues(%s): %v", role, err)
		}
		slugs := visibleSlugSet(t, venues)
		if slugs["writers-room"] != wantVisible {
			t.Fatalf("role %s: expected writers-room visibility %v, got %v", role, wantVisible, slugs["writers-room"])
		}
		if !wantVisible {
			continue
		}
		for _, v := range venues {
			if v.Slug == "writers-room" && v.VisibleBecause != "ewrite_author_surface" {
				t.Fatalf("role %s: expected writers-room visible_because=ewrite_author_surface, got %q", role, v.VisibleBecause)
			}
		}
	}
}

// TestWritersRoomExistsExactlyOnceAndUnderCanonicalLot proves migration 085's
// idempotency and placement directly against the schema, mirroring the
// Kernel 77A test for Audition Hall.
func TestWritersRoomExistsExactlyOnceAndUnderCanonicalLot(t *testing.T) {
	pool := openVisibilityTestPool(t)
	ctx := context.Background()

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM venues WHERE slug = 'writers-room'`).Scan(&count); err != nil {
		t.Fatalf("count writers-room rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one writers-room venue row, got %d", count)
	}

	var lotSlug, locationSlug, name string
	if err := pool.QueryRow(ctx, `
		SELECT lo.slug, l.slug, v.name
		FROM venues v
		JOIN lots lo ON lo.id = v.lot_id
		JOIN locations l ON l.id = lo.location_id
		WHERE v.slug = 'writers-room'
	`).Scan(&lotSlug, &locationSlug, &name); err != nil {
		t.Fatalf("load writers-room lot/location: %v", err)
	}
	if lotSlug != "main-lot" || locationSlug != "amurray-family" {
		t.Fatalf("expected writers-room under amurray-family/main-lot, got %s/%s", locationSlug, lotSlug)
	}
	if name != "Writer's Room" {
		t.Fatalf("expected name %q, got %q", "Writer's Room", name)
	}
}
