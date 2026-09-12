package playerprofile

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
)

type response struct {
	Ok   bool `json:"ok"`
	Data any  `json:"data,omitempty"`
}

// ProjectionChangeNotifier lets the HTTP layer push a trusted
// `player_profile/projection_updated` invalidation after a mutation without
// this package importing the network/websocket package directly (same
// injectable-callback shape as characters.ProjectionChangeNotifier, to
// avoid an import cycle -- Kernel 61 §6.7, §8.6).
type ProjectionChangeNotifier func(ctx context.Context, userID string, changed []string)

func requireAuthenticatedUser(ctx context.Context, pool *pgxpool.Pool, r *http.Request) (string, error) {
	sessionCookie := ""
	if c, err := r.Cookie("victory_session"); err == nil {
		sessionCookie = c.Value
	}

	userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
	if err != nil || strings.TrimSpace(userID) == "" {
		return "", errors.New("not_authenticated")
	}
	return userID, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, response{Ok: true, Data: data})
}

func writeError(w http.ResponseWriter, err error) {
	code := err.Error()
	status := http.StatusBadRequest
	switch {
	case code == "not_authenticated":
		status = http.StatusUnauthorized
	case code == "forbidden" || strings.HasPrefix(code, "field_not_face_eligible") || code == "face_visibility_locked" || code == "face_priority_locked":
		status = http.StatusForbidden
	case strings.HasSuffix(code, "_not_found") || code == "workbook_not_found" || code == "event_not_found" || code == "unknown_page" || code == "profile_not_found":
		status = http.StatusNotFound
	}
	writeJSON(w, status, response{Ok: false, Data: map[string]any{"error": code}})
}

// HandleCatalogue serves the versioned page/question catalogue that drives
// both backend validation and frontend rendering (Kernel 61 §6.2, §9.5).
func HandleCatalogue(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if _, err := requireAuthenticatedUser(ctx, pool, r); err != nil {
			writeError(w, err)
			return
		}

		cat, err := LoadCatalogue()
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, cat)
	}
}

// HandleOwnerWorkbook serves the full owner-only projection (Kernel 61 §9.7).
func HandleOwnerWorkbook(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}

		view, err := ProjectOwnerWorkbook(ctx, pool, userID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, view)
	}
}

type pageCommitRequest struct {
	Answers map[string]any `json:"answers"`
}

// HandlePageCommit handles POST /api/player-profile/pages/{page_key}/commit
// (Kernel 61 §9.5). The server -- never the client -- resolves the
// authenticated user and generates the event summary.
func HandlePageCommit(pool *pgxpool.Pool, notify ProjectionChangeNotifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/api/player-profile/pages/")
		if !strings.HasSuffix(path, "/commit") {
			writeJSON(w, http.StatusNotFound, response{Ok: false})
			return
		}
		pageKey := strings.TrimSuffix(path, "/commit")

		var input pageCommitRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}

		event, changed, err := CommitPlayerProfilePage(ctx, pool, userID, pageKey, input.Answers)
		if err != nil {
			writeError(w, err)
			return
		}

		if changed && notify != nil {
			notify(ctx, userID, []string{"page:" + pageKey})
		}

		writeOK(w, map[string]any{"changed": changed, "event": event})
	}
}

// HandleDeleteEvent handles DELETE /api/player-profile/events/{event_id}
// (Kernel 61 §8.5). Stage-name ledger IDs live in a separate table and are
// never reachable through this path.
func HandleDeleteEvent(pool *pgxpool.Pool, notify ProjectionChangeNotifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}

		eventID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/player-profile/events/"), "/")
		if eventID == "" {
			writeError(w, errors.New("event_id_required"))
			return
		}

		if err := DeletePlayerProfileEvent(ctx, pool, userID, eventID); err != nil {
			writeError(w, err)
			return
		}

		if notify != nil {
			notify(ctx, userID, []string{"event_deleted"})
		}

		writeOK(w, map[string]any{"deleted": true})
	}
}

type faceVisibilityRequest struct {
	FieldKey string `json:"field_key"`
	Mode     string `json:"mode"`
}

// HandleFaceVisibility handles POST /api/player-profile/face-visibility
// (Kernel 61 §9.5, §10.4). mode "inferred" resets to contract behavior.
func HandleFaceVisibility(pool *pgxpool.Pool, notify ProjectionChangeNotifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}

		var input faceVisibilityRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}

		mode := strings.ToLower(strings.TrimSpace(input.Mode))
		if mode == VisibilityInferred {
			err = ReturnFaceVisibilityToInferred(ctx, pool, userID, input.FieldKey)
		} else {
			err = SetFaceVisibility(ctx, pool, userID, input.FieldKey, mode)
		}
		if err != nil {
			writeError(w, err)
			return
		}

		if notify != nil {
			notify(ctx, userID, []string{"face_visibility"})
		}
		writeOK(w, map[string]any{"ok": true})
	}
}

