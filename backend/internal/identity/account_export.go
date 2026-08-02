package identity

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// exportExpiry bounds how long a completed export archive stays downloadable
// before it is treated as expired and swept from disk (Kernel 77 §7.5).
const exportExpiry = 48 * time.Hour

// ExportJobStatus mirrors account_export_jobs.status.
type ExportJobStatus struct {
	ID          string     `json:"id"`
	Status      string     `json:"status"`
	RequestedAt time.Time  `json:"requested_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	ByteSize    *int64     `json:"byte_size,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// HandleExportCollection serves both POST (request a new export) and DELETE
// (remove the current one) on /api/account/export, matching Kernel 77 §12's
// route shape. The underlying stdlib ServeMux in main.go dispatches by exact
// path, not method, so both verbs have to share one registration here.
func HandleExportCollection(pool *pgxpool.Pool, storageRoot, exportsRoot string) http.HandlerFunc {
	requestHandler := HandleRequestExport(pool, storageRoot, exportsRoot)
	deleteHandler := HandleDeleteExport(pool)
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			requestHandler(w, r)
		case http.MethodDelete:
			deleteHandler(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
		}
	}
}

// HandleRequestExport starts a background export job for the authenticated
// user. It never accepts a target user id -- the account is always resolved
// from the session (Kernel 77 §7.3).
func HandleRequestExport(pool *pgxpool.Pool, storageRoot, exportsRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := currentUserID(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		var existingStatus string
		err = pool.QueryRow(ctx, `
			SELECT status FROM account_export_jobs
			WHERE user_id = $1 AND status IN ('pending', 'running')
			ORDER BY requested_at DESC LIMIT 1
		`, userID).Scan(&existingStatus)
		if err == nil {
			writeJSON(w, http.StatusConflict, map[string]any{"ok": false, "error": "export_already_in_progress"})
			return
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "export_check_failed"})
			return
		}

		var jobID string
		if err := pool.QueryRow(ctx, `
			INSERT INTO account_export_jobs (user_id, status) VALUES ($1, 'pending') RETURNING id::text
		`, userID).Scan(&jobID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "export_create_failed"})
			return
		}

		// Runs in the background past the lifetime of this request -- a real
		// account's export can take longer than is reasonable to hold one
		// HTTP request open for (§7.1). A fresh context is used deliberately;
		// r.Context() would be cancelled the moment this handler returns.
		go runExportJob(jobID, userID, storageRoot, exportsRoot, pool)

		writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "data": map[string]any{"id": jobID, "status": "pending"}})
	}
}

func HandleExportStatus(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := currentUserID(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		job, err := latestExportJob(ctx, pool, userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "no_export_found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "export_status_failed"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": job})
	}
}

