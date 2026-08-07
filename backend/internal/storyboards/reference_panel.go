package storyboards

// Reference Panel CRUD (Kernel 82, spec 3). Fields follow the same
// UUID-identity + deterministic-slug + explicit-sort_order contract
// columns/bands/rows already have (Kernel 81A) -- slug is allocated once
// at AddReferenceField and never touched by rename/reorder/content edits.
//
// Authority mirrors card-content vs. board-structure exactly: content
// edits (field text, list/paired-list items) require CanEditCards
// (Crew+); structural edits (add/remove/rename/reorder/type-change a
// field) require CanEditStructure (Director+/owner/Operator) -- spec
// 3.4's role table maps directly onto authority.go's existing two tiers,
// no new authority function needed.

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrReferenceFieldNotFound              = errors.New("reference_field_not_found")
	ErrReferenceItemNotFound               = errors.New("reference_item_not_found")
	ErrReferenceFieldTypeInvalid           = errors.New("reference_field_type_invalid")
	ErrReferenceFieldNotTextType           = errors.New("reference_field_not_text_type")
	ErrReferenceFieldNotListType           = errors.New("reference_field_not_list_type")
	ErrReferenceFieldTypeChangeUnsafe      = errors.New("reference_field_type_change_unsafe")
	ErrReferenceFieldDeleteRequiresConfirm = errors.New("reference_field_delete_requires_confirm")
	ErrReferenceItemSideInvalid            = errors.New("reference_item_side_invalid")
)

func validFieldType(t string) bool {
	switch t {
	case FieldTypeShortText, FieldTypeLongText, FieldTypeList, FieldTypePairedList:
		return true
	}
	return false
}

func loadReferenceField(ctx context.Context, pool *pgxpool.Pool, boardID, fieldID string) (*ReferenceField, error) {
	var f ReferenceField
	err := pool.QueryRow(ctx, `
		SELECT id::text, storyboard_id::text, slug, label, field_type, sort_order,
		       text_content, sublabel_a, sublabel_b, created_at, updated_at
		FROM storyboard_reference_fields WHERE id = $1 AND storyboard_id = $2
	`, fieldID, boardID).Scan(&f.ID, &f.StoryboardID, &f.Slug, &f.Label, &f.FieldType, &f.SortOrder,
		&f.TextContent, &f.SublabelA, &f.SublabelB, &f.CreatedAt, &f.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrReferenceFieldNotFound
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// ListReferenceFields returns a board's Reference Panel fields in display
// order. Item content for list/paired_list fields is loaded separately
// via ListReferenceItemsForFields -- callers that need the full panel
// (snapshot.go) compose the two.
func ListReferenceFields(ctx context.Context, pool *pgxpool.Pool, boardID string) ([]ReferenceField, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, storyboard_id::text, slug, label, field_type, sort_order,
		       text_content, sublabel_a, sublabel_b, created_at, updated_at
		FROM storyboard_reference_fields WHERE storyboard_id = $1 ORDER BY sort_order
	`, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReferenceField{}
	for rows.Next() {
		var f ReferenceField
		if err := rows.Scan(&f.ID, &f.StoryboardID, &f.Slug, &f.Label, &f.FieldType, &f.SortOrder,
			&f.TextContent, &f.SublabelA, &f.SublabelB, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// ListReferenceItemsForFields returns every item belonging to any of
// fieldIDs, grouped by field ID, each group ordered by (side, sort_order)
// so a paired_list field's two sides come back already separable by
// Side.
func ListReferenceItemsForFields(ctx context.Context, pool *pgxpool.Pool, fieldIDs []string) (map[string][]ReferenceItem, error) {
	out := map[string][]ReferenceItem{}
	if len(fieldIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT id::text, field_id::text, side, sort_order, content, created_at, updated_at
		FROM storyboard_reference_items
		WHERE field_id = ANY($1)
		ORDER BY field_id, side, sort_order
	`, fieldIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var it ReferenceItem
		if err := rows.Scan(&it.ID, &it.FieldID, &it.Side, &it.SortOrder, &it.Content, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		out[it.FieldID] = append(out[it.FieldID], it)
	}
	return out, rows.Err()
}

// referenceFieldHasContent reports whether a field has anything a delete
// or type-change should treat as "nonempty" (spec 3.6's "deleting a
// nonempty field requires deliberate confirmation").
func referenceFieldHasContent(ctx context.Context, pool *pgxpool.Pool, field *ReferenceField) (bool, error) {
	switch field.FieldType {
	case FieldTypeShortText, FieldTypeLongText:
		return field.TextContent != "", nil
	case FieldTypeList, FieldTypePairedList:
		var count int
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM storyboard_reference_items WHERE field_id = $1`, field.ID).Scan(&count); err != nil {
			return false, err
		}
		return count > 0, nil
	}
	return false, nil
}

// AddReferenceField requires CanEditStructure. sublabelA/sublabelB are
// only meaningful for FieldTypePairedList and ignored otherwise.
func AddReferenceField(ctx context.Context, pool *pgxpool.Pool, userID, boardID, label, fieldType, sublabelA, sublabelB string) (*ReferenceField, error) {
	if !validFieldType(fieldType) {
		return nil, ErrReferenceFieldTypeInvalid
	}
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return nil, err
	}
	if ok, err := CanEditStructure(ctx, pool, userID, board); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrNotAuthorized
	}
	if fieldType != FieldTypePairedList {
		sublabelA, sublabelB = "", ""
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM storyboard_reference_fields WHERE storyboard_id = $1`, boardID).Scan(&count); err != nil {
		return nil, err
	}

	id, err := allocateUniqueSlug(ctx, pool, referenceFieldSlugExistsSQL, boardID, label, func(ctx context.Context, slug string) (string, error) {
		var newID string
		err := pool.QueryRow(ctx, `
			INSERT INTO storyboard_reference_fields (storyboard_id, slug, label, field_type, sort_order, sublabel_a, sublabel_b)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id::text
		`, boardID, slug, label, fieldType, count, sublabelA, sublabelB).Scan(&newID)
		return newID, err
	})
	if err != nil {
		return nil, err
	}
	return loadReferenceField(ctx, pool, boardID, id)
}

// RenameReferenceField requires CanEditStructure. Updates label and (for
// paired_list fields) both sublabels; never touches slug, matching the
// slug-assigned-once-at-creation contract every other structural rename
// in this package already follows.
func RenameReferenceField(ctx context.Context, pool *pgxpool.Pool, userID, boardID, fieldID, label, sublabelA, sublabelB string) (*ReferenceField, error) {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return nil, err
	}
	if ok, err := CanEditStructure(ctx, pool, userID, board); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrNotAuthorized
	}
	field, err := loadReferenceField(ctx, pool, boardID, fieldID)
	if err != nil {
		return nil, err
	}
	if field.FieldType != FieldTypePairedList {
		sublabelA, sublabelB = field.SublabelA, field.SublabelB
	}
	tag, err := pool.Exec(ctx, `
		UPDATE storyboard_reference_fields
		SET label = $3, sublabel_a = $4, sublabel_b = $5, updated_at = NOW()
		WHERE id = $1 AND storyboard_id = $2
	`, fieldID, boardID, label, sublabelA, sublabelB)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrReferenceFieldNotFound
	}
	return loadReferenceField(ctx, pool, boardID, fieldID)
}

