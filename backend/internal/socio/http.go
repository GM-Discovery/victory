package socio

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
	"victory/backend/internal/venuecoordination"
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
	case code == "character_not_on_show":
		status = http.StatusConflict
	}
	writeJSON(w, status, response{Ok: false, Data: map[string]any{"error": code}})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
}

// HandleStatusRegistry handles GET /api/socio/statuses.
func HandleStatusRegistry(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		if _, err := requireAuthenticatedUser(ctx, pool, r); err != nil {
			writeError(w, err)
			return
		}
		defs, err := ListStatusDefinitions(ctx, pool)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"statuses": defs})
	}
}

// HandleGameStatus handles GET
// /api/shows/{show_id}/cohorts/{cohort_id}/game-status ({cohort_id} may be
// the literal "ungrouped").
func HandleGameStatus(pool *pgxpool.Pool) http.HandlerFunc {
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
		cohortID := strings.TrimSpace(r.PathValue("cohort_id"))
		blocks, err := BuildGameStatusForCohort(ctx, pool, userID, showID, cohortID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"characters": blocks})
	}
}

// HandleCharacterPool handles POST
// /api/shows/{show_id}/characters/{character_card_id}/socio/pools/{pool_key}
// (body: {"current": N, "max": N}).
func HandleCharacterPool(pool *pgxpool.Pool) http.HandlerFunc {
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
		characterCardID := strings.TrimSpace(r.PathValue("character_card_id"))
		poolKey := strings.TrimSpace(r.PathValue("pool_key"))

		var body struct {
			Current int `json:"current"`
			Max     int `json:"max"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		state, err := SetPool(ctx, pool, userID, showID, characterCardID, PoolKey(poolKey), body.Current, body.Max)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"state": state})
	}
}

// HandleCharacterStatus handles POST
// /api/shows/{show_id}/characters/{character_card_id}/socio/statuses (body:
// {"status_key": "...", "intensity": N}) and DELETE
// /api/shows/{show_id}/characters/{character_card_id}/socio/statuses/{status_key}.
func HandleCharacterStatus(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		showID := strings.TrimSpace(r.PathValue("show_id"))
		characterCardID := strings.TrimSpace(r.PathValue("character_card_id"))

		switch r.Method {
		case http.MethodPost:
			var body struct {
				StatusKey string `json:"status_key"`
				Intensity *int   `json:"intensity"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			applied, err := ApplyStatus(ctx, pool, userID, showID, characterCardID, body.StatusKey, body.Intensity)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"status": applied})
		case http.MethodDelete:
			statusKey := strings.TrimSpace(r.PathValue("status_key"))
			if err := ClearStatus(ctx, pool, userID, showID, characterCardID, statusKey); err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"cleared": true})
		default:
			methodNotAllowed(w)
		}
	}
}

// HandleSocioProjection handles GET
// /api/shows/{show_id}/characters/{character_card_id}/socio/view -- the one
// tier-shaped read path (kernel-88 §24): owning Player or Director+ only,
// enforced inside ProjectSocioState.
func HandleSocioProjection(pool *pgxpool.Pool) http.HandlerFunc {
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
		characterCardID := strings.TrimSpace(r.PathValue("character_card_id"))
		proj, err := ProjectSocioState(ctx, pool, userID, showID, characterCardID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"projection": proj})
	}
}

// HandleCharacterMechanics handles GET
// /api/shows/{show_id}/characters/{character_card_id}/socio/mechanics -- the
// Player HUD's rollable-mechanic list, resolved from the same Show-Run roster
// Character the roll path itself uses (see mechanics.go).
func HandleCharacterMechanics(pool *pgxpool.Pool) http.HandlerFunc {
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
		characterCardID := strings.TrimSpace(r.PathValue("character_card_id"))
		mechanics, tier, err := ListCharacterMechanics(ctx, pool, userID, showID, characterCardID)
		if err != nil {
			writeError(w, err)
			return
		}
		// Best-effort: the name is a nicety, never a reason to fail the list.
		name, _ := CharacterDisplayName(ctx, pool, characterCardID)
		writeOK(w, map[string]any{"mechanics": mechanics, "tier": tier, "character_name": name})
	}
}

