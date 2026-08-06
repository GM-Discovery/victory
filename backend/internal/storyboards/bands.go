package storyboards

// Band structural CRUD (spec 1.6, 6.3). Structural mutations (add, reorder,
// remove, lock/unlock) require CanEditStructure (Director+/owner/Operator).
// Label/description edits are allowed to Crew too, but only while the band
// is unlocked (spec 1.6 "Crew may edit band labels"; 5.6 "A locked band
// prevents ... label edits within the band until unlocked").

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrBandNotFound = errors.New("band_not_found")
	ErrBandLocked   = errors.New("band_locked")
	ErrBandOccupied = errors.New("band_occupied")
)

func loadBand(ctx context.Context, pool *pgxpool.Pool, boardID, bandID string) (*StoryboardBand, error) {
	var b StoryboardBand
	err := pool.QueryRow(ctx, `
		SELECT id::text, storyboard_id::text, label, description, sort_order, is_collapsed, is_locked, created_at, updated_at
		FROM storyboard_bands WHERE id = $1 AND storyboard_id = $2
	`, bandID, boardID).Scan(&b.ID, &b.StoryboardID, &b.Label, &b.Description, &b.SortOrder, &b.IsCollapsed, &b.IsLocked, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrBandNotFound
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// ListBands returns a board's bands in display order.
func ListBands(ctx context.Context, pool *pgxpool.Pool, boardID string) ([]StoryboardBand, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, storyboard_id::text, label, description, sort_order, is_collapsed, is_locked, created_at, updated_at
		FROM storyboard_bands WHERE storyboard_id = $1 ORDER BY sort_order
	`, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StoryboardBand{}
	for rows.Next() {
		var b StoryboardBand
		if err := rows.Scan(&b.ID, &b.StoryboardID, &b.Label, &b.Description, &b.SortOrder, &b.IsCollapsed, &b.IsLocked, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// AddBand requires CanEditStructure.
func AddBand(ctx context.Context, pool *pgxpool.Pool, userID, boardID, label string) (*StoryboardBand, error) {
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
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM storyboard_bands WHERE storyboard_id = $1`, boardID).Scan(&count); err != nil {
		return nil, err
	}
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO storyboard_bands (storyboard_id, label, sort_order)
		VALUES ($1, $2, $3)
		RETURNING id::text
	`, boardID, label, count).Scan(&id); err != nil {
		return nil, err
	}
	return loadBand(ctx, pool, boardID, id)
}

// UpdateBandLabel: Crew may edit while unlocked; Director+/owner may edit
// regardless of lock (they can unlock it themselves anyway).
func UpdateBandLabel(ctx context.Context, pool *pgxpool.Pool, userID, boardID, bandID, label, description string) (*StoryboardBand, error) {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return nil, err
	}
	band, err := loadBand(ctx, pool, boardID, bandID)
	if err != nil {
		return nil, err
	}
	if canStructure, err := CanEditStructure(ctx, pool, userID, board); err != nil {
		return nil, err
	} else if !canStructure {
		canCards, err := CanEditCards(ctx, pool, userID, board)
		if err != nil {
			return nil, err
		}
		if !canCards {
			return nil, ErrNotAuthorized
		}
		if band.IsLocked {
			return nil, ErrBandLocked
		}
	}
	if _, err := pool.Exec(ctx, `
		UPDATE storyboard_bands SET label = $3, description = $4, updated_at = NOW()
		WHERE id = $1 AND storyboard_id = $2
	`, bandID, boardID, label, description); err != nil {
		return nil, err
	}
	return loadBand(ctx, pool, boardID, bandID)
}

// SetBandLock requires CanEditStructure (Director+/owner only may lock or
// unlock, spec 5.4).
func SetBandLock(ctx context.Context, pool *pgxpool.Pool, userID, boardID, bandID string, locked bool) (*StoryboardBand, error) {
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
		UPDATE storyboard_bands SET is_locked = $3, updated_at = NOW()
		WHERE id = $1 AND storyboard_id = $2
	`, bandID, boardID, locked)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrBandNotFound
	}
	return loadBand(ctx, pool, boardID, bandID)
}

