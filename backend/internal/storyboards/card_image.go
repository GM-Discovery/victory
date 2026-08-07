package storyboards

// Card image storage-scope resolution (Kernel 81 spec 2.6/9.1/12).
//
// Every Victory asset is stored and quota-accounted against a producer +
// location (see assets/upload.go), but Storyboards boards are personal and
// have no Production/location of their own (Kernel 80's confirmed scope
// boundary). Rather than requiring the *uploading* Crew+ editor to also be
// a producer somewhere -- which assets.HandleWorkshopUpload's Producer-only
// gate would otherwise require, silently locking out most Crew editors --
// storage is scoped to the *board owner*: their own producer membership if
// they have one, else the same single-instance default location an
// Operator upload already falls back to (assets/upload.go's
// resolveProducerScope, Operator branch). Only board ownership determines
// whose storage quota is charged; the uploader's own membership is
// irrelevant, matching canMutateCard's Crew+ authority for this action.

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func resolveCardImageStorageScope(ctx context.Context, pool *pgxpool.Pool, ownerUserID string) (producerUserID, locationID string, err error) {
	err = pool.QueryRow(ctx, `
		SELECT lm.user_id, lm.location_id
		FROM location_memberships lm
		WHERE lm.user_id = $1 AND lm.role = 'producer' AND lm.active = TRUE
		ORDER BY lm.created_at ASC
		LIMIT 1
	`, ownerUserID).Scan(&producerUserID, &locationID)
	if err == nil {
		return producerUserID, locationID, nil
	}

	var fallbackLocationID string
	if fbErr := pool.QueryRow(ctx, `SELECT id FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&fallbackLocationID); fbErr != nil {
		return "", "", fbErr
	}
	return ownerUserID, fallbackLocationID, nil
}
