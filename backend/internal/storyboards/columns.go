package storyboards

// Column structural CRUD (spec 1.4, 6.2). All mutations require
// CanEditStructure (Director+/owner/Operator).

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MaxColumns is the 200-column technical safety limit (spec 1.4) --
// enforced here in Go, not a DB constraint, since it's a soft
// implementation safeguard with a specific client-facing error, not a hard
// schema rule.
const MaxColumns = 200

var (
	ErrColumnLimitExceeded     = errors.New("column_limit_exceeded")
	ErrColumnNotFound          = errors.New("column_not_found")
	ErrColumnOccupied          = errors.New("column_occupied")
	ErrInvalidResolution       = errors.New("invalid_resolution")
	ErrBoundaryColumnProtected = errors.New("boundary_column_protected")
	ErrBoundaryColumnDisplaced = errors.New("boundary_column_displaced")
)

func loadColumn(ctx context.Context, pool *pgxpool.Pool, boardID, columnID string) (*StoryboardColumn, error) {
	var c StoryboardColumn
	err := pool.QueryRow(ctx, `
		SELECT id::text, storyboard_id::text, title, COALESCE(slug, ''), column_role, sort_order, created_at, updated_at
		FROM storyboard_columns WHERE id = $1 AND storyboard_id = $2
	`, columnID, boardID).Scan(&c.ID, &c.StoryboardID, &c.Title, &c.Slug, &c.ColumnRole, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrColumnNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ListColumns returns a board's columns in display order.
func ListColumns(ctx context.Context, pool *pgxpool.Pool, boardID string) ([]StoryboardColumn, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, storyboard_id::text, title, COALESCE(slug, ''), column_role, sort_order, created_at, updated_at
		FROM storyboard_columns WHERE storyboard_id = $1 ORDER BY sort_order
	`, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StoryboardColumn{}
	for rows.Next() {
		var c StoryboardColumn
		if err := rows.Scan(&c.ID, &c.StoryboardID, &c.Title, &c.Slug, &c.ColumnRole, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// AddColumn appends a new column, rejecting the 201st with a clear error.
// slug is allocated once here (Kernel 81A) via allocateUniqueSlug and
// never touched again -- RenameColumn below only ever updates title.
func AddColumn(ctx context.Context, pool *pgxpool.Pool, userID, boardID, title string) (*StoryboardColumn, error) {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return nil, err
	}
	if ok, err := CanEditStructure(ctx, pool, userID, board); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrNotAuthorized
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM storyboard_columns WHERE storyboard_id = $1`, boardID).Scan(&count); err != nil {
		return nil, err
	}
	if count >= MaxColumns {
		return nil, ErrColumnLimitExceeded
	}

	id, err := allocateUniqueSlug(ctx, pool, columnSlugExistsSQL, boardID, title, func(ctx context.Context, slug string) (string, error) {
		var newID string
		err := pool.QueryRow(ctx, `
			INSERT INTO storyboard_columns (storyboard_id, title, slug, sort_order)
			VALUES ($1, $2, $3, $4)
			RETURNING id::text
		`, boardID, title, slug, count).Scan(&newID)
		return newID, err
	})
	if err != nil {
		return nil, err
	}
	return loadColumn(ctx, pool, boardID, id)
}

// RenameColumn requires CanEditStructure.
func RenameColumn(ctx context.Context, pool *pgxpool.Pool, userID, boardID, columnID, title string) (*StoryboardColumn, error) {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return nil, err
	}
	if ok, err := CanEditStructure(ctx, pool, userID, board); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrNotAuthorized
	}
	tag, err := pool.Exec(ctx, `
		UPDATE storyboard_columns SET title = $3, updated_at = NOW()
		WHERE id = $1 AND storyboard_id = $2
	`, columnID, boardID, title)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrColumnNotFound
	}
	return loadColumn(ctx, pool, boardID, columnID)
}

