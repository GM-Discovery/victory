package drawing

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/venuecoordination"
)

// fixture mirrors rollaudience_dbtest_test.go's fixture shape exactly (same
// minimal Session -> Show -> Cohort graph via raw SQL) plus a
// show_run_roster_members row so authority/scope tests exercise the same
// Show/Cohort resolution IC chat also depends on.
type fixture struct {
	sessionID    string
	showID       string
	showRunID    string
	directorID   string
	cohortAID    string
	cohortBID    string
	cohortAUser1 string
	cohortAUser2 string
	cohortBUser1 string
	ungroupedID  string
	audienceID   string
}

func buildFixture(ctx context.Context, t *testing.T, pool *pgxpool.Pool) fixture {
	t.Helper()
	suffix := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "_")) + "_" + time.Now().UTC().Format("150405.000000000")

	var locationID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&locationID); err != nil {
		t.Fatalf("fixture location: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO productions (location_id, name, slug)
		SELECT $1::uuid, 'Fixture Production', 'fixture-production'
		WHERE NOT EXISTS (SELECT 1 FROM productions WHERE location_id = $1::uuid)
	`, locationID); err != nil {
		t.Fatalf("fixture production: %v", err)
	}
	var productionID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM productions WHERE location_id = $1::uuid LIMIT 1`, locationID).Scan(&productionID); err != nil {
		t.Fatalf("load fixture production: %v", err)
	}

	var fx fixture
	mkUser := func(prefix string) string {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO users (handle, display_name) VALUES ($1, $2) RETURNING id::text
		`, prefix+"_"+suffix, prefix).Scan(&id); err != nil {
			t.Fatalf("fixture user %s: %v", prefix, err)
		}
		t.Cleanup(func() {
			bg := context.Background()
			_, _ = pool.Exec(bg, `DELETE FROM drawing_objects WHERE creator_user_id = $1`, id)
			_, _ = pool.Exec(bg, `DELETE FROM show_cohort_assignments WHERE user_id = $1 OR assigned_by_user_id = $1`, id)
			_, _ = pool.Exec(bg, `DELETE FROM show_cohorts WHERE created_by_user_id = $1`, id)
			_, _ = pool.Exec(bg, `DELETE FROM show_run_roster_members WHERE user_id = $1 OR added_by_user_id = $1`, id)
			_, _ = pool.Exec(bg, `DELETE FROM session_participants WHERE user_id = $1`, id)
			_, _ = pool.Exec(bg, `UPDATE sessions SET show_id = NULL WHERE show_id IN (SELECT id FROM shows WHERE created_by_user_id = $1)`, id)
			_, _ = pool.Exec(bg, `DELETE FROM stage_drawing_settings WHERE show_id IN (SELECT id FROM shows WHERE created_by_user_id = $1)`, id)
			_, _ = pool.Exec(bg, `DELETE FROM shows WHERE created_by_user_id = $1`, id)
			_, _ = pool.Exec(bg, `DELETE FROM show_runs WHERE created_by_user_id = $1`, id)
			_, _ = pool.Exec(bg, `DELETE FROM character_cards WHERE owner_user_id = $1`, id)
			_, _ = pool.Exec(bg, `DELETE FROM users WHERE id = $1`, id)
		})
		return id
	}
	fx.directorID = mkUser("dw_dir")
	fx.cohortAUser1 = mkUser("dw_coa1")
	fx.cohortAUser2 = mkUser("dw_coa2")
	fx.cohortBUser1 = mkUser("dw_cob1")
	fx.ungroupedID = mkUser("dw_ungr")
	fx.audienceID = mkUser("dw_aud")

	if err := pool.QueryRow(ctx, `
		INSERT INTO show_runs (location_id, production_id, title, slug, created_by_user_id)
		VALUES ($1::uuid, $2::uuid, 'Fixture Show Run', 'fixture-show-run-'||$3, $4::uuid)
		RETURNING id::text
	`, locationID, productionID, suffix, fx.directorID).Scan(&fx.showRunID); err != nil {
		t.Fatalf("fixture show run: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM show_runs WHERE id = $1`, fx.showRunID) })

	if err := pool.QueryRow(ctx, `
		INSERT INTO shows (show_run_id, slug, title, created_by_user_id)
		VALUES ($1::uuid, 'fixture-show-'||$2, 'Fixture Show', $3::uuid)
		RETURNING id::text
	`, fx.showRunID, suffix, fx.directorID).Scan(&fx.showID); err != nil {
		t.Fatalf("fixture show: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM shows WHERE id = $1`, fx.showID) })

	if err := pool.QueryRow(ctx, `
		INSERT INTO sessions (venue_id, status, show_id)
		SELECT v.id, 'rehearsal', $1::uuid FROM venues v WHERE v.slug = 'catharsis'
		RETURNING id::text
	`, fx.showID).Scan(&fx.sessionID); err != nil {
		t.Fatalf("fixture session: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `UPDATE sessions SET status = 'closed' WHERE id = $1::uuid`, fx.sessionID)
	})

	roles := map[string]string{
		fx.directorID:   "director",
		fx.cohortAUser1: "cast",
		fx.cohortAUser2: "cast",
		fx.cohortBUser1: "cast",
		fx.ungroupedID:  "cast",
		fx.audienceID:   "audience",
	}
	for userID, role := range roles {
		if _, err := pool.Exec(ctx, `
			INSERT INTO session_participants (session_id, user_id, role)
			VALUES ($1::uuid, $2::uuid, $3::location_role)
			ON CONFLICT (session_id, user_id) DO UPDATE SET role = EXCLUDED.role
		`, fx.sessionID, userID, role); err != nil {
			t.Fatalf("fixture participant %s: %v", role, err)
		}
	}
	// Roster rows (Kernel 71 character-selection surface) for every cast
	// participant -- role must match show_run_roster_members' own CHECK
	// constraint (no "cast"; Players are "player").
	for _, userID := range []string{fx.cohortAUser1, fx.cohortAUser2, fx.cohortBUser1, fx.ungroupedID} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO show_run_roster_members (show_run_id, user_id, role, added_by_user_id)
			VALUES ($1::uuid, $2::uuid, 'player', $3::uuid)
		`, fx.showRunID, userID, fx.directorID); err != nil {
			t.Fatalf("fixture roster row: %v", err)
		}
	}

	if err := pool.QueryRow(ctx, `
		INSERT INTO show_cohorts (show_id, serial_number, slug, name, created_by_user_id)
		VALUES ($1::uuid, 1, 'cohort-a-'||$2, 'Cohort A', $3::uuid)
		RETURNING id::text
	`, fx.showID, suffix, fx.directorID).Scan(&fx.cohortAID); err != nil {
		t.Fatalf("fixture cohort A: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_cohorts (show_id, serial_number, slug, name, created_by_user_id)
		VALUES ($1::uuid, 2, 'cohort-b-'||$2, 'Cohort B', $3::uuid)
		RETURNING id::text
	`, fx.showID, suffix, fx.directorID).Scan(&fx.cohortBID); err != nil {
		t.Fatalf("fixture cohort B: %v", err)
	}

	assign := func(userID, cohortID string) {
		if _, err := pool.Exec(ctx, `
			INSERT INTO show_cohort_assignments (show_id, user_id, cohort_id, assigned_by_user_id)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid)
		`, fx.showID, userID, cohortID, fx.directorID); err != nil {
			t.Fatalf("fixture cohort assignment: %v", err)
		}
	}
	assign(fx.cohortAUser1, fx.cohortAID)
	assign(fx.cohortAUser2, fx.cohortAID)
	assign(fx.cohortBUser1, fx.cohortBID)

	return fx
}