// SetBandCollapsed requires CanEditStructure -- collapse is a structural
// display concern (spec 1.6 lists collapse alongside lock/reorder/remove
// as Director+ authority), distinct from the label-only Crew allowance.
func SetBandCollapsed(ctx context.Context, pool *pgxpool.Pool, userID, boardID, bandID string, collapsed bool) (*StoryboardBand, error) {
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
		UPDATE storyboard_bands SET is_collapsed = $3, updated_at = NOW()
		WHERE id = $1 AND storyboard_id = $2
	`, bandID, boardID, collapsed)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrBandNotFound
	}
	return loadBand(ctx, pool, boardID, bandID)
}

// ReorderBands requires CanEditStructure and orderedBandIDs to be exactly
// the board's existing band set.
func ReorderBands(ctx context.Context, pool *pgxpool.Pool, userID, boardID string, orderedBandIDs []string) error {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return err
	}
	if ok, err := CanEditStructure(ctx, pool, userID, board); err != nil {
		return err
	} else if !ok {
		return ErrNotAuthorized
	}

	existing, err := ListBands(ctx, pool, boardID)
	if err != nil {
		return err
	}
	existingSet := map[string]bool{}
	for _, b := range existing {
		existingSet[b.ID] = true
	}
	if len(orderedBandIDs) != len(existing) {
		return ErrInvalidResolution
	}
	seen := map[string]bool{}
	for _, id := range orderedBandIDs {
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
	if _, err := tx.Exec(ctx, `UPDATE storyboard_bands SET sort_order = sort_order + $2 WHERE storyboard_id = $1`, boardID, shift); err != nil {
		return err
	}
	for i, id := range orderedBandIDs {
		if _, err := tx.Exec(ctx, `UPDATE storyboard_bands SET sort_order = $3, updated_at = NOW() WHERE id = $1 AND storyboard_id = $2`, id, boardID, i); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// RemoveBand implements the spec 6.3 required resolution flow: an empty
// band (no rows) deletes outright. An occupied band requires resolution
// "delete_rows" (removes the band's rows and their cards too, via FK
// cascade) or "move_rows" (targetBandID required, must be a different band
// on the same board -- rows are appended to the end of the target band's
// row order, preserving each row's own cards and column positions
// untouched). Passing resolution "" against an occupied band returns
// ErrBandOccupied rather than silently destroying rows/cards.
func RemoveBand(ctx context.Context, pool *pgxpool.Pool, userID, boardID, bandID, resolution, targetBandID string) error {
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

	var rowCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM storyboard_rows WHERE band_id = $1`, bandID).Scan(&rowCount); err != nil {
		return err
	}

	if rowCount == 0 {
		_, err := pool.Exec(ctx, `DELETE FROM storyboard_bands WHERE id = $1 AND storyboard_id = $2`, bandID, boardID)
		return err
	}

	switch resolution {
	case "delete_rows":
		// FK ON DELETE CASCADE on storyboard_rows.band_id and
		// storyboard_cards.row_id removes rows and their cards together.
		_, err := pool.Exec(ctx, `DELETE FROM storyboard_bands WHERE id = $1 AND storyboard_id = $2`, bandID, boardID)
		return err

	case "move_rows":
		if targetBandID == "" || targetBandID == bandID {
			return ErrInvalidResolution
		}
		if _, err := loadBand(ctx, pool, boardID, targetBandID); err != nil {
			return err
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback(ctx) }()

		var nextOrder int
		if err := tx.QueryRow(ctx, `
			SELECT COALESCE(MAX(sort_order_in_band) + 1, 0) FROM storyboard_rows WHERE band_id = $1
		`, targetBandID).Scan(&nextOrder); err != nil {
			return err
		}

		rows, err := tx.Query(ctx, `SELECT id::text FROM storyboard_rows WHERE band_id = $1 ORDER BY sort_order_in_band`, bandID)
		if err != nil {
			return err
		}
		var rowIDs []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			rowIDs = append(rowIDs, id)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}

		for _, id := range rowIDs {
			if _, err := tx.Exec(ctx, `
				UPDATE storyboard_rows SET band_id = $2, sort_order_in_band = $3, updated_at = NOW()
				WHERE id = $1
			`, id, targetBandID, nextOrder); err != nil {
				return err
			}
			nextOrder++
		}

		if _, err := tx.Exec(ctx, `DELETE FROM storyboard_bands WHERE id = $1 AND storyboard_id = $2`, bandID, boardID); err != nil {
			return err
		}
		return tx.Commit(ctx)

	default:
		return ErrBandOccupied
	}
}
