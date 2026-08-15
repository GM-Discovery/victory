package socio

// Kernel 88 §13: a small nested pending-action stack for Interrupt/Help
// resolution. LIFO ordering is enforced entirely by validation order, not a
// queue structure: OpenInterrupt requires its parent to still be 'open',
// and ResolveInterrupt/CancelPendingAction require a node's own children to
// already be resolved/cancelled -- so the stack can only ever be unwound
// from the top down. Primary-action authority is Player-declared, gated on
// Current Turn (confirmed by product owner): the narrator/Director advances
// Current Turn (coordination.go), and the Character whose turn it is may
// have their owning Player open their own primary action; Director+ may
// also open on a Player's behalf. Interrupts themselves stay
// Director-adjudicated (spec §13.2: Director decides whether Help is
// fictionally possible and sets target complexity) -- only the primary
// action's opening authority moved to the Player.

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
	"victory/backend/internal/venuecoordination"
)

type PendingAction struct {
	ID                   string     `json:"id"`
	ShowID               string     `json:"show_id"`
	SessionID            string     `json:"session_id,omitempty"`
	ParentID             string     `json:"parent_id,omitempty"`
	Kind                 string     `json:"kind"`
	ActorCharacterCardID string     `json:"actor_character_card_id"`
	Title                string     `json:"title,omitempty"`
	SkillID              string     `json:"skill_id,omitempty"`
	TargetComplexity     *int       `json:"target_complexity,omitempty"`
	Status               string     `json:"status"`
	RollActionID         string     `json:"roll_action_id,omitempty"`
	RollTotal            *int       `json:"roll_total,omitempty"`
	OverageBonus         int        `json:"overage_bonus"`
	OverageConsumed      bool       `json:"overage_consumed"`
	OpenedByUserID       string     `json:"opened_by_user_id"`
	OpenedAt             time.Time  `json:"opened_at"`
	ResolvedAt           *time.Time `json:"resolved_at,omitempty"`
}

const pendingActionColumns = `
	id::text, show_id::text, COALESCE(session_id::text, ''), COALESCE(parent_id::text, ''),
	kind, actor_character_card_id::text, title, COALESCE(skill_id, ''), target_complexity,
	status, COALESCE(roll_action_id::text, ''), roll_total, overage_bonus,
	overage_consumed_at IS NOT NULL, opened_by_user_id::text, opened_at, resolved_at
`

func scanPendingAction(row pgx.Row) (PendingAction, error) {
	var a PendingAction
	if err := row.Scan(
		&a.ID, &a.ShowID, &a.SessionID, &a.ParentID,
		&a.Kind, &a.ActorCharacterCardID, &a.Title, &a.SkillID, &a.TargetComplexity,
		&a.Status, &a.RollActionID, &a.RollTotal, &a.OverageBonus,
		&a.OverageConsumed, &a.OpenedByUserID, &a.OpenedAt, &a.ResolvedAt,
	); err != nil {
		return PendingAction{}, err
	}
	return a, nil
}

func loadPendingAction(ctx context.Context, pool *pgxpool.Pool, id string) (PendingAction, error) {
	row := pool.QueryRow(ctx, `SELECT `+pendingActionColumns+` FROM socio_pending_actions WHERE id = $1`, id)
	a, err := scanPendingAction(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PendingAction{}, errors.New("pending_action_not_found")
		}
		return PendingAction{}, err
	}
	return a, nil
}

