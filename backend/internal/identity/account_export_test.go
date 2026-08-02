package identity

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"victory/backend/internal/sessions"
)

func TestAccountExportOwnDataEndToEnd(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()

	suffix := time.Now().UTC().Format("150405.000000")
	handle := "export_owner_" + suffix
	userID := insertAccountTestUser(t, pool, handle, "Export Owner")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM account_export_jobs WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	locationID := resolveAccountTestLocationID(t, pool, "amurray-family")
	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, 'audience', TRUE)
	`, locationID, userID); err != nil {
		t.Fatalf("insert membership: %v", err)
	}

	var characterID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO character_cards (owner_user_id, location_id, name)
		VALUES ($1, $2, 'Export Test Character')
		RETURNING id::text
	`, userID, locationID).Scan(&characterID); err != nil {
		t.Fatalf("insert character card: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO character_journals (character_card_id, author_user_id, body)
		VALUES ($1, $2, 'this is my private journal entry, export me please')
	`, characterID, userID); err != nil {
		t.Fatalf("insert journal: %v", err)
	}

	raw, _, err := sessions.CreateSession(ctx, pool, userID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	cookie := &http.Cookie{Name: sessions.CookieName, Value: raw}

	storageRoot := t.TempDir()
	exportsRoot := t.TempDir()

	reqReq := httptest.NewRequest(http.MethodPost, "/api/account/export", nil)
	reqReq.AddCookie(cookie)
	reqRec := httptest.NewRecorder()
	HandleExportCollection(pool, storageRoot, exportsRoot).ServeHTTP(reqRec, reqReq)
	if reqRec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 accepted, got %d: %s", reqRec.Code, reqRec.Body.String())
	}

	// Poll status until ready (the archive is built in a background goroutine).
	var lastStatus string
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		statusReq := httptest.NewRequest(http.MethodGet, "/api/account/export/status", nil)
		statusReq.AddCookie(cookie)
		statusRec := httptest.NewRecorder()
		HandleExportStatus(pool).ServeHTTP(statusRec, statusReq)
		var payload struct {
			Ok   bool            `json:"ok"`
			Data ExportJobStatus `json:"data"`
		}
		if err := json.Unmarshal(statusRec.Body.Bytes(), &payload); err == nil {
			lastStatus = payload.Data.Status
			if lastStatus == "ready" {
				break
			}
			if lastStatus == "failed" {
				t.Fatalf("export job failed: %s", payload.Data.Error)
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	if lastStatus != "ready" {
		t.Fatalf("export did not become ready in time, last status %q", lastStatus)
	}

	downReq := httptest.NewRequest(http.MethodGet, "/api/account/export/download", nil)
	downReq.AddCookie(cookie)
	downRec := httptest.NewRecorder()
	HandleExportDownload(pool).ServeHTTP(downRec, downReq)
	if downRec.Code != http.StatusOK {
		t.Fatalf("expected 200 download, got %d", downRec.Code)
	}

	zipBytes := downRec.Body.Bytes()
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("archive did not reopen as a valid zip: %v", err)
	}

	var names []string
	var manifestData, journalData []byte
	for _, f := range zr.File {
		names = append(names, f.Name)
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		data, _ := io.ReadAll(rc)
		rc.Close()
		if f.Name == "manifest.json" {
			manifestData = data
		}
		if strings.Contains(f.Name, "journal.md") {
			journalData = data
		}
	}

	if !containsAll(names, "README.md", "manifest.json", "account/profile.json", "characters/index.json") {
		t.Fatalf("archive missing expected top-level files, got %v", names)
	}
	if journalData == nil || !strings.Contains(string(journalData), "this is my private journal entry") {
		t.Fatalf("expected the private journal entry to be present in the export, files: %v", names)
	}

	var manifest map[string]any
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("manifest.json did not parse: %v", err)
	}
	counts, _ := manifest["counts"].(map[string]any)
	if counts == nil || counts["characters"] != float64(1) {
		t.Fatalf("expected manifest counts.characters == 1, got %+v", manifest["counts"])
	}

	// No secrets anywhere in the archive.
	full := string(zipBytes)
	for _, forbidden := range []string{"argon2id", "password_hash", raw} {
		if strings.Contains(full, forbidden) {
			t.Fatalf("export archive leaked a secret-shaped string: %q", forbidden)
		}
	}
}

func TestAccountExportCrossUserCannotDownloadOrRequest(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()

	suffix := time.Now().UTC().Format("150405.000000")
	ownerID := insertAccountTestUser(t, pool, "export_a_"+suffix, "Export Owner A")
	otherID := insertAccountTestUser(t, pool, "export_b_"+suffix, "Export Other B")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM account_export_jobs WHERE user_id IN ($1, $2)`, ownerID, otherID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id IN ($1, $2)`, ownerID, otherID)
	})

	// Directly insert a ready job for the owner so we don't need to wait on
	// the background builder for this test.
	storageRoot := t.TempDir()
	exportsRoot := t.TempDir()
	archivePath := exportsRoot + "/owner-export.zip"
	if err := os.WriteFile(archivePath, []byte("PK\x03\x04fake"), 0o600); err != nil {
		t.Fatalf("write fake archive: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_export_jobs (user_id, status, archive_path, expires_at)
		VALUES ($1, 'ready', $2, NOW() + interval '1 hour')
	`, ownerID, archivePath); err != nil {
		t.Fatalf("insert ready job: %v", err)
	}

	otherRaw, _, err := sessions.CreateSession(ctx, pool, otherID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create other session: %v", err)
	}

	downReq := httptest.NewRequest(http.MethodGet, "/api/account/export/download", nil)
	downReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: otherRaw})
	downRec := httptest.NewRecorder()
	HandleExportDownload(pool).ServeHTTP(downRec, downReq)
	if downRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for another user's export, got %d: %s", downRec.Code, downRec.Body.String())
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/api/account/export/status", nil)
	statusReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: otherRaw})
	statusRec := httptest.NewRecorder()
	HandleExportStatus(pool).ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 status for a user with no export of their own, got %d", statusRec.Code)
	}

	unauthReq := httptest.NewRequest(http.MethodPost, "/api/account/export", nil)
	unauthRec := httptest.NewRecorder()
	HandleExportCollection(pool, storageRoot, exportsRoot).ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated export request, got %d", unauthRec.Code)
	}
}

