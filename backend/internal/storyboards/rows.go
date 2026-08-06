package storyboards

// Row structural CRUD (spec 1.5, 6.4). All row mutations require
// CanEditStructure (Director+/owner/Operator) -- unlike bands, the spec
// grants Crew no row-label editing allowance (5.3 lists only "band
// labels"), so rows have no Crew-if-unlocked carve-out.

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRowNotFound = errors.New("row_not_found")
	ErrRowOccupied = errors.New("row_occupied")
)

func loadRow(ctx context.Context, pool *pgxpool.Pool, boardID, rowID string) (*StoryboardRow, error) {
	var r StoryboardRow
	err := pool.QueryRow(ctx, `
		SELECT id::text, storyboard_id::text, band_id::text, label, description, sort_order_in_band, created_at, updated_at
		FROM storyboard_rows WHERE id = $1 AND storyboard_id = $2
	`, rowID, boardID).Scan(&r.ID, &r.StoryboardID, &r.BandID, &r.Label, &r.Description, &r.SortOrderInBand, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRowNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// ListRows returns a board's rows in (band sort_order, row
// sort_order_in_band) display order.
func ListRows(ctx context.Context, pool *pgxpool.Pool, boardID string) ([]StoryboardRow, error) {
	rows, err := pool.Query(ctx, `
		SELECT r.id::text, r.storyboard_id::text, r.band_id::text, r.label, r.description, r.sort_order_in_band, r.created_at, r.updated_at
		FROM storyboard_rows r
		JOIN storyboard_bands b ON b.id = r.band_id
		WHERE r.storyboard_id = $1
		ORDER BY b.sort_order, r.sort_order_in_band
	`, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StoryboardRow{}
	for rows.Next() {
		var r StoryboardRow
		if err := rows.Scan(&r.ID, &r.StoryboardID, &r.BandID, &r.Label, &r.Description, &r.SortOrderInBand, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// AddRow appends a new row to the given band. Requires CanEditStructure.
func AddRow(ctx context.Context, pool *pgxpool.Pool, userID, boardID, bandID, label string) (*StoryboardRow, error) {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return nil, err
	}
	if ok, err := CanEditStructure(ctx, pool, userID, board); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrNotAuthorized
	}
	if _, err := loadBand(ctx, pool, boardID, bandID); err != nil {
		return nil, err
	}
	var nextOrder int
	if err := pool.QueryRow(ctx, `SELECT COALESCE(MAX(sort_order_in_band) + 1, 0) FROM storyboard_rows WHERE band_id = $1`, bandID).Scan(&nextOrder); err != nil {
		return nil, err
	}
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO storyboard_rows (storyboard_id, band_id, label, sort_order_in_band)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text
	`, boardID, bandID, label, nextOrder).Scan(&id); err != nil {
		return nil, err
	}
	return loadRow(ctx, pool, boardID, id)
}

// RenameRow requires CanEditStructure.
func RenameRow(ctx context.Context, pool *pgxpool.Pool, userID, boardID, rowID, label, description string) (*StoryboardRow, error) {
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
		UPDATE storyboard_rows SET label = $3, description = $4, updated_at = NOW()
		WHERE id = $1 AND storyboard_id = $2
	`, rowID, boardID, label, description)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrRowNotFound
	}
	return loadRow(ctx, pool, boardID, rowID)
}

// ReorderRowsWithinBand requires CanEditStructure and orderedRowIDs to be
// exactly the band's existing row set.
func ReorderRowsWithinBand(ctx context.Context, pool *pgxpool.Pool, userID, boardID, bandID string, orderedRowIDs []string) error {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return err
	}
	if ok, err := CanEditStructure(ctx, pool, userID, board); err != nil {
		return err
	} else if !ok {
		return ErrNotAuthorized
	}
	if _, err := loadBand(ctx, pool, boardID, bandID); err != nil {
		return err
	}

	rows, err := pool.Query(ctx, `SELECT id::text FROM storyboard_rows WHERE band_id = $1`, bandID)
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
	if len(orderedRowIDs) != count {
		return ErrInvalidResolution
	}
	seen := map[string]bool{}
	for _, id := range orderedRowIDs {
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
	if _, err := tx.Exec(ctx, `UPDATE storyboard_rows SET sort_order_in_band = sort_order_in_band + $2 WHERE band_id = $1`, bandID, shift); err != nil {
		return err
	}
	for i, id := range orderedRowIDs {
		if _, err := tx.Exec(ctx, `UPDATE storyboard_rows SET sort_order_in_band = $3, updated_at = NOW() WHERE id = $1 AND band_id = $2`, id, bandID, i); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// MoveRowToBand moves a row to a different band, appended at the end of
// the target band's row order. Preserves row identity, cards, card column
// positions, and within-cell ordering (spec 6.4) -- nothing about the
// row's own cards is touched, only band_id/sort_order_in_band on the row
// itself.
func MoveRowToBand(ctx context.Context, pool *pgxpool.Pool, userID, boardID, rowID, targetBandID string) (*StoryboardRow, error) {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return nil, err
	}
	if ok, err := CanEditStructure(ctx, pool, userID, board); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrNotAuthorized
	}
	if _, err := loadRow(ctx, pool, boardID, rowID); err != nil {
		return nil, err
	}
	if _, err := loadBand(ctx, pool, boardID, targetBandID); err != nil {
		return nil, err
	}

	var nextOrder int
	if err := pool.QueryRow(ctx, `SELECT COALESCE(MAX(sort_order_in_band) + 1, 0) FROM storyboard_rows WHERE band_id = $1`, targetBandID).Scan(&nextOrder); err != nil {
		return nil, err
	}
	if _, err := pool.Exec(ctx, `
		UPDATE storyboard_rows SET band_id = $2, sort_order_in_band = $3, updated_at = NOW()
		WHERE id = $1
	`, rowID, targetBandID, nextOrder); err != nil {
		return nil, err
	}
	return loadRow(ctx, pool, boardID, rowID)
}

// RemoveRow implements the spec 6.4 required explicit resolution: an
// unoccupied row (no cards) deletes outright. An occupied row requires
// resolution "delete_cards" or "move_cards" (targetRowID required, must be
// a different row -- cards keep their column, appended to the end of each
// target cell). Passing resolution "" against an occupied row returns
// ErrRowOccupied.
func RemoveRow(ctx context.Context, pool *pgxpool.Pool, userID, boardID, rowID, resolution, targetRowID string) error {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return err
	}
	if ok, err := CanEditStructure(ctx, pool, userID, board); err != nil {
		return err
	} else if !ok {
		return ErrNotAuthorized
	}
	if _, err := loadRow(ctx, pool, boardID, rowID); err != nil {
		return err
	}

	var cardCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM storyboard_cards WHERE row_id = $1`, rowID).Scan(&cardCount); err != nil {
		return err
	}

	if cardCount == 0 {
		_, err := pool.Exec(ctx, `DELETE FROM storyboard_rows WHERE id = $1 AND storyboard_id = $2`, rowID, boardID)
		return err
	}

	switch resolution {
	case "delete_cards":
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, err := tx.Exec(ctx, `DELETE FROM storyboard_cards WHERE row_id = $1`, rowID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM storyboard_rows WHERE id = $1 AND storyboard_id = $2`, rowID, boardID); err != nil {
			return err
		}
		return tx.Commit(ctx)

	case "move_cards":
		if targetRowID == "" || targetRowID == rowID {
			return ErrInvalidResolution
		}
		if _, err := loadRow(ctx, pool, boardID, targetRowID); err != nil {
			return err
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback(ctx) }()

		rows, err := tx.Query(ctx, `SELECT id::text, column_id::text FROM storyboard_cards WHERE row_id = $1`, rowID)
		if err != nil {
			return err
		}
		type moving struct{ id, columnID string }
		var toMove []moving
		for rows.Next() {
			var m moving
			if err := rows.Scan(&m.id, &m.columnID); err != nil {
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
			`, targetRowID, m.columnID).Scan(&nextOrder); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				UPDATE storyboard_cards SET row_id = $2, sort_order_in_cell = $3, updated_at = NOW()
				WHERE id = $1
			`, m.id, targetRowID, nextOrder); err != nil {
				return err
			}
		}

		if _, err := tx.Exec(ctx, `DELETE FROM storyboard_rows WHERE id = $1 AND storyboard_id = $2`, rowID, boardID); err != nil {
			return err
		}
		return tx.Commit(ctx)

	default:
		return ErrRowOccupied
	}
}
