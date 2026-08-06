package storyboards

import (
	"context"
	"sync"
	"testing"

	"victory/backend/internal/dbtest"
)

func TestMultiCardCellOrderingStable(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Ordering Board")
	rows, _ := ListRows(ctx, pool, board.ID)
	cols, _ := ListColumns(ctx, pool, board.ID)
	rowID, colID := rows[0].ID, cols[0].ID

	var ids []string
	for i := 0; i < 4; i++ {
		c, err := CreateCard(ctx, pool, owner, board.ID, rowID, colID, "Card", "", "")
		if err != nil {
			t.Fatalf("create card %d: %v", i, err)
		}
		ids = append(ids, c.ID)
	}

	all, err := ListCardsForBoard(ctx, pool, board.ID)
	if err != nil {
		t.Fatalf("list cards: %v", err)
	}
	if len(all) != 4 {
		t.Fatalf("expected 4 cards, got %d", len(all))
	}
	for i, c := range all {
		if c.ID != ids[i] {
			t.Fatalf("expected stable creation order at index %d: want %s got %s", i, ids[i], c.ID)
		}
	}

	// Reverse the order explicitly.
	reversed := []string{ids[3], ids[2], ids[1], ids[0]}
	if err := ReorderCardsInCell(ctx, pool, owner, board.ID, rowID, colID, reversed); err != nil {
		t.Fatalf("reorder: %v", err)
	}
	all, _ = ListCardsForBoard(ctx, pool, board.ID)
	for i, c := range all {
		if c.ID != reversed[i] {
			t.Fatalf("expected reversed order at index %d: want %s got %s", i, reversed[i], c.ID)
		}
	}
}

func TestMoveCardBetweenCells(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Move Board")
	rows, _ := ListRows(ctx, pool, board.ID)
	cols, _ := ListColumns(ctx, pool, board.ID)
	rowID, colID := rows[0].ID, cols[0].ID

	col2, err := AddColumn(ctx, pool, owner, board.ID, "Col 2")
	if err != nil {
		t.Fatalf("add col2: %v", err)
	}

	card, err := CreateCard(ctx, pool, owner, board.ID, rowID, colID, "Movable", "", "")
	if err != nil {
		t.Fatalf("create card: %v", err)
	}

	moved, err := MoveCard(ctx, pool, owner, board.ID, card.ID, card.Version, rowID, col2.ID)
	if err != nil {
		t.Fatalf("move card: %v", err)
	}
	if moved.ColumnID != col2.ID {
		t.Fatalf("expected card in col2, got %s", moved.ColumnID)
	}
	if moved.Version != card.Version+1 {
		t.Fatalf("expected version bumped, got %d want %d", moved.Version, card.Version+1)
	}
}

