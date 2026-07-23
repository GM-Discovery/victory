package merchant

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HandlePrepareLockedCourtyardOpening handles POST
// /api/shows/{show_id}/prepare-locked-courtyard-opening -- the Kernel 73A
// idempotent Director action (spec S8) that wires up the Courtyard Scene
// placement, Kessa's participant interaction, her stage token, and the
// binding between them, reporting exactly which steps already existed vs
// were newly created plus any unmet prerequisite ("gap").
func HandlePrepareLockedCourtyardOpening(pool *pgxpool.Pool) http.HandlerFunc {
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
		showID := strings.TrimSpace(r.PathValue("show_id"))
		result, err := PrepareLockedCourtyardOpening(ctx, pool, userID, showID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"result": result})
	}
}

// HandleKessaReachabilityDiagnostics handles GET
// /api/shows/{show_id}/kessa-reachability -- read-only diagnostics, no
// mutation, available to anyone who can view the Show's backstage (Kernel
// 73A S9).
func HandleKessaReachabilityDiagnostics(pool *pgxpool.Pool) http.HandlerFunc {
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
		result, err := KessaReachabilityDiagnostics(ctx, pool, userID, showID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"result": result})
	}
}
