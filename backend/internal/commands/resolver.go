package commands

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/characters"
)

// ResolveActiveCharacter composes the two active-character sources that
// exist in the codebase today but are never combined anywhere else:
// session-scoped persona equip (characters.ActivePersonaForSession) and the
// account-level active character (characters.ActiveCharacterForUser).
//
// Precedence: if sessionID is non-empty and a persona is equipped for that
// session, it wins. Otherwise fall back to the account-level active
// character. If neither exists, returns (nil, "", nil) -- callers should
// surface this as "no_active_character", not an error.
func ResolveActiveCharacter(ctx context.Context, pool *pgxpool.Pool, userID, sessionID string) (map[string]any, string, error) {
	userID = strings.TrimSpace(userID)
	sessionID = strings.TrimSpace(sessionID)
	if userID == "" {
		return nil, "", nil
	}

	if sessionID != "" {
		persona, err := characters.ActivePersonaForSession(ctx, pool, sessionID, userID)
		if err == nil {
			return persona, "session", nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, "", err
		}
	}

	account, err := characters.ActiveCharacterForUser(ctx, pool, userID)
	if err != nil {
		return nil, "", err
	}
	if account != nil {
		return account, "account", nil
	}

	return nil, "", nil
}
