package access

import (
	"context"
	"testing"
	"time"

	"victory/backend/internal/dbtest"
)

// TestCurrentDefaultLocationRoleDoesNotCrossLocations is the Kernel 97 fix
// proof for the removed CurrentLocationRole: a Producer at a second,
// unrelated Location must not resolve as Producer for THIS install's own
// default location. Before this fix, CurrentLocationRole ignored
// location_id entirely and would have returned "producer" here regardless
// -- the exact cross-Location leak spec Sec29 asks K97 to prove doesn't
// happen.
func TestCurrentDefaultLocationRoleDoesNotCrossLocations(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()

	suffix := time.Now().UTC().Format("150405.000000000")

	var otherLocationID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO locations (name, slug, is_default)
		VALUES ($1, $2, FALSE)
		RETURNING id::text
	`, "Kernel 97 Other Location "+suffix, "k97-other-location-"+suffix).Scan(&otherLocationID); err != nil {
		t.Fatalf("insert second location: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM locations WHERE id = $1`, otherLocationID) })

	var userID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name)
		VALUES ($1, $2)
		RETURNING id::text
	`, "k97_cross_loc_"+suffix, "K97 Cross-Location Test").Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID) })

	// Producer at the OTHER location only -- no membership at this
	// install's own default location at all.
	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, 'producer', TRUE)
	`, otherLocationID, userID); err != nil {
		t.Fatalf("grant producer at other location: %v", err)
	}

	role, err := CurrentDefaultLocationRole(ctx, pool, userID)
	if err != nil {
		t.Fatalf("CurrentDefaultLocationRole: %v", err)
	}
	if role != "audience" {
		t.Fatalf("expected %q (no membership at the default location), got %q -- this is the Kernel 97 cross-Location leak if it says \"producer\"", "audience", role)
	}
}
