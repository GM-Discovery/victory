// Package showtime implements Kernel 71's `/showtime` orchestration: the
// single Director-facing action that resolves a Show by its short code,
// derives which theater venue it belongs on, and starts or resumes the
// live Session for it via the existing (unmodified) Kernel 70A
// shows.StartShowSession -- without the Director separately operating
// Production/Show-Run/Show/Session state or a manual venue picker.
package showtime

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/scenes"
	"victory/backend/internal/showings"
	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
)

// StartResult is /showtime's response shape, formatted by the HTTP layer
// into the exact example block from spec §9.3.
type StartResult struct {
	ShowID               string
	ShowTitle            string
	ShortCode            string
	VenueSlug            string
	VenueName            string
	CurrentSceneName     string
	PlayersReady         int
	PlayersNeedCharacter int
	MicOn                bool // always false -- StartShowSession never touches mic state
	SessionID            string
	WasResumed           bool

	VenueBusyWithOtherShow *shows.VenueBusyInfo

	// NeedsVenueChoice is set (with everything else above zero) when no
	// safe venue could be derived automatically -- the Show has zero
	// staged placements, or its placements resolve to more than one
	// distinct venue. The caller re-invokes Start with chosenVenueSlug set
	// to one of CandidateVenues (or any supported venue, if
	// CandidateVenues is empty).
	NeedsVenueChoice bool
	CandidateVenues  []string
}

// EndResult is /showtime end's response shape.
type EndResult struct {
	ShowID    string
	ShowTitle string
	SessionID string
}

// Start resolves shortCode, derives the venue (unless chosenVenueSlug is
// already given, e.g. a re-invocation after a NeedsVenueChoice response),
// and starts/resumes the Show's live Session.
func Start(ctx context.Context, pool *pgxpool.Pool, actorUserID, shortCode string, forceReattach bool, chosenVenueSlug string) (StartResult, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return StartResult{}, errors.New("not_authenticated")
	}

	s, err := shows.LoadShowByCode(ctx, pool, actorUserID, shortCode)
	if err != nil {
		if err.Error() == "show_code_not_found" || err.Error() == "show_code_ambiguous" {
			return StartResult{}, err
		}
		return StartResult{}, err
	}

	sr, err := showruns.LoadShowRunByID(ctx, pool, s.ShowRunID)
	if err != nil {
		return StartResult{}, err
	}
	canManage, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return StartResult{}, err
	}
	if !canManage {
		return StartResult{}, errors.New("not_authorized")
	}

	venueSlug := strings.ToLower(strings.TrimSpace(chosenVenueSlug))
	if venueSlug == "" {
		derived, needsChoice, candidates, err := deriveVenueSlug(ctx, pool, s)
		if err != nil {
			return StartResult{}, err
		}
		if needsChoice {
			return StartResult{
				ShowID: s.ID, ShowTitle: s.Title, ShortCode: s.ShortCode,
				NeedsVenueChoice: true, CandidateVenues: candidates,
			}, nil
		}
		venueSlug = derived
	}

	result, err := shows.StartShowSession(ctx, pool, actorUserID, s.ID, venueSlug, forceReattach)
	if err != nil {
		return StartResult{}, err
	}
	if result.VenueBusyWithOtherShow != nil {
		return StartResult{
			ShowID: s.ID, ShowTitle: s.Title, ShortCode: s.ShortCode,
			VenueBusyWithOtherShow: result.VenueBusyWithOtherShow,
		}, nil
	}

	venueName, err := loadVenueName(ctx, pool, venueSlug)
	if err != nil {
		return StartResult{}, err
	}

	s, err = shows.LoadShowByID(ctx, pool, s.ID)
	if err != nil {
		return StartResult{}, err
	}
	currentSceneName := ""
	if s.CurrentShowScenePlacementID != nil {
		if placements, err := scenes.ListPlacementsForShow(ctx, pool, s.ID); err == nil {
			for _, p := range placements {
				if p.Placement.ID == *s.CurrentShowScenePlacementID {
					currentSceneName = p.Scene.Title
					break
				}
			}
		}
	}

	ready, needCharacter, err := countRosterCharacterReadiness(ctx, pool, s.ShowRunID)
	if err != nil {
		return StartResult{}, err
	}

	return StartResult{
		ShowID:               s.ID,
		ShowTitle:            s.Title,
		ShortCode:            s.ShortCode,
		VenueSlug:            venueSlug,
		VenueName:            venueName,
		CurrentSceneName:     currentSceneName,
		PlayersReady:         ready,
		PlayersNeedCharacter: needCharacter,
		MicOn:                false,
		SessionID:            result.SessionID,
		WasResumed:           result.WasResumed,
	}, nil
}

