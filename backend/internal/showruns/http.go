package showruns

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

func projectRosterList(ctx context.Context, pool *pgxpool.Pool, viewerUserID string, members []RosterMember) []RosterMemberProjection {
	out := make([]RosterMemberProjection, 0, len(members))
	for _, m := range members {
		proj, err := ProjectRosterMember(ctx, pool, viewerUserID, m)
		if err != nil {
			// A single broken/legacy workbook shouldn't 500 the whole roster
			// for every other viewer -- skip it, matching thirdplace's
			// HandleCollection precedent.
			continue
		}
		out = append(out, proj)
	}
	return out
}

// HandleCollection handles GET /api/show-runs (list) and POST /api/show-runs
// (create).
func HandleCollection(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}

		switch r.Method {
		case http.MethodGet:
			runs, err := ListShowRunsVisibleToUser(ctx, pool, userID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"show_runs": runs})

		case http.MethodPost:
			var body struct {
				ProductionID     string `json:"production_id"`
				Title            string `json:"title"`
				Slug             string `json:"slug"`
				Description      string `json:"description"`
				ShowFormat       string `json:"show_format"`
				CustomShowFormat string `json:"custom_show_format"`
				CohortName       string `json:"cohort_name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			sr, err := CreateShowRun(ctx, pool, userID, body.ProductionID, CreateShowRunInput{
				Title:            body.Title,
				Slug:             body.Slug,
				Description:      body.Description,
				ShowFormat:       body.ShowFormat,
				CustomShowFormat: body.CustomShowFormat,
				CohortName:       body.CohortName,
			})
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"show_run": sr})

		default:
			methodNotAllowed(w)
		}
	}
}

// HandleByID handles GET /api/show-runs/{id} and PATCH /api/show-runs/{id}.
func HandleByID(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		showRunID := strings.TrimSpace(r.PathValue("id"))

		switch r.Method {
		case http.MethodGet:
			sr, err := LoadShowRunByID(ctx, pool, showRunID)
			if err != nil {
				writeError(w, err)
				return
			}
			canManage, err := CanManageShowRun(ctx, pool, userID, sr.LocationID)
			if err != nil {
				writeError(w, err)
				return
			}
			// Backstage detail (Kernel 68 §3.6) -- stricter than the plain
			// Audience Program bar (CanViewShowRun, used by the dedicated
			// /audience-program route). Audience/Player-only users get a
			// clean rejection here rather than backstage fields.
			canViewBackstage, err := CanViewBackstage(ctx, pool, userID, sr.LocationID)
			if err != nil {
				writeError(w, err)
				return
			}
			if !canViewBackstage {
				writeError(w, errors.New("not_authorized"))
				return
			}
			writeOK(w, map[string]any{"show_run": sr, "can_manage": canManage})

		case http.MethodPatch:
			var body struct {
				Title                   *string `json:"title"`
				Description             *string `json:"description"`
				ShowFormat              *string `json:"show_format"`
				CustomShowFormat        *string `json:"custom_show_format"`
				CohortName              *string `json:"cohort_name"`
				Status                  *string `json:"status"`
				AudienceSelfJoinEnabled *bool   `json:"audience_self_join_enabled"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			sr, err := UpdateShowRun(ctx, pool, userID, showRunID, UpdateShowRunPatch{
				Title:                   body.Title,
				Description:             body.Description,
				ShowFormat:              body.ShowFormat,
				CustomShowFormat:        body.CustomShowFormat,
				CohortName:              body.CohortName,
				Status:                  body.Status,
				AudienceSelfJoinEnabled: body.AudienceSelfJoinEnabled,
			})
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"show_run": sr})

		default:
			methodNotAllowed(w)
		}
	}
}

// HandleArchive handles POST /api/show-runs/{id}/archive.
func HandleArchive(pool *pgxpool.Pool) http.HandlerFunc {
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
		sr, err := ArchiveShowRun(ctx, pool, userID, strings.TrimSpace(r.PathValue("id")))
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"show_run": sr})
	}
}

// HandleUnarchive handles POST /api/show-runs/{id}/unarchive.
func HandleUnarchive(pool *pgxpool.Pool) http.HandlerFunc {
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
		sr, err := UnarchiveShowRun(ctx, pool, userID, strings.TrimSpace(r.PathValue("id")))
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"show_run": sr})
	}
}

