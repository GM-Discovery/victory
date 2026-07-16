// Package shows_test (external, not internal package shows) so this file
// can import cues, which itself imports shows -- an internal shows test
// file importing cues would be a real import cycle (shows[test] -> cues ->
// shows), the same class of bug Kernel 70A hit and fixed in
// world/snapshot_test.go. See that fix's note for the general pattern.
package shows_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/cues"
	"victory/backend/internal/dbtest"
	"victory/backend/internal/scenes"
	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
	"victory/backend/internal/world"
)

func resumeTestSuffix(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000000")
}

func resumeTestUser(t *testing.T, pool *pgxpool.Pool, handlePrefix string) string {
	t.Helper()
	ctx := context.Background()
	handle := handlePrefix + "_" + resumeTestSuffix(t)

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
		_, _ = pool.Exec(context.Background(), `DELETE FROM show_runs WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.sessions WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})
	return userID
}

func resumeTestGrantRole(t *testing.T, pool *pgxpool.Pool, locationID, userID, role string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, $3::location_role, TRUE)
		ON CONFLICT (location_id, user_id, role) DO UPDATE SET active = TRUE
	`, locationID, userID, role); err != nil {
		t.Fatalf("grant location role %q: %v", role, err)
	}
}

// resumeFixture builds a location -> production -> show run -> show, with
// two Scene placements (A, B), the same shape cues_test.go's buildCueFixture
// builds -- duplicated here (small, deliberate) rather than exported from
// package cues purely for test reuse, matching this kernel's established
// "small duplicated SQL/fixture beats a new cross-package coupling" rule.
type resumeFixture struct {
	locationID   string
	showRunID    string
	showID       string
	placementAID string
	placementBID string
}

func buildResumeFixture(t *testing.T, pool *pgxpool.Pool, producerUserID string) resumeFixture {
	t.Helper()
	ctx := context.Background()
	suffix := resumeTestSuffix(t)

	var f resumeFixture
	if err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&f.locationID); err != nil {
		t.Fatalf("load amurray-family location: %v", err)
	}
	resumeTestGrantRole(t, pool, f.locationID, producerUserID, "producer")

	var productionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug) VALUES ($1, $2, $3) RETURNING id::text
	`, f.locationID, "Resume Test Production "+suffix, "resume-test-production-"+suffix).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id = $1`, productionID) })

	sr, err := showruns.CreateShowRun(ctx, pool, producerUserID, productionID, showruns.CreateShowRunInput{
		Title: "Resume Test Show Run " + suffix, Slug: "resume-test-show-run-" + suffix,
	})
	if err != nil {
		t.Fatalf("create show run fixture: %v", err)
	}
	f.showRunID = sr.ID

	s, err := shows.CreateShow(ctx, pool, producerUserID, sr.ID, shows.CreateShowInput{
		Title: "Resume Test Show " + suffix, Slug: "resume-test-show-" + suffix,
	})
	if err != nil {
		t.Fatalf("create show fixture: %v", err)
	}
	f.showID = s.ID

	sceneA, err := scenes.CreateScene(ctx, pool, producerUserID, f.locationID, productionID, scenes.CreateSceneInput{
		Title: "Resume Test Scene A", Slug: "resume-test-scene-a-" + suffix,
	})
	if err != nil {
		t.Fatalf("create scene A: %v", err)
	}
	pA, err := scenes.CreatePlacement(ctx, pool, producerUserID, f.showID, scenes.CreatePlacementInput{SceneID: sceneA.ID})
	if err != nil {
		t.Fatalf("create placement A: %v", err)
	}
	f.placementAID = pA.ID

	sceneB, err := scenes.CreateScene(ctx, pool, producerUserID, f.locationID, productionID, scenes.CreateSceneInput{
		Title: "Resume Test Scene B", Slug: "resume-test-scene-b-" + suffix,
	})
	if err != nil {
		t.Fatalf("create scene B: %v", err)
	}
	pB, err := scenes.CreatePlacement(ctx, pool, producerUserID, f.showID, scenes.CreatePlacementInput{SceneID: sceneB.ID})
	if err != nil {
		t.Fatalf("create placement B: %v", err)
	}
	f.placementBID = pB.ID

	return f
}

