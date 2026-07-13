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
	id::text, production_id::text, slug, title, COALESCE(short_title, ''),
	default_venue_id::text, COALESCE(audience_title, ''), COALESCE(audience_summary, ''),
	COALESCE(player_brief, ''), COALESCE(director_notes, ''), COALESCE(operator_notes, ''),
	COALESCE(source_ref, ''), status, config_json::text, created_by_user_id::text,
	created_at, updated_at, archived_at
`

func scanScene(row pgx.Row) (Scene, error) {
	var s Scene
	var defaultVenueID, createdByUserID *string
	var configText string
	if err := row.Scan(
		&s.ID, &s.ProductionID, &s.Slug, &s.Title, &s.ShortTitle,
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
	s.DefaultVenueID = defaultVenueID
	s.CreatedByUserID = createdByUserID
	if configText != "" {
		s.ConfigJSON = json.RawMessage(configText)
	}
	return s, nil
}

// CreateScene authority-checks the actor against the Production (resolved
// server-side, never trusted from the client) and inserts a new Scene
// defaulting to status "draft". Slug uniqueness is scoped to the
// Production, enforced by the DB's UNIQUE(production_id, slug) constraint.
func CreateScene(ctx context.Context, pool *pgxpool.Pool, actorUserID, productionID string, in CreateSceneInput) (Scene, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return Scene{}, errors.New("not_authenticated")
	}
	if strings.TrimSpace(in.Title) == "" {
		return Scene{}, errors.New("title_required")
	}
	if strings.TrimSpace(in.Slug) == "" {
		return Scene{}, errors.New("slug_required")
	}

	ok, err := CanManageScenesForProduction(ctx, pool, actorUserID, productionID)
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

	row := pool.QueryRow(ctx, `
		INSERT INTO scenes (
			production_id, slug, title, short_title, default_venue_id,
			audience_title, audience_summary, player_brief, director_notes,
			operator_notes, source_ref, created_by_user_id
		)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, NULLIF($6, ''), NULLIF($7, ''),
			NULLIF($8, ''), NULLIF($9, ''), NULLIF($10, ''), NULLIF($11, ''), $12)
		RETURNING `+sceneColumns,
		productionID, in.Slug, in.Title, in.ShortTitle, defaultVenueID,
		in.AudienceTitle, in.AudienceSummary, in.PlayerBrief, in.DirectorNotes,
		in.OperatorNotes, in.SourceRef, actorUserID)
	scene, err := scanScene(row)
	if err != nil && strings.Contains(err.Error(), "scenes_production_id_slug_key") {
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

// UpdateScene authority-checks against the Scene's Production, then applies
// only the fields present in the patch.
func UpdateScene(ctx context.Context, pool *pgxpool.Pool, actorUserID, sceneID string, patch UpdateScenePatch) (Scene, error) {
	s, err := LoadSceneByID(ctx, pool, sceneID)
	if err != nil {
		return Scene{}, err
	}
	ok, err := CanManageScenesForProduction(ctx, pool, actorUserID, s.ProductionID)
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

// ListScenesForProduction returns every Scene under one Production. Callers
// must already have passed a Scenes visibility check for this Production
// before calling.
func ListScenesForProduction(ctx context.Context, pool *pgxpool.Pool, productionID string) ([]SceneSummary, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, slug, title, COALESCE(short_title, ''), default_venue_id::text, status, created_at
		FROM scenes
		WHERE production_id = $1
		ORDER BY created_at DESC
	`, productionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SceneSummary
	for rows.Next() {
		var s SceneSummary
		var defaultVenueID *string
		if err := rows.Scan(&s.ID, &s.Slug, &s.Title, &s.ShortTitle, &defaultVenueID, &s.Status, &s.CreatedAt); err != nil {
			return nil, err
		}
		s.DefaultVenueID = defaultVenueID
		out = append(out, s)
	}
	return out, rows.Err()
}
