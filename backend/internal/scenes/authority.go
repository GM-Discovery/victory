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

// CanManageScenesForLocation is Operator, or Producer/Director at the
// given location -- the same location-scoped authority
// showruns.CanManageShowRun already enforces for Show Runs and Shows, reused
// here rather than duplicated (Kernel 67's precedent for security-relevant
// authority code). Scene reuse is location-scoped (Kernel 70 SS3.1), so this
// takes a location_id directly rather than resolving one from a Production.
func CanManageScenesForLocation(ctx context.Context, pool *pgxpool.Pool, userID, locationID string) (bool, error) {
	return showruns.CanManageShowRun(ctx, pool, userID, locationID)
}

// CanViewScenesForLocation is CanManageScenesForLocation plus an active
// Show Run crew roster row at the location, matching
// showruns.CanViewBackstage's own visibility-only exception.
func CanViewScenesForLocation(ctx context.Context, pool *pgxpool.Pool, userID, locationID string) (bool, error) {
	return showruns.CanViewBackstage(ctx, pool, userID, locationID)
}

// CanManageScenesForProduction resolves a Production's location, then
// delegates to CanManageScenesForLocation -- kept for callers that only
// know a Production (e.g. legacy provenance-only creation flows), not as
// an access-boundary distinction: a Scene's Production never restricts
// reuse (Kernel 70 SS3.1).
func CanManageScenesForProduction(ctx context.Context, pool *pgxpool.Pool, userID, productionID string) (bool, error) {
	locationID, err := resolveProductionLocationID(ctx, pool, productionID)
	if err != nil {
		return false, err
	}
	return CanManageScenesForLocation(ctx, pool, userID, locationID)
}

// CanViewScenesBackstage resolves a Production's location, then delegates
// to CanViewScenesForLocation. Kept for the same legacy-caller reason as
// CanManageScenesForProduction.
func CanViewScenesBackstage(ctx context.Context, pool *pgxpool.Pool, userID, productionID string) (bool, error) {
	locationID, err := resolveProductionLocationID(ctx, pool, productionID)
	if err != nil {
		return false, err
	}
	return CanViewScenesForLocation(ctx, pool, userID, locationID)
}
