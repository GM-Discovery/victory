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

	"victory/backend/internal/identity"
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
	ChatBridgeOn         bool // Kernel 92: actually reflects the Discord thread bridge, see identity.TryTurnOnChatBridgeForVenue
	ChatBridgeMessage    string
	SessionID            string
	WasResumed           bool

	// AlreadyLive is set when the Show already had a `live` session before
	// this call -- Start returns immediately without re-running
	// StartShowSession or re-toggling the chat bridge (kernel doc §21:
	// repeated Showtime must be idempotent, "You're already live.").
	AlreadyLive bool

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

	// AlreadyEnded is set when there was no active session to end -- End
	// is idempotent (kernel doc §21) rather than erroring on a repeat call.
	AlreadyEnded bool

	// AftercareEligibleCount is the number of active player roster members
	// with a selected Character, i.e. how many would receive an Aftercare
	// offer -- used by the GUI to decide whether to show the "Send
	// Aftercare? / Not Yet" reminder (kernel doc §23). End itself never
	// sends Aftercare; Kernel 89's existing send action remains a
	// separate, human-confirmed step.
	AftercareEligibleCount int
}

// Start resolves shortCode or showID (exactly one should be set -- showID
// takes precedence, letting the Kernel 92 GUI popup select a Showing
// directly without the Director typing a short code, per kernel doc §2),
// derives the venue (unless chosenVenueSlug is already given, e.g. a
// re-invocation after a NeedsVenueChoice response), and starts/resumes the
// Show's live Session plus its chat bridge.
func Start(ctx context.Context, pool *pgxpool.Pool, discordCfg identity.DiscordServerLinkConfig, actorUserID, shortCode, showID string, forceReattach bool, chosenVenueSlug string) (StartResult, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return StartResult{}, errors.New("not_authenticated")
	}

	s, err := resolveShow(ctx, pool, actorUserID, shortCode, showID)
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

	if alreadyLive, existingVenueSlug, existingSessionID, err := loadLiveSessionForShow(ctx, pool, s.ID); err != nil {
		return StartResult{}, err
	} else if alreadyLive {
		venueName, _ := loadVenueName(ctx, pool, existingVenueSlug)
		ready, needCharacter, err := countRosterCharacterReadiness(ctx, pool, s.ShowRunID)
		if err != nil {
			return StartResult{}, err
		}
		bridgeReady, _ := identity.ChatBridgeReady(ctx, pool, existingVenueSlug)
		return StartResult{
			ShowID: s.ID, ShowTitle: s.Title, ShortCode: s.ShortCode,
			VenueSlug: existingVenueSlug, VenueName: venueName,
			PlayersReady: ready, PlayersNeedCharacter: needCharacter,
			ChatBridgeOn: bridgeReady, SessionID: existingSessionID,
			AlreadyLive: true,
		}, nil
	}

	venueSlug := strings.ToLower(strings.TrimSpace(chosenVenueSlug))
	if venueSlug == "" {
		derived, needsChoice, candidates, err := deriveVenueSlug(ctx, pool, s, false)
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

	// Chat-bridge connection is a warning-level readiness signal, never a
	// blocker (kernel doc §14 vs §15) -- Showtime must still succeed even
	// if Discord isn't configured for this venue.
	bridgeMessage, bridgeReady, _ := identity.TryTurnOnChatBridgeForVenue(ctx, pool, discordCfg, venueSlug, actorUserID)

	return StartResult{
		ShowID:               s.ID,
		ShowTitle:            s.Title,
		ShortCode:            s.ShortCode,
		VenueSlug:            venueSlug,
		VenueName:            venueName,
		CurrentSceneName:     currentSceneName,
		PlayersReady:         ready,
		PlayersNeedCharacter: needCharacter,
		ChatBridgeOn:         bridgeReady,
		ChatBridgeMessage:    bridgeMessage,
		SessionID:            result.SessionID,
		WasResumed:           result.WasResumed,
	}, nil
}

// resolveShow prefers showID (the Kernel 92 GUI's selection) and falls back
// to shortCode (the /showtime <code> command's only argument), so both
// entry points share this single resolution path and, downstream, the same
// Start/Status/End logic (kernel doc §38).
func resolveShow(ctx context.Context, pool *pgxpool.Pool, actorUserID, shortCode, showID string) (shows.Show, error) {
	if strings.TrimSpace(showID) != "" {
		return shows.LoadShowByID(ctx, pool, strings.TrimSpace(showID))
	}
	return shows.LoadShowByCode(ctx, pool, actorUserID, shortCode)
}

