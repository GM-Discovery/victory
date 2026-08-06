package storyboards

import (
	"context"
	"testing"

	"victory/backend/internal/dbtest"
)

func TestColumnLimitEnforced(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Column Limit Board")

	// mustCreateBoard already inserted one default column; add up to the
	// limit.
	cols, err := ListColumns(ctx, pool, board.ID)
	if err != nil {
		t.Fatalf("list columns: %v", err)
	}
	for i := len(cols); i < MaxColumns; i++ {
		if _, err := AddColumn(ctx, pool, owner, board.ID, "Col"); err != nil {
			t.Fatalf("add column %d: %v", i, err)
		}
	}
	if _, err := AddColumn(ctx, pool, owner, board.ID, "Overflow"); err != ErrColumnLimitExceeded {
		t.Fatalf("expected ErrColumnLimitExceeded for the 201st column, got %v", err)
	}
}

func TestRowBelongsToExactlyOneBandAndBandsNeverOverlap(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Band Board")

	bandA, err := AddBand(ctx, pool, owner, board.ID, "Act I")
	if err != nil {
		t.Fatalf("add band A: %v", err)
	}
	bandB, err := AddBand(ctx, pool, owner, board.ID, "Act II")
	if err != nil {
		t.Fatalf("add band B: %v", err)
	}

	row1, err := AddRow(ctx, pool, owner, board.ID, bandA.ID, "Scene 1")
	if err != nil {
		t.Fatalf("add row1: %v", err)
	}
	row2, err := AddRow(ctx, pool, owner, board.ID, bandA.ID, "Scene 2")
	if err != nil {
		t.Fatalf("add row2: %v", err)
	}

	// Every row belongs to exactly one band -- both rows are in bandA.
	allRows, err := ListRows(ctx, pool, board.ID)
	if err != nil {
		t.Fatalf("list rows: %v", err)
	}
	bandOf := map[string]string{}
	for _, r := range allRows {
		if _, dup := bandOf[r.ID]; dup {
			t.Fatalf("row %s appeared twice in listing -- would indicate shared/duplicated band membership", r.ID)
		}
		bandOf[r.ID] = r.BandID
	}
	if bandOf[row1.ID] != bandA.ID || bandOf[row2.ID] != bandA.ID {
		t.Fatalf("expected both rows in bandA")
	}

	// Move row2 to bandB; bands never share rows -- row2 must now belong
	// only to bandB, never to both.
	moved, err := MoveRowToBand(ctx, pool, owner, board.ID, row2.ID, bandB.ID)
	if err != nil {
		t.Fatalf("move row: %v", err)
	}
	if moved.BandID != bandB.ID {
		t.Fatalf("expected row2 in bandB after move, got %s", moved.BandID)
	}
	allRows, _ = ListRows(ctx, pool, board.ID)
	for _, r := range allRows {
		if r.ID == row2.ID && r.BandID != bandB.ID {
			t.Fatalf("row2 still shows bandA membership after move -- band overlap")
		}
	}
}

func TestOccupiedColumnRemovalRequiresResolution(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Occupied Column Board")
	rows, _ := ListRows(ctx, pool, board.ID)
	cols, _ := ListColumns(ctx, pool, board.ID)
	rowID, colID := rows[0].ID, cols[0].ID

	card, err := CreateCard(ctx, pool, owner, board.ID, rowID, colID, "Occupant", "", "")
	if err != nil {
		t.Fatalf("create card: %v", err)
	}

	// No resolution -> refused, card untouched.
	if err := RemoveColumn(ctx, pool, owner, board.ID, colID, "", ""); err != ErrColumnOccupied {
		t.Fatalf("expected ErrColumnOccupied, got %v", err)
	}
	if _, err := LoadCard(ctx, pool, board.ID, card.ID); err != nil {
		t.Fatalf("card should survive a refused removal: %v", err)
	}

	col2, err := AddColumn(ctx, pool, owner, board.ID, "Col 2")
	if err != nil {
		t.Fatalf("add col2: %v", err)
	}

	// move_cards resolution relocates the card, then removes the column.
	if err := RemoveColumn(ctx, pool, owner, board.ID, colID, "move_cards", col2.ID); err != nil {
		t.Fatalf("remove column with move_cards: %v", err)
	}
	moved, err := LoadCard(ctx, pool, board.ID, card.ID)
	if err != nil {
		t.Fatalf("load moved card: %v", err)
	}
	if moved.ColumnID != col2.ID {
		t.Fatalf("expected card moved to col2, got %s", moved.ColumnID)
	}
	if _, err := loadColumn(ctx, pool, board.ID, colID); err != ErrColumnNotFound {
		t.Fatalf("expected removed column to be gone, got %v", err)
	}
}

func TestOccupiedBandRemovalRequiresResolution(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Occupied Band Board")
	bands, _ := ListBands(ctx, pool, board.ID)
	firstBandID := bands[0].ID

	if err := RemoveBand(ctx, pool, owner, board.ID, firstBandID, "", ""); err != ErrBandOccupied {
		t.Fatalf("expected ErrBandOccupied (band has the board's default row), got %v", err)
	}

	otherBand, err := AddBand(ctx, pool, owner, board.ID, "Other Band")
	if err != nil {
		t.Fatalf("add other band: %v", err)
	}
	if err := RemoveBand(ctx, pool, owner, board.ID, firstBandID, "move_rows", otherBand.ID); err != nil {
		t.Fatalf("remove band with move_rows: %v", err)
	}
	if _, err := loadBand(ctx, pool, board.ID, firstBandID); err != ErrBandNotFound {
		t.Fatalf("expected band removed, got %v", err)
	}
	rows, _ := ListRows(ctx, pool, board.ID)
	for _, r := range rows {
		if r.BandID != otherBand.ID {
			t.Fatalf("expected all rows moved to otherBand, found row in %s", r.BandID)
		}
	}
}

func TestCrewCannotAlterStructureButCanEditUnlockedBandLabel(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	crew := insertTestUser(t, pool, "sb_crew")
	board := mustCreateBoard(t, pool, owner, "Crew Board")
	var crewHandle string
	pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, crew).Scan(&crewHandle)
	if _, err := AddGrant(ctx, pool, owner, board.ID, crewHandle, "crew"); err != nil {
		t.Fatalf("grant crew: %v", err)
	}
	bands, _ := ListBands(ctx, pool, board.ID)
	bandID := bands[0].ID

	if _, err := UpdateBandLabel(ctx, pool, crew, board.ID, bandID, "Renamed by Crew", ""); err != nil {
		t.Fatalf("crew should edit unlocked band label: %v", err)
	}

	if _, err := SetBandLock(ctx, pool, owner, board.ID, bandID, true); err != nil {
		t.Fatalf("owner lock band: %v", err)
	}
	if _, err := UpdateBandLabel(ctx, pool, crew, board.ID, bandID, "Blocked", ""); err != ErrBandLocked {
		t.Fatalf("expected ErrBandLocked for crew on a locked band, got %v", err)
	}

	if _, err := AddColumn(ctx, pool, crew, board.ID, "Nope"); err != ErrNotAuthorized {
		t.Fatalf("expected ErrNotAuthorized for crew adding a column, got %v", err)
	}
	if _, err := AddBand(ctx, pool, crew, board.ID, "Nope"); err != ErrNotAuthorized {
		t.Fatalf("expected ErrNotAuthorized for crew adding a band, got %v", err)
	}
	if _, err := AddRow(ctx, pool, crew, board.ID, bandID, "Nope"); err != ErrNotAuthorized {
		t.Fatalf("expected ErrNotAuthorized for crew adding a row, got %v", err)
	}
}