// endResumeTestSession is a direct, test-only stand-in for
// network.endSessionControl's core status transition (unexported, and not
// worth exporting purely for this test to reach across packages) -- closes
// the session the same way the real "End Session" action does, without
// exercising the Showing-close side effect, which this test doesn't assert
// on.
func endResumeTestSession(t *testing.T, pool *pgxpool.Pool, sessionID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		UPDATE sessions SET status = 'closed', ended_at = COALESCE(ended_at, NOW()) WHERE id = $1
	`, sessionID); err != nil {
		t.Fatalf("end session %s: %v", sessionID, err)
	}
}

// TestShowStagePersistsAcrossSequentialCatharsisSessions is the Kernel 70A
// §2.3 proof: a Show's current-Scene pointer is Show-owned, not
// Session-owned, so it must survive a full start -> end -> start -> GO ->
// end -> start cycle at Catharsis (the sole real functional venue for this
// kernel) with no "rebuild the stage" step required after a resume.
func TestShowStagePersistsAcrossSequentialCatharsisSessions(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	producer := resumeTestUser(t, pool, "resume_producer")
	f := buildResumeFixture(t, pool, producer)

	// Set Placement A current before any Session exists -- proves
	// go_to_scene doesn't require an active session either.
	c, err := cues.CreateCue(ctx, pool, producer, f.placementAID, cues.CreateCueInput{
		InternalName: "Go To Scene A",
		Actions: []cues.CueAction{
			{Type: cues.ActionTypeGoToScene, GoToScene: &cues.GoToSceneAction{ShowScenePlacementID: f.placementAID}},
		},
	})
	if err != nil {
		t.Fatalf("create go-to-A cue: %v", err)
	}
	if result, err := cues.ExecuteCue(ctx, pool, producer, c.ID, "resume-key-a-"+resumeTestSuffix(t)); err != nil || result.Status != "succeeded" {
		t.Fatalf("execute go-to-A cue (no session active yet): result=%+v err=%v", result, err)
	}

	// Session 1: start, confirm the live snapshot reflects Placement A.
	start1, err := shows.StartShowSession(ctx, pool, producer, f.showID, "catharsis", false)
	if err != nil {
		t.Fatalf("start session 1: %v", err)
	}
	if start1.VenueBusyWithOtherShow != nil {
		t.Fatalf("expected session 1 to start cleanly, got venue_busy: %+v", start1.VenueBusyWithOtherShow)
	}
	if start1.WasResumed {
		t.Fatalf("expected session 1 to be a fresh start, not a resume")
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, start1.SessionID) })

	snap1, err := world.LoadVenueSnapshot(ctx, pool, "producer", producer, "catharsis")
	if err != nil {
		t.Fatalf("load snapshot after session 1 start: %v", err)
	}
	if snap1.Session.CurrentShowScenePlacementID != f.placementAID {
		t.Fatalf("expected snapshot to show Placement A after session 1 start, got %q", snap1.Session.CurrentShowScenePlacementID)
	}

	// End session 1. With no active session, the Show's own persisted
	// state -- not a live snapshot -- is the only place left to check, and
	// must still show Placement A.
	endResumeTestSession(t, pool, start1.SessionID)
	showAfterEnd1, err := shows.LoadShowByID(ctx, pool, f.showID)
	if err != nil {
		t.Fatalf("load show after ending session 1: %v", err)
	}
	if showAfterEnd1.CurrentShowScenePlacementID == nil || *showAfterEnd1.CurrentShowScenePlacementID != f.placementAID {
		t.Fatalf("expected Show to still point at Placement A after session 1 ended, got %v", showAfterEnd1.CurrentShowScenePlacementID)
	}

	// Session 2: start again for the same Show. The snapshot must show
	// Placement A immediately -- no cue re-fired, no rebuild action.
	start2, err := shows.StartShowSession(ctx, pool, producer, f.showID, "catharsis", false)
	if err != nil {
		t.Fatalf("start session 2: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, start2.SessionID) })

	snap2, err := world.LoadVenueSnapshot(ctx, pool, "producer", producer, "catharsis")
	if err != nil {
		t.Fatalf("load snapshot after session 2 start: %v", err)
	}
	if snap2.Session.CurrentShowScenePlacementID != f.placementAID {
		t.Fatalf("expected snapshot to show Placement A on session 2 resume with no rebuild, got %q", snap2.Session.CurrentShowScenePlacementID)
	}

	// GO to Placement B during session 2.
	cB, err := cues.CreateCue(ctx, pool, producer, f.placementAID, cues.CreateCueInput{
		InternalName: "Go To Scene B",
		Actions: []cues.CueAction{
			{Type: cues.ActionTypeGoToScene, GoToScene: &cues.GoToSceneAction{ShowScenePlacementID: f.placementBID}},
		},
	})
	if err != nil {
		t.Fatalf("create go-to-B cue: %v", err)
	}
	if result, err := cues.ExecuteCue(ctx, pool, producer, cB.ID, "resume-key-b-"+resumeTestSuffix(t)); err != nil || result.Status != "succeeded" {
		t.Fatalf("execute go-to-B cue: result=%+v err=%v", result, err)
	}

	showAfterGoB, err := shows.LoadShowByID(ctx, pool, f.showID)
	if err != nil {
		t.Fatalf("load show after GO to B: %v", err)
	}
	if showAfterGoB.CurrentShowScenePlacementID == nil || *showAfterGoB.CurrentShowScenePlacementID != f.placementBID {
		t.Fatalf("expected Show to point at Placement B after GO, got %v", showAfterGoB.CurrentShowScenePlacementID)
	}

	// End session 2, start session 3: Placement B must persist into a
	// third, independent session the same way A persisted into the second.
	endResumeTestSession(t, pool, start2.SessionID)
	start3, err := shows.StartShowSession(ctx, pool, producer, f.showID, "catharsis", false)
	if err != nil {
		t.Fatalf("start session 3: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, start3.SessionID) })

	snap3, err := world.LoadVenueSnapshot(ctx, pool, "producer", producer, "catharsis")
	if err != nil {
		t.Fatalf("load snapshot after session 3 start: %v", err)
	}
	if snap3.Session.CurrentShowScenePlacementID != f.placementBID {
		t.Fatalf("expected snapshot to show Placement B in session 3, got %q", snap3.Session.CurrentShowScenePlacementID)
	}
}
