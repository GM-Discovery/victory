package storyboards

// Kernel 81 bounded backend additions: SetCardImage (spec 2.6/9.1) and
// SwapCards (spec 7.4). Everything else the presentation rework needed
// (grid model, drag reconcile, occupied-cell dialog, palette, card visuals)
// is frontend-only and has no backend test surface.

import (
	"context"
	"testing"

	"victory/backend/internal/dbtest"
)

func TestSetCardImageRoundTripAndClear(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Image Board")
	rows, _ := ListRows(ctx, pool, board.ID)
	cols, _ := ListColumns(ctx, pool, board.ID)

	card, err := CreateCard(ctx, pool, owner, board.ID, rows[0].ID, cols[0].ID, "Pic", "", "")
	if err != nil {
		t.Fatalf("create card: %v", err)
	}
	if card.ImageAssetID != "" {
		t.Fatalf("expected no image on creation, got %q", card.ImageAssetID)
	}

	fakeAssetID := "00000000-0000-0000-0000-000000000001"
	withImage, err := SetCardImage(ctx, pool, owner, board.ID, card.ID, fakeAssetID)
	if err != nil {
		t.Fatalf("set image: %v", err)
	}
	if withImage.ImageAssetID != fakeAssetID {
		t.Fatalf("expected image_asset_id %s, got %s", fakeAssetID, withImage.ImageAssetID)
	}
	if withImage.Version != card.Version+1 {
		t.Fatalf("expected version bump on image set, got %d want %d", withImage.Version, card.Version+1)
	}

	reloaded, err := LoadCard(ctx, pool, board.ID, card.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.ImageAssetID != fakeAssetID {
		t.Fatalf("image_asset_id did not persist through reload")
	}

	cleared, err := SetCardImage(ctx, pool, owner, board.ID, card.ID, "")
	if err != nil {
		t.Fatalf("clear image: %v", err)
	}
	if cleared.ImageAssetID != "" {
		t.Fatalf("expected image cleared, got %q", cleared.ImageAssetID)
	}
}

// TestCardImageReferenceHasNoForeignKey proves the deliberate no-FK design
// from migration 092's comment: storyboard_cards.image_asset_id is never
// validated against (or cascaded from) the assets table, so a card's
// pinned-image reference survives regardless of what later happens to the
// underlying asset row -- the frontend is solely responsible for treating
// an unresolvable id as a gravestone at render time.
func TestCardImageReferenceHasNoForeignKey(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Gravestone Board")
	rows, _ := ListRows(ctx, pool, board.ID)
	cols, _ := ListColumns(ctx, pool, board.ID)

	card, err := CreateCard(ctx, pool, owner, board.ID, rows[0].ID, cols[0].ID, "Pic", "", "")
	if err != nil {
		t.Fatalf("create card: %v", err)
	}
	// This id has never existed in `assets` at all -- if the column carried
	// a foreign key, this insert would fail outright.
	neverExistedAssetID := "00000000-0000-0000-0000-0000000000ff"
	if _, err := SetCardImage(ctx, pool, owner, board.ID, card.ID, neverExistedAssetID); err != nil {
		t.Fatalf("expected no FK violation setting a nonexistent asset id, got: %v", err)
	}
	reloaded, err := LoadCard(ctx, pool, board.ID, card.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.ImageAssetID != neverExistedAssetID {
		t.Fatalf("expected reference preserved even though the asset never existed, got %q", reloaded.ImageAssetID)
	}
}

func TestSetCardImageRespectsLockAndAuthority(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	crew := insertTestUser(t, pool, "sb_crew")
	audience := insertTestUser(t, pool, "sb_audience")
	board := mustCreateBoard(t, pool, owner, "Image Auth Board")
	var crewHandle, audienceHandle string
	pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, crew).Scan(&crewHandle)
	pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, audience).Scan(&audienceHandle)
	if _, err := AddGrant(ctx, pool, owner, board.ID, crewHandle, "crew"); err != nil {
		t.Fatalf("grant crew: %v", err)
	}
	if _, err := AddGrant(ctx, pool, owner, board.ID, audienceHandle, "audience"); err != nil {
		t.Fatalf("grant audience: %v", err)
	}
	rows, _ := ListRows(ctx, pool, board.ID)
	cols, _ := ListColumns(ctx, pool, board.ID)
	card, err := CreateCard(ctx, pool, owner, board.ID, rows[0].ID, cols[0].ID, "Pic", "", "")
	if err != nil {
		t.Fatalf("create card: %v", err)
	}

	if _, err := SetCardImage(ctx, pool, audience, board.ID, card.ID, "00000000-0000-0000-0000-000000000002"); err != ErrNotAuthorized {
		t.Fatalf("expected audience denied setting image, got %v", err)
	}
	withImage, err := SetCardImage(ctx, pool, crew, board.ID, card.ID, "00000000-0000-0000-0000-000000000002")
	if err != nil {
		t.Fatalf("crew should be able to set image: %v", err)
	}

	locked, err := SetCardLock(ctx, pool, owner, board.ID, card.ID, true)
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	_ = withImage
	if _, err := SetCardImage(ctx, pool, crew, board.ID, card.ID, "00000000-0000-0000-0000-000000000003"); err != ErrCardLocked {
		t.Fatalf("expected crew blocked by lock, got %v", err)
	}
	if _, err := SetCardImage(ctx, pool, owner, board.ID, card.ID, "00000000-0000-0000-0000-000000000003"); err != nil {
		t.Fatalf("owner should override lock: %v", err)
	}
	_ = locked
}

