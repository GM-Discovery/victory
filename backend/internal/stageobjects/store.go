package stageobjects

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Operation names the canonical state change being requested. These four are
// exactly Kernel 90 §21's required Cue actions and exactly §11's manual
// Director operations, which is the point: one vocabulary, so "the Director
// hid it" and "a Cue hid it" are not two different verbs that happen to
// agree.
const (
	OpHide              = "hide_object"
	OpReveal            = "reveal_object"
	OpEnableInteraction = "enable_interaction"
	OpDisableInteraction = "disable_interaction"
	// OpSetScopes replaces an object's grant set. Manual-only: §21 does not
	// list a scope Cue action, and inventing one would put a scope model into
	// Cue payloads before a Director has ever asked for it.
	OpSetScopes = "set_scopes"
)

// Mutation is one requested change to canonical stage-object state.
type Mutation struct {
	ShowID  string
	Ref     Ref
	Op      string
	ActorID string
	// Scopes is read only by OpSetScopes and by OpHide (where it is the
	// convenience form of "hide, and reveal to these"). Nil means "leave the
	// existing grants alone"; an explicitly empty non-nil slice clears them.
	Scopes []Scope
	scopesSet bool
}

// WithScopes returns m with an explicit grant set, distinguishing "clear all
// grants" (empty slice) from "do not touch grants" (never called).
func (m Mutation) WithScopes(scopes []Scope) Mutation {
	m.Scopes = scopes
	m.scopesSet = true
	return m
}

func validOp(op string) bool {
	switch op {
	case OpHide, OpReveal, OpEnableInteraction, OpDisableInteraction, OpSetScopes:
		return true
	default:
		return false
	}
}