// ReorderColumns requires CanEditStructure and orderedColumnIDs to be
// exactly the board's existing column set (no adds/drops/duplicates).
// Reordering is done via a shift-out-of-range pass then a final pass, so
// the (storyboard_id, sort_order) unique constraint never collides
// mid-transaction.
func ReorderColumns(ctx context.Context, pool *pgxpool.Pool, userID, boardID string, orderedColumnIDs []string) error {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return err
	}
	if ok, err := CanEditStructure(ctx, pool, userID, board); err != nil {
		return err
	} else if !ok {
		return ErrNotAuthorized
	}

	existing, err := ListColumns(ctx, pool, boardID)
	if err != nil {
		return err
	}
	existingSet := map[string]bool{}
	roleByID := map[string]string{}
	for _, c := range existing {
		existingSet[c.ID] = true
		roleByID[c.ID] = c.ColumnRole
	}
	if len(orderedColumnIDs) != len(existing) {
		return ErrInvalidResolution
	}
	seen := map[string]bool{}
	for _, id := range orderedColumnIDs {
		if !existingSet[id] || seen[id] {
			return ErrInvalidResolution
		}
		seen[id] = true
	}

	// Boundary protection (Kernel 82 spec 2.3): a 'beginning' column must
	// stay at index 0 and an 'ending' column must stay at the last index,
	// regardless of what order the rest of the board is in. This is a
	// server-side check, not just a UI omission -- hiding a control is
	// never itself a security boundary in this codebase. A no-op for
	// every Blank-mode board, since no column there ever carries either
	// role.
	for i, id := range orderedColumnIDs {
		switch roleByID[id] {
		case ColumnRoleBeginning:
			if i != 0 {
				return ErrBoundaryColumnDisplaced
			}
		case ColumnRoleEnding:
			if i != len(orderedColumnIDs)-1 {
				return ErrBoundaryColumnDisplaced
			}
		}
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const shift = 100000
	if _, err := tx.Exec(ctx, `UPDATE storyboard_columns SET sort_order = sort_order + $2 WHERE storyboard_id = $1`, boardID, shift); err != nil {
		return err
	}
	for i, id := range orderedColumnIDs {
		if _, err := tx.Exec(ctx, `UPDATE storyboard_columns SET sort_order = $3, updated_at = NOW() WHERE id = $1 AND storyboard_id = $2`, id, boardID, i); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// RemoveColumn implements the spec 6.2 required resolution flow: an
// unoccupied column deletes outright; an occupied column requires the
// caller to explicitly choose resolution "delete_cards" (removes the
// column's cards too) or "move_cards" (targetColumnID required, must be a
// different column on the same board -- cards are appended to the end of
// their row's cell in the target column). Passing resolution "" against an
// occupied column returns ErrColumnOccupied rather than silently deleting
// cards (spec: "Do not silently delete cards").
func RemoveColumn(ctx context.Context, pool *pgxpool.Pool, userID, boardID, columnID, resolution, targetColumnID string) error {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return err
	}
	if ok, err := CanEditStructure(ctx, pool, userID, board); err != nil {
		return err
	} else if !ok {
		return ErrNotAuthorized
	}
	col, err := loadColumn(ctx, pool, boardID, columnID)
	if err != nil {
		return err
	}
	// Boundary protection (Kernel 82 spec 2.2/2.3): Beginning/Ending are
	// protected structural roles. Deleting one would be the ultimate form
	// of "displaced outside the timeline" the spec's own FAIL criteria
	// names -- rejected outright, with no resolution flow, unlike an
	// ordinary occupied-column removal.
	if col.ColumnRole == ColumnRoleBeginning || col.ColumnRole == ColumnRoleEnding {
		return ErrBoundaryColumnProtected
	}

	var cardCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM storyboard_cards WHERE column_id = $1`, columnID).Scan(&cardCount); err != nil {
		return err
	}

	if cardCount == 0 {
		_, err := pool.Exec(ctx, `DELETE FROM storyboard_columns WHERE id = $1 AND storyboard_id = $2`, columnID, boardID)
		return err
	}

	switch resolution {
	case "delete_cards":
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, err := tx.Exec(ctx, `DELETE FROM storyboard_cards WHERE column_id = $1`, columnID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM storyboard_columns WHERE id = $1 AND storyboard_id = $2`, columnID, boardID); err != nil {
			return err
		}
		return tx.Commit(ctx)

	case "move_cards":
		if targetColumnID == "" || targetColumnID == columnID {
			return ErrInvalidResolution
		}
		if _, err := loadColumn(ctx, pool, boardID, targetColumnID); err != nil {
			return err
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback(ctx) }()

		rows, err := tx.Query(ctx, `SELECT id::text, row_id::text FROM storyboard_cards WHERE column_id = $1`, columnID)
		if err != nil {
			return err
		}
		type moving struct{ id, rowID string }
		var toMove []moving
		for rows.Next() {
			var m moving
			if err := rows.Scan(&m.id, &m.rowID); err != nil {
				rows.Close()
				return err
			}
			toMove = append(toMove, m)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}

		for _, m := range toMove {
			var nextOrder int
			if err := tx.QueryRow(ctx, `
				SELECT COALESCE(MAX(sort_order_in_cell) + 1, 0) FROM storyboard_cards
				WHERE row_id = $1 AND column_id = $2
			`, m.rowID, targetColumnID).Scan(&nextOrder); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				UPDATE storyboard_cards SET column_id = $2, sort_order_in_cell = $3, updated_at = NOW()
				WHERE id = $1
			`, m.id, targetColumnID, nextOrder); err != nil {
				return err
			}
		}

		if _, err := tx.Exec(ctx, `DELETE FROM storyboard_columns WHERE id = $1 AND storyboard_id = $2`, columnID, boardID); err != nil {
			return err
		}
		return tx.Commit(ctx)

	default:
		return ErrColumnOccupied
	}
}