func TestSwapCardsAtomicExchange(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Swap Board")
	rows, _ := ListRows(ctx, pool, board.ID)
	cols, _ := ListColumns(ctx, pool, board.ID)
	col2, err := AddColumn(ctx, pool, owner, board.ID, "Col 2")
	if err != nil {
		t.Fatalf("add col2: %v", err)
	}

	cardA, err := CreateCard(ctx, pool, owner, board.ID, rows[0].ID, cols[0].ID, "A", "", "")
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	cardB, err := CreateCard(ctx, pool, owner, board.ID, rows[0].ID, col2.ID, "B", "", "")
	if err != nil {
		t.Fatalf("create B: %v", err)
	}

	newA, newB, err := SwapCards(ctx, pool, owner, board.ID, cardA.ID, cardA.Version, cardB.ID, cardB.Version)
	if err != nil {
		t.Fatalf("swap: %v", err)
	}
	if newA.ColumnID != col2.ID || newA.RowID != rows[0].ID {
		t.Fatalf("expected A to land in B's original cell, got row=%s col=%s", newA.RowID, newA.ColumnID)
	}
	if newB.ColumnID != cols[0].ID || newB.RowID != rows[0].ID {
		t.Fatalf("expected B to land in A's original cell, got row=%s col=%s", newB.RowID, newB.ColumnID)
	}
	if newA.Version != cardA.Version+1 || newB.Version != cardB.Version+1 {
		t.Fatalf("expected both versions bumped exactly once")
	}

	all, err := ListCardsForBoard(ctx, pool, board.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected exactly 2 cards to still exist after swap, got %d", len(all))
	}
}

func TestSwapCardsRejectsStaleVersion(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Swap Stale Board")
	rows, _ := ListRows(ctx, pool, board.ID)
	cols, _ := ListColumns(ctx, pool, board.ID)
	col2, _ := AddColumn(ctx, pool, owner, board.ID, "Col 2")

	cardA, _ := CreateCard(ctx, pool, owner, board.ID, rows[0].ID, cols[0].ID, "A", "", "")
	cardB, _ := CreateCard(ctx, pool, owner, board.ID, rows[0].ID, col2.ID, "B", "", "")

	// Stale A's version by editing it out from under the swap attempt.
	if _, err := UpdateCard(ctx, pool, owner, board.ID, cardA.ID, cardA.Version, CardEdit{Title: "Changed"}); err != nil {
		t.Fatalf("prime staleness: %v", err)
	}

	if _, _, err := SwapCards(ctx, pool, owner, board.ID, cardA.ID, cardA.Version, cardB.ID, cardB.Version); err != ErrCardVersionConflict {
		t.Fatalf("expected ErrCardVersionConflict on stale A, got %v", err)
	}

	// Board must be unchanged -- swap is all-or-nothing.
	reloadedA, _ := LoadCard(ctx, pool, board.ID, cardA.ID)
	reloadedB, _ := LoadCard(ctx, pool, board.ID, cardB.ID)
	if reloadedA.ColumnID != cols[0].ID || reloadedB.ColumnID != col2.ID {
		t.Fatalf("expected no placement change after a rejected swap")
	}
}

func TestSwapCardsRespectsLockAndBandLock(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	crew := insertTestUser(t, pool, "sb_crew")
	board := mustCreateBoard(t, pool, owner, "Swap Lock Board")
	var crewHandle string
	pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, crew).Scan(&crewHandle)
	if _, err := AddGrant(ctx, pool, owner, board.ID, crewHandle, "crew"); err != nil {
		t.Fatalf("grant crew: %v", err)
	}
	rows, _ := ListRows(ctx, pool, board.ID)
	cols, _ := ListColumns(ctx, pool, board.ID)
	col2, _ := AddColumn(ctx, pool, owner, board.ID, "Col 2")

	cardA, _ := CreateCard(ctx, pool, crew, board.ID, rows[0].ID, cols[0].ID, "A", "", "")
	cardB, _ := CreateCard(ctx, pool, crew, board.ID, rows[0].ID, col2.ID, "B", "", "")

	lockedA, err := SetCardLock(ctx, pool, owner, board.ID, cardA.ID, true)
	if err != nil {
		t.Fatalf("lock A: %v", err)
	}
	if _, _, err := SwapCards(ctx, pool, crew, board.ID, lockedA.ID, lockedA.Version, cardB.ID, cardB.Version); err != ErrCardLocked {
		t.Fatalf("expected crew blocked swapping a locked card, got %v", err)
	}
	if _, _, err := SwapCards(ctx, pool, owner, board.ID, lockedA.ID, lockedA.Version, cardB.ID, cardB.Version); err != nil {
		t.Fatalf("owner should override lock on swap: %v", err)
	}
}

func TestSwapCardsRejectsSameCard(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Swap Same Board")
	rows, _ := ListRows(ctx, pool, board.ID)
	cols, _ := ListColumns(ctx, pool, board.ID)
	card, _ := CreateCard(ctx, pool, owner, board.ID, rows[0].ID, cols[0].ID, "A", "", "")

	if _, _, err := SwapCards(ctx, pool, owner, board.ID, card.ID, card.Version, card.ID, card.Version); err != ErrInvalidResolution {
		t.Fatalf("expected ErrInvalidResolution swapping a card with itself, got %v", err)
	}
}
