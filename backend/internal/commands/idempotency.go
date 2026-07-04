package commands

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ExecuteIdempotent runs fn at most once per (actorUserID, commandPath,
// idempotencyKey), modeled on the one real idempotency precedent in this
// codebase: characters/coin_flip.go's unique-index + ON CONFLICT DO NOTHING
// + read-back pattern (NOT the non-deduplicating dice request_id field,
// which is stored but never checked for conflicts).
//
// If a receipt row already exists for this key, fn is not re-run and the
// previously stored result is returned with replayed=true. If fn returns an
// error, the receipt row is deleted so a genuine transient failure can be
// retried with the same key.
func ExecuteIdempotent(ctx context.Context, pool *pgxpool.Pool, actorUserID, commandPath, idempotencyKey string, fn func(ctx context.Context) (map[string]any, error)) (map[string]any, bool, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	commandPath = strings.TrimSpace(commandPath)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if actorUserID == "" {
		return nil, false, errors.New("not_authenticated")
	}
	if idempotencyKey == "" {
		// No key supplied: execute directly, no dedupe tracking.
		result, err := fn(ctx)
		return result, false, err
	}

	var receiptID string
	err := pool.QueryRow(ctx, `
		INSERT INTO command_execution_receipts (actor_user_id, command_path, idempotency_key, result)
		VALUES ($1, $2, $3, '{}'::jsonb)
		ON CONFLICT (actor_user_id, command_path, idempotency_key) DO NOTHING
		RETURNING id::text
	`, actorUserID, commandPath, idempotencyKey).Scan(&receiptID)

	if errors.Is(err, pgx.ErrNoRows) {
		// Conflict: another attempt with this key already ran (or is running).
		var raw []byte
		if readErr := pool.QueryRow(ctx, `
			SELECT result
			FROM command_execution_receipts
			WHERE actor_user_id = $1 AND command_path = $2 AND idempotency_key = $3
		`, actorUserID, commandPath, idempotencyKey).Scan(&raw); readErr != nil {
			return nil, false, readErr
		}
		var result map[string]any
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &result); err != nil {
				return nil, false, err
			}
		}
		return result, true, nil
	}
	if err != nil {
		return nil, false, err
	}

	result, fnErr := fn(ctx)
	if fnErr != nil {
		_, _ = pool.Exec(ctx, `DELETE FROM command_execution_receipts WHERE id = $1::uuid`, receiptID)
		return nil, false, fnErr
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		_, _ = pool.Exec(ctx, `DELETE FROM command_execution_receipts WHERE id = $1::uuid`, receiptID)
		return nil, false, err
	}
	if _, err := pool.Exec(ctx, `
		UPDATE command_execution_receipts SET result = $2::jsonb WHERE id = $1::uuid
	`, receiptID, string(resultJSON)); err != nil {
		return nil, false, err
	}

	return result, false, nil
}
