package showruns

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/playerprofile"
)

func openShowRunsTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return dbtest.OpenTestPool(t)
}

func testSuffix(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000")
}

func insertShowRunTestUser(t *testing.T, pool *pgxpool.Pool, handlePrefix string) string {
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

// insertShowRunFixture creates a real location->production->show_run chain
// for tests, rather than assuming any specific production already exists on
// a fresh database (Kernel 64's lesson: raw migrations alone don't fully
// seed a fresh DB with runtime-shaped data).
func insertShowRunFixture(t *testing.T, pool *pgxpool.Pool, creatorUserID string) (locationID, productionID, showRunID string) {
	t.Helper()
	ctx := context.Background()

	if err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&locationID); err != nil {
		t.Fatalf("load amurray-family location: %v", err)
	}

	suffix := testSuffix(t)
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

	// The fixture's creator needs producer authority at this location to
	// pass CreateShowRun's own authority check; tests that care about a
	// *different* actor's authority grant/revoke roles separately after
	// this fixture exists.
	grantLocationRole(t, pool, locationID, creatorUserID, "producer")

	sr, err := CreateShowRun(ctx, pool, creatorUserID, productionID, CreateShowRunInput{
		Title: "Test Show Run " + suffix,
		Slug:  "test-show-run-" + suffix,
	})
	if err != nil {
		t.Fatalf("create show run fixture: %v", err)
	}
	return locationID, productionID, sr.ID
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

func setStageName(t *testing.T, pool *pgxpool.Pool, userID, stageName string) {
	t.Helper()
	if _, err := playerprofile.ChangeStageName(context.Background(), pool, userID, userID, stageName); err != nil {
		t.Fatalf("change stage name: %v", err)
	}
}

func withOperator(t *testing.T, userID string) {
	t.Helper()
	prev := os.Getenv("OPERATOR_USER_ID")
	_ = os.Setenv("OPERATOR_USER_ID", userID)
	t.Cleanup(func() { _ = os.Setenv("OPERATOR_USER_ID", prev) })
}

func TestCreateShowRunRequiresProducerOrDirectorAtCorrectLocation(t *testing.T) {
	pool := openShowRunsTestPool(t)
	producer := insertShowRunTestUser(t, pool, "sr_producer")
	outsider := insertShowRunTestUser(t, pool, "sr_outsider")

	locationID, productionID, _ := insertShowRunFixture(t, pool, producer)
	grantLocationRole(t, pool, locationID, producer, "producer")

	// Producer at the correct location succeeds.
	if _, err := CreateShowRun(context.Background(), pool, producer, productionID, CreateShowRunInput{
		Title: "Second Run", Slug: "second-run-" + testSuffix(t),
	}); err != nil {
		t.Fatalf("expected producer to create show run: %v", err)
	}

	// A user with no membership at this location at all is rejected.
	if _, err := CreateShowRun(context.Background(), pool, outsider, productionID, CreateShowRunInput{
		Title: "Outsider Run", Slug: "outsider-run-" + testSuffix(t),
	}); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for outsider, got %v", err)
	}

	// A Producer at a *different* location must not pass the check here --
	// this is exactly the bug CurrentLocationRoleForLocation exists to
	// prevent (CurrentLocationRole alone would have let this through).
	// insertShowRunFixture always resolves to the single seeded
	// "amurray-family" location, so a genuinely different location is
	// inserted directly here.
	var otherLocationID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO locations (name, slug) VALUES ($1, $2) RETURNING id::text
	`, "Elsewhere "+testSuffix(t), "elsewhere-"+testSuffix(t)).Scan(&otherLocationID); err != nil {
		t.Fatalf("insert other location: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM locations WHERE id = $1`, otherLocationID)
	})
	elsewhereProducer := insertShowRunTestUser(t, pool, "sr_elsewhere_producer")
	grantLocationRole(t, pool, otherLocationID, elsewhereProducer, "producer")
	if _, err := CreateShowRun(context.Background(), pool, elsewhereProducer, productionID, CreateShowRunInput{
		Title: "Cross Location Run", Slug: "cross-location-run-" + testSuffix(t),
	}); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for producer at a different location, got %v", err)
	}

	// Operator bypasses entirely.
	withOperator(t, outsider)
	if _, err := CreateShowRun(context.Background(), pool, outsider, productionID, CreateShowRunInput{
		Title: "Operator Run", Slug: "operator-run-" + testSuffix(t),
	}); err != nil {
		t.Fatalf("expected operator to bypass authority check: %v", err)
	}
}

