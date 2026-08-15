package socio

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
)

// FateState is a Character's canonical live-play Fate balance. Distinct
// from Chapter 2 Character Creation's own FP bookkeeping (chapter2_stage.go,
// JSON-blob-backed) -- see characters.HandoffCreationFateToSocio for the
// one-time bridge between the two.
type FateState struct {
	CharacterCardID string    `json:"character_card_id"`
	Balance         int       `json:"balance"`
	CreationMode    bool      `json:"creation_mode"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// FateLedgerEntry is one append-only record of a Fate balance change --
// "never silently spent by the system" (kernel-88 spec §4.5) requires this
// to be auditable, not just the resulting balance.
type FateLedgerEntry struct {
	ID              string    `json:"id"`
	CharacterCardID string    `json:"character_card_id"`
	ShowID          string    `json:"show_id"`
	Delta           int       `json:"delta"`
	Reason          string    `json:"reason"`
	Note            string    `json:"note"`
	ActorUserID     string    `json:"actor_user_id"`
	BalanceAfter    int       `json:"balance_after"`
	CreatedAt       time.Time `json:"created_at"`
}

const fateStateColumns = `character_card_id::text, balance, creation_mode, updated_at`

func scanFateState(row pgx.Row) (FateState, error) {
	var s FateState
	if err := row.Scan(&s.CharacterCardID, &s.Balance, &s.CreationMode, &s.UpdatedAt); err != nil {
		return FateState{}, err
	}
	return s, nil
}

// ensureFateRow lazily creates a Character's Fate row (balance 0, not in
// creation mode) if it doesn't exist yet, matching character_socio_state's
// lazy-row convention.
func ensureFateRow(ctx context.Context, pool *pgxpool.Pool, characterCardID string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO character_socio_fate (character_card_id) VALUES ($1)
		ON CONFLICT (character_card_id) DO NOTHING
	`, characterCardID)
	return err
}

// GetFate returns a Character's Fate balance, creating a zeroed row on
// first read. No authority check -- callers gate read access, same pattern
// as GetState.
func GetFate(ctx context.Context, pool *pgxpool.Pool, characterCardID string) (FateState, error) {
	characterCardID = strings.TrimSpace(characterCardID)
	if characterCardID == "" {
		return FateState{}, errors.New("character_card_id_required")
	}
	if err := ensureFateRow(ctx, pool, characterCardID); err != nil {
		return FateState{}, err
	}
	row := pool.QueryRow(ctx, `SELECT `+fateStateColumns+` FROM character_socio_fate WHERE character_card_id = $1`, characterCardID)
	return scanFateState(row)
}

// requireOwnerOrShowCharacterAuthority allows either Director+/Producer/
// Operator (requireShowCharacterAuthority) or the Player currently bound to
// characterCardID via their own Show Run roster selection
// (showruns.LoadMyRosterMember) -- the authority shape Fate spends and
// Stance changes share: a Player may act on their own Character, a
// Director may act on anyone's.
func requireOwnerOrShowCharacterAuthority(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, characterCardID string) error {
	if err := requireShowCharacterAuthority(ctx, pool, actorUserID, showID, characterCardID); err == nil {
		return nil
	}

	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return errors.New("not_authenticated")
	}
	s, err := shows.LoadShowByID(ctx, pool, showID)
	if err != nil {
		return err
	}
	member, err := showruns.LoadMyRosterMember(ctx, pool, actorUserID, s.ShowRunID)
	if err != nil {
		return errors.New("not_authorized")
	}
	if member.CharacterCardID == "" || member.CharacterCardID != characterCardID {
		return errors.New("not_authorized")
	}
	return nil
}

func fateCap(creationMode bool) int {
	if creationMode {
		return 12
	}
	return 7
}

func insertFateLedgerEntry(ctx context.Context, pool *pgxpool.Pool, characterCardID, showID string, delta int, reason, note, actorUserID string, balanceAfter int) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO character_socio_fate_ledger
			(character_card_id, show_id, delta, reason, note, actor_user_id, balance_after)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, characterCardID, showID, delta, reason, note, actorUserID, balanceAfter)
	return err
}

