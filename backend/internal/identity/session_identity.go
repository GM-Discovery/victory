package identity

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

type SessionIdentity struct {
	UserID               string `json:"user_id"`
	Handle               string `json:"handle"`
	DisplayName          string `json:"display_name"`
	Role                 string `json:"role"`
	SessionID            string `json:"session_id"`
	SessionParticipantID string `json:"session_participant_id"`
	Persona              any    `json:"persona"`
}

type identityQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func ResolveActiveCaveSessionID(ctx context.Context, q identityQuerier, userID string) (string, error) {
	var sessionID string

	err := q.QueryRow(ctx, `
		SELECT s.id::text
		FROM sessions s
		JOIN venues v ON v.id = s.venue_id
		JOIN session_participants sp ON sp.session_id = s.id
		WHERE v.slug = 'the-cave'
		  AND s.status IN ('rehearsal', 'live')
		  AND sp.user_id = $1
		ORDER BY s.started_at DESC
		LIMIT 1
	`, userID).Scan(&sessionID)
	if err != nil {
		return "", err
	}

	return sessionID, nil
}

func ResolveSessionIdentity(ctx context.Context, q identityQuerier, sessionID, userID string) (SessionIdentity, error) {
	sessionID = strings.TrimSpace(sessionID)
	userID = strings.TrimSpace(userID)
	if sessionID == "" {
		return SessionIdentity{}, errors.New("session_id is required")
	}
	if userID == "" {
		return SessionIdentity{}, errors.New("user_id is required")
	}

	var ident SessionIdentity
	var personaCardID, personaName, personaPronouns, personaPortraitURL, personaColor, personaTagline string

	err := q.QueryRow(ctx, `
		SELECT
			sp.id::text,
			s.id::text,
			u.id::text,
			COALESCE(NULLIF(u.handle, ''), LEFT(u.id::text, 8), 'Unknown Participant'),
			COALESCE(NULLIF(u.display_name, ''), NULLIF(u.handle, ''), LEFT(u.id::text, 8), 'Unknown Participant'),
			COALESCE(sp.role::text, 'audience'),
			COALESCE(cc.id::text, ''),
			COALESCE(cc.name, ''),
			COALESCE(cc.pronouns, ''),
			COALESCE(cc.portrait_url, ''),
			COALESCE(cc.color, ''),
			COALESCE(cc.tagline, '')
		FROM sessions s
		JOIN venues v ON v.id = s.venue_id
		JOIN session_participants sp ON sp.session_id = s.id
		JOIN users u ON u.id = sp.user_id
		LEFT JOIN current_session_personas csp ON csp.session_id = s.id AND csp.user_id = sp.user_id
		LEFT JOIN character_cards cc ON cc.id = csp.character_card_id AND cc.is_deleted = FALSE
		WHERE v.slug = 'the-cave'
		  AND s.id = $1::uuid
		  AND sp.user_id = $2
		  AND s.status IN ('rehearsal', 'live')
		LIMIT 1
	`, sessionID, userID).Scan(&ident.SessionParticipantID, &ident.SessionID, &ident.UserID, &ident.Handle, &ident.DisplayName, &ident.Role, &personaCardID, &personaName, &personaPronouns, &personaPortraitURL, &personaColor, &personaTagline)
	if err != nil {
		return SessionIdentity{}, err
	}

	ident.Persona = nil
	if personaCardID != "" {
		ident.Persona = map[string]any{
			"character_card_id": personaCardID,
			"name":              personaName,
			"display_name":      personaName,
			"pronouns":          personaPronouns,
			"portrait_url":      personaPortraitURL,
			"color":             personaColor,
			"tagline":           personaTagline,
		}
	}
	return ident, nil
}

func (i SessionIdentity) ActorMap() map[string]any {
	return map[string]any{
		"user_id":      i.UserID,
		"handle":       i.Handle,
		"display_name": i.DisplayName,
		"role":         i.Role,
		"persona":      i.Persona,
	}
}
