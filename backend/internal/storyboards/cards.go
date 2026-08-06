package storyboards

// Card CRUD (spec 1.3, 1.12, 6.5). Mutations require CanEditCards (Crew+).
// A locked card or a card in a locked band additionally requires
// CanEditStructure (Director+/owner/Operator) -- lock only ever restricts
// Crew, never Director+/owner, matching bands.go's UpdateBandLabel
// treatment of band locks. version is optimistic concurrency (spec 4.7):
// UpdateCard/MoveCard/DeleteCard all take the caller's last-seen version and
// fail with ErrCardVersionConflict if it no longer matches, so two
// concurrent edits of the same card can never silently overwrite each
// other -- and because every mutation is a single-row UPDATE/DELETE (never
// an insert-then-delete), a card can never be duplicated by a race either.

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCardNotFound        = errors.New("card_not_found")
	ErrCardVersionConflict = errors.New("card_version_conflict")
	ErrCardLocked          = errors.New("card_locked")
)

func loadCardRow(ctx context.Context, pool *pgxpool.Pool, boardID, cardID string) (*StoryboardCard, error) {
	var c StoryboardCard
	var authorID *string
	err := pool.QueryRow(ctx, `
		SELECT c.id::text, c.storyboard_id::text, c.row_id::text, c.column_id::text, c.sort_order_in_cell,
		       c.title, c.front_text, c.back_text, c.category, c.color_token,
		       c.hidden_from_audience, c.is_locked, c.author_user_id::text, COALESCE(u.handle, ''),
		       c.version, c.created_at, c.updated_at
		FROM storyboard_cards c
		LEFT JOIN users u ON u.id = c.author_user_id
		WHERE c.id = $1 AND c.storyboard_id = $2
	`, cardID, boardID).Scan(&c.ID, &c.StoryboardID, &c.RowID, &c.ColumnID, &c.SortOrderInCell,
		&c.Title, &c.FrontText, &c.BackText, &c.Category, &c.ColorToken,
		&c.HiddenFromAudience, &c.IsLocked, &authorID, &c.AuthorHandle,
		&c.Version, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCardNotFound
	}
	if err != nil {
		return nil, err
	}
	if authorID != nil {
		c.AuthorUserID = *authorID
	}
	return &c, nil
}

// LoadCard fetches one card by ID, scoped to boardID.
func LoadCard(ctx context.Context, pool *pgxpool.Pool, boardID, cardID string) (*StoryboardCard, error) {
	return loadCardRow(ctx, pool, boardID, cardID)
}

