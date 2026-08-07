package identity

// Regression test for a real bug: `permission_requests` (read/written by
// requests.go and permissions.go since this feature was first built) had
// no creating migration at all -- see migrations/093_fix_missing_
// permission_requests_table.sql. Every /api/requests/create call,
// including Audition Hall's default Catharsis/cast auto-approve path,
// failed with a 500. Found via a real user report (a genuinely fresh
// account, zero location memberships, submitting the default form) rather
// than by this suite, because no test previously exercised this handler
// at all. This test proves the exact path that was broken: a brand-new
// user, with no pre-existing membership anywhere, submits the default
// request and ends up able to see Catharsis on the map afterward.

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

func permReqTestSuffix(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000")
}

func insertFreshUserWithNoMemberships(t *testing.T, handlePrefix string) string {
	t.Helper()
	pool := openDiscordTestPool(t)
	ctx := context.Background()
	handle := handlePrefix + "_" + permReqTestSuffix(t)

	var userID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name)
		VALUES ($1, $2)
		RETURNING id::text
	`, handle, handlePrefix).Scan(&userID); err != nil {
		t.Fatalf("insert user %q: %v", handle, err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `DELETE FROM auth.sessions WHERE user_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM permission_requests WHERE user_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM access_grants WHERE user_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM users WHERE id = $1`, userID)
	})
	return userID
}

func permReqSessionCookie(t *testing.T, userID string) *http.Cookie {
	t.Helper()
	pool := openDiscordTestPool(t)
	raw, _, err := sessions.CreateSession(context.Background(), pool, userID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	return &http.Cookie{Name: sessions.CookieName, Value: raw}
}

func TestFreshAccountDefaultCatharsisRequestAutoApprovesAndUnlocksVenue(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()

	// A genuinely fresh account: no location_memberships, no access_grants,
	// exactly the shape of a brand-new Discord-OAuth signup -- this is what
	// made the missing-table bug invisible until a real second account hit
	// it (the pre-existing Operator account never needed to submit a
	// request at all).
	userID := insertFreshUserWithNoMemberships(t, "permreq_fresh")
	cookie := permReqSessionCookie(t, userID)

	before, err := access.ResolveVisibleVenues(ctx, pool, userID)
	if err != nil {
		t.Fatalf("resolve visible venues (before): %v", err)
	}
	for _, v := range before {
		if v.Slug == "catharsis" {
			t.Fatalf("catharsis should not be visible before any request is made")
		}
	}

	// This mirrors Audition Hall's own default <select> values exactly
	// (frontend/venues/audition-hall/index.html): venue_slug=catharsis,
	// requested_role=cast, submitted with nothing else changed.
	req := httptest.NewRequest(http.MethodPost, "/api/requests/create", strings.NewReader(`{"venue_slug":"catharsis","requested_role":"cast","note":""}`))
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	HandleCreatePermissionRequest(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		OK           bool `json:"ok"`
		AutoApproved bool `json:"auto_approved"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	if !payload.OK || !payload.AutoApproved {
		t.Fatalf("expected ok+auto_approved response, got %+v (body=%s)", payload, rec.Body.String())
	}

	var membershipCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM location_memberships
		WHERE user_id = $1 AND role = 'cast' AND active = TRUE
	`, userID).Scan(&membershipCount); err != nil {
		t.Fatalf("count memberships: %v", err)
	}
	if membershipCount != 1 {
		t.Fatalf("expected exactly one active cast membership, got %d", membershipCount)
	}

	var grantCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM access_grants ag
		JOIN venues v ON v.id = ag.venue_id
		WHERE ag.user_id = $1 AND v.slug = 'catharsis' AND ag.revoked_at IS NULL
	`, userID).Scan(&grantCount); err != nil {
		t.Fatalf("count grants: %v", err)
	}
	if grantCount < 1 {
		t.Fatalf("expected at least one active catharsis access_grant, got %d", grantCount)
	}

	// The part that actually matters to the user: Catharsis must now show
	// up on their map, exactly the symptom that was reported as broken.
	after, err := access.ResolveVisibleVenues(ctx, pool, userID)
	if err != nil {
		t.Fatalf("resolve visible venues (after): %v", err)
	}
	found := false
	for _, v := range after {
		if v.Slug == "catharsis" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected catharsis to be visible after auto-approve, got %+v", after)
	}
}