// loadLiveSessionForShow reports whether the Show already has an active
// (rehearsal or live) session, matching Status()'s own definition of
// "currently running" -- used for Start's idempotency check. Sessions are
// created with status='rehearsal' (network.StartSessionControl) and never
// promoted to 'live' by any code path today, so "rehearsal or live" is the
// only definition of "already going" that actually matches production
// behavior; checking 'live' alone would never short-circuit anything.
func loadLiveSessionForShow(ctx context.Context, pool *pgxpool.Pool, showID string) (alreadyLive bool, venueSlug, sessionID string, err error) {
	err = pool.QueryRow(ctx, `
		SELECT v.slug, sess.id::text
		FROM sessions sess
		JOIN venues v ON v.id = sess.venue_id
		WHERE sess.show_id = $1 AND sess.status IN ('rehearsal', 'live')
		ORDER BY sess.started_at DESC LIMIT 1
	`, showID).Scan(&venueSlug, &sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, "", "", nil
	}
	if err != nil {
		return false, "", "", err
	}
	return true, venueSlug, sessionID, nil
}

// Status reports the same shape as Start without starting or changing
// anything -- resolves the Show and its current live state only.
func Status(ctx context.Context, pool *pgxpool.Pool, actorUserID, shortCode, showID string) (StartResult, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return StartResult{}, errors.New("not_authenticated")
	}
	s, err := resolveShow(ctx, pool, actorUserID, shortCode, showID)
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

// End ends the technical live Session for a Show (resolved by short code or
// showID, see resolveShow). It deliberately touches nothing else -- no
// write path here reaches current_show_scene_placement_id,
// show_run_roster_members, or character_card_id, so the Show's persistent
// state and every Player's Character selection survive by construction
// (spec §9.4), provable by a before/after snapshot test.
func End(ctx context.Context, pool *pgxpool.Pool, discordCfg identity.DiscordServerLinkConfig, actorUserID, shortCode, showID string) (EndResult, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return EndResult{}, errors.New("not_authenticated")
	}
	s, err := resolveShow(ctx, pool, actorUserID, shortCode, showID)
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

	aftercareCount, err := countAftercareEligible(ctx, pool, s.ShowRunID)
	if err != nil {
		return EndResult{}, err
	}

	var sessionID, venueSlug string
	err = pool.QueryRow(ctx, `
		SELECT sess.id::text, v.slug FROM sessions sess
		JOIN venues v ON v.id = sess.venue_id
		WHERE sess.show_id = $1 AND sess.status IN ('rehearsal', 'live')
		ORDER BY sess.started_at DESC LIMIT 1
	`, s.ID).Scan(&sessionID, &venueSlug)
	if errors.Is(err, pgx.ErrNoRows) {
		return EndResult{
			ShowID: s.ID, ShowTitle: s.Title,
			AlreadyEnded: true, AftercareEligibleCount: aftercareCount,
		}, nil
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

	// Best-effort: End Showtime must not fail because Discord is
	// unreachable (kernel doc §22's preserve-state guarantee is about Show/
	// Session state, not the chat bridge's own connectivity).
	_, _ = identity.TurnOffChatBridge(ctx, pool, discordCfg, venueSlug, actorUserID)

	return EndResult{
		ShowID: s.ID, ShowTitle: s.Title, SessionID: sessionID,
		AftercareEligibleCount: aftercareCount,
	}, nil
}

// countAftercareEligible returns how many active player roster members
// have a selected Character -- the same eligibility rule
// merchant.resolveAftercareTargets uses, inlined here as a simple read
// rather than importing the merchant package for one COUNT query.
func countAftercareEligible(ctx context.Context, pool *pgxpool.Pool, showRunID string) (int, error) {
	var count int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM show_run_roster_members
		WHERE show_run_id = $1 AND role = 'player' AND removed_at IS NULL AND character_card_id IS NOT NULL
	`, showRunID).Scan(&count)
	return count, err
}

// deriveVenueSlug implements spec §9.2's venue-resolution fallback: prefer
// the Show's already-persistent current Scene's venue; otherwise, if every
// staged placement resolves to the same single venue, use it (and set it
// as the current Scene as a side effect, since the Show had none yet);
// otherwise ask. dryRun suppresses that side effect -- Kernel 92's Preflight
// must not mutate Show state before the Director actually presses SHOWTIME.
func deriveVenueSlug(ctx context.Context, pool *pgxpool.Pool, s shows.Show, dryRun bool) (venueSlug string, needsChoice bool, candidates []string, err error) {
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
	// derivation every time it starts. Skipped entirely for a dry run.
	if !dryRun {
		if _, err := shows.SetCurrentScenePlacementTrusted(ctx, pool, s.ID, firstPlacementID); err != nil {
			return "", false, nil, err
		}
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

// countValidTickets rolls up Kernel 71's two-punch player ticket count for
// a Show Run -- tickets are Show-Run-scoped, not per-Showing (kernel doc
// §27 explicitly defers rebuilding ticketing), so Preflight reads this by
// the Show's parent ShowRunID.
func countValidTickets(ctx context.Context, pool *pgxpool.Pool, showRunID string) (int, error) {
	var count int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM show_run_tickets WHERE show_run_id = $1 AND status = 'valid'
	`, showRunID).Scan(&count)
	return count, err
}

