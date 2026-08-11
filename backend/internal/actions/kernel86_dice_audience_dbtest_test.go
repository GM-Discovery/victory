package actions

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
)

// diceAudienceFixture builds a minimal Session -> Show -> Cohort graph for
// exercising StoreDiceRoll's Kernel 86 audience resolution end to end.
// rollerID holds session_participants role 'producer' (canActDiceRoll,
// Kernel 52, gates roll/dice to Director/Producer/Operator -- see
// Construction/OperatorLogs/kernel-52-reportback.md; this is a deliberate,
// pre-existing authority boundary that Kernel 86 preserves untouched, not
// something this fixture works around) while still holding its own Show
// Cohort assignment, since session role and cohort membership are
// independent tables.
type diceAudienceFixture struct {
	sessionID     string
	showID        string
	rollerID      string
	cohortID      string
	cohortMateID  string
	otherCohortID string
	directorID    string
}

func buildDiceAudienceFixture(ctx context.Context, t *testing.T, pool *pgxpool.Pool) diceAudienceFixture {
	t.Helper()
	suffix := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "_")) + "_" + time.Now().UTC().Format("150405.000000000")

	var locationID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&locationID); err != nil {
		t.Fatalf("fixture location: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO productions (location_id, name, slug)
		SELECT $1::uuid, 'Fixture Production', 'fixture-production'
		WHERE NOT EXISTS (SELECT 1 FROM productions WHERE location_id = $1::uuid)
	`, locationID); err != nil {
		t.Fatalf("fixture production: %v", err)
	}
	var productionID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM productions WHERE location_id = $1::uuid LIMIT 1`, locationID).Scan(&productionID); err != nil {
		t.Fatalf("load fixture production: %v", err)
	}

	var fx diceAudienceFixture
	mkUser := func(prefix string) string {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO users (handle, display_name) VALUES ($1, $2) RETURNING id::text
		`, prefix+"_"+suffix, prefix).Scan(&id); err != nil {
			t.Fatalf("fixture user %s: %v", prefix, err)
		}
		t.Cleanup(func() {
			bg := context.Background()
			_, _ = pool.Exec(bg, `DELETE FROM actions WHERE actor_id = $1`, id)
			_, _ = pool.Exec(bg, `DELETE FROM show_cohort_assignments WHERE user_id = $1 OR assigned_by_user_id = $1`, id)
			_, _ = pool.Exec(bg, `DELETE FROM show_cohorts WHERE created_by_user_id = $1`, id)
			_, _ = pool.Exec(bg, `DELETE FROM session_participants WHERE user_id = $1`, id)
			_, _ = pool.Exec(bg, `UPDATE sessions SET show_id = NULL WHERE show_id IN (SELECT id FROM shows WHERE created_by_user_id = $1)`, id)
			_, _ = pool.Exec(bg, `DELETE FROM shows WHERE created_by_user_id = $1`, id)
			_, _ = pool.Exec(bg, `DELETE FROM show_runs WHERE created_by_user_id = $1`, id)
			_, _ = pool.Exec(bg, `DELETE FROM users WHERE id = $1`, id)
		})
		return id
	}
	fx.rollerID = mkUser("da_roller")
	fx.cohortMateID = mkUser("da_mate")
	fx.otherCohortID = mkUser("da_other")
	fx.directorID = mkUser("da_dir")

	var showRunID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_runs (location_id, production_id, title, slug, created_by_user_id)
		VALUES ($1::uuid, $2::uuid, 'Fixture Show Run', 'fixture-show-run-'||$3, $4::uuid)
		RETURNING id::text
	`, locationID, productionID, suffix, fx.directorID).Scan(&showRunID); err != nil {
		t.Fatalf("fixture show run: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM show_runs WHERE id = $1`, showRunID) })

	if err := pool.QueryRow(ctx, `
		INSERT INTO shows (show_run_id, slug, title, created_by_user_id)
		VALUES ($1::uuid, 'fixture-show-'||$2, 'Fixture Show', $3::uuid)
		RETURNING id::text
	`, showRunID, suffix, fx.directorID).Scan(&fx.showID); err != nil {
		t.Fatalf("fixture show: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM shows WHERE id = $1`, fx.showID) })

	if err := pool.QueryRow(ctx, `
		INSERT INTO sessions (venue_id, status, show_id)
		SELECT v.id, 'rehearsal', $1::uuid FROM venues v WHERE v.slug = 'catharsis'
		RETURNING id::text
	`, fx.showID).Scan(&fx.sessionID); err != nil {
		t.Fatalf("fixture session: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `UPDATE sessions SET status = 'closed' WHERE id = $1::uuid`, fx.sessionID)
	})

	if _, err := pool.Exec(ctx, `
		INSERT INTO showings (session_id, production_id, venue_id, status, created_by)
		SELECT s.id, p.id, s.venue_id, 'rehearsal', $2::uuid
		FROM sessions s, productions p
		WHERE s.id = $1::uuid AND p.location_id = $3::uuid
		ORDER BY p.created_at ASC LIMIT 1
		ON CONFLICT (session_id) DO NOTHING
	`, fx.sessionID, fx.rollerID, locationID); err != nil {
		t.Fatalf("fixture showing: %v", err)
	}

	roles := map[string]string{
		fx.rollerID:      "producer",
		fx.cohortMateID:  "cast",
		fx.otherCohortID: "cast",
		fx.directorID:    "director",
	}
	for userID, role := range roles {
		if _, err := pool.Exec(ctx, `
			INSERT INTO session_participants (session_id, user_id, role)
			VALUES ($1::uuid, $2::uuid, $3::location_role)
			ON CONFLICT (session_id, user_id) DO UPDATE SET role = EXCLUDED.role
		`, fx.sessionID, userID, role); err != nil {
			t.Fatalf("fixture participant %s: %v", role, err)
		}
	}

	var otherCohortID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_cohorts (show_id, serial_number, slug, name, created_by_user_id)
		VALUES ($1::uuid, 1, 'cohort-a-'||$2, 'Cohort A', $3::uuid)
		RETURNING id::text
	`, fx.showID, suffix, fx.directorID).Scan(&fx.cohortID); err != nil {
		t.Fatalf("fixture cohort A: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_cohorts (show_id, serial_number, slug, name, created_by_user_id)
		VALUES ($1::uuid, 2, 'cohort-b-'||$2, 'Cohort B', $3::uuid)
		RETURNING id::text
	`, fx.showID, suffix, fx.directorID).Scan(&otherCohortID); err != nil {
		t.Fatalf("fixture cohort B: %v", err)
	}

	assign := func(userID, cohortID string) {
		if _, err := pool.Exec(ctx, `
			INSERT INTO show_cohort_assignments (show_id, user_id, cohort_id, assigned_by_user_id)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid)
		`, fx.showID, userID, cohortID, fx.directorID); err != nil {
			t.Fatalf("fixture cohort assignment: %v", err)
		}
	}
	assign(fx.rollerID, fx.cohortID)
	assign(fx.cohortMateID, fx.cohortID)
	assign(fx.otherCohortID, otherCohortID)

	return fx
}

func TestStoreDiceRollAudienceModes(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fx := buildDiceAudienceFixture(ctx, t, pool)

	cases := []struct {
		name         string
		visibility   string
		wantMode     string
		wantCohortID string
	}{
		{"default resolves to cohort", "", "cohort", fx.cohortID},
		{"explicit cohort resolves to cohort", "cohort", "cohort", fx.cohortID},
		{"explicit show", "show", "show", ""},
		{"legacy public alias", "public", "show", ""},
		{"explicit director", "director", "director", ""},
		{"explicit private", "private", "private", ""},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := StoreDiceRoll(ctx, pool, DiceRollRequest{
				SessionID:  fx.sessionID,
				ActorID:    fx.rollerID,
				RequestID:  "req-" + strings.ToLower(t.Name()) + "-" + time.Now().UTC().Format("150405.000000000"),
				Expression: "1d6",
				Visibility: tc.visibility,
			})
			if err != nil {
				t.Fatalf("StoreDiceRoll: %v", err)
			}
			_ = i
			if mode, _ := stored.Visibility["audienceMode"].(string); mode != tc.wantMode {
				t.Fatalf("audienceMode = %q, want %q", mode, tc.wantMode)
			}
			if cohortID, _ := stored.Visibility["cohortId"].(string); cohortID != tc.wantCohortID {
				t.Fatalf("cohortId = %q, want %q", cohortID, tc.wantCohortID)
			}
			if pmode, _ := stored.Payload["visibility_mode"].(string); pmode != tc.wantMode {
				t.Fatalf("payload.visibility_mode = %q, want %q", pmode, tc.wantMode)
			}
		})
	}

	if _, err := StoreDiceRoll(ctx, pool, DiceRollRequest{
		SessionID:  fx.sessionID,
		ActorID:    fx.rollerID,
		RequestID:  "req-bogus-mode",
		Expression: "1d6",
		Visibility: "not-a-real-mode",
	}); err == nil {
		t.Fatal("expected an error for an unsupported visibility mode")
	}
}

func TestStoreDiceRollUngroupedFallsBackToShow(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fx := buildDiceAudienceFixture(ctx, t, pool)

	// The director rolls but has no cohort assignment of their own in this
	// fixture -- Kernel 86 §1.10's Ungrouped safe fallback should still
	// resolve their default/explicit-cohort request to Show, not error or
	// silently narrow to an empty cohort.
	for _, visibility := range []string{"", "cohort"} {
		stored, err := StoreDiceRoll(ctx, pool, DiceRollRequest{
			SessionID:  fx.sessionID,
			ActorID:    fx.directorID,
			RequestID:  "req-ungrouped-" + visibility + "-" + time.Now().UTC().Format("150405.000000000"),
			Expression: "1d20",
			Visibility: visibility,
		})
		if err != nil {
			t.Fatalf("StoreDiceRoll(visibility=%q): %v", visibility, err)
		}
		if mode, _ := stored.Visibility["audienceMode"].(string); mode != "show" {
			t.Fatalf("visibility=%q: audienceMode = %q, want %q", visibility, mode, "show")
		}
	}
}
