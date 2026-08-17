package stageobjects

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

// ErrObjectNotFound means the reference named a row that does not exist, no
// longer exists, or does not belong to the Show being addressed.
//
// Both the manual and the Cue path surface this rather than silently
// no-oping, which is §38's "invalid object target rejected" and §39's "Cue
// cannot target unsupported ephemeral effect". A Cue whose target has since
// been deleted therefore records a clean per-action failure (cues'
// executeOneAction already reports one) instead of appearing to succeed.
var ErrObjectNotFound = errors.New("stage_object_not_found")

// ErrObjectNotSupported means the row exists but this kind of row is
// deliberately outside the canonical visibility model.
var ErrObjectNotSupported = errors.New("stage_object_not_supported")

// Resolved is a validated reference: the row exists, belongs to the Show,
// and its dimensions are known.
type Resolved struct {
	Ref Ref
	// SupportsVisibility and SupportsInteraction are recomputed here from
	// the resolved row rather than trusted from the Ref, because a
	// scene_stage_element's eligibility depends on its stored kind.
	SupportsVisibility  bool
	SupportsInteraction bool
	// Label is a human-readable name for audit rows and Director UI. Never
	// used as identity (§22).
	Label string
}

// ResolveRef validates that ref names a real durable stage object owned by
// showID, and reports which state dimensions it supports.
//
// Every mutation path goes through here first. That is what stops a client
// from inventing an object id, from targeting an object in another Show, and
// from reaching the map through the generic object system.
func ResolveRef(ctx context.Context, q Querier, showID string, ref Ref) (Resolved, error) {
	showID = strings.TrimSpace(showID)
	if showID == "" {
		return Resolved{}, errors.New("show_id_required")
	}
	if err := ref.validate(); err != nil {
		return Resolved{}, err
	}
	n := ref.normalized()

	switch n.Kind {
	case KindVenueLayoutElement:
		return resolveVenueLayoutElement(ctx, q, showID, n)
	case KindSceneStageElement:
		return resolveSceneStageElement(ctx, q, showID, n)
	case KindDrawingObject:
		return resolveDrawingObject(ctx, q, showID, n)
	case KindParticipantInteraction:
		return resolveParticipantInteraction(ctx, q, showID, n)
	default:
		return Resolved{}, ErrObjectNotSupported
	}
}

// resolveVenueLayoutElement validates a live warehouse stage object.
//
// "Belongs to this Show" is necessarily indirect for this kind: elements are
// placed per VENUE (venue_layout_elements.venue_id), not per Show, because
// they predate Shows entirely. The honest containment check available is
// therefore "this Show has a session in that venue" -- which is exactly the
// same reachability the existing actions/reveal.go check expresses via
// sessions JOIN venues, just anchored on the Show rather than on one
// session id so state survives Kernel 70A's session replacement.
//
// Surface is required to be 'stage'. A non-stage element is chrome, not a
// stage object, and reveal.go already refuses those with "element is not
// revealable".
func resolveVenueLayoutElement(ctx context.Context, q Querier, showID string, ref Ref) (Resolved, error) {
	var name, elementType string
	err := q.QueryRow(ctx, `
		SELECT COALESCE(e.name, ''), e.element_type
		FROM elements e
		JOIN venue_layout_elements vle ON vle.element_id = e.id
		WHERE e.id = $1::uuid
		  AND LOWER(TRIM(vle.surface)) = 'stage'
		  AND vle.venue_id IN (
		    SELECT s.venue_id FROM sessions s WHERE s.show_id = $2::uuid
		  )
		LIMIT 1
	`, ref.ID, showID).Scan(&name, &elementType)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Resolved{}, ErrObjectNotFound
		}
		return Resolved{}, err
	}
	return Resolved{
		Ref:                ref,
		SupportsVisibility: true,
		// A live element is not itself interactable. Interaction lives on a
		// bound participant interaction, which is its own kind.
		SupportsInteraction: false,
		Label:               name,
	}, nil
}

// resolveSceneStageElement validates a Kernel 73A composition element and
// enforces Kernel 90 §3's map boundary.
//
// map_backdrop and grid_config are refused. The map keeps its own existing
// disappear/remove behavior, and changing Scene remains the intended way to
// radically change the stage -- §3 is explicit that per-Player or
// Cohort-specific map hiding must not be built here. grid_config is not an
// object a viewer perceives at all; it is stage configuration.
//
// Containment is checked through the Show's own placements, so a Scene
// element only resolves for a Show that has actually staged that Scene.
func resolveSceneStageElement(ctx context.Context, q Querier, showID string, ref Ref) (Resolved, error) {
	var kind, label string
	err := q.QueryRow(ctx, `
		SELECT sse.kind, COALESCE(sse.label, '')
		FROM scene_stage_elements sse
		WHERE sse.id = $1::uuid
		  AND sse.scene_id IN (
		    SELECT ssp.scene_id FROM show_scene_placements ssp
		    WHERE ssp.show_id = $2::uuid
		  )
		LIMIT 1
	`, ref.ID, showID).Scan(&kind, &label)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Resolved{}, ErrObjectNotFound
		}
		return Resolved{}, err
	}
	switch kind {
	case "token", "index_card":
		// Supported.
	default:
		// map_backdrop, grid_config, and anything a later kernel adds.
		return Resolved{}, ErrObjectNotSupported
	}
	return Resolved{
		Ref:                 ref,
		SupportsVisibility:  true,
		SupportsInteraction: false,
		Label:               label,
	}, nil
}

