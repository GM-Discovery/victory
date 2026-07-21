package merchant

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showruns"
)

const equipmentItemColumns = `
	id::text, location_id::text, COALESCE(production_id::text, ''), name, slug,
	COALESCE(image_asset_id::text, ''), short_description, descriptors_json,
	quantity_mode, active, COALESCE(created_by_user_id::text, ''), created_at, updated_at,
	category, cost_credits, stats_json
`

func scanEquipmentItem(row pgx.Row) (EquipmentItem, error) {
	var e EquipmentItem
	var descriptorsRaw []byte
	var statsRaw []byte
	if err := row.Scan(
		&e.ID, &e.LocationID, &e.ProductionID, &e.Name, &e.Slug,
		&e.ImageAssetID, &e.ShortDescription, &descriptorsRaw,
		&e.QuantityMode, &e.Active, &e.CreatedByUserID, &e.CreatedAt, &e.UpdatedAt,
		&e.Category, &e.CostCredits, &statsRaw,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return EquipmentItem{}, errors.New("equipment_item_not_found")
		}
		return EquipmentItem{}, err
	}
	_ = json.Unmarshal(descriptorsRaw, &e.DescriptorsJSON)
	_ = json.Unmarshal(statsRaw, &e.StatsJSON)
	return e, nil
}

// LoadEquipmentItemByID is used both by the editor and by the purchase path
// (to check active + build the confirmation payload).
func LoadEquipmentItemByID(ctx context.Context, pool *pgxpool.Pool, id string) (EquipmentItem, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return EquipmentItem{}, errors.New("equipment_item_not_found")
	}
	row := pool.QueryRow(ctx, `SELECT `+equipmentItemColumns+` FROM equipment_items WHERE id = $1`, id)
	return scanEquipmentItem(row)
}