// Status reports the same shape as Start without starting or changing
// anything -- resolves the Show and its current live state only.
func Status(ctx context.Context, pool *pgxpool.Pool, actorUserID, shortCode string) (StartResult, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return StartResult{}, errors.New("not_authenticated")
	}
	s, err := shows.LoadShowByCode(ctx, pool, actorUserID, shortCode)
	if err != nil {
		return StartResult{}, err
	}
	sr, err := showruns.LoadShowRunByID(ctx, pool, s.ShowRunID)
	if err != nil {
		return StartResult{}, err
	}
	canManage, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return StartResult{}, err
	}
	if !canManage {
		return StartResult{}, errors.New("not_authorized")
	}

	ready, needCharacter, err := countRosterCharacterReadiness(ctx, pool, s.ShowRunID)
	if err != nil {
		return StartResult{}, err
	}

	var venueSlug string
	var sessionID string
	var wasLive bool
	err = pool.QueryRow(ctx, `
		SELECT v.slug, sess.id::text, sess.status = 'live'
		FROM sessions sess
		JOIN venues v ON v.id = sess.venue_id
		WHERE sess.show_id = $1 AND sess.status IN ('rehearsal', 'live')
		ORDER BY sess.started_at DESC LIMIT 1
	`, s.ID).Scan(&venueSlug, &sessionID, &wasLive)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return StartResult{}, err
	}

	venueName := ""
	if venueSlug != "" {
		venueName, _ = loadVenueName(ctx, pool, venueSlug)
	}

	return StartResult{
		ShowID: s.ID, ShowTitle: s.Title, ShortCode: s.ShortCode,
		VenueSlug: venueSlug, VenueName: venueName,
		PlayersReady: ready, PlayersNeedCharacter: needCharacter,
		SessionID: sessionID,
	}, nil
}