func rectRequest() CreateRequest {
	return CreateRequest{
		ObjectType:  TypeRectangle,
		Geometry:    map[string]any{"x": 10.0, "y": 10.0, "width": 50.0, "height": 40.0},
		StrokeColor: "#112233",
	}
}

// --- Authority --------------------------------------------------------

func TestCanDraw_DirectorAlwaysAllowed(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := buildFixture(ctx, t, pool)

	allowed, err := CanDraw(ctx, pool, nil, fx.sessionID, fx.showID, fx.directorID)
	if err != nil {
		t.Fatalf("CanDraw: %v", err)
	}
	if !allowed {
		t.Fatal("expected Director to always be allowed to draw")
	}
}

func TestCanDraw_DirectorOnlyModeDeniesPlayer(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := buildFixture(ctx, t, pool)
	// Default mode with no settings row is director_only.

	allowed, err := CanDraw(ctx, pool, nil, fx.sessionID, fx.showID, fx.cohortAUser1)
	if err != nil {
		t.Fatalf("CanDraw: %v", err)
	}
	if allowed {
		t.Fatal("expected ordinary Player to be denied in director_only mode")
	}
}

func TestCanDraw_TurnLeaderMode(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := buildFixture(ctx, t, pool)

	if _, err := SaveSettings(ctx, pool, fx.directorID, fx.sessionID, fx.showID, Settings{
		DrawingMode: ModeTurnLeader, ScaleGridUnits: 1, ScaleRealUnits: 5, ScaleUnitLabel: "ft", DiagonalPolicy: DiagonalAlternating,
	}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	reg := venuecoordination.NewRegistry()

	// Nobody is Group Leader/Current Turn yet -- cohortAUser1 still denied.
	allowed, err := CanDraw(ctx, pool, reg, fx.sessionID, fx.showID, fx.cohortAUser1)
	if err != nil {
		t.Fatalf("CanDraw: %v", err)
	}
	if allowed {
		t.Fatal("expected non-leader Player to be denied before any assignment")
	}

	if _, err := AssignCurrentTurn(ctx, pool, reg, fx.sessionID, fx.showID, fx.directorID, fx.cohortAUser1); err != nil {
		t.Fatalf("AssignCurrentTurn: %v", err)
	}

	allowed, err = CanDraw(ctx, pool, reg, fx.sessionID, fx.showID, fx.cohortAUser1)
	if err != nil {
		t.Fatalf("CanDraw: %v", err)
	}
	if !allowed {
		t.Fatal("expected Current Turn holder to be allowed to draw in turn_leader mode")
	}

	// A different Player, not holding Current Turn/Group Leader, is still denied.
	allowed, err = CanDraw(ctx, pool, reg, fx.sessionID, fx.showID, fx.cohortAUser2)
	if err != nil {
		t.Fatalf("CanDraw: %v", err)
	}
	if allowed {
		t.Fatal("expected a non-holder Player to remain denied")
	}
}

func TestCanDraw_FreeformAllowsEligibleParticipant(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := buildFixture(ctx, t, pool)

	if _, err := SaveSettings(ctx, pool, fx.directorID, fx.sessionID, fx.showID, Settings{
		DrawingMode: ModeFreeform, ScaleGridUnits: 1, ScaleRealUnits: 5, ScaleUnitLabel: "ft", DiagonalPolicy: DiagonalAlternating,
	}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	allowed, err := CanDraw(ctx, pool, nil, fx.sessionID, fx.showID, fx.cohortAUser1)
	if err != nil {
		t.Fatalf("CanDraw: %v", err)
	}
	if !allowed {
		t.Fatal("expected eligible Player to be allowed in freeform mode")
	}

	allowed, err = CanDraw(ctx, pool, nil, fx.sessionID, fx.showID, fx.audienceID)
	if err != nil {
		t.Fatalf("CanDraw: %v", err)
	}
	if allowed {
		t.Fatal("expected Audience to remain ineligible even in freeform mode")
	}
}

func TestCreate_UnauthorizedInDirectorOnlyMode(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := buildFixture(ctx, t, pool)

	req := rectRequest()
	req.SessionID = fx.sessionID
	req.Scope = "show"
	_, err := Create(ctx, pool, nil, fx.cohortAUser1, req)
	if err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized, got %v", err)
	}
}

func TestEditOwnership_PlayerCannotEditAnothersObject(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := buildFixture(ctx, t, pool)

	req := rectRequest()
	req.SessionID = fx.sessionID
	req.Scope = "show"
	obj, err := Create(ctx, pool, nil, fx.directorID, req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	newColor := "#ff0000"
	_, err = Update(ctx, pool, fx.cohortAUser1, obj.ID, UpdateRequest{SessionID: fx.sessionID, StrokeColor: &newColor})
	if err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for another Player editing Director's object, got %v", err)
	}

	if err := Delete(ctx, pool, fx.cohortAUser1, fx.sessionID, obj.ID); err == nil {
		t.Fatal("expected not_authorized deleting another Player's object")
	}
}

func TestEditOwnership_CreatorCanEditOwnObject(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := buildFixture(ctx, t, pool)

	if _, err := SaveSettings(ctx, pool, fx.directorID, fx.sessionID, fx.showID, Settings{
		DrawingMode: ModeFreeform, ScaleGridUnits: 1, ScaleRealUnits: 5, ScaleUnitLabel: "ft", DiagonalPolicy: DiagonalAlternating,
	}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	req := rectRequest()
	req.SessionID = fx.sessionID
	req.Scope = "show"
	obj, err := Create(ctx, pool, nil, fx.cohortAUser1, req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	newColor := "#00ff00"
	updated, err := Update(ctx, pool, fx.cohortAUser1, obj.ID, UpdateRequest{SessionID: fx.sessionID, StrokeColor: &newColor})
	if err != nil {
		t.Fatalf("Update by creator: %v", err)
	}
	if updated.StrokeColor != newColor {
		t.Fatalf("stroke color = %q, want %q", updated.StrokeColor, newColor)
	}
}

func TestDirectorCanCorrectAnyonesObject(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := buildFixture(ctx, t, pool)

	if _, err := SaveSettings(ctx, pool, fx.directorID, fx.sessionID, fx.showID, Settings{
		DrawingMode: ModeFreeform, ScaleGridUnits: 1, ScaleRealUnits: 5, ScaleUnitLabel: "ft", DiagonalPolicy: DiagonalAlternating,
	}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	req := rectRequest()
	req.SessionID = fx.sessionID
	req.Scope = "show"
	obj, err := Create(ctx, pool, nil, fx.cohortAUser1, req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := Delete(ctx, pool, fx.directorID, fx.sessionID, obj.ID); err != nil {
		t.Fatalf("Director delete of a Player's object: %v", err)
	}
	objs, err := List(ctx, pool, fx.sessionID, fx.directorID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, o := range objs {
		if o.ID == obj.ID {
			t.Fatal("expected object deleted by Director to be gone")
		}
	}
}

// --- Scope --------------------------------------------------------------

func TestScope_CohortDrawingDoesNotLeakToOtherCohort(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := buildFixture(ctx, t, pool)

	if _, err := SaveSettings(ctx, pool, fx.directorID, fx.sessionID, fx.showID, Settings{
		DrawingMode: ModeFreeform, ScaleGridUnits: 1, ScaleRealUnits: 5, ScaleUnitLabel: "ft", DiagonalPolicy: DiagonalAlternating,
	}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	req := rectRequest()
	req.SessionID = fx.sessionID
	req.Scope = "cohort"
	obj, err := Create(ctx, pool, nil, fx.cohortAUser1, req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if obj.CohortID != fx.cohortAID {
		t.Fatalf("expected object scoped to cohort A, got %q", obj.CohortID)
	}

	// Cohort B member must not see it.
	objsB, err := List(ctx, pool, fx.sessionID, fx.cohortBUser1)
	if err != nil {
		t.Fatalf("List as cohort B: %v", err)
	}
	for _, o := range objsB {
		if o.ID == obj.ID {
			t.Fatal("Cohort A drawing leaked to Cohort B viewer")
		}
	}

	// Cohort A member sees it.
	objsA, err := List(ctx, pool, fx.sessionID, fx.cohortAUser2)
	if err != nil {
		t.Fatalf("List as cohort A: %v", err)
	}
	found := false
	for _, o := range objsA {
		if o.ID == obj.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("expected Cohort A's own member to see the Cohort A drawing")
	}

	// Director sees everything.
	objsDir, err := List(ctx, pool, fx.sessionID, fx.directorID)
	if err != nil {
		t.Fatalf("List as director: %v", err)
	}
	found = false
	for _, o := range objsDir {
		if o.ID == obj.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("expected Director+ to see Cohort-scoped drawings")
	}
}

func TestScope_ShowDrawingReachesEveryone(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := buildFixture(ctx, t, pool)

	req := rectRequest()
	req.SessionID = fx.sessionID
	req.Scope = "show"
	obj, err := Create(ctx, pool, nil, fx.directorID, req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	for _, viewer := range []string{fx.cohortAUser1, fx.cohortBUser1, fx.ungroupedID} {
		objs, err := List(ctx, pool, fx.sessionID, viewer)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		found := false
		for _, o := range objs {
			if o.ID == obj.ID {
				found = true
			}
		}
		if !found {
			t.Fatalf("Show-scoped drawing did not reach viewer %s", viewer)
		}
	}
}

func TestScope_ReconnectDoesNotLeakCohort(t *testing.T) {
	// "Reconnect" is simulated exactly as the real client does it: a
	// second independent List() call (the same GET the frontend issues
	// after a WS reconnect) -- there is no separate reconnect code path
	// to diverge from the live one.
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := buildFixture(ctx, t, pool)

	if _, err := SaveSettings(ctx, pool, fx.directorID, fx.sessionID, fx.showID, Settings{
		DrawingMode: ModeFreeform, ScaleGridUnits: 1, ScaleRealUnits: 5, ScaleUnitLabel: "ft", DiagonalPolicy: DiagonalAlternating,
	}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	req := rectRequest()
	req.SessionID = fx.sessionID
	req.Scope = "cohort"
	obj, err := Create(ctx, pool, nil, fx.cohortAUser1, req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	for i := 0; i < 2; i++ {
		objs, err := List(ctx, pool, fx.sessionID, fx.cohortBUser1)
		if err != nil {
			t.Fatalf("List (reconnect simulation %d): %v", i, err)
		}
		for _, o := range objs {
			if o.ID == obj.ID {
				t.Fatalf("Cohort leaked on simulated reconnect #%d", i)
			}
		}
	}
}

// --- Persistence ----------------------------------------------------------

func TestPersistence_CreateEditDeleteZOrderLock(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := buildFixture(ctx, t, pool)

	req1 := rectRequest()
	req1.SessionID = fx.sessionID
	req1.Scope = "show"
	obj1, err := Create(ctx, pool, nil, fx.directorID, req1)
	if err != nil {
		t.Fatalf("Create 1: %v", err)
	}
	req2 := rectRequest()
	req2.SessionID = fx.sessionID
	req2.Scope = "show"
	obj2, err := Create(ctx, pool, nil, fx.directorID, req2)
	if err != nil {
		t.Fatalf("Create 2: %v", err)
	}
	if obj2.ZOrder <= obj1.ZOrder {
		t.Fatalf("expected obj2 z-order > obj1: %d vs %d", obj2.ZOrder, obj1.ZOrder)
	}

	// Edit persists.
	newWidth := 9.0
	updated, err := Update(ctx, pool, fx.directorID, obj1.ID, UpdateRequest{SessionID: fx.sessionID, StrokeWidth: &newWidth})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.StrokeWidth != newWidth {
		t.Fatalf("stroke width = %v, want %v", updated.StrokeWidth, newWidth)
	}
	reloaded, err := loadObject(ctx, pool, obj1.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.StrokeWidth != newWidth {
		t.Fatal("edit did not persist across reload")
	}

	// Lock persists.
	locked, err := SetLock(ctx, pool, fx.directorID, fx.sessionID, obj1.ID, true)
	if err != nil {
		t.Fatalf("SetLock: %v", err)
	}
	if !locked.Locked {
		t.Fatal("expected locked = true")
	}
	reloaded, _ = loadObject(ctx, pool, obj1.ID)
	if !reloaded.Locked {
		t.Fatal("lock did not persist across reload")
	}

	// z-order "send to back" persists and is reflected in ordering.
	sentBack, err := Reorder(ctx, pool, fx.directorID, fx.sessionID, obj2.ID, ZBack)
	if err != nil {
		t.Fatalf("Reorder: %v", err)
	}
	if sentBack.ZOrder >= reloaded.ZOrder {
		t.Fatalf("expected obj2 sent-to-back z-order below obj1's, got %d vs %d", sentBack.ZOrder, reloaded.ZOrder)
	}

	// Delete persists (soft-delete, excluded from List).
	if err := Delete(ctx, pool, fx.directorID, fx.sessionID, obj1.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := loadObject(ctx, pool, obj1.ID); err == nil {
		t.Fatal("expected deleted object to be excluded from loadObject")
	}
}

// --- Geometry -------------------------------------------------------------

func TestGeometry_FreehandPointsBounded(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := buildFixture(ctx, t, pool)

	pts := make([]any, MaxPointsPerStroke+1)
	for i := range pts {
		pts[i] = map[string]any{"x": float64(i), "y": float64(i)}
	}
	req := CreateRequest{SessionID: fx.sessionID, Scope: "show", ObjectType: TypeFreehand, Geometry: map[string]any{"points": pts}}
	_, err := Create(ctx, pool, nil, fx.directorID, req)
	if err == nil || err.Error() != "too_many_points" {
		t.Fatalf("expected too_many_points, got %v", err)
	}
}

func TestGeometry_ShapeBoundsValidated(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := buildFixture(ctx, t, pool)

	req := CreateRequest{SessionID: fx.sessionID, Scope: "show", ObjectType: TypeRectangle, Geometry: map[string]any{"x": 1.0}}
	_, err := Create(ctx, pool, nil, fx.directorID, req)
	if err == nil || err.Error() != "shape_bounds_required" {
		t.Fatalf("expected shape_bounds_required, got %v", err)
	}
}

func TestGeometry_CoordinatesRemainMapRelativeAcrossReads(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := buildFixture(ctx, t, pool)

	req := rectRequest()
	req.SessionID = fx.sessionID
	req.Scope = "show"
	obj, err := Create(ctx, pool, nil, fx.directorID, req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Geometry is opaque map-relative "world" coordinates -- reading it
	// back (the Detail View / pan-zoom code path) must never mutate
	// storage. Two independent reads must be byte-identical.
	a, err := loadObject(ctx, pool, obj.ID)
	if err != nil {
		t.Fatalf("load 1: %v", err)
	}
	b, err := loadObject(ctx, pool, obj.ID)
	if err != nil {
		t.Fatalf("load 2: %v", err)
	}
	if a.Geometry["x"] != b.Geometry["x"] || a.Geometry["y"] != b.Geometry["y"] {
		t.Fatal("geometry drifted between two reads")
	}
}

// --- Measurement settings --------------------------------------------------

func TestSettings_PersistAndDefault(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := buildFixture(ctx, t, pool)

	defaults, err := LoadSettings(ctx, pool, fx.showID)
	if err != nil {
		t.Fatalf("LoadSettings default: %v", err)
	}
	if defaults.DrawingMode != ModeDirectorOnly || defaults.DiagonalPolicy != DiagonalAlternating {
		t.Fatalf("unexpected defaults: %+v", defaults)
	}

	saved, err := SaveSettings(ctx, pool, fx.directorID, fx.sessionID, fx.showID, Settings{
		DrawingMode: ModeFreeform, ScaleGridUnits: 1, ScaleRealUnits: 2, ScaleUnitLabel: "mi", DiagonalPolicy: DiagonalEuclidean,
	})
	if err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	if saved.ScaleUnitLabel != "mi" {
		t.Fatal("expected saved unit label")
	}

	reloaded, err := LoadSettings(ctx, pool, fx.showID)
	if err != nil {
		t.Fatalf("LoadSettings reload: %v", err)
	}
	if reloaded.DrawingMode != ModeFreeform || reloaded.ScaleRealUnits != 2 || reloaded.DiagonalPolicy != DiagonalEuclidean {
		t.Fatalf("settings did not persist: %+v", reloaded)
	}

	// Non-Director cannot change settings.
	_, err = SaveSettings(ctx, pool, fx.cohortAUser1, fx.sessionID, fx.showID, Settings{
		DrawingMode: ModeDirectorOnly, ScaleGridUnits: 1, ScaleRealUnits: 5, ScaleUnitLabel: "ft", DiagonalPolicy: DiagonalAlternating,
	})
	if err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for non-Director settings change, got %v", err)
	}
}
