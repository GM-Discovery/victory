package socio

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StanceDefinition is one row of socio_stances (migration 101) -- registry
// data, not a Go const list, matching socio_statuses' convention: adding a
// Stance for a later module/expansion is a seed-row INSERT.
type StanceDefinition struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	WheelOrder  int    `json:"wheel_order"`
}

// Stance is a Character's currently-set Stance, or an unset one (Key ==
// "" is a valid state -- no Stance chosen yet).
type Stance struct {
	CharacterCardID string    `json:"character_card_id"`
	Key             string    `json:"key,omitempty"`
	Label           string    `json:"label,omitempty"`
	SetByUserID     string    `json:"set_by_user_id,omitempty"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ListStanceDefinitions returns the full Stance registry, ordered for wheel
// display.
func ListStanceDefinitions(ctx context.Context, pool *pgxpool.Pool) ([]StanceDefinition, error) {
	rows, err := pool.Query(ctx, `SELECT key, label, description, wheel_order FROM socio_stances ORDER BY wheel_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StanceDefinition{}
	for rows.Next() {
		var d StanceDefinition
		if err := rows.Scan(&d.Key, &d.Label, &d.Description, &d.WheelOrder); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ensureStanceRow lazily creates a Character's Stance row (unset) if it
// doesn't exist yet.
func ensureStanceRow(ctx context.Context, pool *pgxpool.Pool, characterCardID string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO character_socio_stance (character_card_id) VALUES ($1)
		ON CONFLICT (character_card_id) DO NOTHING
	`, characterCardID)
	return err
}

// GetStance returns a Character's current Stance, creating an unset row on
// first read. No authority check -- callers gate read access.
func GetStance(ctx context.Context, pool *pgxpool.Pool, characterCardID string) (Stance, error) {
	characterCardID = strings.TrimSpace(characterCardID)
	if characterCardID == "" {
		return Stance{}, errors.New("character_card_id_required")
	}
	if err := ensureStanceRow(ctx, pool, characterCardID); err != nil {
		return Stance{}, err
	}
	var s Stance
	var key, label, setBy *string
	err := pool.QueryRow(ctx, `
		SELECT cs.character_card_id::text, cs.stance_key, s.label, cs.set_by_user_id::text, cs.updated_at
		FROM character_socio_stance cs
		LEFT JOIN socio_stances s ON s.key = cs.stance_key
		WHERE cs.character_card_id = $1
	`, characterCardID).Scan(&s.CharacterCardID, &key, &label, &setBy, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Stance{}, errors.New("stance_not_found")
		}
		return Stance{}, err
	}
	if key != nil {
		s.Key = *key
	}
	if label != nil {
		s.Label = *label
	}
	if setBy != nil {
		s.SetByUserID = *setBy
	}
	return s, nil
}

// SetStance changes a Character's Stance. Authority: the owning Player
// (via their own Show Run roster selection) or Director+ -- unlike pool/
// status mutations, this is one a Player may perform on their own
// Character (kernel-88 spec §5.3). There is no cooldown/rule-gate in v1
// (no such rule exists in the corpus yet); this is the point a future
// "only when rules permit" check plugs in.
func SetStance(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, characterCardID, stanceKey string) (Stance, error) {
	if err := requireOwnerOrShowCharacterAuthority(ctx, pool, actorUserID, showID, characterCardID); err != nil {
		return Stance{}, err
	}
	stanceKey = strings.TrimSpace(stanceKey)
	if stanceKey == "" {
		return Stance{}, errors.New("stance_key_required")
	}
	if err := ensureStanceRow(ctx, pool, characterCardID); err != nil {
		return Stance{}, err
	}
	_, err := pool.Exec(ctx, `
		UPDATE character_socio_stance
		SET stance_key = $2, set_by_user_id = $3, updated_at = NOW()
		WHERE character_card_id = $1
	`, characterCardID, stanceKey, actorUserID)
	if err != nil {
		if strings.Contains(err.Error(), "character_socio_stance_stance_key_fkey") {
			return Stance{}, errors.New("invalid_stance_key")
		}
		return Stance{}, err
	}
	return GetStance(ctx, pool, characterCardID)
}
