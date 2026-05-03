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
		return Decision{Allowed: false, Reason: "unknown_action"}, nil
	}
	if target.Kind != "element" {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}
	if target.ElementID == "" && target.ElementSlug == "" {
		return Decision{Allowed: false, Reason: "unknown_target"}, nil
	}

	return canActRevealInCave(ctx, q, userID, sessionID, target)
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

	if !canRoleRevealHide(participantRole, actorsCanReveal) {
		if normalizeActionRole(participantRole) == "cast" || normalizeActionRole(participantRole) == "actor" {
			return Decision{Allowed: false, Reason: "policy_denied"}, nil
		}
		return Decision{Allowed: false, Reason: "insufficient_role"}, nil
	}

	return Decision{Allowed: true, Reason: "allowed"}, nil
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
	return strings.EqualFold(strings.TrimSpace(target.ElementSlug), "first-fire")
}

func normalizeActionRole(role string) string {
	return strings.ToLower(strings.TrimSpace(role))
}
