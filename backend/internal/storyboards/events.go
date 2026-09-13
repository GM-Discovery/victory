package storyboards

// Live WS events (spec 8.2, 6). All server-authored, all delivered
// per-viewer through network.Hub.SendToBoardWatcher rather than a single
// Hub.Broadcast -- the Hub itself does no role filtering (a standing
// invariant of that package), so every event that could touch a hidden
// card is shaped once per watcher, right here, before it ever reaches the
// Hub. Non-card structural events carry the same payload to every tier
// (columns/bands/rows/board metadata are never hidden-from-audience
// content in Kernel 80); card events are the one category that can differ
// per viewer.

import (
	"context"
	"encoding/json"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/network"
)

const (
	EventBoardMetadataChanged = "storyboard/metadata_changed"
	EventGrantChanged         = "storyboard/grant_changed"
	EventColumnAdded          = "storyboard/column_added"
	EventColumnUpdated        = "storyboard/column_updated"
	EventColumnsReordered     = "storyboard/columns_reordered"
	EventColumnRemoved        = "storyboard/column_removed"
	EventBandAdded            = "storyboard/band_added"
	EventBandUpdated          = "storyboard/band_updated"
	EventBandsReordered       = "storyboard/bands_reordered"
	EventBandRemoved          = "storyboard/band_removed"
	EventBandLockChanged      = "storyboard/band_lock_changed"
	EventRowAdded             = "storyboard/row_added"
	EventRowUpdated           = "storyboard/row_updated"
	EventRowsReordered        = "storyboard/rows_reordered"
	EventRowMoved             = "storyboard/row_moved"
	EventRowRemoved           = "storyboard/row_removed"
	EventCardAdded            = "storyboard/card_added"
	EventCardUpdated          = "storyboard/card_updated"
	EventCardMoved            = "storyboard/card_moved"
	EventCardSwapped          = "storyboard/card_swapped"
	EventCardsReordered       = "storyboard/cards_reordered"
	EventCardRemoved          = "storyboard/card_removed"
	EventCardLockChanged      = "storyboard/card_lock_changed"
	EventBoardArchived        = "storyboard/archived"
	// EventReferencePanelChanged (Kernel 82) covers every Reference Panel
	// mutation (field add/rename/reorder/type-change/remove, item add/
	// edit/reorder/remove) with one event type -- panel content is never
	// hidden-from-audience (spec 3.4 gates editing by role, not viewing),
	// so there is no per-viewer shaping to do, unlike card events.
	EventReferencePanelChanged = "storyboard/reference_panel_changed"
	// EventCoordinationChanged and EventPresenceChanged (Kernel 83) cover
	// Group Leader/Current Turn assignment and Presence Tray roster
	// changes respectively. Both are broadcast via
	// network.Hub.BroadcastBoardWatchers, not emitBoardEvent -- see
	// coordination_http.go's broadcastCoordinationChanged/
	// broadcastPresenceChanged.
	EventCoordinationChanged = "storyboard/coordination_changed"
	EventPresenceChanged     = "storyboard/presence_changed"
)

// emitBoardEvent resolves every current watcher of boardID, calls build
// once per watcher with that watcher's own server-resolved tier, and
// delivers the result only to that watcher -- build returning nil means
// "send this watcher nothing at all" (not a redacted stub), which is what
// keeps a hidden card's existence, count, and position from leaking via a
// content-free "something changed" ping (spec 1.11, 9). A failure loading
// the board or resolving a watcher's tier is logged and that watcher is
// skipped, never treated as fatal to the others.
func emitBoardEvent(ctx context.Context, pool *pgxpool.Pool, hub *network.Hub, boardID, eventType string, build func(tier string) any) {
	if hub == nil {
		return
	}
	board, err := LoadBoard(ctx, pool, boardID)
	if err != nil {
		log.Printf("storyboards: emitBoardEvent: load board %s: %v", boardID, err)
		return
	}
	for _, userID := range hub.BoardWatcherUserIDs(boardID) {
		tier, err := resolveViewerTier(ctx, pool, userID, board)
		if err != nil {
			log.Printf("storyboards: emitBoardEvent: resolve tier for %s: %v", userID, err)
			continue
		}
		if tier == TierNone {
			continue
		}
		data := build(tier)
		if data == nil {
			continue
		}
		msg, err := json.Marshal(map[string]any{
			"type":     eventType,
			"board_id": boardID,
			"data":     data,
		})
		if err != nil {
			log.Printf("storyboards: emitBoardEvent: marshal %s: %v", eventType, err)
			continue
		}
		hub.SendToBoardWatcher(boardID, userID, msg)
	}
}