// requireCurrentTurnOrManageAuthority checks that actorUserID either holds
// Director+/Producer/Operator authority on showID, or currently holds
// Current Turn for cohortID (via the same venuecoordination state
// coordination.go writes) *and* owns characterCardID on their own roster
// selection -- the Player-declares-primary-action authority shape.
func requireCurrentTurnOrManageAuthority(ctx context.Context, pool *pgxpool.Pool, reg *venuecoordination.Registry, actorUserID, showID, cohortID, characterCardID string) error {
	if err := requireShowManageAuthority(ctx, pool, actorUserID, showID); err == nil {
		return nil
	}

	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return errors.New("not_authenticated")
	}
	if reg == nil {
		return errors.New("not_authorized")
	}
	state, active := reg.Get(SocioVenueSessionID(showID, cohortID))
	if !active || state.CurrentTurnUserID == "" || state.CurrentTurnUserID != actorUserID {
		return errors.New("not_authorized")
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

// OpenPrimaryAction opens a new top-of-stack primary action for
// characterCardID. Authority: that Character's owning Player, but only
// while their Character currently holds Current Turn for cohortID
// (requireCurrentTurnOrManageAuthority) -- or Director+ opening on a
// Player's behalf as a fallback.
func OpenPrimaryAction(ctx context.Context, pool *pgxpool.Pool, reg *venuecoordination.Registry, actorUserID, showID, sessionID, cohortID, characterCardID, title string) (PendingAction, error) {
	if err := requireCurrentTurnOrManageAuthority(ctx, pool, reg, actorUserID, showID, cohortID, characterCardID); err != nil {
		return PendingAction{}, err
	}
	characterCardID = strings.TrimSpace(characterCardID)
	if characterCardID == "" {
		return PendingAction{}, errors.New("character_card_id_required")
	}

	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO socio_pending_actions
			(show_id, session_id, kind, actor_character_card_id, title, opened_by_user_id)
		VALUES ($1, NULLIF($2, '')::uuid, 'primary', $3, $4, $5)
		RETURNING id::text
	`, showID, sessionID, characterCardID, strings.TrimSpace(title), actorUserID).Scan(&id)
	if err != nil {
		return PendingAction{}, err
	}
	return loadPendingAction(ctx, pool, id)
}

// OpenInterrupt attaches a Help/interrupt action to parentID -- the
// interrupt's own parent must still be 'open' (this is what keeps the
// stack LIFO: you can only interrupt the thing currently at the top).
// Director+-only: the Director decides whether Help is fictionally
// possible and sets the target complexity (spec §13.2).
func OpenInterrupt(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, sessionID, parentID, helperCharacterCardID, title string, targetComplexity int) (PendingAction, error) {
	if err := requireShowManageAuthority(ctx, pool, actorUserID, showID); err != nil {
		return PendingAction{}, err
	}
	helperCharacterCardID = strings.TrimSpace(helperCharacterCardID)
	if helperCharacterCardID == "" {
		return PendingAction{}, errors.New("character_card_id_required")
	}
	parent, err := loadPendingAction(ctx, pool, parentID)
	if err != nil {
		return PendingAction{}, err
	}
	if parent.Status != "open" {
		return PendingAction{}, errors.New("parent_not_open")
	}

	var id string
	err = pool.QueryRow(ctx, `
		INSERT INTO socio_pending_actions
			(show_id, session_id, parent_id, kind, actor_character_card_id, title, target_complexity, opened_by_user_id)
		VALUES ($1, NULLIF($2, '')::uuid, $3, 'interrupt', $4, $5, $6, $7)
		RETURNING id::text
	`, showID, sessionID, parentID, helperCharacterCardID, strings.TrimSpace(title), targetComplexity, actorUserID).Scan(&id)
	if err != nil {
		return PendingAction{}, err
	}
	return loadPendingAction(ctx, pool, id)
}

// hasOpenChildren reports whether id has any 'open' child rows -- resolving
// or cancelling a node before its children are settled would break LIFO
// ordering, so both ResolveInterrupt and CancelPendingAction require this
// to be false first.
func hasOpenChildren(ctx context.Context, pool *pgxpool.Pool, id string) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM socio_pending_actions WHERE parent_id = $1 AND status = 'open')
	`, id).Scan(&ok)
	return ok, err
}

