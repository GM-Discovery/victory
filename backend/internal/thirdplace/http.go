package thirdplace

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
	"victory/backend/internal/playerprofile"
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

// requireCommonsAdmission enforces Third Place's admission rule: a viewer must
// have a ready Trailer Face of their own before they can see anyone else's.
// This is the same check that gates placing a Headshot (Kernel 68 §3.2).
func requireCommonsAdmission(ctx context.Context, pool *pgxpool.Pool, userID string) error {
	ready, err := playerprofile.TrailerFaceReady(ctx, pool, userID)
	if err != nil {
		return err
	}
	if !ready.Ready {
		return errors.New("trailer_face_not_ready")
	}
	return nil
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
	case code == "trailer_face_not_ready":
		status = http.StatusForbidden
	case strings.HasSuffix(code, "_not_found"):
		status = http.StatusNotFound
	}
	writeJSON(w, status, response{Ok: false, Data: map[string]any{"error": code}})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
}

// HandleCollection handles GET /api/third-place/headshots -- the Headshot
// Commons list, active Headshots only, most recently placed first
// (Kernel 65 §7). Anonymous requests are rejected before anything is read.
func HandleCollection(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		viewerUserID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}

		// Kernel 76 (K76-H02): being signed in used to be the whole gate here,
		// so any account -- including one a stranger had just created -- could
		// read every member's stage name and headline facts. Reading the room
		// now takes the same Trailer Face readiness that entering it does.
		if err := requireCommonsAdmission(ctx, pool, viewerUserID); err != nil {
			writeError(w, err)
			return
		}

		rows, err := ListActiveHeadshots(ctx, pool)
		if err != nil {
			writeError(w, err)
			return
		}

		projections := make([]HeadshotProjection, 0, len(rows))
		for _, row := range rows {
			proj, err := ProjectHeadshot(ctx, pool, viewerUserID, row)
			if err != nil {
				// A single broken/legacy workbook shouldn't 500 the whole
				// commons for every other viewer -- skip it, not fail the list.
				continue
			}
			projections = append(projections, proj)
		}

		writeOK(w, map[string]any{"headshots": projections})
	}
}

// HandleMe handles /api/third-place/headshots/me (Kernel 65 §7): GET returns
// the caller's own active Headshot or null, POST leaves/refreshes it, DELETE
// removes it. All three always act on the session-derived user; there is no
// request field that could name a different account (Kernel 65 §9.2).
func HandleMe(pool *pgxpool.Pool) http.HandlerFunc {
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
			h, err := GetMyHeadshot(ctx, pool, userID)
			if err != nil {
				writeError(w, err)
				return
			}
			if h == nil {
				writeOK(w, map[string]any{"headshot": nil})
				return
			}
			proj, err := ProjectHeadshot(ctx, pool, userID, *h)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"headshot": proj})

		case http.MethodPost:
			// Placing a Headshot is the "enter Third Place" action -- gated on
			// Trailer Face readiness (Kernel 68 §3.2). Viewing the commons list,
			// one's own current Headshot, or history is left open; nothing not
			// already ready ever gets placed into the commons to begin with.
			ready, err := playerprofile.TrailerFaceReady(ctx, pool, userID)
			if err != nil {
				writeError(w, err)
				return
			}
			if !ready.Ready {
				writeError(w, errors.New("trailer_face_not_ready"))
				return
			}

			h, created, err := LeaveHeadshot(ctx, pool, userID)
			if err != nil {
				writeError(w, err)
				return
			}
			proj, err := ProjectHeadshot(ctx, pool, userID, h)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"headshot": proj, "created": created})

		case http.MethodDelete:
			if err := RemoveHeadshot(ctx, pool, userID); err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"removed": true})

		default:
			methodNotAllowed(w)
		}
	}
}

// HandleMyHistory handles GET /api/third-place/headshots/me/history -- the
// caller's own placement/removal ledger, never anyone else's, and never
// containing old Face content (Kernel 65 §3.5, §7, §9.5).
func HandleMyHistory(pool *pgxpool.Pool) http.HandlerFunc {
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

		rows, err := ListMyHeadshotHistory(ctx, pool, userID)
		if err != nil {
			writeError(w, err)
			return
		}

		entries := make([]HistoryEntry, 0, len(rows))
		for _, row := range rows {
			entries = append(entries, HistoryEntry{
				ID:        row.ID,
				Status:    row.Status,
				PlacedAt:  row.PlacedAt,
				RemovedAt: row.RemovedAt,
			})
		}
		writeOK(w, map[string]any{"history": entries})
	}
}