// HandleFate handles POST
// /api/shows/{show_id}/characters/{character_card_id}/socio/fate/spend
// (body {"amount": N, "note": ""}).
func HandleFateSpend(pool *pgxpool.Pool) http.HandlerFunc {
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
		characterCardID := strings.TrimSpace(r.PathValue("character_card_id"))
		var body struct {
			Amount int    `json:"amount"`
			Note   string `json:"note"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		state, err := SpendFate(ctx, pool, userID, showID, characterCardID, body.Amount, body.Note)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"fate": state})
	}
}

// HandleFateAward handles POST
// /api/shows/{show_id}/characters/{character_card_id}/socio/fate/award
// (body {"delta": N, "reason": "", "note": ""}). Director+ only.
func HandleFateAward(pool *pgxpool.Pool) http.HandlerFunc {
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
		characterCardID := strings.TrimSpace(r.PathValue("character_card_id"))
		var body struct {
			Delta  int    `json:"delta"`
			Reason string `json:"reason"`
			Note   string `json:"note"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		state, err := AwardFate(ctx, pool, userID, showID, characterCardID, body.Delta, body.Reason, body.Note)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"fate": state})
	}
}

// HandleFateCreationMode handles POST
// /api/shows/{show_id}/characters/{character_card_id}/socio/fate/creation-mode
// (body {"on": true}). Director+ only; turning it off is the clamp-to-7
// point (see fate.go's SetCreationMode).
func HandleFateCreationMode(pool *pgxpool.Pool) http.HandlerFunc {
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
		characterCardID := strings.TrimSpace(r.PathValue("character_card_id"))
		var body struct {
			On bool `json:"on"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		state, err := SetCreationMode(ctx, pool, userID, showID, characterCardID, body.On)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"fate": state})
	}
}

// HandleStanceRegistry handles GET /api/socio/stances.
func HandleStanceRegistry(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		if _, err := requireAuthenticatedUser(ctx, pool, r); err != nil {
			writeError(w, err)
			return
		}
		defs, err := ListStanceDefinitions(ctx, pool)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"stances": defs})
	}
}

// HandleCharacterStance handles POST
// /api/shows/{show_id}/characters/{character_card_id}/socio/stance (body
// {"stance_key": ""}). Owning Player or Director+.
func HandleCharacterStance(pool *pgxpool.Pool) http.HandlerFunc {
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
		characterCardID := strings.TrimSpace(r.PathValue("character_card_id"))
		var body struct {
			StanceKey string `json:"stance_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		stance, err := SetStance(ctx, pool, userID, showID, characterCardID, body.StanceKey)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"stance": stance})
	}
}

// HandleCharacterFlags handles POST
// /api/shows/{show_id}/characters/{character_card_id}/socio/flags (body
// {"label": ""}) and DELETE .../socio/flags/{flag_id}. Director+ only --
// Kernel 88's blank state fields, distinct from canonical statuses
// (HandleCharacterStatus above).
func HandleCharacterFlags(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		showID := strings.TrimSpace(r.PathValue("show_id"))
		characterCardID := strings.TrimSpace(r.PathValue("character_card_id"))

		switch r.Method {
		case http.MethodPost:
			var body struct {
				Label string `json:"label"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			flag, err := AddFlag(ctx, pool, userID, showID, characterCardID, body.Label)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"flag": flag})
		case http.MethodDelete:
			flagID := strings.TrimSpace(r.PathValue("flag_id"))
			if err := ClearFlag(ctx, pool, userID, showID, characterCardID, flagID); err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"cleared": true})
		default:
			methodNotAllowed(w)
		}
	}
}

// HandleCurrentTurn handles GET/POST
// /api/shows/{show_id}/cohorts/{cohort_id}/socio/current-turn (POST body
// {"target_user_id": ""}).
func HandleCurrentTurn(pool *pgxpool.Pool, reg *venuecoordination.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		showID := strings.TrimSpace(r.PathValue("show_id"))
		cohortID := strings.TrimSpace(r.PathValue("cohort_id"))

		switch r.Method {
		case http.MethodGet:
			view, err := BuildCoordinationView(ctx, pool, reg, showID, cohortID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"coordination": view})
		case http.MethodPost:
			var body struct {
				TargetUserID string `json:"target_user_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			if _, err := AssignCurrentTurn(ctx, pool, reg, userID, showID, cohortID, body.TargetUserID); err != nil {
				writeError(w, err)
				return
			}
			view, err := BuildCoordinationView(ctx, pool, reg, showID, cohortID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"coordination": view})
		default:
			methodNotAllowed(w)
		}
	}
}