// ListCardsForBoard returns every card on the board, ordered by cell then
// sort_order_in_cell -- callers needing per-cell grouping (snapshot.go)
// group this slice themselves by (RowID, ColumnID).
func ListCardsForBoard(ctx context.Context, pool *pgxpool.Pool, boardID string) ([]StoryboardCard, error) {
	rows, err := pool.Query(ctx, `
		SELECT c.id::text, c.storyboard_id::text, c.row_id::text, c.column_id::text, c.sort_order_in_cell,
		       c.title, c.front_text, c.back_text, c.category, c.color_token,
		       c.hidden_from_audience, c.is_locked, c.author_user_id::text, COALESCE(u.handle, ''),
		       c.version, c.created_at, c.updated_at
		FROM storyboard_cards c
		LEFT JOIN users u ON u.id = c.author_user_id
		WHERE c.storyboard_id = $1
		ORDER BY c.row_id, c.column_id, c.sort_order_in_cell
	`, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StoryboardCard{}
	for rows.Next() {
		var c StoryboardCard
		var authorID *string
		if err := rows.Scan(&c.ID, &c.StoryboardID, &c.RowID, &c.ColumnID, &c.SortOrderInCell,
			&c.Title, &c.FrontText, &c.BackText, &c.Category, &c.ColorToken,
			&c.HiddenFromAudience, &c.IsLocked, &authorID, &c.AuthorHandle,
			&c.Version, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		if authorID != nil {
			c.AuthorUserID = *authorID
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// isBandLockedForRow reports whether rowID's band is currently locked
// (spec 5.6: a locked band blocks card creation/editing/movement within
// it).
func isBandLockedForRow(ctx context.Context, pool *pgxpool.Pool, rowID string) (bool, error) {
	var locked bool
	err := pool.QueryRow(ctx, `
		SELECT b.is_locked FROM storyboard_rows r
		JOIN storyboard_bands b ON b.id = r.band_id
		WHERE r.id = $1
	`, rowID).Scan(&locked)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrRowNotFound
	}
	return locked, err
}

// canMutateCard: Crew+ in general; if the card itself is locked or its row's
// band is locked, only Director+/owner/Operator may proceed (lock never
// restricts Director+, matching bands.go's UpdateBandLabel).
func canMutateCard(ctx context.Context, pool *pgxpool.Pool, userID string, board *Storyboard, card *StoryboardCard) (bool, error) {
	canCards, err := CanEditCards(ctx, pool, userID, board)
	if err != nil {
		return false, err
	}
	if !canCards {
		return false, nil
	}
	locked := false
	if card != nil {
		locked = card.IsLocked
	}
	if !locked && card != nil {
		bandLocked, err := isBandLockedForRow(ctx, pool, card.RowID)
		if err != nil {
			return false, err
		}
		locked = bandLocked
	}
	if !locked {
		return true, nil
	}
	return CanEditStructure(ctx, pool, userID, board)
}

// CreateCard requires CanEditCards (Crew+), and the target row/column must
// belong to boardID and the target row's band must be unlocked (or the
// caller must be Director+).
func CreateCard(ctx context.Context, pool *pgxpool.Pool, userID, boardID, rowID, columnID, title, frontText, backText string) (*StoryboardCard, error) {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return nil, err
	}
	if _, err := loadRow(ctx, pool, boardID, rowID); err != nil {
		return nil, err
	}
	if _, err := loadColumn(ctx, pool, boardID, columnID); err != nil {
		return nil, err
	}

	canCards, err := CanEditCards(ctx, pool, userID, board)
	if err != nil {
		return nil, err
	}
	if !canCards {
		return nil, ErrNotAuthorized
	}
	bandLocked, err := isBandLockedForRow(ctx, pool, rowID)
	if err != nil {
		return nil, err
	}
	if bandLocked {
		if canStructure, err := CanEditStructure(ctx, pool, userID, board); err != nil {
			return nil, err
		} else if !canStructure {
			return nil, ErrBandLocked
		}
	}

	var nextOrder int
	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(sort_order_in_cell) + 1, 0) FROM storyboard_cards WHERE row_id = $1 AND column_id = $2
	`, rowID, columnID).Scan(&nextOrder); err != nil {
		return nil, err
	}

	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO storyboard_cards (storyboard_id, row_id, column_id, sort_order_in_cell, title, front_text, back_text, author_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::text
	`, boardID, rowID, columnID, nextOrder, title, frontText, backText, userID).Scan(&id); err != nil {
		return nil, err
	}
	return loadCardRow(ctx, pool, boardID, id)
}

// CardEdit carries only the fields UpdateCard should change. Hidden state
// is a *bool so callers that lack Director+ authority (spec 1.9: hidden
// toggling is a structural-adjacent capability, not a Crew one) can omit
// it entirely rather than needing to know the current value; the check
// below still enforces who may set it.
type CardEdit struct {
	Title              string
	FrontText          string
	BackText           string
	Category           string
	ColorToken         string
	HiddenFromAudience *bool
}

// UpdateCard applies edit with optimistic concurrency: baseVersion must
// match the card's current version or ErrCardVersionConflict is returned.
// Setting HiddenFromAudience requires CanEditStructure (Director+/owner) --
// Crew never sees or sets this toggle (spec 1.9's role table, 9).
func UpdateCard(ctx context.Context, pool *pgxpool.Pool, userID, boardID, cardID string, baseVersion int, edit CardEdit) (*StoryboardCard, error) {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return nil, err
	}
	card, err := loadCardRow(ctx, pool, boardID, cardID)
	if err != nil {
		return nil, err
	}
	if ok, err := canMutateCard(ctx, pool, userID, board, card); err != nil {
		return nil, err
	} else if !ok {
		if card.IsLocked {
			return nil, ErrCardLocked
		}
		return nil, ErrNotAuthorized
	}

	hidden := card.HiddenFromAudience
	if edit.HiddenFromAudience != nil {
		if ok, err := CanEditStructure(ctx, pool, userID, board); err != nil {
			return nil, err
		} else if !ok {
			return nil, ErrNotAuthorized
		}
		hidden = *edit.HiddenFromAudience
	}

	tag, err := pool.Exec(ctx, `
		UPDATE storyboard_cards
		SET title = $3, front_text = $4, back_text = $5, category = $6, color_token = $7,
		    hidden_from_audience = $8, version = version + 1, updated_at = NOW()
		WHERE id = $1 AND storyboard_id = $9 AND version = $2
	`, cardID, baseVersion, edit.Title, edit.FrontText, edit.BackText, edit.Category, edit.ColorToken, hidden, boardID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrCardVersionConflict
	}
	return loadCardRow(ctx, pool, boardID, cardID)
}

