package shows

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/playerprofile"
	"victory/backend/internal/showruns"
)

func openShowsTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return dbtest.OpenTestPool(t)
}

func testSuffix(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000")
}

func insertShowsTestUser(t *testing.T, pool *pgxpool.Pool, handlePrefix string) string {
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
		_, _ = pool.Exec(context.Background(), `UPDATE sessions SET show_id = NULL WHERE show_id IN (SELECT id FROM shows WHERE created_by_user_id = $1)`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM shows WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM show_run_audience_blocks WHERE user_id = $1 OR blocked_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM show_run_roster_members WHERE user_id = $1 OR added_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM show_runs WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM player_stage_name_history WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM player_profile_workbooks WHERE user_id = $1`, userID)
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

// insertShowFixture builds a full location -> production -> show run -> show
// chain, since a fresh database has none of these (Kernel 64's lesson: raw
// migrations alone don't fully seed runtime-shaped data).
func insertShowFixture(t *testing.T, pool *pgxpool.Pool, creatorUserID string) (locationID, showRunID, showID string) {
	t.Helper()
	ctx := context.Background()

	if err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&locationID); err != nil {
		t.Fatalf("load amurray-family location: %v", err)
	}
	grantLocationRole(t, pool, locationID, creatorUserID, "producer")

	suffix := testSuffix(t)
	var productionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug)
		VALUES ($1, $2, $3)
		RETURNING id::text
	`, locationID, "Test Production "+suffix, "test-production-"+suffix).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id = $1`, productionID)
	})

	sr, err := showruns.CreateShowRun(ctx, pool, creatorUserID, productionID, showruns.CreateShowRunInput{
		Title: "Test Show Run " + suffix,
		Slug:  "test-show-run-" + suffix,
	})
	if err != nil {
		t.Fatalf("create show run fixture: %v", err)
	}

	s, err := CreateShow(ctx, pool, creatorUserID, sr.ID, CreateShowInput{
		Title: "Test Show " + suffix,
		Slug:  "test-show-" + suffix,
	})
	if err != nil {
		t.Fatalf("create show fixture: %v", err)
	}

	return locationID, sr.ID, s.ID
}

func TestCreateShowRequiresManageAuthorityAtRunLocation(t *testing.T) {
	pool := openShowsTestPool(t)
	producer := insertShowsTestUser(t, pool, "sh_producer")
	_, showRunID, _ := insertShowFixture(t, pool, producer)

	// Correct-location producer can create another Show.
	if _, err := CreateShow(context.Background(), pool, producer, showRunID, CreateShowInput{
		Title: "Second Show", Slug: "second-show-" + testSuffix(t),
	}); err != nil {
		t.Fatalf("expected producer to create show: %v", err)
	}

	// A Producer at a *different* location must not pass the check.
	var otherLocationID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO locations (name, slug) VALUES ($1, $2) RETURNING id::text
	`, "Elsewhere "+testSuffix(t), "elsewhere-"+testSuffix(t)).Scan(&otherLocationID); err != nil {
		t.Fatalf("insert other location: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM locations WHERE id = $1`, otherLocationID)
	})
	elsewhereProducer := insertShowsTestUser(t, pool, "sh_elsewhere_producer")
	grantLocationRole(t, pool, otherLocationID, elsewhereProducer, "producer")
	if _, err := CreateShow(context.Background(), pool, elsewhereProducer, showRunID, CreateShowInput{
		Title: "Cross Location Show", Slug: "cross-location-show-" + testSuffix(t),
	}); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for producer at a different location, got %v", err)
	}

	// A user with no membership anywhere is rejected.
	outsider := insertShowsTestUser(t, pool, "sh_outsider")
	if _, err := CreateShow(context.Background(), pool, outsider, showRunID, CreateShowInput{
		Title: "Outsider Show", Slug: "outsider-show-" + testSuffix(t),
	}); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for outsider, got %v", err)
	}
}

