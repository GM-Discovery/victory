// Package audienceprojection implements Kernel 93's one canonical
// Showing-level Audience projection configuration object (spec §4, §26):
// Dice Rolls / Presence / Character Health Statuses, each a plain on/off
// toggle, persisted per Showing (not a Scene, not an account preference --
// spec §20).
package audienceprojection

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showings"
	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
)

// Config is one row of audience_projection_configs. ShowDiceRolls defaults
// TRUE to match current public rollaudience.ModeShow behavior (spec §5: do
// not silently change existing public dice behavior). ShowPresence and
// ShowHealthStatuses default FALSE -- Director opts in per Showing.
type Config struct {
	ShowingID          string    `json:"showing_id"`
	ShowDiceRolls      bool      `json:"show_dice_rolls"`
	ShowPresence       bool      `json:"show_presence"`
	ShowHealthStatuses bool      `json:"show_health_statuses"`
	UpdatedByUserID    string    `json:"updated_by_user_id,omitempty"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func defaultConfig(showingID string) Config {
	return Config{ShowingID: showingID, ShowDiceRolls: true, ShowPresence: false, ShowHealthStatuses: false}
}

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// ForShowing loads a Showing's Audience projection config, returning the
// documented defaults (spec §5) if no row has been persisted yet -- a
// Showing with no explicit Director configuration is not an error, it is
// simply unconfigured.
func ForShowing(ctx context.Context, q querier, showingID string) (Config, error) {
	showingID = strings.TrimSpace(showingID)
	if showingID == "" {
		return Config{}, errors.New("showing_id_required")
	}
	var c Config
	err := q.QueryRow(ctx, `
		SELECT showing_id::text, show_dice_rolls, show_presence, show_health_statuses,
		       COALESCE(updated_by_user_id::text, ''), updated_at
		FROM audience_projection_configs
		WHERE showing_id = $1
	`, showingID).Scan(&c.ShowingID, &c.ShowDiceRolls, &c.ShowPresence, &c.ShowHealthStatuses, &c.UpdatedByUserID, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return defaultConfig(showingID), nil
	}
	if err != nil {
		return Config{}, err
	}
	return c, nil
}

// DiceRollsHiddenFromAudience reports whether sessionID's Showing has the
// Kernel 93 Audience Dice Rolls toggle turned off. A session with no Showing
// (a legacy/tutorial venue never wrapped in Showtime) is unaffected -- dice
// stays visible, matching current public rollaudience.ModeShow behavior
// (spec §5: preserve existing public behavior as the default). Callers
// combine this with their own "is this viewer actually Audience" check --
// this function has no opinion on viewer role, only on the Showing's
// configuration.
func DiceRollsHiddenFromAudience(ctx context.Context, pool *pgxpool.Pool, sessionID string) (bool, error) {
	showing, err := showings.LoadBySession(ctx, pool, sessionID)
	if err != nil {
		return false, nil
	}
	cfg, err := ForShowing(ctx, pool, showing.ID)
	if err != nil {
		return false, err
	}
	return !cfg.ShowDiceRolls, nil
}

// resolveManagingShowAndShowing duplicates
// audienceadmission.resolveManagingShowAndShowing (small, deliberate
// per-package duplication of the Kernel 92 Showtime session-selection query
// and its authority check, matching this codebase's established
// convention -- see that function's own comment).
func resolveManagingShowAndShowing(ctx context.Context, pool *pgxpool.Pool, actorUserID, shortCode string) (showings.Showing, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return showings.Showing{}, errors.New("not_authenticated")
	}

	s, err := shows.LoadShowByCode(ctx, pool, actorUserID, shortCode)
	if err != nil {
		return showings.Showing{}, err
	}
	sr, err := showruns.LoadShowRunByID(ctx, pool, s.ShowRunID)
	if err != nil {
		return showings.Showing{}, err
	}
	canManage, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return showings.Showing{}, err
	}
	if !canManage {
		return showings.Showing{}, errors.New("not_authorized")
	}

	var sessionID string
	err = pool.QueryRow(ctx, `
		SELECT sess.id::text
		FROM sessions sess
		WHERE sess.show_id = $1 AND sess.status IN ('rehearsal', 'live')
		ORDER BY sess.started_at DESC LIMIT 1
	`, s.ID).Scan(&sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return showings.Showing{}, errors.New("no_active_showing")
	}
	if err != nil {
		return showings.Showing{}, err
	}
	return showings.LoadBySession(ctx, pool, sessionID)
}

// UpdateForShowCode is the Director-facing write path: resolve shortCode's
// currently live/rehearsal Showing (Kernel 92 Showtime's own selector, spec
// §25) and upsert its Audience projection config.
func UpdateForShowCode(ctx context.Context, pool *pgxpool.Pool, actorUserID, shortCode string, diceOn, presenceOn, healthOn bool) (Config, error) {
	showing, err := resolveManagingShowAndShowing(ctx, pool, actorUserID, shortCode)
	if err != nil {
		return Config{}, err
	}

	var c Config
	row := pool.QueryRow(ctx, `
		INSERT INTO audience_projection_configs (showing_id, show_dice_rolls, show_presence, show_health_statuses, updated_by_user_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (showing_id) DO UPDATE
		SET show_dice_rolls = EXCLUDED.show_dice_rolls,
		    show_presence = EXCLUDED.show_presence,
		    show_health_statuses = EXCLUDED.show_health_statuses,
		    updated_by_user_id = EXCLUDED.updated_by_user_id,
		    updated_at = NOW()
		RETURNING showing_id::text, show_dice_rolls, show_presence, show_health_statuses,
		          COALESCE(updated_by_user_id::text, ''), updated_at
	`, showing.ID, diceOn, presenceOn, healthOn, actorUserID)
	if err := row.Scan(&c.ShowingID, &c.ShowDiceRolls, &c.ShowPresence, &c.ShowHealthStatuses, &c.UpdatedByUserID, &c.UpdatedAt); err != nil {
		return Config{}, err
	}
	return c, nil
}

// ForShowCode is the Director-facing read path, mirroring UpdateForShowCode's
// Showing resolution.
func ForShowCode(ctx context.Context, pool *pgxpool.Pool, actorUserID, shortCode string) (Config, error) {
	showing, err := resolveManagingShowAndShowing(ctx, pool, actorUserID, shortCode)
	if err != nil {
		return Config{}, err
	}
	return ForShowing(ctx, pool, showing.ID)
}
