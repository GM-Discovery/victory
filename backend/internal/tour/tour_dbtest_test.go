package tour

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/venues"
)

func openTourTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return dbtest.OpenTestPool(t)
}

func tourTestSuffix(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000")
}

func insertTourTestUser(t *testing.T, pool *pgxpool.Pool, handlePrefix string) string {
	t.Helper()
	ctx := context.Background()
	handle := handlePrefix + "_" + tourTestSuffix(t)

	var userID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name)
		VALUES ($1, $2)
		RETURNING id::text
	`, handle, handlePrefix).Scan(&userID); err != nil {
		t.Fatalf("insert user %q: %v", handle, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM tour_completions WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})
	return userID
}

func grantTourLocationRole(t *testing.T, pool *pgxpool.Pool, locationID, userID, role string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, $3::location_role, TRUE)
		ON CONFLICT (location_id, user_id, role) DO UPDATE SET active = TRUE
	`, locationID, userID, role); err != nil {
		t.Fatalf("grant location role %q: %v", role, err)
	}
}

func loadCatharsisLocationID(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	_, locationID, err := venues.ResolveVenueLocation(context.Background(), pool, "catharsis")
	if err != nil {
		t.Fatalf("resolve catharsis location: %v", err)
	}
	return locationID
}

// grantCatharsisVenueAccess satisfies access.ResolveVisibleVenues'
// approved_performer_surface branch (location_memberships role AND a live
// access_grants row -- both, not either), the same condition
// KeyGreenroomIntro's own map-visibility gate re-derives. A bare
// location_memberships row alone (grantTourLocationRole) is not enough for
// Catharsis specifically.
func grantCatharsisVenueAccess(t *testing.T, pool *pgxpool.Pool, locationID, userID string) {
	t.Helper()
	ctx := context.Background()
	grantTourLocationRole(t, pool, locationID, userID, "cast")
	var venueID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM venues WHERE slug = 'catharsis'`).Scan(&venueID); err != nil {
		t.Fatalf("resolve catharsis venue id: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO access_grants (location_id, user_id, grant_type, venue_id)
		VALUES ($1, $2, 'venue_access', $3)
	`, locationID, userID, venueID); err != nil {
		t.Fatalf("insert access_grants: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM access_grants WHERE user_id = $1`, userID)
	})
}

