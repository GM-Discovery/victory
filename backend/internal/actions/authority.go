package actions

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

type ActionTarget struct {
	Kind        string
	ElementID   string
	ElementSlug string
	VenueSlug   string
	Layer       string
}

type Decision struct {
	Allowed bool
	Reason  string
}

type ActionDeniedError struct {
	Reason string
}

func (e *ActionDeniedError) Error() string {
	if e == nil {
		return "action denied"
	}

	if strings.TrimSpace(e.Reason) == "" {
		return "action denied"
	}

	return fmt.Sprintf("action denied: %s", e.Reason)
}

type actionQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func CanAct(ctx context.Context, q actionQuerier, userID, actionType string, sessionID string, target ActionTarget) (Decision, error) {
	userID = strings.TrimSpace(userID)
	actionType = strings.TrimSpace(actionType)
	sessionID = strings.TrimSpace(sessionID)
	target.ElementID = strings.TrimSpace(target.ElementID)
	target.ElementSlug = strings.TrimSpace(target.ElementSlug)
	target.Layer = strings.TrimSpace(target.Layer)
	target.Kind = strings.TrimSpace(target.Kind)

	if userID == "" {
		return Decision{Allowed: false, Reason: "not_authenticated"}, nil
	}
	if sessionID == "" {
		return Decision{Allowed: false, Reason: "not_session_participant"}, nil
	}
	if actionType != "act/reveal_element" && actionType != "act/hide_element" {
		if actionType != "create/index_card" && actionType != "update/index_card" && actionType != "delete/index_card" && actionType != "act/place_element" && actionType != "act/duplicate_element" && actionType != "act/remove_element" && actionType != "act/show_overlay" && actionType != "act/hide_overlay" && actionType != "act/set_element_lock" && actionType != "act/set_nameplate_visibility" && actionType != "chat/message" && actionType != "persona/equip" && actionType != "persona/unequip" {
			return Decision{Allowed: false, Reason: "unknown_action"}, nil
		}
	}

	var showingStatus string
	err := q.QueryRow(ctx, `
		SELECT COALESCE(status::text, '')
		FROM showings
		WHERE session_id = $1
		LIMIT 1
	`, sessionID).Scan(&showingStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Decision{Allowed: false, Reason: "showing_not_found"}, nil
		}
		return Decision{}, err
	}
	if strings.EqualFold(strings.TrimSpace(showingStatus), "closed") {
		return Decision{Allowed: false, Reason: "showing_closed"}, nil
	}

	if actionType == "create/index_card" || actionType == "update/index_card" || actionType == "delete/index_card" || actionType == "act/place_element" {
		if actionType == "act/place_element" {
			return canActPlaceElement(ctx, q, userID, sessionID, target)
		}
		return canActIndexCardInCave(ctx, q, userID, sessionID, target, actionType)
	}

	if actionType == "act/duplicate_element" {
		return canActDuplicateElement(ctx, q, userID, sessionID, target)
	}

	if actionType == "act/remove_element" {
		return canActRemoveElement(ctx, q, userID, sessionID, target)
	}

	if actionType == "act/show_overlay" || actionType == "act/hide_overlay" {
		return canActOverlayInCave(ctx, q, userID, sessionID, target)
	}

	if actionType == "act/set_element_lock" {
		return canActSetElementLock(ctx, q, userID, sessionID, target)
	}

	if actionType == "act/set_nameplate_visibility" {
		return canActSetNameplateVisibility(ctx, q, userID, sessionID, target)
	}

	if actionType == "chat/message" {
		return canActChatMessage(ctx, q, userID, sessionID)
	}

	if actionType == "persona/equip" || actionType == "persona/unequip" {
		return canActPersona(ctx, q, userID, sessionID, target, actionType)
	}

	if target.Kind != "element" {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}
	if target.ElementID == "" && target.ElementSlug == "" {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}

	return canActRevealInCave(ctx, q, userID, sessionID, target)
}

