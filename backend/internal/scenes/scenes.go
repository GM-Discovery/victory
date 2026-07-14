package scenes

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const sceneColumns = `
	id::text, location_id::text, source_production_id::text, slug, title, COALESCE(short_title, ''),
	default_venue_id::text, COALESCE(audience_title, ''), COALESCE(audience_summary, ''),
	COALESCE(player_brief, ''), COALESCE(director_notes, ''), COALESCE(operator_notes, ''),
	COALESCE(source_ref, ''), status, config_json::text, created_by_user_id::text,
	created_at, updated_at, archived_at
`

func scanScene(row pgx.Row) (Scene, error) {
	var s Scene
	var sourceProductionID, defaultVenueID, createdByUserID *string
	var configText string
	if err := row.Scan(
		&s.ID, &s.LocationID, &sourceProductionID, &s.Slug, &s.Title, &s.ShortTitle,
		&defaultVenueID, &s.AudienceTitle, &s.AudienceSummary,
		&s.PlayerBrief, &s.DirectorNotes, &s.OperatorNotes,
		&s.SourceRef, &s.Status, &configText, &createdByUserID,
		&s.CreatedAt, &s.UpdatedAt, &s.ArchivedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Scene{}, errors.New("scene_not_found")
		}
		return Scene{}, err
	}
	s.SourceProductionID = sourceProductionID
	s.DefaultVenueID = defaultVenueID
	s.CreatedByUserID = createdByUserID
	if configText != "" {
		s.ConfigJSON = json.RawMessage(configText)
	}
	return s, nil
}

// CreateScene authority-checks the actor against the target location
// (resolved server-side, never trusted from the client) and inserts a new
// Scene defaulting to status "draft". Slug uniqueness is scoped to the
// location, enforced by the DB's UNIQUE(location_id, slug) constraint.
// sourceProductionID is optional provenance only -- it never gates reuse
// (Kernel 70 SS1.2, SS3.1).
func CreateScene(ctx context.Context, pool *pgxpool.Pool, actorUserID, locationID, sourceProductionID string, in CreateSceneInput) (Scene, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return Scene{}, errors.New("not_authenticated")
	}
	locationID = strings.TrimSpace(locationID)
	if locationID == "" {
		return Scene{}, errors.New("location_id_required")
	}
	if strings.TrimSpace(in.Title) == "" {
		return Scene{}, errors.New("title_required")
	}
	if strings.TrimSpace(in.Slug) == "" {
		return Scene{}, errors.New("slug_required")
	}

	ok, err := CanManageScenesForLocation(ctx, pool, actorUserID, locationID)
	if err != nil {
		return Scene{}, err
	}
	if !ok {
		return Scene{}, errors.New("not_authorized")
	}

	var defaultVenueID *string
	if v := strings.TrimSpace(in.DefaultVenueID); v != "" {
		defaultVenueID = &v
	}
	var sourceProductionPtr *string
	if v := strings.TrimSpace(sourceProductionID); v != "" {
		sourceProductionPtr = &v
	}

	row := pool.QueryRow(ctx, `
		INSERT INTO scenes (
			location_id, source_production_id, slug, title, short_title, default_venue_id,
			audience_title, audience_summary, player_brief, director_notes,
			operator_notes, source_ref, created_by_user_id
		)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, NULLIF($7, ''), NULLIF($8, ''),
			NULLIF($9, ''), NULLIF($10, ''), NULLIF($11, ''), NULLIF($12, ''), $13)
		RETURNING `+sceneColumns,
		locationID, sourceProductionPtr, in.Slug, in.Title, in.ShortTitle, defaultVenueID,
		in.AudienceTitle, in.AudienceSummary, in.PlayerBrief, in.DirectorNotes,
		in.OperatorNotes, in.SourceRef, actorUserID)
	scene, err := scanScene(row)
	if err != nil && strings.Contains(err.Error(), "scenes_location_id_slug_key") {
		return Scene{}, errors.New("slug_already_used")
	}
	return scene, err
}

// LoadSceneByID returns the raw row with no authority check -- callers that
// expose this to a viewer must authority-check separately, matching Show's
// own "existence isn't secret" convention.
func LoadSceneByID(ctx context.Context, pool *pgxpool.Pool, sceneID string) (Scene, error) {
	sceneID = strings.TrimSpace(sceneID)
	if sceneID == "" {
		return Scene{}, errors.New("scene_not_found")
	}
	row := pool.QueryRow(ctx, `SELECT `+sceneColumns+` FROM scenes WHERE id = $1`, sceneID)
	return scanScene(row)
}

