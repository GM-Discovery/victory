package tour

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
	"victory/backend/internal/venues"
)

// roleEligible reports whether the caller's resolved role/operator status
// satisfies a Definition's RequiredRoles. Empty RequiredRoles means venue
// access alone is the gate.
func roleEligible(requiredRoles []string, role string, isOperator bool) bool {
	if len(requiredRoles) == 0 {
		return true
	}
	for _, want := range requiredRoles {
		if want == "operator" {
			if isOperator {
				return true
			}
			continue
		}
		if want == role {
			return true
		}
	}
	return false
}

// resolveRole resolves the caller's server-authoritative role for a
// venue-scoped Definition (kernel-91 S37: eligibility must use canonical
// role/access truth, never a client-supplied role). Campus-scoped
// Definitions (VenueSlug == "") have no location to resolve against, so
// role is irrelevant there -- RequiredRoles is expected to be empty for
// those and roleEligible short-circuits true.
func resolveRole(ctx context.Context, pool *pgxpool.Pool, userID, venueSlug string) (role string, isOperator bool, err error) {
	isOperator, err = access.IsOperatorUser(ctx, pool, userID)
	if err != nil {
		return "", false, err
	}
	if strings.TrimSpace(venueSlug) == "" {
		return "", isOperator, nil
	}
	_, locationID, err := venues.ResolveVenueLocation(ctx, pool, venueSlug)
	if err != nil {
		// Venue not found/not resolvable: no role to grant beyond operator.
		return "", isOperator, nil
	}
	role, err = access.CurrentLocationRoleForLocation(ctx, pool, userID, locationID)
	if err != nil {
		return "", isOperator, err
	}
	return role, isOperator, nil
}

// EligibleTours returns the Definitions subj may currently see for the given
// venue context (empty venueSlug = campus/map scope), excluding any already
// completed or skipped. This is the one place role/venue eligibility is
// decided (kernel-91 S37, S46) -- the frontend engine trusts this list
// completely and contains no eligibility logic of its own.
func EligibleTours(ctx context.Context, pool *pgxpool.Pool, subj Subject, venueSlug string) ([]Definition, error) {
	if err := subj.valid(); err != nil {
		return nil, err
	}
	venueSlug = strings.TrimSpace(venueSlug)

	role, isOperator, err := resolveRole(ctx, pool, subj.UserID, venueSlug)
	if err != nil {
		return nil, err
	}

	var out []Definition
	for _, key := range allTourKeys {
		def := Definitions[key]
		if def.VenueSlug != venueSlug {
			continue
		}
		if !roleEligible(def.RequiredRoles, role, isOperator) {
			continue
		}
		if key == KeyCampusContinuation {
			// The continuation only becomes relevant after the mandatory
			// campus tour is behind the user (kernel-91 S3: "first return to
			// campus"). Checked here in Go, not as a DB foreign key, since
			// it's a sequencing rule rather than a referential constraint.
			done, err := HasCompletion(ctx, pool, subj, KeyCampusMandatory, "", "")
			if err != nil {
				return nil, err
			}
			if !done {
				continue
			}
			// Kernel 93 Pass C: finishing the mandatory tour says nothing
			// about whether Catharsis is actually enterable yet -- pointing
			// at a venue the user has no admission/access to is exactly the
			// "force-routed to Catharsis with no permission" bug this
			// closes. Reuse the same access_grants-or-audience_admissions
			// check the map itself uses to decide tile visibility (see
			// access.ResolveVisibleVenues), not location_memberships' role
			// lookup -- that one defaults every user to "audience" and would
			// never actually gate anything here.
			canAccess, err := access.UserCanAccessVenueSlug(ctx, pool, subj.UserID, "catharsis")
			if err != nil {
				return nil, err
			}
			if !canAccess {
				continue
			}
		}
		if key == KeyGreenroomIntro {
			// Kernel 93 Pass C: the Greenroom pin ("your Character workbooks
			// live here") is only relevant once there is a workbook to point
			// at -- reuse the exact condition that already makes the
			// Greenroom map tile itself visible (access.ResolveVisibleVenues'
			// owned_workbook_surface branch: any non-deleted character_cards
			// row, not scoped to a particular venue's location since
			// character creation only happens at Catharsis today anyway)
			// rather than a second hand-rolled copy of the same check that
			// could drift out of sync with it. See migration 112's split of
			// this out of campus_continuation.
			hasCharacter, err := access.UserCanAccessVenueSlug(ctx, pool, subj.UserID, "greenroom")
			if err != nil {
				return nil, err
			}
			if !hasCharacter {
				continue
			}
		}
		roleKey := roleKeyForDefinition(def, role, isOperator)
		done, err := HasCompletion(ctx, pool, subj, key, def.VenueSlug, roleKey)
		if err != nil {
			return nil, err
		}
		if done {
			continue
		}
		resumed, err := resumeFromProgress(ctx, pool, subj, def, roleKey)
		if err != nil {
			return nil, err
		}
		out = append(out, resumed)
	}
	return out, nil
}

// resumeFromProgress slices a Definition's Steps to start after the last
// step_reached recorded in tour_progress, so a click-gated step whose
// target navigates away (Audition Hall, Trailer) resumes on return instead
// of restarting the whole tour from step 0 -- see migration 108's comment
// for the production bug this closes. If there's no recorded progress, or
// the recorded step no longer matches any step in this Definition (content
// changed under a stale cursor), the full step list is returned unchanged.
func resumeFromProgress(ctx context.Context, pool *pgxpool.Pool, subj Subject, def Definition, roleKey string) (Definition, error) {
	stepReached, err := LoadProgressStep(ctx, pool, subj, def.Key, def.VenueSlug, roleKey)
	if err != nil {
		return Definition{}, err
	}
	if stepReached == "" {
		return def, nil
	}
	for i, step := range def.Steps {
		if step.Key == stepReached {
			resumed := def
			resumed.Steps = def.Steps[i+1:]
			return resumed, nil
		}
	}
	return def, nil
}

// roleKeyForDefinition is the role_key value a completion for this
// Definition is recorded/looked-up under: the specific role the caller
// currently holds when the tour is role-gated, empty otherwise. Using the
// caller's actual role (rather than joining every RequiredRoles value) keeps
// a Director's completion distinct from an Operator's, even though both may
// satisfy the same Definition.
func roleKeyForDefinition(def Definition, role string, isOperator bool) string {
	if len(def.RequiredRoles) == 0 {
		return ""
	}
	if isOperator {
		for _, want := range def.RequiredRoles {
			if want == "operator" {
				return "operator"
			}
		}
	}
	return role
}

// ResumeMandatory returns the mandatory campus tour if subj has not yet
// completed it, else nil. Mandatory tours have no skip-bypass (kernel-91
// S15): HasCompletion with status='completed' is the only way this clears,
// enforced in http.go's skip handler rejecting Mandatory tour keys outright.
func ResumeMandatory(ctx context.Context, pool *pgxpool.Pool, subj Subject) (*Definition, error) {
	if err := subj.valid(); err != nil {
		return nil, err
	}
	done, err := HasCompletion(ctx, pool, subj, KeyCampusMandatory, "", "")
	if err != nil {
		return nil, err
	}
	if done {
		return nil, nil
	}
	def := Definitions[KeyCampusMandatory]
	resumed, err := resumeFromProgress(ctx, pool, subj, def, "")
	if err != nil {
		return nil, err
	}
	return &resumed, nil
}
