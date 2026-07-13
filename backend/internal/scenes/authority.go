package scenes

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showruns"
)

// resolveProductionLocationID resolves a Production's location server-side,
// mirroring showruns.CreateShowRun's exact lookup -- a client-supplied
// production_id is never trusted for anything beyond "which row to look up."
func resolveProductionLocationID(ctx context.Context, pool *pgxpool.Pool, productionID string) (string, error) {
	productionID = strings.TrimSpace(productionID)
	if productionID == "" {
		return "", errors.New("production_id_required")
	}
	var locationID string
	if err := pool.QueryRow(ctx, `SELECT location_id::text FROM productions WHERE id = $1`, productionID).Scan(&locationID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errors.New("production_not_found")
		}
		return "", err
	}
	return locationID, nil
}

// CanManageScenesForProduction is Operator, or Producer/Director at the
// Production's resolved location -- the same location-scoped authority
// showruns.CanManageShowRun already enforces for Show Runs and Shows, reused
// here rather than duplicated (Kernel 67's precedent for security-relevant
// authority code).
func CanManageScenesForProduction(ctx context.Context, pool *pgxpool.Pool, userID, productionID string) (bool, error) {
	locationID, err := resolveProductionLocationID(ctx, pool, productionID)
	if err != nil {
		return false, err
	}
	return showruns.CanManageShowRun(ctx, pool, userID, locationID)
}

// CanViewScenesBackstage is CanManageScenesForProduction plus an active
// Show Run crew roster row at the Production's location, matching
// showruns.CanViewBackstage's own visibility-only exception.
func CanViewScenesBackstage(ctx context.Context, pool *pgxpool.Pool, userID, productionID string) (bool, error) {
	locationID, err := resolveProductionLocationID(ctx, pool, productionID)
	if err != nil {
		return false, err
	}
	return showruns.CanViewBackstage(ctx, pool, userID, locationID)
}
