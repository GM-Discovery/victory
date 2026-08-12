package drawing

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
	"victory/backend/internal/venuecoordination"
)

type response struct {
	Ok   bool `json:"ok"`
	Data any  `json:"data,omitempty"`
}

func ctx10(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 10*time.Second)
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

// notifyChanged pushes a content-free "something changed, refetch"
// invalidation to every client on sessions linked to showID (reusing
// network.BroadcastShowStageInvalidation's existing session-resolution
// path). Deliberately carries no drawing payload: the follow-up GET
// /drawing-objects call each client makes is itself scope-filtered per
// viewer (List), so this is the same "no Cohort leakage" guarantee the
// invalidation event itself gets for free by construction, not something
// this function has to get right a second time.
func notifyChanged(ctx context.Context, hub *network.Hub, pool *pgxpool.Pool, showID string) {
	network.BroadcastShowStageInvalidation(ctx, hub, pool, showID, "drawing_changed")
}

// HandleObjectsCollection handles GET (list, scope-filtered for the
// caller) and POST (create) for /api/sessions/{session_id}/drawing-objects.
func HandleObjectsCollection(pool *pgxpool.Pool, hub *network.Hub, reg *venuecoordination.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := ctx10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		sessionID := strings.TrimSpace(r.PathValue("session_id"))

		switch r.Method {
		case http.MethodGet:
			objs, err := List(ctx, pool, sessionID, userID)
			if err != nil {
				writeError(w, err)
				return
			}
			if objs == nil {
				objs = []Object{}
			}
			writeOK(w, map[string]any{"objects": objs})
		case http.MethodPost:
			var req CreateRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			req.SessionID = sessionID
			obj, err := Create(ctx, pool, reg, userID, req)
			if err != nil {
				writeError(w, err)
				return
			}
			notifyChanged(ctx, hub, pool, obj.ShowID)
			writeOK(w, map[string]any{"object": obj})
		default:
			methodNotAllowed(w)
		}
	}
}

// HandleObjectItem handles PATCH (update) and DELETE for
// /api/sessions/{session_id}/drawing-objects/{object_id}.
func HandleObjectItem(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := ctx10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		sessionID := strings.TrimSpace(r.PathValue("session_id"))
		objectID := strings.TrimSpace(r.PathValue("object_id"))

		switch r.Method {
		case http.MethodPatch:
			var req UpdateRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			req.SessionID = sessionID
			obj, err := Update(ctx, pool, userID, objectID, req)
			if err != nil {
				writeError(w, err)
				return
			}
			notifyChanged(ctx, hub, pool, obj.ShowID)
			writeOK(w, map[string]any{"object": obj})
		case http.MethodDelete:
			obj, err := loadObject(ctx, pool, objectID)
			if err != nil {
				writeError(w, err)
				return
			}
			if err := Delete(ctx, pool, userID, sessionID, objectID); err != nil {
				writeError(w, err)
				return
			}
			notifyChanged(ctx, hub, pool, obj.ShowID)
			writeOK(w, map[string]any{"deleted": true})
		default:
			methodNotAllowed(w)
		}
	}
}

// HandleObjectLock handles POST
// /api/sessions/{session_id}/drawing-objects/{object_id}/lock
// (body: {"locked": true|false}).
func HandleObjectLock(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := ctx10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		sessionID := strings.TrimSpace(r.PathValue("session_id"))
		objectID := strings.TrimSpace(r.PathValue("object_id"))
		var body struct {
			Locked bool `json:"locked"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		obj, err := SetLock(ctx, pool, userID, sessionID, objectID, body.Locked)
		if err != nil {
			writeError(w, err)
			return
		}
		notifyChanged(ctx, hub, pool, obj.ShowID)
		writeOK(w, map[string]any{"object": obj})
	}
}

// HandleObjectZOrder handles POST
// /api/sessions/{session_id}/drawing-objects/{object_id}/z-order
// (body: {"direction": "front"|"back"|"forward"|"backward"}).
func HandleObjectZOrder(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := ctx10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		sessionID := strings.TrimSpace(r.PathValue("session_id"))
		objectID := strings.TrimSpace(r.PathValue("object_id"))
		var body struct {
			Direction string `json:"direction"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		obj, err := Reorder(ctx, pool, userID, sessionID, objectID, body.Direction)
		if err != nil {
			writeError(w, err)
			return
		}
		notifyChanged(ctx, hub, pool, obj.ShowID)
		writeOK(w, map[string]any{"object": obj})
	}
}

// HandleSettings handles GET/PUT for /api/shows/{show_id}/drawing-settings.
// PUT requires session_id in the body -- Director+ authority is checked
// session-scoped, same as every other authority check in this package.
func HandleSettings(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := ctx10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		showID := strings.TrimSpace(r.PathValue("show_id"))

		switch r.Method {
		case http.MethodGet:
			s, err := LoadSettings(ctx, pool, showID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"settings": s})
		case http.MethodPut:
			var body struct {
				SessionID string `json:"session_id"`
				Settings
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			s, err := SaveSettings(ctx, pool, userID, body.SessionID, showID, body.Settings)
			if err != nil {
				writeError(w, err)
				return
			}
			notifyChanged(ctx, hub, pool, showID)
			writeOK(w, map[string]any{"settings": s})
		default:
			methodNotAllowed(w)
		}
	}
}

// HandleStampPalette handles GET /api/drawing/stamps.
func HandleStampPalette() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		type stamp struct {
			Key   string `json:"key"`
			Label string `json:"label"`
		}
		out := make([]stamp, 0, len(ValidStamps))
		for k := range ValidStamps {
			out = append(out, stamp{Key: k, Label: StampLabels[k]})
		}
		writeOK(w, map[string]any{"stamps": out})
	}
}

// HandleCoordination handles GET (current Group Leader/Current Turn) and
// POST (assign) for /api/sessions/{session_id}/drawing-coordination/{role}
// where {role} is "group-leader" or "current-turn".
func HandleCoordination(pool *pgxpool.Pool, reg *venuecoordination.Registry, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := ctx10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		sessionID := strings.TrimSpace(r.PathValue("session_id"))
		role := strings.TrimSpace(r.PathValue("role"))
		showID, err := resolveSessionShowID(ctx, pool, sessionID)
		if err != nil {
			writeError(w, err)
			return
		}

		switch r.Method {
		case http.MethodGet:
			writeOK(w, map[string]any{"coordination": GetCoordination(reg, sessionID, showID)})
		case http.MethodPost:
			var body struct {
				TargetUserID string `json:"target_user_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			var state any
			var assignErr error
			if role == "group-leader" {
				state, assignErr = AssignGroupLeader(ctx, pool, reg, sessionID, showID, userID, body.TargetUserID)
			} else if role == "current-turn" {
				state, assignErr = AssignCurrentTurn(ctx, pool, reg, sessionID, showID, userID, body.TargetUserID)
			} else {
				writeError(w, errors.New("invalid_role"))
				return
			}
			if assignErr != nil {
				writeError(w, assignErr)
				return
			}
			notifyChanged(ctx, hub, pool, showID)
			writeOK(w, map[string]any{"coordination": state})
		default:
			methodNotAllowed(w)
		}
	}
}