// PreflightResult is the Kernel 92 "Ready for Showtime" summary (kernel doc
// §12/§39): everything the Director needs to decide whether to press
// SHOWTIME, split into true blockers (Start would fail) and warnings
// (Start still succeeds -- kernel doc §13/§15 explicitly forbid making
// Character completeness or an empty ticket count block live play).
type PreflightResult struct {
	ShowID    string
	ShowTitle string
	Nickname  string
	ShortCode string

	VenueSlug        string
	VenueName        string
	CurrentSceneName string

	CastAdmittedCount        int
	TicketCount              int
	ReadyCharacterCount      int
	IncompleteCharacterCount int
	ChatBridgeReady          bool

	IsLive bool

	Blockers []string
	Warnings []string
}

// Preflight resolves a Showing's readiness without starting or changing
// anything (kernel doc §12/§13) -- venue/Scene resolution runs in dry-run
// mode so it never persists a current-Scene placement as a side effect.
func Preflight(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID string) (PreflightResult, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return PreflightResult{}, errors.New("not_authenticated")
	}
	s, err := resolveShow(ctx, pool, actorUserID, "", showID)
	if err != nil {
		return PreflightResult{}, err
	}
	sr, err := showruns.LoadShowRunByID(ctx, pool, s.ShowRunID)
	if err != nil {
		return PreflightResult{}, err
	}
	canManage, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return PreflightResult{}, err
	}
	if !canManage {
		return PreflightResult{}, errors.New("not_authorized")
	}

	result := PreflightResult{
		ShowID: s.ID, ShowTitle: s.Title, Nickname: s.Nickname, ShortCode: s.ShortCode,
	}

	alreadyLive, venueSlug, _, err := loadLiveSessionForShow(ctx, pool, s.ID)
	if err != nil {
		return PreflightResult{}, err
	}
	result.IsLive = alreadyLive

	if !alreadyLive {
		derived, needsChoice, candidates, err := deriveVenueSlug(ctx, pool, s, true)
		if err != nil {
			return PreflightResult{}, err
		}
		if needsChoice {
			if len(candidates) > 1 {
				result.Blockers = append(result.Blockers, "venue_ambiguous")
			} else {
				result.Blockers = append(result.Blockers, "no_venue_derivable")
			}
		} else {
			venueSlug = derived
		}
	}

	if venueSlug != "" {
		result.VenueSlug = venueSlug
		result.VenueName, _ = loadVenueName(ctx, pool, venueSlug)
		if busy, err := venueBusyWithOtherShow(ctx, pool, venueSlug, s.ID); err != nil {
			return PreflightResult{}, err
		} else if busy {
			result.Blockers = append(result.Blockers, "venue_busy_with_other_show")
		}
		result.ChatBridgeReady, _ = identity.ChatBridgeReady(ctx, pool, venueSlug)
		if !result.ChatBridgeReady {
			result.Warnings = append(result.Warnings, "chat_bridge_not_ready")
		}
	}

	if s.CurrentShowScenePlacementID != nil {
		if placements, err := scenes.ListPlacementsForShow(ctx, pool, s.ID); err == nil {
			for _, p := range placements {
				if p.Placement.ID == *s.CurrentShowScenePlacementID {
					result.CurrentSceneName = p.Scene.Title
					break
				}
			}
		}
	}

	ready, needCharacter, err := countRosterCharacterReadiness(ctx, pool, s.ShowRunID)
	if err != nil {
		return PreflightResult{}, err
	}
	result.ReadyCharacterCount = ready
	result.IncompleteCharacterCount = needCharacter
	result.CastAdmittedCount = ready + needCharacter
	if needCharacter > 0 {
		result.Warnings = append(result.Warnings, "characters_incomplete")
	}

	ticketCount, err := countValidTickets(ctx, pool, s.ShowRunID)
	if err != nil {
		return PreflightResult{}, err
	}
	result.TicketCount = ticketCount
	if ticketCount == 0 {
		result.Warnings = append(result.Warnings, "no_tickets_issued")
	}

	return result, nil
}

// venueBusyWithOtherShow reports whether venueSlug already has an active
// session tied to a different Show -- the same signal
// shows.StartShowSession derives at Start time, read here without the
// write path.
func venueBusyWithOtherShow(ctx context.Context, pool *pgxpool.Pool, venueSlug, showID string) (bool, error) {
	var otherShowID *string
	err := pool.QueryRow(ctx, `
		SELECT sess.show_id::text FROM sessions sess
		JOIN venues v ON v.id = sess.venue_id
		WHERE v.slug = $1 AND sess.status IN ('rehearsal', 'live')
		ORDER BY sess.started_at DESC LIMIT 1
	`, venueSlug).Scan(&otherShowID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return otherShowID != nil && *otherShowID != showID, nil
}