// resolveDrawingObject validates a Kernel 87 drawing object.
//
// Soft-deleted objects (deleted_at IS NOT NULL) resolve as not-found: a
// deleted drawing is gone, and hiding it would be meaningless. This is the
// §38 "stale/deleted object target handled cleanly" case.
func resolveDrawingObject(ctx context.Context, q Querier, showID string, ref Ref) (Resolved, error) {
	var objectType string
	err := q.QueryRow(ctx, `
		SELECT object_type
		FROM drawing_objects
		WHERE id = $1::uuid AND show_id = $2::uuid AND deleted_at IS NULL
	`, ref.ID, showID).Scan(&objectType)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Resolved{}, ErrObjectNotFound
		}
		return Resolved{}, err
	}
	return Resolved{
		Ref:                 ref,
		SupportsVisibility:  true,
		SupportsInteraction: false,
		Label:               "drawing " + objectType,
	}, nil
}

// resolveParticipantInteraction validates an interaction that has durable
// stage presence, which is the qualifier that matters (§5: "participant
// interaction with durable stage presence").
//
// An interaction earns a place in the canonical model by being bound to a
// placed stage element through stage_element_bindings. An unbound
// participant interaction -- one reached only through a Program, with no
// object on the stage -- is refused, because there is nothing on stage whose
// perception could be scoped. §20 asked for that boundary to be documented:
// this function is where it is drawn, and unbound interactions keep their
// existing domain behavior untouched.
func resolveParticipantInteraction(ctx context.Context, q Querier, showID string, ref Ref) (Resolved, error) {
	var label string
	err := q.QueryRow(ctx, `
		SELECT COALESCE(NULLIF(pi.stage_button_label, ''), pi.interaction_type)
		FROM participant_interactions pi
		JOIN stage_element_bindings seb ON seb.participant_interaction_id = pi.id
		JOIN scene_stage_elements sse ON sse.id = seb.scene_stage_element_id
		WHERE pi.id = $1::uuid
		  AND sse.scene_id IN (
		    SELECT ssp.scene_id FROM show_scene_placements ssp
		    WHERE ssp.show_id = $2::uuid
		  )
		LIMIT 1
	`, ref.ID, showID).Scan(&label)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Resolved{}, ErrObjectNotFound
		}
		return Resolved{}, err
	}
	return Resolved{
		Ref: ref,
		// Visibility is NOT settable here: a bound interaction is perceived
		// exactly when its element is, and a second visibility switch for
		// the same pixels is the competing-truth failure §20 forbids.
		SupportsVisibility:  false,
		SupportsInteraction: true,
		Label:               label,
	}, nil
}

// validateScopeTargets checks that every cohort/Character a grant names
// actually belongs to this Show, so a Director cannot grant visibility to a
// cohort from someone else's production and a client cannot probe for ids
// by trial (§17: do not trust a client-supplied Character id as authority).
func validateScopeTargets(ctx context.Context, q Querier, showID string, scopes []Scope) error {
	for _, raw := range scopes {
		s := raw.normalized()
		if err := s.validate(); err != nil {
			return err
		}
		switch s.Kind {
		case ScopeCohort:
			var ok bool
			if err := q.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1 FROM show_cohorts WHERE id = $1::uuid AND show_id = $2::uuid
				)
			`, s.ID, showID).Scan(&ok); err != nil {
				return err
			}
			if !ok {
				return errors.New("cohort_not_in_show")
			}
		case ScopeCharacter:
			// A Character qualifies if it is on this Show's roster. Resolved
			// through the Show Run roster (the Kernel 83+ authority) rather
			// than through current_session_personas, which §32 names as
			// legacy authority to contain rather than depend on.
			var ok bool
			if err := q.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1
					FROM show_run_roster_members srrm
					JOIN shows sh ON sh.show_run_id = srrm.show_run_id
					WHERE sh.id = $2::uuid
					  AND srrm.character_card_id = $1::uuid
					  AND srrm.removed_at IS NULL
				)
			`, s.ID, showID).Scan(&ok); err != nil {
				return err
			}
			if !ok {
				return errors.New("character_not_in_show")
			}
		}
	}
	return nil
}
