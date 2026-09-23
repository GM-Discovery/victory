// Package participation is the Kernel 71 canonical answer to "what is this
// user's role/access here." Before this package existed, four independent
// tables each answered a slightly different question with no agreed
// precedence: location_memberships (Kernel 66+, location-scoped, the source
// Show-Run/Cue authority already trusts), the older memberships table
// (location+production+venue-scoped, under-populated by newer code paths),
// access_grants (a boolean venue/production reach grant, no role), and
// show_run_roster_members (show-run-scoped, its own role vocabulary). The
// concrete bug this closed: main.go's lookupVenueRole read memberships only,
// so a Producer/Director whose sole grant was a location_memberships row
// (the normal shape for anyone bootstrapped or signed up after Kernel 66)
// resolved as viewerRole "none" at their own venue.
package participation

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
)

// ViewerMode is the resolver's single verdict on how a user relates to
// whatever location/show-run/venue was resolved from the hints given to
// ResolveParticipationContext.
type ViewerMode string

const (
	ViewerModeOperator ViewerMode = "operator"
	ViewerModeProducer ViewerMode = "producer"
	ViewerModeDirector ViewerMode = "director"
	ViewerModeCrew     ViewerMode = "crew"
	ViewerModeCast     ViewerMode = "cast"
	ViewerModePlayer   ViewerMode = "player"
	ViewerModeAudience ViewerMode = "audience"
	ViewerModeNone     ViewerMode = "none"
)

// Context is the resolver's full verdict. Every field is best-effort filled
// from whichever of venueSlug/showRunID was given to
// ResolveParticipationContext -- callers that only have one hint still get
// everything derivable from it.
type Context struct {
	UserID       string
	LocationID   string
	LocationRole string // raw location_memberships role, "" if none
	ShowRunID    string
	ShowRunRole  string // raw show_run_roster_members role, "" if none

	ViewerMode       ViewerMode
	CanEnterVenue    bool
	CanParticipate   bool
	CanManage        bool
	CanViewBackstage bool
	Reason           string
}

