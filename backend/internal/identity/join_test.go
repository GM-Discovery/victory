package identity

import (
	"context"
	"testing"
	"time"

	"victory/backend/internal/access"
)

// TestResolveRoleForUserOperatorWinsOverLesserLocationRole is the Kernel 101
// real-bug closure: resolveRoleForUser used to only promote to "producer"
// for a genuine Operator when the underlying location_memberships role was
// specifically "audience" -- checked as a narrow fallback, not first. An
// Operator who also held a real, non-audience location role (Cast, Crew,
// anything but Producer/Director/Audience) got that lesser role back
// instead of their actual Operator authority, for every session join that
// goes through this path (Catharsis, First Theater, the-cave). Confirmed
// live: Grant's account resolved "cast" here despite being a genuine
// Operator, because his one real location_memberships row is "cast", not
// "audience".
func TestResolveRoleForUserOperatorWinsOverLesserLocationRole(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()

	userID := insertAccountTestUser(t, pool, "join_operator_cast_"+time.Now().UTC().Format("150405.000000000"), "Join Operator Cast")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	locationID, err := access.DefaultLocationID(ctx, pool)
	if err != nil {
		t.Fatalf("resolve default location id: %v", err)
	}
	if locationID == "" {
		t.Fatalf("test setup invalid: no default location found")
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, 'cast', TRUE)
	`, locationID, userID); err != nil {
		t.Fatalf("grant cast location_memberships row: %v", err)
	}

	// Before granting Operator: confirm the real location role is genuinely
	// "cast" -- the exact non-"audience" shape that used to defeat the old
	// promotion check.
	role, err := resolveRoleForUser(ctx, pool, userID)
	if err != nil {
		t.Fatalf("resolveRoleForUser (pre-operator): %v", err)
	}
	if role != "cast" {
		t.Fatalf("test setup invalid: expected plain role %q before granting operator, got %q", "cast", role)
	}

	// Grant Operator via the same OPERATOR_HANDLE env mechanism
	// access.IsOperatorUser reads, matching every other test of Operator
	// behavior in this package.
	var handle string
	if err := pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, userID).Scan(&handle); err != nil {
		t.Fatalf("load handle: %v", err)
	}
	t.Setenv("OPERATOR_HANDLE", handle)
	t.Setenv("OPERATOR_USER_ID", "")

	role, err = resolveRoleForUser(ctx, pool, userID)
	if err != nil {
		t.Fatalf("resolveRoleForUser (operator): %v", err)
	}
	if role != "producer" {
		t.Fatalf("expected Operator to resolve as %q despite the real \"cast\" location role, got %q (this is the bug if it says \"cast\")", "producer", role)
	}
}
