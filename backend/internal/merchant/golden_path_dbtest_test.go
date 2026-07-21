package merchant

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/showings"
)

// TestKessaGoldenPath drives the entire Kernel 73 chain against the real
// migrated test database, end to end: a Director places the seeded
// Courtyard Scene into a real Show and attaches the seeded Kessa
// interaction to it, a Player (roster member with a selected, owned
// Character) opens Equip Mode, tries all five stances, previews and
// attempts Haggle, and purchases equipment -- proving it lands in durable
// inventory and a retried purchase does not double it. This is exactly the
// kind of multi-gate chain (venue capability -> roster role -> current
// Scene Placement -> Character ownership -> skill lookup -> canonical
// dice -> purchase idempotency) that stayed silently broken in Kernel 72
// until actions.TestStoreCreateTokenOnCatharsisStage started exercising it
// for real.
func TestKessaGoldenPath(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fx := buildGoldenPathFixture(ctx, t, pool)

	// --- Open Equip Mode --------------------------------------------------
	equip, err := OpenEquipMode(ctx, pool, fx.playerUserID, fx.interactionID)
	if err != nil {
		t.Fatalf("OpenEquipMode: %v", err)
	}
	if equip.Packet.Slug != "kessa" {
		t.Fatalf("expected kessa packet, got %q", equip.Packet.Slug)
	}
	if len(equip.Packet.Stock) == 0 {
		t.Fatal("expected seeded Kessa stock to be non-empty")
	}
	if equip.CharacterCardID != fx.characterCardID {
		t.Fatalf("CharacterCardID = %q, want %q", equip.CharacterCardID, fx.characterCardID)
	}

	// Audience must never be able to open it.
	if _, err := OpenEquipMode(ctx, pool, fx.audienceUserID, fx.interactionID); err == nil {
		t.Fatal("audience should not be able to open Equip Mode")
	}
	// Anonymous (empty actor) must be refused, not silently pass.
	if _, err := OpenEquipMode(ctx, pool, "", fx.interactionID); err == nil {
		t.Fatal("anonymous actor should be refused")
	}

	// --- Stances: all five, disposition matches the authored table --------
	wantDisposition := map[string]string{
		"command":    "reject",
		"convince":   "reject",
		"insight":    "positive",
		"follow":     "neutral",
		"sympathize": "neutral",
	}
	for _, stance := range FixedStanceKeys {
		result, err := AttemptStance(ctx, pool, fx.playerUserID, fx.interactionID, stance)
		if err != nil {
			t.Fatalf("AttemptStance(%s): %v", stance, err)
		}
		if result.Disposition != wantDisposition[stance] {
			t.Fatalf("stance %s disposition = %q, want %q", stance, result.Disposition, wantDisposition[stance])
		}
		if strings.TrimSpace(result.Response) == "" {
			t.Fatalf("stance %s: empty response text", stance)
		}
	}
	if _, err := AttemptStance(ctx, pool, fx.playerUserID, fx.interactionID, "not_a_real_stance"); err == nil {
		t.Fatal("expected invalid_stance rejection")
	}

	// --- Haggle: preview then attempt, TV 5 / d6 skilled / d4 unskilled ---
	preview, err := PreviewHaggle(ctx, pool, fx.playerUserID, fx.interactionID)
	if err != nil {
		t.Fatalf("PreviewHaggle: %v", err)
	}
	if preview.TargetValue != 5 {
		t.Fatalf("target value = %d, want 5", preview.TargetValue)
	}
	if !preview.HasSkill || preview.Die != "d6" {
		t.Fatalf("expected skilled d6, got hasSkill=%v die=%q", preview.HasSkill, preview.Die)
	}
	if preview.ImpossibleToReach {
		t.Fatal("skilled d6 vs TV 5 should not be impossible")
	}

	haggleResult, err := AttemptHaggle(ctx, pool, fx.playerUserID, fx.interactionID)
	if err != nil {
		t.Fatalf("AttemptHaggle: %v", err)
	}
	if haggleResult.Die != "d6" || haggleResult.TargetValue != 5 {
		t.Fatalf("unexpected haggle result shape: %+v", haggleResult)
	}
	if haggleResult.Success && strings.TrimSpace(haggleResult.Text) == "" {
		t.Fatal("success text should not be empty")
	}

	// Unskilled preview: the audience-ineligible checks above already prove
	// role gating; here prove the unskilled-character die/impossible-target
	// math directly against the bounded roll helper.
	unskilledRoll, err := RollSkillGatedDie(ctx, pool, fx.playerUserID, fx.unskilledCharacterCardID, "haggle", "d6", "d4")
	if err != nil {
		t.Fatalf("RollSkillGatedDie (unskilled): %v", err)
	}
	if unskilledRoll.HasSkill {
		t.Fatal("unskilled character should not have the Haggle skill")
	}
	if unskilledRoll.Die != "d4" {
		t.Fatalf("unskilled die = %q, want d4", unskilledRoll.Die)
	}
	if dieMaxFace("d4") >= 5 {
		t.Fatal("d4 max face should be below Target Value 5 (impossible-to-reach warning)")
	}

	// --- Purchase: idempotent, lands in durable inventory ------------------
	// The seeded stock mixes quantity_mode: pick a stackable item
	// (rations-1day) for the increment proof and a unique item (dagger) for
	// the clamp-at-1 proof, matching the Kernel 73 follow-up's curated
	// Kessa catalog (migration 063) rather than assuming Stock[0] is
	// stackable.
	var stockItem, uniqueItem EquipmentItem
	for _, it := range equip.Packet.Stock {
		switch it.Slug {
		case "rations-1day":
			stockItem = it
		case "dagger":
			uniqueItem = it
		}
	}
	if stockItem.ID == "" || uniqueItem.ID == "" {
		t.Fatalf("expected seeded rations-1day (stackable) and dagger (unique) in stock, got %+v", equip.Packet.Stock)
	}

	entry, err := AttemptPurchase(ctx, pool, fx.playerUserID, fx.interactionID, stockItem.ID, "purchase-key-1")
	if err != nil {
		t.Fatalf("AttemptPurchase: %v", err)
	}
	if entry.Quantity != 1 {
		t.Fatalf("first purchase quantity = %d, want 1", entry.Quantity)
	}

	// Retry with the SAME idempotency key must not double the quantity.
	retry, err := AttemptPurchase(ctx, pool, fx.playerUserID, fx.interactionID, stockItem.ID, "purchase-key-1")
	if err != nil {
		t.Fatalf("AttemptPurchase (retry): %v", err)
	}
	if retry.Quantity != 1 {
		t.Fatalf("retried purchase quantity = %d, want 1 (idempotency failed)", retry.Quantity)
	}

	// A genuinely new purchase (new idempotency key) increments a stackable item.
	second, err := AttemptPurchase(ctx, pool, fx.playerUserID, fx.interactionID, stockItem.ID, "purchase-key-2")
	if err != nil {
		t.Fatalf("AttemptPurchase (second): %v", err)
	}
	if second.Quantity != 2 {
		t.Fatalf("second deliberate purchase quantity = %d, want 2", second.Quantity)
	}

	// A unique-mode item clamps at quantity 1 even across two DELIBERATE
	// (different idempotency key) purchases -- a repeat purchase is a
	// no-op, not a second unit (spec S5.2 "non-stackable behavior must be
	// explicit").
	uniqueFirst, err := AttemptPurchase(ctx, pool, fx.playerUserID, fx.interactionID, uniqueItem.ID, "unique-key-1")
	if err != nil {
		t.Fatalf("AttemptPurchase (unique first): %v", err)
	}
	if uniqueFirst.Quantity != 1 {
		t.Fatalf("unique item first purchase quantity = %d, want 1", uniqueFirst.Quantity)
	}
	uniqueSecond, err := AttemptPurchase(ctx, pool, fx.playerUserID, fx.interactionID, uniqueItem.ID, "unique-key-2")
	if err != nil {
		t.Fatalf("AttemptPurchase (unique second): %v", err)
	}
	if uniqueSecond.Quantity != 1 {
		t.Fatalf("unique item second purchase quantity = %d, want 1 (clamp failed)", uniqueSecond.Quantity)
	}

	// Inventory read shows it, persisted, joined with item detail.
	inventory, err := ListInventoryForCharacter(ctx, pool, fx.playerUserID, fx.characterCardID)
	if err != nil {
		t.Fatalf("ListInventoryForCharacter: %v", err)
	}
	found := false
	for _, e := range inventory {
		if e.EquipmentItemID == stockItem.ID {
			found = true
			if e.Quantity != 2 {
				t.Fatalf("inventory quantity = %d, want 2", e.Quantity)
			}
			if e.Item == nil || e.Item.Name != stockItem.Name {
				t.Fatal("expected joined item detail in inventory read")
			}
		}
	}
	if !found {
		t.Fatal("purchased item not found in character inventory")
	}

	// Security: a Player cannot purchase for a Character they do not own.
	if _, err := AttemptPurchase(ctx, pool, fx.otherPlayerUserID, fx.interactionID, stockItem.ID, "steal-key"); err == nil {
		t.Fatal("a different player's purchase should resolve to THEIR OWN roster character, not fx.characterCardID")
	}

	// Inactive equipment cannot be purchased.
	if _, err := AttemptPurchase(ctx, pool, fx.playerUserID, fx.interactionID, fx.inactiveEquipmentItemID, "inactive-key"); err == nil {
		t.Fatal("expected inactive equipment purchase to be refused")
	}

	// Scene-not-current: moving the Show's current placement elsewhere
	// must deny further interaction use without touching the shared Scene
	// pointer as a side effect of THIS test asserting it (spec S2.1 proof:
	// the interaction handlers themselves never write current_show_scene_placement_id).
	if _, err := pool.Exec(ctx, `UPDATE shows SET current_show_scene_placement_id = NULL WHERE id = $1`, fx.showID); err != nil {
		t.Fatalf("test setup: clear current placement: %v", err)
	}
	if _, err := AttemptStance(ctx, pool, fx.playerUserID, fx.interactionID, "insight"); err == nil {
		t.Fatal("expected scene_not_current refusal once the Show's current placement no longer matches")
	}
}

