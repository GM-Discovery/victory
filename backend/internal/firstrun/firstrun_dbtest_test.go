package firstrun

import (
	"context"
	"testing"

	"victory/backend/internal/dbtest"
)

// TestBootstrapFirstOperatorLeavesShowLive is the Kernel 101 closure for a
// real bug found live on murray-vserver's first genuine fresh install:
// BootstrapFirstOperator started a Session via showtime.Start (which only
// ever transitions the Session/Showing to "live") but never touched the
// Show's own status column, leaving it at CreateShow's "draft" default
// forever. A Show stuck at "draft" does not read as "mounted" to a
// genuinely fresh Audience viewer -- exactly the onboarding experience
// this function exists to deliver. No test existed for this function at
// all before this one.
func TestBootstrapFirstOperatorLeavesShowLive(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()

	var userID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name) VALUES ($1, $1) RETURNING id::text
	`, "firstrun_test_operator").Scan(&userID); err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `DELETE FROM sessions WHERE show_id IN (SELECT id FROM shows WHERE created_by_user_id = $1)`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM shows WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM show_runs WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM productions WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM messages WHERE to_user_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM users WHERE id = $1`, userID)
	})

	BootstrapFirstOperator(ctx, pool, userID)

	var status string
	var actualStartAt *string
	if err := pool.QueryRow(ctx, `
		SELECT status, actual_start_at::text FROM shows WHERE slug = 'the-locked-courtyard' AND created_by_user_id = $1
	`, userID).Scan(&status, &actualStartAt); err != nil {
		t.Fatalf("load bootstrapped show: %v", err)
	}
	if status != "live" {
		t.Fatalf("expected the bootstrapped Show's own status to be \"live\", got %q -- this is the exact bug: showtime.Start only sets the Session/Showing live, never the Show itself", status)
	}
	if actualStartAt == nil {
		t.Fatal("expected actual_start_at to be set once the Show is live")
	}

	var sessionStatus string
	if err := pool.QueryRow(ctx, `
		SELECT se.status FROM sessions se
		JOIN shows s ON s.id = se.show_id
		WHERE s.slug = 'the-locked-courtyard' AND s.created_by_user_id = $1
	`, userID).Scan(&sessionStatus); err != nil {
		t.Fatalf("load session for bootstrapped show: %v", err)
	}
	if sessionStatus != "rehearsal" {
		t.Fatalf("expected the Session to be in rehearsal (Start's own normal behavior), got %q", sessionStatus)
	}
}
