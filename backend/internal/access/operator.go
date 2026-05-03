package access

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var operatorUserIDEnv = strings.TrimSpace(os.Getenv("OPERATOR_USER_ID"))
var operatorHandleEnv = strings.ToLower(strings.TrimSpace(os.Getenv("OPERATOR_HANDLE")))

func IsOperatorUser(ctx context.Context, pool *pgxpool.Pool, userID string) (bool, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return false, nil
	}

	if operatorUserIDEnv != "" && userID == operatorUserIDEnv {
		return true, nil
	}

	if operatorHandleEnv == "" {
		return false, nil
	}

	var handle string
	err := pool.QueryRow(ctx, `
		SELECT lower(COALESCE(NULLIF(handle, ''), ''))
		FROM users
		WHERE id = $1
		LIMIT 1
	`, userID).Scan(&handle)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return handle == operatorHandleEnv, nil
}