// HandleExportDownload streams the authenticated user's own ready, unexpired
// export archive. Resolution is entirely session-based -- no token or id is
// accepted from the client (Kernel 77 §7.3, §7.5's "authenticated download"
// alternative to an unguessable token).
func HandleExportDownload(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := currentUserID(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		var archivePath string
		var expiresAt time.Time
		err = pool.QueryRow(ctx, `
			SELECT archive_path, expires_at FROM account_export_jobs
			WHERE user_id = $1 AND status = 'ready'
			ORDER BY requested_at DESC LIMIT 1
		`, userID).Scan(&archivePath, &expiresAt)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "no_export_ready"})
			return
		}
		if time.Now().After(expiresAt) {
			_ = expireExportJob(ctx, pool, userID)
			writeJSON(w, http.StatusGone, map[string]any{"ok": false, "error": "export_expired"})
			return
		}

		f, err := os.Open(archivePath)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "export_file_missing"})
			return
		}
		defer f.Close()

		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="victory-export.zip"`)
		_, _ = io.Copy(w, f)
	}
}

func HandleDeleteExport(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := currentUserID(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		if err := expireExportJob(ctx, pool, userID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "export_delete_failed"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func latestExportJob(ctx context.Context, pool *pgxpool.Pool, userID string) (*ExportJobStatus, error) {
	job := &ExportJobStatus{}
	var byteSize *int64
	var completedAt, expiresAt *time.Time
	var errMsg *string
	if err := pool.QueryRow(ctx, `
		SELECT id::text, status, requested_at, completed_at, expires_at, byte_size, error
		FROM account_export_jobs
		WHERE user_id = $1
		ORDER BY requested_at DESC LIMIT 1
	`, userID).Scan(&job.ID, &job.Status, &job.RequestedAt, &completedAt, &expiresAt, &byteSize, &errMsg); err != nil {
		return nil, err
	}
	job.CompletedAt = completedAt
	job.ExpiresAt = expiresAt
	job.ByteSize = byteSize
	if errMsg != nil {
		job.Error = *errMsg
	}

	if job.Status == "ready" && expiresAt != nil && time.Now().After(*expiresAt) {
		_ = expireExportJob(ctx, pool, userID)
		job.Status = "expired"
	}
	return job, nil
}

func expireExportJob(ctx context.Context, pool *pgxpool.Pool, userID string) error {
	var archivePath *string
	if err := pool.QueryRow(ctx, `
		SELECT archive_path FROM account_export_jobs
		WHERE user_id = $1 AND status IN ('ready', 'pending', 'running')
		ORDER BY requested_at DESC LIMIT 1
	`, userID).Scan(&archivePath); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	if archivePath != nil && *archivePath != "" {
		_ = os.Remove(*archivePath)
	}
	_, err := pool.Exec(ctx, `
		UPDATE account_export_jobs SET status = 'expired', archive_path = NULL
		WHERE user_id = $1 AND status IN ('ready', 'pending', 'running')
	`, userID)
	return err
}

// SweepExpiredExports removes archives (and marks jobs expired) for every
// export past its expiry, regardless of whether anyone has checked on it.
// Intended to run periodically from main() -- see Kernel 77 §7.5 "cleanup
// of expired archives."
func SweepExpiredExports(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, archive_path FROM account_export_jobs
		WHERE status = 'ready' AND expires_at < NOW()
	`)
	if err != nil {
		return 0, err
	}
	type row struct{ id, path string }
	var expired []row
	for rows.Next() {
		var r row
		var path *string
		if err := rows.Scan(&r.id, &path); err != nil {
			rows.Close()
			return 0, err
		}
		if path != nil {
			r.path = *path
		}
		expired = append(expired, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	for _, r := range expired {
		if r.path != "" {
			_ = os.Remove(r.path)
		}
		if _, err := pool.Exec(ctx, `UPDATE account_export_jobs SET status = 'expired', archive_path = NULL WHERE id = $1`, r.id); err != nil {
			return len(expired), err
		}
	}
	return len(expired), nil
}

var exportSafeNamePattern = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func exportSafeName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "unnamed"
	}
	return exportSafeNamePattern.ReplaceAllString(s, "-")
}

