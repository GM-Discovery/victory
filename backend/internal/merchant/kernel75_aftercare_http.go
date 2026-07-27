package merchant

// Player-facing HTTP surface for Aftercare (kernel-75 S8).
//
// Every route is Show-keyed and scoped to the authenticated caller through
// ResolveShowParticipation. A client never names another user: identity
// comes from the session cookie and the Character comes from the roster, so
// there is no field a Player could set to write into someone else's
// Aftercare or skip count.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/aftercare"
	"victory/backend/internal/storysofar"
)

func aftercareParticipation(p ShowParticipation) aftercare.Participation {
	return aftercare.Participation{
		UserID:          p.ActorUserID,
		CharacterCardID: p.CharacterCardID,
		ShowID:          p.ShowID,
		ShowRunID:       p.ShowRunID,
		SessionID:       p.SessionID,
	}
}

// requireAftercareEnabled is the fail-closed capability check (S8).
//
// The flag lives on the venue hosting the Show's current placement. A Show
// with no placement, or a venue without the flag, offers no Aftercare at all
// rather than a form whose submissions go nowhere.
func requireAftercareEnabled(ctx context.Context, pool *pgxpool.Pool, showID string) error {
	// The venue join MUST be COALESCE(p.venue_id, s.default_venue_id), the
	// same rule resolvePlacementVenueSlug uses. A placement usually inherits
	// its venue from the Scene and leaves venue_id NULL, so joining on
	// venue_id alone silently matches nothing and fails every capability
	// check -- which is exactly the bug the Kernel 75 golden-journey proof
	// caught on live data.
	var enabled bool
	err := pool.QueryRow(ctx, `
		SELECT COALESCE((v.config ->> 'aftercare_enabled')::boolean, FALSE)
		FROM shows s
		JOIN show_scene_placements ssp ON ssp.show_id = s.id
		JOIN scenes sc ON sc.id = ssp.scene_id
		JOIN venues v ON v.id = COALESCE(ssp.venue_id, sc.default_venue_id)
		WHERE s.id = $1
		ORDER BY (ssp.id = s.current_show_scene_placement_id) DESC
		LIMIT 1
	`, showID).Scan(&enabled)
	if err != nil || !enabled {
		return errors.New("unknown_target")
	}
	return nil
}

// HandleShowAftercare handles GET (offer) and POST (Save and Close) for
// /api/shows/{show_id}/aftercare.
func HandleShowAftercare(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
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
		participation, err := ResolveShowParticipation(ctx, pool, userID, showID)
		if err != nil {
			writeError(w, err)
			return
		}
		p := aftercareParticipation(participation)

		switch r.Method {
		case http.MethodGet:
			state, err := aftercare.Offer(ctx, pool, p)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"aftercare": state})

		case http.MethodPost:
			var body struct {
				Responses map[string]string `json:"responses"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			submission, err := aftercare.Submit(ctx, pool, p, body.Responses)
			if err != nil {
				writeError(w, err)
				return
			}
			// S8.3: a submission creates a Story So Far event. Private like
			// every other entry -- the fact that they reflected is part of
			// the Character's history; the content stays in Aftercare.
			_ = storysofar.Persist(ctx, pool, storysofar.Ref{
				CharacterCardID: participation.CharacterCardID,
				OwnerUserID:     participation.ActorUserID,
				ShowRunID:       participation.ShowRunID,
				ShowID:          participation.ShowID,
				SessionID:       participation.SessionID,
			}, []storysofar.DraftEvent{{
				EventType:  storysofar.EventAftercare,
				Title:      "Aftercare",
				Summary:    "You wrote Aftercare after this Show.",
				SourceKind: storysofar.SourceAftercare,
				SourceRef:  "submitted",
				OccurredAt: submission.SubmittedAt,
			}})
			writeOK(w, map[string]any{"submission": submission})

		default:
			methodNotAllowed(w)
		}
	}
}

// HandleShowAftercareDraft handles PUT /api/shows/{show_id}/aftercare/draft.
//
// This is the repo's first server-side form autosave. Server-side rather
// than localStorage on purpose: a Player may finish on a different device,
// and reflection text must not linger in browser storage after a logout on
// a shared machine.
//
// A draft is never a submission and never touches the skip count (S8.2).
func HandleShowAftercareDraft(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
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
		if err := requireAftercareEnabled(ctx, pool, showID); err != nil {
			writeError(w, err)
			return
		}
		participation, err := ResolveShowParticipation(ctx, pool, userID, showID)
		if err != nil {
			writeError(w, err)
			return
		}
		var body struct {
			Responses map[string]string `json:"responses"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		draft, err := aftercare.SaveDraft(ctx, pool, aftercareParticipation(participation), body.Responses)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"draft": draft})
	}
}

// HandleShowAftercareSkip handles POST /api/shows/{show_id}/aftercare/skip.
//
// Requires {"confirmed": true} (S8.4/S1.14): the skip is a two-step act, and
// the count must reflect a deliberate choice rather than an accidental one.
// The server refuses an unconfirmed skip rather than trusting the client to
// have shown the warning panel.
//
// This is the ONLY writer of aftercare_skips. There is no DELETE route and
// no client-supplied count, which together are why the skip count cannot be
// forged (S12).
func HandleShowAftercareSkip(pool *pgxpool.Pool) http.HandlerFunc {
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
		if err := requireAftercareEnabled(ctx, pool, showID); err != nil {
			writeError(w, err)
			return
		}
		participation, err := ResolveShowParticipation(ctx, pool, userID, showID)
		if err != nil {
			writeError(w, err)
			return
		}
		var body struct {
			Confirmed bool `json:"confirmed"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		if err := aftercare.Skip(ctx, pool, aftercareParticipation(participation), body.Confirmed); err != nil {
			writeError(w, err)
			return
		}
		// S8.4: a minimal Story So Far event. Records that the Player was
		// offered the moment and declined it -- not a judgement, just the
		// same completeness the rest of the history has.
		_ = storysofar.Persist(ctx, pool, storysofar.Ref{
			CharacterCardID: participation.CharacterCardID,
			OwnerUserID:     participation.ActorUserID,
			ShowRunID:       participation.ShowRunID,
			ShowID:          participation.ShowID,
			SessionID:       participation.SessionID,
		}, []storysofar.DraftEvent{{
			EventType:  storysofar.EventAftercare,
			Title:      "Aftercare",
			Summary:    "You chose not to write Aftercare after this Show.",
			SourceKind: storysofar.SourceAftercare,
			SourceRef:  "skipped",
		}})

		state, err := aftercare.Offer(ctx, pool, aftercareParticipation(participation))
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"aftercare": state})
	}
}
