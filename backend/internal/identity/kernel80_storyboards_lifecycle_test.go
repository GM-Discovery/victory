package identity

// Kernel 80 account-lifecycle integration: owned Storyboards block
// deletion until resolved (active or archived alike, a confirmed decision
// -- see account_deletion.go's comment), authored cards and grantor
// attribution are anonymized to the tombstone account on deletion, and a
// non-owner's own grants disappear (via FK cascade) without a trace. Uses
// raw SQL against the storyboards tables directly, matching how this
// package cannot import the storyboards Go package (import-cycle
// constraint documented in account_deletion.go/account_export.go).

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/sessions"
)

func insertTestStoryboard(t *testing.T, pool *pgxpool.Pool, ownerID, title string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO storyboards (owner_user_id, title) VALUES ($1, $2) RETURNING id::text
	`, ownerID, title).Scan(&id); err != nil {
		t.Fatalf("insert test storyboard: %v", err)
	}
	return id
}

func TestAccountDeletionBlocksOwnedStoryboardUntilResolved(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()
	suffix := time.Now().UTC().Format("150405.000000")
	handle := "del_sb_owner_" + suffix
	userID := insertAccountTestUser(t, pool, handle, "Storyboard Owner")
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID) })

	boardID := insertTestStoryboard(t, pool, userID, "Del Test Board")
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM storyboards WHERE id = $1`, boardID) })

	raw, _, err := sessions.CreateSession(ctx, pool, userID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	body, _ := json.Marshal(executeDeletionRequest{ConfirmHandle: handle})

	delReq := httptest.NewRequest(http.MethodPost, "/api/account/delete", strings.NewReader(string(body)))
	delReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: raw})
	rec := httptest.NewRecorder()
	HandleAccountDelete(pool, t.TempDir()).ServeHTTP(rec, delReq)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 deletion_blocked while board is owned, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Storyboard") {
		t.Fatalf("expected blocker to mention the Storyboard, got %s", rec.Body.String())
	}

	// Archiving alone does not clear the blocker (confirmed decision:
	// archived boards block the same as active ones).
	if _, err := pool.Exec(ctx, `UPDATE storyboards SET archived_at = NOW() WHERE id = $1`, boardID); err != nil {
		t.Fatalf("archive board: %v", err)
	}
	delReq2 := httptest.NewRequest(http.MethodPost, "/api/account/delete", strings.NewReader(string(body)))
	delReq2.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: raw})
	rec2 := httptest.NewRecorder()
	HandleAccountDelete(pool, t.TempDir()).ServeHTTP(rec2, delReq2)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("expected archived board to still block deletion, got %d: %s", rec2.Code, rec2.Body.String())
	}

	// Deleting the board entirely clears the blocker.
	if _, err := pool.Exec(ctx, `DELETE FROM storyboards WHERE id = $1`, boardID); err != nil {
		t.Fatalf("delete board: %v", err)
	}
	delReq3 := httptest.NewRequest(http.MethodPost, "/api/account/delete", strings.NewReader(string(body)))
	delReq3.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: raw})
	rec3 := httptest.NewRecorder()
	HandleAccountDelete(pool, t.TempDir()).ServeHTTP(rec3, delReq3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("expected deletion to succeed once no owned boards remain, got %d: %s", rec3.Code, rec3.Body.String())
	}
}