// ResolveParticipationContext is the one function every venue-role /
// theater-entry / backstage-gate call site should converge on. venueSlug
// and showRunID are optional narrowing hints -- pass "" for whichever isn't
// known; the resolver derives locationID from whichever hint is given.
//
// Precedence (highest wins; nothing lower can ever downgrade a role a
// higher step already found):
//  1. Operator (access.IsOperatorUser) -- full authority.
//  2. Active location_memberships row at the resolved location.
//  3. Active show_run_roster_members row at the resolved show run.
//  4. Fallback: any active location_memberships row at all -> audience.
//  5. Only reached if 1-4 all yielded "none": legacy memberships/
//     access_grants, read as an additive upgrade only. This is the exact
//     mechanism that stops a missing legacy row from downgrading a valid
//     canonical role, and stops a present legacy row from ever overriding
//     one -- step 5 never runs unless every canonical step above it found
//     nothing.
func ResolveParticipationContext(ctx context.Context, pool *pgxpool.Pool, userID, venueSlug, showRunID string) (Context, error) {
	userID = strings.TrimSpace(userID)
	venueSlug = strings.ToLower(strings.TrimSpace(venueSlug))
	showRunID = strings.TrimSpace(showRunID)

	result := Context{UserID: userID, ShowRunID: showRunID, ViewerMode: ViewerModeNone}
	if userID == "" {
		return result, nil
	}

	locationID, err := resolveLocationID(ctx, pool, venueSlug, showRunID)
	if err != nil {
		return Context{}, err
	}
	result.LocationID = locationID

	// Step 1: Operator.
	if ok, err := access.IsOperatorUser(ctx, pool, userID); err != nil {
		return Context{}, err
	} else if ok {
		result.ViewerMode = ViewerModeOperator
		result.CanManage = true
		result.CanViewBackstage = true
		result.CanEnterVenue = true
		result.CanParticipate = true
		result.Reason = "operator"
		return result, nil
	}

	// Step 2: location_memberships (canonical location-scoped role).
	if locationID != "" {
		role, err := access.CurrentLocationRoleForLocation(ctx, pool, userID, locationID)
		if err != nil {
			return Context{}, err
		}
		if role != "audience" {
			result.LocationRole = role
		}
		switch role {
		case "producer", "director":
			result.ViewerMode = ViewerMode(role)
			result.CanManage = true
			result.CanViewBackstage = true
			result.CanEnterVenue = true
			result.CanParticipate = true
			result.Reason = "location_membership"
		case "crew":
			// Location-level Crew gets the same full-backstage-state
			// treatment as a show-run-roster Crew row below (Step 3) --
			// the product model (Docs/Product/Glossary.md) is explicit
			// that Crew always sees full backstage state, never gated on
			// a specific show run's roster.
			result.ViewerMode = ViewerModeCrew
			result.CanViewBackstage = true
			result.CanEnterVenue = true
			result.CanParticipate = true
			result.Reason = "location_membership"
		case "cast":
			// Kernel 101 101-19/101-20/101-22 bug closure: this switch used
			// to only recognize "producer"/"director", so a location-level
			// Cast account (approved via Audition Hall, the exact Kernel
			// 101-17 authority) with no roster seat in the venue's
			// currently-live Show Run fell through every step below to
			// Step 4's blanket "any active location_memberships row at all
			// -> audience" fallback -- misresolving Cast as Audience for
			// every /api/world/* snapshot (world.LoadVenueSnapshot's
			// viewerRole, which resolveTheaterContext and its
			// theaterContextBackstageTier consume), even though the
			// session-join endpoint correctly reported "cast". Deliberately
			// NOT granted CanViewBackstage here -- hidden-object perception
			// stays gated on actual Show Run roster membership
			// (stageobjects.isShowCast), never on the bare location role.
			result.ViewerMode = ViewerModeCast
			result.CanEnterVenue = true
			result.CanParticipate = true
			result.Reason = "location_membership"
		}
	}

	// Step 3: show_run_roster_members (canonical show-run-scoped role).
	// Can only elevate a viewer who wasn't already resolved to
	// producer/director/crew by step 2 -- a Show Run crew/player row never
	// downgrades a location-level Producer/Director, and (since location
	// Crew was added to step 2 above) never downgrades a location-level
	// Crew member to mere "player" just because this specific show run's
	// roster happens to list them as a player too -- Crew always keeps
	// full backstage state per the product model. Location-level Cast has
	// no such protection: a "player"/"crew"/"producer"/"director" roster
	// role for THIS show run is always a promotion or a lateral,
	// same-capability move from Cast, never a downgrade.
	if showRunID != "" {
		rosterRole, err := activeRosterRole(ctx, pool, userID, showRunID)
		if err != nil {
			return Context{}, err
		}
		result.ShowRunRole = rosterRole
		if result.ViewerMode != ViewerModeProducer && result.ViewerMode != ViewerModeDirector && result.ViewerMode != ViewerModeCrew {
			switch rosterRole {
			case "producer", "director":
				result.ViewerMode = ViewerMode(rosterRole)
				result.CanManage = true
				result.CanViewBackstage = true
				result.CanEnterVenue = true
				result.CanParticipate = true
				result.Reason = "show_run_roster"
			case "crew":
				result.ViewerMode = ViewerModeCrew
				result.CanViewBackstage = true
				result.CanEnterVenue = true
				result.CanParticipate = true
				result.Reason = "show_run_roster"
			case "player":
				result.ViewerMode = ViewerModePlayer
				result.CanParticipate = true
				result.CanEnterVenue = true
				result.Reason = "show_run_roster"
			}
		}
	}

	// Step 3.5 (Kernel 93): a pure Audience account with zero
	// location_memberships rows -- the exact shape Kernel 90 found could not
	// enter Catharsis at all -- resolves here via its Showing-scoped
	// audience_admissions row, not the location-membership fallback below.
	// Scoped to venueSlug's own currently live/rehearsal session, mirroring
	// access.ResolveVisibleVenues's Catharsis branch exactly so the two
	// checks can never disagree about which Showing admits this user.
	if result.ViewerMode == ViewerModeNone && venueSlug != "" {
		admitted, err := audienceAdmissionGrantsVenueEntry(ctx, pool, userID, venueSlug)
		if err != nil {
			return Context{}, err
		}
		if admitted {
			result.ViewerMode = ViewerModeAudience
			result.CanEnterVenue = true
			result.Reason = "audience_admission"
		}
	}

	// Step 4: fallback -- any active location_memberships row at all
	// (including a plain "audience" role) means the user is a known
	// audience member at this location, not a total stranger.
	if result.ViewerMode == ViewerModeNone && locationID != "" {
		has, err := access.HasActiveLocationMembership(ctx, pool, userID, locationID)
		if err != nil {
			return Context{}, err
		}
		if has {
			result.ViewerMode = ViewerModeAudience
			result.CanEnterVenue = true
			result.Reason = "location_membership_audience"
		}
	}

	// Step 5: legacy compatibility read. Only reached if every canonical
	// step above yielded "none" -- a legacy row can upgrade a stranger
	// into something, but can never override a canonical role already
	// found above.
	if result.ViewerMode == ViewerModeNone {
		legacyMode, err := legacyRole(ctx, pool, userID, venueSlug)
		if err != nil {
			return Context{}, err
		}
		if legacyMode != "" {
			result.ViewerMode = ViewerMode(legacyMode)
			result.CanEnterVenue = true
			if legacyMode == "producer" || legacyMode == "director" {
				result.CanManage = true
				result.CanViewBackstage = true
			}
			result.Reason = "legacy_membership_compat"
		}
	}

	return result, nil
}

