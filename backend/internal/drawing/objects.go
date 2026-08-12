package drawing

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/rollaudience"
	"victory/backend/internal/venuecoordination"
)

var hexColorRE = regexp.MustCompile(`^#[0-9a-fA-F]{3,8}$`)

func resolveSessionShowID(ctx context.Context, pool *pgxpool.Pool, sessionID string) (string, error) {
	var showID *string
	err := pool.QueryRow(ctx, `SELECT show_id::text FROM sessions WHERE id = $1`, sessionID).Scan(&showID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errors.New("session_not_found")
		}
		return "", err
	}
	if showID == nil || *showID == "" {
		return "", errors.New("session_has_no_show")
	}
	return *showID, nil
}

func validColor(c string) bool {
	c = strings.TrimSpace(c)
	if c == "" {
		return true
	}
	return hexColorRE.MatchString(c) && len(c) <= 9
}

// validateGeometry enforces kernel §16's payload bounds per object type
// and requires every point/vertex to be a finite {x,y} number pair. It
// does not otherwise interpret geometry -- shape semantics stay client
// (rendering) and this-package (storage) agnostic, matching the kernel's
// "exact schema follows repository architecture" instruction.
func validateGeometry(objectType string, geometry map[string]any) error {
	if geometry == nil {
		return errors.New("geometry_required")
	}
	switch objectType {
	case TypeFreehand, TypePolyline:
		pts, ok := geometry["points"].([]any)
		if !ok || len(pts) < 2 {
			return errors.New("geometry_points_required")
		}
		if len(pts) > MaxPointsPerStroke {
			return errors.New("too_many_points")
		}
		return validatePointList(pts)
	case TypeLine:
		pts, ok := geometry["points"].([]any)
		if !ok || len(pts) != 2 {
			return errors.New("line_requires_two_points")
		}
		return validatePointList(pts)
	case TypePolygon:
		pts, ok := geometry["points"].([]any)
		if !ok || len(pts) < 3 {
			return errors.New("polygon_requires_three_points")
		}
		if len(pts) > MaxVertices {
			return errors.New("too_many_vertices")
		}
		return validatePointList(pts)
	case TypeRectangle, TypeEllipse:
		if !hasFiniteNumber(geometry["x"]) || !hasFiniteNumber(geometry["y"]) ||
			!hasFiniteNumber(geometry["width"]) || !hasFiniteNumber(geometry["height"]) {
			return errors.New("shape_bounds_required")
		}
		return nil
	case TypeText:
		if !hasFiniteNumber(geometry["x"]) || !hasFiniteNumber(geometry["y"]) {
			return errors.New("text_position_required")
		}
		return nil
	case TypeStamp:
		if !hasFiniteNumber(geometry["x"]) || !hasFiniteNumber(geometry["y"]) {
			return errors.New("stamp_position_required")
		}
		return nil
	default:
		return errors.New("unsupported_object_type")
	}
}

func validatePointList(pts []any) error {
	for _, p := range pts {
		m, ok := p.(map[string]any)
		if !ok || !hasFiniteNumber(m["x"]) || !hasFiniteNumber(m["y"]) {
			return errors.New("invalid_point")
		}
	}
	return nil
}

func hasFiniteNumber(v any) bool {
	f, ok := v.(float64)
	if !ok {
		return false
	}
	return f == f && f > -1e12 && f < 1e12 // excludes NaN, bounds absurd coordinates
}