// ResolveInterrupt resolves an interrupt against its target complexity.
// Success (rollTotal >= target_complexity) computes overage = rollTotal -
// target_complexity, stored for the parent action's own roll to consume
// later (§7's "Director situational modifier" delivery mechanism); a
// failed roll resolves with overage_bonus = 0 and adds no bonus (spec
// §13.5). This never auto-resolves the parent -- the primary action's own
// roll is what pulls the accumulated overage in (see ConsumeOverageFor).
// Authority: Director+ or the helper's own owning Player.
func ResolveInterrupt(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, interruptID, rollActionID string, rollTotal int) (PendingAction, error) {
	action, err := loadPendingAction(ctx, pool, interruptID)
	if err != nil {
		return PendingAction{}, err
	}
	if action.Kind != "interrupt" {
		return PendingAction{}, errors.New("not_an_interrupt")
	}
	if err := requireOwnerOrShowCharacterAuthority(ctx, pool, actorUserID, showID, action.ActorCharacterCardID); err != nil {
		return PendingAction{}, err
	}
	if action.Status != "open" {
		return PendingAction{}, errors.New("interrupt_not_open")
	}
	if open, err := hasOpenChildren(ctx, pool, interruptID); err != nil {
		return PendingAction{}, err
	} else if open {
		return PendingAction{}, errors.New("interrupt_has_open_children")
	}
	if action.TargetComplexity == nil {
		return PendingAction{}, errors.New("target_complexity_not_set")
	}

	overage := 0
	if rollTotal >= *action.TargetComplexity {
		overage = rollTotal - *action.TargetComplexity
	}

	_, err = pool.Exec(ctx, `
		UPDATE socio_pending_actions
		SET status = 'resolved', roll_action_id = NULLIF($2, '')::uuid, roll_total = $3,
		    overage_bonus = $4, resolved_at = NOW()
		WHERE id = $1
	`, interruptID, rollActionID, rollTotal, overage)
	if err != nil {
		return PendingAction{}, err
	}
	return loadPendingAction(ctx, pool, interruptID)
}

// CancelPendingAction cancels a node (and, since a cancelled parent leaves
// its children orphaned mid-resolution, cascades cancellation to any still-
// open descendants). Director+ only.
func CancelPendingAction(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, id string) error {
	if err := requireShowManageAuthority(ctx, pool, actorUserID, showID); err != nil {
		return err
	}
	_, err := pool.Exec(ctx, `
		WITH RECURSIVE descendants AS (
			SELECT id FROM socio_pending_actions WHERE id = $1
			UNION ALL
			SELECT p.id FROM socio_pending_actions p
			JOIN descendants d ON p.parent_id = d.id
		)
		UPDATE socio_pending_actions
		SET status = 'cancelled', resolved_at = NOW()
		WHERE id IN (SELECT id FROM descendants) AND status = 'open'
	`, id)
	return err
}

// ConsumeOverageFor sums unconsumed overage_bonus from primaryActionID's
// resolved child interrupts and marks them consumed -- called once, at the
// moment the primary action's own roll is made, so a bonus is never applied
// twice (migration 103's overage_consumed_at column).
func ConsumeOverageFor(ctx context.Context, pool *pgxpool.Pool, primaryActionID string) (int, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, overage_bonus FROM socio_pending_actions
		WHERE parent_id = $1 AND status = 'resolved' AND overage_consumed_at IS NULL AND overage_bonus > 0
	`, primaryActionID)
	if err != nil {
		return 0, err
	}
	var ids []string
	total := 0
	for rows.Next() {
		var id string
		var bonus int
		if err := rows.Scan(&id, &bonus); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, id)
		total += bonus
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	if _, err := pool.Exec(ctx, `
		UPDATE socio_pending_actions SET overage_consumed_at = NOW() WHERE id = ANY($1::uuid[])
	`, ids); err != nil {
		return 0, err
	}
	return total, nil
}

// LoadOpenPrimaryActionForRoll loads a pending action that a canonical roll
// is about to resolve -- exported for actions.StorePlayerMechanicRoll,
// which needs to confirm the primary action belongs to the same Character
// making the roll before consuming its banked overage (kernel-88 §7 step
// 4, §13's overage-delivery mechanism).
func LoadOpenPrimaryActionForRoll(ctx context.Context, pool *pgxpool.Pool, id string) (PendingAction, error) {
	action, err := loadPendingAction(ctx, pool, id)
	if err != nil {
		return PendingAction{}, err
	}
	if action.Kind != "primary" {
		return PendingAction{}, errors.New("not_a_primary_action")
	}
	if action.Status != "open" {
		return PendingAction{}, errors.New("primary_action_not_open")
	}
	return action, nil
}

// ListOpenStack returns every non-resolved/cancelled pending action for a
// Show, for the Director's stack view.
func ListOpenStack(ctx context.Context, pool *pgxpool.Pool, showID string) ([]PendingAction, error) {
	rows, err := pool.Query(ctx, `
		SELECT `+pendingActionColumns+` FROM socio_pending_actions
		WHERE show_id = $1 AND status = 'open'
		ORDER BY opened_at ASC
	`, showID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PendingAction{}
	for rows.Next() {
		a, err := scanPendingAction(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