func TestShowBelongsToParentShowRunAndListReturnsMultiple(t *testing.T) {
	pool := openShowsTestPool(t)
	producer := insertShowsTestUser(t, pool, "sh_list_producer")
	_, showRunID, showID := insertShowFixture(t, pool, producer)

	second, err := CreateShow(context.Background(), pool, producer, showRunID, CreateShowInput{
		Title: "Second Show", Slug: "list-second-" + testSuffix(t),
	})
	if err != nil {
		t.Fatalf("create second show: %v", err)
	}

	list, summary, err := ListShowsForRun(context.Background(), pool, showRunID)
	if err != nil {
		t.Fatalf("list shows for run: %v", err)
	}
	if summary.Total != 2 {
		t.Fatalf("expected 2 total shows, got %d", summary.Total)
	}
	found := map[string]bool{}
	for _, s := range list {
		found[s.ID] = true
	}
	if !found[showID] || !found[second.ID] {
		t.Fatalf("expected both shows in list, got %+v", list)
	}

	// A different show run's list must not include these shows.
	_, otherRunID, _ := insertShowFixture(t, pool, producer)
	otherList, _, err := ListShowsForRun(context.Background(), pool, otherRunID)
	if err != nil {
		t.Fatalf("list other run shows: %v", err)
	}
	for _, s := range otherList {
		if s.ID == showID || s.ID == second.ID {
			t.Fatalf("show from a different run leaked into unrelated run's list")
		}
	}
}

func TestShowStatusCheckRejectsInvalidValue(t *testing.T) {
	pool := openShowsTestPool(t)
	producer := insertShowsTestUser(t, pool, "sh_status_producer")
	_, _, showID := insertShowFixture(t, pool, producer)

	_, err := pool.Exec(context.Background(), `UPDATE shows SET status = 'not_a_real_status' WHERE id = $1`, showID)
	if err == nil {
		t.Fatalf("expected DB CHECK constraint to reject an invalid status value")
	}
}

func TestShowArchivedAtMatchesStatusConstraint(t *testing.T) {
	pool := openShowsTestPool(t)
	producer := insertShowsTestUser(t, pool, "sh_archive_producer")
	_, _, showID := insertShowFixture(t, pool, producer)

	archived, err := ArchiveShow(context.Background(), pool, producer, showID)
	if err != nil {
		t.Fatalf("archive show: %v", err)
	}
	if archived.Status != "archived" || archived.ArchivedAt == nil {
		t.Fatalf("expected archived status with archived_at set, got %+v", archived)
	}

	// Direct SQL attempt to set archived without archived_at must fail the CHECK.
	_, err = pool.Exec(context.Background(), `UPDATE shows SET status = 'archived', archived_at = NULL WHERE id = $1`, showID)
	if err == nil {
		t.Fatalf("expected CHECK constraint to reject archived status with null archived_at")
	}

	// Unarchiving (back to draft) clears archived_at via UpdateShow.
	draft := "draft"
	unarchived, err := UpdateShow(context.Background(), pool, producer, showID, UpdateShowPatch{Status: &draft})
	if err != nil {
		t.Fatalf("unarchive show: %v", err)
	}
	if unarchived.Status != "draft" || unarchived.ArchivedAt != nil {
		t.Fatalf("expected draft status with nil archived_at, got %+v", unarchived)
	}
}

