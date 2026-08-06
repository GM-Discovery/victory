package storyboards

// ProjectBoardSnapshot is the single composer of what a given viewer sees
// on a board, mirroring world.LoadVenueSnapshot's role in venue rendering.
// Both the HTTP snapshot/list endpoints and the WS connect/reconnect path
// (spec 8.4) funnel through this -- there is exactly one hidden-card
// filtering implementation in the whole package (spec 1.11, 9): for
// Audience/Cast tiers, hidden cards are omitted entirely from the result,
// not marked-empty and not counted, so neither their content nor their
// existence leaks through card counts or cell positions. export.go reuses
// this too rather than reimplementing the filter.

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// BoardSnapshot is the full authoritative state of one board, already
// filtered for one specific viewer.
type BoardSnapshot struct {
	Board      Storyboard         `json:"board"`
	ViewerTier string             `json:"viewer_tier"`
	Columns    []StoryboardColumn `json:"columns"`
	Bands      []StoryboardBand   `json:"bands"`
	Rows       []StoryboardRow    `json:"rows"`
	Cards      []StoryboardCard   `json:"cards"`
}

// ProjectBoardSnapshot requires the viewer to have some tier of access
// (CanViewBoard's equivalent, resolved internally); ErrNotAuthorized if
// not.
func ProjectBoardSnapshot(ctx context.Context, pool *pgxpool.Pool, userID string, board *Storyboard) (*BoardSnapshot, error) {
	tier, err := resolveViewerTier(ctx, pool, userID, board)
	if err != nil {
		return nil, err
	}
	if tier == TierNone {
		return nil, ErrNotAuthorized
	}
	return projectBoardSnapshotForTier(ctx, pool, tier, board)
}

// projectBoardSnapshotForTier is split out from ProjectBoardSnapshot so
// export.go (which has already run its own CanExportBoard check against
// the exporter's own tier) can request a projection without a second
// redundant tier resolution.
func projectBoardSnapshotForTier(ctx context.Context, pool *pgxpool.Pool, tier string, board *Storyboard) (*BoardSnapshot, error) {
	columns, err := ListColumns(ctx, pool, board.ID)
	if err != nil {
		return nil, err
	}
	bands, err := ListBands(ctx, pool, board.ID)
	if err != nil {
		return nil, err
	}
	rows, err := ListRows(ctx, pool, board.ID)
	if err != nil {
		return nil, err
	}
	cards, err := ListCardsForBoard(ctx, pool, board.ID)
	if err != nil {
		return nil, err
	}

	if !hiddenCardsVisibleForTier(tier) {
		visible := make([]StoryboardCard, 0, len(cards))
		for _, c := range cards {
			if !c.HiddenFromAudience {
				visible = append(visible, c)
			}
		}
		cards = visible
	}

	return &BoardSnapshot{
		Board:      *board,
		ViewerTier: tier,
		Columns:    columns,
		Bands:      bands,
		Rows:       rows,
		Cards:      cards,
	}, nil
}
