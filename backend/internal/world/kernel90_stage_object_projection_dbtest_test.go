package world

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/stageobjects"
)

// This is the Kernel 90 §36 information-leakage proof, at the only layer where
// it can actually be made: LoadVenueSnapshot, the payload a client receives.
//
// §36 and §54 both turn on the difference between "the client hides it" and
// "the client never had it". Everything in stageobjects proves the DECISION is
// right; only this proves the decision is enforced by OMISSION from the wire
// payload. If this test passed while the object were present-but-flagged, the
// kernel would be a FAIL under §54 no matter how correct the projector is.

// placeStageElement places the fixture's element on the venue's stage surface,
// which is what makes it a durable stage object rather than venue chrome.
func placeStageElement(t *testing.T, pool *pgxpool.Pool, f worldFixture) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO venue_layout_elements (venue_id, element_id, surface, position, visibility, is_default)
		VALUES ($1::uuid, $2::uuid, 'stage', '{"anchor":"stage","x":10,"y":10}'::jsonb,
		        '{"toRoles":["audience","cast","crew","director","producer"],"privateTo":[],"visible":true}'::jsonb, FALSE)
		ON CONFLICT (venue_id, element_id, surface) DO NOTHING
	`, f.venueID, f.elementID); err != nil {
		t.Fatalf("place stage element: %v", err)
	}
}

func elementInSnapshot(snap *Snapshot, elementID string) (PlacedElement, bool) {
	for _, el := range snap.Elements {
		if el.ElementID == elementID {
			return el, true
		}
	}
	return PlacedElement{}, false
}

func TestSnapshotOmitsHiddenObjectEntirelyForOrdinaryViewers(t *testing.T) {
	pool := openWorldTestPool(t)
	ctx := context.Background()
	director := insertWorldTestUser(t, pool, "k90p_director")
	player := insertWorldTestUser(t, pool, "k90p_player")
	audience := insertWorldTestUser(t, pool, "k90p_audience")

	f := buildWorldFixture(t, pool, director)
	placeStageElement(t, pool, f)
	sessionID := insertSession(t, pool, f.venueID, f.showID, "live")
	for userID, role := range map[string]string{director: "director", player: "cast", audience: "audience"} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO session_participants (session_id, user_id, role)
			VALUES ($1::uuid, $2::uuid, $3::location_role)
			ON CONFLICT (session_id, user_id) DO UPDATE SET role = EXCLUDED.role
		`, sessionID, userID, role); err != nil {
			t.Fatalf("fixture participant %s: %v", role, err)
		}
	}

	ref := stageobjects.Ref{Kind: stageobjects.KindVenueLayoutElement, ID: f.elementID}

	// Baseline: with no canonical state, everyone sees it (§10).
	for _, v := range []struct{ role, user string }{
		{"director", director}, {"cast", player}, {"audience", audience},
	} {
		snap, err := LoadVenueSnapshot(ctx, pool, v.role, v.user, f.venueSlug)
		if err != nil {
			t.Fatalf("baseline snapshot (%s): %v", v.role, err)
		}
		if _, ok := elementInSnapshot(snap, f.elementID); !ok {
			t.Fatalf("%s did not receive a visible object with no state row", v.role)
		}
	}

	if _, err := stageobjects.ApplyMutation(ctx, pool, stageobjects.Mutation{
		ShowID: f.showID, Ref: ref, Op: stageobjects.OpHide, ActorID: director,
	}); err != nil {
		t.Fatalf("hide: %v", err)
	}

	// The Player's payload must not contain the object AT ALL. Asserting on
	// absence from Elements is the whole point: a present-but-flagged element
	// would satisfy a naive "the Player cannot see it" test while shipping the
	// hidden object's label, position, and data to a client that can read them.
	for _, v := range []struct{ role, user string }{{"cast", player}, {"audience", audience}} {
		snap, err := LoadVenueSnapshot(ctx, pool, v.role, v.user, f.venueSlug)
		if err != nil {
			t.Fatalf("hidden snapshot (%s): %v", v.role, err)
		}
		if el, ok := elementInSnapshot(snap, f.elementID); ok {
			t.Errorf("%s received a hidden object: name=%q data=%v", v.role, el.Name, el.Data)
		}
	}

	// §14: the Director keeps it, marked.
	snap, err := LoadVenueSnapshot(ctx, pool, "director", director, f.venueSlug)
	if err != nil {
		t.Fatalf("director snapshot: %v", err)
	}
	el, ok := elementInSnapshot(snap, f.elementID)
	if !ok {
		t.Fatal("the Director lost the hidden object from their working stage")
	}
	if el.State["hidden_backstage_only"] != true {
		t.Errorf("the Director was not told the object is hidden: state=%v", el.State)
	}
	if el.State["visible"] != false {
		t.Errorf("expected visible=false for a hidden object, got state=%v", el.State)
	}
	// §5/§22: the Director receives canonical identity, which is what the
	// visibility controls target.
	if el.State["stage_object_kind"] != stageobjects.KindVenueLayoutElement {
		t.Errorf("missing canonical object kind: state=%v", el.State)
	}
	if el.State["stage_object_id"] != f.elementID {
		t.Errorf("missing canonical object id: state=%v", el.State)
	}

	// Reveal restores it for everyone, from the same reference.
	if _, err := stageobjects.ApplyMutation(ctx, pool, stageobjects.Mutation{
		ShowID: f.showID, Ref: ref, Op: stageobjects.OpReveal, ActorID: director,
	}); err != nil {
		t.Fatalf("reveal: %v", err)
	}
	snap, err = LoadVenueSnapshot(ctx, pool, "cast", player, f.venueSlug)
	if err != nil {
		t.Fatalf("post-reveal snapshot: %v", err)
	}
	if _, ok := elementInSnapshot(snap, f.elementID); !ok {
		t.Fatal("revealing did not restore the object for the Player")
	}
}

