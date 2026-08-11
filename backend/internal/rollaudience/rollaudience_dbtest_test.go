package rollaudience

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
)

// fixture is a minimal Session -> Show -> Cohort graph, built with raw SQL
// (matching actions/token_dbtest_test.go's createTokenFixture convention)
// rather than the full showruns/tickets roster-invite flow -- Resolve,
// VisibleToViewer, and LiveRecipients only ever touch sessions.show_id,
// show_cohorts/show_cohort_assignments, and session_participants, so
// there's no need to drive the heavier Show Run roster machinery to
// exercise them.
type fixture struct {
	sessionID    string
	showID       string
	directorID   string
	cohortAID    string
	cohortBID    string
	cohortAUser1 string
	cohortAUser2 string
	cohortBUser1 string
	ungroupedID  string
}

func buildFixture(ctx context.Context, t *testing.T, pool *pgxpool.Pool) fixture {
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

	var fx fixture
	mkUser := func(prefix string) string {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO users (handle, display_name) VALUES ($1, $2) RETURNING id::text
		`, prefix+"_"+suffix, prefix).Scan(&id); err != nil {
			t.Fatalf("fixture user %s: %v", prefix, err)
		}
		t.Cleanup(func() {
			bg := context.Background()
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
	fx.directorID = mkUser("ra_dir")
	fx.cohortAUser1 = mkUser("ra_coa1")
	fx.cohortAUser2 = mkUser("ra_coa2")
	fx.cohortBUser1 = mkUser("ra_cob1")
	fx.ungroupedID = mkUser("ra_ungr")

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

	roles := map[string]string{
		fx.directorID:   "director",
		fx.cohortAUser1: "cast",
		fx.cohortAUser2: "cast",
		fx.cohortBUser1: "cast",
		fx.ungroupedID:  "cast",
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

	if err := pool.QueryRow(ctx, `
		INSERT INTO show_cohorts (show_id, serial_number, slug, name, created_by_user_id)
		VALUES ($1::uuid, 1, 'cohort-a-'||$2, 'Cohort A', $3::uuid)
		RETURNING id::text
	`, fx.showID, suffix, fx.directorID).Scan(&fx.cohortAID); err != nil {
		t.Fatalf("fixture cohort A: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_cohorts (show_id, serial_number, slug, name, created_by_user_id)
		VALUES ($1::uuid, 2, 'cohort-b-'||$2, 'Cohort B', $3::uuid)
		RETURNING id::text
	`, fx.showID, suffix, fx.directorID).Scan(&fx.cohortBID); err != nil {
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
	assign(fx.cohortAUser1, fx.cohortAID)
	assign(fx.cohortAUser2, fx.cohortAID)
	assign(fx.cohortBUser1, fx.cohortBID)

	return fx
}

func TestResolve(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fx := buildFixture(ctx, t, pool)

	cases := []struct {
		name       string
		actorID    string
		requested  string
		wantMode   string
		wantCohort string
	}{
		{"explicit cohort member resolves own cohort", fx.cohortAUser1, "cohort", ModeCohort, fx.cohortAID},
		{"default (empty) resolves cohort for a grouped actor", fx.cohortAUser1, "", ModeCohort, fx.cohortAID},
		{"default falls back to show for an ungrouped actor", fx.ungroupedID, "", ModeShow, ""},
		{"explicit cohort falls back to show for an ungrouped actor", fx.ungroupedID, "cohort", ModeShow, ""},
		{"legacy public alias resolves to show", fx.cohortAUser1, "public", ModeShow, ""},
		{"explicit show", fx.cohortAUser1, "show", ModeShow, ""},
		{"explicit director", fx.cohortAUser1, "director", ModeDirector, ""},
		{"explicit private", fx.cohortAUser1, "private", ModePrivate, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mode := NormalizeMode(tc.requested)
			d, err := Resolve(ctx, pool, fx.sessionID, tc.actorID, mode)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if d.Mode != tc.wantMode {
				t.Fatalf("mode = %q, want %q", d.Mode, tc.wantMode)
			}
			if d.CohortID != tc.wantCohort {
				t.Fatalf("cohortID = %q, want %q", d.CohortID, tc.wantCohort)
			}
		})
	}

	if _, err := Resolve(ctx, pool, fx.sessionID, fx.cohortAUser1, "bogus-mode"); err == nil {
		t.Fatal("expected an error normalizing an already-invalid mode string")
	}
}

