package identity

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"victory/backend/internal/access"

	"github.com/jackc/pgx/v5/pgxpool"
)

// backupStatusFile is written by scripts/backup/backup.sh after every run.
// It never contains remote tokens, credentials, or private filenames --
// see backup.sh's write_status, which writes only mode/status/timestamp/a
// short detail string.
const backupStatusFile = "/opt/victory/backups/backup-status.json"

type backupStatusPayload struct {
	Mode      string `json:"mode"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Detail    string `json:"detail"`
}

// HandleBackupStatus exposes only the non-sensitive fields Kernel 77 §12
// allows: last run's mode/status/timestamp and how long ago that was. It
// requires an authenticated operator and never returns a remote path,
// filename beyond what backup.sh itself already put in `detail`, token, or
// restore control.
func HandleBackupStatus(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID, err := currentUserID(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}
		isOperator, err := access.IsOperatorUser(ctx, pool, userID)
		if err != nil || !isOperator {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "operator_required"})
			return
		}

		data, err := os.ReadFile(backupStatusFile)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": map[string]any{"status": "no_backup_recorded_yet"}})
			return
		}

		var status backupStatusPayload
		if err := json.Unmarshal(data, &status); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "status_file_unreadable"})
			return
		}

		ageSeconds := -1.0
		if ts, err := time.Parse(time.RFC3339, status.Timestamp); err == nil {
			ageSeconds = time.Since(ts).Seconds()
		}

		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": map[string]any{
			"mode":        status.Mode,
			"status":      status.Status,
			"timestamp":   status.Timestamp,
			"detail":      status.Detail,
			"age_seconds": ageSeconds,
		}})
	}
}
