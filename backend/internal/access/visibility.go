package access

import (
	"context"
	"strings"

	"victory/backend/internal/sessions"

	"github.com/jackc/pgx/v5/pgxpool"
)

type VisibleVenue struct {
	Slug              string `json:"slug"`
	Name              string `json:"name"`
	Kind              string `json:"kind"`
	VisibleBecause    string `json:"visible_because"`
	NotificationCount int    `json:"notification_count,omitempty"`
}

var hiddenMainMapVenueSlugs = map[string]struct{}{
	"gateway-thread":         {},
	"gateway-thread-fixture": {},
	"gateway-thread-venue":   {},
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
		return []VisibleVenue{}, nil
	}

	if ok, err := IsOperatorUser(ctx, pool, userID); err == nil && ok {
		// Operator visibility is intentionally broad, but gateway/thread fixtures
		// are still stripped from the main map so Discord backend plumbing does
		// not leak into the public venue surface.
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
			if isHiddenMainMapVenueSlug(v.Slug) {
				continue
			}
			out = append(out, v)
		}
		return out, rows.Err()
	}

	rows, err := pool.Query(ctx, `
		WITH visible AS (
			SELECT v.id, v.slug, v.name, v.kind, 1 AS reason_rank, 'authenticated_surface'::text AS visible_because
			FROM venues v
			WHERE v.slug IN ('audition-hall')

			UNION

			SELECT v.id, v.slug, v.name, v.kind, 2 AS reason_rank, 'approved_performer_surface'::text AS visible_because
			FROM venues v
			JOIN lots l ON l.id = v.lot_id
			JOIN location_memberships lm ON lm.location_id = l.location_id
			JOIN access_grants ag ON ag.venue_id = v.id
			WHERE v.slug = 'catharsis'
			  AND lm.user_id = $1
			  AND lm.active = TRUE
			  AND lm.role IN ('producer', 'director', 'cast', 'crew')
			  AND ag.user_id = $1
			  AND ag.revoked_at IS NULL
			  AND (ag.expires_at IS NULL OR ag.expires_at > NOW())

			UNION

			SELECT v.id, v.slug, v.name, v.kind, 3 AS reason_rank, 'performer_surface'::text AS visible_because
			FROM venues v
			JOIN lots l ON l.id = v.lot_id
			JOIN location_memberships lm ON lm.location_id = l.location_id
			WHERE v.slug IN ('trailers', 'third-place')
			  AND lm.user_id = $1
			  AND lm.active = TRUE
			  AND lm.role IN ('producer', 'director', 'cast', 'crew')

			UNION

			SELECT v.id, v.slug, v.name, v.kind, 4 AS reason_rank, 'owned_workbook_surface'::text AS visible_because
			FROM venues v
			WHERE v.slug = 'greenroom'
			  AND EXISTS (
				SELECT 1
				FROM character_cards cc
				WHERE cc.owner_user_id = $1
				  AND cc.is_deleted = FALSE
			  )

			UNION

			-- Show Runs (Kernel 66): unlike 'trailers'/'third-place' above, this
			-- surface is visible to Audience too, not just producer/director/
			-- cast/crew -- the operator's explicit "Audience gets the best
			-- seats" instruction means Audience must not be excluded from even
			-- seeing the venue tile that leads to the Audience Program.
			SELECT v.id, v.slug, v.name, v.kind, 3 AS reason_rank, 'location_member_surface'::text AS visible_because
			FROM venues v
			JOIN lots l ON l.id = v.lot_id
			JOIN location_memberships lm ON lm.location_id = l.location_id
			WHERE v.slug = 'show-runs'
			  AND lm.user_id = $1
			  AND lm.active = TRUE
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
		if isHiddenMainMapVenueSlug(v.Slug) {
			continue
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if counts, err := resolveVenueNotificationCounts(ctx, pool); err == nil {
		for i := range out {
			if count := counts[out[i].Slug]; count > 0 {
				out[i].NotificationCount = count
			}
		}
	}

	return out, nil
}

func isHiddenMainMapVenueSlug(slug string) bool {
	_, ok := hiddenMainMapVenueSlugs[strings.ToLower(strings.TrimSpace(slug))]
	return ok
}

func resolveVenueNotificationCounts(ctx context.Context, pool *pgxpool.Pool) (map[string]int, error) {
	rows, err := pool.Query(ctx, `
		SELECT venue_slug, COUNT(*)::int
		FROM (
			SELECT 'producers-office'::text AS venue_slug
			FROM permission_requests
			WHERE status = 'pending'
			  AND requested_role::text = 'director'

			UNION ALL

			SELECT 'directors-chair'::text AS venue_slug
			FROM permission_requests
			WHERE status = 'pending'
			  AND requested_role::text IN ('cast', 'crew')
		) queued
		GROUP BY venue_slug
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var slug string
		var count int
		if err := rows.Scan(&slug, &count); err != nil {
			return nil, err
		}
		counts[slug] = count
	}
	return counts, rows.Err()
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