// HandleRosterCollection handles GET /api/show-runs/{id}/roster (internal
// roster, Producer/Director/Operator only) and POST (add a member).
func HandleRosterCollection(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		showRunID := strings.TrimSpace(r.PathValue("id"))
		sr, err := LoadShowRunByID(ctx, pool, showRunID)
		if err != nil {
			writeError(w, err)
			return
		}

		switch r.Method {
		case http.MethodGet:
			canManage, err := CanManageShowRun(ctx, pool, userID, sr.LocationID)
			if err != nil {
				writeError(w, err)
				return
			}
			if !canManage {
				writeError(w, errors.New("not_authorized"))
				return
			}
			members, err := ListInternalRoster(ctx, pool, showRunID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"roster": projectRosterList(ctx, pool, userID, members)})

		case http.MethodPost:
			var body struct {
				TargetProfileID string `json:"target_profile_id"`
				Role            string `json:"role"`
				CustomRoleLabel string `json:"custom_role_label"`
				ProgramVisible  *bool  `json:"program_visible"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			programVisible := true
			if body.ProgramVisible != nil {
				programVisible = *body.ProgramVisible
			}
			member, err := AddRosterMember(ctx, pool, userID, showRunID, body.TargetProfileID, body.Role, body.CustomRoleLabel, programVisible)
			if err != nil {
				writeError(w, err)
				return
			}
			proj, err := ProjectRosterMember(ctx, pool, userID, member)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"member": proj})

		default:
			methodNotAllowed(w)
		}
	}
}

// HandleRosterUpdate handles PATCH /api/show-runs/{id}/roster/{member_id}.
func HandleRosterUpdate(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
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
		var body struct {
			Role            string `json:"role"`
			CustomRoleLabel string `json:"custom_role_label"`
			ProgramVisible  *bool  `json:"program_visible"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		programVisible := true
		if body.ProgramVisible != nil {
			programVisible = *body.ProgramVisible
		}
		member, err := UpdateRosterMemberRole(ctx, pool, userID, strings.TrimSpace(r.PathValue("id")), strings.TrimSpace(r.PathValue("member_id")), body.Role, body.CustomRoleLabel, programVisible)
		if err != nil {
			writeError(w, err)
			return
		}
		proj, err := ProjectRosterMember(ctx, pool, userID, member)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"member": proj})
	}
}

// HandleRosterRemove handles DELETE /api/show-runs/{id}/roster/{member_id}.
func HandleRosterRemove(pool *pgxpool.Pool) http.HandlerFunc {
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
		if err := RemoveRosterMember(ctx, pool, userID, strings.TrimSpace(r.PathValue("id")), strings.TrimSpace(r.PathValue("member_id"))); err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"removed": true})
	}
}

// HandleSelfJoin handles POST /api/show-runs/{id}/roster/self-join.
func HandleSelfJoin(pool *pgxpool.Pool) http.HandlerFunc {
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
		member, err := SelfJoinAsAudience(ctx, pool, userID, strings.TrimSpace(r.PathValue("id")))
		if err != nil {
			writeError(w, err)
			return
		}
		proj, err := ProjectRosterMember(ctx, pool, userID, member)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"member": proj})
	}
}

// HandleMyRosterMember handles GET /api/show-runs/{id}/roster/me.
func HandleMyRosterMember(pool *pgxpool.Pool) http.HandlerFunc {
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
		member, err := LoadMyRosterMember(ctx, pool, userID, strings.TrimSpace(r.PathValue("id")))
		if err != nil {
			writeError(w, err)
			return
		}
		proj, err := ProjectRosterMember(ctx, pool, userID, member)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"member": proj})
	}
}

// HandleSelectCharacter handles POST /api/show-runs/{id}/roster/me/character.
// Always acts on the caller's own roster row -- the body never carries a
// target user id.
func HandleSelectCharacter(pool *pgxpool.Pool) http.HandlerFunc {
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
		var body struct {
			CharacterCardID string `json:"character_card_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		member, err := SelectCharacter(ctx, pool, userID, strings.TrimSpace(r.PathValue("id")), body.CharacterCardID)
		if err != nil {
			writeError(w, err)
			return
		}
		proj, err := ProjectRosterMember(ctx, pool, userID, member)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"member": proj})
	}
}

// HandleAudienceProgram handles GET /api/show-runs/{id}/audience-program --
// the curated view, available to any viewer with access to the run
// (CanViewShowRun), not just managers.
func HandleAudienceProgram(pool *pgxpool.Pool) http.HandlerFunc {
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
		showRunID := strings.TrimSpace(r.PathValue("id"))
		sr, err := LoadShowRunByID(ctx, pool, showRunID)
		if err != nil {
			writeError(w, err)
			return
		}
		canView, err := CanViewShowRun(ctx, pool, userID, sr.LocationID)
		if err != nil {
			writeError(w, err)
			return
		}
		if !canView {
			writeError(w, errors.New("not_authorized"))
			return
		}

		members, err := ListAudienceProgramMembers(ctx, pool, showRunID)
		if err != nil {
			writeError(w, err)
			return
		}
		entries := make([]AudienceProgramEntry, 0, len(members))
		for _, m := range members {
			entry, err := ProjectAudienceProgramEntry(ctx, pool, userID, m)
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
		blocked, err := IsUserBlocked(ctx, pool, showRunID, userID)
		if err != nil {
			writeError(w, err)
			return
		}
		canSelfJoin := sr.AudienceSelfJoinEnabled && !alreadyOnRoster && !blocked

		writeOK(w, map[string]any{
			"show_run":      sr,
			"program":       entries,
			"can_self_join": canSelfJoin,
		})
	}
}

// HandleBlock handles POST /api/show-runs/{id}/blocks.
func HandleBlock(pool *pgxpool.Pool) http.HandlerFunc {
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
		var body struct {
			TargetProfileID string `json:"target_profile_id"`
			Reason          string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		showRunID := strings.TrimSpace(r.PathValue("id"))
		targetUserID, err := resolveProfileUserID(ctx, pool, body.TargetProfileID)
		if err != nil {
			writeError(w, err)
			return
		}
		block, err := BlockUser(ctx, pool, userID, showRunID, targetUserID, body.Reason)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"block": block})
	}
}

// HandleUnblock handles DELETE /api/show-runs/{id}/blocks/{block_id}.
func HandleUnblock(pool *pgxpool.Pool) http.HandlerFunc {
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
		if err := UnblockUser(ctx, pool, userID, strings.TrimSpace(r.PathValue("id")), strings.TrimSpace(r.PathValue("block_id"))); err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"unblocked": true})
	}
}