func TestConcurrentMoveCardNeverDuplicates(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Concurrency Board")
	rows, _ := ListRows(ctx, pool, board.ID)
	cols, _ := ListColumns(ctx, pool, board.ID)
	rowID, colID := rows[0].ID, cols[0].ID

	col2, err := AddColumn(ctx, pool, owner, board.ID, "Col 2")
	if err != nil {
		t.Fatalf("add col2: %v", err)
	}
	col3, err := AddColumn(ctx, pool, owner, board.ID, "Col 3")
	if err != nil {
		t.Fatalf("add col3: %v", err)
	}

	card, err := CreateCard(ctx, pool, owner, board.ID, rowID, colID, "Racer", "", "")
	if err != nil {
		t.Fatalf("create card: %v", err)
	}
	baseVersion := card.Version

	var wg sync.WaitGroup
	results := make([]error, 2)
	targets := []string{col2.ID, col3.ID}
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := MoveCard(ctx, pool, owner, board.ID, card.ID, baseVersion, rowID, targets[i])
			results[i] = err
		}(i)
	}
	wg.Wait()

	successCount := 0
	conflictCount := 0
	for _, err := range results {
		if err == nil {
			successCount++
		} else if err == ErrCardVersionConflict {
			conflictCount++
		} else {
			t.Fatalf("unexpected error from concurrent move: %v", err)
		}
	}
	if successCount != 1 || conflictCount != 1 {
		t.Fatalf("expected exactly one success and one version conflict, got success=%d conflict=%d", successCount, conflictCount)
	}

	var total int
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM storyboard_cards WHERE id = $1`, card.ID).Scan(&total)
	if total != 1 {
		t.Fatalf("expected exactly one row for the card after concurrent moves, found %d", total)
	}
}

func TestLockedCardBlocksCrewDirectorCanUnlock(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	crew := insertTestUser(t, pool, "sb_crew")
	board := mustCreateBoard(t, pool, owner, "Lock Board")
	var crewHandle string
	pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, crew).Scan(&crewHandle)
	if _, err := AddGrant(ctx, pool, owner, board.ID, crewHandle, "crew"); err != nil {
		t.Fatalf("grant crew: %v", err)
	}
	rows, _ := ListRows(ctx, pool, board.ID)
	cols, _ := ListColumns(ctx, pool, board.ID)
	rowID, colID := rows[0].ID, cols[0].ID

	card, err := CreateCard(ctx, pool, crew, board.ID, rowID, colID, "Lockable", "", "")
	if err != nil {
		t.Fatalf("crew create card: %v", err)
	}

	locked, err := SetCardLock(ctx, pool, owner, board.ID, card.ID, true)
	if err != nil {
		t.Fatalf("owner lock card: %v", err)
	}
	if !locked.IsLocked {
		t.Fatalf("expected card locked")
	}

	edit := CardEdit{Title: "Blocked Edit"}
	if _, err := UpdateCard(ctx, pool, crew, board.ID, card.ID, locked.Version, edit); err != ErrCardLocked {
		t.Fatalf("expected ErrCardLocked for crew editing a locked card, got %v", err)
	}

	// Director+/owner may still edit a locked card directly.
	edited, err := UpdateCard(ctx, pool, owner, board.ID, card.ID, locked.Version, edit)
	if err != nil {
		t.Fatalf("owner should edit locked card: %v", err)
	}
	if edited.Title != "Blocked Edit" {
		t.Fatalf("expected owner's edit to apply")
	}

	unlocked, err := SetCardLock(ctx, pool, owner, board.ID, card.ID, false)
	if err != nil {
		t.Fatalf("unlock: %v", err)
	}
	if _, err := UpdateCard(ctx, pool, crew, board.ID, card.ID, unlocked.Version, CardEdit{Title: "Now OK"}); err != nil {
		t.Fatalf("crew should edit after unlock: %v", err)
	}
}

func TestLockedBandBlocksCardCreation(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	crew := insertTestUser(t, pool, "sb_crew")
	board := mustCreateBoard(t, pool, owner, "Locked Band Board")
	var crewHandle string
	pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, crew).Scan(&crewHandle)
	if _, err := AddGrant(ctx, pool, owner, board.ID, crewHandle, "crew"); err != nil {
		t.Fatalf("grant crew: %v", err)
	}
	rows, _ := ListRows(ctx, pool, board.ID)
	cols, _ := ListColumns(ctx, pool, board.ID)
	rowID, colID := rows[0].ID, cols[0].ID
	bandID := rows[0].BandID

	if _, err := SetBandLock(ctx, pool, owner, board.ID, bandID, true); err != nil {
		t.Fatalf("lock band: %v", err)
	}

	if _, err := CreateCard(ctx, pool, crew, board.ID, rowID, colID, "Nope", "", ""); err != ErrBandLocked {
		t.Fatalf("expected ErrBandLocked for crew creating a card in a locked band, got %v", err)
	}

	// Director+/owner may still create in a locked band.
	if _, err := CreateCard(ctx, pool, owner, board.ID, rowID, colID, "OK", "", ""); err != nil {
		t.Fatalf("owner should create card in locked band: %v", err)
	}
}

func TestFrontBackPreservedThroughEdit(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Front Back Board")
	rows, _ := ListRows(ctx, pool, board.ID)
	cols, _ := ListColumns(ctx, pool, board.ID)

	card, err := CreateCard(ctx, pool, owner, board.ID, rows[0].ID, cols[0].ID, "T", "front stuff", "back stuff")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if card.FrontText != "front stuff" || card.BackText != "back stuff" {
		t.Fatalf("front/back not preserved on create")
	}
	edited, err := UpdateCard(ctx, pool, owner, board.ID, card.ID, card.Version, CardEdit{
		Title: "T2", FrontText: "front2", BackText: "back2",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if edited.FrontText != "front2" || edited.BackText != "back2" {
		t.Fatalf("front/back not preserved on update")
	}
}
