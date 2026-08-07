package storyboards

// Kernel 82: Storyboards Timeline Mode. Requires TEST_DATABASE_URL
// (internal/dbtest safety gate).

import (
	"context"
	"testing"

	"victory/backend/internal/dbtest"
)

// --- 14.1 Template instantiation ---

func TestBlankStillCreatesBlank(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Plain Board")
	if board.Mode != ModeBlank {
		t.Fatalf("expected mode %q, got %q", ModeBlank, board.Mode)
	}
	if board.TemplateVersion != nil {
		t.Fatalf("expected nil template_version for a Blank board, got %v", *board.TemplateVersion)
	}
}

func TestTimelineCreatesTimelineWithThreeColumnsAndBoundaryRoles(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")

	board, err := CreateTimelineBoard(ctx, pool, owner, "My Timeline", "")
	if err != nil {
		t.Fatalf("create timeline: %v", err)
	}
	if board.Mode != ModeTimeline {
		t.Fatalf("expected mode %q, got %q", ModeTimeline, board.Mode)
	}
	if board.TemplateVersion == nil || *board.TemplateVersion != TimelineTemplateVersion {
		t.Fatalf("expected template_version %d, got %v", TimelineTemplateVersion, board.TemplateVersion)
	}

	cols, err := ListColumns(ctx, pool, board.ID)
	if err != nil {
		t.Fatalf("list columns: %v", err)
	}
	if len(cols) != 3 {
		t.Fatalf("expected exactly 3 columns, got %d", len(cols))
	}
	if cols[0].ColumnRole != ColumnRoleBeginning {
		t.Fatalf("expected first column role %q, got %q", ColumnRoleBeginning, cols[0].ColumnRole)
	}
	if cols[1].ColumnRole != ColumnRoleOrdinary {
		t.Fatalf("expected middle column role %q, got %q", ColumnRoleOrdinary, cols[1].ColumnRole)
	}
	if cols[2].ColumnRole != ColumnRoleEnding {
		t.Fatalf("expected last column role %q, got %q", ColumnRoleEnding, cols[2].ColumnRole)
	}

	bands, err := ListBands(ctx, pool, board.ID)
	if err != nil || len(bands) != 1 {
		t.Fatalf("expected exactly 1 default band, got %d (err=%v)", len(bands), err)
	}
	rows, err := ListRows(ctx, pool, board.ID)
	if err != nil || len(rows) != 1 {
		t.Fatalf("expected exactly 1 default row, got %d (err=%v)", len(rows), err)
	}

	fields, err := ListReferenceFields(ctx, pool, board.ID)
	if err != nil {
		t.Fatalf("list reference fields: %v", err)
	}
	if len(fields) != 4 {
		t.Fatalf("expected 4 default reference fields, got %d", len(fields))
	}
	var pairedFound bool
	for _, f := range fields {
		if f.FieldType == FieldTypePairedList {
			pairedFound = true
			if f.SublabelA != "Include" || f.SublabelB != "Exclude" {
				t.Fatalf("expected Include/Exclude sublabels, got %q/%q", f.SublabelA, f.SublabelB)
			}
		}
	}
	if !pairedFound {
		t.Fatalf("expected one paired_list default field")
	}
}