func TestRecordCompletionIdempotent(t *testing.T) {
	pool := openTourTestPool(t)
	userID := insertTourTestUser(t, pool, "tour_idem")
	subj := Subject{UserID: userID}

	for i := 0; i < 2; i++ {
		if err := RecordCompletion(context.Background(), pool, subj, KeyCampusMandatory, "", "", StatusCompleted, ""); err != nil {
			t.Fatalf("RecordCompletion call %d: %v", i, err)
		}
	}

	var count int
	if err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM tour_completions WHERE user_id = $1 AND tour_key = $2
	`, userID, KeyCampusMandatory).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 row after 2 idempotent calls, got %d", count)
	}
}

func TestHasCompletionScopedPerUserAndVenue(t *testing.T) {
	pool := openTourTestPool(t)
	userA := insertTourTestUser(t, pool, "tour_scope_a")
	userB := insertTourTestUser(t, pool, "tour_scope_b")

	if err := RecordCompletion(context.Background(), pool, Subject{UserID: userA}, KeyCatharsisCast, "catharsis", "cast", StatusCompleted, ""); err != nil {
		t.Fatalf("RecordCompletion: %v", err)
	}

	hasA, err := HasCompletion(context.Background(), pool, Subject{UserID: userA}, KeyCatharsisCast, "catharsis", "cast")
	if err != nil || !hasA {
		t.Fatalf("expected userA to have completion, got hasA=%v err=%v", hasA, err)
	}

	hasB, err := HasCompletion(context.Background(), pool, Subject{UserID: userB}, KeyCatharsisCast, "catharsis", "cast")
	if err != nil || hasB {
		t.Fatalf("expected userB (different user) to have no completion, got hasB=%v err=%v", hasB, err)
	}

	hasWrongVenue, err := HasCompletion(context.Background(), pool, Subject{UserID: userA}, KeyCatharsisCast, "greenroom", "cast")
	if err != nil || hasWrongVenue {
		t.Fatalf("expected no completion for a different venue scope, got %v err=%v", hasWrongVenue, err)
	}
}

func TestEligibleToursRoleGating(t *testing.T) {
	pool := openTourTestPool(t)
	locationID := loadCatharsisLocationID(t, pool)

	audienceUser := insertTourTestUser(t, pool, "tour_audience")
	castUser := insertTourTestUser(t, pool, "tour_cast")
	grantTourLocationRole(t, pool, locationID, castUser, "cast")

	audienceEligible, err := EligibleTours(context.Background(), pool, Subject{UserID: audienceUser}, "catharsis")
	if err != nil {
		t.Fatalf("EligibleTours (audience): %v", err)
	}
	for _, def := range audienceEligible {
		if def.Key == KeyCatharsisCast {
			t.Fatalf("audience-role user must not be eligible for %s", KeyCatharsisCast)
		}
	}

	castEligible, err := EligibleTours(context.Background(), pool, Subject{UserID: castUser}, "catharsis")
	if err != nil {
		t.Fatalf("EligibleTours (cast): %v", err)
	}
	found := false
	for _, def := range castEligible {
		if def.Key == KeyCatharsisCast {
			found = true
		}
	}
	if !found {
		t.Fatalf("cast-role user must be eligible for %s, got %+v", KeyCatharsisCast, castEligible)
	}
}

func TestEligibleToursOperatorBypassesLocationRole(t *testing.T) {
	pool := openTourTestPool(t)
	operatorUserID := insertTourTestUser(t, pool, "tour_operator")

	t.Setenv("OPERATOR_USER_ID", operatorUserID)
	t.Setenv("OPERATOR_HANDLE", "")

	eligible, err := EligibleTours(context.Background(), pool, Subject{UserID: operatorUserID}, "directors-chair")
	if err != nil {
		t.Fatalf("EligibleTours (operator): %v", err)
	}
	found := false
	for _, def := range eligible {
		if def.Key == KeyDirectorsChairDirectorToolbox {
			found = true
		}
	}
	if !found {
		t.Fatalf("operator must be eligible for %s without any location_role, got %+v", KeyDirectorsChairDirectorToolbox, eligible)
	}
}

func TestMandatoryTourCannotBeRecordedAsSkipped(t *testing.T) {
	pool := openTourTestPool(t)
	userID := insertTourTestUser(t, pool, "tour_mandatory_skip")

	// RecordCompletion itself does not know about Mandatory (that check
	// lives in http.go's handleWrite, ahead of the call) -- this test
	// documents that boundary lives at the HTTP layer, not the data layer,
	// by confirming the data layer will happily write either status; the
	// skip-rejection behavior itself is covered by the HTTP-level test.
	if err := RecordCompletion(context.Background(), pool, Subject{UserID: userID}, KeyCampusMandatory, "", "", StatusSkipped, ""); err != nil {
		t.Fatalf("RecordCompletion: %v", err)
	}
	has, err := HasCompletion(context.Background(), pool, Subject{UserID: userID}, KeyCampusMandatory, "", "")
	if err != nil || !has {
		t.Fatalf("expected row to exist, got has=%v err=%v", has, err)
	}
}

func TestResumeMandatorySkipsPastRecordedProgress(t *testing.T) {
	pool := openTourTestPool(t)
	userID := insertTourTestUser(t, pool, "tour_resume")
	subj := Subject{UserID: userID}

	fresh, err := ResumeMandatory(context.Background(), pool, subj)
	if err != nil {
		t.Fatalf("ResumeMandatory (fresh): %v", err)
	}
	if fresh == nil || len(fresh.Steps) != 2 {
		t.Fatalf("expected the full 2-step mandatory tour before any progress, got %+v", fresh)
	}
	firstStepKey := fresh.Steps[0].Key

	// Simulate the real production bug: the user clicked the first
	// click-gated step (Audition Hall), which navigated away before an
	// ordinary fetch could land -- only the sendBeacon-delivered progress
	// call survives.
	if err := RecordProgress(context.Background(), pool, subj, KeyCampusMandatory, "", "", firstStepKey); err != nil {
		t.Fatalf("RecordProgress: %v", err)
	}

	resumed, err := ResumeMandatory(context.Background(), pool, subj)
	if err != nil {
		t.Fatalf("ResumeMandatory (after progress): %v", err)
	}
	if resumed == nil {
		t.Fatalf("expected the mandatory tour to still be pending (not completed) after progress alone")
	}
	if len(resumed.Steps) != 1 {
		t.Fatalf("expected resume to skip the already-reached step and return exactly 1 remaining step, got %+v", resumed.Steps)
	}
	if resumed.Steps[0].Key == firstStepKey {
		t.Fatalf("resume must not re-offer the already-reached step %q", firstStepKey)
	}

	// Completing the tour must clear the now-stale progress cursor.
	if err := RecordCompletion(context.Background(), pool, subj, KeyCampusMandatory, "", "", StatusCompleted, resumed.Steps[0].Key); err != nil {
		t.Fatalf("RecordCompletion: %v", err)
	}
	var progressRows int
	if err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM tour_progress WHERE user_id = $1 AND tour_key = $2
	`, userID, KeyCampusMandatory).Scan(&progressRows); err != nil {
		t.Fatalf("count progress rows: %v", err)
	}
	if progressRows != 0 {
		t.Fatalf("expected RecordCompletion to clear the progress cursor, found %d row(s)", progressRows)
	}
}