// HandlePendingActions handles GET
// /api/shows/{show_id}/socio/pending-actions (the open stack) and POST
// (open a primary action, body {"session_id","cohort_id","character_card_id","title"}).
func HandlePendingActions(pool *pgxpool.Pool, reg *venuecoordination.Registry) http.HandlerFunc {
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
			stack, err := ListOpenStack(ctx, pool, showID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"pending_actions": stack})
		case http.MethodPost:
			var body struct {
				SessionID       string `json:"session_id"`
				CohortID        string `json:"cohort_id"`
				CharacterCardID string `json:"character_card_id"`
				Title           string `json:"title"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			action, err := OpenPrimaryAction(ctx, pool, reg, userID, showID, body.SessionID, body.CohortID, body.CharacterCardID, body.Title)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"pending_action": action})
		default:
			methodNotAllowed(w)
		}
	}
}

// HandleInterrupt handles POST
// /api/shows/{show_id}/socio/pending-actions/{action_id}/interrupt (body
// {"session_id","helper_character_card_id","title","target_complexity"}).
// Director+ only.
func HandleInterrupt(pool *pgxpool.Pool) http.HandlerFunc {
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
		parentID := strings.TrimSpace(r.PathValue("action_id"))
		var body struct {
			SessionID             string `json:"session_id"`
			HelperCharacterCardID string `json:"helper_character_card_id"`
			Title                 string `json:"title"`
			TargetComplexity      int    `json:"target_complexity"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		action, err := OpenInterrupt(ctx, pool, userID, showID, body.SessionID, parentID, body.HelperCharacterCardID, body.Title, body.TargetComplexity)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"pending_action": action})
	}
}

// HandleInterruptResolve handles POST
// /api/shows/{show_id}/socio/pending-actions/{action_id}/resolve (body
// {"roll_action_id","roll_total"}) -- called after the helper's own roll
// (StorePlayerMechanicRoll, or a Director-triggered roll) has already been
// stored elsewhere; this attaches that result to the interrupt and computes
// overage. Authority: Director+ or the helper's own owning Player (see
// ResolveInterrupt).
func HandleInterruptResolve(pool *pgxpool.Pool) http.HandlerFunc {
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
		actionID := strings.TrimSpace(r.PathValue("action_id"))
		var body struct {
			RollActionID string `json:"roll_action_id"`
			RollTotal    int    `json:"roll_total"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		action, err := ResolveInterrupt(ctx, pool, userID, showID, actionID, body.RollActionID, body.RollTotal)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"pending_action": action})
	}
}

// HandlePendingActionCancel handles POST
// /api/shows/{show_id}/socio/pending-actions/{action_id}/cancel. Director+
// only.
func HandlePendingActionCancel(pool *pgxpool.Pool) http.HandlerFunc {
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
		actionID := strings.TrimSpace(r.PathValue("action_id"))
		if err := CancelPendingAction(ctx, pool, userID, showID, actionID); err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"cancelled": true})
	}
}