// ApplyMutation is THE canonical state write. Kernel 90 §24's manual/Cue
// parity is not a test that compares two code paths -- there is one code
// path, and both callers reach it here.
//
// It runs in a transaction so a hide-plus-grants change is never observed
// half-applied by a concurrently reconnecting viewer's projection read.
//
// Authority is NOT checked here. Callers check it: the HTTP handler via
// RequireDirector, and Cue execution via cues.CanTriggerCue, which is
// already the stronger, Cue-specific gate (a Cue may be Director-only or
// player-triggerable by its own trigger_scope, and re-deriving that here
// would either duplicate or contradict it). This function refuses to run
// without a non-empty ActorID so a caller cannot forget to establish one.
func ApplyMutation(ctx context.Context, pool *pgxpool.Pool, m Mutation) (State, error) {
	m.ShowID = strings.TrimSpace(m.ShowID)
	m.ActorID = strings.TrimSpace(m.ActorID)
	m.Op = strings.ToLower(strings.TrimSpace(m.Op))

	if m.ShowID == "" {
		return State{}, errors.New("show_id_required")
	}
	if m.ActorID == "" {
		return State{}, errors.New("actor_required")
	}
	if !validOp(m.Op) {
		return State{}, errors.New("unsupported_stage_object_operation")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return State{}, err
	}
	defer tx.Rollback(ctx)

	resolved, err := ResolveRef(ctx, tx, m.ShowID, m.Ref)
	if err != nil {
		return State{}, err
	}

	switch m.Op {
	case OpHide, OpReveal:
		if !resolved.SupportsVisibility {
			return State{}, ErrObjectNotSupported
		}
	case OpEnableInteraction, OpDisableInteraction:
		if !resolved.SupportsInteraction {
			return State{}, ErrObjectNotSupported
		}
	case OpSetScopes:
		if !resolved.SupportsVisibility {
			return State{}, ErrObjectNotSupported
		}
	}

	if m.scopesSet {
		if err := validateScopeTargets(ctx, tx, m.ShowID, m.Scopes); err != nil {
			return State{}, err
		}
	}

	stateID, err := upsertStateRow(ctx, tx, m, resolved)
	if err != nil {
		return State{}, err
	}

	if m.scopesSet {
		if err := replaceScopes(ctx, tx, stateID, m); err != nil {
			return State{}, err
		}
	}

	recordAudit(ctx, tx, m.ShowID, m.ActorID, m.Op, m.Ref.normalized(), resolved)

	out, err := loadStateByID(ctx, tx, stateID)
	if err != nil {
		return State{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return State{}, err
	}
	return out, nil
}

// upsertStateRow writes the state row, creating it on first non-default use.
//
// Each operation touches only its own dimension. That is what keeps §8's
// "hidden is not disabled, disabled is not hidden" true in storage rather
// than only in documentation: disabling an interaction cannot accidentally
// clear a visibility state, and revealing an object cannot re-enable an
// interaction the Director switched off.
func upsertStateRow(ctx context.Context, tx pgx.Tx, m Mutation, resolved Resolved) (string, error) {
	ref := m.Ref.normalized()

	// The seed values used only when the row does not exist yet, chosen so
	// an absent row and a freshly created one describe the same object.
	seedVisibility := VisibilityVisible
	var seedInteraction *bool
	if resolved.SupportsInteraction {
		enabled := true
		seedInteraction = &enabled
	}

	switch m.Op {
	case OpHide:
		seedVisibility = VisibilityHidden
	case OpEnableInteraction:
		enabled := true
		seedInteraction = &enabled
	case OpDisableInteraction:
		disabled := false
		seedInteraction = &disabled
	}

	// visibility is updated only by the visibility operations, and
	// interaction_enabled only by the interaction operations; every other
	// operation preserves the stored value via the EXCLUDED-vs-existing
	// CASE below.
	updateVisibility := m.Op == OpHide || m.Op == OpReveal
	updateInteraction := m.Op == OpEnableInteraction || m.Op == OpDisableInteraction

	var stateID string
	err := tx.QueryRow(ctx, `
		INSERT INTO stage_object_states (
			show_id, object_kind, object_id, visibility, interaction_enabled, updated_by_user_id
		)
		VALUES ($1::uuid, $2, $3::uuid, $4, $5, $6::uuid)
		ON CONFLICT (show_id, object_kind, object_id) DO UPDATE
		SET visibility = CASE WHEN $7 THEN EXCLUDED.visibility ELSE stage_object_states.visibility END,
		    interaction_enabled = CASE WHEN $8 THEN EXCLUDED.interaction_enabled ELSE stage_object_states.interaction_enabled END,
		    updated_by_user_id = EXCLUDED.updated_by_user_id,
		    updated_at = NOW()
		RETURNING id::text
	`, m.ShowID, ref.Kind, ref.ID, seedVisibility, seedInteraction, m.ActorID,
		updateVisibility, updateInteraction).Scan(&stateID)
	return stateID, err
}

// replaceScopes swaps an object's whole grant set atomically. Replace rather
// than merge, because the Director UI edits a set ("who may see this?") and
// an add-only API would make removing a Cohort impossible without a separate
// delete endpoint per grant.
func replaceScopes(ctx context.Context, tx pgx.Tx, stateID string, m Mutation) error {
	if _, err := tx.Exec(ctx, `
		DELETE FROM stage_object_scope_grants WHERE stage_object_state_id = $1::uuid
	`, stateID); err != nil {
		return err
	}
	seen := map[Scope]bool{}
	for _, raw := range m.Scopes {
		s := raw.normalized()
		if seen[s] {
			continue
		}
		seen[s] = true
		if _, err := tx.Exec(ctx, `
			INSERT INTO stage_object_scope_grants (
				stage_object_state_id, scope_kind, scope_id, created_by_user_id
			)
			VALUES ($1::uuid, $2, NULLIF($3, '')::uuid, $4::uuid)
		`, stateID, s.Kind, s.ID, m.ActorID); err != nil {
			return err
		}
	}
	return nil
}

func loadStateByID(ctx context.Context, q Querier, stateID string) (State, error) {
	var st State
	var updatedBy *string
	var visibility string
	var interaction *bool
	var kind, objectID string
	var updatedAt time.Time
	if err := q.QueryRow(ctx, `
		SELECT id::text, show_id::text, object_kind, object_id::text, visibility,
		       interaction_enabled, updated_by_user_id::text, updated_at
		FROM stage_object_states WHERE id = $1::uuid
	`, stateID).Scan(&st.ID, &st.ShowID, &kind, &objectID, &visibility,
		&interaction, &updatedBy, &updatedAt); err != nil {
		return State{}, err
	}
	st.Ref = Ref{Kind: kind, ID: objectID}
	st.Visibility = visibility
	st.InteractionEnabled = interaction
	st.UpdatedAt = updatedAt
	if updatedBy != nil {
		st.UpdatedByUserID = *updatedBy
	}

	rows, err := q.Query(ctx, `
		SELECT scope_kind, COALESCE(scope_id::text, '')
		FROM stage_object_scope_grants
		WHERE stage_object_state_id = $1::uuid
		ORDER BY scope_kind, scope_id
	`, stateID)
	if err != nil {
		return State{}, err
	}
	defer rows.Close()
	st.Scopes = []Scope{}
	for rows.Next() {
		var s Scope
		if err := rows.Scan(&s.Kind, &s.ID); err != nil {
			return State{}, err
		}
		st.Scopes = append(st.Scopes, s)
	}
	return st, rows.Err()
}

// LoadShowStates returns every non-default state row for a Show, keyed by
// Ref. One query for states plus one for grants regardless of how many
// objects the Show has, because this runs on the snapshot path -- which is
// already the hottest read in the product.
//
// This is a raw canonical read with no viewer filtering. It is Director-only
// data (it describes what is hidden and from whom) and must never be handed
// to a Player-facing response; ProjectorFor is the viewer-safe entry point.
func LoadShowStates(ctx context.Context, q Querier, showID string) (map[Ref]State, error) {
	showID = strings.TrimSpace(showID)
	if showID == "" {
		return map[Ref]State{}, nil
	}

	rows, err := q.Query(ctx, `
		SELECT id::text, object_kind, object_id::text, visibility,
		       interaction_enabled, COALESCE(updated_by_user_id::text, ''), updated_at
		FROM stage_object_states
		WHERE show_id = $1::uuid
	`, showID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byRef := map[Ref]State{}
	byID := map[string]Ref{}
	for rows.Next() {
		var st State
		var kind, objectID string
		if err := rows.Scan(&st.ID, &kind, &objectID, &st.Visibility,
			&st.InteractionEnabled, &st.UpdatedByUserID, &st.UpdatedAt); err != nil {
			return nil, err
		}
		st.ShowID = showID
		st.Ref = Ref{Kind: kind, ID: objectID}
		st.Scopes = []Scope{}
		byRef[st.Ref] = st
		byID[st.ID] = st.Ref
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(byRef) == 0 {
		return byRef, nil
	}

	grantRows, err := q.Query(ctx, `
		SELECT g.stage_object_state_id::text, g.scope_kind, COALESCE(g.scope_id::text, '')
		FROM stage_object_scope_grants g
		JOIN stage_object_states s ON s.id = g.stage_object_state_id
		WHERE s.show_id = $1::uuid
	`, showID)
	if err != nil {
		return nil, err
	}
	defer grantRows.Close()
	for grantRows.Next() {
		var stateID string
		var s Scope
		if err := grantRows.Scan(&stateID, &s.Kind, &s.ID); err != nil {
			return nil, err
		}
		ref, ok := byID[stateID]
		if !ok {
			continue
		}
		st := byRef[ref]
		st.Scopes = append(st.Scopes, s)
		byRef[ref] = st
	}
	return byRef, grantRows.Err()
}
