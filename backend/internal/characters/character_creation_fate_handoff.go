package characters

// Kernel 88: one-time handoff of Chapter 2's creation-time FP balance
// (chapter2_stage.go, JSON-blob-backed, StartingFP=12) into Socio's
// canonical live-play Fate table (character_socio_fate, migration 100).
// This package deliberately does NOT import backend/internal/socio: socio
// imports cohorts, which imports network, which imports actions, which
// imports characters (actions/player_roll.go) -- so characters -> socio
// would close an import cycle. This file duplicates the minimal insert/
// ledger SQL instead, matching this codebase's established convention for
// resolving this exact class of cycle (see rollaudience.go's header
// comment for the same tradeoff, and socio/coordination.go's
// isActivePlayerOnShow). socio.AwardFate/SpendFate/SetCreationMode remain
// the only writers for every *other* Fate mutation -- this is the one
// narrow, well-scoped exception, and it only ever fires once per
// Character, at Chapter 2 completion, before any live Show exists to
// attribute a Show-scoped ledger entry to.

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HandoffCreationFateToSocio seeds a Character's canonical live-play Fate
// balance from their finished Chapter 2 creation balance, clamping to the
// normal cap of 7 if creation finished above it -- "excess above 7 is lost"
// (kernel-88 spec §4.3), applied here exactly once rather than via
// socio.SetCreationMode's Director-gated toggle, since Chapter 2 completion
// has no Show/Director context to gate against. Called from
// HandleChapter2Stage exactly when result.NextStage == 0 (chapter
// complete). Idempotent: re-running (e.g. a retried request) overwrites the
// same seeded balance rather than compounding it.
func HandoffCreationFateToSocio(ctx context.Context, pool *pgxpool.Pool, actorUserID, characterCardID string, finalCreationBalance int) error {
	if characterCardID == "" {
		return errors.New("character_card_id_required")
	}

	var ownerUserID string
	if err := pool.QueryRow(ctx, `
		SELECT owner_user_id::text FROM character_cards WHERE id = $1 AND is_deleted = FALSE
	`, characterCardID).Scan(&ownerUserID); err != nil {
		return err
	}
	if ownerUserID != actorUserID {
		return errors.New("not_authorized")
	}

	seeded := finalCreationBalance
	if seeded < 0 {
		seeded = 0
	}
	clamped := seeded > 7
	if clamped {
		seeded = 7
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO character_socio_fate (character_card_id, balance, creation_mode)
		VALUES ($1, $2, FALSE)
		ON CONFLICT (character_card_id) DO UPDATE SET balance = $2, creation_mode = FALSE, updated_at = NOW()
	`, characterCardID, seeded); err != nil {
		return err
	}

	note := "seeded from Chapter 2 Character Creation"
	if clamped {
		note = fmt.Sprintf("seeded from Chapter 2 (balance %d), clamped to 7 at creation completion", finalCreationBalance)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO character_socio_fate_ledger
			(character_card_id, show_id, delta, reason, note, actor_user_id, balance_after)
		VALUES ($1, NULL, $2, 'creation_handoff', $3, $4, $5)
	`, characterCardID, seeded, note, actorUserID, seeded); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
