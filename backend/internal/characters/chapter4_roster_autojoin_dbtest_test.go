package characters

// Kernel: Catharsis roster auto-join regression test. Grant reported that
// after finishing character creation, Kessa's shop refused him even though
// the character existed and was active. Root cause, confirmed against the
// live database: CommitChapter4FirstSkill's roster auto-enrollment (below)
// checked show_runs.status = 'active', but a Show Run's own status is a
// separately-toggled Director field that can sit at 'planning' indefinitely
// while the Show underneath it is genuinely status='live' with an active
// session -- exactly what had happened live ("Opening Socio" show_run
// stayed 'planning' the whole time its Show was 'live'). The INSERT
// silently matched zero rows (no error), leaving the new character with no
// show_run_roster_members row at all, which is what Kessa's shop
// (merchant.ResolveShowParticipation -> showruns.LoadMyRosterMember)
// actually checks.
//
// The fix keys off shows.status = 'live' instead. These tests build a
// minimal fixture (fresh location/lot/venue slug 'catharsis', a Show Run
// left at its default 'planning' status, a Show explicitly 'live') and
// prove the roster row now gets created -- and that a non-live Show still
// correctly results in no roster row (the gate isn't "always enroll").

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
)

func chapter4RosterTestSuffix(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000")
}

type chapter4RosterFixture struct {
	locationID string
	showRunID  string
	actorID    string
	cardID     string
}

// buildChapter4RosterFixture creates a fresh throwaway location/lot/venue
// (slug 'catharsis', scoped to this new location -- venues.slug is only
// UNIQUE per lot_id, so this never collides with the real seeded Catharsis
// venue) with a Show Run and a Show whose status is the caller's choice,
// plus a character card already past chapter2/chapter3 so
// CommitChapter4FirstSkill only needs to resolve chapter4.
func buildChapter4RosterFixture(t *testing.T, pool *pgxpool.Pool, showStatus string) chapter4RosterFixture {
	t.Helper()
	ctx := context.Background()
	suffix := chapter4RosterTestSuffix(t)

	var fx chapter4RosterFixture
	var locationID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO locations (slug, name) VALUES ($1, 'Chapter4 Roster Test Location') RETURNING id::text
	`, "ch4-roster-loc-"+suffix).Scan(&locationID); err != nil {
		t.Fatalf("fixture location: %v", err)
	}
	fx.locationID = locationID
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM locations WHERE id = $1`, locationID)
	})

	var lotID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO lots (location_id, name, slug) VALUES ($1::uuid, 'Main Lot', 'main-lot') RETURNING id::text
	`, locationID).Scan(&lotID); err != nil {
		t.Fatalf("fixture lot: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO venues (lot_id, name, slug, kind, is_public)
		VALUES ($1::uuid, 'Catharsis', 'catharsis', 'plaza', TRUE)
	`, lotID); err != nil {
		t.Fatalf("fixture venue: %v", err)
	}

	var productionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug) VALUES ($1::uuid, 'Fixture Production', 'fixture-production') RETURNING id::text
	`, locationID).Scan(&productionID); err != nil {
		t.Fatalf("fixture production: %v", err)
	}

	director := chapter4RosterTestUser(t, pool, "ch4_director")

	var showRunID string
	// status left at its column default ('planning') deliberately -- the
	// whole point of this fixture is proving the fix no longer depends on
	// show_runs.status.
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_runs (location_id, production_id, title, slug, created_by_user_id)
		VALUES ($1::uuid, $2::uuid, 'Fixture Run', 'fixture-run', $3::uuid)
		RETURNING id::text
	`, locationID, productionID, director).Scan(&showRunID); err != nil {
		t.Fatalf("fixture show_run: %v", err)
	}
	fx.showRunID = showRunID

	if _, err := pool.Exec(ctx, `
		INSERT INTO shows (show_run_id, slug, title, status, created_by_user_id)
		VALUES ($1::uuid, 'fixture-show', 'Fixture Show', $2, $3::uuid)
	`, showRunID, showStatus, director); err != nil {
		t.Fatalf("fixture show: %v", err)
	}

	actor := chapter4RosterTestUser(t, pool, "ch4_actor")
	fx.actorID = actor
	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1::uuid, $2::uuid, 'cast', TRUE)
	`, locationID, actor); err != nil {
		t.Fatalf("fixture actor membership: %v", err)
	}

	var cardID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO character_cards (owner_user_id, location_id, name) VALUES ($1::uuid, $2::uuid, 'Fixture Character') RETURNING id::text
	`, actor, locationID).Scan(&cardID); err != nil {
		t.Fatalf("fixture character card: %v", err)
	}
	fx.cardID = cardID
	if _, err := pool.Exec(ctx, `
		UPDATE character_cards
		SET workbook_context = '{"chapter2": {"stages": {"10": {"completed": true}}, "attributes": {"Awareness": 5}}, "chapter3": {"confirmed": true, "archetype_key": "custom", "primary_attribute": "Awareness"}}'::jsonb
		WHERE id = $1
	`, cardID); err != nil {
		t.Fatalf("fixture workbook context: %v", err)
	}

	return fx
}

func chapter4RosterTestUser(t *testing.T, pool *pgxpool.Pool, prefix string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO users (handle, display_name) VALUES ($1, $1) RETURNING id::text
	`, prefix+"_"+chapter4RosterTestSuffix(t)).Scan(&id); err != nil {
		t.Fatalf("fixture user %s: %v", prefix, err)
	}
	return id
}

func rosterCharacterCardID(t *testing.T, pool *pgxpool.Pool, showRunID, userID string) (string, bool) {
	t.Helper()
	var cardID string
	err := pool.QueryRow(context.Background(), `
		SELECT COALESCE(character_card_id::text, '')
		FROM show_run_roster_members
		WHERE show_run_id = $1 AND user_id = $2 AND removed_at IS NULL
	`, showRunID, userID).Scan(&cardID)
	if err != nil {
		return "", false
	}
	return cardID, true
}

func TestCommitChapter4FirstSkillAutoJoinsRosterWhenShowIsLive(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	fx := buildChapter4RosterFixture(t, pool, "live")

	if _, _, err := CommitChapter4FirstSkill(context.Background(), pool, fx.actorID, fx.cardID, "SKILL_AWARENESS_ALERTNESS", nil, false); err != nil {
		t.Fatalf("CommitChapter4FirstSkill: %v", err)
	}

	cardID, ok := rosterCharacterCardID(t, pool, fx.showRunID, fx.actorID)
	if !ok {
		t.Fatal("expected a show_run_roster_members row for the actor after completing chapter 4, found none")
	}
	if cardID != fx.cardID {
		t.Fatalf("roster character_card_id = %q, want %q", cardID, fx.cardID)
	}
}

func TestCommitChapter4FirstSkillDoesNotRosterWhenShowIsNotLive(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	fx := buildChapter4RosterFixture(t, pool, "draft")

	if _, _, err := CommitChapter4FirstSkill(context.Background(), pool, fx.actorID, fx.cardID, "SKILL_AWARENESS_ALERTNESS", nil, false); err != nil {
		t.Fatalf("CommitChapter4FirstSkill: %v", err)
	}

	if _, ok := rosterCharacterCardID(t, pool, fx.showRunID, fx.actorID); ok {
		t.Fatal("expected no show_run_roster_members row when the Show is not live -- the gate should not always-enroll")
	}
}