// ChangeReferenceFieldType requires CanEditStructure, and only succeeds
// when the field currently carries no content -- spec 3.4 allows
// Director+ to "change field type where safe"; converting a field that
// already has text or list items would require deciding how to map that
// content into the new type's shape, which this kernel does not attempt.
// Clear the field first (spec's own delete-with-confirmation path for
// items, or overwrite text content to empty) to change its type.
func ChangeReferenceFieldType(ctx context.Context, pool *pgxpool.Pool, userID, boardID, fieldID, newType string) (*ReferenceField, error) {
	if !validFieldType(newType) {
		return nil, ErrReferenceFieldTypeInvalid
	}
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return nil, err
	}
	if ok, err := CanEditStructure(ctx, pool, userID, board); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrNotAuthorized
	}
	field, err := loadReferenceField(ctx, pool, boardID, fieldID)
	if err != nil {
		return nil, err
	}
	hasContent, err := referenceFieldHasContent(ctx, pool, field)
	if err != nil {
		return nil, err
	}
	if hasContent {
		return nil, ErrReferenceFieldTypeChangeUnsafe
	}
	sublabelA, sublabelB := field.SublabelA, field.SublabelB
	if newType != FieldTypePairedList {
		sublabelA, sublabelB = "", ""
	}
	if _, err := pool.Exec(ctx, `
		UPDATE storyboard_reference_fields
		SET field_type = $3, sublabel_a = $4, sublabel_b = $5, updated_at = NOW()
		WHERE id = $1 AND storyboard_id = $2
	`, fieldID, boardID, newType, sublabelA, sublabelB); err != nil {
		return nil, err
	}
	return loadReferenceField(ctx, pool, boardID, fieldID)
}

