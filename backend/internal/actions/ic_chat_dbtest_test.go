package actions

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
)

type icChatFixture struct {
	sessionID   string
	showRunID   string
	userA       string
	userB       string
	characterA1 string // owned by userA
	characterA2 string // owned by userA, a second Character to switch to
}

func buildICChatFixture(ctx context.Context, t *testing.T, pool *pgxpool.Pool) icChatFixture {
	t.Helper()
	var fx icChatFixture
	suffix := time.Now().UTC().Format("150405.000000000") + "_" + strings.ToLower(strings.ReplaceAll(t.Name(), "/", "_"))

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

	mkUser := func(prefix string) string {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO users (handle, display_name) VALUES ($1, $2) RETURNING id::text
		`, prefix+"_"+suffix, prefix).Scan(&id); err != nil {
			t.Fatalf("fixture user %s: %v", prefix, err)
		}
		return id
	}
	fx.userA = mkUser("ic_a")
	fx.userB = mkUser("ic_b")

	mkCharacter := func(owner, name string) string {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO character_cards (owner_user_id, location_id, name, portrait_url)
			VALUES ($1::uuid, $2::uuid, $3, 'https://example.test/portrait.png')
			RETURNING id::text
		`, owner, locationID, name).Scan(&id); err != nil {
			t.Fatalf("fixture character %s: %v", name, err)
		}
		return id
	}
	fx.characterA1 = mkCharacter(fx.userA, "Aldric the First")
	fx.characterA2 = mkCharacter(fx.userA, "Aldric the Second")

	if err := pool.QueryRow(ctx, `
		INSERT INTO show_runs (location_id, production_id, title, slug, created_by_user_id)
		VALUES ($1::uuid, $2::uuid, 'IC Fixture Show Run', 'ic-fixture-show-run-'||$3, $4::uuid)
		RETURNING id::text
	`, locationID, productionID, suffix, fx.userA).Scan(&fx.showRunID); err != nil {
		t.Fatalf("fixture show run: %v", err)
	}

	var showID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO shows (show_run_id, slug, title, created_by_user_id)
		VALUES ($1::uuid, 'ic-fixture-show-'||$2, 'IC Fixture Show', $3::uuid)
		RETURNING id::text
	`, fx.showRunID, suffix, fx.userA).Scan(&showID); err != nil {
		t.Fatalf("fixture show: %v", err)
	}

	if err := pool.QueryRow(ctx, `
		INSERT INTO sessions (venue_id, status, show_id)
		SELECT v.id, 'rehearsal', $1::uuid FROM venues v WHERE v.slug = 'catharsis'
		RETURNING id::text
	`, showID).Scan(&fx.sessionID); err != nil {
		t.Fatalf("fixture session: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO showings (session_id, production_id, venue_id, status, created_by)
		SELECT s.id, p.id, s.venue_id, 'rehearsal', $2::uuid
		FROM sessions s, productions p
		WHERE s.id = $1::uuid AND p.location_id = $3::uuid
		ORDER BY p.created_at ASC LIMIT 1
		ON CONFLICT (session_id) DO NOTHING
	`, fx.sessionID, fx.userA, locationID); err != nil {
		t.Fatalf("fixture showing: %v", err)
	}

	for user, role := range map[string]string{fx.userA: "cast", fx.userB: "cast"} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO session_participants (session_id, user_id, role)
			VALUES ($1::uuid, $2::uuid, $3::location_role)
			ON CONFLICT (session_id, user_id) DO UPDATE SET role = EXCLUDED.role
		`, fx.sessionID, user, role); err != nil {
			t.Fatalf("fixture participant: %v", err)
		}
	}

	// userA has selected characterA1; userB has no roster row at all (no
	// Character selected -- the "choose a Character first" case).
	if _, err := pool.Exec(ctx, `
		INSERT INTO show_run_roster_members (show_run_id, user_id, role, character_card_id, added_by_user_id)
		VALUES ($1::uuid, $2::uuid, 'player', $3::uuid, $2::uuid)
	`, fx.showRunID, fx.userA, fx.characterA1); err != nil {
		t.Fatalf("fixture roster row: %v", err)
	}

	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `UPDATE sessions SET status = 'closed', show_id = NULL WHERE id = $1::uuid`, fx.sessionID)
		_, _ = pool.Exec(bg, `DELETE FROM actions WHERE session_id = $1::uuid`, fx.sessionID)
		_, _ = pool.Exec(bg, `DELETE FROM showings WHERE session_id = $1::uuid`, fx.sessionID)
		_, _ = pool.Exec(bg, `DELETE FROM session_participants WHERE session_id = $1::uuid`, fx.sessionID)
		_, _ = pool.Exec(bg, `DELETE FROM sessions WHERE id = $1::uuid`, fx.sessionID)
		_, _ = pool.Exec(bg, `DELETE FROM shows WHERE id = $1::uuid`, showID)
		_, _ = pool.Exec(bg, `DELETE FROM show_run_roster_members WHERE show_run_id = $1::uuid`, fx.showRunID)
		_, _ = pool.Exec(bg, `DELETE FROM show_runs WHERE id = $1::uuid`, fx.showRunID)
		_, _ = pool.Exec(bg, `DELETE FROM character_cards WHERE owner_user_id = $1::uuid`, fx.userA)
		_, _ = pool.Exec(bg, `DELETE FROM users WHERE id IN ($1::uuid, $2::uuid)`, fx.userA, fx.userB)
	})

	return fx
}

func TestICChat_ResolvesCharacterServerSide(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := buildICChatFixture(ctx, t, pool)

	stored, err := StoreICChatMessage(ctx, pool, ICChatMessageRequest{
		SessionID: fx.sessionID, ActorID: fx.userA, Text: "Hail, traveler.",
	})
	if err != nil {
		t.Fatalf("StoreICChatMessage: %v", err)
	}
	if stored.Payload["character_id"] != fx.characterA1 {
		t.Fatalf("character_id = %v, want %v", stored.Payload["character_id"], fx.characterA1)
	}
	if stored.Payload["character_name"] != "Aldric the First" {
		t.Fatalf("character_name = %v, want Aldric the First", stored.Payload["character_name"])
	}
	// Accountable user identity is always stored alongside the Character.
	if stored.ActorID != fx.userA {
		t.Fatalf("actor_id = %v, want %v", stored.ActorID, fx.userA)
	}
}

func TestICChat_NoCharacterSelectedFailsCleanly(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := buildICChatFixture(ctx, t, pool)

	_, err := StoreICChatMessage(ctx, pool, ICChatMessageRequest{
		SessionID: fx.sessionID, ActorID: fx.userB, Text: "I have nothing to say.",
	})
	if !errors.Is(err, ErrNoCharacterSelected) {
		t.Fatalf("expected ErrNoCharacterSelected, got %v", err)
	}
}

func TestICChat_CharacterSwitchAffectsFutureMessagesOnly(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := buildICChatFixture(ctx, t, pool)

	first, err := StoreICChatMessage(ctx, pool, ICChatMessageRequest{
		SessionID: fx.sessionID, ActorID: fx.userA, Text: "First message.",
	})
	if err != nil {
		t.Fatalf("first message: %v", err)
	}
	if first.Payload["character_id"] != fx.characterA1 {
		t.Fatalf("first message character = %v, want %v", first.Payload["character_id"], fx.characterA1)
	}

	// Switch selected Character.
	if _, err := pool.Exec(ctx, `
		UPDATE show_run_roster_members SET character_card_id = $1::uuid
		WHERE show_run_id = $2::uuid AND user_id = $3::uuid
	`, fx.characterA2, fx.showRunID, fx.userA); err != nil {
		t.Fatalf("switch character: %v", err)
	}

	second, err := StoreICChatMessage(ctx, pool, ICChatMessageRequest{
		SessionID: fx.sessionID, ActorID: fx.userA, Text: "Second message.",
	})
	if err != nil {
		t.Fatalf("second message: %v", err)
	}
	if second.Payload["character_id"] != fx.characterA2 {
		t.Fatalf("second message character = %v, want %v", second.Payload["character_id"], fx.characterA2)
	}

	// Old message (loaded fresh, simulating history re-fetch) still shows
	// the Character used when it was sent, not the now-current one.
	var storedCharID string
	if err := pool.QueryRow(ctx, `
		SELECT payload ->> 'character_id' FROM actions WHERE id = $1
	`, first.ID).Scan(&storedCharID); err != nil {
		t.Fatalf("reload first message: %v", err)
	}
	if storedCharID != fx.characterA1 {
		t.Fatalf("old message character drifted: got %v, want %v", storedCharID, fx.characterA1)
	}
}

func TestICChat_ForgedCharacterFieldIsIgnored(t *testing.T) {
	// ICChatMessageRequest has no character_id field at all -- there is
	// nothing for a forged request to populate. This test documents that
	// invariant at the type level: attempting to build a request with an
	// extra field would be a compile error, and the resolved speaker
	// always comes from resolveSpeakerCharacter, never from req.
	pool := dbtest.OpenTestPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := buildICChatFixture(ctx, t, pool)

	stored, err := StoreICChatMessage(ctx, pool, ICChatMessageRequest{
		SessionID: fx.sessionID, ActorID: fx.userA, Text: "No impersonation possible.",
	})
	if err != nil {
		t.Fatalf("StoreICChatMessage: %v", err)
	}
	if stored.Payload["character_id"] != fx.characterA1 {
		t.Fatalf("expected server-resolved character, got %v", stored.Payload["character_id"])
	}
}
