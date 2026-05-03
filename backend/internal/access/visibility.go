package access

import (
	"context"
	"strings"

	"victory/backend/internal/sessions"

	"github.com/jackc/pgx/v5/pgxpool"
)

type VisibleVenue struct {
	Slug           string `json:"slug"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	VisibleBecause string `json:"visible_because"`
}

func CurrentLocationRole(ctx context.Context, pool *pgxpool.Pool, userID string) (string, error) {
	if strings.TrimSpace(userID) == "" {
		return "audience", nil
	}

	var role string
	err := pool.QueryRow(ctx, `
		SELECT m.role::text
		FROM location_memberships m
		WHERE m.user_id = $1
		  AND m.active = TRUE
		ORDER BY
		  CASE m.role
			WHEN 'producer' THEN 1
			WHEN 'director' THEN 2
			WHEN 'cast' THEN 3
			WHEN 'crew' THEN 4
			WHEN 'audience' THEN 5
			ELSE 99
		  END,
		  m.created_at ASC
		LIMIT 1
	`, userID).Scan(&role)
	if err != nil {
		return "audience", nil
	}

	switch role {
	case "producer", "director", "cast", "crew", "audience":
		return role, nil
	default:
		return "audience", nil
	}
}

func IsPerformerRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "producer", "director", "cast", "crew":
		return true
	default:
		return false
	}
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
			SELECT v.slug, v.name, v.kind, 'public'::text AS visible_because
			FROM venues v
			WHERE v.is_public = TRUE
			ORDER BY v.slug
		`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var out []VisibleVenue
		for rows.Next() {
			var v VisibleVenue
			if err := rows.Scan(&v.Slug, &v.Name, &v.Kind, &v.VisibleBecause); err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, rows.Err()
	}

	if ok, err := IsOperatorUser(ctx, pool, userID); err == nil && ok {
		rows, err := pool.Query(ctx, `
			SELECT v.slug, v.name, v.kind, 'operator'::text AS visible_because
			FROM venues v
			ORDER BY v.slug
		`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var out []VisibleVenue
		for rows.Next() {
			var v VisibleVenue
			if err := rows.Scan(&v.Slug, &v.Name, &v.Kind, &v.VisibleBecause); err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, rows.Err()
	}

	rows, err := pool.Query(ctx, `
		WITH visible AS (
			SELECT v.id, v.slug, v.name, v.kind, 1 AS reason_rank, 'public'::text AS visible_because
			FROM venues v
			WHERE v.is_public = TRUE

			UNION

			SELECT v.id, v.slug, v.name, v.kind, 2 AS reason_rank, 'performer_surface'::text AS visible_because
			FROM venues v
			JOIN lots l ON l.id = v.lot_id
			JOIN location_memberships lm ON lm.location_id = l.location_id
			WHERE v.slug IN ('audition-hall', 'greenroom', 'trailers', 'workshop')
			  AND lm.user_id = $1
			  AND lm.active = TRUE
			  AND lm.role IN ('producer', 'director', 'cast', 'crew')

			UNION

			SELECT v.id, v.slug, v.name, v.kind, 3 AS reason_rank, 'venue_membership'::text AS visible_because
			FROM venues v
			JOIN memberships m ON m.venue_id = v.id
			WHERE m.user_id = $1
			AND m.active = TRUE

			UNION

			SELECT v.id, v.slug, v.name, v.kind, 4 AS reason_rank, 'grant'::text AS visible_because
			FROM venues v
			JOIN access_grants ag ON ag.venue_id = v.id
			WHERE ag.user_id = $1
			AND ag.revoked_at IS NULL
			AND (ag.expires_at IS NULL OR ag.expires_at > NOW())

			UNION

			SELECT v.id, v.slug, v.name, v.kind, 5 AS reason_rank, 'location_role'::text AS visible_because
			FROM venues v
			JOIN lots l ON l.id = v.lot_id
			JOIN location_memberships lm ON lm.location_id = l.location_id
			WHERE lm.user_id = $1
			AND lm.active = TRUE
			AND lm.role IN ('producer', 'director', 'cast', 'crew')

			UNION

			SELECT v.id, v.slug, v.name, v.kind, 6 AS reason_rank, 'production_role'::text AS visible_because
			FROM venues v
			JOIN memberships m ON m.venue_id = v.id
			WHERE m.user_id = $1
			AND m.active = TRUE
			AND m.production_id IS NOT NULL
			AND m.role IN ('director', 'cast', 'crew')
		),
		ranked AS (
			SELECT
				id,
				slug,
				name,
				kind,
				visible_because,
				reason_rank,
				ROW_NUMBER() OVER (PARTITION BY id ORDER BY reason_rank ASC) AS rn
			FROM visible
		)
		SELECT slug, name, kind, visible_because
		FROM ranked
		WHERE rn = 1
		ORDER BY slug
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []VisibleVenue
	for rows.Next() {
		var v VisibleVenue
		if err := rows.Scan(&v.Slug, &v.Name, &v.Kind, &v.VisibleBecause); err != nil {
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

func ResolveWorkshopVenues(ctx context.Context, pool *pgxpool.Pool, userID string) ([]VisibleVenue, error) {
	rows, err := pool.Query(ctx, `
		SELECT v.slug, v.name, v.kind, 'index_cards_enabled'::text AS visible_because
		FROM venues v
		WHERE COALESCE((v.config ->> 'index_cards_enabled')::boolean, FALSE) = TRUE
		ORDER BY v.slug
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []VisibleVenue
	for rows.Next() {
		var v VisibleVenue
		if err := rows.Scan(&v.Slug, &v.Name, &v.Kind, &v.VisibleBecause); err != nil {
			return nil, err
		}
		allowed, err := UserCanAccessVenueSlug(ctx, pool, userID, v.Slug)
		if err != nil {
			return nil, err
		}
		if !allowed {
			continue
		}
		out = append(out, v)
	}

	return out, rows.Err()
}
