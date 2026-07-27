// Package projection is Kernel 74's participant-local stage projection: a
// temporary, per-Player presentation layered over the shared stage, scoped
// to the tutorial handoff.
//
// The boundary this package exists to hold (kernel-74 S11.2):
//
//	shows.current_show_scene_placement_id
//	  -> the shared, authoritative Scene for the table. Nothing here writes
//	     it. A Player entering the handoff does not move anyone else, does
//	     not change the Audience projection, and does not require Director
//	     approval.
//
//	participant_local_projections
//	  -> which Scene ONE viewer's snapshot resolves instead. Cleared when a
//	     Director+ later flies a shared Scene outside the tutorial sequence.
//
// Critically this is not a second object model. cues/types.go:43-48
// deliberately deferred per-participant object visibility because it would
// require a second show-scoped source feeding world/snapshot.go's
// visibility derivation. This package does not do that: it substitutes the
// Scene whose composition is loaded, and every element in that Scene then
// passes through the one existing filter unchanged.
package projection

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Active is one Player's live local projection.
type Active struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"user_id"`
	CharacterCardID    string    `json:"character_card_id"`
	ShowID             string    `json:"show_id"`
	DestinationSceneID string    `json:"destination_scene_id"`
	SceneSlug          string    `json:"scene_slug"`
	SceneTitle         string    `json:"scene_title"`
	EnteredAt          time.Time `json:"entered_at"`
}

// VenueLocalProjectionEnabled checks the venues.config
// participant_local_projection_enabled flag, following the same fail-closed
// capability pattern as merchant.VenueParticipantInteractionsEnabled: an
// unknown venue or a missing flag is false, never true.
func VenueLocalProjectionEnabled(ctx context.Context, pool *pgxpool.Pool, venueSlug string) (bool, error) {
	venueSlug = strings.ToLower(strings.TrimSpace(venueSlug))
	if venueSlug == "" {
		return false, nil
	}
	var enabled bool
	err := pool.QueryRow(ctx, `
		SELECT COALESCE((config ->> 'participant_local_projection_enabled')::boolean, FALSE)
		FROM venues WHERE slug = $1 LIMIT 1
	`, venueSlug).Scan(&enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return enabled, err
}

// ResolveDestinationScene resolves a destination Scene slug within a
// Location to a scene_id.
//
// The Player never supplies this (S13: "Player cannot select an arbitrary
// local projection destination") -- it comes from the authored dialogue
// packet. Resolving by (location, slug) rather than accepting an ID is what
// makes that structurally true rather than merely conventional.
func ResolveDestinationScene(ctx context.Context, pool *pgxpool.Pool, locationID, sceneSlug string) (string, error) {
	sceneSlug = strings.TrimSpace(sceneSlug)
	if sceneSlug == "" {
		return "", errors.New("destination_scene_not_configured")
	}
	var sceneID string
	err := pool.QueryRow(ctx, `
		SELECT id::text FROM scenes
		WHERE location_id = $1 AND slug = $2 AND archived_at IS NULL
		LIMIT 1
	`, locationID, sceneSlug).Scan(&sceneID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("destination_scene_not_found")
	}
	return sceneID, err
}

// Open enters a Player into a local projection, idempotently.
//
// The partial unique index on (user_id, character_card_id, show_id) WHERE
// cleared_at IS NULL means a retried Leave Ra cannot create a second
// projection for that Character (S5.3); the
// ON CONFLICT turns that from a 500 into a no-op, and the subsequent read
// returns whichever row is actually active.
func Open(ctx context.Context, pool *pgxpool.Pool, userID, characterCardID, showID, destinationSceneID, originPlacementID string) (Active, error) {
	if strings.TrimSpace(userID) == "" {
		return Active{}, errors.New("not_authenticated")
	}
	if strings.TrimSpace(destinationSceneID) == "" {
		return Active{}, errors.New("destination_scene_not_configured")
	}
	var originArg any
	if strings.TrimSpace(originPlacementID) != "" {
		originArg = originPlacementID
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO participant_local_projections (
			user_id, character_card_id, show_id, destination_scene_id, origin_show_scene_placement_id
		)
		VALUES ($1, $2, $3, $4, $5::uuid)
		ON CONFLICT DO NOTHING
	`, userID, characterCardID, showID, destinationSceneID, originArg); err != nil {
		return Active{}, err
	}
	active, err := LoadActiveForViewer(ctx, pool, userID, characterCardID, showID)
	if err != nil {
		return Active{}, err
	}
	if active == nil {
		return Active{}, errors.New("local_projection_not_opened")
	}
	return *active, nil
}

