package storyboards

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotAuthorized = errors.New("not_authorized")
	ErrInvalidTitle  = errors.New("title_required")
)

func loadBoardRow(ctx context.Context, tx pgxQuerier, boardID string) (*Storyboard, error) {
	var b Storyboard
	err := tx.QueryRow(ctx, `
		SELECT s.id::text, s.owner_user_id::text, COALESCE(u.handle, ''), s.title, s.description,
		       s.archived_at, s.created_at, s.updated_at
		FROM storyboards s
		LEFT JOIN users u ON u.id = s.owner_user_id
		WHERE s.id = $1
	`, boardID).Scan(&b.ID, &b.OwnerUserID, &b.OwnerHandle, &b.Title, &b.Description,
		&b.ArchivedAt, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrBoardNotFound
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// pgxQuerier is satisfied by both *pgxpool.Pool and pgx.Tx, so loadBoardRow
// can be called either standalone or inside a transaction.
type pgxQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// LoadBoard fetches one board by ID, or ErrBoardNotFound.
func LoadBoard(ctx context.Context, pool *pgxpool.Pool, boardID string) (*Storyboard, error) {
	return loadBoardRow(ctx, pool, boardID)
}

// CreateBoard makes ownerUserID the board's owner and creates the minimum
// valid structure in the same transaction: at least one column, one band,
// and one row assigned to that band (spec 6.1). Empty columnTitles/
// bandLabel/rowLabels fall back to single sensible defaults so the board is
// always valid regardless of what the caller supplies.
func CreateBoard(ctx context.Context, pool *pgxpool.Pool, ownerUserID, title, description string, columnTitles []string, bandLabel string, rowLabels []string) (*Storyboard, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrInvalidTitle
	}
	if len(columnTitles) == 0 {
		columnTitles = []string{"Column 1"}
	}
	bandLabel = strings.TrimSpace(bandLabel)
	if bandLabel == "" {
		bandLabel = "Band 1"
	}
	if len(rowLabels) == 0 {
		rowLabels = []string{"Row 1"}
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var boardID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO storyboards (owner_user_id, title, description)
		VALUES ($1, $2, $3)
		RETURNING id::text
	`, ownerUserID, title, description).Scan(&boardID); err != nil {
		return nil, err
	}

	for i, colTitle := range columnTitles {
		if _, err := tx.Exec(ctx, `
			INSERT INTO storyboard_columns (storyboard_id, title, sort_order)
			VALUES ($1, $2, $3)
		`, boardID, colTitle, i); err != nil {
			return nil, err
		}
	}

	var bandID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO storyboard_bands (storyboard_id, label, sort_order)
		VALUES ($1, $2, 0)
		RETURNING id::text
	`, boardID, bandLabel).Scan(&bandID); err != nil {
		return nil, err
	}

	for i, rowLabel := range rowLabels {
		if _, err := tx.Exec(ctx, `
			INSERT INTO storyboard_rows (storyboard_id, band_id, label, sort_order_in_band)
			VALUES ($1, $2, $3, $4)
		`, boardID, bandID, rowLabel, i); err != nil {
			return nil, err
		}
	}

	board, err := loadBoardRow(ctx, tx, boardID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return board, nil
}

// ListOwnedBoards returns every board this user owns, newest-activity
// first.
func ListOwnedBoards(ctx context.Context, pool *pgxpool.Pool, userID string) ([]Storyboard, error) {
	rows, err := pool.Query(ctx, `
		SELECT s.id::text, s.owner_user_id::text, COALESCE(u.handle, ''), s.title, s.description,
		       s.archived_at, s.created_at, s.updated_at
		FROM storyboards s
		LEFT JOIN users u ON u.id = s.owner_user_id
		WHERE s.owner_user_id = $1
		ORDER BY s.updated_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Storyboard{}
	for rows.Next() {
		var b Storyboard
		if err := rows.Scan(&b.ID, &b.OwnerUserID, &b.OwnerHandle, &b.Title, &b.Description,
			&b.ArchivedAt, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// ListSharedBoards returns every board explicitly granted to this user
// (never boards they merely own), newest-activity first.
func ListSharedBoards(ctx context.Context, pool *pgxpool.Pool, userID string) ([]Storyboard, error) {
	rows, err := pool.Query(ctx, `
		SELECT s.id::text, s.owner_user_id::text, COALESCE(u.handle, ''), s.title, s.description,
		       s.archived_at, s.created_at, s.updated_at
		FROM storyboards s
		LEFT JOIN users u ON u.id = s.owner_user_id
		JOIN storyboard_grants g ON g.storyboard_id = s.id
		WHERE g.user_id = $1
		ORDER BY s.updated_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Storyboard{}
	for rows.Next() {
		var b Storyboard
		if err := rows.Scan(&b.ID, &b.OwnerUserID, &b.OwnerHandle, &b.Title, &b.Description,
			&b.ArchivedAt, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// UpdateBoardMetadata requires CanEditStructure (Director+/owner, spec
// 5.4's "change board metadata").
func UpdateBoardMetadata(ctx context.Context, pool *pgxpool.Pool, userID, boardID, title, description string) (*Storyboard, error) {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return nil, err
	}
	if ok, err := CanEditStructure(ctx, pool, userID, board); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrNotAuthorized
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrInvalidTitle
	}
	if _, err := pool.Exec(ctx, `
		UPDATE storyboards SET title = $2, description = $3, updated_at = NOW()
		WHERE id = $1
	`, boardID, title, description); err != nil {
		return nil, err
	}
	return LoadBoard(ctx, pool, boardID)
}

// ArchiveBoard/UnarchiveBoard require CanArchiveOrDeleteBoard (owner or
// Operator only, spec 5.5).
func ArchiveBoard(ctx context.Context, pool *pgxpool.Pool, userID, boardID string) (*Storyboard, error) {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return nil, err
	}
	if ok, err := CanArchiveOrDeleteBoard(ctx, pool, userID, board); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrNotAuthorized
	}
	if _, err := pool.Exec(ctx, `
		UPDATE storyboards SET archived_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND archived_at IS NULL
	`, boardID); err != nil {
		return nil, err
	}
	return LoadBoard(ctx, pool, boardID)
}

func UnarchiveBoard(ctx context.Context, pool *pgxpool.Pool, userID, boardID string) (*Storyboard, error) {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return nil, err
	}
	if ok, err := CanArchiveOrDeleteBoard(ctx, pool, userID, board); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrNotAuthorized
	}
	if _, err := pool.Exec(ctx, `
		UPDATE storyboards SET archived_at = NULL, updated_at = NOW()
		WHERE id = $1
	`, boardID); err != nil {
		return nil, err
	}
	return LoadBoard(ctx, pool, boardID)
}

// DeleteBoard requires CanArchiveOrDeleteBoard (owner or Operator only).
// Cascades to grants/columns/bands/rows/cards via FK ON DELETE CASCADE.
func DeleteBoard(ctx context.Context, pool *pgxpool.Pool, userID, boardID string) error {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return err
	}
	if ok, err := CanArchiveOrDeleteBoard(ctx, pool, userID, board); err != nil {
		return err
	} else if !ok {
		return ErrNotAuthorized
	}
	_, err = pool.Exec(ctx, `DELETE FROM storyboards WHERE id = $1`, boardID)
	return err
}
