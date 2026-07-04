package commands

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/characters"
)

// ExecuteJournalAdd wraps characters.SaveCharacterJournal, now threading the
// venue/session the entry was written from (columns existed since migration
// 031 but were never populated by any call site until this). Owner-only
// enforcement is unchanged -- it happens inside SaveCharacterJournal via
// ActiveCharacterForUser + author_user_id. Never mirrored to actions/chat.
func ExecuteJournalAdd(ctx context.Context, pool *pgxpool.Pool, actorUserID, text, venueID, sessionID string) (characters.CharacterJournalEntry, error) {
	return characters.SaveCharacterJournal(ctx, pool, actorUserID, text, "private", venueID, sessionID)
}

// ExecuteJournalRecent wraps characters.ListCharacterJournals unchanged.
func ExecuteJournalRecent(ctx context.Context, pool *pgxpool.Pool, actorUserID string) ([]characters.CharacterJournalEntry, error) {
	return characters.ListCharacterJournals(ctx, pool, actorUserID)
}