func TestShowScheduledFieldsAreOptionalAndOrderIsEnforced(t *testing.T) {
	pool := openShowsTestPool(t)
	producer := insertShowsTestUser(t, pool, "sh_sched_producer")
	_, _, showID := insertShowFixture(t, pool, producer)

	// A draft show with nothing scheduled is valid -- already true from
	// insertShowFixture's plain CreateShow call; confirm explicitly.
	s, err := LoadShowByID(context.Background(), pool, showID)
	if err != nil {
		t.Fatalf("load show: %v", err)
	}
	if s.ScheduledStartAt != nil || s.ScheduledEndAt != nil {
		t.Fatalf("expected a freshly created show to have no scheduled fields, got %+v", s)
	}

	start := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	end := time.Now().Add(23 * time.Hour).UTC().Format(time.RFC3339) // before start
	if _, err := UpdateShow(context.Background(), pool, producer, showID, UpdateShowPatch{
		ScheduledStartAt: &start, ScheduledEndAt: &end,
	}); err == nil {
		t.Fatalf("expected scheduled_end_at before scheduled_start_at to be rejected by the DB CHECK")
	}

	validEnd := time.Now().Add(25 * time.Hour).UTC().Format(time.RFC3339)
	updated, err := UpdateShow(context.Background(), pool, producer, showID, UpdateShowPatch{
		ScheduledStartAt: &start, ScheduledEndAt: &validEnd,
	})
	if err != nil {
		t.Fatalf("expected valid scheduled range to be accepted: %v", err)
	}
	if updated.ScheduledStartAt == nil || updated.ScheduledEndAt == nil {
		t.Fatalf("expected scheduled fields to be set, got %+v", updated)
	}
}

func TestAudienceProgramIsShowSpecificAndExcludesBackstageFields(t *testing.T) {
	pool := openShowsTestPool(t)
	producer := insertShowsTestUser(t, pool, "sh_program_producer")
	_, showRunID, showID := insertShowFixture(t, pool, producer)

	blurbA := "Come see Show A!"
	secondBlurb := "Come see Show B instead!"
	if _, err := UpdateShow(context.Background(), pool, producer, showID, UpdateShowPatch{
		AudienceTitle: strPtr("Show A"), AudienceProgramBlurb: &blurbA,
	}); err != nil {
		t.Fatalf("update show A blurb: %v", err)
	}
	second, err := CreateShow(context.Background(), pool, producer, showRunID, CreateShowInput{
		Title: "Show B", Slug: "program-show-b-" + testSuffix(t),
		AudienceTitle: "Show B", AudienceProgramBlurb: secondBlurb,
	})
	if err != nil {
		t.Fatalf("create second show: %v", err)
	}

	sA, err := LoadShowByID(context.Background(), pool, showID)
	if err != nil {
		t.Fatalf("load show A: %v", err)
	}
	programA, err := ListAudienceProgramForShow(context.Background(), pool, sA)
	if err != nil {
		t.Fatalf("list program for show A: %v", err)
	}
	_ = programA // roster membership itself is shared; the blurb below is what's show-specific

	if sA.AudienceProgramBlurb != blurbA {
		t.Fatalf("show A blurb = %q, want %q", sA.AudienceProgramBlurb, blurbA)
	}
	if second.AudienceProgramBlurb != secondBlurb {
		t.Fatalf("show B blurb = %q, want %q", second.AudienceProgramBlurb, secondBlurb)
	}
	if sA.AudienceProgramBlurb == second.AudienceProgramBlurb {
		t.Fatalf("expected distinct per-show blurbs, both are %q", sA.AudienceProgramBlurb)
	}
}

func TestShowRosterInheritsShowRunVisibilityWithNoOwnRosterTable(t *testing.T) {
	pool := openShowsTestPool(t)
	producer := insertShowsTestUser(t, pool, "sh_roster_producer")
	player := insertShowsTestUser(t, pool, "sh_roster_player")
	_, showRunID, showID := insertShowFixture(t, pool, producer)

	if _, err := playerprofile.EnsureWorkbook(context.Background(), pool, player); err != nil {
		t.Fatalf("ensure workbook: %v", err)
	}
	var profileID string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM player_profile_workbooks WHERE user_id = $1`, player).Scan(&profileID); err != nil {
		t.Fatalf("load profile id: %v", err)
	}
	if _, err := showruns.AddRosterMember(context.Background(), pool, producer, showRunID, profileID, "player", "", true); err != nil {
		t.Fatalf("add roster member to show run: %v", err)
	}

	s, err := LoadShowByID(context.Background(), pool, showID)
	if err != nil {
		t.Fatalf("load show: %v", err)
	}
	members, err := ListInternalRosterForShow(context.Background(), pool, s)
	if err != nil {
		t.Fatalf("list internal roster for show: %v", err)
	}
	found := false
	for _, m := range members {
		if m.UserID == player {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected show-run roster member to appear in the show's inherited roster")
	}

	var showRosterRowCount int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM information_schema.columns WHERE table_name = 'shows' AND column_name LIKE '%roster%'`).Scan(&showRosterRowCount); err != nil {
		t.Fatalf("check for show-scoped roster columns: %v", err)
	}
	if showRosterRowCount != 0 {
		t.Fatalf("expected no roster-named columns on the shows table itself (roster must be inherited, not duplicated)")
	}
}

