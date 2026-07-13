package shows

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
	"victory/backend/internal/showruns"
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

func writeOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, response{Ok: true, Data: data})
}

func writeError(w http.ResponseWriter, err error) {
	code := err.Error()
	status := http.StatusBadRequest
	switch {
	case code == "not_authenticated":
		status = http.StatusUnauthorized
	case code == "not_authorized" || code == "user_blocked" || code == "self_join_disabled":
		status = http.StatusForbidden
	case strings.HasSuffix(code, "_not_found"):
		status = http.StatusNotFound
	}
	writeJSON(w, status, response{Ok: false, Data: map[string]any{"error": code}})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
}

// HandleShowRunShows handles GET /api/show-runs/{show_run_id}/shows (list +
// bucket summary) and POST (create, manage-authorized).
func HandleShowRunShows(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		showRunID := strings.TrimSpace(r.PathValue("show_run_id"))

		sr, err := showruns.LoadShowRunByID(ctx, pool, showRunID)
		if err != nil {
			writeError(w, err)
			return
		}

		switch r.Method {
		case http.MethodGet:
			// Backstage listing (Kernel 68 §3.6), not the Audience Program bar.
			canView, err := showruns.CanViewBackstage(ctx, pool, userID, sr.LocationID)
			if err != nil {
				writeError(w, err)
				return
			}
			if !canView {
				writeError(w, errors.New("not_authorized"))
				return
			}
			list, summary, err := ListShowsForRun(ctx, pool, showRunID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"shows": list, "summary": summary})

		case http.MethodPost:
			var body struct {
				Title                string `json:"title"`
				Slug                 string `json:"slug"`
				Description          string `json:"description"`
				AudienceTitle        string `json:"audience_title"`
				AudienceProgramBlurb string `json:"audience_program_blurb"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			s, err := CreateShow(ctx, pool, userID, showRunID, CreateShowInput{
				Title:                body.Title,
				Slug:                 body.Slug,
				Description:          body.Description,
				AudienceTitle:        body.AudienceTitle,
				AudienceProgramBlurb: body.AudienceProgramBlurb,
			})
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"show": s})

		default:
			methodNotAllowed(w)
		}
	}
}

// HandleShowByID handles GET /api/shows/{show_id} (internal detail,
// viewer must pass the parent Show Run's view authority; includes the
// inherited roster for managers) and PATCH (update, manage-authorized).
func HandleShowByID(pool *pgxpool.Pool) http.HandlerFunc {
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
			s, err := LoadShowByID(ctx, pool, showID)
			if err != nil {
				writeError(w, err)
				return
			}
			sr, err := showruns.LoadShowRunByID(ctx, pool, s.ShowRunID)
			if err != nil {
				writeError(w, err)
				return
			}
			canManage, err := showruns.CanManageShowRun(ctx, pool, userID, sr.LocationID)
			if err != nil {
				writeError(w, err)
				return
			}
			// Backstage detail (Kernel 68 §3.6), not the Audience Program bar.
			canView, err := showruns.CanViewBackstage(ctx, pool, userID, sr.LocationID)
			if err != nil {
				writeError(w, err)
				return
			}
			if !canView {
				writeError(w, errors.New("not_authorized"))
				return
			}

			var roster []showruns.RosterMemberProjection
			if canManage {
				members, err := ListInternalRosterForShow(ctx, pool, s)
				if err != nil {
					writeError(w, err)
					return
				}
				roster = make([]showruns.RosterMemberProjection, 0, len(members))
				for _, m := range members {
					proj, err := showruns.ProjectRosterMember(ctx, pool, userID, m)
					if err != nil {
						continue
					}
					roster = append(roster, proj)
				}
			}

			// production_id is surfaced here (not a Show field itself) so
			// callers -- e.g. the Scenes section on the Show detail page,
			// Kernel 69 -- can fetch the Production-scoped Scene Library
			// without a second round trip through the Show Run.
			writeOK(w, map[string]any{"show": s, "can_manage": canManage, "roster": roster, "production_id": sr.ProductionID})

		case http.MethodPatch:
			var body struct {
				Title                *string `json:"title"`
				Description          *string `json:"description"`
				AudienceTitle        *string `json:"audience_title"`
				AudienceProgramBlurb *string `json:"audience_program_blurb"`
				Status               *string `json:"status"`
				ScheduledStartAt     *string `json:"scheduled_start_at"`
				ScheduledEndAt       *string `json:"scheduled_end_at"`
				ActualStartAt        *string `json:"actual_start_at"`
				ActualEndAt          *string `json:"actual_end_at"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			s, err := UpdateShow(ctx, pool, userID, showID, UpdateShowPatch{
				Title:                body.Title,
				Description:          body.Description,
				AudienceTitle:        body.AudienceTitle,
				AudienceProgramBlurb: body.AudienceProgramBlurb,
				Status:               body.Status,
				ScheduledStartAt:     body.ScheduledStartAt,
				ScheduledEndAt:       body.ScheduledEndAt,
				ActualStartAt:        body.ActualStartAt,
				ActualEndAt:          body.ActualEndAt,
			})
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"show": s})

		default:
			methodNotAllowed(w)
		}
	}
}