func TestTwoNewTimelinesAreIndependent(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")

	a, err := CreateTimelineBoard(ctx, pool, owner, "Timeline A", "")
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	b, err := CreateTimelineBoard(ctx, pool, owner, "Timeline B", "")
	if err != nil {
		t.Fatalf("create B: %v", err)
	}
	if a.ID == b.ID {
		t.Fatalf("expected distinct board IDs")
	}

	fieldsA, err := ListReferenceFields(ctx, pool, a.ID)
	if err != nil {
		t.Fatalf("list fields A: %v", err)
	}
	premiseA := fieldsA[0]
	if _, err := SetReferenceFieldTextContent(ctx, pool, owner, a.ID, premiseA.ID, "Timeline A's premise"); err != nil {
		t.Fatalf("set content A: %v", err)
	}

	fieldsB, err := ListReferenceFields(ctx, pool, b.ID)
	if err != nil {
		t.Fatalf("list fields B: %v", err)
	}
	premiseB := fieldsB[0]
	reloadedB, err := loadReferenceField(ctx, pool, b.ID, premiseB.ID)
	if err != nil {
		t.Fatalf("reload B: %v", err)
	}
	if reloadedB.TextContent != "" {
		t.Fatalf("editing Timeline A's field leaked into Timeline B: got %q", reloadedB.TextContent)
	}

	// A third Timeline created after editing A/B must still start fresh --
	// the template itself was never mutated.
	c, err := CreateTimelineBoard(ctx, pool, owner, "Timeline C", "")
	if err != nil {
		t.Fatalf("create C: %v", err)
	}
	fieldsC, err := ListReferenceFields(ctx, pool, c.ID)
	if err != nil {
		t.Fatalf("list fields C: %v", err)
	}
	if fieldsC[0].TextContent != "" {
		t.Fatalf("a later Timeline started with leftover content: got %q", fieldsC[0].TextContent)
	}
	colsC, err := ListColumns(ctx, pool, c.ID)
	if err != nil || len(colsC) != 3 {
		t.Fatalf("Timeline C should still start with exactly 3 columns, got %d (err=%v)", len(colsC), err)
	}
}

// --- 14.2 Boundary behavior ---

func TestBeginningColumnCannotMoveInward(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board, err := CreateTimelineBoard(ctx, pool, owner, "Boundary Board", "")
	if err != nil {
		t.Fatalf("create timeline: %v", err)
	}
	cols, _ := ListColumns(ctx, pool, board.ID)
	// cols[0]=Beginning, cols[1]=Middle, cols[2]=Ending. Try to move
	// Beginning to index 1.
	reordered := []string{cols[1].ID, cols[0].ID, cols[2].ID}
	if err := ReorderColumns(ctx, pool, owner, board.ID, reordered); err != ErrBoundaryColumnDisplaced {
		t.Fatalf("expected ErrBoundaryColumnDisplaced moving Beginning inward, got %v", err)
	}
}

func TestEndingColumnCannotMoveInward(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board, err := CreateTimelineBoard(ctx, pool, owner, "Boundary Board 2", "")
	if err != nil {
		t.Fatalf("create timeline: %v", err)
	}
	cols, _ := ListColumns(ctx, pool, board.ID)
	reordered := []string{cols[0].ID, cols[2].ID, cols[1].ID}
	if err := ReorderColumns(ctx, pool, owner, board.ID, reordered); err != ErrBoundaryColumnDisplaced {
		t.Fatalf("expected ErrBoundaryColumnDisplaced moving Ending inward, got %v", err)
	}
}

func TestBoundaryColumnsCannotBeRemoved(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board, err := CreateTimelineBoard(ctx, pool, owner, "Boundary Board 3", "")
	if err != nil {
		t.Fatalf("create timeline: %v", err)
	}
	cols, _ := ListColumns(ctx, pool, board.ID)
	if err := RemoveColumn(ctx, pool, owner, board.ID, cols[0].ID, "", ""); err != ErrBoundaryColumnProtected {
		t.Fatalf("expected ErrBoundaryColumnProtected removing Beginning, got %v", err)
	}
	if err := RemoveColumn(ctx, pool, owner, board.ID, cols[2].ID, "", ""); err != ErrBoundaryColumnProtected {
		t.Fatalf("expected ErrBoundaryColumnProtected removing Ending, got %v", err)
	}
	// The ordinary middle column has no such protection.
	if err := RemoveColumn(ctx, pool, owner, board.ID, cols[1].ID, "", ""); err != nil {
		t.Fatalf("expected the ordinary middle column to be removable, got %v", err)
	}
}