func TestVisibleToViewer(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fx := buildFixture(ctx, t, pool)

	cohortRoll := Decision{Mode: ModeCohort, CohortID: fx.cohortAID}
	directorRoll := Decision{Mode: ModeDirector}
	privateRoll := Decision{Mode: ModePrivate}
	showRoll := Decision{Mode: ModeShow}

	cases := []struct {
		name    string
		d       Decision
		actorID string
		viewer  string
		role    string
		want    bool
	}{
		{"cohort: actor always sees own roll", cohortRoll, fx.cohortAUser1, fx.cohortAUser1, "cast", true},
		{"cohort: same-cohort member sees it", cohortRoll, fx.cohortAUser1, fx.cohortAUser2, "cast", true},
		{"cohort: other cohort does not see it", cohortRoll, fx.cohortAUser1, fx.cohortBUser1, "cast", false},
		{"cohort: ungrouped viewer does not see it", cohortRoll, fx.cohortAUser1, fx.ungroupedID, "cast", false},
		{"cohort: director sees every cohort", cohortRoll, fx.cohortAUser1, fx.directorID, "director", true},
		{"director mode: other player cannot see it", directorRoll, fx.cohortAUser1, fx.cohortAUser2, "cast", false},
		{"director mode: director sees it", directorRoll, fx.cohortAUser1, fx.directorID, "director", true},
		{"director mode: actor sees own roll", directorRoll, fx.cohortAUser1, fx.cohortAUser1, "cast", true},
		{"private: only the actor sees it", privateRoll, fx.cohortAUser1, fx.cohortAUser1, "cast", true},
		{"private: director does NOT see another user's private roll", privateRoll, fx.cohortAUser1, fx.directorID, "director", false},
		{"private: other player does not see it", privateRoll, fx.cohortAUser1, fx.cohortAUser2, "cast", false},
		{"show: everyone sees it", showRoll, fx.cohortAUser1, fx.cohortBUser1, "cast", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := VisibleToViewer(ctx, pool, tc.d, tc.actorID, fx.showID, tc.viewer, tc.role)
			if err != nil {
				t.Fatalf("VisibleToViewer: %v", err)
			}
			if got != tc.want {
				t.Fatalf("VisibleToViewer = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLiveRecipients(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fx := buildFixture(ctx, t, pool)

	contains := func(ids []string, id string) bool {
		for _, v := range ids {
			if v == id {
				return true
			}
		}
		return false
	}

	t.Run("show mode delegates to session broadcast", func(t *testing.T) {
		ids, useBroadcast, err := LiveRecipients(ctx, pool, Decision{Mode: ModeShow}, fx.cohortAUser1, fx.sessionID)
		if err != nil {
			t.Fatalf("LiveRecipients: %v", err)
		}
		if !useBroadcast {
			t.Fatal("expected useSessionBroadcast=true for show mode")
		}
		if len(ids) != 0 {
			t.Fatalf("expected no enumerated recipients for show mode, got %v", ids)
		}
	})

	t.Run("cohort mode includes cohort members and director, excludes other cohort", func(t *testing.T) {
		ids, useBroadcast, err := LiveRecipients(ctx, pool, Decision{Mode: ModeCohort, CohortID: fx.cohortAID}, fx.cohortAUser1, fx.sessionID)
		if err != nil {
			t.Fatalf("LiveRecipients: %v", err)
		}
		if useBroadcast {
			t.Fatal("expected useSessionBroadcast=false for cohort mode")
		}
		for _, want := range []string{fx.cohortAUser1, fx.cohortAUser2, fx.directorID} {
			if !contains(ids, want) {
				t.Fatalf("expected %s in recipients, got %v", want, ids)
			}
		}
		if contains(ids, fx.cohortBUser1) {
			t.Fatalf("cohort B member must not receive cohort A's roll, got %v", ids)
		}
	})

	t.Run("director mode includes only the actor and director+", func(t *testing.T) {
		ids, useBroadcast, err := LiveRecipients(ctx, pool, Decision{Mode: ModeDirector}, fx.cohortAUser1, fx.sessionID)
		if err != nil {
			t.Fatalf("LiveRecipients: %v", err)
		}
		if useBroadcast {
			t.Fatal("expected useSessionBroadcast=false for director mode")
		}
		if !contains(ids, fx.cohortAUser1) || !contains(ids, fx.directorID) {
			t.Fatalf("expected actor and director in recipients, got %v", ids)
		}
		if contains(ids, fx.cohortAUser2) || contains(ids, fx.cohortBUser1) {
			t.Fatalf("director mode leaked to a non-director player, got %v", ids)
		}
	})

	t.Run("private mode is the actor alone", func(t *testing.T) {
		ids, useBroadcast, err := LiveRecipients(ctx, pool, Decision{Mode: ModePrivate}, fx.cohortAUser1, fx.sessionID)
		if err != nil {
			t.Fatalf("LiveRecipients: %v", err)
		}
		if useBroadcast {
			t.Fatal("expected useSessionBroadcast=false for private mode")
		}
		if len(ids) != 1 || ids[0] != fx.cohortAUser1 {
			t.Fatalf("expected exactly [actor], got %v", ids)
		}
	})
}

func TestIsDirectorPlus(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fx := buildFixture(ctx, t, pool)

	if ok, err := IsDirectorPlus(ctx, pool, fx.sessionID, fx.directorID); err != nil || !ok {
		t.Fatalf("expected director to be Director+, ok=%v err=%v", ok, err)
	}
	if ok, err := IsDirectorPlus(ctx, pool, fx.sessionID, fx.cohortAUser1); err != nil || ok {
		t.Fatalf("expected a cast participant to not be Director+, ok=%v err=%v", ok, err)
	}
}
