package access

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func EnsureKernel16VenueSurface(ctx context.Context, pool *pgxpool.Pool) error {
	type venueSeed struct {
		slug       string
		name       string
		kind       string
		config     string
		isPublic   bool
		isWorkshop bool
	}

	seeds := []venueSeed{
		{
			slug:       "library",
			name:       "Library",
			kind:       "library",
			config:     `{ "surface": "public_library", "reference_only": true }`,
			isPublic:   true,
			isWorkshop: false,
		},
		{
			slug:       "grants-cabin",
			name:       "Grant's Cabin",
			kind:       "cabin",
			config:     `{ "surface": "invite_only", "invite_only": true, "contact": "grant@amurray.family" }`,
			isPublic:   false,
			isWorkshop: false,
		},
		{
			slug:       "catharsis",
			name:       "Catharsis",
			kind:       "plaza",
			config:     `{ "surface": "audience", "ticketing": "planned", "permission_gate": true, "project": "Socio" }`,
			isPublic:   false,
			isWorkshop: false,
		},
		{
			slug:       "warehouse",
			name:       "Warehouse",
			kind:       "warehouse",
			config:     `{ "surface": "restricted_storage", "reference_only": true }`,
			isPublic:   false,
			isWorkshop: false,
		},
		{
			slug:       "soil-experts",
			name:       "Soil Experts",
			kind:       "farm",
			config:     `{ "surface": "public_placeholder", "reference_only": true, "project": "Soil Experts" }`,
			isPublic:   true,
			isWorkshop: false,
		},
	}

	var locationID string
	if err := pool.QueryRow(ctx, `
		SELECT id::text
		FROM locations
		WHERE slug = 'amurray-family'
		LIMIT 1
	`).Scan(&locationID); err != nil {
		return err
	}

	var lotID string
	if err := pool.QueryRow(ctx, `
		SELECT id::text
		FROM lots
		WHERE location_id = $1::uuid
		  AND slug = 'main-lot'
		LIMIT 1
	`, locationID).Scan(&lotID); err != nil {
		return err
	}

	for _, seed := range seeds {
		if _, err := pool.Exec(ctx, `
			INSERT INTO venues (lot_id, name, slug, kind, config, is_public, is_workshop)
			SELECT $1::uuid, $2, $3, $4, $5::jsonb, $6, $7
			WHERE NOT EXISTS (
				SELECT 1
				FROM venues
				WHERE lot_id = $1::uuid
				  AND slug = $3
			)
		`, lotID, seed.name, seed.slug, seed.kind, seed.config, seed.isPublic, seed.isWorkshop); err != nil {
			return err
		}
	}

	return nil
}