func TestSnapshotWithholdsScopeMetadataFromPlayers(t *testing.T) {
	// §35: "Cohort A cannot infer Cohort B-only object metadata". A granted
	// Cohort DOES receive the object, so this is the case where withholding the
	// object is not enough -- the grant list names other cohorts and Characters
	// and must not travel with it.
	pool := openWorldTestPool(t)
	ctx := context.Background()
	director := insertWorldTestUser(t, pool, "k90p_scope_director")
	player := insertWorldTestUser(t, pool, "k90p_scope_player")

	f := buildWorldFixture(t, pool, director)
	placeStageElement(t, pool, f)
	sessionID := insertSession(t, pool, f.venueID, f.showID, "live")

	var cohortID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_cohorts (show_id, serial_number, slug, name, created_by_user_id)
		VALUES ($1::uuid, 1, 'k90p-cohort-'||$2, 'Cohort A', $3::uuid)
		RETURNING id::text
	`, f.showID, testSuffix(t), director).Scan(&cohortID); err != nil {
		t.Fatalf("insert cohort: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO show_cohort_assignments (show_id, user_id, cohort_id, assigned_by_user_id)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid)
	`, f.showID, player, cohortID, director); err != nil {
		t.Fatalf("assign cohort: %v", err)
	}
	for userID, role := range map[string]string{director: "director", player: "cast"} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO session_participants (session_id, user_id, role)
			VALUES ($1::uuid, $2::uuid, $3::location_role)
			ON CONFLICT (session_id, user_id) DO UPDATE SET role = EXCLUDED.role
		`, sessionID, userID, role); err != nil {
			t.Fatalf("fixture participant %s: %v", role, err)
		}
	}

	ref := stageobjects.Ref{Kind: stageobjects.KindVenueLayoutElement, ID: f.elementID}
	if _, err := stageobjects.ApplyMutation(ctx, pool, stageobjects.Mutation{
		ShowID: f.showID, Ref: ref, Op: stageobjects.OpHide, ActorID: director,
	}.WithScopes([]stageobjects.Scope{{Kind: stageobjects.ScopeCohort, ID: cohortID}})); err != nil {
		t.Fatalf("hide with cohort grant: %v", err)
	}

	// §16: the granted Cohort sees it, with one Scene and no fork.
	snap, err := LoadVenueSnapshot(ctx, pool, "cast", player, f.venueSlug)
	if err != nil {
		t.Fatalf("cohort snapshot: %v", err)
	}
	el, ok := elementInSnapshot(snap, f.elementID)
	if !ok {
		t.Fatal("the granted Cohort did not receive the cohort-scoped object")
	}
	if _, leaked := el.State["visibility_scopes"]; leaked {
		t.Errorf("scope metadata leaked to a Player: state=%v", el.State)
	}
	if _, leaked := el.State["hidden_backstage_only"]; leaked {
		t.Errorf("backstage hidden marker leaked to a Player: state=%v", el.State)
	}

	// The Director sees the grant list.
	dsnap, err := LoadVenueSnapshot(ctx, pool, "director", director, f.venueSlug)
	if err != nil {
		t.Fatalf("director snapshot: %v", err)
	}
	del, ok := elementInSnapshot(dsnap, f.elementID)
	if !ok {
		t.Fatal("the Director lost the scoped object")
	}
	if _, present := del.State["visibility_scopes"]; !present {
		t.Errorf("the Director could not read the grant list: state=%v", del.State)
	}
}

func TestSnapshotDoesNotReplayLegacyRevealActions(t *testing.T) {
	// Kernel 90 deleted the act/reveal_element action replay in favour of
	// canonical state, and per Grant's instruction did NOT backfill. So a
	// pre-existing act/hide_element row must have no effect on projection.
	//
	// This test exists to make that consequence explicit and intentional rather
	// than a surprise during Grant's walkthrough: if it ever starts failing,
	// someone has reintroduced a second visibility mechanism.
	pool := openWorldTestPool(t)
	ctx := context.Background()
	director := insertWorldTestUser(t, pool, "k90p_legacy_director")
	player := insertWorldTestUser(t, pool, "k90p_legacy_player")

	f := buildWorldFixture(t, pool, director)
	placeStageElement(t, pool, f)
	sessionID := insertSession(t, pool, f.venueID, f.showID, "live")
	for userID, role := range map[string]string{director: "director", player: "cast"} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO session_participants (session_id, user_id, role)
			VALUES ($1::uuid, $2::uuid, $3::location_role)
			ON CONFLICT (session_id, user_id) DO UPDATE SET role = EXCLUDED.role
		`, sessionID, userID, role); err != nil {
			t.Fatalf("fixture participant %s: %v", role, err)
		}
	}

	// A legacy hide action, exactly as actions.StoreReveal used to write it.
	if _, err := pool.Exec(ctx, `
		INSERT INTO actions (session_id, show_id, moment_id, actor_id, type, target, payload)
		VALUES ($1::uuid, $2::uuid, 1, $3::uuid, 'act/hide_element',
		        jsonb_build_object('element_id', $4::text, 'layer', 'audience'),
		        '{"visible":false}'::jsonb)
	`, sessionID, f.showID, director, f.elementID); err != nil {
		t.Fatalf("insert legacy hide action: %v", err)
	}

	snap, err := LoadVenueSnapshot(ctx, pool, "cast", player, f.venueSlug)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if _, ok := elementInSnapshot(snap, f.elementID); !ok {
		t.Fatal("a legacy act/hide_element row still hid an object -- the old replay is back")
	}

	// And canonical state does hide it, proving the object is hideable and the
	// test above is not passing for the wrong reason.
	if _, err := stageobjects.ApplyMutation(ctx, pool, stageobjects.Mutation{
		ShowID:  f.showID,
		Ref:     stageobjects.Ref{Kind: stageobjects.KindVenueLayoutElement, ID: f.elementID},
		Op:      stageobjects.OpHide,
		ActorID: director,
	}); err != nil {
		t.Fatalf("canonical hide: %v", err)
	}
	snap, err = LoadVenueSnapshot(ctx, pool, "cast", player, f.venueSlug)
	if err != nil {
		t.Fatalf("snapshot after canonical hide: %v", err)
	}
	if _, ok := elementInSnapshot(snap, f.elementID); ok {
		t.Fatal("canonical state failed to hide the object")
	}
}
