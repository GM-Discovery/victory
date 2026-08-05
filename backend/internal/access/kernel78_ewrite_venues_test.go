package access

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// insertVisibilityTestCharacterCard creates a minimal character card owned
// by userID at locationID, for proving the Library's Catharsis-character
// gate (Kernel 79 operator amendment).
func insertVisibilityTestCharacterCard(t *testing.T, pool *pgxpool.Pool, ownerUserID, locationID string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO character_cards (owner_user_id, location_id, name)
		VALUES ($1, $2, 'Visibility Test Character')
		RETURNING id::text
	`, ownerUserID, locationID).Scan(&id); err != nil {
		t.Fatalf("insert character card: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM character_cards WHERE id = $1`, id)
	})
	return id
}

// insertVisibilityTestLocation creates a disposable second location, for
// proving the Library's Catharsis-character gate is scoped to Catharsis's
// own location and not "any character anywhere."
func insertVisibilityTestLocation(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	slug := "vis-test-loc-" + visibilityTestSuffix(t)
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO locations (slug, name) VALUES ($1, 'Visibility Test Location') RETURNING id::text
	`, slug).Scan(&id); err != nil {
		t.Fatalf("insert test location: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM locations WHERE id = $1`, id)
	})
	return id
}

// catharsisLocationID resolves the location that owns the seeded Catharsis
// venue, the same join ResolveVisibleVenues uses for the Library gate.
func catharsisLocationID(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		SELECT l.location_id::text FROM venues v JOIN lots l ON l.id = v.lot_id WHERE v.slug = 'catharsis'
	`).Scan(&id); err != nil {
		t.Fatalf("load catharsis location: %v", err)
	}
	return id
}

// TestResolveVisibleVenuesLibraryGatedOnCatharsisCharacter proves the
// Kernel 79 operator amendment: the Library is no longer open to every
// authenticated account (Kernel 78's original authenticated_surface
// behavior) -- it only appears once the user has a character card at
// Catharsis's own location. Readership authority for individual eWritings
// is still enforced per publication by the /api/library routes, not by
// this map-tile gate.
func TestResolveVisibleVenuesLibraryGatedOnCatharsisCharacter(t *testing.T) {
	pool := openVisibilityTestPool(t)
	locationID := loadAmurrayFamilyLocationID(t, pool)
	audience := insertVisibilityTestUser(t, pool, "vis_ewrite_audience")
	grantVisibilityLocationRole(t, pool, locationID, audience, "audience")

	venues, err := ResolveVisibleVenues(context.Background(), pool, audience)
	if err != nil {
		t.Fatalf("ResolveVisibleVenues: %v", err)
	}
	slugs := visibleSlugSet(t, venues)
	if slugs["library"] {
		t.Fatal("expected an audience-role account with no Catharsis character to NOT see the Library")
	}
	if slugs["writers-room"] {
		t.Fatal("expected a plain audience-role account to NOT see the Writer's Room -- authoring surface must stay Director+ only")
	}

	insertVisibilityTestCharacterCard(t, pool, audience, catharsisLocationID(t, pool))

	venues, err = ResolveVisibleVenues(context.Background(), pool, audience)
	if err != nil {
		t.Fatalf("ResolveVisibleVenues after character creation: %v", err)
	}
	slugs = visibleSlugSet(t, venues)
	if !slugs["library"] {
		t.Fatal("expected the Library to become visible once the user has a character at Catharsis's location")
	}
	for _, v := range venues {
		if v.Slug == "library" {
			if v.VisibleBecause != "catharsis_character_surface" {
				t.Fatalf("expected library visible_because=catharsis_character_surface, got %q", v.VisibleBecause)
			}
			if v.Name != "Library" {
				t.Fatalf("expected display name %q, got %q", "Library", v.Name)
			}
		}
	}
}

// TestResolveVisibleVenuesLibraryIgnoresCharacterAtOtherLocation proves the
// gate is scoped to Catharsis's own location specifically, not "any
// character anywhere" (that broader rule is what Greenroom already uses).
func TestResolveVisibleVenuesLibraryIgnoresCharacterAtOtherLocation(t *testing.T) {
	pool := openVisibilityTestPool(t)
	locationID := loadAmurrayFamilyLocationID(t, pool)
	user := insertVisibilityTestUser(t, pool, "vis_ewrite_other_loc")
	grantVisibilityLocationRole(t, pool, locationID, user, "audience")

	otherLocationID := insertVisibilityTestLocation(t, pool)
	insertVisibilityTestCharacterCard(t, pool, user, otherLocationID)

	venues, err := ResolveVisibleVenues(context.Background(), pool, user)
	if err != nil {
		t.Fatalf("ResolveVisibleVenues: %v", err)
	}
	if visibleSlugSet(t, venues)["library"] {
		t.Fatal("a character card at an unrelated location must not unlock the Library")
	}
}

// TestResolveVisibleVenuesWritersRoomDirectorPlusOnly proves the Writer's
// Room UNION arm role by role: producer/director see the tile
// (ewrite_author_surface); crew and cast do not. Kernel 79 narrowed this
// from the original Kernel 78 Crew+ default -- crew is the sharpest
// negative here since it used to be positive.
func TestResolveVisibleVenuesWritersRoomDirectorPlusOnly(t *testing.T) {
	pool := openVisibilityTestPool(t)
	locationID := loadAmurrayFamilyLocationID(t, pool)

	seen := map[string]bool{"producer": true, "director": true, "crew": false, "cast": false}
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
