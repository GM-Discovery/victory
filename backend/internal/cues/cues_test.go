package cues

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/scenes"
	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
)

func openCuesTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return dbtest.OpenTestPool(t)
}

func testSuffix(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000")
}

func insertCuesTestUser(t *testing.T, pool *pgxpool.Pool, handlePrefix string) string {
	t.Helper()
	ctx := context.Background()
	handle := handlePrefix + "_" + testSuffix(t)

	var userID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name)
		VALUES ($1, $2)
		RETURNING id::text
	`, handle, handlePrefix).Scan(&userID); err != nil {
		t.Fatalf("insert user %q: %v", handle, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM cue_executions WHERE triggered_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM cues WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `
			UPDATE sessions SET show_id = NULL WHERE show_id IN (SELECT id FROM shows WHERE created_by_user_id = $1)
		`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM show_scene_placements WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM scenes WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM shows WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM show_run_audience_blocks WHERE user_id = $1 OR blocked_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM show_run_roster_members WHERE user_id = $1 OR added_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM show_runs WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.sessions WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})
	return userID
}

func grantLocationRole(t *testing.T, pool *pgxpool.Pool, locationID, userID, role string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, $3::location_role, TRUE)
		ON CONFLICT (location_id, user_id, role) DO UPDATE SET active = TRUE
	`, locationID, userID, role); err != nil {
		t.Fatalf("grant location role %q: %v", role, err)
	}
}

// cueFixture builds a full location -> production -> show run -> show ->
// (two) scene placements chain, since a fresh database has none of these.
type cueFixture struct {
	locationID   string
	showRunID    string
	showID       string
	placementID  string
	placement2ID string
}

func buildCueFixture(t *testing.T, pool *pgxpool.Pool, producerUserID string) cueFixture {
	t.Helper()
	ctx := context.Background()
	suffix := testSuffix(t) + "_" + time.Now().UTC().Format("150405.000000000")

	var f cueFixture
	if err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&f.locationID); err != nil {
		t.Fatalf("load amurray-family location: %v", err)
	}
	grantLocationRole(t, pool, f.locationID, producerUserID, "producer")

	var productionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug) VALUES ($1, $2, $3) RETURNING id::text
	`, f.locationID, "Cue Test Production "+suffix, "cue-test-production-"+suffix).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id = $1`, productionID) })

	sr, err := showruns.CreateShowRun(ctx, pool, producerUserID, productionID, showruns.CreateShowRunInput{
		Title: "Cue Test Show Run " + suffix, Slug: "cue-test-show-run-" + suffix,
	})
	if err != nil {
		t.Fatalf("create show run fixture: %v", err)
	}
	f.showRunID = sr.ID

	s, err := shows.CreateShow(ctx, pool, producerUserID, sr.ID, shows.CreateShowInput{
		Title: "Cue Test Show " + suffix, Slug: "cue-test-show-" + suffix,
	})
	if err != nil {
		t.Fatalf("create show fixture: %v", err)
	}
	f.showID = s.ID

	scene1, err := scenes.CreateScene(ctx, pool, producerUserID, f.locationID, productionID, scenes.CreateSceneInput{
		Title: "Cue Test Scene A", Slug: "cue-test-scene-a-" + suffix,
	})
	if err != nil {
		t.Fatalf("create scene A: %v", err)
	}
	p1, err := scenes.CreatePlacement(ctx, pool, producerUserID, f.showID, scenes.CreatePlacementInput{SceneID: scene1.ID})
	if err != nil {
		t.Fatalf("create placement A: %v", err)
	}
	f.placementID = p1.ID

	scene2, err := scenes.CreateScene(ctx, pool, producerUserID, f.locationID, productionID, scenes.CreateSceneInput{
		Title: "Cue Test Scene B", Slug: "cue-test-scene-b-" + suffix,
	})
	if err != nil {
		t.Fatalf("create scene B: %v", err)
	}
	p2, err := scenes.CreatePlacement(ctx, pool, producerUserID, f.showID, scenes.CreatePlacementInput{SceneID: scene2.ID})
	if err != nil {
		t.Fatalf("create placement B: %v", err)
	}
	f.placement2ID = p2.ID

	return f
}

func addRoster(t *testing.T, pool *pgxpool.Pool, actorUserID, showRunID, targetUserID, role string) {
	t.Helper()
	ctx := context.Background()
	if err := ensureProfile(ctx, pool, targetUserID); err != nil {
		t.Fatalf("ensure profile for roster member: %v", err)
	}
	var profileID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM player_profile_workbooks WHERE user_id = $1`, targetUserID).Scan(&profileID); err != nil {
		t.Fatalf("load profile id: %v", err)
	}
	if _, err := showruns.AddRosterMember(ctx, pool, actorUserID, showRunID, profileID, role, "", true); err != nil {
		t.Fatalf("add roster member role %q: %v", role, err)
	}
}

// ensureProfile is a minimal inline stand-in for playerprofile.EnsureWorkbook
// to avoid importing that package purely for a fixture helper.
func ensureProfile(ctx context.Context, pool *pgxpool.Pool, userID string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO player_profile_workbooks (user_id)
		VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING
	`, userID)
	return err
}