// ReorderReferenceFields requires CanEditStructure and orderedFieldIDs to
// be exactly the board's existing field set -- same shift-then-write
// pattern as ReorderColumns, so the unique (storyboard_id, sort_order)-
// shaped ordering never collides mid-transaction. (No DB uniqueness on
// sort_order here since duplicate labels/order collisions were never a
// hazard for this table the way MaxColumns-adjacent tables have one, but
// the shift keeps behavior consistent with the rest of the package.)
func ReorderReferenceFields(ctx context.Context, pool *pgxpool.Pool, userID, boardID string, orderedFieldIDs []string) error {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return err
	}
	if ok, err := CanEditStructure(ctx, pool, userID, board); err != nil {
		return err
	} else if !ok {
		return ErrNotAuthorized
	}
	existing, err := ListReferenceFields(ctx, pool, boardID)
	if err != nil {
		return err
	}
	existingSet := map[string]bool{}
	for _, f := range existing {
		existingSet[f.ID] = true
	}
	if len(orderedFieldIDs) != len(existing) {
		return ErrInvalidResolution
	}
	seen := map[string]bool{}
	for _, id := range orderedFieldIDs {
		if !existingSet[id] || seen[id] {
			return ErrInvalidResolution
		}
		seen[id] = true
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	const shift = 100000
	if _, err := tx.Exec(ctx, `UPDATE storyboard_reference_fields SET sort_order = sort_order + $2 WHERE storyboard_id = $1`, boardID, shift); err != nil {
		return err
	}
	for i, id := range orderedFieldIDs {
		if _, err := tx.Exec(ctx, `UPDATE storyboard_reference_fields SET sort_order = $3, updated_at = NOW() WHERE id = $1 AND storyboard_id = $2`, id, boardID, i); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// RemoveReferenceField requires CanEditStructure. A field with content
// requires confirmed == true (spec 3.6: "deleting a nonempty field
// requires deliberate confirmation") -- an empty field deletes outright,
// matching the rest of this package's "only occupied/nonempty targets
// need an explicit resolution" convention.
func RemoveReferenceField(ctx context.Context, pool *pgxpool.Pool, userID, boardID, fieldID string, confirmed bool) error {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return err
	}
	if ok, err := CanEditStructure(ctx, pool, userID, board); err != nil {
		return err
	} else if !ok {
		return ErrNotAuthorized
	}
	field, err := loadReferenceField(ctx, pool, boardID, fieldID)
	if err != nil {
		return err
	}
	hasContent, err := referenceFieldHasContent(ctx, pool, field)
	if err != nil {
		return err
	}
	if hasContent && !confirmed {
		return ErrReferenceFieldDeleteRequiresConfirm
	}
	_, err = pool.Exec(ctx, `DELETE FROM storyboard_reference_fields WHERE id = $1 AND storyboard_id = $2`, fieldID, boardID)
	return err
}

// SetReferenceFieldTextContent requires CanEditCards (Crew+, spec 3.4).
// Only valid for short_text/long_text fields.
func SetReferenceFieldTextContent(ctx context.Context, pool *pgxpool.Pool, userID, boardID, fieldID, text string) (*ReferenceField, error) {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return nil, err
	}
	if ok, err := CanEditCards(ctx, pool, userID, board); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrNotAuthorized
	}
	field, err := loadReferenceField(ctx, pool, boardID, fieldID)
	if err != nil {
		return nil, err
	}
	if field.FieldType != FieldTypeShortText && field.FieldType != FieldTypeLongText {
		return nil, ErrReferenceFieldNotTextType
	}
	if _, err := pool.Exec(ctx, `
		UPDATE storyboard_reference_fields SET text_content = $3, updated_at = NOW()
		WHERE id = $1 AND storyboard_id = $2
	`, fieldID, boardID, text); err != nil {
		return nil, err
	}
	return loadReferenceField(ctx, pool, boardID, fieldID)
}

// resolveItemSide validates side against the field's type: a plain list
// field only ever takes ItemSideSingle; a paired_list field only ever
// takes ItemSideA/ItemSideB.
func resolveItemSide(fieldType, side string) (string, error) {
	switch fieldType {
	case FieldTypeList:
		if side != "" && side != ItemSideSingle {
			return "", ErrReferenceItemSideInvalid
		}
		return ItemSideSingle, nil
	case FieldTypePairedList:
		if side != ItemSideA && side != ItemSideB {
			return "", ErrReferenceItemSideInvalid
		}
		return side, nil
	default:
		return "", ErrReferenceFieldNotListType
	}
}

