package storyboards

// HTTP handlers for Kernel 83's Group Leader/Current Turn mutation routes,
// plus attachLiveCoordination, the shared helper that enriches a
// BoardSnapshot with live Presence/Coordination data before it goes out
// over HTTP (http.go's HandleBoardItem GET) or WS (ws.go's watch_board
// snapshot). Same envelope/auth conventions as the rest of this package.

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/network"
	"victory/backend/internal/venuecoordination"
)

// attachLiveCoordination fills in snap.Presence and snap.Coordination from
// live hub/registry state. Best-effort: a failed identity lookup is
// logged and leaves the snapshot's board/column/card content intact
// rather than failing the whole request, since Presence/Coordination are
// supplementary live context, not the board itself.
func attachLiveCoordination(ctx context.Context, pool *pgxpool.Pool, hub *network.Hub, reg *venuecoordination.Registry, board *Storyboard, snap *BoardSnapshot) {
	if snap == nil || board == nil {
		return
	}
	presence, err := BuildPresenceRoster(ctx, pool, hub, board)
	if err != nil {
		log.Printf("storyboards: attachLiveCoordination: presence roster for %s: %v", board.ID, err)
	} else {
		snap.Presence = presence
	}
	view, err := BuildCoordinationView(ctx, pool, hub, reg, board)
	if err != nil {
		log.Printf("storyboards: attachLiveCoordination: coordination view for %s: %v", board.ID, err)
		return
	}
	snap.Coordination = &view
}

// broadcastCoordinationChanged and broadcastPresenceChanged notify every
// current watcher of boardID that live session state changed. Both event
// types are plain, unfiltered broadcasts (network.Hub.BroadcastBoardWatchers)
// rather than the per-viewer emitBoardEvent path in events.go -- neither
// Presence nor Group Leader/Current Turn is ever hidden-from-audience
// content (spec 3.4, 9), so there is no per-viewer shaping to do. The
// frontend's existing "any storyboard/* event -> reload the snapshot"
// handling (socket.js) needs no new client-side code for these.
func broadcastCoordinationChanged(ctx context.Context, pool *pgxpool.Pool, hub *network.Hub, boardID string, view CoordinationView) {
	if hub == nil {
		return
	}
	msg, err := json.Marshal(map[string]any{
		"type":     EventCoordinationChanged,
		"board_id": boardID,
		"data":     view,
	})
	if err != nil {
		log.Printf("storyboards: broadcastCoordinationChanged: marshal: %v", err)
		return
	}
	hub.BroadcastBoardWatchers(boardID, msg)
}

func broadcastPresenceChanged(hub *network.Hub, boardID string, roster []PresenceEntry) {
	if hub == nil {
		return
	}
	msg, err := json.Marshal(map[string]any{
		"type":     EventPresenceChanged,
		"board_id": boardID,
		"data":     roster,
	})
	if err != nil {
		log.Printf("storyboards: broadcastPresenceChanged: marshal: %v", err)
		return
	}
	hub.BroadcastBoardWatchers(boardID, msg)
}

func coordinationHandler(pool *pgxpool.Pool, hub *network.Hub, reg *venuecoordination.Registry, isLeader bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		board, err := LoadBoard(ctx, pool, boardID)
		if err != nil {
			writeError(w, err)
			return
		}
		// A viewer with no board access at all must not learn about, or
		// affect, live coordination state -- resolve viewer tier the same
		// way every other mutation route in this package does before
		// going anywhere near assignment authority.
		if ok, err := CanViewBoard(ctx, pool, userID, board); err != nil {
			writeError(w, err)
			return
		} else if !ok {
			writeError(w, ErrBoardNotFound)
			return
		}

		var body struct {
			TargetUserID string `json:"target_user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}

		if isLeader {
			_, err = AssignGroupLeader(ctx, pool, hub, reg, board, userID, body.TargetUserID)
		} else {
			_, err = AssignCurrentTurn(ctx, pool, hub, reg, board, userID, body.TargetUserID)
		}
		if err != nil {
			writeError(w, err)
			return
		}

		view, err := BuildCoordinationView(ctx, pool, hub, reg, board)
		if err != nil {
			writeError(w, err)
			return
		}
		broadcastCoordinationChanged(ctx, pool, hub, board.ID, view)
		writeOK(w, map[string]any{"coordination": view})
	}
}

// HandleCoordinationGroupLeader serves POST
// /api/storyboards/{board_id}/coordination/group-leader.
func HandleCoordinationGroupLeader(pool *pgxpool.Pool, hub *network.Hub, reg *venuecoordination.Registry) http.HandlerFunc {
	return coordinationHandler(pool, hub, reg, true)
}

// HandleCoordinationCurrentTurn serves POST
// /api/storyboards/{board_id}/coordination/current-turn.
func HandleCoordinationCurrentTurn(pool *pgxpool.Pool, hub *network.Hub, reg *venuecoordination.Registry) http.HandlerFunc {
	return coordinationHandler(pool, hub, reg, false)
}
