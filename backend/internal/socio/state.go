package socio

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
)

// requireShowCharacterAuthority checks that actorUserID has Director+/
// Producer/Operator authority over showID, and that characterCardID is
// actually a Character on that Show's roster -- both are re-checked on
// every mutation server-side, never trusted from the client (kernel-85
// S10).
func requireShowCharacterAuthority(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, characterCardID string) error {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return errors.New("not_authenticated")
	}
	characterCardID = strings.TrimSpace(characterCardID)
	if characterCardID == "" {
		return errors.New("character_card_id_required")
	}
	s, err := shows.LoadShowByID(ctx, pool, showID)
	if err != nil {
		return err
	}
	sr, err := showruns.LoadShowRunByID(ctx, pool, s.ShowRunID)
	if err != nil {
		return err
	}
	ok, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("not_authorized")
	}

	var onRoster bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM show_run_roster_members
			WHERE show_run_id = $1 AND character_card_id = $2 AND removed_at IS NULL
		)
	`, s.ShowRunID, characterCardID).Scan(&onRoster)
	if err != nil {
		return err
	}
	if !onRoster {
		return errors.New("character_not_on_show")
	}
	return nil
}

const stateColumns = `
	character_card_id::text,
	health_current, health_max, psyche_current, psyche_max,
	motion_current, motion_max, will_current, will_max,
	essence_current, essence_max, focus_current, focus_max,
	perception_current, perception_max, heart_current, heart_max,
	updated_at
`

func scanState(row pgx.Row) (State, error) {
	var s State
	var health, psyche, motion, will, essence, focus, perception, heart Pool
	if err := row.Scan(
		&s.CharacterCardID,
		&health.Current, &health.Max, &psyche.Current, &psyche.Max,
		&motion.Current, &motion.Max, &will.Current, &will.Max,
		&essence.Current, &essence.Max, &focus.Current, &focus.Max,
		&perception.Current, &perception.Max, &heart.Current, &heart.Max,
		&s.UpdatedAt,
	); err != nil {
		return State{}, err
	}
	pools := map[PoolKey]Pool{
		PoolHealth: health, PoolPsyche: psyche, PoolMotion: motion, PoolWill: will,
		PoolEssence: essence, PoolFocus: focus, PoolPerception: perception, PoolHeart: heart,
	}
	for _, meta := range PoolLabels {
		p := pools[meta.Key]
		p.Key = meta.Key
		p.Label = meta.Label
		s.Pools = append(s.Pools, p)
	}
	return s, nil
}

// ensureStateRow lazily creates a character's socio_state row (all pools
// zeroed) if it doesn't exist yet -- there is no backfill for every
// existing character_cards row, only created on first read/write.
func ensureStateRow(ctx context.Context, pool *pgxpool.Pool, characterCardID string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO character_socio_state (character_card_id) VALUES ($1)
		ON CONFLICT (character_card_id) DO NOTHING
	`, characterCardID)
	return err
}

// GetState returns a Character's eight HP pools, creating a zeroed row on
// first read. No authority check -- callers (Game Status list, HTTP layer)
// gate read access themselves, matching this codebase's existing
// "existence isn't secret, read access is checked by the caller" pattern.
func GetState(ctx context.Context, pool *pgxpool.Pool, characterCardID string) (State, error) {
	characterCardID = strings.TrimSpace(characterCardID)
	if characterCardID == "" {
		return State{}, errors.New("character_card_id_required")
	}
	if err := ensureStateRow(ctx, pool, characterCardID); err != nil {
		return State{}, err
	}
	row := pool.QueryRow(ctx, `SELECT `+stateColumns+` FROM character_socio_state WHERE character_card_id = $1`, characterCardID)
	return scanState(row)
}

// SetPool is the one canonical write path for an HP pool value (kernel-85
// S2.2: "must call the canonical Socio mechanics/value operation... must
// not write directly to a second status store"). current is clamped into
// [0, max]; max itself is never allowed negative. Director+ authority on
// the given Show, and characterCardID must actually be on that Show's
// roster -- both re-checked here, not trusted from the caller.
func SetPool(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, characterCardID string, poolKey PoolKey, current, max int) (State, error) {
	if err := requireShowCharacterAuthority(ctx, pool, actorUserID, showID, characterCardID); err != nil {
		return State{}, err
	}
	if !IsValidPoolKey(string(poolKey)) {
		return State{}, errors.New("invalid_pool_key")
	}
	if max < 0 {
		max = 0
	}
	if current < 0 {
		current = 0
	}
	if current > max {
		current = max
	}
	if err := ensureStateRow(ctx, pool, characterCardID); err != nil {
		return State{}, err
	}

	var query string
	switch poolKey {
	case PoolHealth:
		query = `UPDATE character_socio_state SET health_current = $2, health_max = $3, updated_at = NOW() WHERE character_card_id = $1`
	case PoolPsyche:
		query = `UPDATE character_socio_state SET psyche_current = $2, psyche_max = $3, updated_at = NOW() WHERE character_card_id = $1`
	case PoolMotion:
		query = `UPDATE character_socio_state SET motion_current = $2, motion_max = $3, updated_at = NOW() WHERE character_card_id = $1`
	case PoolWill:
		query = `UPDATE character_socio_state SET will_current = $2, will_max = $3, updated_at = NOW() WHERE character_card_id = $1`
	case PoolEssence:
		query = `UPDATE character_socio_state SET essence_current = $2, essence_max = $3, updated_at = NOW() WHERE character_card_id = $1`
	case PoolFocus:
		query = `UPDATE character_socio_state SET focus_current = $2, focus_max = $3, updated_at = NOW() WHERE character_card_id = $1`
	case PoolPerception:
		query = `UPDATE character_socio_state SET perception_current = $2, perception_max = $3, updated_at = NOW() WHERE character_card_id = $1`
	case PoolHeart:
		query = `UPDATE character_socio_state SET heart_current = $2, heart_max = $3, updated_at = NOW() WHERE character_card_id = $1`
	}
	if _, err := pool.Exec(ctx, query, characterCardID, current, max); err != nil {
		return State{}, err
	}
	return GetState(ctx, pool, characterCardID)
}
