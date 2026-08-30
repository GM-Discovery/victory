// Kernel 93 sub-kernel: bridges Scene Configuration to the live stage.
//
// Before this file, "Scene Configuration" (Activate/Update/Save as New
// Scene) only ever read and wrote scene_stage_elements -- a completely
// separate table from the live stage's own elements/venue_layout_elements,
// with no code path connecting them (see LoadResolvedComposition, which
// only ever queried scene_stage_elements). Activating a Scene changed which
// composition was considered "current" for interaction-button purposes,
// but never touched what was actually rendered on stage.
//
// Grant's ask (2026-08-23): build stage compositions live, in the venue,
// using the normal token/card/map/grid tools already in place -- then
// "take a picture" of the current live arrangement via Scene Configuration,
// and switch between two or more such pictures later with a hard cut (no
// transition animation), visible to every connected viewer including
// Audience. This file is that bridge: CaptureLiveVenueComposition freezes
// the live stage into a Scene's own scene_stage_elements; ApplySceneToLive
// Venue does the reverse, replacing the live stage with a Scene's stored
// composition.
package scenes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/venues"
)

// resolveLiveLibraryID mirrors actions/indexcard.go's resolveIndexCardLibrary
// and actions/token.go's identical helper -- small, deliberate per-package
// duplication of a find-or-create query, matching this codebase's
// established convention (see assets/read.go's assetIsActiveTheaterMap for
// the same rule applied elsewhere) rather than exporting an internal helper
// across an unrelated package boundary.
func resolveLiveLibraryID(ctx context.Context, tx pgx.Tx, locationID string) (string, error) {
	var libraryID string
	err := tx.QueryRow(ctx, `
		SELECT id::text FROM libraries WHERE location_id = $1 AND name = 'house-library' LIMIT 1
	`, locationID).Scan(&libraryID)
	if err == nil {
		return libraryID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO libraries (location_id, name) VALUES ($1, 'house-library') RETURNING id::text
	`, locationID).Scan(&libraryID); err != nil {
		return "", err
	}
	return libraryID, nil
}

// CaptureLiveVenueComposition snapshots venueSlug's current live stage --
// every stage-surface token, every world-pinned index card, the active map,
// and the grid config -- into sceneID's own base-layer scene_stage_elements
// (show_scene_placement_id IS NULL), replacing whatever that scene's
// base-layer composition previously held. Deliberately excludes tray-only
// (screen-pinned) index cards: those are a viewer's own floating notes, not
// part of the shared stage a scene transition should capture or restore.
func CaptureLiveVenueComposition(ctx context.Context, pool *pgxpool.Pool, actorUserID, sceneID, venueSlug string) error {
	venueID, locationID, err := venues.ResolveVenueLocation(ctx, pool, venueSlug)
	if err != nil {
		return err
	}
	_ = locationID

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Kernel 93 live-testing bug (2026-08-29): this used to delete every
	// Base-layer row for the scene with no filter at all, including
	// interaction_hotspot elements (e.g. Kessa's door) -- which the
	// re-population loop below never recreates, since hotspots aren't part
	// of the live venue_layout_elements/elements tables it reads from in
	// the first place (they render straight from scene_stage_elements, see
	// world/snapshot.go's loadCompositionRows). A first fix scoped the
	// delete to kind IN ('token', 'index_card', 'map_backdrop',
	// 'grid_config') -- which stopped hotspots from being eaten, but Kessa
	// herself is ALSO kind='token' and was still deleted by the very next
	// Capture, with nothing in the live tables to restore her from (she was
	// never a genuine live-table element to begin with -- see
	// merchant/prepare.go's direct CreateSceneStageElement). The real
	// distinction isn't kind at all: Kessa and the door are authored,
	// bound fixtures (stage_element_bindings), not ad-hoc live placements,
	// and Capture -- a completely separate system from whatever created
	// them -- must never delete anything bound like that, regardless of
	// its kind.
	if _, err := tx.Exec(ctx, `
		DELETE FROM scene_stage_elements sse
		WHERE sse.scene_id = $1 AND sse.show_scene_placement_id IS NULL
		  AND sse.kind IN ('token', 'index_card', 'map_backdrop', 'grid_config')
		  AND NOT EXISTS (
		    SELECT 1 FROM stage_element_bindings seb
		    WHERE seb.scene_stage_element_id = sse.id
		  )
	`, sceneID); err != nil {
		return err
	}

	rows, err := tx.Query(ctx, `
		SELECT e.element_type, e.name, e.data, vle.position, vle.visibility
		FROM venue_layout_elements vle
		JOIN elements e ON e.id = vle.element_id
		WHERE vle.venue_id = $1
		  AND (
		    (vle.surface = 'stage' AND e.element_type = 'token')
		    OR (e.element_type = 'index_card' AND e.data->>'pin_mode' = 'world')
		  )
		ORDER BY e.created_at ASC
	`, venueID)
	if err != nil {
		return err
	}
	type liveElement struct {
		elementType string
		name        string
		data        []byte
		position    []byte
		visibility  []byte
	}
	var liveElements []liveElement
	for rows.Next() {
		var le liveElement
		if err := rows.Scan(&le.elementType, &le.name, &le.data, &le.position, &le.visibility); err != nil {
			rows.Close()
			return err
		}
		liveElements = append(liveElements, le)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	sortOrder := 0
	for _, le := range liveElements {
		// label round-trips the live element's own display name (Kernel
		// 93 live-testing bug, 2026-08-29): ApplySceneToLiveVenue names a
		// recreated token from data["asset_name"], which Kessa's data
		// deliberately never has (backend/internal/merchant/prepare.go
		// strips it on every repair -- her real name lives in this label
		// column). Capturing it here is what makes that round-trip work
		// for any token, not just ones whose data happens to include
		// asset_name.
		if _, err := tx.Exec(ctx, `
			INSERT INTO scene_stage_elements (scene_id, kind, label, data, position, visibility, sort_order, created_by_user_id)
			VALUES ($1, $2, $3, $4::jsonb, $5::jsonb, $6::jsonb, $7, $8)
		`, sceneID, le.elementType, le.name, le.data, le.position, le.visibility, sortOrder, actorUserID); err != nil {
			return err
		}
		sortOrder++
	}

	var mapAssetID, mapFit, mapDisplayMode string
	var mapCropX, mapCropY, mapScale float64
	var mapSafeMargin int
	err = tx.QueryRow(ctx, `
		SELECT asset_id::text, fit, crop_x, crop_y, scale, safe_margin, display_mode
		FROM venue_active_maps WHERE venue_id = $1
	`, venueID).Scan(&mapAssetID, &mapFit, &mapCropX, &mapCropY, &mapScale, &mapSafeMargin, &mapDisplayMode)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if err == nil {
		mapData, _ := json.Marshal(map[string]any{
			"asset_id": mapAssetID, "fit": mapFit, "crop_x": mapCropX, "crop_y": mapCropY,
			"scale": mapScale, "safe_margin": mapSafeMargin, "display_mode": mapDisplayMode,
		})
		if _, err := tx.Exec(ctx, `
			INSERT INTO scene_stage_elements (scene_id, kind, data, sort_order, created_by_user_id)
			VALUES ($1, $2, $3::jsonb, $4, $5)
		`, sceneID, StageElementKindMapBackdrop, mapData, sortOrder, actorUserID); err != nil {
			return err
		}
		sortOrder++
	}
	// No active map is itself a valid captured state: simply write no
	// map_backdrop element, so Activate later leaves the venue blank,
	// matching the live "no map = blank" behavior (2026-08-23).

	var gridType, gridHexOrientation, gridLineStyle string
	var gridCellSize, gridOffsetX, gridOffsetY, gridLineWidth, gridOpacity float64
	var gridVisible bool
	err = tx.QueryRow(ctx, `
		SELECT grid_type, hex_orientation, cell_size, offset_x, offset_y, line_width, opacity, line_style, visible
		FROM venue_grid_configs WHERE venue_id = $1
	`, venueID).Scan(&gridType, &gridHexOrientation, &gridCellSize, &gridOffsetX, &gridOffsetY, &gridLineWidth, &gridOpacity, &gridLineStyle, &gridVisible)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if err == nil {
		gridData, _ := json.Marshal(map[string]any{
			"grid_type": gridType, "hex_orientation": gridHexOrientation, "cell_size": gridCellSize,
			"offset_x": gridOffsetX, "offset_y": gridOffsetY, "line_width": gridLineWidth,
			"opacity": gridOpacity, "line_style": gridLineStyle, "visible": gridVisible,
		})
		if _, err := tx.Exec(ctx, `
			INSERT INTO scene_stage_elements (scene_id, kind, data, sort_order, created_by_user_id)
			VALUES ($1, $2, $3::jsonb, $4, $5)
		`, sceneID, StageElementKindGridConfig, gridData, sortOrder, actorUserID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// ApplySceneToLiveVenue replaces venueSlug's current live stage (every
// stage-surface token and world-pinned index card, the active map, and the
// grid config) with placementID's own resolved composition -- a hard cut,
// deliberately: Grant asked for "no fancy transitions," just a new
// configuration in place of the old one. Broadcasting so every connected
// viewer (Audience included) actually sees the change is the caller's
// responsibility: this only writes the new state, matching how
// SetCurrentScenePlacement/ActivateSceneForCohort already leave broadcasting
// to their own HTTP handlers.
func ApplySceneToLiveVenue(ctx context.Context, pool *pgxpool.Pool, actorUserID, venueSlug, placementID string) error {
	venueID, locationID, err := venues.ResolveVenueLocation(ctx, pool, venueSlug)
	if err != nil {
		return err
	}

	resolved, err := LoadResolvedComposition(ctx, pool, placementID)
	if err != nil {
		return err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Clear the live stage first -- elements table row deletion cascades to
	// its venue_layout_elements row (ON DELETE CASCADE, migrations/001_
	// init.sql). Screen-pinned (tray-only) index cards are deliberately left
	// alone, same reasoning as the capture side.
	if _, err := tx.Exec(ctx, `
		DELETE FROM elements
		WHERE id IN (
			SELECT vle.element_id
			FROM venue_layout_elements vle
			JOIN elements e ON e.id = vle.element_id
			WHERE vle.venue_id = $1
			  AND (
			    (vle.surface = 'stage' AND e.element_type = 'token')
			    OR (e.element_type = 'index_card' AND e.data->>'pin_mode' = 'world')
			  )
		)
	`, venueID); err != nil {
		return err
	}

	libraryID, err := resolveLiveLibraryID(ctx, tx, locationID)
	if err != nil {
		return err
	}

	sawMapBackdrop := false
	for _, el := range resolved.Elements {
		if el.Binding != nil {
			// A bound element (Kessa, the door) is already rendered directly
			// from scene_stage_elements by world/snapshot.go's
			// loadCompositionRows, entirely independent of this live-table
			// bridge -- it never needed a live-table mirror. Blindly copying
			// el.Position here also actively broke it: Position is Kessa's
			// normalized 0-1 composition coordinate ({0.503, 0.55}), but a
			// live token's position is read as raw pixels, so the copy
			// rendered one pixel from the stage's top-left corner --
			// present, but invisible to everyone. Skipping bound elements
			// here fixes both the duplicate and the mispositioning at once.
			continue
		}
		var data map[string]any
		_ = json.Unmarshal(el.Data, &data)

		switch el.Kind {
		case StageElementKindToken:
			// el.Label (the scene_stage_elements row's own label column,
			// now round-tripped by CaptureLiveVenueComposition above) wins
			// over data["asset_name"] -- a captured element may have a
			// label but no asset_name at all (Kessa's data deliberately
			// never has one; see prepare.go's repair), and unconditionally
			// falling back to literal "Token" lost her name entirely.
			name := el.Label
			if name == "" {
				name = stringFromAny(data["asset_name"], "Token")
			}
			var elementID string
			if err := tx.QueryRow(ctx, `
				INSERT INTO elements (library_id, name, slug, element_type, context_class, state, data)
				VALUES ($1, $2, $3, 'token', 'token', 'library', $4::jsonb)
				RETURNING id::text
			`, libraryID, name, uniqueLiveSlug("token"), el.Data).Scan(&elementID); err != nil {
				return err
			}
			position := el.Position
			if len(position) == 0 {
				position = []byte(`{"anchor":"stage","x":0,"y":0,"z":0}`)
			}
			visibility := el.Visibility
			if len(visibility) == 0 {
				visibility = []byte(`{"toRoles":["audience","cast","crew","director","producer"],"privateTo":[],"visible":true,"nameplate_visible":true,"locked":false}`)
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO venue_layout_elements (venue_id, element_id, surface, position, visibility, is_default)
				VALUES ($1, $2, 'stage', $3::jsonb, $4::jsonb, FALSE)
			`, venueID, elementID, position, visibility); err != nil {
				return err
			}

		case StageElementKindIndexCard:
			var elementID string
			if err := tx.QueryRow(ctx, `
				INSERT INTO elements (library_id, name, slug, element_type, context_class, state, data)
				VALUES ($1, $2, $3, 'index_card', 'card', 'library', $4::jsonb)
				RETURNING id::text
			`, libraryID, stringFromAny(data["front_text"], "Index card"), uniqueLiveSlug("index-card"), el.Data).Scan(&elementID); err != nil {
				return err
			}
			position := el.Position
			if len(position) == 0 {
				position = []byte(`{"anchor":"stage","x":0,"y":0,"z":0}`)
			}
			visibility := el.Visibility
			if len(visibility) == 0 {
				visibility = []byte(`{"toRoles":["audience","cast","crew","director","producer"],"privateTo":[],"visible":true,"nameplate_visible":true,"locked":false}`)
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO venue_layout_elements (venue_id, element_id, surface, position, visibility, is_default)
				VALUES ($1, $2, 'stage', $3::jsonb, $4::jsonb, FALSE)
			`, venueID, elementID, position, visibility); err != nil {
				return err
			}

		case StageElementKindMapBackdrop:
			sawMapBackdrop = true
			assetID := stringFromAny(data["asset_id"], "")
			if assetID == "" {
				continue
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO venue_active_maps (
					venue_id, asset_id, fit, crop_x, crop_y, scale, safe_margin, display_mode,
					created_by_user_id, updated_by_user_id, updated_at
				)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9, NOW())
				ON CONFLICT (venue_id) DO UPDATE
				SET asset_id = EXCLUDED.asset_id, fit = EXCLUDED.fit, crop_x = EXCLUDED.crop_x,
				    crop_y = EXCLUDED.crop_y, scale = EXCLUDED.scale, safe_margin = EXCLUDED.safe_margin,
				    display_mode = EXCLUDED.display_mode, updated_by_user_id = EXCLUDED.updated_by_user_id,
				    updated_at = NOW()
			`, venueID, assetID,
				stringFromAny(data["fit"], "contain"), floatFromAny(data["crop_x"], 0.5), floatFromAny(data["crop_y"], 0.5),
				floatFromAny(data["scale"], 1), int(floatFromAny(data["safe_margin"], 24)), stringFromAny(data["display_mode"], "theater"),
				actorUserID); err != nil {
				return err
			}

		case StageElementKindGridConfig:
			if _, err := tx.Exec(ctx, `
				INSERT INTO venue_grid_configs (
					venue_id, grid_type, hex_orientation, cell_size, offset_x, offset_y,
					line_width, opacity, line_style, visible, updated_by_user_id, updated_at
				)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
				ON CONFLICT (venue_id) DO UPDATE
				SET grid_type = EXCLUDED.grid_type, hex_orientation = EXCLUDED.hex_orientation,
				    cell_size = EXCLUDED.cell_size, offset_x = EXCLUDED.offset_x, offset_y = EXCLUDED.offset_y,
				    line_width = EXCLUDED.line_width, opacity = EXCLUDED.opacity, line_style = EXCLUDED.line_style,
				    visible = EXCLUDED.visible, updated_by_user_id = EXCLUDED.updated_by_user_id, updated_at = NOW()
			`, venueID, stringFromAny(data["grid_type"], "none"), stringFromAny(data["hex_orientation"], "flat-top"),
				floatFromAny(data["cell_size"], 50), floatFromAny(data["offset_x"], 0), floatFromAny(data["offset_y"], 0),
				floatFromAny(data["line_width"], 1), floatFromAny(data["opacity"], 0.45), stringFromAny(data["line_style"], "neutral"),
				boolFromAny(data["visible"], true), actorUserID); err != nil {
				return err
			}
		}
	}

	// The captured scene had no map_backdrop element at all, meaning "no
	// map" was itself the captured state -- remove any map currently active
	// on the live venue so Activate actually reproduces that blank state,
	// rather than leaving whatever map happened to be there before.
	if !sawMapBackdrop {
		if _, err := tx.Exec(ctx, `DELETE FROM venue_active_maps WHERE venue_id = $1`, venueID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func stringFromAny(v any, fallback string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return fallback
}

func floatFromAny(v any, fallback float64) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return fallback
}

func boolFromAny(v any, fallback bool) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return fallback
}

var liveSlugCounter uint64

// uniqueLiveSlug mirrors actions/token.go's uniqueTokenSlug and actions/
// indexcard.go's uniqueIndexCardSlug (small, deliberate per-package
// duplication, same convention as resolveLiveLibraryID above) -- these
// elements aren't user-named, so a timestamp+counter suffix is sufficient
// rather than slugifying a title.
func uniqueLiveSlug(prefix string) string {
	n := atomic.AddUint64(&liveSlugCounter, 1)
	return fmt.Sprintf("%s-scene-%d-%d", prefix, time.Now().UnixNano(), n)
}