func TestAccountDeletionAnonymizesStoryboardAuthorshipAndRemovesGrant(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()
	suffix := time.Now().UTC().Format("150405.000000")

	ownerHandle := "del_sb_owner2_" + suffix
	ownerID := insertAccountTestUser(t, pool, ownerHandle, "Board Owner")
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, ownerID) })

	memberHandle := "del_sb_member_" + suffix
	memberID := insertAccountTestUser(t, pool, memberHandle, "Board Member")

	boardID := insertTestStoryboard(t, pool, ownerID, "Anon Test Board")
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM storyboards WHERE id = $1`, boardID) })

	var bandID, rowID, colID, cardID, grantID string
	if err := pool.QueryRow(ctx, `INSERT INTO storyboard_bands (storyboard_id, label, sort_order) VALUES ($1, 'Band', 0) RETURNING id::text`, boardID).Scan(&bandID); err != nil {
		t.Fatalf("insert band: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO storyboard_rows (storyboard_id, band_id, label, sort_order_in_band) VALUES ($1, $2, 'Row', 0) RETURNING id::text`, boardID, bandID).Scan(&rowID); err != nil {
		t.Fatalf("insert row: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO storyboard_columns (storyboard_id, title, sort_order) VALUES ($1, 'Col', 0) RETURNING id::text`, boardID).Scan(&colID); err != nil {
		t.Fatalf("insert column: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO storyboard_cards (storyboard_id, row_id, column_id, sort_order_in_cell, title, author_user_id)
		VALUES ($1, $2, $3, 0, 'Member Card', $4) RETURNING id::text
	`, boardID, rowID, colID, memberID).Scan(&cardID); err != nil {
		t.Fatalf("insert card: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO storyboard_grants (storyboard_id, user_id, granted_role, granted_by)
		VALUES ($1, $2, 'crew', $3) RETURNING id::text
	`, boardID, memberID, memberID).Scan(&grantID); err != nil {
		t.Fatalf("insert grant: %v", err)
	}

	raw, _, err := sessions.CreateSession(ctx, pool, memberID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	body, _ := json.Marshal(executeDeletionRequest{ConfirmHandle: memberHandle})
	delReq := httptest.NewRequest(http.MethodPost, "/api/account/delete", strings.NewReader(string(body)))
	delReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: raw})
	rec := httptest.NewRecorder()
	HandleAccountDelete(pool, t.TempDir()).ServeHTTP(rec, delReq)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected non-owner deletion to succeed, got %d: %s", rec.Code, rec.Body.String())
	}

	var authorID string
	if err := pool.QueryRow(ctx, `SELECT author_user_id::text FROM storyboard_cards WHERE id = $1`, cardID).Scan(&authorID); err != nil {
		t.Fatalf("reload card: %v", err)
	}
	if authorID != tombstoneUserID {
		t.Fatalf("expected card author reassigned to tombstone, got %s", authorID)
	}

	var ghostGrantExists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM storyboard_grants WHERE id = $1)`, grantID).Scan(&ghostGrantExists); err != nil {
		t.Fatalf("check grant: %v", err)
	}
	if ghostGrantExists {
		t.Fatalf("expected the deleted member's own grant row to be gone (FK cascade), found a ghost grant")
	}
}

func TestAccountExportIncludesOwnedStoryboardsAndGrants(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()
	suffix := time.Now().UTC().Format("150405.000000")
	ownerHandle := "del_sb_export_" + suffix
	ownerID := insertAccountTestUser(t, pool, ownerHandle, "Export Owner")
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, ownerID) })

	boardID := insertTestStoryboard(t, pool, ownerID, "Export Board")
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM storyboards WHERE id = $1`, boardID) })

	raw, _, err := sessions.CreateSession(ctx, pool, ownerID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/account/export", nil)
	req.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: raw})
	rec := httptest.NewRecorder()
	tmpDir := t.TempDir()
	HandleRequestExport(pool, tmpDir, tmpDir).ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected export request accepted, got %d: %s", rec.Code, rec.Body.String())
	}

	// runExportJob runs in a background goroutine (see HandleRequestExport);
	// poll status briefly rather than assuming synchronous completion.
	var status string
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		statusReq := httptest.NewRequest(http.MethodGet, "/api/account/export", nil)
		statusReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: raw})
		statusRec := httptest.NewRecorder()
		HandleExportStatus(pool).ServeHTTP(statusRec, statusReq)
		var out struct {
			Ok   bool `json:"ok"`
			Data struct {
				Status string `json:"status"`
			} `json:"data"`
		}
		_ = json.Unmarshal(statusRec.Body.Bytes(), &out)
		status = out.Data.Status
		if status == "ready" || status == "failed" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if status != "ready" {
		t.Fatalf("expected export job to reach ready, got status=%q", status)
	}
}