// ListEquipmentItems is the minimal equipment editor's list view (spec
// S5.4) -- every item at a location, active and archived, for management.
func ListEquipmentItems(ctx context.Context, pool *pgxpool.Pool, locationID string) ([]EquipmentItem, error) {
	rows, err := pool.Query(ctx, `
		SELECT `+equipmentItemColumns+`
		FROM equipment_items
		WHERE location_id = $1
		ORDER BY name ASC
	`, locationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []EquipmentItem
	for rows.Next() {
		e, err := scanEquipmentItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

type EquipmentItemInput struct {
	Name             string
	Slug             string
	ImageAssetID     string
	ShortDescription string
	Descriptors      []string
	QuantityMode     string
	Active           *bool
	// Category/CostCredits/StatsJSON are reference-only catalog fields
	// (Kernel 73 follow-up) -- see EquipmentItem's doc comment.
	Category    string
	CostCredits *float64
	StatsJSON   map[string]any
}

func canManageEquipment(ctx context.Context, pool *pgxpool.Pool, actorUserID, locationID string) (bool, error) {
	// Same authority the Director authoring surface for Cues uses (spec
	// S11: "Crew may edit only if the existing non-destructive authority
	// model safely covers this action") -- no new authority concept.
	return showruns.CanCrewPerformNonDestructiveEdit(ctx, pool, actorUserID, locationID)
}

// CreateEquipmentItem is the minimal reusable equipment editor's create path
// (spec S5.4). Supports only: name, image, short description, descriptors,
// stackable behavior, active state -- no slots/encumbrance/durability/
// resale/crafting/weight/price fields exist to set.
func CreateEquipmentItem(ctx context.Context, pool *pgxpool.Pool, actorUserID, locationID string, in EquipmentItemInput) (EquipmentItem, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	locationID = strings.TrimSpace(locationID)
	if actorUserID == "" {
		return EquipmentItem{}, errors.New("not_authenticated")
	}
	name := strings.TrimSpace(in.Name)
	slug := strings.ToLower(strings.TrimSpace(in.Slug))
	if name == "" || slug == "" {
		return EquipmentItem{}, errors.New("name_and_slug_required")
	}
	quantityMode := strings.TrimSpace(in.QuantityMode)
	if quantityMode == "" {
		quantityMode = QuantityModeStackable
	}
	if quantityMode != QuantityModeStackable && quantityMode != QuantityModeUnique {
		return EquipmentItem{}, errors.New("invalid_quantity_mode")
	}

	allowed, err := canManageEquipment(ctx, pool, actorUserID, locationID)
	if err != nil {
		return EquipmentItem{}, err
	}
	if !allowed {
		return EquipmentItem{}, errors.New("not_authorized")
	}

	active := true
	if in.Active != nil {
		active = *in.Active
	}
	descriptorsJSON, _ := json.Marshal(in.Descriptors)
	if in.Descriptors == nil {
		descriptorsJSON = []byte("[]")
	}

	var imageAssetID any
	if strings.TrimSpace(in.ImageAssetID) != "" {
		imageAssetID = in.ImageAssetID
	}
	statsJSON, _ := json.Marshal(in.StatsJSON)
	if in.StatsJSON == nil {
		statsJSON = []byte("{}")
	}

	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO equipment_items (
			location_id, name, slug, image_asset_id, short_description,
			descriptors_json, quantity_mode, active, created_by_user_id,
			category, cost_credits, stats_json
		)
		VALUES ($1::uuid, $2, $3, $4::uuid, $5, $6::jsonb, $7, $8, $9::uuid, $10, $11, $12::jsonb)
		RETURNING id::text
	`, locationID, name, slug, imageAssetID, strings.TrimSpace(in.ShortDescription),
		descriptorsJSON, quantityMode, active, actorUserID,
		strings.TrimSpace(in.Category), in.CostCredits, statsJSON).Scan(&id); err != nil {
		return EquipmentItem{}, err
	}
	return LoadEquipmentItemByID(ctx, pool, id)
}

type EquipmentItemPatch struct {
	Name             *string
	ImageAssetID     *string
	ShortDescription *string
	Descriptors      *[]string
	Active           *bool
	// Category/CostCredits/StatsJSON are reference-only catalog fields
	// (Kernel 73 follow-up) -- see EquipmentItem's doc comment. A provided
	// CostCredits always sets the value; there is no patch-level way to
	// clear it back to NULL (matches this editor's existing minor scope --
	// unwired to any frontend yet).
	Category    *string
	CostCredits *float64
	StatsJSON   *map[string]any
}

func UpdateEquipmentItem(ctx context.Context, pool *pgxpool.Pool, actorUserID, id string, patch EquipmentItemPatch) (EquipmentItem, error) {
	existing, err := LoadEquipmentItemByID(ctx, pool, id)
	if err != nil {
		return EquipmentItem{}, err
	}
	allowed, err := canManageEquipment(ctx, pool, actorUserID, existing.LocationID)
	if err != nil {
		return EquipmentItem{}, err
	}
	if !allowed {
		return EquipmentItem{}, errors.New("not_authorized")
	}

	name := existing.Name
	if patch.Name != nil {
		name = strings.TrimSpace(*patch.Name)
	}
	shortDescription := existing.ShortDescription
	if patch.ShortDescription != nil {
		shortDescription = strings.TrimSpace(*patch.ShortDescription)
	}
	active := existing.Active
	if patch.Active != nil {
		active = *patch.Active
	}
	descriptorsJSON, _ := json.Marshal(existing.DescriptorsJSON)
	if patch.Descriptors != nil {
		descriptorsJSON, _ = json.Marshal(*patch.Descriptors)
	}
	var imageAssetID any
	imageAssetIDStr := existing.ImageAssetID
	if patch.ImageAssetID != nil {
		imageAssetIDStr = strings.TrimSpace(*patch.ImageAssetID)
	}
	if imageAssetIDStr != "" {
		imageAssetID = imageAssetIDStr
	}
	category := existing.Category
	if patch.Category != nil {
		category = strings.TrimSpace(*patch.Category)
	}
	costCredits := existing.CostCredits
	if patch.CostCredits != nil {
		costCredits = patch.CostCredits
	}
	statsJSON, _ := json.Marshal(existing.StatsJSON)
	if existing.StatsJSON == nil {
		statsJSON = []byte("{}")
	}
	if patch.StatsJSON != nil {
		statsJSON, _ = json.Marshal(*patch.StatsJSON)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE equipment_items
		SET name = $2, image_asset_id = $3::uuid, short_description = $4,
		    descriptors_json = $5::jsonb, active = $6, updated_at = NOW(),
		    category = $7, cost_credits = $8, stats_json = $9::jsonb
		WHERE id = $1
	`, id, name, imageAssetID, shortDescription, descriptorsJSON, active,
		category, costCredits, statsJSON); err != nil {
		return EquipmentItem{}, err
	}
	return LoadEquipmentItemByID(ctx, pool, id)
}

// LocationIDForVenueSlug resolves a venue slug to its location_id -- the
// equipment/packet editor routes take an explicit venue (spec S11's Scene
// Setup control is reached from a specific venue's Stage Management), never
// assume a single install-wide location.
func LocationIDForVenueSlug(ctx context.Context, pool *pgxpool.Pool, venueSlug string) (string, error) {
	venueSlug = strings.ToLower(strings.TrimSpace(venueSlug))
	if venueSlug == "" {
		return "", errors.New("venue_slug_required")
	}
	var locationID string
	err := pool.QueryRow(ctx, `
		SELECT l.id::text
		FROM venues v
		JOIN lots lo ON lo.id = v.lot_id
		JOIN locations l ON l.id = lo.location_id
		WHERE v.slug = $1
		LIMIT 1
	`, venueSlug).Scan(&locationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("unknown_target")
	}
	if err != nil {
		return "", err
	}
	return locationID, nil
}