// LoadActiveForViewer returns the caller's own active projection for a
// Show and a SPECIFIC Character, or nil. This is the function
// world.LoadVenueSnapshot consults; a nil result means "render the shared
// stage", which is every viewer's normal path.
//
// characterCardID is part of the lookup, not decoration (S5.2). Keying on
// (user, show) alone was a real bug: a Player who finished the tutorial as
// one Character and then switched stayed stranded on the handoff map with a
// Character who had never played it. Switching Character now returns nil
// here, which is exactly "render the shared stage".
func LoadActiveForViewer(ctx context.Context, pool *pgxpool.Pool, userID, characterCardID, showID string) (*Active, error) {
	userID = strings.TrimSpace(userID)
	characterCardID = strings.TrimSpace(characterCardID)
	showID = strings.TrimSpace(showID)
	if userID == "" || characterCardID == "" || showID == "" {
		return nil, nil
	}
	var a Active
	err := pool.QueryRow(ctx, `
		SELECT p.id::text, p.user_id::text, p.character_card_id::text, p.show_id::text,
		       p.destination_scene_id::text, s.slug, s.title, p.entered_at
		FROM participant_local_projections p
		JOIN scenes s ON s.id = p.destination_scene_id
		WHERE p.user_id = $1 AND p.character_card_id = $2 AND p.show_id = $3 AND p.cleared_at IS NULL
		LIMIT 1
	`, userID, characterCardID, showID).Scan(
		&a.ID, &a.UserID, &a.CharacterCardID, &a.ShowID,
		&a.DestinationSceneID, &a.SceneSlug, &a.SceneTitle, &a.EnteredAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ClearForShow clears every active local projection in a Show and returns
// the affected user IDs so the caller can push each of them an
// invalidation.
//
// Called when a Director+ flies a later shared Scene (S11.3): the Player
// stops being on their private handoff map and joins whatever the Director
// just put on stage. Returning the user list is what lets that be a
// targeted push rather than relying on a session-wide broadcast.
func ClearForShow(ctx context.Context, pool *pgxpool.Pool, showID, reason string) ([]string, error) {
	showID = strings.TrimSpace(showID)
	if showID == "" {
		return nil, nil
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "shared_scene_advanced"
	}
	rows, err := pool.Query(ctx, `
		UPDATE participant_local_projections
		SET cleared_at = NOW(), cleared_reason = $2
		WHERE show_id = $1 AND cleared_at IS NULL
		RETURNING user_id::text
	`, showID, reason)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ClearForSharedSceneAdvance is the S11.3 clear rule, expressed without a
// "tutorial sequence" registry.
//
// The rule is simply: a projection survives while the shared stage is still
// on the Scene the Player left from, and clears the moment the Director
// moves the table anywhere else. That is exactly "cleared by a later shared
// Scene outside the tutorial sequence" for the tutorial's actual shape --
// the Player's projection originates at the Courtyard placement, so any
// advance away from the Courtyard clears it -- and it needs no list of
// which placements count as tutorial, which would rot the first time
// someone renames a Scene.
//
// A no-op re-fly of the same placement (a Director pressing GO twice) does
// not yank a Player out of their handoff.
func ClearForSharedSceneAdvance(ctx context.Context, pool *pgxpool.Pool, showID, newPlacementID string) ([]string, error) {
	showID = strings.TrimSpace(showID)
	if showID == "" {
		return nil, nil
	}
	var placementArg any
	if strings.TrimSpace(newPlacementID) != "" {
		placementArg = newPlacementID
	}
	rows, err := pool.Query(ctx, `
		UPDATE participant_local_projections
		SET cleared_at = NOW(), cleared_reason = 'shared_scene_advanced'
		WHERE show_id = $1
		  AND cleared_at IS NULL
		  AND (origin_show_scene_placement_id IS DISTINCT FROM $2::uuid)
		RETURNING user_id::text
	`, showID, placementArg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ClearForViewer clears one Player's projections in a Show -- the
// authorized backstage "bring this Player back to the shared stage" control
// (S11.3). Deliberately clears every Character's projection for that user:
// the operator is acting on a person who is stuck, and does not know or
// care which Character they were presenting as.
func ClearForViewer(ctx context.Context, pool *pgxpool.Pool, userID, showID, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "backstage_cleared"
	}
	_, err := pool.Exec(ctx, `
		UPDATE participant_local_projections
		SET cleared_at = NOW(), cleared_reason = $3
		WHERE user_id = $1 AND show_id = $2 AND cleared_at IS NULL
	`, strings.TrimSpace(userID), strings.TrimSpace(showID), reason)
	return err
}
