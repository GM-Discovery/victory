package identity

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"victory/backend/internal/access"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type incomingPermissionRequestRow struct {
	ID            string `json:"id"`
	UserID        string `json:"user_id"`
	Handle        string `json:"handle"`
	DisplayName   string `json:"display_name"`
	VenueSlug     string `json:"venue_slug"`
	RequestedRole string `json:"requested_role"`
	Note          string `json:"note"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
}

type productionRow struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	LocationSlug string `json:"location_slug"`
}

type respondPermissionRequestInput struct {
	RequestID string `json:"request_id"`
	Decision  string `json:"decision"`
}

func HandleListIncomingPermissionRequests(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID, err := currentUserID(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		allowed, err := access.IsOperatorUser(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "operator_lookup_failed"})
			return
		}

		locationRole, locationID, err := resolveInviteAuthorityScope(ctx, pool, userID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "authority_lookup_failed"})
			return
		}

		if !allowed && locationRole != "producer" && locationRole != "director" {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
			return
		}

		rows, err := pool.Query(ctx, `
			SELECT
			  pr.id::text,
			  pr.user_id::text,
			  COALESCE(NULLIF(u.handle, ''), LEFT(u.id::text, 8), 'Unknown Participant'),
			  COALESCE(NULLIF(u.display_name, ''), NULLIF(u.handle, ''), LEFT(u.id::text, 8), 'Unknown Participant'),
			  pr.venue_slug,
			  pr.requested_role::text,
			  COALESCE(pr.note, ''),
			  pr.status,
			  pr.created_at::text
			FROM permission_requests pr
			JOIN users u ON u.id = pr.user_id
			JOIN venues v ON v.slug = pr.venue_slug
			JOIN lots l ON l.id = v.lot_id
			WHERE pr.status = 'pending'
			  AND (
			    $1::boolean = TRUE
			    OR l.location_id = $3::uuid
			  )
			  AND (
			    $1::boolean = TRUE
			    OR ($2 = 'producer' AND pr.requested_role::text = 'director')
			    OR ($2 = 'director' AND pr.requested_role::text IN ('cast', 'crew'))
			  )
			ORDER BY pr.created_at DESC
		`, allowed, locationRole, locationID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "query_failed"})
			return
		}
		defer rows.Close()

		var out []incomingPermissionRequestRow
		for rows.Next() {
			var row incomingPermissionRequestRow
			if err := rows.Scan(&row.ID, &row.UserID, &row.Handle, &row.DisplayName, &row.VenueSlug, &row.RequestedRole, &row.Note, &row.Status, &row.CreatedAt); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "scan_failed"})
				return
			}
			out = append(out, row)
		}
		if err := rows.Err(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "query_failed"})
			return
		}

		_ = locationID

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"data": out,
		})
	}
}

// HandleProductionsCollection handles GET (list) and POST (create,
// Kernel 68 §3.7) on /api/productions.
func HandleProductionsCollection(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleListProductions(pool, w, r)
		case http.MethodPost:
			handleCreateProduction(pool, w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
		}
	}
}

func handleListProductions(pool *pgxpool.Pool, w http.ResponseWriter, r *http.Request) {
	{
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID, err := currentUserID(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		if ok, err := access.IsOperatorUser(ctx, pool, userID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "operator_lookup_failed"})
			return
		} else if ok {
			rows, err := pool.Query(ctx, `
				SELECT
				  p.id::text,
				  p.name,
				  p.slug,
				  l.slug
				FROM productions p
				JOIN locations l ON l.id = p.location_id
				ORDER BY p.created_at ASC
			`)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "query_failed"})
				return
			}
			defer rows.Close()

			var out []productionRow
			for rows.Next() {
				var row productionRow
				if err := rows.Scan(&row.ID, &row.Name, &row.Slug, &row.LocationSlug); err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "scan_failed"})
					return
				}
				out = append(out, row)
			}
			if err := rows.Err(); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "query_failed"})
				return
			}

			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": out})
			return
		}

		// Kernel 76 (K76-M01): resolveInviteAuthorityScope resolves a Location
		// for every membership role including audience, so any member of a
		// Location -- previously, anyone who had just signed up -- got the full
		// list of that Location's Production names and slugs. Listing the
		// Productions of a Location is staff work; audience and cast have no
		// call for it.
		role, locationID, err := resolveInviteAuthorityScope(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
			return
		}
		switch role {
		case "producer", "director":
		default:
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
			return
		}

		rows, err := pool.Query(ctx, `
			SELECT
			  p.id::text,
			  p.name,
			  p.slug,
			  l.slug
			FROM productions p
			JOIN locations l ON l.id = p.location_id
			WHERE p.location_id = $1
			ORDER BY p.created_at ASC
		`, locationID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "query_failed"})
			return
		}
		defer rows.Close()

		var out []productionRow
		for rows.Next() {
			var row productionRow
			if err := rows.Scan(&row.ID, &row.Name, &row.Slug, &row.LocationSlug); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "scan_failed"})
				return
			}
			out = append(out, row)
		}
		if err := rows.Err(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "query_failed"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": out})
	}
}

type createProductionInput struct {
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	LocationSlug string `json:"location_slug"`
}

// handleCreateProduction closes the onboarding gap identified in Kernel 66:
// Show Run creation consumes /api/productions but there was never an
// in-app create route (Kernel 68 §3.7). Authority: Operator, or a Producer/
// Director creating for their own resolved location -- location_id is
// always resolved server-side via resolveInviteAuthorityScope, never taken
// from client input, except that an Operator may target a specific location
// by slug (still resolved to an id server-side, not trusted verbatim).
func handleCreateProduction(pool *pgxpool.Pool, w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	userID, err := currentUserID(ctx, pool, r)
	if err != nil || strings.TrimSpace(userID) == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
		return
	}

	var input createProductionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_json"})
		return
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "name_required"})
		return
	}

	isOperator, err := access.IsOperatorUser(ctx, pool, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "operator_lookup_failed"})
		return
	}

	var locationID string
	if isOperator && strings.TrimSpace(input.LocationSlug) != "" {
		if err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = $1`, strings.TrimSpace(input.LocationSlug)).Scan(&locationID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "location_not_found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "location_lookup_failed"})
			return
		}
	} else {
		role, resolvedLocationID, err := resolveInviteAuthorityScope(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
			return
		}
		if !isOperator && role != "producer" && role != "director" {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
			return
		}
		locationID = resolvedLocationID
	}

	slug := slugifyProductionCandidate(input.Slug)
	if slug == "" {
		slug = slugifyProductionCandidate(name)
	}
	if slug == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "slug_required"})
		return
	}

	var row productionRow
	err = pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug, created_by_user_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, name, slug
	`, locationID, name, slug, userID).Scan(&row.ID, &row.Name, &row.Slug)
	if err != nil {
		if strings.Contains(err.Error(), "productions_location_id_slug_key") {
			writeJSON(w, http.StatusConflict, map[string]any{"ok": false, "error": "slug_already_used"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "insert_failed"})
		return
	}
	if err := pool.QueryRow(ctx, `SELECT slug FROM locations WHERE id = $1`, locationID).Scan(&row.LocationSlug); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "location_lookup_failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": row})
}

func slugifyProductionCandidate(candidate string) string {
	lower := strings.ToLower(strings.TrimSpace(candidate))
	var b strings.Builder
	lastHyphen := false
	for _, r := range lower {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			lastHyphen = false
		default:
			if !lastHyphen && b.Len() > 0 {
				b.WriteRune('-')
				lastHyphen = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func HandleRespondPermissionRequest(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := currentUserID(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		authorityRole, locationID, err := resolveInviteAuthorityScope(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
			return
		}

		allowedToReview := authorityRole == "producer" || authorityRole == "director"
		if !allowedToReview {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
			return
		}

		var input respondPermissionRequestInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_json"})
			return
		}

		requestID := strings.TrimSpace(input.RequestID)
		decision := strings.ToLower(strings.TrimSpace(input.Decision))
		if requestID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "request_id_required"})
			return
		}
		if decision != "approve" && decision != "deny" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "decision_required"})
			return
		}

		var requestUserID, venueSlug, requestedRole, requestStatus, venueID string
		err = pool.QueryRow(ctx, `
			SELECT
			  pr.user_id::text,
			  pr.venue_slug,
			  pr.requested_role::text,
			  pr.status,
			  v.id::text
			FROM permission_requests pr
			JOIN venues v ON v.slug = pr.venue_slug
			JOIN lots l ON l.id = v.lot_id
			WHERE pr.id = $1::uuid
			  AND l.location_id = $2::uuid
			LIMIT 1
		`, requestID, locationID).Scan(&requestUserID, &venueSlug, &requestedRole, &requestStatus, &venueID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "request_not_found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "query_failed"})
			return
		}

		if requestStatus != "pending" {
			writeJSON(w, http.StatusConflict, map[string]any{"ok": false, "error": "request_already_reviewed"})
			return
		}

		switch authorityRole {
		case "producer":
		case "director":
			if requestedRole == "director" {
				writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "insufficient_role"})
				return
			}
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "tx_begin_failed"})
			return
		}
		defer tx.Rollback(ctx)

		if decision == "approve" {
			switch requestedRole {
			case "director", "cast", "crew":
				var productionID string
				if err := tx.QueryRow(ctx, `
					SELECT p.id::text
					FROM productions p
					WHERE p.location_id = $1::uuid
					ORDER BY p.created_at ASC
					LIMIT 1
				`, locationID).Scan(&productionID); err != nil {
					writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "production_required"})
					return
				}

				_, err = tx.Exec(ctx, `
					INSERT INTO memberships (location_id, user_id, role, production_id, granted_by_user_id, active)
					VALUES ($1, $2, $3::location_role, $4::uuid, $5, TRUE)
					ON CONFLICT DO NOTHING
				`, locationID, requestUserID, requestedRole, productionID, userID)
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "membership_create_failed"})
					return
				}
			case "audience":
				// Audience is a per-Showing ticket, never a standing venue
				// grant (unlike cast/crew/director membership above) --
				// route through Kernel 93's audience_admissions instead of
				// access_grants (see visibility.go's audience_admission_
				// surface, which only reads audience_admissions, not
				// access_grants). The approver's manage authority over this
				// venue's location was already established above via
				// resolveInviteAuthorityScope/authorityRole, so this only
				// needs to resolve which Showing is currently live/
				// rehearsal at the venue -- same "sessions row for this
				// venue_id with status IN ('rehearsal','live')" definition
				// of current-ness used everywhere else, not a new selector.
				tag, err := tx.Exec(ctx, `
					INSERT INTO audience_admissions (showing_id, user_id, issued_by_user_id)
					SELECT sh.id, $2::uuid, $3::uuid
					FROM sessions s
					JOIN showings sh ON sh.session_id = s.id
					WHERE s.venue_id = $1::uuid AND s.status IN ('rehearsal', 'live')
					ORDER BY s.started_at DESC
					LIMIT 1
					ON CONFLICT (showing_id, user_id) DO UPDATE
					SET issued_by_user_id = EXCLUDED.issued_by_user_id,
					    issued_at = NOW()
				`, venueID, requestUserID, userID)
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "admission_create_failed"})
					return
				}
				if tag.RowsAffected() == 0 {
					writeJSON(w, http.StatusConflict, map[string]any{"ok": false, "error": "no_active_showing"})
					return
				}
			default:
				_, err = tx.Exec(ctx, `
					INSERT INTO access_grants (location_id, user_id, grant_type, venue_id, granted_by_user_id, created_at)
					VALUES ($1, $2, 'venue_access', $3::uuid, $4, NOW())
					ON CONFLICT DO NOTHING
				`, locationID, requestUserID, venueID, userID)
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "grant_create_failed"})
					return
				}
			}
		}

		_, err = tx.Exec(ctx, `
			UPDATE permission_requests
			SET status = $2,
			    reviewed_by = $3,
			    reviewed_at = NOW()
			WHERE id = $1::uuid
		`, requestID, map[string]string{"approve": "approved", "deny": "denied"}[decision], userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "request_update_failed"})
			return
		}

		if err := tx.Commit(ctx); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "tx_commit_failed"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true,
			"data": map[string]any{
				"request_id":     requestID,
				"decision":       decision,
				"venue_slug":     venueSlug,
				"requested_role": requestedRole,
			},
		})
	}
}

func resolveInviteAuthorityScope(ctx context.Context, pool *pgxpool.Pool, userID string) (string, string, error) {
	if ok, err := access.IsOperatorUser(ctx, pool, userID); err != nil {
		return "", "", err
	} else if ok {
		var locationID string
		if err := pool.QueryRow(ctx, `
			SELECT id::text
			FROM locations
			WHERE slug = 'amurray-family'
			LIMIT 1
		`).Scan(&locationID); err != nil {
			return "", "", err
		}
		return "producer", locationID, nil
	}

	var locationID, role string
	err := pool.QueryRow(ctx, `
		SELECT lm.location_id::text, lm.role::text
		FROM location_memberships lm
		WHERE lm.user_id = $1
		  AND lm.active = TRUE
		ORDER BY
		  CASE lm.role
			WHEN 'producer' THEN 1
			WHEN 'director' THEN 2
			WHEN 'cast' THEN 3
			WHEN 'crew' THEN 4
			WHEN 'audience' THEN 5
			ELSE 99
		  END,
		  lm.created_at ASC
		LIMIT 1
	`, userID).Scan(&locationID, &role)
	if err != nil {
		return "", "", err
	}

	switch role {
	case "producer", "director", "cast", "crew", "audience":
		return role, locationID, nil
	default:
		return "", "", pgx.ErrNoRows
	}
}