// LegacyLookupVenueRole is a drop-in replacement for main.go's original
// lookupVenueRole (and network/ws.go's dead duplicate of it) -- same
// signature, same "none" no-match convention consumed by
// world.LoadVenueSnapshot/LoadCaveSnapshot, but backed by the canonical
// resolver instead of a memberships-only query. Operator resolves to
// "producer" here (rather than a distinct "operator" string) so every
// existing consumer of this return value keeps working unchanged --
// Context.ViewerMode is where callers that need to distinguish Operator
// specifically should look instead.
//
// Kernel 97: this used to always pass showRunID="" to
// ResolveParticipationContext, which skips Step 3 (show_run_roster_members)
// entirely -- the ONLY step that resolves a real roster Cast/Crew/Player
// role, per that function's own precedence order. A roster Player with no
// location_memberships row (the normal case) fell through to Step 4's bare
// "audience" fallback, misclassified exactly as spec Sec5 describes, on
// every /api/world/* snapshot (the-cave, catharsis, first-theater -- every
// live caller of this function). Resolving whichever Show Run currently has
// a live/rehearsal session at this venue fixes it without requiring the
// three call sites in main.go to change at all.
func LegacyLookupVenueRole(ctx context.Context, pool *pgxpool.Pool, userID, venueSlug string) (string, error) {
	showRunID, err := resolveActiveShowRunIDForVenue(ctx, pool, venueSlug)
	if err != nil {
		return "", err
	}
	result, err := ResolveParticipationContext(ctx, pool, userID, venueSlug, showRunID)
	if err != nil {
		return "", err
	}
	switch result.ViewerMode {
	case ViewerModeNone, "":
		return "none", nil
	case ViewerModeOperator:
		return "producer", nil
	default:
		return string(result.ViewerMode), nil
	}
}