func TestAddRosterMemberEnforcesOneActiveRowPerUser(t *testing.T) {
	pool := openShowRunsTestPool(t)
	producer := insertShowRunTestUser(t, pool, "sr_roster_producer")
	member := insertShowRunTestUser(t, pool, "sr_roster_member")
	locationID, _, showRunID := insertShowRunFixture(t, pool, producer)
	grantLocationRole(t, pool, locationID, producer, "producer")

	var memberProfileID string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM player_profile_workbooks WHERE user_id = $1`, member).Scan(&memberProfileID); err != nil {
		if _, err := playerprofile.EnsureWorkbook(context.Background(), pool, member); err != nil {
			t.Fatalf("ensure workbook: %v", err)
		}
		if err := pool.QueryRow(context.Background(), `SELECT id::text FROM player_profile_workbooks WHERE user_id = $1`, member).Scan(&memberProfileID); err != nil {
			t.Fatalf("load member profile id: %v", err)
		}
	}

	first, err := AddRosterMember(context.Background(), pool, producer, showRunID, memberProfileID, "player", "", true)
	if err != nil {
		t.Fatalf("add roster member: %v", err)
	}
	if first.Role != "player" {
		t.Fatalf("expected role player, got %q", first.Role)
	}

	// Adding again updates the role of the same row rather than creating a
	// second one (DB-level enforcement, not just an application check).
	second, err := AddRosterMember(context.Background(), pool, producer, showRunID, memberProfileID, "crew", "", true)
	if err != nil {
		t.Fatalf("re-add roster member: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected same roster row id on re-add, got %q vs %q", first.ID, second.ID)
	}
	if second.Role != "crew" {
		t.Fatalf("expected role updated to crew, got %q", second.Role)
	}

	roster, err := ListInternalRoster(context.Background(), pool, showRunID)
	if err != nil {
		t.Fatalf("list internal roster: %v", err)
	}
	count := 0
	for _, m := range roster {
		if m.UserID == member {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 active roster row for member, got %d", count)
	}

	// Role label must say "Player", never "Cast", for role="player".
	updated, err := ProjectRosterMember(context.Background(), pool, producer, second)
	if err != nil {
		t.Fatalf("project updated roster member: %v", err)
	}
	if updated.RoleLabel != "Crew" {
		t.Fatalf("expected role label Crew, got %q", updated.RoleLabel)
	}
	if roleDisplayLabel("player", "") != "Player" || roleDisplayLabel("player", "") == "Cast" {
		t.Fatalf("expected role_display_label(player) to be Player, never Cast")
	}
}

func TestSelfJoinAsAudienceRespectsFlagAndBlocks(t *testing.T) {
	pool := openShowRunsTestPool(t)
	producer := insertShowRunTestUser(t, pool, "sr_selfjoin_producer")
	audience := insertShowRunTestUser(t, pool, "sr_selfjoin_audience")
	locationID, _, showRunID := insertShowRunFixture(t, pool, producer)
	grantLocationRole(t, pool, locationID, producer, "producer")
	grantLocationRole(t, pool, locationID, audience, "audience")

	// Disabled by default -- self-join rejected.
	if _, err := SelfJoinAsAudience(context.Background(), pool, audience, showRunID); err == nil || err.Error() != "self_join_disabled" {
		t.Fatalf("expected self_join_disabled, got %v", err)
	}

	enabled := true
	if _, err := UpdateShowRun(context.Background(), pool, producer, showRunID, UpdateShowRunPatch{AudienceSelfJoinEnabled: &enabled}); err != nil {
		t.Fatalf("enable self join: %v", err)
	}

	member, err := SelfJoinAsAudience(context.Background(), pool, audience, showRunID)
	if err != nil {
		t.Fatalf("self join: %v", err)
	}
	if member.Role != "audience" {
		t.Fatalf("expected role audience, got %q", member.Role)
	}

	// Idempotent: repeated self-join refreshes rather than erroring.
	if _, err := SelfJoinAsAudience(context.Background(), pool, audience, showRunID); err != nil {
		t.Fatalf("repeated self join should be idempotent: %v", err)
	}

	// Blocked user cannot self-join.
	blockedUser := insertShowRunTestUser(t, pool, "sr_blocked_audience")
	grantLocationRole(t, pool, locationID, blockedUser, "audience")
	var blockedProfileID string
	if _, err := playerprofile.EnsureWorkbook(context.Background(), pool, blockedUser); err != nil {
		t.Fatalf("ensure workbook: %v", err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM player_profile_workbooks WHERE user_id = $1`, blockedUser).Scan(&blockedProfileID); err != nil {
		t.Fatalf("load blocked user profile id: %v", err)
	}
	if _, err := BlockUser(context.Background(), pool, producer, showRunID, blockedUser, "test block"); err != nil {
		t.Fatalf("block user: %v", err)
	}
	if _, err := SelfJoinAsAudience(context.Background(), pool, blockedUser, showRunID); err == nil || err.Error() != "user_blocked" {
		t.Fatalf("expected user_blocked, got %v", err)
	}

	// Non-operator adding a blocked user as Audience is also rejected.
	if _, err := AddRosterMember(context.Background(), pool, producer, showRunID, blockedProfileID, "audience", "", true); err == nil || err.Error() != "user_blocked" {
		t.Fatalf("expected user_blocked when adding blocked user as audience, got %v", err)
	}

	// Operator override: adding a blocked user as Audience is allowed.
	withOperator(t, producer)
	if _, err := AddRosterMember(context.Background(), pool, producer, showRunID, blockedProfileID, "audience", "", true); err != nil {
		t.Fatalf("expected operator override to add blocked user as audience: %v", err)
	}
	_ = os.Setenv("OPERATOR_USER_ID", "")

	// Unblock restores self-join eligibility.
	var blockID string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM show_run_audience_blocks WHERE show_run_id = $1 AND user_id = $2 AND lifted_at IS NULL`, showRunID, blockedUser).Scan(&blockID); err != nil {
		t.Fatalf("load active block: %v", err)
	}
	if err := UnblockUser(context.Background(), pool, producer, showRunID, blockID); err != nil {
		t.Fatalf("unblock user: %v", err)
	}
	if _, err := SelfJoinAsAudience(context.Background(), pool, blockedUser, showRunID); err != nil {
		t.Fatalf("expected self join to succeed after unblock: %v", err)
	}
}

func TestListAudienceProgramMembersOrdersAudienceFirstAndRespectsVisibility(t *testing.T) {
	pool := openShowRunsTestPool(t)
	producer := insertShowRunTestUser(t, pool, "sr_program_producer")
	player := insertShowRunTestUser(t, pool, "sr_program_player")
	audience := insertShowRunTestUser(t, pool, "sr_program_audience")
	hidden := insertShowRunTestUser(t, pool, "sr_program_hidden")
	locationID, _, showRunID := insertShowRunFixture(t, pool, producer)
	grantLocationRole(t, pool, locationID, producer, "producer")

	addMember := func(userID, role string, programVisible bool) {
		if _, err := playerprofile.EnsureWorkbook(context.Background(), pool, userID); err != nil {
			t.Fatalf("ensure workbook: %v", err)
		}
		var profileID string
		if err := pool.QueryRow(context.Background(), `SELECT id::text FROM player_profile_workbooks WHERE user_id = $1`, userID).Scan(&profileID); err != nil {
			t.Fatalf("load profile id: %v", err)
		}
		if _, err := AddRosterMember(context.Background(), pool, producer, showRunID, profileID, role, "", programVisible); err != nil {
			t.Fatalf("add roster member role=%s: %v", role, err)
		}
	}

	// Player added first, then Audience -- Audience must still sort first in
	// the program view (Kernel 66's explicit "best seats" ordering).
	addMember(player, "player", true)
	addMember(audience, "audience", true)
	addMember(hidden, "crew", false)

	entries, err := ListAudienceProgramMembers(context.Background(), pool, showRunID)
	if err != nil {
		t.Fatalf("list audience program members: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 program-visible entries (hidden crew member excluded), got %d", len(entries))
	}
	if entries[0].Role != "audience" {
		t.Fatalf("expected audience entry first, got role %q", entries[0].Role)
	}
	for _, e := range entries {
		if e.UserID == hidden {
			t.Fatalf("expected program_visible=false member to be excluded from audience program")
		}
	}
}

func TestProjectRosterMemberReflectsLiveTrailerFace(t *testing.T) {
	pool := openShowRunsTestPool(t)
	producer := insertShowRunTestUser(t, pool, "sr_live_producer")
	player := insertShowRunTestUser(t, pool, "sr_live_player")
	locationID, _, showRunID := insertShowRunFixture(t, pool, producer)
	grantLocationRole(t, pool, locationID, producer, "producer")

	setStageName(t, pool, player, "Original Roster Name")
	if _, err := playerprofile.EnsureWorkbook(context.Background(), pool, player); err != nil {
		t.Fatalf("ensure workbook: %v", err)
	}
	var profileID string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM player_profile_workbooks WHERE user_id = $1`, player).Scan(&profileID); err != nil {
		t.Fatalf("load profile id: %v", err)
	}

	member, err := AddRosterMember(context.Background(), pool, producer, showRunID, profileID, "player", "", true)
	if err != nil {
		t.Fatalf("add roster member: %v", err)
	}

	proj, err := ProjectRosterMember(context.Background(), pool, producer, member)
	if err != nil {
		t.Fatalf("project: %v", err)
	}
	if proj.StageName != "Original Roster Name" {
		t.Fatalf("stage name = %q, want %q", proj.StageName, "Original Roster Name")
	}

	setStageName(t, pool, player, "Updated Roster Name")
	updated, err := ProjectRosterMember(context.Background(), pool, producer, member)
	if err != nil {
		t.Fatalf("re-project: %v", err)
	}
	if updated.StageName != "Updated Roster Name" {
		t.Fatalf("updated stage name = %q, want %q", updated.StageName, "Updated Roster Name")
	}
}

