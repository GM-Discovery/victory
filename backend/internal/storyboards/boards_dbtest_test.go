package storyboards

import (
	"context"
	"testing"

	"victory/backend/internal/dbtest"
)

func TestCreateBoardOwnerAndDefaultStructure(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")

	board, err := CreateBoard(ctx, pool, owner, "My Board", "a board", nil, "", nil)
	if err != nil {
		t.Fatalf("create board: %v", err)
	}
	if board.OwnerUserID != owner {
		t.Fatalf("expected owner %s, got %s", owner, board.OwnerUserID)
	}

	var colCount, bandCount, rowCount int
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM storyboard_columns WHERE storyboard_id = $1`, board.ID).Scan(&colCount)
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM storyboard_bands WHERE storyboard_id = $1`, board.ID).Scan(&bandCount)
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM storyboard_rows WHERE storyboard_id = $1`, board.ID).Scan(&rowCount)
	if colCount < 1 || bandCount < 1 || rowCount < 1 {
		t.Fatalf("expected a valid board (>=1 column/band/row), got cols=%d bands=%d rows=%d", colCount, bandCount, rowCount)
	}
}

func TestOwnerHasFullAuthorityNonOwnerHasNone(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	stranger := insertTestUser(t, pool, "sb_stranger")

	board := mustCreateBoard(t, pool, owner, "Owner Board")

	if ok, err := CanViewBoard(ctx, pool, owner, board); err != nil || !ok {
		t.Fatalf("owner should view: ok=%v err=%v", ok, err)
	}
	if ok, err := CanEditStructure(ctx, pool, owner, board); err != nil || !ok {
		t.Fatalf("owner should edit structure: ok=%v err=%v", ok, err)
	}
	if ok, err := CanManageSharing(ctx, pool, owner, board); err != nil || !ok {
		t.Fatalf("owner should manage sharing: ok=%v err=%v", ok, err)
	}
	if ok, err := CanArchiveOrDeleteBoard(ctx, pool, owner, board); err != nil || !ok {
		t.Fatalf("owner should archive/delete: ok=%v err=%v", ok, err)
	}

	if ok, err := CanViewBoard(ctx, pool, stranger, board); err != nil || ok {
		t.Fatalf("stranger should NOT view: ok=%v err=%v", ok, err)
	}
	if ok, err := CanEditCards(ctx, pool, stranger, board); err != nil || ok {
		t.Fatalf("stranger should NOT edit cards: ok=%v err=%v", ok, err)
	}

	if _, err := UpdateBoardMetadata(ctx, pool, stranger, board.ID, "Hacked", ""); err != ErrNotAuthorized {
		t.Fatalf("expected ErrNotAuthorized for stranger metadata edit, got %v", err)
	}
	if err := DeleteBoard(ctx, pool, stranger, board.ID); err != ErrNotAuthorized {
		t.Fatalf("expected ErrNotAuthorized for stranger delete, got %v", err)
	}
}

func TestArchiveAndDeleteBoard(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Archive Me")

	archived, err := ArchiveBoard(ctx, pool, owner, board.ID)
	if err != nil {
		t.Fatalf("archive: %v", err)
	}
	if archived.ArchivedAt == nil {
		t.Fatalf("expected archived_at to be set")
	}

	unarchived, err := UnarchiveBoard(ctx, pool, owner, board.ID)
	if err != nil {
		t.Fatalf("unarchive: %v", err)
	}
	if unarchived.ArchivedAt != nil {
		t.Fatalf("expected archived_at to be cleared")
	}

	if err := DeleteBoard(ctx, pool, owner, board.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := LoadBoard(ctx, pool, board.ID); err != ErrBoardNotFound {
		t.Fatalf("expected ErrBoardNotFound after delete, got %v", err)
	}
}
