package identity

// Regression test for a real bug (reported live, not found by this suite):
// after a Director approved a Cast permission request via Audition Hall,
// the requester's own account still showed as "audience". Root cause:
// HandleRespondPermissionRequest's approve path for director/cast/crew
// wrote only to `memberships` (the newer, production-scoped table used by
// character/participation features) and never to `location_memberships` --
// the table account.go's AccountSummary and access.CurrentLocationRole*
// actually read to answer "what is this account's role". The two tables
// are not kept in sync automatically; this handler has to write both, the
// same way the older auto-approve path in requests.go already does.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"victory/backend/internal/access"
	"victory/backend/internal/sessions"
)

func TestApproveCastPermissionRequestGrantsLocationMembership(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()
	suffix := time.Now().UTC().Format("150405.000000")

	var locationID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO locations (slug, name) VALUES ($1, $2) RETURNING id::text
	`, "castapproval-loc-"+suffix, "Cast Approval Test Location").Scan(&locationID); err != nil {
		t.Fatalf("insert location: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM locations WHERE id = $1`, locationID) })

	var lotID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO lots (location_id, name, slug) VALUES ($1, 'Test Lot', $2) RETURNING id::text
	`, locationID, "castapproval-lot-"+suffix).Scan(&lotID); err != nil {
		t.Fatalf("insert lot: %v", err)
	}

	venueSlug := "castapproval-venue-" + suffix
	if _, err := pool.Exec(ctx, `
		INSERT INTO venues (lot_id, name, slug) VALUES ($1, 'Test Venue', $2)
	`, lotID, venueSlug); err != nil {
		t.Fatalf("insert venue: %v", err)
	}

	var productionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug) VALUES ($1, 'Test Production', $2) RETURNING id::text
	`, locationID, "castapproval-prod-"+suffix).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}

	directorID := insertAccountTestUser(t, pool, "castapproval_director_"+suffix, "Director")
	castID := insertAccountTestUser(t, pool, "castapproval_cast_"+suffix, "Cast Hopeful")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = ANY($1)`, []string{directorID, castID})
	})

	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, 'director', TRUE)
	`, locationID, directorID); err != nil {
		t.Fatalf("insert director membership: %v", err)
	}

	var requestID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO permission_requests (user_id, venue_slug, requested_role, note, status)
		VALUES ($1, $2, 'cast', '', 'pending')
		RETURNING id::text
	`, castID, venueSlug).Scan(&requestID); err != nil {
		t.Fatalf("insert permission request: %v", err)
	}

	// Sanity check on the reported symptom: before approval, the account's
	// own resolved role at this location is "audience" (the no-membership
	// fallback) -- not yet meaningful, but confirms the baseline.
	before, err := access.CurrentLocationRoleForLocation(ctx, pool, castID, locationID)
	if err != nil {
		t.Fatalf("resolve role before approval: %v", err)
	}
	if before != "audience" {
		t.Fatalf("expected baseline role audience before approval, got %q", before)
	}

	raw, _, err := sessions.CreateSession(ctx, pool, directorID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create director session: %v", err)
	}

	body, _ := json.Marshal(respondPermissionRequestInput{RequestID: requestID, Decision: "approve"})
	req := httptest.NewRequest(http.MethodPost, "/api/requests/respond", strings.NewReader(string(body)))
	req.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: raw})
	rec := httptest.NewRecorder()
	HandleRespondPermissionRequest(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 approving request, got %d: %s", rec.Code, rec.Body.String())
	}

	// The bug: this table (production-scoped, drives character/participation
	// features) got the row -- that's why the bug was easy to miss, other
	// cast features "worked".
	var membershipsCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM memberships
		WHERE user_id = $1 AND location_id = $2 AND role = 'cast' AND production_id = $3 AND active = TRUE
	`, castID, locationID, productionID).Scan(&membershipsCount); err != nil {
		t.Fatalf("count memberships: %v", err)
	}
	if membershipsCount != 1 {
		t.Fatalf("expected exactly one active cast membership row, got %d", membershipsCount)
	}

	// The fix: location_memberships -- what the account page and every
	// location-role gate actually read -- must also have the row now.
	var locationMembershipsCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM location_memberships
		WHERE user_id = $1 AND location_id = $2 AND role = 'cast' AND active = TRUE
	`, castID, locationID).Scan(&locationMembershipsCount); err != nil {
		t.Fatalf("count location_memberships: %v", err)
	}
	if locationMembershipsCount != 1 {
		t.Fatalf("expected exactly one active cast location_membership row, got %d", locationMembershipsCount)
	}

	// End to end: the account's own resolved role must now read "cast",
	// not "audience" -- this is the actual reported symptom.
	after, err := access.CurrentLocationRoleForLocation(ctx, pool, castID, locationID)
	if err != nil {
		t.Fatalf("resolve role after approval: %v", err)
	}
	if after != "cast" {
		t.Fatalf("expected role cast after approval, got %q", after)
	}
}