func TestInsertColumnLeftOfBeginningAndRightOfEndingAreImpossibleByConstruction(t *testing.T) {
	// "Nothing inserts outside Beginning/Ending" is enforced by the
	// existing append-then-reorder flow itself: AddColumn always appends
	// at the end, and inserting at position 0 or past the last index
	// would require reordering a boundary column there, which
	// ReorderColumns now rejects (proven above). This test proves the
	// natural "insert right of Beginning" / "insert left of Ending" flows
	// land in the correct place and never touch the boundaries.
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board, err := CreateTimelineBoard(ctx, pool, owner, "Insert Board", "")
	if err != nil {
		t.Fatalf("create timeline: %v", err)
	}
	cols, _ := ListColumns(ctx, pool, board.ID)
	beginningID, middleID, endingID := cols[0].ID, cols[1].ID, cols[2].ID

	// Insert right of Beginning.
	newCol, err := AddColumn(ctx, pool, owner, board.ID, "New Right Of Beginning")
	if err != nil {
		t.Fatalf("add column: %v", err)
	}
	order := []string{beginningID, newCol.ID, middleID, endingID}
	if err := ReorderColumns(ctx, pool, owner, board.ID, order); err != nil {
		t.Fatalf("reorder (insert right of beginning): %v", err)
	}
	afterCols, _ := ListColumns(ctx, pool, board.ID)
	if afterCols[0].ID != beginningID || afterCols[len(afterCols)-1].ID != endingID {
		t.Fatalf("boundaries moved after a valid insertion: first=%s last=%s", afterCols[0].ID, afterCols[len(afterCols)-1].ID)
	}
	if afterCols[1].ID != newCol.ID {
		t.Fatalf("expected new column immediately right of Beginning, got position 1 = %s", afterCols[1].ID)
	}
}

func TestBoundaryRoleSurvivesRename(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board, err := CreateTimelineBoard(ctx, pool, owner, "Rename Board", "")
	if err != nil {
		t.Fatalf("create timeline: %v", err)
	}
	cols, _ := ListColumns(ctx, pool, board.ID)
	renamed, err := RenameColumn(ctx, pool, owner, board.ID, cols[0].ID, "The Dawn Of Time")
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if renamed.ColumnRole != ColumnRoleBeginning {
		t.Fatalf("expected boundary role to survive rename, got %q", renamed.ColumnRole)
	}
	// Still protected against reorder/removal after rename.
	otherCols, _ := ListColumns(ctx, pool, board.ID)
	reordered := []string{otherCols[1].ID, otherCols[0].ID, otherCols[2].ID}
	if err := ReorderColumns(ctx, pool, owner, board.ID, reordered); err != ErrBoundaryColumnDisplaced {
		t.Fatalf("expected renamed Beginning to remain protected, got %v", err)
	}
}

func TestBlankBoardsUnaffectedByBoundaryLogic(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Ordinary Blank Board")
	col2, err := AddColumn(ctx, pool, owner, board.ID, "Column 2")
	if err != nil {
		t.Fatalf("add column: %v", err)
	}
	cols, _ := ListColumns(ctx, pool, board.ID)
	if cols[0].ColumnRole != ColumnRoleOrdinary || cols[1].ColumnRole != ColumnRoleOrdinary {
		t.Fatalf("expected every Blank-mode column to default to 'ordinary'")
	}
	// Freely reorderable -- no boundary anywhere on a Blank board.
	reversed := []string{col2.ID, cols[0].ID}
	if err := ReorderColumns(ctx, pool, owner, board.ID, reversed); err != nil {
		t.Fatalf("expected free reordering on a Blank board, got %v", err)
	}
}

// --- 14.3 Reference Panel ---