// MoveCard relocates a card to a (possibly different) row/column, appended
// at the end of the target cell's order. Optimistic concurrency via
// baseVersion, same as UpdateCard.
func MoveCard(ctx context.Context, pool *pgxpool.Pool, userID, boardID, cardID string, baseVersion int, targetRowID, targetColumnID string) (*StoryboardCard, error) {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return nil, err
	}
	card, err := loadCardRow(ctx, pool, boardID, cardID)
	if err != nil {
		return nil, err
	}
	if ok, err := canMutateCard(ctx, pool, userID, board, card); err != nil {
		return nil, err
	} else if !ok {
		if card.IsLocked {
			return nil, ErrCardLocked
		}
		return nil, ErrNotAuthorized
	}
	if _, err := loadRow(ctx, pool, boardID, targetRowID); err != nil {
		return nil, err
	}
	if _, err := loadColumn(ctx, pool, boardID, targetColumnID); err != nil {
		return nil, err
	}
	targetBandLocked, err := isBandLockedForRow(ctx, pool, targetRowID)
	if err != nil {
		return nil, err
	}
	if targetBandLocked {
		if canStructure, err := CanEditStructure(ctx, pool, userID, board); err != nil {
			return nil, err
		} else if !canStructure {
			return nil, ErrBandLocked
		}
	}

	var nextOrder int
	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(sort_order_in_cell) + 1, 0) FROM storyboard_cards WHERE row_id = $1 AND column_id = $2
	`, targetRowID, targetColumnID).Scan(&nextOrder); err != nil {
		return nil, err
	}

	tag, err := pool.Exec(ctx, `
		UPDATE storyboard_cards
		SET row_id = $3, column_id = $4, sort_order_in_cell = $5, version = version + 1, updated_at = NOW()
		WHERE id = $1 AND storyboard_id = $6 AND version = $2
	`, cardID, baseVersion, targetRowID, targetColumnID, nextOrder, boardID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrCardVersionConflict
	}
	return loadCardRow(ctx, pool, boardID, cardID)
}

// ReorderCardsInCell requires CanEditCards and orderedCardIDs to be
// exactly the cell's existing card set. No unique-order DB constraint
// exists on sort_order_in_cell, so this can write final positions directly
// without a shift pass.
func ReorderCardsInCell(ctx context.Context, pool *pgxpool.Pool, userID, boardID, rowID, columnID string, orderedCardIDs []string) error {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return err
	}
	if ok, err := CanEditCards(ctx, pool, userID, board); err != nil {
		return err
	} else if !ok {
		return ErrNotAuthorized
	}

	rows, err := pool.Query(ctx, `SELECT id::text FROM storyboard_cards WHERE row_id = $1 AND column_id = $2`, rowID, columnID)
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
	if len(orderedCardIDs) != count {
		return ErrInvalidResolution
	}
	seen := map[string]bool{}
	for _, id := range orderedCardIDs {
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
	for i, id := range orderedCardIDs {
		if _, err := tx.Exec(ctx, `
			UPDATE storyboard_cards SET sort_order_in_cell = $2, version = version + 1, updated_at = NOW()
			WHERE id = $1
		`, id, i); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// DeleteCard requires canMutateCard (Crew+, Director+ if locked), with the
// same optimistic-concurrency contract as UpdateCard/MoveCard.
func DeleteCard(ctx context.Context, pool *pgxpool.Pool, userID, boardID, cardID string, baseVersion int) error {
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		return err
	}
	card, err := loadCardRow(ctx, pool, boardID, cardID)
	if err != nil {
		return err
	}
	if ok, err := canMutateCard(ctx, pool, userID, board, card); err != nil {
		return err
	} else if !ok {
		if card.IsLocked {
			return ErrCardLocked
		}
		return ErrNotAuthorized
	}
	tag, err := pool.Exec(ctx, `DELETE FROM storyboard_cards WHERE id = $1 AND storyboard_id = $2 AND version = $3`, cardID, boardID, baseVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCardVersionConflict
	}
	return nil
}

// SetCardLock requires CanEditStructure (Director+/owner only, spec 5.4).
func SetCardLock(ctx context.Context, pool *pgxpool.Pool, userID, boardID, cardID string, locked bool) (*StoryboardCard, error) {
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
		UPDATE storyboard_cards SET is_locked = $3, version = version + 1, updated_at = NOW()
		WHERE id = $1 AND storyboard_id = $2
	`, cardID, boardID, locked)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrCardNotFound
	}
	return loadCardRow(ctx, pool, boardID, cardID)
}