// Create validates authority, scope, and geometry, then persists a new
// drawing object (kernel §8 "server validates ... payload/geometry
// bounds"). scope resolution reuses rollaudience.Resolve exactly as
// Kernel 86 resolves roll audience -- "cohort" (default) narrows to the
// actor's current Cohort (or falls back to Show if ungrouped/no Show),
// "show" is always Show-wide.
func Create(ctx context.Context, pool *pgxpool.Pool, reg *venuecoordination.Registry, actorUserID string, req CreateRequest) (Object, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return Object{}, errors.New("session_id_required")
	}
	if !ValidObjectTypes[req.ObjectType] {
		return Object{}, errors.New("invalid_object_type")
	}
	if err := validateGeometry(req.ObjectType, req.Geometry); err != nil {
		return Object{}, err
	}
	if !validColor(req.StrokeColor) || !validColor(req.FillColor) {
		return Object{}, errors.New("invalid_color")
	}
	lineStyle := req.LineStyle
	if lineStyle == "" {
		lineStyle = LineSolid
	}
	if !ValidLineStyles[lineStyle] {
		return Object{}, errors.New("invalid_line_style")
	}
	if req.StrokeWidth < 0 || req.StrokeWidth > 200 {
		return Object{}, errors.New("invalid_stroke_width")
	}
	if req.Opacity < 0 || req.Opacity > 1 {
		return Object{}, errors.New("invalid_opacity")
	}
	if utf8Len(req.TextContent) > MaxTextLength {
		return Object{}, errors.New("text_too_long")
	}
	if len(req.StampKey) > MaxStampKeyLength {
		return Object{}, errors.New("invalid_stamp_key")
	}
	if req.ObjectType == TypeStamp && !ValidStamps[req.StampKey] {
		return Object{}, errors.New("unknown_stamp")
	}
	if req.ObjectType == TypeText && strings.TrimSpace(req.TextContent) == "" {
		return Object{}, errors.New("text_content_required")
	}

	showID, err := resolveSessionShowID(ctx, pool, sessionID)
	if err != nil {
		return Object{}, err
	}

	allowed, err := CanDraw(ctx, pool, reg, sessionID, showID, actorUserID)
	if err != nil {
		return Object{}, err
	}
	if !allowed {
		return Object{}, errors.New("not_authorized")
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM drawing_objects WHERE show_id = $1 AND deleted_at IS NULL`, showID).Scan(&count); err != nil {
		return Object{}, err
	}
	if count >= MaxObjectsPerShow {
		return Object{}, errors.New("too_many_objects")
	}

	mode := rollaudience.NormalizeMode(req.Scope)
	if mode == "" {
		mode = rollaudience.ModeCohort
	}
	if mode != rollaudience.ModeCohort && mode != rollaudience.ModeShow {
		return Object{}, errors.New("invalid_scope")
	}
	decision, err := rollaudience.Resolve(ctx, pool, sessionID, actorUserID, mode)
	if err != nil {
		return Object{}, err
	}

	var creatorCharacterID any
	if strings.TrimSpace(req.CreatorCharacterID) != "" {
		owned, err := userOwnsCharacter(ctx, pool, actorUserID, req.CreatorCharacterID)
		if err != nil {
			return Object{}, err
		}
		if owned {
			creatorCharacterID = req.CreatorCharacterID
		}
	}

	var cohortID any
	if decision.Mode == rollaudience.ModeCohort && decision.CohortID != "" {
		cohortID = decision.CohortID
	}

	geomJSON, err := json.Marshal(req.Geometry)
	if err != nil {
		return Object{}, err
	}

	var fillColor any
	if strings.TrimSpace(req.FillColor) != "" {
		fillColor = req.FillColor
	}

	var nextZ int
	if err := pool.QueryRow(ctx, `SELECT COALESCE(MAX(z_order), 0) + 1 FROM drawing_objects WHERE show_id = $1 AND deleted_at IS NULL`, showID).Scan(&nextZ); err != nil {
		return Object{}, err
	}

	strokeColor := req.StrokeColor
	if strokeColor == "" {
		strokeColor = "#2b2622"
	}
	opacity := req.Opacity
	if opacity == 0 {
		opacity = 1
	}

	var obj Object
	err = pool.QueryRow(ctx, `
		INSERT INTO drawing_objects (
			show_id, cohort_id, creator_user_id, creator_character_id, object_type, geometry,
			stroke_color, fill_color, stroke_width, opacity, line_style, rotation, z_order,
			text_content, stamp_key
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING id::text, show_id::text, COALESCE(cohort_id::text, ''), creator_user_id::text,
			COALESCE(creator_character_id::text, ''), object_type, geometry, stroke_color,
			COALESCE(fill_color, ''), stroke_width, opacity, line_style, rotation, z_order, locked,
			COALESCE(text_content, ''), COALESCE(stamp_key, ''), created_at, updated_at
	`, showID, cohortID, actorUserID, creatorCharacterID, req.ObjectType, geomJSON,
		strokeColor, fillColor, req.StrokeWidth, opacity, lineStyle, req.Rotation, nextZ,
		nullIfEmpty(req.TextContent), nullIfEmpty(req.StampKey)).
		Scan(&obj.ID, &obj.ShowID, &obj.CohortID, &obj.CreatorUserID, &obj.CreatorCharacterID,
			&obj.ObjectType, &geomJSON, &obj.StrokeColor, &obj.FillColor, &obj.StrokeWidth,
			&obj.Opacity, &obj.LineStyle, &obj.Rotation, &obj.ZOrder, &obj.Locked,
			&obj.TextContent, &obj.StampKey, &obj.CreatedAt, &obj.UpdatedAt)
	if err != nil {
		return Object{}, err
	}
	_ = json.Unmarshal(geomJSON, &obj.Geometry)
	return obj, nil
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func utf8Len(s string) int { return len([]rune(s)) }

func userOwnsCharacter(ctx context.Context, pool *pgxpool.Pool, userID, characterID string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM character_cards WHERE id = $1 AND owner_user_id = $2)`, characterID, userID).Scan(&exists)
	return exists, err
}