func TestAccountExportExpiredArchiveRefused(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()

	suffix := time.Now().UTC().Format("150405.000000")
	userID := insertAccountTestUser(t, pool, "export_expired_"+suffix, "Export Expired")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM account_export_jobs WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	exportsRoot := t.TempDir()
	archivePath := exportsRoot + "/expired-export.zip"
	if err := os.WriteFile(archivePath, []byte("PK\x03\x04fake"), 0o600); err != nil {
		t.Fatalf("write fake archive: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_export_jobs (user_id, status, archive_path, expires_at)
		VALUES ($1, 'ready', $2, NOW() - interval '1 hour')
	`, userID, archivePath); err != nil {
		t.Fatalf("insert expired job: %v", err)
	}

	raw, _, err := sessions.CreateSession(ctx, pool, userID, 24*time.Hour, httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	downReq := httptest.NewRequest(http.MethodGet, "/api/account/export/download", nil)
	downReq.AddCookie(&http.Cookie{Name: sessions.CookieName, Value: raw})
	downRec := httptest.NewRecorder()
	HandleExportDownload(pool).ServeHTTP(downRec, downReq)
	if downRec.Code != http.StatusGone {
		t.Fatalf("expected 410 for expired export, got %d: %s", downRec.Code, downRec.Body.String())
	}

	if _, err := os.Stat(archivePath); !os.IsNotExist(err) {
		t.Fatalf("expected expired archive file to be removed from disk, stat err = %v", err)
	}
}

func containsAll(haystack []string, needles ...string) bool {
	set := map[string]bool{}
	for _, h := range haystack {
		set[h] = true
	}
	for _, n := range needles {
		if !set[n] {
			return false
		}
	}
	return true
}