func canActPersona(ctx context.Context, q actionQuerier, userID, sessionID string, target ActionTarget, actionType string) (Decision, error) {
	var participantUserID string
	err := q.QueryRow(ctx, `
		SELECT sp.user_id::text
		FROM session_participants sp
		WHERE sp.session_id = $1
		  AND sp.user_id = $2
		LIMIT 1
	`, sessionID, userID).Scan(&participantUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Decision{Allowed: false, Reason: "not_session_participant"}, nil
		}
		return Decision{}, err
	}

	if actionType == "persona/unequip" {
		return Decision{Allowed: true, Reason: "allowed"}, nil
	}

	if strings.TrimSpace(target.ElementID) == "" {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}

	var ownerUserID string
	err = q.QueryRow(ctx, `
		SELECT owner_user_id::text
		FROM character_cards
		WHERE id = $1
		  AND is_deleted = FALSE
		LIMIT 1
	`, target.ElementID).Scan(&ownerUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Decision{Allowed: false, Reason: "unknown_target"}, nil
		}
		return Decision{}, err
	}
	if strings.TrimSpace(ownerUserID) != userID {
		return Decision{Allowed: false, Reason: "forbidden"}, nil
	}

	return Decision{Allowed: true, Reason: "allowed"}, nil
}

func canActRevealInCave(ctx context.Context, q actionQuerier, userID, sessionID string, target ActionTarget) (Decision, error) {
	var (
		venueSlug        string
		participantRole  string
		actorsCanReveal  bool
		sessionVenueID   string
		sessionFoundUser string
	)

	err := q.QueryRow(ctx, `
		SELECT
			v.slug,
			sp.role::text,
			COALESCE((v.config ->> 'actors_can_reveal')::boolean, FALSE),
			s.venue_id::text,
			sp.user_id::text
		FROM sessions s
		JOIN venues v ON v.id = s.venue_id
		JOIN session_participants sp ON sp.session_id = s.id
		WHERE s.id = $1
		  AND sp.user_id = $2
		LIMIT 1
	`, sessionID, userID).Scan(&venueSlug, &participantRole, &actorsCanReveal, &sessionVenueID, &sessionFoundUser)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Decision{Allowed: false, Reason: "not_session_participant"}, nil
		}
		return Decision{}, err
	}

	if strings.TrimSpace(sessionFoundUser) == "" || strings.TrimSpace(sessionVenueID) == "" {
		return Decision{Allowed: false, Reason: "not_session_participant"}, nil
	}

	var venueAccess bool
	err = q.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM sessions s
			JOIN venues v ON v.id = s.venue_id
			WHERE s.id = $1
			  AND v.slug = 'the-cave'
		)
	`, sessionID).Scan(&venueAccess)
	if err != nil {
		return Decision{}, err
	}
	if !venueAccess {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}

	if venueSlug != "the-cave" {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}

	if !isRevealableCaveTarget(target) {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}

	locked, found, err := isVenueLayoutElementLocked(ctx, q, sessionID, target.ElementID, target.ElementSlug)
	if err != nil {
		return Decision{}, err
	}
	if found && locked {
		return Decision{Allowed: false, Reason: "locked"}, nil
	}

	if !canRoleRevealHide(participantRole, actorsCanReveal) {
		if normalizeActionRole(participantRole) == "cast" || normalizeActionRole(participantRole) == "actor" {
			return Decision{Allowed: false, Reason: "policy_denied"}, nil
		}
		return Decision{Allowed: false, Reason: "insufficient_role"}, nil
	}

	return Decision{Allowed: true, Reason: "allowed"}, nil
}

func canActOverlayInCave(ctx context.Context, q actionQuerier, userID, sessionID string, target ActionTarget) (Decision, error) {
	var (
		venueSlug       string
		participantRole string
		surface         string
	)

	base := `
		SELECT
			v.slug,
			sp.role::text,
			vle.surface
		FROM sessions s
		JOIN venues v ON v.id = s.venue_id
		JOIN session_participants sp ON sp.session_id = s.id
		JOIN venue_layout_elements vle ON vle.venue_id = v.id
		JOIN elements e ON e.id = vle.element_id
		WHERE s.id = $1
		  AND sp.user_id = $2
	`

	var err error
	switch {
	case strings.TrimSpace(target.ElementID) != "":
		err = q.QueryRow(ctx, base+` AND e.id = $3 LIMIT 1`, sessionID, userID, target.ElementID).Scan(&venueSlug, &participantRole, &surface)
	case strings.TrimSpace(target.ElementSlug) != "":
		err = q.QueryRow(ctx, base+` AND e.slug = $3 LIMIT 1`, sessionID, userID, target.ElementSlug).Scan(&venueSlug, &participantRole, &surface)
	default:
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Decision{Allowed: false, Reason: "unknown_target"}, nil
		}
		return Decision{}, err
	}

	locked, found, err := isVenueLayoutElementLocked(ctx, q, sessionID, target.ElementID, target.ElementSlug)
	if err != nil {
		return Decision{}, err
	}
	if found && locked {
		return Decision{Allowed: false, Reason: "locked"}, nil
	}

	if venueSlug != "the-cave" {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}
	if strings.ToLower(strings.TrimSpace(surface)) != "stage" {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}

	switch normalizeActionRole(participantRole) {
	case "producer", "director":
		return Decision{Allowed: true, Reason: "allowed"}, nil
	default:
		return Decision{Allowed: false, Reason: "insufficient_role"}, nil
	}
}

func canActIndexCardInCave(ctx context.Context, q actionQuerier, userID, sessionID string, target ActionTarget, actionType string) (Decision, error) {
	var (
		venueSlug       string
		participantRole string
	)

	err := q.QueryRow(ctx, `
		SELECT
			v.slug,
			sp.role::text
		FROM sessions s
		JOIN venues v ON v.id = s.venue_id
		JOIN session_participants sp ON sp.session_id = s.id
		WHERE s.id = $1
		  AND sp.user_id = $2
		LIMIT 1
	`, sessionID, userID).Scan(&venueSlug, &participantRole)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Decision{Allowed: false, Reason: "not_session_participant"}, nil
		}
		return Decision{}, err
	}

	if venueSlug != "the-cave" {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}

	if actionType == "update/index_card" {
		if strings.TrimSpace(target.ElementID) == "" && strings.TrimSpace(target.ElementSlug) == "" {
			return Decision{Allowed: false, Reason: "unknown_target"}, nil
		}
		locked, found, err := isVenueLayoutElementLocked(ctx, q, sessionID, target.ElementID, target.ElementSlug)
		if err != nil {
			return Decision{}, err
		}
		if found && locked {
			return Decision{Allowed: false, Reason: "locked"}, nil
		}
	}

	if actionType == "delete/index_card" {
		if strings.TrimSpace(target.ElementID) == "" && strings.TrimSpace(target.ElementSlug) == "" {
			return Decision{Allowed: false, Reason: "unknown_target"}, nil
		}
		locked, found, err := isVenueLayoutElementLocked(ctx, q, sessionID, target.ElementID, target.ElementSlug)
		if err != nil {
			return Decision{}, err
		}
		if found && locked {
			return Decision{Allowed: false, Reason: "locked"}, nil
		}
	}

	if actionType == "act/place_element" {
		if strings.TrimSpace(target.ElementID) == "" && strings.TrimSpace(target.ElementSlug) == "" {
			return Decision{Allowed: false, Reason: "unknown_target"}, nil
		}
	}

	switch normalizeActionRole(participantRole) {
	case "producer", "director":
		return Decision{Allowed: true, Reason: "allowed"}, nil
	default:
		return Decision{Allowed: false, Reason: "insufficient_role"}, nil
	}
}

func canActPlaceElement(ctx context.Context, q actionQuerier, userID, sessionID string, target ActionTarget) (Decision, error) {
	if strings.TrimSpace(target.VenueSlug) == "" {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}
	if strings.TrimSpace(target.ElementID) == "" && strings.TrimSpace(target.ElementSlug) == "" {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}
	if strings.TrimSpace(target.Layer) == "" {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}

	var participantRole string
	err := q.QueryRow(ctx, `
		SELECT sp.role::text
		FROM session_participants sp
		WHERE sp.session_id = $1
		  AND sp.user_id = $2
		LIMIT 1
	`, sessionID, userID).Scan(&participantRole)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Decision{Allowed: false, Reason: "not_session_participant"}, nil
		}
		return Decision{}, err
	}

	var venueSlug string
	var enabled bool
	err = q.QueryRow(ctx, `
		SELECT
			v.slug,
			COALESCE((v.config ->> 'index_cards_enabled')::boolean, FALSE)
		FROM venues v
		WHERE v.slug = $1
		LIMIT 1
	`, target.VenueSlug).Scan(&venueSlug, &enabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Decision{Allowed: false, Reason: "unknown_target"}, nil
		}
		return Decision{}, err
	}

	if !enabled {
		return Decision{Allowed: false, Reason: "policy_denied"}, nil
	}

	locked, found, err := isVenueLayoutElementLocked(ctx, q, sessionID, target.ElementID, target.ElementSlug)
	if err != nil {
		return Decision{}, err
	}
	if found && locked {
		return Decision{Allowed: false, Reason: "locked"}, nil
	}

	switch normalizeActionRole(participantRole) {
	case "producer", "director":
		return Decision{Allowed: true, Reason: "allowed"}, nil
	default:
		return Decision{Allowed: false, Reason: "insufficient_role"}, nil
	}
}

func canActDuplicateElement(ctx context.Context, q actionQuerier, userID, sessionID string, target ActionTarget) (Decision, error) {
	if strings.TrimSpace(target.VenueSlug) == "" {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}
	if strings.TrimSpace(target.ElementID) == "" && strings.TrimSpace(target.ElementSlug) == "" {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}
	if strings.TrimSpace(target.Layer) == "" {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}

	state, err := resolveVenueLayoutElementState(ctx, q, sessionID, target.ElementID, target.ElementSlug)
	if err != nil {
		return Decision{}, err
	}
	if state.VenueSlug != "the-cave" {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}
	if strings.ToLower(strings.TrimSpace(state.Surface)) != "stage" {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}
	if !isDuplicableStageElement(state.ElementType, state.ContextClass) {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}
	if visibilityBool(state.Visibility, "locked", false) {
		return Decision{Allowed: false, Reason: "locked"}, nil
	}

	var participantRole string
	err = q.QueryRow(ctx, `
		SELECT sp.role::text
		FROM session_participants sp
		WHERE sp.session_id = $1
		  AND sp.user_id = $2
		LIMIT 1
	`, sessionID, userID).Scan(&participantRole)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Decision{Allowed: false, Reason: "not_session_participant"}, nil
		}
		return Decision{}, err
	}

	switch normalizeActionRole(participantRole) {
	case "producer", "director":
		return Decision{Allowed: true, Reason: "allowed"}, nil
	default:
		return Decision{Allowed: false, Reason: "insufficient_role"}, nil
	}
}

func canActRemoveElement(ctx context.Context, q actionQuerier, userID, sessionID string, target ActionTarget) (Decision, error) {
	if strings.TrimSpace(target.ElementID) == "" && strings.TrimSpace(target.ElementSlug) == "" {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}

	state, err := resolveVenueLayoutElementState(ctx, q, sessionID, target.ElementID, target.ElementSlug)
	if err != nil {
		return Decision{}, err
	}
	if state.VenueSlug != "the-cave" {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}
	if strings.ToLower(strings.TrimSpace(state.Surface)) != "stage" {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}
	if visibilityBool(state.Visibility, "locked", false) {
		return Decision{Allowed: false, Reason: "locked"}, nil
	}

	var participantRole string
	err = q.QueryRow(ctx, `
		SELECT sp.role::text
		FROM session_participants sp
		WHERE sp.session_id = $1
		  AND sp.user_id = $2
		LIMIT 1
	`, sessionID, userID).Scan(&participantRole)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Decision{Allowed: false, Reason: "not_session_participant"}, nil
		}
		return Decision{}, err
	}

	switch normalizeActionRole(participantRole) {
	case "producer", "director":
		return Decision{Allowed: true, Reason: "allowed"}, nil
	default:
		return Decision{Allowed: false, Reason: "insufficient_role"}, nil
	}
}

func canRoleRevealHide(role string, actorsCanReveal bool) bool {
	switch normalizeActionRole(role) {
	case "producer", "director":
		return true
	case "cast", "actor":
		return actorsCanReveal
	default:
		return false
	}
}

func isRevealableCaveTarget(target ActionTarget) bool {
	return strings.TrimSpace(target.ElementID) != "" || strings.TrimSpace(target.ElementSlug) != ""
}

func canActSetElementLock(ctx context.Context, q actionQuerier, userID, sessionID string, target ActionTarget) (Decision, error) {
	state, err := resolveVenueLayoutElementState(ctx, q, sessionID, target.ElementID, target.ElementSlug)
	if err != nil {
		return Decision{}, err
	}
	if state.VenueSlug != "the-cave" {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}

	var participantRole string
	err = q.QueryRow(ctx, `
		SELECT sp.role::text
		FROM session_participants sp
		WHERE sp.session_id = $1
		  AND sp.user_id = $2
		LIMIT 1
	`, sessionID, userID).Scan(&participantRole)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Decision{Allowed: false, Reason: "not_session_participant"}, nil
		}
		return Decision{}, err
	}

	switch normalizeActionRole(participantRole) {
	case "producer", "director":
		return Decision{Allowed: true, Reason: "allowed"}, nil
	default:
		return Decision{Allowed: false, Reason: "insufficient_role"}, nil
	}
}

func canActSetNameplateVisibility(ctx context.Context, q actionQuerier, userID, sessionID string, target ActionTarget) (Decision, error) {
	state, err := resolveVenueLayoutElementState(ctx, q, sessionID, target.ElementID, target.ElementSlug)
	if err != nil {
		return Decision{}, err
	}
	if state.VenueSlug != "the-cave" {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}

	locked := visibilityBool(state.Visibility, "locked", false)
	if locked {
		return Decision{Allowed: false, Reason: "locked"}, nil
	}

	var participantRole string
	err = q.QueryRow(ctx, `
		SELECT sp.role::text
		FROM session_participants sp
		WHERE sp.session_id = $1
		  AND sp.user_id = $2
		LIMIT 1
	`, sessionID, userID).Scan(&participantRole)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Decision{Allowed: false, Reason: "not_session_participant"}, nil
		}
		return Decision{}, err
	}

	switch normalizeActionRole(participantRole) {
	case "producer", "director":
		return Decision{Allowed: true, Reason: "allowed"}, nil
	default:
		return Decision{Allowed: false, Reason: "insufficient_role"}, nil
	}
}

func isVenueLayoutElementLocked(ctx context.Context, q actionQuerier, sessionID, elementID, elementSlug string) (bool, bool, error) {
	state, err := resolveVenueLayoutElementState(ctx, q, sessionID, elementID, elementSlug)
	if err != nil {
		if strings.Contains(err.Error(), "index_card_not_found") {
			return false, false, nil
		}
		return false, false, err
	}
	return visibilityBool(state.Visibility, "locked", false), true, nil
}

func normalizeActionRole(role string) string {
	return strings.ToLower(strings.TrimSpace(role))
}