// AddReferenceItem requires CanEditCards (Crew+). side is required for a
// paired_list field (ItemSideA/ItemSideB) and optional/ignored-to-
// ItemSideSingle for a plain list field.
func AddReferenceItem(ctx context.Context, pool *pgxpool.Pool, userID, boardID, fieldID, side, content string) (*ReferenceItem, error) {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return nil, err
	}
	if ok, err := CanEditCards(ctx, pool, userID, board); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrNotAuthorized
	}
	field, err := loadReferenceField(ctx, pool, boardID, fieldID)
	if err != nil {
		return nil, err
	}
	resolvedSide, err := resolveItemSide(field.FieldType, side)
	if err != nil {
		return nil, err
	}

	var nextOrder int
	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(sort_order) + 1, 0) FROM storyboard_reference_items WHERE field_id = $1 AND side = $2
	`, fieldID, resolvedSide).Scan(&nextOrder); err != nil {
		return nil, err
	}

	var it ReferenceItem
	err = pool.QueryRow(ctx, `
		INSERT INTO storyboard_reference_items (field_id, side, sort_order, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, field_id::text, side, sort_order, content, created_at, updated_at
	`, fieldID, resolvedSide, nextOrder, content).Scan(&it.ID, &it.FieldID, &it.Side, &it.SortOrder, &it.Content, &it.CreatedAt, &it.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &it, nil
}

// UpdateReferenceItemContent requires CanEditCards (Crew+).
func UpdateReferenceItemContent(ctx context.Context, pool *pgxpool.Pool, userID, boardID, fieldID, itemID, content string) (*ReferenceItem, error) {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return nil, err
	}
	if ok, err := CanEditCards(ctx, pool, userID, board); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrNotAuthorized
	}
	if _, err := loadReferenceField(ctx, pool, boardID, fieldID); err != nil {
		return nil, err
	}
	tag, err := pool.Exec(ctx, `
		UPDATE storyboard_reference_items SET content = $3, updated_at = NOW()
		WHERE id = $1 AND field_id = $2
	`, itemID, fieldID, content)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrReferenceItemNotFound
	}
	var it ReferenceItem
	err = pool.QueryRow(ctx, `
		SELECT id::text, field_id::text, side, sort_order, content, created_at, updated_at
		FROM storyboard_reference_items WHERE id = $1
	`, itemID).Scan(&it.ID, &it.FieldID, &it.Side, &it.SortOrder, &it.Content, &it.CreatedAt, &it.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &it, nil
}

// RemoveReferenceItem requires CanEditCards (Crew+). A single item never
// needs the field-level "nonempty confirmation" flow -- that's about
// deleting the whole field, not one entry in it.
func RemoveReferenceItem(ctx context.Context, pool *pgxpool.Pool, userID, boardID, fieldID, itemID string) error {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return err
	}
	if ok, err := CanEditCards(ctx, pool, userID, board); err != nil {
		return err
	} else if !ok {
		return ErrNotAuthorized
	}
	if _, err := loadReferenceField(ctx, pool, boardID, fieldID); err != nil {
		return err
	}
	tag, err := pool.Exec(ctx, `DELETE FROM storyboard_reference_items WHERE id = $1 AND field_id = $2`, itemID, fieldID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrReferenceItemNotFound
	}
	return nil
}

// ReorderReferenceItems requires CanEditCards (Crew+) and orderedItemIDs
// to be exactly the (fieldID, side)'s existing item set.
func ReorderReferenceItems(ctx context.Context, pool *pgxpool.Pool, userID, boardID, fieldID, side string, orderedItemIDs []string) error {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return err
	}
	if ok, err := CanEditCards(ctx, pool, userID, board); err != nil {
		return err
	} else if !ok {
		return ErrNotAuthorized
	}
	field, err := loadReferenceField(ctx, pool, boardID, fieldID)
	if err != nil {
		return err
	}
	resolvedSide, err := resolveItemSide(field.FieldType, side)
	if err != nil {
		return err
	}

	rows, err := pool.Query(ctx, `SELECT id::text FROM storyboard_reference_items WHERE field_id = $1 AND side = $2`, fieldID, resolvedSide)
	if err != nil {
		return err
	}
	existingSet := map[string]bool{}
	count := 0
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		existingSet[id] = true
		count++
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(orderedItemIDs) != count {
		return ErrInvalidResolution
	}
	seen := map[string]bool{}
	for _, id := range orderedItemIDs {
		if !existingSet[id] || seen[id] {
			return ErrInvalidResolution
		}
		seen[id] = true
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for i, id := range orderedItemIDs {
		if _, err := tx.Exec(ctx, `UPDATE storyboard_reference_items SET sort_order = $2, updated_at = NOW() WHERE id = $1`, id, i); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
