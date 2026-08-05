package access

import (
	"context"
	"sort"
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

// VenueReadinessChecker resolves whether userID meets some venue-specific
// readiness condition. access sits below playerprofile in the import graph
// (playerprofile already imports access), so main.go injects this callback
// at startup instead of access importing playerprofile directly -- the same
// avoid-import-cycle shape as playerprofile.ProjectionChangeNotifier
// (Kernel 68 §3.2).
type VenueReadinessChecker func(ctx context.Context, pool *pgxpool.Pool, userID string) (bool, error)

var thirdPlaceReadinessChecker VenueReadinessChecker

// SetThirdPlaceReadinessChecker wires the Trailer Face readiness check used
// to gate the Third Place venue tile (Kernel 68 §3.2). Must be called once
// at startup before any request is served.
func SetThirdPlaceReadinessChecker(fn VenueReadinessChecker) {
	thirdPlaceReadinessChecker = fn
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
			WHERE v.slug IN ('audition-hall', 'trailers')

			UNION

			-- Library (Kernel 79 operator amendment): gated on having a
			-- character card at Catharsis's own location, not open to every
			-- authenticated account -- the Library only becomes relevant
			-- once a player actually has a character to look rules up for.
			-- Readership authority for individual eWritings is still
			-- enforced per publication by /api/library routes regardless of
			-- this map-tile gate.
			SELECT v.id, v.slug, v.name, v.kind, 4 AS reason_rank, 'catharsis_character_surface'::text AS visible_because
			FROM venues v
			JOIN lots l ON l.id = v.lot_id
			WHERE v.slug = 'library'
			  AND EXISTS (
				SELECT 1
				FROM character_cards cc
				JOIN venues cv ON cv.slug = 'catharsis'
				JOIN lots cl ON cl.id = cv.lot_id
				WHERE cc.owner_user_id = $1
				  AND cc.is_deleted = FALSE
				  AND cc.location_id = cl.location_id
			  )

			UNION

			-- Writer's Room (Kernel 78; narrowed Kernel 79): the eWrite
			-- authoring venue. Director+ only -- producer/director at any
			-- active Location membership. Crew previously saw this tile too
			-- (Kernel 78 default); the operator narrowed it to Director+
			-- since authoring official rules content is a Director-level
			-- responsibility, not general Crew. Cast/Audience never see the
			-- tile; they read published eWritings in the Library instead.
			-- Map visibility only: every /api/ewrite/* route re-checks
			-- authoring authority server-side per request (visibility
			-- filtering is never invocation authorization --
			-- operator-notes.md).
			SELECT v.id, v.slug, v.name, v.kind, 3 AS reason_rank, 'ewrite_author_surface'::text AS visible_because
			FROM venues v
			JOIN lots l ON l.id = v.lot_id
			JOIN location_memberships lm ON lm.location_id = l.location_id
			WHERE v.slug = 'writers-room'
			  AND lm.user_id = $1
			  AND lm.active = TRUE
			  AND lm.role IN ('producer', 'director')

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

			-- Stage Management (Kernel 68, internal slug still 'show-runs'):
			-- backstage authority only -- Operator/Producer/Director at the
			-- venue's own location. Audience/Player no longer see this tile;
			-- curated Audience Program access happens through its own direct
			-- link/route, unaffected by map-tile visibility (Kernel 68 §1.4,
			-- §3.6).
			SELECT v.id, v.slug, v.name, v.kind, 3 AS reason_rank, 'backstage_authority_surface'::text AS visible_because
			FROM venues v
			JOIN lots l ON l.id = v.lot_id
			JOIN location_memberships lm ON lm.location_id = l.location_id
			WHERE v.slug = 'show-runs'
			  AND lm.user_id = $1
			  AND lm.active = TRUE
			  AND lm.role IN ('producer', 'director')

			UNION

			-- Stage Management crew exception: a Show Run crew roster row at
			-- this venue's location also unlocks the tile, without granting
			-- any elevated edit authority (Kernel 68 §1.4, §3.8 -- visibility
			-- only, deferred edit authority documented in the reportback).
			SELECT v.id, v.slug, v.name, v.kind, 3 AS reason_rank, 'show_run_crew_surface'::text AS visible_because
			FROM venues v
			JOIN lots l ON l.id = v.lot_id
			WHERE v.slug = 'show-runs'
			  AND EXISTS (
				SELECT 1
				FROM show_run_roster_members rm
				JOIN show_runs sr ON sr.id = rm.show_run_id
				WHERE sr.location_id = l.location_id
				  AND rm.user_id = $1
				  AND rm.role = 'crew'
				  AND rm.removed_at IS NULL
			  )
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

	if thirdPlaceReadinessChecker != nil {
		if ready, err := thirdPlaceReadinessChecker(ctx, pool, userID); err == nil && ready {
			var v VisibleVenue
			err := pool.QueryRow(ctx, `
				SELECT slug, name, kind
				FROM venues
				WHERE slug = 'third-place'
				LIMIT 1
			`).Scan(&v.Slug, &v.Name, &v.Kind)
			if err == nil && !isHiddenMainMapVenueSlug(v.Slug) {
				v.VisibleBecause = "trailer_face_ready"
				out = append(out, v)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })

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