// List returns every non-deleted drawing object visible to viewerUserID
// for sessionID, in ascending z-order. Cohort-scoped objects are filtered
// out for a viewer who is neither Director+ nor a member of that same
// Cohort -- kernel §1 "Do not leak Cohort drawings through live delivery,
// reconnect, snapshot, history, or export," enforced here so every
// reader (GET, reconnect, export) shares one code path.
func List(ctx context.Context, pool *pgxpool.Pool, sessionID, viewerUserID string) ([]Object, error) {
	sessionID = strings.TrimSpace(sessionID)
	showID, err := resolveSessionShowID(ctx, pool, sessionID)
	if err != nil {
		return nil, err
	}

	isDirector, err := rollaudience.IsDirectorPlus(ctx, pool, sessionID, viewerUserID)
	if err != nil {
		return nil, err
	}
	viewerCohortID, err := rollaudience.Resolve(ctx, pool, sessionID, viewerUserID, rollaudience.ModeCohort)
	if err != nil {
		return nil, err
	}

	rows, err := pool.Query(ctx, `
		SELECT id::text, show_id::text, COALESCE(cohort_id::text, ''), creator_user_id::text,
			COALESCE(creator_character_id::text, ''), object_type, geometry, stroke_color,
			COALESCE(fill_color, ''), stroke_width, opacity, line_style, rotation, z_order, locked,
			COALESCE(text_content, ''), COALESCE(stamp_key, ''), created_at, updated_at
		FROM drawing_objects
		WHERE show_id = $1 AND deleted_at IS NULL
		ORDER BY z_order ASC, created_at ASC
	`, showID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Object
	for rows.Next() {
		var obj Object
		var geomRaw []byte
		if err := rows.Scan(&obj.ID, &obj.ShowID, &obj.CohortID, &obj.CreatorUserID, &obj.CreatorCharacterID,
			&obj.ObjectType, &geomRaw, &obj.StrokeColor, &obj.FillColor, &obj.StrokeWidth,
			&obj.Opacity, &obj.LineStyle, &obj.Rotation, &obj.ZOrder, &obj.Locked,
			&obj.TextContent, &obj.StampKey, &obj.CreatedAt, &obj.UpdatedAt); err != nil {
			return nil, err
		}
		if obj.CohortID != "" && !isDirector {
			if viewerCohortID.CohortID == "" || viewerCohortID.CohortID != obj.CohortID {
				continue
			}
		}
		_ = json.Unmarshal(geomRaw, &obj.Geometry)
		out = append(out, obj)
	}
	return out, rows.Err()
}

func loadObject(ctx context.Context, pool *pgxpool.Pool, id string) (Object, error) {
	var obj Object
	var geomRaw []byte
	err := pool.QueryRow(ctx, `
		SELECT id::text, show_id::text, COALESCE(cohort_id::text, ''), creator_user_id::text,
			COALESCE(creator_character_id::text, ''), object_type, geometry, stroke_color,
			COALESCE(fill_color, ''), stroke_width, opacity, line_style, rotation, z_order, locked,
			COALESCE(text_content, ''), COALESCE(stamp_key, ''), created_at, updated_at
		FROM drawing_objects WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&obj.ID, &obj.ShowID, &obj.CohortID, &obj.CreatorUserID, &obj.CreatorCharacterID,
		&obj.ObjectType, &geomRaw, &obj.StrokeColor, &obj.FillColor, &obj.StrokeWidth,
		&obj.Opacity, &obj.LineStyle, &obj.Rotation, &obj.ZOrder, &obj.Locked,
		&obj.TextContent, &obj.StampKey, &obj.CreatedAt, &obj.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Object{}, errors.New("drawing_object_not_found")
		}
		return Object{}, err
	}
	_ = json.Unmarshal(geomRaw, &obj.Geometry)
	return obj, nil
}

// Update applies a partial edit. Authority: creator (unlocked) or
// Director+ (kernel §1). Geometry, when present, is fully replaced and
// re-validated against the object's existing type.
func Update(ctx context.Context, pool *pgxpool.Pool, actorUserID string, objectID string, req UpdateRequest) (Object, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return Object{}, errors.New("session_id_required")
	}
	obj, err := loadObject(ctx, pool, objectID)
	if err != nil {
		return Object{}, err
	}
	allowed, err := CanEditObject(ctx, pool, sessionID, actorUserID, obj)
	if err != nil {
		return Object{}, err
	}
	if !allowed {
		return Object{}, errors.New("not_authorized")
	}

	geometry := obj.Geometry
	if req.Geometry != nil {
		if err := validateGeometry(obj.ObjectType, req.Geometry); err != nil {
			return Object{}, err
		}
		geometry = req.Geometry
	}
	strokeColor := obj.StrokeColor
	if req.StrokeColor != nil {
		if !validColor(*req.StrokeColor) {
			return Object{}, errors.New("invalid_color")
		}
		strokeColor = *req.StrokeColor
	}
	fillColor := obj.FillColor
	if req.FillColor != nil {
		if !validColor(*req.FillColor) {
			return Object{}, errors.New("invalid_color")
		}
		fillColor = *req.FillColor
	}
	strokeWidth := obj.StrokeWidth
	if req.StrokeWidth != nil {
		if *req.StrokeWidth < 0 || *req.StrokeWidth > 200 {
			return Object{}, errors.New("invalid_stroke_width")
		}
		strokeWidth = *req.StrokeWidth
	}
	opacity := obj.Opacity
	if req.Opacity != nil {
		if *req.Opacity < 0 || *req.Opacity > 1 {
			return Object{}, errors.New("invalid_opacity")
		}
		opacity = *req.Opacity
	}
	lineStyle := obj.LineStyle
	if req.LineStyle != nil {
		if !ValidLineStyles[*req.LineStyle] {
			return Object{}, errors.New("invalid_line_style")
		}
		lineStyle = *req.LineStyle
	}
	rotation := obj.Rotation
	if req.Rotation != nil {
		rotation = *req.Rotation
	}
	textContent := obj.TextContent
	if req.TextContent != nil {
		if utf8Len(*req.TextContent) > MaxTextLength {
			return Object{}, errors.New("text_too_long")
		}
		textContent = *req.TextContent
	}

	geomJSON, err := json.Marshal(geometry)
	if err != nil {
		return Object{}, err
	}

	var fillArg any
	if strings.TrimSpace(fillColor) != "" {
		fillArg = fillColor
	}

	_, err = pool.Exec(ctx, `
		UPDATE drawing_objects SET
			geometry = $1, stroke_color = $2, fill_color = $3, stroke_width = $4, opacity = $5,
			line_style = $6, rotation = $7, text_content = $8, updated_at = NOW()
		WHERE id = $9 AND deleted_at IS NULL
	`, geomJSON, strokeColor, fillArg, strokeWidth, opacity, lineStyle, rotation,
		nullIfEmpty(textContent), objectID)
	if err != nil {
		return Object{}, err
	}
	return loadObject(ctx, pool, objectID)
}

// Delete soft-deletes objectID. Authority: creator (unlocked) or Director+.
func Delete(ctx context.Context, pool *pgxpool.Pool, actorUserID, sessionID, objectID string) error {
	obj, err := loadObject(ctx, pool, objectID)
	if err != nil {
		return err
	}
	allowed, err := CanEditObject(ctx, pool, sessionID, actorUserID, obj)
	if err != nil {
		return err
	}
	if !allowed {
		return errors.New("not_authorized")
	}
	_, err = pool.Exec(ctx, `UPDATE drawing_objects SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1`, objectID)
	return err
}

// SetLock sets objectID's locked flag. Authority: creator or Director+
// (creators may lock/unlock their own work; Director+ may override anyone
// -- kernel §5).
func SetLock(ctx context.Context, pool *pgxpool.Pool, actorUserID, sessionID, objectID string, locked bool) (Object, error) {
	obj, err := loadObject(ctx, pool, objectID)
	if err != nil {
		return Object{}, err
	}
	isDirector, err := rollaudience.IsDirectorPlus(ctx, pool, sessionID, actorUserID)
	if err != nil {
		return Object{}, err
	}
	if !isDirector && obj.CreatorUserID != actorUserID {
		return Object{}, errors.New("not_authorized")
	}
	_, err = pool.Exec(ctx, `UPDATE drawing_objects SET locked = $1, updated_at = NOW() WHERE id = $2`, locked, objectID)
	if err != nil {
		return Object{}, err
	}
	return loadObject(ctx, pool, objectID)
}

const (
	ZFront    = "front"
	ZBack     = "back"
	ZForward  = "forward"
	ZBackward = "backward"
)

// Reorder applies one z-order operation (kernel §5: bring forward/send
// backward/bring to front/send to back), scoped to the object's own Show
// (all objects on one Show's map share one z-order axis regardless of
// Cohort scope, so Director+ correcting order always sees consistent
// results; a Cohort-scoped object simply never competes for stacking
// order against content the Cohort itself cannot see, since List already
// filters those out client-side).
func Reorder(ctx context.Context, pool *pgxpool.Pool, actorUserID, sessionID, objectID, direction string) (Object, error) {
	obj, err := loadObject(ctx, pool, objectID)
	if err != nil {
		return Object{}, err
	}
	allowed, err := CanEditObject(ctx, pool, sessionID, actorUserID, obj)
	if err != nil {
		return Object{}, err
	}
	if !allowed {
		return Object{}, errors.New("not_authorized")
	}

	switch direction {
	case ZFront:
		var maxZ int
		if err := pool.QueryRow(ctx, `SELECT COALESCE(MAX(z_order),0) FROM drawing_objects WHERE show_id = $1 AND deleted_at IS NULL`, obj.ShowID).Scan(&maxZ); err != nil {
			return Object{}, err
		}
		if _, err := pool.Exec(ctx, `UPDATE drawing_objects SET z_order = $1, updated_at = NOW() WHERE id = $2`, maxZ+1, objectID); err != nil {
			return Object{}, err
		}
	case ZBack:
		var minZ int
		if err := pool.QueryRow(ctx, `SELECT COALESCE(MIN(z_order),0) FROM drawing_objects WHERE show_id = $1 AND deleted_at IS NULL`, obj.ShowID).Scan(&minZ); err != nil {
			return Object{}, err
		}
		if _, err := pool.Exec(ctx, `UPDATE drawing_objects SET z_order = $1, updated_at = NOW() WHERE id = $2`, minZ-1, objectID); err != nil {
			return Object{}, err
		}
	case ZForward, ZBackward:
		cmp, order := ">", "ASC"
		if direction == ZBackward {
			cmp, order = "<", "DESC"
		}
		var neighborID string
		var neighborZ int
		err := pool.QueryRow(ctx, `
			SELECT id::text, z_order FROM drawing_objects
			WHERE show_id = $1 AND deleted_at IS NULL AND id != $2 AND z_order `+cmp+` $3
			ORDER BY z_order `+order+` LIMIT 1
		`, obj.ShowID, objectID, obj.ZOrder).Scan(&neighborID, &neighborZ)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return obj, nil // already at that extreme; no-op
			}
			return Object{}, err
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return Object{}, err
		}
		defer tx.Rollback(ctx)
		if _, err := tx.Exec(ctx, `UPDATE drawing_objects SET z_order = $1, updated_at = NOW() WHERE id = $2`, neighborZ, objectID); err != nil {
			return Object{}, err
		}
		if _, err := tx.Exec(ctx, `UPDATE drawing_objects SET z_order = $1, updated_at = NOW() WHERE id = $2`, obj.ZOrder, neighborID); err != nil {
			return Object{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Object{}, err
		}
	default:
		return Object{}, errors.New("invalid_direction")
	}
	return loadObject(ctx, pool, objectID)
}