func TestCampusContinuationRequiresMandatoryFirst(t *testing.T) {
	pool := openTourTestPool(t)
	userID := insertTourTestUser(t, pool, "tour_continuation")
	subj := Subject{UserID: userID}

	before, err := EligibleTours(context.Background(), pool, subj, "")
	if err != nil {
		t.Fatalf("EligibleTours (before mandatory): %v", err)
	}
	for _, def := range before {
		if def.Key == KeyCampusContinuation {
			t.Fatalf("campus_continuation must not be eligible before campus_mandatory is completed")
		}
	}

	if err := RecordCompletion(context.Background(), pool, subj, KeyCampusMandatory, "", "", StatusCompleted, ""); err != nil {
		t.Fatalf("RecordCompletion: %v", err)
	}

	// Kernel 93 Pass C: completing campus_mandatory alone is no longer
	// sufficient -- campus_continuation's Catharsis pin also requires real
	// Catharsis access (the same check the map itself uses to decide tile
	// visibility), so a user with no admission is correctly still not
	// offered it.
	stillGated, err := EligibleTours(context.Background(), pool, subj, "")
	if err != nil {
		t.Fatalf("EligibleTours (mandatory done, no catharsis access): %v", err)
	}
	for _, def := range stillGated {
		if def.Key == KeyCampusContinuation {
			t.Fatalf("campus_continuation must not be eligible before the user has real Catharsis access")
		}
	}

	locationID := loadCatharsisLocationID(t, pool)
	grantCatharsisVenueAccess(t, pool, locationID, userID)

	after, err := EligibleTours(context.Background(), pool, subj, "")
	if err != nil {
		t.Fatalf("EligibleTours (after mandatory + catharsis access): %v", err)
	}
	found := false
	for _, def := range after {
		if def.Key == KeyCampusContinuation {
			found = true
		}
	}
	if !found {
		t.Fatalf("campus_continuation must become eligible once campus_mandatory is completed and Catharsis is accessible, got %+v", after)
	}
}

func TestGreenroomIntroRequiresStartedCharacter(t *testing.T) {
	pool := openTourTestPool(t)
	userID := insertTourTestUser(t, pool, "tour_greenroom")
	subj := Subject{UserID: userID}

	if err := RecordCompletion(context.Background(), pool, subj, KeyCampusMandatory, "", "", StatusCompleted, ""); err != nil {
		t.Fatalf("RecordCompletion: %v", err)
	}

	before, err := EligibleTours(context.Background(), pool, subj, "")
	if err != nil {
		t.Fatalf("EligibleTours (before character): %v", err)
	}
	for _, def := range before {
		if def.Key == KeyGreenroomIntro {
			t.Fatalf("greenroom_intro must not be eligible before any character exists")
		}
	}

	locationID := loadCatharsisLocationID(t, pool)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO character_cards (owner_user_id, location_id, name)
		VALUES ($1, $2, 'Tour Test Character')
	`, userID, locationID); err != nil {
		t.Fatalf("insert character_cards: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM character_cards WHERE owner_user_id = $1`, userID)
	})

	after, err := EligibleTours(context.Background(), pool, subj, "")
	if err != nil {
		t.Fatalf("EligibleTours (after character): %v", err)
	}
	found := false
	for _, def := range after {
		if def.Key == KeyGreenroomIntro {
			found = true
		}
	}
	if !found {
		t.Fatalf("greenroom_intro must become eligible once a character exists, got %+v", after)
	}
}