type facePriorityRequest struct {
	FieldKey string `json:"field_key"`
	Mode     string `json:"mode"`
	Score    int    `json:"score"`
}

// HandleFacePriority handles POST /api/player-profile/face-priority
// (Kernel 61 §9.5, §10.4). mode "inferred" resets to the catalogue default.
func HandleFacePriority(pool *pgxpool.Pool, notify ProjectionChangeNotifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}

		var input facePriorityRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}

		if strings.ToLower(strings.TrimSpace(input.Mode)) == PriorityInferred {
			err = ReturnFacePriorityToInferred(ctx, pool, userID, input.FieldKey)
		} else {
			err = SetFacePriority(ctx, pool, userID, input.FieldKey, input.Score)
		}
		if err != nil {
			writeError(w, err)
			return
		}

		if notify != nil {
			notify(ctx, userID, []string{"face_priority"})
		}
		writeOK(w, map[string]any{"ok": true})
	}
}

// HandleFaceReadiness handles GET /api/player-profile/me/face-readiness
// (Kernel 68 §3.1, §3.3) -- lets Trailers show the unready/ready prompt
// without duplicating the readiness computation client-side.
func HandleFaceReadiness(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}

		result, err := TrailerFaceReady(ctx, pool, userID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, result)
	}
}

type stageNameRequest struct {
	StageName string `json:"stage_name"`
}

// HandleStageName handles POST /api/player-profile/stage-name (Kernel 61
// §8.3, §9.5). Only the owner may change their own stage name -- the
// target user is always the authenticated caller, never a client-supplied
// value.
func HandleStageName(pool *pgxpool.Pool, notify ProjectionChangeNotifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}

		var input stageNameRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}

		entry, err := ChangeStageName(ctx, pool, userID, userID, input.StageName)
		if err != nil {
			writeError(w, err)
			return
		}

		if notify != nil {
			notify(ctx, userID, []string{"stage_name"})
		}
		writeOK(w, entry)
	}
}

// HandleSocialFace handles GET /api/player-profile/{profile_id}/face
// (Kernel 61 §8.2). profile_id is the Player Workbook ID -- a deliberately
// opaque identifier so ordinary Trailer URLs don't need to carry the raw
// account UUID (Kernel 61 §6.1). Anonymous requests are rejected before the
// path is even parsed further.
func HandleSocialFace(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		viewerUserID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/api/player-profile/")
		if !strings.HasSuffix(path, "/face") {
			writeJSON(w, http.StatusNotFound, response{Ok: false})
			return
		}
		profileID := strings.TrimSuffix(path, "/face")
		if profileID == "" {
			writeError(w, errors.New("profile_not_found"))
			return
		}

		targetUserID, err := resolveUserIDForWorkbookID(ctx, pool, profileID)
		if err != nil {
			writeError(w, err)
			return
		}

		// Kernel 96 finding: this endpoint is what the Third Place commons
		// list (GET /api/third-place/headshots) links every OTHER Trailer
		// to, but until now it only required being logged in at all -- the
		// commons list's own "pay for a ready Trailer Face before seeing
		// anyone else's" admission rule wasn't enforced on the page that
		// rule exists to gate. Same check the list uses, applied here --
		// except for a self-view, which must always work regardless of
		// readiness (this is also how an account previews its own
		// in-progress Trailer before it's ready to publish).
		if targetUserID != viewerUserID {
			ready, err := TrailerFaceReady(ctx, pool, viewerUserID)
			if err != nil {
				writeError(w, err)
				return
			}
			if !ready.Ready {
				writeError(w, errors.New("trailer_face_not_ready"))
				return
			}
		}

		face, err := ProjectTrailerFace(ctx, pool, targetUserID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, face)
	}
}

func resolveUserIDForWorkbookID(ctx context.Context, pool *pgxpool.Pool, workbookID string) (string, error) {
	var userID string
	if err := pool.QueryRow(ctx, `
		SELECT user_id::text FROM player_profile_workbooks WHERE id = $1
	`, workbookID).Scan(&userID); err != nil {
		return "", errors.New("profile_not_found")
	}
	return userID, nil
}