// HandleShowArchive handles POST /api/shows/{show_id}/archive.
func HandleShowArchive(pool *pgxpool.Pool) http.HandlerFunc {
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
		s, err := ArchiveShow(ctx, pool, userID, strings.TrimSpace(r.PathValue("show_id")))
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"show": s})
	}
}

// HandleShowProgram handles GET /api/shows/{show_id}/program -- the
// Show's own curated Audience Program: its audience_title/
// audience_program_blurb plus the parent Show Run's roster projected as
// Audience Program entries. can_self_join delegates to the Show Run's
// existing audience_self_join_enabled/block state -- joining a Show's
// audience is really joining its Show Run's roster (showruns.SelfJoinAsAudience),
// there is no Show-scoped self-join mechanism.
func HandleShowProgram(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
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

		s, err := LoadShowByID(ctx, pool, showID)
		if err != nil {
			writeError(w, err)
			return
		}
		sr, err := showruns.LoadShowRunByID(ctx, pool, s.ShowRunID)
		if err != nil {
			writeError(w, err)
			return
		}
		canView, err := showruns.CanViewShowRun(ctx, pool, userID, sr.LocationID)
		if err != nil {
			writeError(w, err)
			return
		}
		if !canView {
			writeError(w, errors.New("not_authorized"))
			return
		}

		members, err := ListAudienceProgramForShow(ctx, pool, s)
		if err != nil {
			writeError(w, err)
			return
		}
		entries := make([]showruns.AudienceProgramEntry, 0, len(members))
		for _, m := range members {
			entry, err := showruns.ProjectAudienceProgramEntry(ctx, pool, userID, m)
			if err != nil {
				continue
			}
			entries = append(entries, entry)
		}

		alreadyOnRoster := false
		for _, m := range members {
			if m.UserID == userID {
				alreadyOnRoster = true
				break
			}
		}
		blocked, err := showruns.IsUserBlocked(ctx, pool, sr.ID, userID)
		if err != nil {
			writeError(w, err)
			return
		}
		canSelfJoin := sr.AudienceSelfJoinEnabled && !alreadyOnRoster && !blocked

		writeOK(w, map[string]any{
			"show":          s,
			"program":       entries,
			"can_self_join": canSelfJoin,
		})
	}
}

// HandleShowSessionLink handles POST /api/shows/{show_id}/sessions/{session_id}/link.
func HandleShowSessionLink(pool *pgxpool.Pool) http.HandlerFunc {
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
		sessionID := strings.TrimSpace(r.PathValue("session_id"))
		if err := LinkSessionToShow(ctx, pool, userID, showID, sessionID); err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"linked": true})
	}
}

// HandleShowSessionUnlink handles POST /api/shows/{show_id}/sessions/{session_id}/unlink.
func HandleShowSessionUnlink(pool *pgxpool.Pool) http.HandlerFunc {
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
		sessionID := strings.TrimSpace(r.PathValue("session_id"))
		if err := UnlinkSessionFromShow(ctx, pool, userID, showID, sessionID); err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"unlinked": true})
	}
}
