package thirdplace

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/playerprofile"
	"victory/backend/internal/playerrelationships"
)

func openThirdPlaceTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return dbtest.OpenTestPool(t)
}

func insertThirdPlaceTestUser(t *testing.T, pool *pgxpool.Pool, handlePrefix string) string {
	t.Helper()
	ctx := context.Background()
	suffix := strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000")
	handle := handlePrefix + "_" + suffix

	var userID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name)
		VALUES ($1, $2)
		RETURNING id::text
	`, handle, handlePrefix).Scan(&userID); err != nil {
		t.Fatalf("insert user %q: %v", handle, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM third_place_headshots WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM player_relationships WHERE observer_user_id = $1 OR subject_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM player_stage_name_history WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM player_profile_workbooks WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.sessions WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})
	return userID
}

func setStageNameAndFace(t *testing.T, pool *pgxpool.Pool, userID, stageName, portraitURL, shortIntro string) {
	t.Helper()
	ctx := context.Background()

	if _, err := playerprofile.ChangeStageName(ctx, pool, userID, userID, stageName); err != nil {
		t.Fatalf("change stage name: %v", err)
	}
	answers := map[string]any{}
	if portraitURL != "" {
		answers["portrait_url"] = portraitURL
	}
	if shortIntro != "" {
		answers["short_intro"] = shortIntro
	}
	if len(answers) > 0 {
		if _, _, err := playerprofile.CommitPlayerProfilePage(ctx, pool, userID, "identity_presentation", answers); err != nil {
			t.Fatalf("commit identity_presentation page: %v", err)
		}
	}
}

func TestLeaveHeadshotIsIdempotentAndReleaveAfterRemovalCreatesNewRow(t *testing.T) {
	pool := openThirdPlaceTestPool(t)
	userID := insertThirdPlaceTestUser(t, pool, "headshot_owner")

	first, created, err := LeaveHeadshot(context.Background(), pool, userID)
	if err != nil {
		t.Fatalf("first leave: %v", err)
	}
	if !created {
		t.Fatalf("expected first leave to create a new row")
	}

	second, created, err := LeaveHeadshot(context.Background(), pool, userID)
	if err != nil {
		t.Fatalf("second leave: %v", err)
	}
	if created {
		t.Fatalf("expected second leave to refresh the existing row, not create a new one")
	}
	if second.ID != first.ID {
		t.Fatalf("expected same headshot id on repeated leave, got %q vs %q", first.ID, second.ID)
	}

	active, err := ListActiveHeadshots(context.Background(), pool)
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	activeCount := 0
	for _, h := range active {
		if h.UserID == userID {
			activeCount++
		}
	}
	if activeCount != 1 {
		t.Fatalf("expected exactly 1 active headshot for user, got %d", activeCount)
	}

	if err := RemoveHeadshot(context.Background(), pool, userID); err != nil {
		t.Fatalf("remove: %v", err)
	}
	// Idempotent second remove.
	if err := RemoveHeadshot(context.Background(), pool, userID); err != nil {
		t.Fatalf("second remove: %v", err)
	}

	mine, err := GetMyHeadshot(context.Background(), pool, userID)
	if err != nil {
		t.Fatalf("get my headshot after remove: %v", err)
	}
	if mine != nil {
		t.Fatalf("expected no active headshot after remove, got %+v", mine)
	}

	history, err := ListMyHeadshotHistory(context.Background(), pool, userID)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected exactly 1 history row after single remove, got %d", len(history))
	}
	if history[0].Status != "removed" || history[0].RemovedAt == nil {
		t.Fatalf("expected removed status with removed_at set, got %+v", history[0])
	}

	third, created, err := LeaveHeadshot(context.Background(), pool, userID)
	if err != nil {
		t.Fatalf("re-leave: %v", err)
	}
	if !created {
		t.Fatalf("expected re-leave after removal to create a new row")
	}
	if third.ID == first.ID {
		t.Fatalf("expected a new headshot id after re-leaving, got the same id %q", third.ID)
	}

	history, err = ListMyHeadshotHistory(context.Background(), pool, userID)
	if err != nil {
		t.Fatalf("history after re-leave: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 history rows (removed + active) after re-leave, got %d", len(history))
	}
}

func TestListActiveHeadshotsExcludesRemoved(t *testing.T) {
	pool := openThirdPlaceTestPool(t)
	userA := insertThirdPlaceTestUser(t, pool, "commons_a")
	userB := insertThirdPlaceTestUser(t, pool, "commons_b")

	if _, _, err := LeaveHeadshot(context.Background(), pool, userA); err != nil {
		t.Fatalf("leave A: %v", err)
	}
	if _, _, err := LeaveHeadshot(context.Background(), pool, userB); err != nil {
		t.Fatalf("leave B: %v", err)
	}
	if err := RemoveHeadshot(context.Background(), pool, userB); err != nil {
		t.Fatalf("remove B: %v", err)
	}

	active, err := ListActiveHeadshots(context.Background(), pool)
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	sawA, sawB := false, false
	for _, h := range active {
		if h.UserID == userA {
			sawA = true
		}
		if h.UserID == userB {
			sawB = true
		}
	}
	if !sawA {
		t.Fatalf("expected user A's active headshot in the commons list")
	}
	if sawB {
		t.Fatalf("expected user B's removed headshot to be absent from the commons list")
	}
}

func TestProjectHeadshotReflectsLiveTrailerFace(t *testing.T) {
	pool := openThirdPlaceTestPool(t)
	userID := insertThirdPlaceTestUser(t, pool, "live_face")
	setStageNameAndFace(t, pool, userID, "Original Stage Name", "https://example.test/portrait-v1.png", "Loves dice towers.")

	h, _, err := LeaveHeadshot(context.Background(), pool, userID)
	if err != nil {
		t.Fatalf("leave: %v", err)
	}

	proj, err := ProjectHeadshot(context.Background(), pool, userID, h)
	if err != nil {
		t.Fatalf("project: %v", err)
	}
	if proj.StageName != "Original Stage Name" {
		t.Fatalf("stage name = %q, want %q", proj.StageName, "Original Stage Name")
	}
	if proj.PortraitURL != "https://example.test/portrait-v1.png" {
		t.Fatalf("portrait url = %q", proj.PortraitURL)
	}
	foundIntro := false
	for _, fact := range proj.HeadlineFacts {
		if fact.Value == "Loves dice towers." {
			foundIntro = true
		}
	}
	if !foundIntro {
		t.Fatalf("expected short_intro among headline facts, got %+v", proj.HeadlineFacts)
	}
	if !strings.Contains(proj.TrailerURL, proj.ProfileID) {
		t.Fatalf("trailer url %q does not reference profile id %q", proj.TrailerURL, proj.ProfileID)
	}

	// Live re-projection after a Face change -- no snapshot to go stale.
	setStageNameAndFace(t, pool, userID, "Updated Stage Name", "https://example.test/portrait-v2.png", "")
	updated, err := ProjectHeadshot(context.Background(), pool, userID, h)
	if err != nil {
		t.Fatalf("re-project: %v", err)
	}
	if updated.StageName != "Updated Stage Name" {
		t.Fatalf("updated stage name = %q, want %q", updated.StageName, "Updated Stage Name")
	}
	if updated.PortraitURL != "https://example.test/portrait-v2.png" {
		t.Fatalf("updated portrait url = %q", updated.PortraitURL)
	}
}

func TestProjectHeadshotRelationshipStateForViewer(t *testing.T) {
	pool := openThirdPlaceTestPool(t)
	owner := insertThirdPlaceTestUser(t, pool, "rel_owner")
	viewer := insertThirdPlaceTestUser(t, pool, "rel_viewer")
	setStageNameAndFace(t, pool, owner, "Rel Owner", "", "")

	h, _, err := LeaveHeadshot(context.Background(), pool, owner)
	if err != nil {
		t.Fatalf("leave: %v", err)
	}

	// Owner viewing their own Headshot: no add/notes affordance at all.
	ownView, err := ProjectHeadshot(context.Background(), pool, owner, h)
	if err != nil {
		t.Fatalf("project as owner: %v", err)
	}
	if !ownView.IsYou || ownView.CanAddToMyPeople || ownView.CanOpenMyNotes {
		t.Fatalf("unexpected self-view projection: %+v", ownView)
	}

	// A viewer with no relationship yet: Add to My People only.
	before, err := ProjectHeadshot(context.Background(), pool, viewer, h)
	if err != nil {
		t.Fatalf("project before relationship: %v", err)
	}
	if before.IsYou || !before.CanAddToMyPeople || before.CanOpenMyNotes || before.RelationshipStateForViewer != "none" {
		t.Fatalf("unexpected pre-relationship projection: %+v", before)
	}

	rel, _, err := playerrelationships.EnsureRelationship(context.Background(), pool, viewer, before.ProfileID)
	if err != nil {
		t.Fatalf("ensure relationship: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM player_relationships WHERE id = $1`, rel.ID)
	})

	after, err := ProjectHeadshot(context.Background(), pool, viewer, h)
	if err != nil {
		t.Fatalf("project after relationship: %v", err)
	}
	if after.CanAddToMyPeople || !after.CanOpenMyNotes || after.RelationshipStateForViewer != "exists" {
		t.Fatalf("unexpected post-relationship projection: %+v", after)
	}
	if after.RelationshipID != rel.ID {
		t.Fatalf("relationship id = %q, want %q", after.RelationshipID, rel.ID)
	}

	// The owner is never told anything changed: their own Headshot and
	// relationship-facing fields are unaffected by someone else adding them.
	ownAfter, err := ProjectHeadshot(context.Background(), pool, owner, h)
	if err != nil {
		t.Fatalf("project as owner after being added: %v", err)
	}
	if ownAfter.CanAddToMyPeople || ownAfter.CanOpenMyNotes || ownAfter.RelationshipStateForViewer != "none" {
		t.Fatalf("owner's own view leaked relationship state: %+v", ownAfter)
	}

	// A third, unrelated viewer still sees "Add to My People" -- the first
	// viewer's private relationship is not visible to anyone else.
	third := insertThirdPlaceTestUser(t, pool, "rel_third")
	thirdView, err := ProjectHeadshot(context.Background(), pool, third, h)
	if err != nil {
		t.Fatalf("project as third viewer: %v", err)
	}
	if !thirdView.CanAddToMyPeople || thirdView.CanOpenMyNotes || thirdView.RelationshipStateForViewer != "none" {
		t.Fatalf("third viewer's projection leaked another viewer's relationship: %+v", thirdView)
	}
}