func TestAudienceCanViewButNotManage(t *testing.T) {
	pool := openShowRunsTestPool(t)
	producer := insertShowRunTestUser(t, pool, "sr_view_producer")
	audience := insertShowRunTestUser(t, pool, "sr_view_audience")
	locationID, _, showRunID := insertShowRunFixture(t, pool, producer)
	grantLocationRole(t, pool, locationID, producer, "producer")
	grantLocationRole(t, pool, locationID, audience, "audience")

	sr, err := LoadShowRunByID(context.Background(), pool, showRunID)
	if err != nil {
		t.Fatalf("load show run: %v", err)
	}

	canView, err := CanViewShowRun(context.Background(), pool, audience, sr.LocationID)
	if err != nil {
		t.Fatalf("can view: %v", err)
	}
	if !canView {
		t.Fatalf("expected audience member to be able to view the show run")
	}

	canManage, err := CanManageShowRun(context.Background(), pool, audience, sr.LocationID)
	if err != nil {
		t.Fatalf("can manage: %v", err)
	}
	if canManage {
		t.Fatalf("expected audience member to NOT be able to manage the show run")
	}

	if _, err := AddRosterMember(context.Background(), pool, audience, showRunID, "irrelevant", "player", "", true); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for audience trying to manage roster, got %v", err)
	}
}

