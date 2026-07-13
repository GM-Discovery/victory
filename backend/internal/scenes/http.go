package scenes

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
	case code == "not_authorized":
		status = http.StatusForbidden
	case strings.HasSuffix(code, "_not_found"):
		status = http.StatusNotFound
	}
	writeJSON(w, status, response{Ok: false, Data: map[string]any{"error": code}})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
}

// HandleScenesCollection handles GET /api/scenes?production_id=... (list,
// backstage-visibility-gated) and POST /api/scenes (create, manage-
// authorized inside CreateScene).
func HandleScenesCollection(pool *pgxpool.Pool) http.HandlerFunc {
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
			productionID := strings.TrimSpace(r.URL.Query().Get("production_id"))
			if productionID == "" {
				writeError(w, errors.New("production_id_required"))
				return
			}
			canView, err := CanViewScenesBackstage(ctx, pool, userID, productionID)
			if err != nil {
				writeError(w, err)
				return
			}
			if !canView {
				writeError(w, errors.New("not_authorized"))
				return
			}
			list, err := ListScenesForProduction(ctx, pool, productionID)
			if err != nil {
				writeError(w, err)
				return
			}
			canManage, err := CanManageScenesForProduction(ctx, pool, userID, productionID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"scenes": list, "can_manage": canManage})

		case http.MethodPost:
			var body struct {
				ProductionID    string `json:"production_id"`
				Slug            string `json:"slug"`
				Title           string `json:"title"`
				ShortTitle      string `json:"short_title"`
				DefaultVenueID  string `json:"default_venue_id"`
				AudienceTitle   string `json:"audience_title"`
				AudienceSummary string `json:"audience_summary"`
				PlayerBrief     string `json:"player_brief"`
				DirectorNotes   string `json:"director_notes"`
				OperatorNotes   string `json:"operator_notes"`
				SourceRef       string `json:"source_ref"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			s, err := CreateScene(ctx, pool, userID, body.ProductionID, CreateSceneInput{
				Slug: body.Slug, Title: body.Title, ShortTitle: body.ShortTitle,
				DefaultVenueID: body.DefaultVenueID, AudienceTitle: body.AudienceTitle,
				AudienceSummary: body.AudienceSummary, PlayerBrief: body.PlayerBrief,
				DirectorNotes: body.DirectorNotes, OperatorNotes: body.OperatorNotes,
				SourceRef: body.SourceRef,
			})
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"scene": s})

		default:
			methodNotAllowed(w)
		}
	}
}

// HandleSceneByID handles GET /api/scenes/{scene_id} (backstage detail) and
// PATCH (update, manage-authorized inside UpdateScene).
func HandleSceneByID(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		sceneID := strings.TrimSpace(r.PathValue("scene_id"))

		switch r.Method {
		case http.MethodGet:
			s, err := LoadSceneByID(ctx, pool, sceneID)
			if err != nil {
				writeError(w, err)
				return
			}
			canView, err := CanViewScenesBackstage(ctx, pool, userID, s.ProductionID)
			if err != nil {
				writeError(w, err)
				return
			}
			if !canView {
				writeError(w, errors.New("not_authorized"))
				return
			}
			canManage, err := CanManageScenesForProduction(ctx, pool, userID, s.ProductionID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"scene": s, "can_manage": canManage})

		case http.MethodPatch:
			var body struct {
				Title           *string `json:"title"`
				ShortTitle      *string `json:"short_title"`
				DefaultVenueID  *string `json:"default_venue_id"`
				AudienceTitle   *string `json:"audience_title"`
				AudienceSummary *string `json:"audience_summary"`
				PlayerBrief     *string `json:"player_brief"`
				DirectorNotes   *string `json:"director_notes"`
				OperatorNotes   *string `json:"operator_notes"`
				SourceRef       *string `json:"source_ref"`
				Status          *string `json:"status"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			s, err := UpdateScene(ctx, pool, userID, sceneID, UpdateScenePatch{
				Title: body.Title, ShortTitle: body.ShortTitle, DefaultVenueID: body.DefaultVenueID,
				AudienceTitle: body.AudienceTitle, AudienceSummary: body.AudienceSummary,
				PlayerBrief: body.PlayerBrief, DirectorNotes: body.DirectorNotes,
				OperatorNotes: body.OperatorNotes, SourceRef: body.SourceRef, Status: body.Status,
			})
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"scene": s})

		default:
			methodNotAllowed(w)
		}
	}
}

