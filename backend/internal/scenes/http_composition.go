package scenes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showruns"
)

type stageElementBody struct {
	Kind       string         `json:"kind"`
	Label      string         `json:"label"`
	Data       map[string]any `json:"data"`
	Position   map[string]any `json:"position"`
	Visibility map[string]any `json:"visibility"`
	SortOrder  int            `json:"sort_order"`
	// Kernel 74 normalized 0-1 size. Pointers so "absent" and "zero" stay
	// distinguishable -- an omitted size must leave the stored value alone,
	// not shrink the element to nothing.
	Width  *float64 `json:"width"`
	Height *float64 `json:"height"`
}

// HandleSceneStageElementsCollection handles GET /api/scenes/{scene_id}/stage-elements
// (Base-layer composition listing) and POST (add a Base-layer element).
func HandleSceneStageElementsCollection(pool *pgxpool.Pool) http.HandlerFunc {
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
			canView, err := CanViewScenesForLocation(ctx, pool, userID, s.LocationID)
			if err != nil {
				writeError(w, err)
				return
			}
			if !canView {
				writeError(w, errors.New("not_authorized"))
				return
			}
			list, err := ListSceneStageElements(ctx, pool, sceneID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"elements": list})

		case http.MethodPost:
			var body stageElementBody
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			e, err := CreateSceneStageElement(ctx, pool, userID, sceneID, CreateStageElementInput{
				Kind: body.Kind, Label: body.Label, Data: body.Data,
				Position: body.Position, Visibility: body.Visibility, SortOrder: body.SortOrder,
				Width: body.Width, Height: body.Height,
			})
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"element": e})

		default:
			methodNotAllowed(w)
		}
	}
}

// HandlePlacementStageElementsCollection handles POST
// /api/shows/{show_id}/scenes/{placement_id}/stage-elements -- add a
// Show-layer element scoped to this placement only.
func HandlePlacementStageElementsCollection(pool *pgxpool.Pool) http.HandlerFunc {
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
		placementID := strings.TrimSpace(r.PathValue("placement_id"))

		var body stageElementBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		e, err := CreatePlacementStageElement(ctx, pool, userID, placementID, CreateStageElementInput{
			Kind: body.Kind, Label: body.Label, Data: body.Data,
			Position: body.Position, Visibility: body.Visibility, SortOrder: body.SortOrder,
			Width: body.Width, Height: body.Height,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"element": e})
	}
}

// HandleStageElementByID handles PATCH and DELETE /api/stage-elements/{element_id}.
func HandleStageElementByID(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		elementID := strings.TrimSpace(r.PathValue("element_id"))

		switch r.Method {
		case http.MethodPatch:
			var body struct {
				Label      *string         `json:"label"`
				Data       *map[string]any `json:"data"`
				Position   *map[string]any `json:"position"`
				Visibility *map[string]any `json:"visibility"`
				SortOrder  *int            `json:"sort_order"`
				Width      *float64        `json:"width"`
				Height     *float64        `json:"height"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			e, err := UpdateStageElement(ctx, pool, userID, elementID, UpdateStageElementPatch{
				Label: body.Label, Data: body.Data, Position: body.Position,
				Visibility: body.Visibility, SortOrder: body.SortOrder,
				Width: body.Width, Height: body.Height,
			})
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"element": e})

		case http.MethodDelete:
			if err := DeleteStageElement(ctx, pool, userID, elementID); err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"deleted": true})

		default:
			methodNotAllowed(w)
		}
	}
}

// HandleStageElementBinding handles POST (bind) and DELETE (unbind)
// /api/stage-elements/{element_id}/binding.
func HandleStageElementBinding(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		elementID := strings.TrimSpace(r.PathValue("element_id"))

		switch r.Method {
		case http.MethodPost:
			var body struct {
				ParticipantInteractionID string `json:"participant_interaction_id"`
				RequiresMilestone        string `json:"requires_milestone"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			b, err := CreateStageElementBinding(ctx, pool, userID, elementID, body.ParticipantInteractionID, body.RequiresMilestone)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"binding": b})

		case http.MethodDelete:
			if err := DeleteStageElementBinding(ctx, pool, userID, elementID); err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"deleted": true})

		default:
			methodNotAllowed(w)
		}
	}
}

// HandlePlacementStageComposition handles GET
// /api/shows/{show_id}/scenes/{placement_id}/stage-composition -- the
// merged Base+Show layer resolved composition, used by the Scene Setup
// editor's Preview as Player mode. There is no POST/"load into session"
// variant any more -- backend/internal/world/snapshot.go's
// LoadVenueSnapshot resolves this same composition fresh on every real
// snapshot read once a placement is the Show's current Scene, so there is
// nothing to separately "load."
func HandlePlacementStageComposition(pool *pgxpool.Pool) http.HandlerFunc {
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
		placementID := strings.TrimSpace(r.PathValue("placement_id"))

		_, sr, err := showRunForShow(ctx, pool, strings.TrimSpace(r.PathValue("show_id")))
		if err != nil {
			writeError(w, err)
			return
		}
		canView, err := showruns.CanViewBackstage(ctx, pool, userID, sr.LocationID)
		if err != nil {
			writeError(w, err)
			return
		}
		if !canView {
			writeError(w, errors.New("not_authorized"))
			return
		}
		composition, err := LoadResolvedComposition(ctx, pool, placementID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"composition": composition})
	}
}