// SpendFate lets the owning Player (or Director+) decrement a Character's
// own Fate balance -- the only path that reduces a balance without Director
// authority. amount must be positive and cannot exceed the current balance
// (a spend can never go negative); this is the sole self-service mutation,
// closing "client cannot award own Fate" by construction: SpendFate never
// increases balance.
func SpendFate(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, characterCardID string, amount int, note string) (FateState, error) {
	if err := requireOwnerOrShowCharacterAuthority(ctx, pool, actorUserID, showID, characterCardID); err != nil {
		return FateState{}, err
	}
	if amount <= 0 {
		return FateState{}, errors.New("amount_must_be_positive")
	}
	current, err := GetFate(ctx, pool, characterCardID)
	if err != nil {
		return FateState{}, err
	}
	if amount > current.Balance {
		return FateState{}, errors.New("insufficient_fate")
	}
	newBalance := current.Balance - amount

	tx, err := pool.Begin(ctx)
	if err != nil {
		return FateState{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE character_socio_fate SET balance = $2, updated_at = NOW() WHERE character_card_id = $1
	`, characterCardID, newBalance); err != nil {
		return FateState{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO character_socio_fate_ledger
			(character_card_id, show_id, delta, reason, note, actor_user_id, balance_after)
		VALUES ($1, $2, $3, 'player_spend', $4, $5, $6)
	`, characterCardID, showID, -amount, note, actorUserID, newBalance); err != nil {
		return FateState{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return FateState{}, err
	}
	return GetFate(ctx, pool, characterCardID)
}

// AwardFate is Director+-only: the sole path that can increase a
// Character's Fate balance. delta may be any signed integer (a correction
// might be negative); the result is clamped into [0, cap] where cap is 7
// normally or 12 while creation_mode is active. reason should be one of the
// character_socio_fate_ledger reason-check values ('director_award',
// 'director_correction', 'creation_guidance', 'debrief_guidance',
// 'creation_handoff').
func AwardFate(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, characterCardID string, delta int, reason, note string) (FateState, error) {
	if err := requireShowCharacterAuthority(ctx, pool, actorUserID, showID, characterCardID); err != nil {
		return FateState{}, err
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "director_award"
	}
	current, err := GetFate(ctx, pool, characterCardID)
	if err != nil {
		return FateState{}, err
	}
	newBalance := current.Balance + delta
	if newBalance < 0 {
		newBalance = 0
	}
	if cap := fateCap(current.CreationMode); newBalance > cap {
		newBalance = cap
	}
	appliedDelta := newBalance - current.Balance

	tx, err := pool.Begin(ctx)
	if err != nil {
		return FateState{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE character_socio_fate SET balance = $2, updated_at = NOW() WHERE character_card_id = $1
	`, characterCardID, newBalance); err != nil {
		return FateState{}, err
	}
	if appliedDelta != 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO character_socio_fate_ledger
				(character_card_id, show_id, delta, reason, note, actor_user_id, balance_after)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, characterCardID, showID, appliedDelta, reason, note, actorUserID, newBalance); err != nil {
			return FateState{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return FateState{}, err
	}
	return GetFate(ctx, pool, characterCardID)
}

// SetCreationMode toggles whether a Character's Fate balance may
// temporarily exceed the normal cap of 7 (kernel-88 spec §4.3). Turning it
// **off** is the one clamp point: if the balance is currently above 7 it is
// clamped down to 7 and the excess is logged as a 'creation_clamp' ledger
// entry ("excess above 7 is lost" -- spec §4.3). Director+ only.
func SetCreationMode(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, characterCardID string, on bool) (FateState, error) {
	if err := requireShowCharacterAuthority(ctx, pool, actorUserID, showID, characterCardID); err != nil {
		return FateState{}, err
	}
	current, err := GetFate(ctx, pool, characterCardID)
	if err != nil {
		return FateState{}, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return FateState{}, err
	}
	defer tx.Rollback(ctx)

	newBalance := current.Balance
	if !on && current.Balance > 7 {
		newBalance = 7
	}

	if _, err := tx.Exec(ctx, `
		UPDATE character_socio_fate SET creation_mode = $2, balance = $3, updated_at = NOW() WHERE character_card_id = $1
	`, characterCardID, on, newBalance); err != nil {
		return FateState{}, err
	}
	if newBalance != current.Balance {
		if _, err := tx.Exec(ctx, `
			INSERT INTO character_socio_fate_ledger
				(character_card_id, show_id, delta, reason, note, actor_user_id, balance_after)
			VALUES ($1, $2, $3, 'creation_clamp', 'excess Fate lost at Character Creation completion', $4, $5)
		`, characterCardID, showID, newBalance-current.Balance, actorUserID, newBalance); err != nil {
			return FateState{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return FateState{}, err
	}
	return GetFate(ctx, pool, characterCardID)
}

// ListFateLedger returns the most recent Fate ledger entries for a
// Character, newest first.
func ListFateLedger(ctx context.Context, pool *pgxpool.Pool, characterCardID string, limit int) ([]FateLedgerEntry, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
		SELECT id::text, character_card_id::text, show_id::text, delta, reason, note, actor_user_id::text, balance_after, created_at
		FROM character_socio_fate_ledger
		WHERE character_card_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, characterCardID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []FateLedgerEntry{}
	for rows.Next() {
		var e FateLedgerEntry
		if err := rows.Scan(&e.ID, &e.CharacterCardID, &e.ShowID, &e.Delta, &e.Reason, &e.Note, &e.ActorUserID, &e.BalanceAfter, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
