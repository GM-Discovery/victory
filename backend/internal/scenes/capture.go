package scenes

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showruns"
)

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

// liveBridgeVenueSlug is the venue Scene Configuration's live bridge
// (CaptureLiveVenueComposition / ApplySceneToLiveVenue) reads from and
// writes to. Hardcoded rather than resolved per-Scene/Placement: every live
// improv session this bridge exists for happens in Catharsis (Kernel 93),
// and a real multi-venue resolution (placement.VenueID -> scene.
// DefaultVenueID -> ???) isn't needed by anything yet. Revisit if Scene
// Configuration ever needs to target a second live venue.
const liveBridgeVenueSlug = "catharsis"

// UpdateCurrentScene captures the live stage's current arrangement --
// every stage-surface token and world-pinned index card, the active map,
// and the grid config -- as the Scene's new Base layer (kernel-85 S6.1,
// extended 2026-08-23 for the Kernel 93 live-stage bridge: "Director+ can
// take a picture of what's live right now and have that become the Scene's
// arrangement," not just promote whatever scene_stage_elements already
// existed). This replaces the Scene's entire Base layer -- any previous
// arrangement it held is gone, matching "save configuration" meaning "this
// is what the Scene looks like now," not an incremental merge.
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

	if err := CaptureLiveVenueComposition(ctx, pool, actorUserID, p.SceneID, liveBridgeVenueSlug); err != nil {
		return ResolvedComposition{}, err
	}

	return LoadResolvedComposition(ctx, pool, placementID)
}

// SaveArrangementAsNewScene captures the live stage's current arrangement
// as a distinct new Scene (kernel-85 S6.2, extended 2026-08-23 for the
// Kernel 93 live-stage bridge). The original Scene and placement are never
// modified. Bindings (stage_element_bindings) don't carry over: they
// reference the original element ids, and re-binding an interaction on a
// freshly authored Scene is the same authoring step as binding one on any
// newly created Scene.
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

	if err := CaptureLiveVenueComposition(ctx, pool, actorUserID, newScene.ID, liveBridgeVenueSlug); err != nil {
		return Scene{}, err
	}

	return newScene, nil
}
