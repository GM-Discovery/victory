package storyboards

// Authority helpers, modeled on ewrite/authority.go and
// showruns/authority.go's shape (Operator short-circuit first), but with no
// location-role floor: Storyboard authority is purely per-board
// (ownership or an explicit storyboard_grants row), never location-scoped.
// Client-supplied role claims are never authority -- every helper here
// re-derives the tier itself from the database. See migration 090's
// comment and Construction/Storyboards/storyboards-permissions.md for the
// owner-vs-grant design rationale.

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
)

// ErrBoardNotFound is returned by resolveViewerTier when the board does
// not exist, so callers can distinguish "not found" from "no access"
// (both should render as 404 to a non-owner, non-Operator caller -- board
// existence itself is not for an unauthorized viewer to learn).
var ErrBoardNotFound = errors.New("storyboard_not_found")

// resolveViewerTier is the one place every authority and projection
// function calls to find out what a user may do on a board. It is the
// server-authoritative answer to "what tier is this viewer" -- nothing
// client-submitted ever substitutes for it.
func resolveViewerTier(ctx context.Context, pool *pgxpool.Pool, userID string, board *Storyboard) (string, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" || board == nil {
		return TierNone, nil
	}
	if ok, err := access.IsOperatorUser(ctx, pool, userID); err != nil {
		return TierNone, err
	} else if ok {
		return TierOperator, nil
	}
	if board.OwnerUserID == userID {
		return TierOwner, nil
	}

	var role string
	err := pool.QueryRow(ctx, `
		SELECT granted_role::text FROM storyboard_grants
		WHERE storyboard_id = $1 AND user_id = $2
	`, board.ID, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return TierNone, nil
	}
	if err != nil {
		return TierNone, err
	}
	return role, nil
}

// ResolveViewerTier is the exported form, used by http.go/ws.go/export.go
// once they already hold a loaded board and just need the tier.
func ResolveViewerTier(ctx context.Context, pool *pgxpool.Pool, userID string, board *Storyboard) (string, error) {
	return resolveViewerTier(ctx, pool, userID, board)
}

func tierAtLeastCrew(tier string) bool {
	switch tier {
	case TierOwner, TierOperator, TierProducer, TierDirector, TierCrew:
		return true
	default:
		return false
	}
}

func tierAtLeastDirector(tier string) bool {
	switch tier {
	case TierOwner, TierOperator, TierProducer, TierDirector:
		return true
	default:
		return false
	}
}

func tierIsOwnerOrOperator(tier string) bool {
	return tier == TierOwner || tier == TierOperator
}

// CanViewBoard: any tier with a resolved grant or ownership (Audience and
// Cast included, per spec 5.2 -- they may list/open/view permitted
// boards).
func CanViewBoard(ctx context.Context, pool *pgxpool.Pool, userID string, board *Storyboard) (bool, error) {
	tier, err := resolveViewerTier(ctx, pool, userID, board)
	if err != nil {
		return false, err
	}
	return tier != TierNone, nil
}

// CanEditCards: Crew+ may create/edit/move/reorder/delete cards and edit
// unlocked band labels (spec 5.3). Lock state is checked separately by the
// caller (cards.go/bands.go), not folded in here -- this answers "does this
// viewer's tier ever get to touch cards on this board," not "right now."
func CanEditCards(ctx context.Context, pool *pgxpool.Pool, userID string, board *Storyboard) (bool, error) {
	tier, err := resolveViewerTier(ctx, pool, userID, board)
	if err != nil {
		return false, err
	}
	return tierAtLeastCrew(tier), nil
}

// CanEditStructure: Director+ (director/producer/Operator) or owner may add/
// remove/reorder columns, rows, and bands; move rows between bands; lock/
// unlock bands and cards; change board metadata (spec 5.4).
func CanEditStructure(ctx context.Context, pool *pgxpool.Pool, userID string, board *Storyboard) (bool, error) {
	tier, err := resolveViewerTier(ctx, pool, userID, board)
	if err != nil {
		return false, err
	}
	return tierAtLeastDirector(tier), nil
}

// CanManageSharing: owner (or Operator) only -- default Kernel 80 posture
// is "only the owner manages Storyboard access grants" (spec 5.4).
func CanManageSharing(ctx context.Context, pool *pgxpool.Pool, userID string, board *Storyboard) (bool, error) {
	tier, err := resolveViewerTier(ctx, pool, userID, board)
	if err != nil {
		return false, err
	}
	return tierIsOwnerOrOperator(tier), nil
}

// CanArchiveOrDeleteBoard: owner (or Operator) only (spec 5.5).
func CanArchiveOrDeleteBoard(ctx context.Context, pool *pgxpool.Pool, userID string, board *Storyboard) (bool, error) {
	tier, err := resolveViewerTier(ctx, pool, userID, board)
	if err != nil {
		return false, err
	}
	return tierIsOwnerOrOperator(tier), nil
}

// CanExportBoard: owner or Director+ (spec 5.3-5.5 list export only under
// Director+/Owner capabilities -- Crew never exports). tierAtLeastDirector
// already includes TierOwner/TierOperator.
func CanExportBoard(ctx context.Context, pool *pgxpool.Pool, userID string, board *Storyboard) (bool, error) {
	tier, err := resolveViewerTier(ctx, pool, userID, board)
	if err != nil {
		return false, err
	}
	return tierAtLeastDirector(tier), nil
}

// hiddenCardsVisibleForTier: Crew+/owner/operator see hidden cards;
// Audience/Cast do not (spec 1.11, 9).
func hiddenCardsVisibleForTier(tier string) bool {
	return tierAtLeastCrew(tier)
}