// HandleSceneArchive handles POST /api/scenes/{scene_id}/archive.
func HandleSceneArchive(pool *pgxpool.Pool) http.HandlerFunc {
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
		s, err := ArchiveScene(ctx, pool, userID, strings.TrimSpace(r.PathValue("scene_id")))
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"scene": s})
	}
}

// HandleShowScenesCollection handles GET /api/shows/{show_id}/scenes
// (backstage listing of Scenes staged in this Show) and POST (add an
// existing reusable Scene to this Show).
func HandleShowScenesCollection(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		showID := strings.TrimSpace(r.PathValue("show_id"))

		_, sr, err := showRunForShow(ctx, pool, showID)
		if err != nil {
			writeError(w, err)
			return
		}

		switch r.Method {
		case http.MethodGet:
			canView, err := showruns.CanViewBackstage(ctx, pool, userID, sr.LocationID)
			if err != nil {
				writeError(w, err)
				return
			}
			if !canView {
				writeError(w, errors.New("not_authorized"))
				return
			}
			list, err := ListPlacementsForShow(ctx, pool, showID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"placements": list})

		case http.MethodPost:
			var body struct {
				SceneID                 string `json:"scene_id"`
				VenueID                 string `json:"venue_id"`
				SortOrder               int    `json:"sort_order"`
				AudienceTitleOverride   string `json:"audience_title_override"`
				AudienceSummaryOverride string `json:"audience_summary_override"`
				DirectorNotesOverride   string `json:"director_notes_override"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			p, err := CreatePlacement(ctx, pool, userID, showID, CreatePlacementInput{
				SceneID: body.SceneID, VenueID: body.VenueID, SortOrder: body.SortOrder,
				AudienceTitleOverride:   body.AudienceTitleOverride,
				AudienceSummaryOverride: body.AudienceSummaryOverride,
				DirectorNotesOverride:   body.DirectorNotesOverride,
			})
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"placement": p})

		default:
			methodNotAllowed(w)
		}
	}
}

// HandleShowScenePlacementByID handles PATCH
// /api/shows/{show_id}/scenes/{placement_id}.
func HandleShowScenePlacementByID(pool *pgxpool.Pool) http.HandlerFunc {
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
		placementID := strings.TrimSpace(r.PathValue("placement_id"))

		var body struct {
			VenueID                 *string `json:"venue_id"`
			SortOrder               *int    `json:"sort_order"`
			Status                  *string `json:"status"`
			AudienceTitleOverride   *string `json:"audience_title_override"`
			AudienceSummaryOverride *string `json:"audience_summary_override"`
			DirectorNotesOverride   *string `json:"director_notes_override"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		p, err := UpdatePlacement(ctx, pool, userID, placementID, UpdatePlacementPatch{
			VenueID: body.VenueID, SortOrder: body.SortOrder, Status: body.Status,
			AudienceTitleOverride:   body.AudienceTitleOverride,
			AudienceSummaryOverride: body.AudienceSummaryOverride,
			DirectorNotesOverride:   body.DirectorNotesOverride,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"placement": p})
	}
}

// HandleShowScenePlacementArchive handles POST
// /api/shows/{show_id}/scenes/{placement_id}/archive -- "Remove Scene from
// Show" in the UI. Never archives or deletes the underlying reusable Scene.
func HandleShowScenePlacementArchive(pool *pgxpool.Pool) http.HandlerFunc {
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
		p, err := ArchivePlacement(ctx, pool, userID, strings.TrimSpace(r.PathValue("placement_id")))
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"placement": p})
	}
}

// HandleShowSceneProgram handles GET /api/shows/{show_id}/scenes/program --
// the curated Audience Program addition for this Show's Scenes (Kernel 69
// SS2.6). Available to any viewer who can view the Show's Audience Program
// at all (showruns.CanViewShowRun), not just backstage managers.
func HandleShowSceneProgram(pool *pgxpool.Pool) http.HandlerFunc {
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

		_, sr, err := showRunForShow(ctx, pool, showID)
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
		entries, err := ListAudiencePlacementsForShow(ctx, pool, showID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"scenes": entries})
	}
}