// TestCanViewBackstageExcludesPlainAudienceButIncludesCrew is Kernel 68
// §3.6/§1.5: Audience-only membership no longer clears the Stage Management
// backstage bar (CanViewShowRun still does, for the separate Audience
// Program route), but an active Show Run crew roster row does -- visibility
// only, not manage authority.
func TestCanViewBackstageExcludesPlainAudienceButIncludesCrew(t *testing.T) {
	pool := openShowRunsTestPool(t)
	ctx := context.Background()
	producer := insertShowRunTestUser(t, pool, "sr_backstage_producer")
	audience := insertShowRunTestUser(t, pool, "sr_backstage_audience")
	crew := insertShowRunTestUser(t, pool, "sr_backstage_crew")
	locationID, _, showRunID := insertShowRunFixture(t, pool, producer)
	grantLocationRole(t, pool, locationID, producer, "producer")
	grantLocationRole(t, pool, locationID, audience, "audience")
	grantLocationRole(t, pool, locationID, crew, "audience")

	sr, err := LoadShowRunByID(ctx, pool, showRunID)
	if err != nil {
		t.Fatalf("load show run: %v", err)
	}

	canView, err := CanViewBackstage(ctx, pool, audience, sr.LocationID)
	if err != nil {
		t.Fatalf("can view backstage (audience): %v", err)
	}
	if canView {
		t.Fatal("expected plain audience-role membership to NOT clear the backstage bar")
	}

	canView, err = CanViewBackstage(ctx, pool, producer, sr.LocationID)
	if err != nil {
		t.Fatalf("can view backstage (producer): %v", err)
	}
	if !canView {
		t.Fatal("expected producer to clear the backstage bar")
	}

	canView, err = CanViewBackstage(ctx, pool, crew, sr.LocationID)
	if err != nil {
		t.Fatalf("can view backstage (crew, before roster row): %v", err)
	}
	if canView {
		t.Fatal("expected a user with no crew roster row to NOT clear the backstage bar")
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO show_run_roster_members (show_run_id, user_id, role, added_by_user_id, program_visible)
		VALUES ($1, $2, 'crew', $3, TRUE)
	`, showRunID, crew, producer); err != nil {
		t.Fatalf("insert crew roster row: %v", err)
	}

	canView, err = CanViewBackstage(ctx, pool, crew, sr.LocationID)
	if err != nil {
		t.Fatalf("can view backstage (crew, after roster row): %v", err)
	}
	if !canView {
		t.Fatal("expected an active crew roster row to clear the backstage bar")
	}

	canManage, err := CanManageShowRun(ctx, pool, crew, sr.LocationID)
	if err != nil {
		t.Fatalf("can manage (crew): %v", err)
	}
	if canManage {
		t.Fatal("expected crew backstage visibility to NOT grant manage authority")
	}
}