// resolveActiveShowRunIDForVenue finds the Show Run currently live or in
// rehearsal at the given venue, if any -- "" (no error) if there isn't one,
// which is the normal case for a venue with nothing currently staged.
func resolveActiveShowRunIDForVenue(ctx context.Context, pool *pgxpool.Pool, venueSlug string) (string, error) {
	venueSlug = strings.ToLower(strings.TrimSpace(venueSlug))
	if venueSlug == "" {
		return "", nil
	}
	var showRunID string
	err := pool.QueryRow(ctx, `
		SELECT sh.show_run_id::text
		FROM sessions se
		JOIN venues v ON v.id = se.venue_id
		JOIN shows sh ON sh.id = se.show_id
		WHERE v.slug = $1 AND se.status IN ('rehearsal', 'live')
		ORDER BY se.started_at DESC
		LIMIT 1
	`, venueSlug).Scan(&showRunID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return showRunID, nil
}

func resolveLocationID(ctx context.Context, pool *pgxpool.Pool, venueSlug, showRunID string) (string, error) {
	if venueSlug != "" {
		var locationID string
		err := pool.QueryRow(ctx, `
			SELECT l.location_id::text
			FROM venues v
			JOIN lots l ON l.id = v.lot_id
			WHERE v.slug = $1
			LIMIT 1
		`, venueSlug).Scan(&locationID)
		if err == nil {
			return locationID, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return "", err
		}
	}
	if showRunID != "" {
		var locationID string
		err := pool.QueryRow(ctx, `SELECT location_id::text FROM show_runs WHERE id = $1`, showRunID).Scan(&locationID)
		if err == nil {
			return locationID, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return "", err
		}
	}
	return "", nil
}

// activeRosterRole mirrors cues/authority.go's private helper of the same
// name (small, deliberate duplication of a three-line query rather than a
// new cross-package dependency, matching this codebase's established
// convention -- see shows/session_resume_test.go's fixture-duplication
// note for the same rule applied elsewhere).
func activeRosterRole(ctx context.Context, pool *pgxpool.Pool, userID, showRunID string) (string, error) {
	var role string
	err := pool.QueryRow(ctx, `
		SELECT role FROM show_run_roster_members
		WHERE show_run_id = $1 AND user_id = $2 AND removed_at IS NULL
		LIMIT 1
	`, showRunID, userID).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return role, nil
}

// legacyMembershipsRole is main.go's original lookupVenueRole query, kept
// verbatim as a private compatibility fallback so its behavior for
// still-legacy-only accounts is unchanged -- only its position in the
// overall precedence changed (last resort, not first and only resort).
func legacyMembershipsRole(ctx context.Context, pool *pgxpool.Pool, userID, venueSlug string) (string, error) {
	if venueSlug == "" {
		return "", nil
	}
	var role string
	err := pool.QueryRow(ctx, `
		SELECT lower(role_text) FROM (
			SELECT m.role::text AS role_text, 1 AS priority
			FROM memberships m
			JOIN venues v ON v.id = m.venue_id
			WHERE m.user_id = $1
			  AND v.slug = $2

			UNION ALL

			SELECT m.role::text AS role_text, 2 AS priority
			FROM memberships m
			WHERE m.user_id = $1
			  AND m.venue_id IS NULL
			  AND m.role::text = 'producer'
		) ranked
		ORDER BY priority
		LIMIT 1
	`, userID, venueSlug).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return role, nil
}

// legacyAccessGrantExists reports an unrevoked, unexpired access_grants row
// for this user at this venue. access_grants carries no role of its own --
// its presence only ever upgrades a stranger to generic venue-entry
// audience, never to a managed role.
func legacyAccessGrantExists(ctx context.Context, pool *pgxpool.Pool, userID, venueSlug string) (bool, error) {
	if venueSlug == "" {
		return false, nil
	}
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM access_grants ag
			JOIN venues v ON v.id = ag.venue_id
			WHERE ag.user_id = $1
			  AND v.slug = $2
			  AND ag.revoked_at IS NULL
			  AND (ag.expires_at IS NULL OR ag.expires_at > NOW())
		)
	`, userID, venueSlug).Scan(&exists)
	return exists, err
}

// audienceAdmissionGrantsVenueEntry duplicates
// access.ResolveVisibleVenues's Kernel 93 Catharsis UNION branch as a
// single-venue boolean check (small, deliberate per-package duplication of
// a three-line query, matching this codebase's established convention --
// see activeRosterRole's comment above for the same rule applied
// elsewhere) rather than importing access's map-tile query, which answers a
// different question (every visible venue) than the one this resolver
// needs (is this one venue enterable).
func audienceAdmissionGrantsVenueEntry(ctx context.Context, pool *pgxpool.Pool, userID, venueSlug string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM venues v
			JOIN sessions s ON s.venue_id = v.id AND s.status IN ('rehearsal', 'live')
			JOIN showings sh ON sh.session_id = s.id
			JOIN audience_admissions aa ON aa.showing_id = sh.id
			WHERE v.slug = $2
			  AND aa.user_id = $1
		)
	`, userID, venueSlug).Scan(&exists)
	return exists, err
}

func legacyRole(ctx context.Context, pool *pgxpool.Pool, userID, venueSlug string) (string, error) {
	if role, err := legacyMembershipsRole(ctx, pool, userID, venueSlug); err != nil {
		return "", err
	} else if role != "" {
		return role, nil
	}
	if hasGrant, err := legacyAccessGrantExists(ctx, pool, userID, venueSlug); err != nil {
		return "", err
	} else if hasGrant {
		return "audience", nil
	}
	return "", nil
}
