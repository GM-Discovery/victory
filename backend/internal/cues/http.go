package cues

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
	"victory/backend/internal/network"
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
	case code == "cue_execution_in_progress":
		status = http.StatusConflict
	case strings.HasSuffix(code, "_not_found"):
		status = http.StatusNotFound
	}
	writeJSON(w, status, response{Ok: false, Data: map[string]any{"error": code}})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
}

func decodeCueActions(raw []json.RawMessage) ([]CueAction, error) {
	out := make([]CueAction, 0, len(raw))
	for _, r := range raw {
		var a CueAction
		if err := json.Unmarshal(r, &a); err != nil {
			return nil, errors.New("invalid_cue_action")
		}
		out = append(out, a)
	}
	return out, nil
}

// HandlePlacementCuesCollection handles GET
// /api/shows/{show_id}/scenes/{placement_id}/cues (list, backstage-
// visibility-gated) and POST (create, Crew-eligible via CreateCue's own
// authority check).
func HandlePlacementCuesCollection(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		placementID := strings.TrimSpace(r.PathValue("placement_id"))

		switch r.Method {
		case http.MethodGet:
			_, _, locationID, err := placementShowShowRunLocation(ctx, pool, placementID)
			if err != nil {
				writeError(w, err)
				return
			}
			canView, err := showruns.CanViewBackstage(ctx, pool, userID, locationID)
			if err != nil {
				writeError(w, err)
				return
			}
			if !canView {
				writeError(w, errors.New("not_authorized"))
				return
			}
			list, err := ListCuesForPlacement(ctx, pool, placementID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"cues": list})

		case http.MethodPost:
			var body struct {
				InternalName     string            `json:"internal_name"`
				StageButtonLabel string            `json:"stage_button_label"`
				TriggerScope     string            `json:"trigger_scope"`
				SortOrder        int               `json:"sort_order"`
				Enabled          *bool             `json:"enabled"`
				Actions          []json.RawMessage `json:"actions"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			decoded, err := decodeCueActions(body.Actions)
			if err != nil {
				writeError(w, err)
				return
			}
			c, err := CreateCue(ctx, pool, userID, placementID, CreateCueInput{
				InternalName: body.InternalName, StageButtonLabel: body.StageButtonLabel,
				TriggerScope: body.TriggerScope, SortOrder: body.SortOrder,
				Enabled: body.Enabled, Actions: decoded,
			})
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"cue": c})

		default:
			methodNotAllowed(w)
		}
	}
}

// HandlePlacementPlayerCues handles GET
// /api/shows/{show_id}/scenes/{placement_id}/player-cues -- the curated
// player-facing stage button listing (Kernel 70 §6.2, §8.2). Unlike
// HandlePlacementCuesCollection's GET, this requires no backstage
// authority -- any authenticated user may call it, and
// ListTriggerableCuesForViewer's own per-Cue CanTriggerCue check (which
// hard-excludes Audience) determines what comes back, including
// potentially nothing.
func HandlePlacementPlayerCues(pool *pgxpool.Pool) http.HandlerFunc {
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
		list, err := ListTriggerableCuesForViewer(ctx, pool, userID, placementID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"cues": list})
	}
}

// HandleCueByID handles GET /api/cues/{cue_id} (backstage detail) and
// PATCH (update, Crew-eligible via UpdateCue's own authority check).
func HandleCueByID(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		cueID := strings.TrimSpace(r.PathValue("cue_id"))

		switch r.Method {
		case http.MethodGet:
			c, err := LoadCueByID(ctx, pool, cueID)
			if err != nil {
				writeError(w, err)
				return
			}
			_, _, locationID, err := placementShowShowRunLocation(ctx, pool, c.ShowScenePlacementID)
			if err != nil {
				writeError(w, err)
				return
			}
			canView, err := showruns.CanViewBackstage(ctx, pool, userID, locationID)
			if err != nil {
				writeError(w, err)
				return
			}
			if !canView {
				writeError(w, errors.New("not_authorized"))
				return
			}
			writeOK(w, map[string]any{"cue": c})

		case http.MethodPatch:
			var body struct {
				InternalName     *string           `json:"internal_name"`
				StageButtonLabel *string           `json:"stage_button_label"`
				TriggerScope     *string           `json:"trigger_scope"`
				SortOrder        *int              `json:"sort_order"`
				Enabled          *bool             `json:"enabled"`
				Actions          []json.RawMessage `json:"actions"`
			}
			raw, err := readAndPeekBody(r)
			if err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			if err := json.Unmarshal(raw, &body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			var actionsPtr *[]CueAction
			if bodyHasField(raw, "actions") {
				decoded, err := decodeCueActions(body.Actions)
				if err != nil {
					writeError(w, err)
					return
				}
				actionsPtr = &decoded
			}
			c, err := UpdateCue(ctx, pool, userID, cueID, UpdateCuePatch{
				InternalName: body.InternalName, StageButtonLabel: body.StageButtonLabel,
				TriggerScope: body.TriggerScope, SortOrder: body.SortOrder,
				Enabled: body.Enabled, Actions: actionsPtr,
			})
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"cue": c})

		default:
			methodNotAllowed(w)
		}
	}
}

// HandleCueGo handles POST /api/cues/{cue_id}/go -- the GO press. Body:
// {"idempotency_key": "..."}. Never reachable by Audience, enforced inside
// ExecuteCue's CanTriggerCue check. On success, broadcasts a show-scoped
// stage invalidation so connected clients refetch (Kernel 70 §6.4's
// "broadcast one authoritative invalidation/update after execution").
func HandleCueGo(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		cueID := strings.TrimSpace(r.PathValue("cue_id"))

		var body struct {
			IdempotencyKey string `json:"idempotency_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		result, err := ExecuteCue(ctx, pool, userID, cueID, body.IdempotencyKey)
		if err != nil {
			writeError(w, err)
			return
		}
		network.BroadcastShowStageInvalidation(ctx, hub, pool, result.ShowID, "cue_executed")
		writeOK(w, map[string]any{"execution": result})
	}
}

func readAndPeekBody(r *http.Request) ([]byte, error) {
	return io.ReadAll(r.Body)
}

func bodyHasField(raw []byte, field string) bool {
	var generic map[string]json.RawMessage
	if err := json.Unmarshal(raw, &generic); err != nil {
		return false
	}
	_, ok := generic[field]
	return ok
}
