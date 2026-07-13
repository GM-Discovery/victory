package playerprofile

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
)

func openReadinessTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return dbtest.OpenTestPool(t)
}

func readinessTestSuffix(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000")
}

func insertReadinessTestUser(t *testing.T, pool *pgxpool.Pool, handlePrefix string) string {
	t.Helper()
	ctx := context.Background()
	handle := handlePrefix + "_" + readinessTestSuffix(t)

	var userID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name)
		VALUES ($1, $2)
		RETURNING id::text
	`, handle, handlePrefix).Scan(&userID); err != nil {
		t.Fatalf("insert user %q: %v", handle, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM player_profile_events WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM player_profile_facts WHERE workbook_id IN (SELECT id FROM player_profile_workbooks WHERE user_id = $1)`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM player_profile_workbooks WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM player_stage_name_history WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})
	return userID
}

func TestTrailerFaceReadyRequiresAuthentication(t *testing.T) {
	pool := openReadinessTestPool(t)
	if _, err := TrailerFaceReady(context.Background(), pool, ""); err == nil {
		t.Fatal("expected not_authenticated error for blank userID")
	}
}

func TestTrailerFaceReadyBrandNewAccountNotReady(t *testing.T) {
	pool := openReadinessTestPool(t)
	userID := insertReadinessTestUser(t, pool, "ready_new")

	result, err := TrailerFaceReady(context.Background(), pool, userID)
	if err != nil {
		t.Fatalf("TrailerFaceReady: %v", err)
	}
	if result.Ready {
		t.Fatal("expected a brand new account to not be ready")
	}
	if result.ReasonCode != ReasonMissingFaceCommit {
		t.Fatalf("expected reason %q, got %q", ReasonMissingFaceCommit, result.ReasonCode)
	}
	if result.HasStageName {
		t.Fatal("expected HasStageName false for a brand new account")
	}
}

func TestTrailerFaceReadyStageNameAloneIsNotEnough(t *testing.T) {
	pool := openReadinessTestPool(t)
	userID := insertReadinessTestUser(t, pool, "ready_stageonly")

	if _, err := ChangeStageName(context.Background(), pool, userID, userID, "Stage Only"); err != nil {
		t.Fatalf("ChangeStageName: %v", err)
	}

	result, err := TrailerFaceReady(context.Background(), pool, userID)
	if err != nil {
		t.Fatalf("TrailerFaceReady: %v", err)
	}
	if result.Ready {
		t.Fatal("expected stage name alone to not be enough for readiness")
	}
	if result.ReasonCode != ReasonMissingVisibleFaceField {
		t.Fatalf("expected reason %q, got %q", ReasonMissingVisibleFaceField, result.ReasonCode)
	}
	if !result.HasStageName {
		t.Fatal("expected HasStageName true")
	}
}

func TestTrailerFaceReadyStageNamePlusVisibleFieldIsReady(t *testing.T) {
	pool := openReadinessTestPool(t)
	userID := insertReadinessTestUser(t, pool, "ready_full")
	ctx := context.Background()

	if _, err := ChangeStageName(ctx, pool, userID, userID, "Full Face"); err != nil {
		t.Fatalf("ChangeStageName: %v", err)
	}
	if _, _, err := CommitPlayerProfilePage(ctx, pool, userID, "identity_presentation", map[string]any{"real_name": "Test Real Name"}); err != nil {
		t.Fatalf("CommitPlayerProfilePage: %v", err)
	}

	result, err := TrailerFaceReady(ctx, pool, userID)
	if err != nil {
		t.Fatalf("TrailerFaceReady: %v", err)
	}
	if !result.Ready {
		t.Fatalf("expected readiness true, got reason %q", result.ReasonCode)
	}
	if result.ReasonCode != ReasonFaceReady {
		t.Fatalf("expected reason %q, got %q", ReasonFaceReady, result.ReasonCode)
	}
	if result.VisibleFieldCount < 1 {
		t.Fatalf("expected at least one visible field, got %d", result.VisibleFieldCount)
	}
}

func TestTrailerFaceReadyHiddenFieldDoesNotCount(t *testing.T) {
	pool := openReadinessTestPool(t)
	userID := insertReadinessTestUser(t, pool, "ready_hidden")
	ctx := context.Background()

	if _, err := ChangeStageName(ctx, pool, userID, userID, "Hidden Field"); err != nil {
		t.Fatalf("ChangeStageName: %v", err)
	}
	if _, _, err := CommitPlayerProfilePage(ctx, pool, userID, "identity_presentation", map[string]any{"real_name": "Test Real Name"}); err != nil {
		t.Fatalf("CommitPlayerProfilePage: %v", err)
	}
	if err := SetFaceVisibility(ctx, pool, userID, "real_name", VisibilityHidden); err != nil {
		t.Fatalf("SetFaceVisibility: %v", err)
	}

	result, err := TrailerFaceReady(ctx, pool, userID)
	if err != nil {
		t.Fatalf("TrailerFaceReady: %v", err)
	}
	if result.Ready {
		t.Fatal("expected readiness false when the only visible-eligible fact is hidden")
	}
	if result.ReasonCode != ReasonMissingVisibleFaceField {
		t.Fatalf("expected reason %q, got %q", ReasonMissingVisibleFaceField, result.ReasonCode)
	}
}
