package commands

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/actions"
)

// ExecuteIC wraps actions.StoreICChatMessage (Kernel 87 §10), mirroring
// ExecuteOOC's shape exactly. The HTTP layer is responsible for
// broadcasting the returned StoredAction over the hub -- this package has
// no hub dependency. The speaker Character is resolved entirely inside
// StoreICChatMessage; nothing here accepts or forwards a client-supplied
// Character ID.
func ExecuteIC(ctx context.Context, pool *pgxpool.Pool, actorUserID, sessionID, text string) (*actions.StoredAction, error) {
	return actions.StoreICChatMessage(ctx, pool, actions.ICChatMessageRequest{
		SessionID: sessionID,
		ActorID:   actorUserID,
		Text:      text,
	})
}