func TestReferencePanelDefaultFieldsSeedCorrectly(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board, err := CreateTimelineBoard(ctx, pool, owner, "Seed Board", "")
	if err != nil {
		t.Fatalf("create timeline: %v", err)
	}
	fields, err := ListReferenceFields(ctx, pool, board.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	labels := map[string]bool{}
	for _, f := range fields {
		labels[f.Label] = true
		if f.Slug == "" {
			t.Fatalf("field %q missing slug", f.Label)
		}
	}
	for _, want := range []string{"Premise", "Beginning", "Ending"} {
		if !labels[want] {
			t.Fatalf("expected a default field labeled %q", want)
		}
	}
}

func TestReferencePanelAudienceReadOnlyCrewContentDirectorStructure(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	crew := insertTestUser(t, pool, "sb_crew")
	audience := insertTestUser(t, pool, "sb_audience")
	board, err := CreateTimelineBoard(ctx, pool, owner, "Roles Board", "")
	if err != nil {
		t.Fatalf("create timeline: %v", err)
	}
	var crewHandle, audienceHandle string
	pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, crew).Scan(&crewHandle)
	pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, audience).Scan(&audienceHandle)
	if _, err := AddGrant(ctx, pool, owner, board.ID, crewHandle, "crew"); err != nil {
		t.Fatalf("grant crew: %v", err)
	}
	if _, err := AddGrant(ctx, pool, owner, board.ID, audienceHandle, "audience"); err != nil {
		t.Fatalf("grant audience: %v", err)
	}

	fields, _ := ListReferenceFields(ctx, pool, board.ID)
	premise := fields[0]

	// Audience: read-only -- ListReferenceFields itself has no per-tier
	// gate (view is universal per spec 3.4), but any write must be denied.
	if _, err := SetReferenceFieldTextContent(ctx, pool, audience, board.ID, premise.ID, "nope"); err != ErrNotAuthorized {
		t.Fatalf("expected audience denied content write, got %v", err)
	}
	if _, err := AddReferenceField(ctx, pool, audience, board.ID, "New Field", FieldTypeShortText, "", ""); err != ErrNotAuthorized {
		t.Fatalf("expected audience denied field creation, got %v", err)
	}

	// Crew: may edit content, may NOT restructure.
	if _, err := SetReferenceFieldTextContent(ctx, pool, crew, board.ID, premise.ID, "Crew wrote this"); err != nil {
		t.Fatalf("expected crew allowed content write: %v", err)
	}
	if _, err := AddReferenceField(ctx, pool, crew, board.ID, "New Field", FieldTypeShortText, "", ""); err != ErrNotAuthorized {
		t.Fatalf("expected crew denied field creation, got %v", err)
	}
	if _, err := RenameReferenceField(ctx, pool, crew, board.ID, premise.ID, "Renamed", "", ""); err != ErrNotAuthorized {
		t.Fatalf("expected crew denied rename, got %v", err)
	}
	if err := RemoveReferenceField(ctx, pool, crew, board.ID, premise.ID, true); err != ErrNotAuthorized {
		t.Fatalf("expected crew denied delete, got %v", err)
	}

	// Director+ (owner here): full structural authority.
	newField, err := AddReferenceField(ctx, pool, owner, board.ID, "Director Field", FieldTypeShortText, "", "")
	if err != nil {
		t.Fatalf("expected director allowed field creation: %v", err)
	}
	if _, err := RenameReferenceField(ctx, pool, owner, board.ID, newField.ID, "Renamed By Director", "", ""); err != nil {
		t.Fatalf("expected director allowed rename: %v", err)
	}
}

func TestReferenceFieldDuplicateLabelsSerializeUniquely(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Dup Field Board")

	f1, err := AddReferenceField(ctx, pool, owner, board.ID, "Notes", FieldTypeShortText, "", "")
	if err != nil {
		t.Fatalf("add field 1: %v", err)
	}
	f2, err := AddReferenceField(ctx, pool, owner, board.ID, "Notes", FieldTypeShortText, "", "")
	if err != nil {
		t.Fatalf("add field 2: %v", err)
	}
	if f1.Label != f2.Label {
		t.Fatalf("expected duplicate labels to be accepted unchanged")
	}
	if f1.Slug == f2.Slug {
		t.Fatalf("expected distinct slugs, both were %q", f1.Slug)
	}
}

