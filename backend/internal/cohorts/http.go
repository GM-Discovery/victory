package cohorts

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
	"victory/backend/internal/network"
)

type response struct {
	Ok   bool `json:"ok"`
	Data any  `json:"data,omitempty"`
}

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

func writeOK(w http.ResponseWriter, data any) { writeJSON(w, http.StatusOK, response{Ok: true, Data: data}) }

func writeError(w http.ResponseWriter, err error) {
	code := err.Error()
	status := http.StatusBadRequest
	switch {
	case code == "not_authenticated":
		status = http.StatusUnauthorized
	case code == "not_authorized":
		status = http.StatusForbidden
	case strings.HasSuffix(code, "_not_found"):
		status = http.StatusNotFound
	case strings.HasSuffix(code, "_mismatch") || code == "cohort_archived" || code == "placement_not_eligible" || code == "user_not_a_show_participant":
		status = http.StatusConflict
	}
	writeJSON(w, status, response{Ok: false, Data: map[string]any{"error": code}})
}

func methodNotAllowed(w http.ResponseWriter) { writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false}) }

// HandleCohortsCollection handles GET (full roster: cohorts+members+
// Ungrouped) and POST (create the next cohort) for /api/shows/{show_id}/cohorts.
func HandleCohortsCollection(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		showID := strings.TrimSpace(r.PathValue("show_id"))

		switch r.Method {
		case http.MethodGet:
			roster, err := ListRosterForShow(ctx, pool, userID, showID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, roster)
		case http.MethodPost:
			c, err := CreateCohort(ctx, pool, userID, showID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"cohort": c})
		default:
			methodNotAllowed(w)
		}
	}
}

// HandleCohortArchive handles POST /api/shows/{show_id}/cohorts/{cohort_id}/archive.
func HandleCohortArchive(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		showID := strings.TrimSpace(r.PathValue("show_id"))
		cohortID := strings.TrimSpace(r.PathValue("cohort_id"))
		if err := ArchiveCohort(ctx, pool, userID, showID, cohortID); err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"archived": true})
	}
}

// HandleCohortAssignment handles POST /api/shows/{show_id}/cohorts/{cohort_id}/assignments
// (body: {"user_id": "..."}) and DELETE
// /api/shows/{show_id}/cohorts/assignments/{user_id} (return to Ungrouped).
func HandleCohortAssignment(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		showID := strings.TrimSpace(r.PathValue("show_id"))

		switch r.Method {
		case http.MethodPost:
			cohortID := strings.TrimSpace(r.PathValue("cohort_id"))
			var body struct {
				UserID string `json:"user_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			if err := AssignParticipant(ctx, pool, userID, showID, cohortID, body.UserID); err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"assigned": true})
		default:
			methodNotAllowed(w)
		}
	}
}

// HandleCohortUnassign handles DELETE
// /api/shows/{show_id}/cohorts/assignments/{user_id}.
func HandleCohortUnassign(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		showID := strings.TrimSpace(r.PathValue("show_id"))
		targetUserID := strings.TrimSpace(r.PathValue("user_id"))
		if err := UnassignParticipant(ctx, pool, userID, showID, targetUserID); err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"unassigned": true})
	}
}

// HandleCohortCurrentScene handles POST
// /api/shows/{show_id}/cohorts/{cohort_id}/current-scene (body:
// {"show_scene_placement_id": "..."}, empty clears it). Broadcasts the same
// show-stage invalidation event Kernel 73A's shared-stage change already
// does -- every connected client refetches its snapshot, and only viewers
// resolved into this cohort will see anything different (kernel-85 S5.3,
// S11).
func HandleCohortCurrentScene(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		showID := strings.TrimSpace(r.PathValue("show_id"))
		cohortID := strings.TrimSpace(r.PathValue("cohort_id"))

		var body struct {
			ShowScenePlacementID string `json:"show_scene_placement_id"`
		}
		if r.ContentLength != 0 {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
		}

		var c Cohort
		reason := "cohort_scene_cleared"
		if strings.TrimSpace(body.ShowScenePlacementID) == "" {
			c, err = ClearSceneForCohort(ctx, pool, userID, showID, cohortID)
		} else {
			c, err = ActivateSceneForCohort(ctx, pool, userID, showID, cohortID, body.ShowScenePlacementID)
			reason = "cohort_scene_set"
		}
		if err != nil {
			writeError(w, err)
			return
		}
		network.BroadcastShowStageInvalidation(ctx, hub, pool, showID, reason)
		writeOK(w, map[string]any{"cohort": c})
	}
}