// End ends the technical live Session for a Show by short code. It
// deliberately touches nothing else -- no write path here reaches
// current_show_scene_placement_id, show_run_roster_members, or
// character_card_id, so the Show's persistent state and every Player's
// Character selection survive by construction (spec §9.4), provable by a
// before/after snapshot test.
func End(ctx context.Context, pool *pgxpool.Pool, actorUserID, shortCode string) (EndResult, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return EndResult{}, errors.New("not_authenticated")
	}
	s, err := shows.LoadShowByCode(ctx, pool, actorUserID, shortCode)
	if err != nil {
		return EndResult{}, err
	}
	sr, err := showruns.LoadShowRunByID(ctx, pool, s.ShowRunID)
	if err != nil {
		return EndResult{}, err
	}
	canManage, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return EndResult{}, err
	}
	if !canManage {
		return EndResult{}, errors.New("not_authorized")
	}

	var sessionID string
	err = pool.QueryRow(ctx, `
		SELECT id::text FROM sessions
		WHERE show_id = $1 AND status IN ('rehearsal', 'live')
		ORDER BY started_at DESC LIMIT 1
	`, s.ID).Scan(&sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return EndResult{}, errors.New("no_active_session_for_show")
	}
	if err != nil {
		return EndResult{}, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return EndResult{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := showings.CloseBySession(ctx, tx, sessionID); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return EndResult{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE sessions SET status = 'closed', ended_at = COALESCE(ended_at, NOW()) WHERE id = $1
	`, sessionID); err != nil {
		return EndResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return EndResult{}, err
	}

	return EndResult{ShowID: s.ID, ShowTitle: s.Title, SessionID: sessionID}, nil
}

// deriveVenueSlug implements spec §9.2's venue-resolution fallback: prefer
// the Show's already-persistent current Scene's venue; otherwise, if every
// staged placement resolves to the same single venue, use it (and set it
// as the current Scene as a side effect, since the Show had none yet);
// otherwise ask.
func deriveVenueSlug(ctx context.Context, pool *pgxpool.Pool, s shows.Show) (venueSlug string, needsChoice bool, candidates []string, err error) {
	placements, err := scenes.ListPlacementsForShow(ctx, pool, s.ID)
	if err != nil {
		return "", false, nil, err
	}

	if s.CurrentShowScenePlacementID != nil {
		for _, p := range placements {
			if p.Placement.ID == *s.CurrentShowScenePlacementID {
				if venueID := resolvedVenueID(p); venueID != nil {
					slug, err := loadVenueSlugByID(ctx, pool, *venueID)
					if err != nil {
						return "", false, nil, err
					}
					return slug, false, nil, nil
				}
				break
			}
		}
		// The current placement has no resolvable venue at all (neither an
		// override nor a Scene default) -- fall through to the
		// staged-placements scan below rather than erroring outright.
	}

	if len(placements) == 0 {
		return "", true, nil, nil
	}

	distinctVenueIDs := map[string]bool{}
	var firstPlacementID string
	for _, p := range placements {
		if venueID := resolvedVenueID(p); venueID != nil {
			distinctVenueIDs[*venueID] = true
		}
		if firstPlacementID == "" {
			firstPlacementID = p.Placement.ID
		}
	}

	if len(distinctVenueIDs) != 1 {
		var candidateSlugs []string
		for venueID := range distinctVenueIDs {
			if slug, err := loadVenueSlugByID(ctx, pool, venueID); err == nil {
				candidateSlugs = append(candidateSlugs, slug)
			}
		}
		return "", true, candidateSlugs, nil
	}

	var onlyVenueID string
	for venueID := range distinctVenueIDs {
		onlyVenueID = venueID
	}
	slug, err := loadVenueSlugByID(ctx, pool, onlyVenueID)
	if err != nil {
		return "", false, nil, err
	}

	// Give the Show a real current Scene going forward, matching the
	// persistence model -- a Show that had never been staged before now
	// has one, rather than /showtime silently relying on placement-scan
	// derivation every time it starts.
	if _, err := shows.SetCurrentScenePlacementTrusted(ctx, pool, s.ID, firstPlacementID); err != nil {
		return "", false, nil, err
	}

	return slug, false, nil, nil
}

func resolvedVenueID(p scenes.PlacementDetail) *string {
	if p.Placement.VenueID != nil && strings.TrimSpace(*p.Placement.VenueID) != "" {
		return p.Placement.VenueID
	}
	if p.Scene.DefaultVenueID != nil && strings.TrimSpace(*p.Scene.DefaultVenueID) != "" {
		return p.Scene.DefaultVenueID
	}
	return nil
}

func loadVenueSlugByID(ctx context.Context, pool *pgxpool.Pool, venueID string) (string, error) {
	var slug string
	err := pool.QueryRow(ctx, `SELECT slug FROM venues WHERE id = $1`, venueID).Scan(&slug)
	return slug, err
}

func loadVenueName(ctx context.Context, pool *pgxpool.Pool, venueSlug string) (string, error) {
	var name string
	err := pool.QueryRow(ctx, `SELECT name FROM venues WHERE slug = $1`, venueSlug).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return name, err
}

// countRosterCharacterReadiness returns (players with a selected Character,
// players without one) for a Show Run's active player roster rows.
func countRosterCharacterReadiness(ctx context.Context, pool *pgxpool.Pool, showRunID string) (ready, needCharacter int, err error) {
	err = pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE character_card_id IS NOT NULL),
			COUNT(*) FILTER (WHERE character_card_id IS NULL)
		FROM show_run_roster_members
		WHERE show_run_id = $1 AND role = 'player' AND removed_at IS NULL
	`, showRunID).Scan(&ready, &needCharacter)
	return ready, needCharacter, err
}