// UpdateScene authority-checks against the Scene's location, then applies
// only the fields present in the patch.
func UpdateScene(ctx context.Context, pool *pgxpool.Pool, actorUserID, sceneID string, patch UpdateScenePatch) (Scene, error) {
	s, err := LoadSceneByID(ctx, pool, sceneID)
	if err != nil {
		return Scene{}, err
	}
	ok, err := CanManageScenesForLocation(ctx, pool, actorUserID, s.LocationID)
	if err != nil {
		return Scene{}, err
	}
	if !ok {
		return Scene{}, errors.New("not_authorized")
	}

	title := s.Title
	if patch.Title != nil {
		title = *patch.Title
	}
	shortTitle := s.ShortTitle
	if patch.ShortTitle != nil {
		shortTitle = *patch.ShortTitle
	}
	defaultVenueID := s.DefaultVenueID
	if patch.DefaultVenueID != nil {
		if strings.TrimSpace(*patch.DefaultVenueID) == "" {
			defaultVenueID = nil
		} else {
			defaultVenueID = patch.DefaultVenueID
		}
	}
	audienceTitle := s.AudienceTitle
	if patch.AudienceTitle != nil {
		audienceTitle = *patch.AudienceTitle
	}
	audienceSummary := s.AudienceSummary
	if patch.AudienceSummary != nil {
		audienceSummary = *patch.AudienceSummary
	}
	playerBrief := s.PlayerBrief
	if patch.PlayerBrief != nil {
		playerBrief = *patch.PlayerBrief
	}
	directorNotes := s.DirectorNotes
	if patch.DirectorNotes != nil {
		directorNotes = *patch.DirectorNotes
	}
	operatorNotes := s.OperatorNotes
	if patch.OperatorNotes != nil {
		operatorNotes = *patch.OperatorNotes
	}
	sourceRef := s.SourceRef
	if patch.SourceRef != nil {
		sourceRef = *patch.SourceRef
	}
	status := s.Status
	if patch.Status != nil {
		status = *patch.Status
	}

	archivedAt := "NULL"
	if status == "archived" {
		archivedAt = "NOW()"
	}

	row := pool.QueryRow(ctx, `
		UPDATE scenes
		SET title = $2, short_title = NULLIF($3, ''), default_venue_id = $4,
		    audience_title = NULLIF($5, ''), audience_summary = NULLIF($6, ''),
		    player_brief = NULLIF($7, ''), director_notes = NULLIF($8, ''),
		    operator_notes = NULLIF($9, ''), source_ref = NULLIF($10, ''),
		    status = $11, archived_at = `+archivedAt+`, updated_at = NOW()
		WHERE id = $1
		RETURNING `+sceneColumns,
		sceneID, title, shortTitle, defaultVenueID, audienceTitle, audienceSummary,
		playerBrief, directorNotes, operatorNotes, sourceRef, status)
	return scanScene(row)
}

// ArchiveScene is a thin wrapper over UpdateScene's status field. Archiving
// a reusable Scene prevents new placements (CreatePlacement rejects a
// non-active Scene) but never touches existing placements -- they remain
// exactly as staged (Kernel 69 SS4.2).
func ArchiveScene(ctx context.Context, pool *pgxpool.Pool, actorUserID, sceneID string) (Scene, error) {
	status := "archived"
	return UpdateScene(ctx, pool, actorUserID, sceneID, UpdateScenePatch{Status: &status})
}

// ListScenesForLocation returns every Scene reusable at one Victory
// location, regardless of which Production originated it (Kernel 70
// SS3.4 -- the Scene Library and Show Scene picker expose reusable Scenes
// from the Show's location, not only its Production). Callers must already
// have passed a Scenes visibility check for this location before calling.
func ListScenesForLocation(ctx context.Context, pool *pgxpool.Pool, locationID string) ([]SceneSummary, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, location_id::text, source_production_id::text, slug, title,
		       COALESCE(short_title, ''), default_venue_id::text, status, created_at
		FROM scenes
		WHERE location_id = $1
		ORDER BY created_at DESC
	`, locationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SceneSummary
	for rows.Next() {
		var s SceneSummary
		var sourceProductionID, defaultVenueID *string
		if err := rows.Scan(&s.ID, &s.LocationID, &sourceProductionID, &s.Slug, &s.Title, &s.ShortTitle, &defaultVenueID, &s.Status, &s.CreatedAt); err != nil {
			return nil, err
		}
		s.SourceProductionID = sourceProductionID
		s.DefaultVenueID = defaultVenueID
		out = append(out, s)
	}
	return out, rows.Err()
}