// runExportJob builds the archive on disk and updates the job row.
// Deliberately uses context.Background(): the HTTP request that triggered
// it has already returned by the time this runs.
func runExportJob(jobID, userID, storageRoot, exportsRoot string, pool *pgxpool.Pool) {
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE account_export_jobs SET status = 'running' WHERE id = $1`, jobID); err != nil {
		return
	}

	archivePath, byteSize, err := buildExportArchive(ctx, pool, userID, jobID, storageRoot, exportsRoot)
	if err != nil {
		errMsg := err.Error()
		_, _ = pool.Exec(ctx, `UPDATE account_export_jobs SET status = 'failed', error = $2 WHERE id = $1`, jobID, errMsg)
		return
	}

	expiresAt := time.Now().UTC().Add(exportExpiry)
	_, _ = pool.Exec(ctx, `
		UPDATE account_export_jobs
		SET status = 'ready', completed_at = NOW(), expires_at = $2, archive_path = $3, byte_size = $4
		WHERE id = $1
	`, jobID, expiresAt, archivePath, byteSize)
}

func buildExportArchive(ctx context.Context, pool *pgxpool.Pool, userID, jobID, storageRoot, exportsRoot string) (string, int64, error) {
	stagingDir := filepath.Join(exportsRoot, "staging-"+jobID)
	if err := os.MkdirAll(stagingDir, 0o700); err != nil {
		return "", 0, err
	}
	defer os.RemoveAll(stagingDir)

	writeJSONFile := func(relPath string, v any) error {
		full := filepath.Join(stagingDir, relPath)
		if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
			return err
		}
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(full, data, 0o600)
	}
	writeTextFile := func(relPath, content string) error {
		full := filepath.Join(stagingDir, relPath)
		if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
			return err
		}
		return os.WriteFile(full, []byte(content), 0o600)
	}

	manifest := map[string]any{
		"generated_at":  time.Now().UTC().Format(time.RFC3339),
		"format_version": 1,
	}
	counts := map[string]int{}

	// account/
	var handle, displayName, email string
	if err := pool.QueryRow(ctx, `SELECT handle, display_name, COALESCE(email, '') FROM users WHERE id = $1`, userID).Scan(&handle, &displayName, &email); err != nil {
		return "", 0, fmt.Errorf("load profile: %w", err)
	}
	if err := writeJSONFile("account/profile.json", map[string]any{
		"handle": handle, "display_name": displayName, "email": email, "user_id": userID,
	}); err != nil {
		return "", 0, err
	}

	membershipRows, err := pool.Query(ctx, `
		SELECT l.slug, l.name, lm.role::text, lm.active, lm.created_at
		FROM location_memberships lm JOIN locations l ON l.id = lm.location_id
		WHERE lm.user_id = $1
	`, userID)
	if err != nil {
		return "", 0, err
	}
	var memberships []map[string]any
	for membershipRows.Next() {
		var slug, name, role string
		var active bool
		var createdAt time.Time
		if err := membershipRows.Scan(&slug, &name, &role, &active, &createdAt); err != nil {
			membershipRows.Close()
			return "", 0, err
		}
		memberships = append(memberships, map[string]any{"location_slug": slug, "location_name": name, "role": role, "active": active, "created_at": createdAt})
	}
	membershipRows.Close()
	if err := membershipRows.Err(); err != nil {
		return "", 0, err
	}
	if err := writeJSONFile("account/memberships.json", memberships); err != nil {
		return "", 0, err
	}
	counts["memberships"] = len(memberships)

	// characters/
	charRows, err := pool.Query(ctx, `
		SELECT id::text, name, pronouns, tagline, public_description, private_notes, workbook_status, created_at
		FROM character_cards WHERE owner_user_id = $1
	`, userID)
	if err != nil {
		return "", 0, err
	}
	type charRow struct {
		id, name, pronouns, tagline, publicDesc, privateNotes, workbookStatus string
		createdAt                                                            time.Time
	}
	var characters []charRow
	for charRows.Next() {
		var c charRow
		if err := charRows.Scan(&c.id, &c.name, &c.pronouns, &c.tagline, &c.publicDesc, &c.privateNotes, &c.workbookStatus, &c.createdAt); err != nil {
			charRows.Close()
			return "", 0, err
		}
		characters = append(characters, c)
	}
	charRows.Close()
	if err := charRows.Err(); err != nil {
		return "", 0, err
	}

	var characterIndex []map[string]any
	for _, c := range characters {
		folder := "characters/" + exportSafeName(c.name) + "-" + c.id[:8]
		characterIndex = append(characterIndex, map[string]any{"id": c.id, "name": c.name, "folder": folder})

		if err := writeJSONFile(folder+"/character.json", map[string]any{
			"id": c.id, "name": c.name, "pronouns": c.pronouns, "tagline": c.tagline,
			"public_description": c.publicDesc, "private_notes": c.privateNotes,
			"workbook_status": c.workbookStatus, "created_at": c.createdAt,
		}); err != nil {
			return "", 0, err
		}

		journalRows, err := pool.Query(ctx, `
			SELECT body, created_at FROM character_journals
			WHERE character_card_id = $1 AND author_user_id = $2
			ORDER BY created_at ASC
		`, c.id, userID)
		if err != nil {
			return "", 0, err
		}
		var journalMD strings.Builder
		journalMD.WriteString("# Journal — " + c.name + "\n\n")
		entryCount := 0
		for journalRows.Next() {
			var body string
			var createdAt time.Time
			if err := journalRows.Scan(&body, &createdAt); err != nil {
				journalRows.Close()
				return "", 0, err
			}
			journalMD.WriteString("## " + createdAt.Format("2006-01-02 15:04") + "\n\n" + body + "\n\n")
			entryCount++
		}
		journalRows.Close()
		if err := journalRows.Err(); err != nil {
			return "", 0, err
		}
		if entryCount > 0 {
			if err := writeTextFile(folder+"/journal.md", journalMD.String()); err != nil {
				return "", 0, err
			}
		}
		counts["journal_entries"] += entryCount

		mechRows, err := pool.Query(ctx, `
			SELECT page_key, entry_type, title, body, payload FROM character_workbook_entries
			WHERE character_card_id = $1 AND author_user_id = $2
		`, c.id, userID)
		if err != nil {
			return "", 0, err
		}
		var mechEntries []map[string]any
		for mechRows.Next() {
			var pageKey, entryType, title, body string
			var payload []byte
			if err := mechRows.Scan(&pageKey, &entryType, &title, &body, &payload); err != nil {
				mechRows.Close()
				return "", 0, err
			}
			var payloadVal any
			_ = json.Unmarshal(payload, &payloadVal)
			mechEntries = append(mechEntries, map[string]any{"page_key": pageKey, "entry_type": entryType, "title": title, "body": body, "payload": payloadVal})
		}
		mechRows.Close()
		if err := mechRows.Err(); err != nil {
			return "", 0, err
		}
		if err := writeJSONFile(folder+"/mechanics.json", mechEntries); err != nil {
			return "", 0, err
		}
	}
	if err := writeJSONFile("characters/index.json", characterIndex); err != nil {
		return "", 0, err
	}
	counts["characters"] = len(characters)

	// relationships/ -- the requester's own observations only; the subject's
	// identity is included as minimal shared context (a display name), never
	// the subject's email, Discord id, or their own private notes about
	// anyone else (Kernel 77 §7.4).
	relRows, err := pool.Query(ctx, `
		SELECT pr.id::text, u.handle, pr.private_nickname, pr.relationship_state, pr.trust_level, pr.closeness_level
		FROM player_relationships pr JOIN users u ON u.id = pr.subject_user_id
		WHERE pr.observer_user_id = $1
	`, userID)
	if err != nil {
		return "", 0, err
	}
	var relationships []map[string]any
	relIDs := map[string]string{}
	for relRows.Next() {
		var id, subjectHandle, nickname, state, trust, closeness string
		if err := relRows.Scan(&id, &subjectHandle, &nickname, &state, &trust, &closeness); err != nil {
			relRows.Close()
			return "", 0, err
		}
		relationships = append(relationships, map[string]any{
			"id": id, "about": subjectHandle, "private_nickname": nickname,
			"relationship_state": state, "trust_level": trust, "closeness_level": closeness,
		})
		relIDs[id] = subjectHandle
	}
	relRows.Close()
	if err := relRows.Err(); err != nil {
		return "", 0, err
	}
	if err := writeJSONFile("relationships/relationships.json", relationships); err != nil {
		return "", 0, err
	}
	counts["relationships"] = len(relationships)

	for relID, aboutHandle := range relIDs {
		noteRows, err := pool.Query(ctx, `
			SELECT title, body, entry_date, note_category, created_at FROM player_relationship_journal_entries
			WHERE relationship_id = $1 AND created_by_user_id = $2
			ORDER BY created_at ASC
		`, relID, userID)
		if err != nil {
			return "", 0, err
		}
		var noteMD strings.Builder
		noteCount := 0
		for noteRows.Next() {
			var title, body, category string
			var entryDate *time.Time
			var createdAt time.Time
			if err := noteRows.Scan(&title, &body, &entryDate, &category, &createdAt); err != nil {
				noteRows.Close()
				return "", 0, err
			}
			noteMD.WriteString("## " + title + " (" + category + ")\n\n" + body + "\n\n")
			noteCount++
		}
		noteRows.Close()
		if err := noteRows.Err(); err != nil {
			return "", 0, err
		}
		if noteCount > 0 {
			if err := writeTextFile("relationships/notes/"+exportSafeName(aboutHandle)+".md", noteMD.String()); err != nil {
				return "", 0, err
			}
		}
	}

	// messages/ -- authored or received, never a route to any other user's
	// private inbox.
	msgRows, err := pool.Query(ctx, `
		SELECT id::text, subject, body, created_at,
			CASE WHEN from_user_id = $1 THEN 'sent' ELSE 'received' END AS direction
		FROM messages WHERE from_user_id = $1 OR to_user_id = $1
		ORDER BY created_at ASC
	`, userID)
	if err != nil {
		return "", 0, err
	}
	var messages []map[string]any
	for msgRows.Next() {
		var id, subject, body, direction string
		var createdAt time.Time
		if err := msgRows.Scan(&id, &subject, &body, &createdAt, &direction); err != nil {
			msgRows.Close()
			return "", 0, err
		}
		messages = append(messages, map[string]any{"id": id, "subject": subject, "body": body, "created_at": createdAt, "direction": direction})
	}
	msgRows.Close()
	if err := msgRows.Err(); err != nil {
		return "", 0, err
	}
	if err := writeJSONFile("messages/messages.json", messages); err != nil {
		return "", 0, err
	}
	counts["messages"] = len(messages)

	// activity/ -- a bounded reference list of the user's own authored
	// Actions, not a full production dump.
	actionRows, err := pool.Query(ctx, `
		SELECT id::text, type, ts FROM actions WHERE actor_id = $1 ORDER BY ts DESC LIMIT 2000
	`, userID)
	if err != nil {
		return "", 0, err
	}
	var actions []map[string]any
	for actionRows.Next() {
		var id, actionType string
		var ts time.Time
		if err := actionRows.Scan(&id, &actionType, &ts); err != nil {
			actionRows.Close()
			return "", 0, err
		}
		actions = append(actions, map[string]any{"id": id, "type": actionType, "ts": ts})
	}
	actionRows.Close()
	if err := actionRows.Err(); err != nil {
		return "", 0, err
	}
	if err := writeJSONFile("activity/authored-actions.json", actions); err != nil {
		return "", 0, err
	}
	counts["authored_actions"] = len(actions)

	// uploads/ -- owned or uploaded files, copied as-is.
	assetRows, err := pool.Query(ctx, `
		SELECT id::text, producer_user_id::text, original_filename, source_ext, byte_size
		FROM assets WHERE (owner_user_id = $1 OR uploader_user_id = $1) AND is_deleted = FALSE
	`, userID)
	if err != nil {
		return "", 0, err
	}
	type assetInfo struct {
		id, producerUserID, originalFilename, sourceExt string
		byteSize                                        int
	}
	var assetList []assetInfo
	for assetRows.Next() {
		var a assetInfo
		if err := assetRows.Scan(&a.id, &a.producerUserID, &a.originalFilename, &a.sourceExt, &a.byteSize); err != nil {
			assetRows.Close()
			return "", 0, err
		}
		assetList = append(assetList, a)
	}
	assetRows.Close()
	if err := assetRows.Err(); err != nil {
		return "", 0, err
	}

	var uploadManifest []map[string]any
	const maxExportBytes = 500 * 1024 * 1024 // 500MB export size ceiling (§7.1 "meaningful size limits")
	var totalCopiedBytes int64
	for _, a := range assetList {
		src := filepath.Join(storageRoot, "producers", a.producerUserID, "assets", a.id, "original", "source"+a.sourceExt)
		info := map[string]any{"id": a.id, "original_filename": a.originalFilename, "byte_size": a.byteSize}
		if totalCopiedBytes+int64(a.byteSize) > maxExportBytes {
			info["skipped"] = "export size limit reached"
			uploadManifest = append(uploadManifest, info)
			continue
		}
		destName := a.id + "-" + exportSafeName(a.originalFilename)
		dest := filepath.Join(stagingDir, "uploads", "files", destName)
		if err := copyFile(src, dest); err != nil {
			info["skipped"] = "file unavailable"
		} else {
			info["file"] = "uploads/files/" + destName
			totalCopiedBytes += int64(a.byteSize)
		}
		uploadManifest = append(uploadManifest, info)
	}
	if err := writeJSONFile("uploads/manifest.json", uploadManifest); err != nil {
		return "", 0, err
	}
	counts["uploads"] = len(assetList)

	manifest["counts"] = counts
	if err := writeJSONFile("manifest.json", manifest); err != nil {
		return "", 0, err
	}

	readme := "# Victory Data Export\n\n" +
		"This archive contains your own Victory data as of " + time.Now().UTC().Format(time.RFC3339) + ".\n\n" +
		"## Included\n\n" +
		"- account/ — your profile and Location memberships\n" +
		"- characters/ — your Characters, their journals (Markdown), and workbook mechanics\n" +
		"- relationships/ — your private relationship records and notes about other members\n" +
		"- messages/ — messages you sent or received\n" +
		"- activity/ — a reference list of Actions you authored in shared Shows\n" +
		"- uploads/ — files you uploaded or own\n\n" +
		"## Not included\n\n" +
		"Password hashes, session tokens, password-reset tokens, OAuth tokens, server secrets, " +
		"and any other member's private information are never exported.\n"
	if err := writeTextFile("README.md", readme); err != nil {
		return "", 0, err
	}

	if err := os.MkdirAll(exportsRoot, 0o700); err != nil {
		return "", 0, err
	}
	zipPath := filepath.Join(exportsRoot, jobID+".zip")
	size, err := zipDirectory(stagingDir, zipPath)
	if err != nil {
		return "", 0, err
	}
	return zipPath, size, nil
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return err
	}
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func zipDirectory(srcDir, destZip string) (int64, error) {
	out, err := os.OpenFile(destZip, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	err = filepath.Walk(srcDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		// filepath.Rel on the same platform never returns a path outside
		// srcDir here (rel is always derived from a Walk under srcDir), but
		// refuse defensively against any archive path traversal (§7.5).
		if strings.HasPrefix(rel, "..") {
			return fmt.Errorf("unsafe export path: %s", rel)
		}
		w, err := zw.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(w, f)
		return err
	})
	if err != nil {
		zw.Close()
		return 0, err
	}
	if err := zw.Close(); err != nil {
		return 0, err
	}

	fi, err := out.Stat()
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}
