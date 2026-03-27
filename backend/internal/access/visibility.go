package access

import (
	"context"
	"strings"

	"victory/backend/internal/sessions"

	"github.com/jackc/pgx/v5/pgxpool"
)

type VisibleVenue struct {
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	IsPublic   bool   `json:"is_public"`
	IsWorkshop bool   `json:"is_workshop"`
	Reason     string `json:"reason"`
}

func CurrentUserIDFromRequest(ctx context.Context, pool *pgxpool.Pool, rawCookie string) (string, error) {
	rawCookie = strings.TrimSpace(rawCookie)
	if rawCookie == "" {
		return "", nil
	}

	rec, err := sessions.GetSessionByRawToken(ctx, pool, rawCookie)
	if err != nil {
		return "", err
	}

	return rec.UserID, nil
}

func ResolveVisibleVenues(ctx context.Context, pool *pgxpool.Pool, userID string) ([]VisibleVenue, error) {
	if strings.TrimSpace(userID) == "" {
		rows, err := pool.Query(ctx, `
			SELECT slug, name, is_public, is_workshop
			FROM venues
			WHERE is_public = TRUE
			ORDER BY slug
		`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var out []VisibleVenue
		for rows.Next() {
			var v VisibleVenue
			if err := rows.Scan(&v.Slug, &v.Name, &v.IsPublic, &v.IsWorkshop); err != nil {
				return nil, err
			}
			v.Reason = "public"
			out = append(out, v)
		}
		return out, rows.Err()
	}

	rows, err := pool.Query(ctx, `
		WITH visible AS (
			SELECT v.slug, v.name, v.is_public, v.is_workshop, 'public'::text AS reason
			FROM venues v
			WHERE v.is_public = TRUE

			UNION

			SELECT v.slug, v.name, v.is_public, v.is_workshop, 'membership'::text AS reason
			FROM venues v
			JOIN memberships m ON m.venue_id = v.id
			WHERE m.user_id = $1
			  AND m.active = TRUE

			UNION

			SELECT v.slug, v.name, v.is_public, v.is_workshop, 'grant'::text AS reason
			FROM venues v
			JOIN access_grants ag ON ag.venue_id = v.id
			WHERE ag.user_id = $1
			  AND ag.revoked_at IS NULL
			  AND (ag.expires_at IS NULL OR ag.expires_at > NOW())

			UNION

			SELECT v.slug, v.name, v.is_public, v.is_workshop, 'location_role'::text AS reason
			FROM venues v
			JOIN lots l ON l.id = v.lot_id
			JOIN locations loc ON loc.id = l.location_id
			JOIN location_memberships lm ON lm.location_id = loc.id
			WHERE lm.user_id = $1
			  AND lm.active = TRUE
			  AND lm.role IN ('producer', 'director', 'cast', 'crew', 'audience')

			UNION

			SELECT v.slug, v.name, v.is_public, v.is_workshop, 'production_role'::text AS reason
			FROM venues v
			JOIN lots l ON l.id = v.lot_id
			JOIN locations loc ON loc.id = l.location_id
			JOIN memberships m ON m.location_id = loc.id
			WHERE m.user_id = $1
			  AND m.active = TRUE
			  AND m.production_id IS NOT NULL
			  AND m.role IN ('director', 'cast', 'crew')
		)
		SELECT DISTINCT slug, name, is_public, is_workshop, reason
		FROM visible
		ORDER BY slug
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []VisibleVenue
	for rows.Next() {
		var v VisibleVenue
		if err := rows.Scan(&v.Slug, &v.Name, &v.IsPublic, &v.IsWorkshop, &v.Reason); err != nil {
			return nil, err
		}
		out = append(out, v)
	}

	return out, rows.Err()
}

func UserCanAccessVenueSlug(ctx context.Context, pool *pgxpool.Pool, userID, slug string) (bool, error) {
	venues, err := ResolveVisibleVenues(ctx, pool, userID)
	if err != nil {
		return false, err
	}

	for _, v := range venues {
		if v.Slug == slug {
			return true, nil
		}
	}

	return false, nil
}
