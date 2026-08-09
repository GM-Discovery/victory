package scenes

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showruns"
)

var defaultElementVisibility = json.RawMessage(`{"toRoles":["audience","cast","crew","director","producer"],"privateTo":[]}`)

// requireDirectorPlus is Director+/Producer/Operator authority only,
// deliberately narrower than canManageComposer's Crew non-destructive-edit
// allowance (kernel-85 S10: Update Current Scene / Save as New Scene are
// Director+ actions, not general composer edits).
func requireDirectorPlus(ctx context.Context, pool *pgxpool.Pool, actorUserID, locationID string) error {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return errors.New("not_authenticated")
	}
	ok, err := showruns.CanManageShowRun(ctx, pool, actorUserID, locationID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("not_authorized")
	}
	return nil
}

// UpdateCurrentScene folds a placement's Show-layer elements into its
// Scene's own Base layer (kernel-85 S6.1, "Director+ can persist the
// current supported arrangement back to the active Scene"). This is a
// promotion, not a delete-and-recreate: each Show-layer row simply has its
// show_scene_placement_id cleared, keeping its id (and any
// stage_element_bindings row) intact. Base-layer elements that already
// existed are untouched. After this call, LoadResolvedComposition for this
// placement returns exactly what it did before -- the same elements, now
// living in the Scene's reusable default instead of this one placement's
// override, so every future Show that stages this Scene starts from this
// arrangement too.
func UpdateCurrentScene(ctx context.Context, pool *pgxpool.Pool, actorUserID, placementID string) (ResolvedComposition, error) {
	p, err := LoadPlacementByID(ctx, pool, placementID)
	if err != nil {
		return ResolvedComposition{}, err
	}
	scene, err := LoadSceneByID(ctx, pool, p.SceneID)
	if err != nil {
		return ResolvedComposition{}, err
	}
	if err := requireDirectorPlus(ctx, pool, actorUserID, scene.LocationID); err != nil {
		return ResolvedComposition{}, err
	}

	if _, err := pool.Exec(ctx, `
		UPDATE scene_stage_elements SET show_scene_placement_id = NULL, updated_at = NOW()
		WHERE show_scene_placement_id = $1
	`, placementID); err != nil {
		return ResolvedComposition{}, err
	}

	return LoadResolvedComposition(ctx, pool, placementID)
}

// SaveArrangementAsNewScene captures a placement's resolved (Base+Show)
// composition as a distinct new Scene (kernel-85 S6.2). The original Scene
// and placement are never modified -- every copied element gets a fresh id,
// so this is a snapshot, the same "instance, not a live link" convention
// Kernel 82's Storyboard templates already established. Bindings
// (stage_element_bindings) are not copied: they reference the original
// element ids, and re-binding an interaction on a freshly authored Scene is
// the same authoring step as binding one on any newly created Scene.
func SaveArrangementAsNewScene(ctx context.Context, pool *pgxpool.Pool, actorUserID, placementID, newTitle, newSlug string) (Scene, error) {
	p, err := LoadPlacementByID(ctx, pool, placementID)
	if err != nil {
		return Scene{}, err
	}
	origScene, err := LoadSceneByID(ctx, pool, p.SceneID)
	if err != nil {
		return Scene{}, err
	}
	if err := requireDirectorPlus(ctx, pool, actorUserID, origScene.LocationID); err != nil {
		return Scene{}, err
	}
	newTitle = strings.TrimSpace(newTitle)
	if newTitle == "" {
		return Scene{}, errors.New("title_required")
	}
	newSlug = strings.TrimSpace(newSlug)
	if newSlug == "" {
		return Scene{}, errors.New("slug_required")
	}

	resolved, err := LoadResolvedComposition(ctx, pool, placementID)
	if err != nil {
		return Scene{}, err
	}

	defaultVenueID := ""
	if origScene.DefaultVenueID != nil {
		defaultVenueID = *origScene.DefaultVenueID
	}
	sourceProductionID := ""
	if origScene.SourceProductionID != nil {
		sourceProductionID = *origScene.SourceProductionID
	}
	newScene, err := CreateScene(ctx, pool, actorUserID, origScene.LocationID, sourceProductionID, CreateSceneInput{
		Title:          newTitle,
		Slug:           newSlug,
		DefaultVenueID: defaultVenueID,
	})
	if err != nil {
		return Scene{}, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return Scene{}, err
	}
	defer tx.Rollback(ctx)

	for _, el := range resolved.Elements {
		data := []byte(el.Data)
		if len(data) == 0 {
			data = []byte("{}")
		}
		position := []byte(el.Position)
		if len(position) == 0 {
			position = []byte("{}")
		}
		visibility := []byte(el.Visibility)
		if len(visibility) == 0 {
			visibility = defaultElementVisibility
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO scene_stage_elements (
				scene_id, show_scene_placement_id, kind, label, data, position, visibility, sort_order, width, height, created_by_user_id
			)
			VALUES ($1, NULL, $2, NULLIF($3, ''), $4::jsonb, $5::jsonb, $6::jsonb, $7, $8, $9, $10::uuid)
		`, newScene.ID, el.Kind, el.Label, data, position, visibility, el.SortOrder, el.Width, el.Height, actorUserID); err != nil {
			return Scene{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Scene{}, err
	}
	return newScene, nil
}