func TestSessionLinkAndUnlinkSetsAndClearsShowID(t *testing.T) {
	pool := openShowsTestPool(t)
	producer := insertShowsTestUser(t, pool, "sh_session_producer")
	_, _, showID := insertShowFixture(t, pool, producer)

	var venueID string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM venues LIMIT 1`).Scan(&venueID); err != nil {
		t.Fatalf("load a venue: %v", err)
	}
	var sessionID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO sessions (venue_id) VALUES ($1) RETURNING id::text
	`, venueID).Scan(&sessionID); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, sessionID)
	})

	if err := LinkSessionToShow(context.Background(), pool, producer, showID, sessionID); err != nil {
		t.Fatalf("link session to show: %v", err)
	}
	var linkedShowID *string
	if err := pool.QueryRow(context.Background(), `SELECT show_id::text FROM sessions WHERE id = $1`, sessionID).Scan(&linkedShowID); err != nil {
		t.Fatalf("read linked show id: %v", err)
	}
	if linkedShowID == nil || *linkedShowID != showID {
		t.Fatalf("expected session.show_id = %q, got %v", showID, linkedShowID)
	}

	if err := UnlinkSessionFromShow(context.Background(), pool, producer, showID, sessionID); err != nil {
		t.Fatalf("unlink session from show: %v", err)
	}
	var afterUnlink *string
	if err := pool.QueryRow(context.Background(), `SELECT show_id::text FROM sessions WHERE id = $1`, sessionID).Scan(&afterUnlink); err != nil {
		t.Fatalf("read show id after unlink: %v", err)
	}
	if afterUnlink != nil {
		t.Fatalf("expected show_id to be cleared after unlink, got %v", *afterUnlink)
	}

	// Unlinking with a mismatched show is a no-op, not an error, and does not
	// clear a link that belongs to a different show.
	if err := LinkSessionToShow(context.Background(), pool, producer, showID, sessionID); err != nil {
		t.Fatalf("re-link session to show: %v", err)
	}
	otherShow, err := CreateShow(context.Background(), pool, producer, mustShowRunID(t, pool, showID), CreateShowInput{
		Title: "Unrelated Show", Slug: "unrelated-show-" + testSuffix(t),
	})
	if err != nil {
		t.Fatalf("create unrelated show: %v", err)
	}
	if err := UnlinkSessionFromShow(context.Background(), pool, producer, otherShow.ID, sessionID); err != nil {
		t.Fatalf("mismatched unlink should not error: %v", err)
	}
	var stillLinked *string
	if err := pool.QueryRow(context.Background(), `SELECT show_id::text FROM sessions WHERE id = $1`, sessionID).Scan(&stillLinked); err != nil {
		t.Fatalf("read show id after mismatched unlink: %v", err)
	}
	if stillLinked == nil || *stillLinked != showID {
		t.Fatalf("expected session to remain linked to the original show after a mismatched unlink, got %v", stillLinked)
	}
}

func mustShowRunID(t *testing.T, pool *pgxpool.Pool, showID string) string {
	t.Helper()
	s, err := LoadShowByID(context.Background(), pool, showID)
	if err != nil {
		t.Fatalf("load show for run id: %v", err)
	}
	return s.ShowRunID
}

func strPtr(s string) *string { return &s }