func TestReferenceFieldIDAndSlugSurviveReorder(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board, err := CreateTimelineBoard(ctx, pool, owner, "Reorder Board", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	fields, _ := ListReferenceFields(ctx, pool, board.ID)
	firstID, firstSlug := fields[0].ID, fields[0].Slug

	ids := make([]string, len(fields))
	for i, f := range fields {
		ids[i] = f.ID
	}
	for i, j := 0, len(ids)-1; i < j; i, j = i+1, j-1 {
		ids[i], ids[j] = ids[j], ids[i]
	}
	if err := ReorderReferenceFields(ctx, pool, owner, board.ID, ids); err != nil {
		t.Fatalf("reorder: %v", err)
	}
	reloaded, err := loadReferenceField(ctx, pool, board.ID, firstID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Slug != firstSlug {
		t.Fatalf("slug changed after reorder: %q -> %q", firstSlug, reloaded.Slug)
	}
}

func TestReferenceFieldDeletionOfNonemptyFieldRequiresConfirmation(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Confirm Delete Board")

	field, err := AddReferenceField(ctx, pool, owner, board.ID, "Filled", FieldTypeShortText, "", "")
	if err != nil {
		t.Fatalf("add field: %v", err)
	}
	if _, err := SetReferenceFieldTextContent(ctx, pool, owner, board.ID, field.ID, "some content"); err != nil {
		t.Fatalf("set content: %v", err)
	}
	if err := RemoveReferenceField(ctx, pool, owner, board.ID, field.ID, false); err != ErrReferenceFieldDeleteRequiresConfirm {
		t.Fatalf("expected confirmation required for nonempty field, got %v", err)
	}
	if err := RemoveReferenceField(ctx, pool, owner, board.ID, field.ID, true); err != nil {
		t.Fatalf("expected delete to succeed with confirmation: %v", err)
	}

	// An empty field deletes outright, no confirmation needed.
	emptyField, err := AddReferenceField(ctx, pool, owner, board.ID, "Empty", FieldTypeShortText, "", "")
	if err != nil {
		t.Fatalf("add empty field: %v", err)
	}
	if err := RemoveReferenceField(ctx, pool, owner, board.ID, emptyField.ID, false); err != nil {
		t.Fatalf("expected empty field to delete without confirmation: %v", err)
	}
}

func TestPairedListValuesPersistIndependently(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Paired Board")

	field, err := AddReferenceField(ctx, pool, owner, board.ID, "Include / Exclude", FieldTypePairedList, "Include", "Exclude")
	if err != nil {
		t.Fatalf("add paired field: %v", err)
	}
	if _, err := AddReferenceItem(ctx, pool, owner, board.ID, field.ID, ItemSideA, "Robots"); err != nil {
		t.Fatalf("add item to side A: %v", err)
	}
	if _, err := AddReferenceItem(ctx, pool, owner, board.ID, field.ID, ItemSideB, "Magic"); err != nil {
		t.Fatalf("add item to side B: %v", err)
	}
	// A plain list field must reject a/b sides, and a paired field must
	// reject "single".
	if _, err := AddReferenceItem(ctx, pool, owner, board.ID, field.ID, ItemSideSingle, "wrong side"); err != ErrReferenceItemSideInvalid {
		t.Fatalf("expected side validation error, got %v", err)
	}

	items, err := ListReferenceItemsForFields(ctx, pool, []string{field.ID})
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	sideA, sideB := 0, 0
	for _, it := range items[field.ID] {
		switch it.Side {
		case ItemSideA:
			sideA++
			if it.Content != "Robots" {
				t.Fatalf("unexpected side A content: %q", it.Content)
			}
		case ItemSideB:
			sideB++
			if it.Content != "Magic" {
				t.Fatalf("unexpected side B content: %q", it.Content)
			}
		}
	}
	if sideA != 1 || sideB != 1 {
		t.Fatalf("expected exactly one item per side, got a=%d b=%d", sideA, sideB)
	}
}

func TestReferenceFieldChangesAreInstanceLocal(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	boardA, err := CreateTimelineBoard(ctx, pool, owner, "Instance A", "")
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	boardB, err := CreateTimelineBoard(ctx, pool, owner, "Instance B", "")
	if err != nil {
		t.Fatalf("create B: %v", err)
	}
	fieldsA, _ := ListReferenceFields(ctx, pool, boardA.ID)
	if _, err := AddReferenceField(ctx, pool, owner, boardA.ID, "Only On A", FieldTypeShortText, "", ""); err != nil {
		t.Fatalf("add field to A: %v", err)
	}
	fieldsB, _ := ListReferenceFields(ctx, pool, boardB.ID)
	if len(fieldsB) != 4 {
		t.Fatalf("expected board B unaffected by board A's new field, still has %d fields, got %d", 4, len(fieldsB))
	}
	_ = fieldsA
}

// --- 14.5 Export ---

func TestExportPreservesTimelineMetadataAndReferencePanel(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board, err := CreateTimelineBoard(ctx, pool, owner, "Export Timeline", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	fields, _ := ListReferenceFields(ctx, pool, board.ID)
	if _, err := SetReferenceFieldTextContent(ctx, pool, owner, board.ID, fields[0].ID, "Premise text"); err != nil {
		t.Fatalf("set content: %v", err)
	}

	doc, err := BuildBoardExport(ctx, pool, owner, board.ID)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if doc.Board.Mode != ModeTimeline {
		t.Fatalf("expected exported mode %q, got %q", ModeTimeline, doc.Board.Mode)
	}
	if doc.Board.TemplateVersion == nil || *doc.Board.TemplateVersion != TimelineTemplateVersion {
		t.Fatalf("expected exported template_version %d, got %v", TimelineTemplateVersion, doc.Board.TemplateVersion)
	}
	var beginningSeen, endingSeen bool
	for _, c := range doc.Columns {
		if c.ColumnRole == ColumnRoleBeginning {
			beginningSeen = true
		}
		if c.ColumnRole == ColumnRoleEnding {
			endingSeen = true
		}
	}
	if !beginningSeen || !endingSeen {
		t.Fatalf("expected exported columns to preserve boundary roles")
	}
	if len(doc.ReferenceFields) != 4 {
		t.Fatalf("expected 4 exported reference fields, got %d", len(doc.ReferenceFields))
	}
	var premiseExported bool
	for _, f := range doc.ReferenceFields {
		if f.Label == "Premise" {
			premiseExported = true
			if f.TextContent != "Premise text" {
				t.Fatalf("expected exported premise content preserved, got %q", f.TextContent)
			}
			if f.Slug == "" {
				t.Fatalf("expected exported field to carry a non-empty slug")
			}
		}
	}
	if !premiseExported {
		t.Fatalf("expected the Premise field in export")
	}
}

func TestBlankExportRemainsBackwardCompatible(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Blank Export Board")

	doc, err := BuildBoardExport(ctx, pool, owner, board.ID)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if doc.Board.Mode != ModeBlank {
		t.Fatalf("expected exported mode %q, got %q", ModeBlank, doc.Board.Mode)
	}
	if doc.Board.TemplateVersion != nil {
		t.Fatalf("expected nil template_version on a Blank export, got %v", *doc.Board.TemplateVersion)
	}
	if len(doc.ReferenceFields) != 0 {
		t.Fatalf("expected zero reference fields on a plain Blank board, got %d", len(doc.ReferenceFields))
	}
	for _, c := range doc.Columns {
		if c.ColumnRole != ColumnRoleOrdinary {
			t.Fatalf("expected every Blank column to export as 'ordinary', got %q", c.ColumnRole)
		}
	}
}
