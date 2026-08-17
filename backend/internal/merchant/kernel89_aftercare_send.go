// Kernel 89 §17: Send Aftercare as a first-class Director operation.
//
// This adds DELIVERY ONLY. Every byte of Aftercare's data model, prompts,
// draft/submit/skip semantics and Directors-read-only review comes from
// Kernel 75's backend/internal/aftercare package, untouched -- §17.3's "do
// not create a second Aftercare system" is satisfied structurally: nothing
// here writes an aftercare_* row. A Director pressing Send does not answer,
// pre-create, or resolve anything on a Player's behalf; it opens the Player's
// own existing form.
//
// It is also deliberately manual (§17.2). Nothing in showtime.End calls this,
// and nothing should: the humans decide when the performance is actually
// finished. A Session that ends because the internet died is not a Show that
// is ready for reflection.
package merchant

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/network"
	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
)

// AftercareRecipient is one Player the Director's send reached.
type AftercareRecipient struct {
	UserID          string `json:"user_id"`
	DisplayName     string `json:"display_name"`
	CharacterCardID string `json:"character_card_id"`
	CharacterName   string `json:"character_name"`
	Delivered       bool   `json:"delivered"`
}

// resolveAftercareTargets lists the Players a Send Aftercare would reach.
//
// The eligibility rule is exactly Kernel 75's own: an active Player roster
// row with a selected Character. Anyone else -- Directors, Crew, Audience,
// a Player who never picked a Character -- has no Aftercare record to write
// and is therefore not a target, which is also what makes §32's "unintended
// users do not receive it" true by construction rather than by filtering.
//
// cohortID narrows to one Cohort (§9.3's targeting vocabulary, reused rather
// than reinvented). Empty means every eligible Player on the Show.
func resolveAftercareTargets(ctx context.Context, pool *pgxpool.Pool, showID, cohortID string) ([]AftercareRecipient, error) {
	show, err := shows.LoadShowByID(ctx, pool, showID)
	if err != nil {
		return nil, err
	}
	cohortID = strings.TrimSpace(cohortID)

	rows, err := pool.Query(ctx, `
		SELECT
			m.user_id::text,
			COALESCE(NULLIF(u.display_name, ''), u.email, 'Player'),
			m.character_card_id::text,
			COALESCE(c.name, '')
		FROM show_run_roster_members m
		JOIN users u ON u.id = m.user_id
		LEFT JOIN character_cards c ON c.id = m.character_card_id
		WHERE m.show_run_id = $1
		  AND m.removed_at IS NULL
		  AND lower(m.role::text) = 'player'
		  AND m.character_card_id IS NOT NULL
		  AND (
		    $2 = ''
		    OR EXISTS (
		      SELECT 1 FROM show_cohort_assignments a
		      WHERE a.show_id = $3 AND a.user_id = m.user_id AND a.cohort_id = $2::uuid
		    )
		  )
		ORDER BY 2
	`, show.ShowRunID, cohortID, showID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []AftercareRecipient{}
	for rows.Next() {
		var r AftercareRecipient
		if err := rows.Scan(&r.UserID, &r.DisplayName, &r.CharacterCardID, &r.CharacterName); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// HandleShowAftercareSend handles POST /api/shows/{show_id}/aftercare/send.
func HandleShowAftercareSend(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
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

		if err := requireAftercareEnabled(ctx, pool, showID); err != nil {
			writeError(w, err)
			return
		}
		if _, err := requireShowDirector(ctx, pool, userID, showID); err != nil {
			writeError(w, err)
			return
		}

		var body struct {
			CohortID string `json:"cohort_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)

		recipients, err := resolveAftercareTargets(ctx, pool, showID, body.CohortID)
		if err != nil {
			writeError(w, err)
			return
		}

		// The Session is resolved server-side rather than taken from the
		// request so a Director cannot aim an Aftercare prompt at a socket
		// belonging to some other Show.
		var sessionID string
		_ = pool.QueryRow(ctx, `
			SELECT id::text FROM sessions
			WHERE show_id = $1 AND status IN ('rehearsal', 'live')
			ORDER BY started_at DESC LIMIT 1
		`, showID).Scan(&sessionID)

		msg, _ := json.Marshal(map[string]any{
			"type":    "aftercare/offer",
			"show_id": showID,
		})
		for i := range recipients {
			if strings.TrimSpace(sessionID) == "" {
				continue
			}
			// Per-user targeting, never a session broadcast: a Director and
			// an Audience member share this session and neither has an
			// Aftercare record to write.
			hub.BroadcastToSessionUser(sessionID, recipients[i].UserID, msg)
			recipients[i].Delivered = true
		}

		delivered := 0
		for _, rec := range recipients {
			if rec.Delivered {
				delivered++
			}
		}

		// A recipient with no live socket is reported honestly rather than
		// silently counted: their Aftercare is still reachable (the form is
		// Show-keyed and survives the Session), but the Director should know
		// the popup did not appear in front of them right now.
		writeOK(w, map[string]any{
			"recipients":      recipients,
			"targeted":        len(recipients),
			"delivered":       delivered,
			"session_id":      sessionID,
			"session_is_live": strings.TrimSpace(sessionID) != "",
		})
	}
}

// requireShowDirector is the Director+ gate for Show-scoped Director
// operations in this package, resolved against the Show's own Show Run
// location -- the same check cohorts.requireManage and
// directorprep.RequireDirector make. It returns that location, since every
// caller that needs the gate also needs the scope it was checked against
// and re-deriving it would be a second chance to derive it differently.
func requireShowDirector(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID string) (string, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return "", errors.New("not_authenticated")
	}
	show, err := shows.LoadShowByID(ctx, pool, showID)
	if err != nil {
		return "", err
	}
	run, err := showruns.LoadShowRunByID(ctx, pool, show.ShowRunID)
	if err != nil {
		return "", err
	}
	ok, err := showruns.CanManageShowRun(ctx, pool, actorUserID, run.LocationID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errors.New("not_authorized")
	}
	return run.LocationID, nil
}