// broadcastGenericBoardEvent delivers the same payload to every watcher
// with any tier of access -- for board metadata, columns, bands, and rows,
// which are never hidden-from-audience content in Kernel 80.
func broadcastGenericBoardEvent(ctx context.Context, pool *pgxpool.Pool, hub *network.Hub, boardID, eventType string, data any) {
	emitBoardEvent(ctx, pool, hub, boardID, eventType, func(string) any { return data })
}

// broadcastCardEvent is the one event category that can differ per
// viewer: if card is hidden-from-audience, Audience/Cast watchers receive
// nothing at all for this event (spec 1.11's "moving a hidden card does
// not broadcast content to viewers"); Crew+ receive the full card.
func broadcastCardEvent(ctx context.Context, pool *pgxpool.Pool, hub *network.Hub, boardID, eventType string, card *StoryboardCard) {
	emitBoardEvent(ctx, pool, hub, boardID, eventType, func(tier string) any {
		if card != nil && card.HiddenFromAudience && !hiddenCardsVisibleForTier(tier) {
			return nil
		}
		return card
	})
}

// broadcastCellReorder is the one structural-looking event that still
// needs per-viewer filtering: the ordered_card_ids list itself would leak
// a hidden card's ID and position to Audience/Cast watchers if sent
// verbatim, even though EventCardsReordered never carries card content.
// Audience/Cast watchers instead receive the same list with any hidden
// card IDs removed (positions closing up around the gap), matching the
// spec 1.11 rule that hidden cards must not leak through position either.
func broadcastCellReorder(ctx context.Context, pool *pgxpool.Pool, hub *network.Hub, boardID, rowID, columnID string, orderedCardIDs []string) {
	if hub == nil {
		return
	}
	hidden := map[string]bool{}
	rows, err := pool.Query(ctx, `SELECT id::text, hidden_from_audience FROM storyboard_cards WHERE id = ANY($1)`, orderedCardIDs)
	if err == nil {
		for rows.Next() {
			var id string
			var h bool
			if scanErr := rows.Scan(&id, &h); scanErr == nil && h {
				hidden[id] = true
			}
		}
		rows.Close()
	}

	emitBoardEvent(ctx, pool, hub, boardID, EventCardsReordered, func(tier string) any {
		ids := orderedCardIDs
		if !hiddenCardsVisibleForTier(tier) && len(hidden) > 0 {
			filtered := make([]string, 0, len(orderedCardIDs))
			for _, id := range orderedCardIDs {
				if !hidden[id] {
					filtered = append(filtered, id)
				}
			}
			ids = filtered
		}
		return map[string]any{"row_id": rowID, "column_id": columnID, "ordered_card_ids": ids}
	})
}

// broadcastGrantChanged is delivered to the board owner's own connections
// only -- grant *content* is a sharing-management detail; the affected
// user simply finds their own authority different on their next action or
// reconnect rather than receiving a live event about their own grant
// (spec 8.2's grant_changed entry, as scoped in
// Construction/Domains/Storyboards/storyboards-live-events.md).
func broadcastGrantChanged(ctx context.Context, pool *pgxpool.Pool, hub *network.Hub, board *Storyboard) {
	if hub == nil || board == nil {
		return
	}
	msg, err := json.Marshal(map[string]any{
		"type":     EventGrantChanged,
		"board_id": board.ID,
		"data":     map[string]any{},
	})
	if err != nil {
		return
	}
	hub.SendToBoardWatcher(board.ID, board.OwnerUserID, msg)
}
