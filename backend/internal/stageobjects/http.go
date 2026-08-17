package stageobjects

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

// DirectorGate authorizes an actor as Director+ for a Show.
//
// Injected rather than implemented here for a hard structural reason: the
// canonical Director+ check lives in showruns.CanManageShowRun, but
// world/snapshot.go imports THIS package for projection, and shows -> network
// -> world means importing shows here would close a real cycle (the same one
// world/snapshot.go's own comment documents about scenes). Injection keeps
// this package a leaf and still leaves exactly one opinion in the product
// about what Director+ means -- main.go wires directorprep.RequireDirector,
// which is that opinion.
type DirectorGate func(ctx context.Context, actorUserID, showID string) error

// Notifier publishes a projection invalidation to every viewer of a Show.
// Injected for the same cycle reason (network imports world imports this).
// main.go wires network.BroadcastShowStageInvalidation.
type Notifier func(ctx context.Context, showID string)

type response struct {
	Ok   bool `json:"ok"`
	Data any  `json:"data,omitempty"`
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
	case errors.Is(err, ErrObjectNotFound):
		status = http.StatusNotFound
	case errors.Is(err, ErrObjectNotSupported):
		status = http.StatusUnprocessableEntity
	case strings.HasSuffix(code, "_not_found"):
		status = http.StatusNotFound
	}
	writeJSON(w, status, response{Ok: false, Data: map[string]any{"error": code}})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
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

// HandleShowObjectStates handles GET and POST
// /api/shows/{show_id}/stage-object-states.
//
// BOTH methods are Director+ only, including the read. The state set is a map
// of what is hidden and from whom, so serving it to a Player would hand them
// precisely the metadata §35 and §36 require be withheld -- the object
// payloads are already omitted from their snapshot, and this endpoint must
// not become the back door that tells them what they are missing.
func HandleShowObjectStates(pool *pgxpool.Pool, gate DirectorGate, notify Notifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		showID := strings.TrimSpace(r.PathValue("show_id"))
		if err := gate(ctx, userID, showID); err != nil {
			writeError(w, err)
			return
		}

		switch r.Method {
		case http.MethodGet:
			states, err := LoadShowStates(ctx, pool, showID)
			if err != nil {
				writeError(w, err)
				return
			}
			// Serialized as a list rather than a map because a Ref is a
			// two-part key with no safe string spelling that a client could
			// not accidentally re-parse into identity.
			out := make([]State, 0, len(states))
			for _, st := range states {
				out = append(out, st)
			}
			writeOK(w, map[string]any{"states": out})

		case http.MethodPost:
			var req mutationRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			m := Mutation{
				ShowID:  showID,
				Ref:     Ref{Kind: req.ObjectKind, ID: req.ObjectID},
				Op:      req.Operation,
				ActorID: userID,
			}
			// A nil Scopes field means "leave grants alone"; an explicit
			// (even empty) array means "replace them with exactly this".
			// json.RawMessage lets those two be distinguished, which a
			// []Scope alone could not.
			if req.Scopes != nil {
				var scopes []Scope
				if err := json.Unmarshal(req.Scopes, &scopes); err != nil {
					writeError(w, errors.New("invalid_scopes"))
					return
				}
				if scopes == nil {
					scopes = []Scope{}
				}
				m = m.WithScopes(scopes)
			}

			state, err := ApplyMutation(ctx, pool, m)
			if err != nil {
				writeError(w, err)
				return
			}
			if notify != nil {
				notify(ctx, showID)
			}
			writeOK(w, map[string]any{"state": state})

		default:
			methodNotAllowed(w)
		}
	}
}

type mutationRequest struct {
	ObjectKind string          `json:"object_kind"`
	ObjectID   string          `json:"object_id"`
	Operation  string          `json:"operation"`
	Scopes     json.RawMessage `json:"scopes,omitempty"`
}

// HandleSupportedScopeTargets handles GET
// /api/shows/{show_id}/stage-object-scope-targets.
//
// Serves the Cohorts and roster Characters a Director may scope to, so the
// scope picker offers real choices instead of asking a Director to paste a
// UUID. Director+ only: the roster and cohort layout of a Show is not
// Player-facing information.
func HandleSupportedScopeTargets(pool *pgxpool.Pool, gate DirectorGate) http.HandlerFunc {
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
		if err := gate(ctx, userID, showID); err != nil {
			writeError(w, err)
			return
		}

		cohorts, err := listCohortTargets(ctx, pool, showID)
		if err != nil {
			writeError(w, err)
			return
		}
		characters, err := listCharacterTargets(ctx, pool, showID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{
			"cohorts":    cohorts,
			"characters": characters,
			// The two tier scopes need no ids, but the client should not
			// hardcode their names either.
			"tiers": []map[string]string{
				{"scope_kind": ScopeCast, "label": "Cast (players, not the house)"},
				{"scope_kind": ScopeAudience, "label": "Audience"},
			},
		})
	}
}

type scopeTarget struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

func listCohortTargets(ctx context.Context, q Querier, showID string) ([]scopeTarget, error) {
	rows, err := q.Query(ctx, `
		SELECT id::text, COALESCE(NULLIF(name, ''), 'Cohort')
		FROM show_cohorts
		WHERE show_id = $1::uuid AND archived_at IS NULL
		ORDER BY created_at ASC
	`, showID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []scopeTarget{}
	for rows.Next() {
		var t scopeTarget
		if err := rows.Scan(&t.ID, &t.Label); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func listCharacterTargets(ctx context.Context, q Querier, showID string) ([]scopeTarget, error) {
	rows, err := q.Query(ctx, `
		SELECT cc.id::text, COALESCE(NULLIF(cc.name, ''), 'Character')
		FROM show_run_roster_members srrm
		JOIN shows sh ON sh.show_run_id = srrm.show_run_id
		JOIN character_cards cc ON cc.id = srrm.character_card_id
		WHERE sh.id = $1::uuid
		  AND srrm.removed_at IS NULL
		  AND srrm.character_card_id IS NOT NULL
		  AND cc.is_deleted = FALSE
		ORDER BY cc.name ASC
	`, showID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []scopeTarget{}
	for rows.Next() {
		var t scopeTarget
		if err := rows.Scan(&t.ID, &t.Label); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