type goldenPathFixture struct {
	playerUserID             string
	otherPlayerUserID        string
	audienceUserID           string
	characterCardID          string
	unskilledCharacterCardID string
	otherCharacterCardID     string
	inactiveEquipmentItemID  string
	interactionID            string
	showID                   string
	showRunID                string
}

func buildGoldenPathFixture(ctx context.Context, t *testing.T, pool *pgxpool.Pool) goldenPathFixture {
	t.Helper()
	suffix := time.Now().UTC().Format("150405.000")
	var fx goldenPathFixture

	var locationID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&locationID); err != nil {
		t.Fatalf("fixture location: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO productions (location_id, name, slug)
		SELECT $1::uuid, 'K73 Fixture Production', 'k73-fixture-production-'||$2
		WHERE NOT EXISTS (SELECT 1 FROM productions WHERE location_id = $1::uuid AND slug LIKE 'k73-fixture-production-%')
	`, locationID, suffix); err != nil {
		t.Fatalf("fixture production: %v", err)
	}
	var productionID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM productions WHERE location_id = $1::uuid AND slug LIKE 'k73-fixture-production-%' LIMIT 1`, locationID).Scan(&productionID); err != nil {
		t.Fatalf("fixture production lookup: %v", err)
	}

	// A dedicated fixture venue, NOT the real "catharsis" row: the shows/
	// showtime packages' own tests assert venue-level session exclusivity
	// against the real catharsis venue (StartShowSession's "venue_busy"
	// check), and since `go test ./...` runs different packages
	// concurrently against the one shared victory_test database, a
	// rehearsal-status session this fixture holds open on the real venue
	// races with theirs. Only the capability flag matters for this
	// package's own logic, so a same-shaped throwaway venue with
	// participant_interactions_enabled avoids the collision entirely
	// while still exercising the real venues.config flag-read path.
	var lotID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM lots WHERE location_id = $1::uuid AND slug = 'main-lot' LIMIT 1`, locationID).Scan(&lotID); err != nil {
		t.Fatalf("fixture lot lookup: %v", err)
	}
	var fixtureVenueID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO venues (lot_id, name, slug, kind, config, is_public)
		VALUES ($1::uuid, 'K73 Fixture Stage', 'k73-fixture-stage-'||$2, 'plaza', '{"participant_interactions_enabled": true}'::jsonb, FALSE)
		RETURNING id::text
	`, lotID, suffix).Scan(&fixtureVenueID); err != nil {
		t.Fatalf("fixture venue: %v", err)
	}

	mustUser := func(handle string) string {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO users (handle, display_name) VALUES ($1, $1) RETURNING id::text
		`, handle).Scan(&id); err != nil {
			t.Fatalf("fixture user %s: %v", handle, err)
		}
		return id
	}
	fx.playerUserID = mustUser("k73_player_" + suffix)
	fx.otherPlayerUserID = mustUser("k73_player2_" + suffix)
	fx.audienceUserID = mustUser("k73_audience_" + suffix)
	directorUserID := mustUser("k73_director_" + suffix)

	var showRunID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_runs (location_id, production_id, title, slug, created_by_user_id)
		VALUES ($1::uuid, $2::uuid, 'K73 Fixture Run', 'k73-fixture-run-'||$3, $4::uuid)
		RETURNING id::text
	`, locationID, productionID, suffix, directorUserID).Scan(&showRunID); err != nil {
		t.Fatalf("fixture show_run: %v", err)
	}
	fx.showRunID = showRunID

	if err := pool.QueryRow(ctx, `
		INSERT INTO shows (show_run_id, slug, title, created_by_user_id)
		VALUES ($1::uuid, 'k73-fixture-show-'||$2, 'K73 Fixture Show', $3::uuid)
		RETURNING id::text
	`, showRunID, suffix, directorUserID).Scan(&fx.showID); err != nil {
		t.Fatalf("fixture show: %v", err)
	}

	// Roster: director (backstage authority also flows through
	// location_memberships below), two players, one audience member.
	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role)
		VALUES ($1::uuid, $2::uuid, 'director')
		ON CONFLICT (location_id, user_id, role) DO NOTHING
	`, locationID, directorUserID); err != nil {
		t.Fatalf("fixture director membership: %v", err)
	}

	mustRoster := func(userID, role string) {
		if _, err := pool.Exec(ctx, `
			INSERT INTO show_run_roster_members (show_run_id, user_id, role, added_by_user_id)
			VALUES ($1::uuid, $2::uuid, $3, $4::uuid)
		`, showRunID, userID, role, directorUserID); err != nil {
			t.Fatalf("fixture roster %s: %v", role, err)
		}
	}
	mustRoster(fx.playerUserID, "player")
	mustRoster(fx.otherPlayerUserID, "player")
	mustRoster(fx.audienceUserID, "audience")

	mustCharacter := func(ownerUserID, name string, haggleSkilled bool) string {
		var cardID string
		if err := pool.QueryRow(ctx, `
			INSERT INTO character_cards (owner_user_id, location_id, name)
			VALUES ($1::uuid, $2::uuid, $3)
			RETURNING id::text
		`, ownerUserID, locationID, name).Scan(&cardID); err != nil {
			t.Fatalf("fixture character %s: %v", name, err)
		}
		if haggleSkilled {
			if _, err := pool.Exec(ctx, `
				INSERT INTO character_skills (character_card_id, skill_id, skill_name, attribute_name, ladder_step)
				VALUES ($1::uuid, 'haggle', 'Haggle', 'wit', 5)
			`, cardID); err != nil {
				t.Fatalf("fixture skill for %s: %v", name, err)
			}
		}
		return cardID
	}
	fx.characterCardID = mustCharacter(fx.playerUserID, "Fixture Hero", true)
	fx.unskilledCharacterCardID = mustCharacter(fx.playerUserID, "Fixture Novice", false)
	fx.otherCharacterCardID = mustCharacter(fx.otherPlayerUserID, "Fixture Rival", true)
	_ = fx.otherCharacterCardID

	// Select the skilled character for the roster row (SelectCharacter's own
	// self-only-by-construction guarantee is exercised via a direct roster
	// update here to keep the fixture focused on the merchant package under
	// test rather than re-driving showruns' own already-tested selection flow).
	if _, err := pool.Exec(ctx, `
		UPDATE show_run_roster_members SET character_card_id = $2::uuid
		WHERE show_run_id = $1::uuid AND user_id = $3::uuid
	`, showRunID, fx.characterCardID, fx.playerUserID); err != nil {
		t.Fatalf("fixture select character: %v", err)
	}

	// A live rehearsal Session linked to the Show (the golden path's
	// "/showtime" precondition -- required for ResolveEligibleContext's
	// active-session check and for StoreGameEvent's session-scoped writes).
	var sessionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO sessions (venue_id, status, show_id)
		VALUES ($1::uuid, 'rehearsal', $2::uuid)
		RETURNING id::text
	`, fixtureVenueID, fx.showID).Scan(&sessionID); err != nil {
		t.Fatalf("fixture session: %v", err)
	}
	// CanAct (the authority path StoreGameEvent's stance/haggle/purchase
	// events go through) hard-requires a showings row for the session --
	// normally created the first time anyone acts in the venue.
	if _, err := showings.EnsureForSession(ctx, pool, sessionID, directorUserID); err != nil {
		t.Fatalf("fixture showing: %v", err)
	}
	// CanAct also requires a session_participants row -- normally created by
	// joining the venue's live session (/api/session/catharsis/join).
	if _, err := pool.Exec(ctx, `
		INSERT INTO session_participants (session_id, user_id, role)
		VALUES ($1::uuid, $2::uuid, 'cast'::location_role)
	`, sessionID, fx.playerUserID); err != nil {
		t.Fatalf("fixture session participant: %v", err)
	}

	// Place the seeded Courtyard Scene into this real Show, and make it current.
	var courtyardSceneID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM scenes WHERE location_id = $1::uuid AND slug = 'courtyard'`, locationID).Scan(&courtyardSceneID); err != nil {
		t.Fatalf("fixture: seeded courtyard scene missing (Phase A migration 057 not applied?): %v", err)
	}
	// venue_id overrides the Scene's own default_venue_id (which points at
	// the real catharsis venue) so resolvePlacementVenueSlug resolves to
	// this fixture's throwaway venue instead.
	var placementID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_scene_placements (show_id, scene_id, venue_id, status)
		VALUES ($1::uuid, $2::uuid, $3::uuid, 'ready')
		RETURNING id::text
	`, fx.showID, courtyardSceneID, fixtureVenueID).Scan(&placementID); err != nil {
		t.Fatalf("fixture placement: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE shows SET current_show_scene_placement_id = $2::uuid WHERE id = $1::uuid`, fx.showID, placementID); err != nil {
		t.Fatalf("fixture set current placement: %v", err)
	}

	// Attach the seeded Kessa interaction to THIS placement (the real
	// authoring flow this fixture stands in for -- Phase E's UI).
	interaction, err := CreateInteraction(ctx, pool, directorUserID, placementID, CreateInteractionInput{
		InternalName:     "Visit Kessa's Shop",
		StageButtonLabel: "Visit Kessa's Shop",
		InteractionType:  InteractionTypeOpenEquipMode,
		Configuration:    map[string]any{"packet_slug": "kessa"},
	})
	if err != nil {
		t.Fatalf("fixture CreateInteraction: %v", err)
	}
	fx.interactionID = interaction.ID

	// An inactive equipment item, for the "inactive cannot be purchased" proof.
	if err := pool.QueryRow(ctx, `
		INSERT INTO equipment_items (location_id, name, slug, active)
		VALUES ($1::uuid, 'Fixture Broken Lantern', 'fixture-broken-lantern-'||$2, FALSE)
		RETURNING id::text
	`, locationID, suffix).Scan(&fx.inactiveEquipmentItemID); err != nil {
		t.Fatalf("fixture inactive equipment: %v", err)
	}

	t.Cleanup(func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer ccancel()
		_, _ = pool.Exec(cctx, `UPDATE sessions SET status = 'closed' WHERE id = $1::uuid`, sessionID)
	})

	return fx
}
