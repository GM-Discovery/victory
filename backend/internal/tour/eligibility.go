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
