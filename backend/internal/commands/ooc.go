package commands

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/actions"
)

// ExecuteOOC wraps actions.StoreOOCMessage. The HTTP layer is responsible for
// broadcasting the returned StoredAction over the hub -- this package has no
// hub dependency.
func ExecuteOOC(ctx context.Context, pool *pgxpool.Pool, actorUserID, sessionID, text string) (*actions.StoredAction, error) {
	return actions.StoreOOCMessage(ctx, pool, actions.OOCMessageRequest{
		SessionID: sessionID,
		ActorID:   actorUserID,
		Text:      text,
	})
}
