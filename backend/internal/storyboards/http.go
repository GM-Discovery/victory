package storyboards

// HTTP handlers for /api/storyboards/*. Envelope, auth, and error
// conventions copied from ewrite/http.go and showruns/http.go. Every
// mutation resolves userID from the session cookie via
// requireAuthenticatedUser and passes it straight into the domain layer
// (boards.go/grants.go/columns.go/bands.go/rows.go/cards.go), which
// re-derives the caller's tier itself (authority.go's resolveViewerTier)
// -- no handler here ever reads a role/tier from the request body, so
// there is no field a forged request could populate to change what a
// caller is allowed to do.

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
	"victory/backend/internal/assets"
	"victory/backend/internal/network"
	"victory/backend/internal/venuecoordination"
)

type response struct {
	Ok   bool `json:"ok"`
	Data any  `json:"data,omitempty"`
}

func context10(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 10*time.Second)
}

func requireAuthenticatedUser(ctx context.Context, pool *pgxpool.Pool, r *http.Request) (string, error) {
	sessionCookie := ""
	if c, err := r.Cookie("victory_session"); err == nil {
		sessionCookie = c.Value
	}
	userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
	if err != nil || strings.TrimSpace(userID) == "" {
		return "", errors.New("not_authenticated")
	}
	return userID, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, response{Ok: true, Data: data})
}

func writeError(w http.ResponseWriter, err error) {
	code := err.Error()
	status := http.StatusBadRequest
	switch {
	case code == "not_authenticated":
		status = http.StatusUnauthorized
	case code == "not_authorized":
		status = http.StatusForbidden
	case strings.HasSuffix(code, "_not_found"):
		status = http.StatusNotFound
	case strings.HasSuffix(code, "_version_conflict"):
		status = http.StatusConflict
	case strings.HasSuffix(code, "_occupied"):
		status = http.StatusConflict
	case strings.HasSuffix(code, "_locked"):
		status = http.StatusLocked
	}
	writeJSON(w, status, response{Ok: false, Data: map[string]any{"error": code}})
}

// --- Boards ---

// HandleBoards serves GET (list owned+shared) and POST (create) at
// /api/storyboards.
func HandleBoards(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		switch r.Method {
		case http.MethodGet:
			owned, err := ListOwnedBoards(ctx, pool, userID)
			if err != nil {
				writeError(w, err)
				return
			}
			shared, err := ListSharedBoards(ctx, pool, userID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"owned": owned, "shared": shared})
		case http.MethodPost:
			var body struct {
				Title        string   `json:"title"`
				Description  string   `json:"description"`
				Mode         string   `json:"mode"`
				ColumnTitles []string `json:"column_titles"`
				BandLabel    string   `json:"band_label"`
				RowLabels    []string `json:"row_labels"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_json"))
				return
			}
			// Kernel 82: mode selects which built-in template to
			// instantiate from. Empty/"blank" is unchanged Kernel 80
			// behavior; "timeline" is the only other built-in mode.
			var board *Storyboard
			var err error
			switch strings.ToLower(strings.TrimSpace(body.Mode)) {
			case "", ModeBlank:
				board, err = CreateBoard(ctx, pool, userID, body.Title, body.Description, body.ColumnTitles, body.BandLabel, body.RowLabels)
			case ModeTimeline:
				board, err = CreateTimelineBoard(ctx, pool, userID, body.Title, body.Description)
			default:
				err = errors.New("mode_invalid")
			}
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"board": board})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
		}
	}
}

// HandleBoardItem serves GET (full filtered snapshot), PATCH (metadata),
// and DELETE at /api/storyboards/{board_id}.
func HandleBoardItem(pool *pgxpool.Pool, hub *network.Hub, reg *venuecoordination.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		switch r.Method {
		case http.MethodGet:
			board, err := LoadBoard(ctx, pool, boardID)
			if err != nil {
				writeError(w, err)
				return
			}
			snap, err := ProjectBoardSnapshot(ctx, pool, userID, board)
			if err != nil {
				writeError(w, err)
				return
			}
			attachLiveCoordination(ctx, pool, hub, reg, board, snap)
			writeOK(w, snap)
		case http.MethodPatch:
			var body struct {
				Title       string `json:"title"`
				Description string `json:"description"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_json"))
				return
			}
			board, err := UpdateBoardMetadata(ctx, pool, userID, boardID, body.Title, body.Description)
			if err != nil {
				writeError(w, err)
				return
			}
			broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventBoardMetadataChanged, board)
			writeOK(w, map[string]any{"board": board})
		case http.MethodDelete:
			if err := DeleteBoard(ctx, pool, userID, boardID); err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"deleted": true})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
		}
	}
}

// HandleBoardArchive serves POST /api/storyboards/{board_id}/archive with
// body {"archived": true|false}.
func HandleBoardArchive(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		var body struct {
			Archived bool `json:"archived"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		var board *Storyboard
		if body.Archived {
			board, err = ArchiveBoard(ctx, pool, userID, boardID)
		} else {
			board, err = UnarchiveBoard(ctx, pool, userID, boardID)
		}
		if err != nil {
			writeError(w, err)
			return
		}
		broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventBoardArchived, map[string]any{"archived": body.Archived})
		writeOK(w, map[string]any{"board": board})
	}
}

// --- Grants ---

// HandleGrants serves GET (list) and POST (add) at
// /api/storyboards/{board_id}/grants.
func HandleGrants(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		switch r.Method {
		case http.MethodGet:
			grants, err := ListGrants(ctx, pool, userID, boardID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"grants": grants})
		case http.MethodPost:
			var body struct {
				ProfileID   string `json:"profile_id"`
				UserHandle  string `json:"user_handle"`
				GrantedRole string `json:"granted_role"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_json"))
				return
			}
			// Kernel 85 §7.4: profile_id (the People Picker's canonical
			// identity) takes precedence when present. user_handle stays
			// working for backward compat / a manual ID-only fallback field
			// -- it was never resolvable from what the sharer could actually
			// see (My People shows stage names, not handles), so profile_id
			// is the path every UI selection now takes.
			var g *StoryboardGrant
			var err error
			if strings.TrimSpace(body.ProfileID) != "" {
				g, err = AddGrantByProfile(ctx, pool, userID, boardID, body.ProfileID, body.GrantedRole)
			} else {
				g, err = AddGrant(ctx, pool, userID, boardID, body.UserHandle, body.GrantedRole)
			}
			if err != nil {
				writeError(w, err)
				return
			}
			if board, loadErr := LoadBoard(ctx, pool, boardID); loadErr == nil {
				broadcastGrantChanged(ctx, pool, hub, board)
			}
			writeOK(w, map[string]any{"grant": g})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
		}
	}
}

// HandleGrantItem serves DELETE /api/storyboards/{board_id}/grants/{grant_id}.
func HandleGrantItem(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		grantID := strings.TrimSpace(r.PathValue("grant_id"))
		if err := RemoveGrant(ctx, pool, userID, boardID, grantID); err != nil {
			writeError(w, err)
			return
		}
		if board, loadErr := LoadBoard(ctx, pool, boardID); loadErr == nil {
			broadcastGrantChanged(ctx, pool, hub, board)
		}
		writeOK(w, map[string]any{"deleted": true})
	}
}

// --- Columns ---

// HandleColumns serves POST (create) at
// /api/storyboards/{board_id}/columns.
func HandleColumns(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		var body struct {
			Title string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		col, err := AddColumn(ctx, pool, userID, boardID, body.Title)
		if err != nil {
			writeError(w, err)
			return
		}
		broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventColumnAdded, col)
		writeOK(w, map[string]any{"column": col})
	}
}

// HandleColumnItem serves PATCH (rename) and DELETE (remove, with
// ?resolution=delete_cards|move_cards&target_column_id=... for an occupied
// column) at /api/storyboards/{board_id}/columns/{column_id}.
func HandleColumnItem(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		columnID := strings.TrimSpace(r.PathValue("column_id"))
		switch r.Method {
		case http.MethodPatch:
			var body struct {
				Title string `json:"title"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_json"))
				return
			}
			col, err := RenameColumn(ctx, pool, userID, boardID, columnID, body.Title)
			if err != nil {
				writeError(w, err)
				return
			}
			broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventColumnUpdated, col)
			writeOK(w, map[string]any{"column": col})
		case http.MethodDelete:
			resolution := r.URL.Query().Get("resolution")
			targetColumnID := r.URL.Query().Get("target_column_id")
			if err := RemoveColumn(ctx, pool, userID, boardID, columnID, resolution, targetColumnID); err != nil {
				writeError(w, err)
				return
			}
			broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventColumnRemoved, map[string]any{"column_id": columnID})
			writeOK(w, map[string]any{"deleted": true})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
		}
	}
}

// HandleColumnsReorder serves POST
// /api/storyboards/{board_id}/columns/reorder with body
// {"ordered_column_ids": [...]}.
func HandleColumnsReorder(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		var body struct {
			OrderedColumnIDs []string `json:"ordered_column_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		if err := ReorderColumns(ctx, pool, userID, boardID, body.OrderedColumnIDs); err != nil {
			writeError(w, err)
			return
		}
		broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventColumnsReordered, map[string]any{"ordered_column_ids": body.OrderedColumnIDs})
		writeOK(w, map[string]any{"reordered": true})
	}
}

// --- Bands ---

// HandleBands serves POST (create) at /api/storyboards/{board_id}/bands.
func HandleBands(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		var body struct {
			Label string `json:"label"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		band, err := AddBand(ctx, pool, userID, boardID, body.Label)
		if err != nil {
			writeError(w, err)
			return
		}
		broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventBandAdded, band)
		writeOK(w, map[string]any{"band": band})
	}
}

// HandleBandItem serves PATCH (label/description -- Crew allowed while
// unlocked) and DELETE (remove, with
// ?resolution=delete_rows|move_rows&target_band_id=... for an occupied
// band) at /api/storyboards/{board_id}/bands/{band_id}.
func HandleBandItem(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		bandID := strings.TrimSpace(r.PathValue("band_id"))
		switch r.Method {
		case http.MethodPatch:
			var body struct {
				Label       string `json:"label"`
				Description string `json:"description"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_json"))
				return
			}
			band, err := UpdateBandLabel(ctx, pool, userID, boardID, bandID, body.Label, body.Description)
			if err != nil {
				writeError(w, err)
				return
			}
			broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventBandUpdated, band)
			writeOK(w, map[string]any{"band": band})
		case http.MethodDelete:
			resolution := r.URL.Query().Get("resolution")
			targetBandID := r.URL.Query().Get("target_band_id")
			if err := RemoveBand(ctx, pool, userID, boardID, bandID, resolution, targetBandID); err != nil {
				writeError(w, err)
				return
			}
			broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventBandRemoved, map[string]any{"band_id": bandID})
			writeOK(w, map[string]any{"deleted": true})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
		}
	}
}

// HandleBandLock serves POST /api/storyboards/{board_id}/bands/{band_id}/lock
// with body {"locked": true|false}.
func HandleBandLock(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		bandID := strings.TrimSpace(r.PathValue("band_id"))
		var body struct {
			Locked bool `json:"locked"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		band, err := SetBandLock(ctx, pool, userID, boardID, bandID, body.Locked)
		if err != nil {
			writeError(w, err)
			return
		}
		broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventBandLockChanged, band)
		writeOK(w, map[string]any{"band": band})
	}
}

// HandleBandCollapse serves POST
// /api/storyboards/{board_id}/bands/{band_id}/collapse with body
// {"collapsed": true|false}.
func HandleBandCollapse(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		bandID := strings.TrimSpace(r.PathValue("band_id"))
		var body struct {
			Collapsed bool `json:"collapsed"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		band, err := SetBandCollapsed(ctx, pool, userID, boardID, bandID, body.Collapsed)
		if err != nil {
			writeError(w, err)
			return
		}
		broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventBandUpdated, band)
		writeOK(w, map[string]any{"band": band})
	}
}

// HandleBandsReorder serves POST
// /api/storyboards/{board_id}/bands/reorder with body
// {"ordered_band_ids": [...]}.
func HandleBandsReorder(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		var body struct {
			OrderedBandIDs []string `json:"ordered_band_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		if err := ReorderBands(ctx, pool, userID, boardID, body.OrderedBandIDs); err != nil {
			writeError(w, err)
			return
		}
		broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventBandsReordered, map[string]any{"ordered_band_ids": body.OrderedBandIDs})
		writeOK(w, map[string]any{"reordered": true})
	}
}

// --- Rows ---

// HandleRows serves POST (create) at /api/storyboards/{board_id}/rows with
// body {"band_id": "...", "label": "..."}.
func HandleRows(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		var body struct {
			BandID string `json:"band_id"`
			Label  string `json:"label"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		row, err := AddRow(ctx, pool, userID, boardID, body.BandID, body.Label)
		if err != nil {
			writeError(w, err)
			return
		}
		broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventRowAdded, row)
		writeOK(w, map[string]any{"row": row})
	}
}

// HandleRowItem serves PATCH (rename) and DELETE (remove, with
// ?resolution=delete_cards|move_cards&target_row_id=... for an occupied
// row) at /api/storyboards/{board_id}/rows/{row_id}.
func HandleRowItem(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		rowID := strings.TrimSpace(r.PathValue("row_id"))
		switch r.Method {
		case http.MethodPatch:
			var body struct {
				Label       string `json:"label"`
				Description string `json:"description"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_json"))
				return
			}
			row, err := RenameRow(ctx, pool, userID, boardID, rowID, body.Label, body.Description)
			if err != nil {
				writeError(w, err)
				return
			}
			broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventRowUpdated, row)
			writeOK(w, map[string]any{"row": row})
		case http.MethodDelete:
			resolution := r.URL.Query().Get("resolution")
			targetRowID := r.URL.Query().Get("target_row_id")
			if err := RemoveRow(ctx, pool, userID, boardID, rowID, resolution, targetRowID); err != nil {
				writeError(w, err)
				return
			}
			broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventRowRemoved, map[string]any{"row_id": rowID})
			writeOK(w, map[string]any{"deleted": true})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
		}
	}
}

// HandleRowMove serves POST /api/storyboards/{board_id}/rows/{row_id}/move
// with body {"target_band_id": "..."}.
func HandleRowMove(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		rowID := strings.TrimSpace(r.PathValue("row_id"))
		var body struct {
			TargetBandID string `json:"target_band_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		row, err := MoveRowToBand(ctx, pool, userID, boardID, rowID, body.TargetBandID)
		if err != nil {
			writeError(w, err)
			return
		}
		broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventRowMoved, row)
		writeOK(w, map[string]any{"row": row})
	}
}

// HandleBandRowsReorder serves POST
// /api/storyboards/{board_id}/bands/{band_id}/rows/reorder with body
// {"ordered_row_ids": [...]}.
func HandleBandRowsReorder(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		bandID := strings.TrimSpace(r.PathValue("band_id"))
		var body struct {
			OrderedRowIDs []string `json:"ordered_row_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		if err := ReorderRowsWithinBand(ctx, pool, userID, boardID, bandID, body.OrderedRowIDs); err != nil {
			writeError(w, err)
			return
		}
		broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventRowsReordered, map[string]any{"band_id": bandID, "ordered_row_ids": body.OrderedRowIDs})
		writeOK(w, map[string]any{"reordered": true})
	}
}

// --- Cards ---

// HandleCards serves POST (create) at /api/storyboards/{board_id}/cards.
func HandleCards(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		var body struct {
			RowID     string `json:"row_id"`
			ColumnID  string `json:"column_id"`
			Title     string `json:"title"`
			FrontText string `json:"front_text"`
			BackText  string `json:"back_text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		card, err := CreateCard(ctx, pool, userID, boardID, body.RowID, body.ColumnID, body.Title, body.FrontText, body.BackText)
		if err != nil {
			writeError(w, err)
			return
		}
		broadcastCardEvent(ctx, pool, hub, boardID, EventCardAdded, card)
		writeOK(w, map[string]any{"card": card})
	}
}

// HandleCardItem serves PATCH (edit) and DELETE (remove) at
// /api/storyboards/{board_id}/cards/{card_id}. Both require
// ?base_version=N (optimistic concurrency, spec 4.7) -- PATCH also accepts
// it in the JSON body as base_version, preferring the body when both are
// present.
func HandleCardItem(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		cardID := strings.TrimSpace(r.PathValue("card_id"))
		switch r.Method {
		case http.MethodPatch:
			var body struct {
				BaseVersion        int    `json:"base_version"`
				Title              string `json:"title"`
				FrontText          string `json:"front_text"`
				BackText           string `json:"back_text"`
				Category           string `json:"category"`
				ColorToken         string `json:"color_token"`
				HiddenFromAudience *bool  `json:"hidden_from_audience"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_json"))
				return
			}
			card, err := UpdateCard(ctx, pool, userID, boardID, cardID, body.BaseVersion, CardEdit{
				Title: body.Title, FrontText: body.FrontText, BackText: body.BackText,
				Category: body.Category, ColorToken: body.ColorToken, HiddenFromAudience: body.HiddenFromAudience,
			})
			if err != nil {
				writeError(w, err)
				return
			}
			broadcastCardEvent(ctx, pool, hub, boardID, EventCardUpdated, card)
			writeOK(w, map[string]any{"card": card})
		case http.MethodDelete:
			baseVersion, _ := strconv.Atoi(r.URL.Query().Get("base_version"))
			deletedCard, loadErr := LoadCard(ctx, pool, boardID, cardID)
			if err := DeleteCard(ctx, pool, userID, boardID, cardID, baseVersion); err != nil {
				writeError(w, err)
				return
			}
			if loadErr == nil {
				broadcastCardEvent(ctx, pool, hub, boardID, EventCardRemoved, deletedCard)
			}
			writeOK(w, map[string]any{"deleted": true})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
		}
	}
}

// HandleCardMove serves POST /api/storyboards/{board_id}/cards/{card_id}/move
// with body {"base_version": N, "target_row_id": "...", "target_column_id": "..."}.
func HandleCardMove(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		cardID := strings.TrimSpace(r.PathValue("card_id"))
		var body struct {
			BaseVersion    int    `json:"base_version"`
			TargetRowID    string `json:"target_row_id"`
			TargetColumnID string `json:"target_column_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		card, err := MoveCard(ctx, pool, userID, boardID, cardID, body.BaseVersion, body.TargetRowID, body.TargetColumnID)
		if err != nil {
			writeError(w, err)
			return
		}
		broadcastCardEvent(ctx, pool, hub, boardID, EventCardMoved, card)
		writeOK(w, map[string]any{"card": card})
	}
}

// HandleCardLock serves POST /api/storyboards/{board_id}/cards/{card_id}/lock
// with body {"locked": true|false}.
func HandleCardLock(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		cardID := strings.TrimSpace(r.PathValue("card_id"))
		var body struct {
			Locked bool `json:"locked"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		card, err := SetCardLock(ctx, pool, userID, boardID, cardID, body.Locked)
		if err != nil {
			writeError(w, err)
			return
		}
		broadcastCardEvent(ctx, pool, hub, boardID, EventCardLockChanged, card)
		writeOK(w, map[string]any{"card": card})
	}
}

// HandleCellReorder serves POST /api/storyboards/{board_id}/cells/reorder
// with body {"row_id": "...", "column_id": "...", "ordered_card_ids": [...]}.
func HandleCellReorder(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		var body struct {
			RowID          string   `json:"row_id"`
			ColumnID       string   `json:"column_id"`
			OrderedCardIDs []string `json:"ordered_card_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		if err := ReorderCardsInCell(ctx, pool, userID, boardID, body.RowID, body.ColumnID, body.OrderedCardIDs); err != nil {
			writeError(w, err)
			return
		}
		broadcastCellReorder(ctx, pool, hub, boardID, body.RowID, body.ColumnID, body.OrderedCardIDs)
		writeOK(w, map[string]any{"reordered": true})
	}
}

// HandleCardSwap serves POST /api/storyboards/{board_id}/cards/swap with
// body {"card_a_id","base_version_a","card_b_id","base_version_b"} -- the
// Swap resolution for a drag dropped onto an occupied cell (spec 7.4).
func HandleCardSwap(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		var body struct {
			CardAID      string `json:"card_a_id"`
			BaseVersionA int    `json:"base_version_a"`
			CardBID      string `json:"card_b_id"`
			BaseVersionB int    `json:"base_version_b"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		cardA, cardB, err := SwapCards(ctx, pool, userID, boardID, body.CardAID, body.BaseVersionA, body.CardBID, body.BaseVersionB)
		if err != nil {
			writeError(w, err)
			return
		}
		broadcastCardEvent(ctx, pool, hub, boardID, EventCardSwapped, cardA)
		broadcastCardEvent(ctx, pool, hub, boardID, EventCardSwapped, cardB)
		writeOK(w, map[string]any{"card_a": cardA, "card_b": cardB})
	}
}

// HandleCardImage serves POST (multipart "file" field -> attach/replace)
// and DELETE (clear) at /api/storyboards/{board_id}/cards/{card_id}/image
// (spec 2.6/9). Authority is canMutateCard's Crew+-unless-locked gate,
// enforced inside SetCardImage -- attaching an image is a card-content
// edit like title or color, not a structural change.
func HandleCardImage(pool *pgxpool.Pool, hub *network.Hub, storageRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		cardID := strings.TrimSpace(r.PathValue("card_id"))

		switch r.Method {
		case http.MethodDelete:
			card, err := SetCardImage(ctx, pool, userID, boardID, cardID, "")
			if err != nil {
				writeError(w, err)
				return
			}
			broadcastCardEvent(ctx, pool, hub, boardID, EventCardUpdated, card)
			writeOK(w, map[string]any{"card": card})
			return
		case http.MethodPost:
			board, err := LoadBoard(ctx, pool, boardID)
			if err != nil {
				writeError(w, err)
				return
			}
			producerUserID, locationID, scopeErr := resolveCardImageStorageScope(ctx, pool, board.OwnerUserID)
			if scopeErr != nil {
				writeError(w, errors.New("card_image_storage_scope_unavailable"))
				return
			}

			r.Body = http.MaxBytesReader(w, r.Body, assets.MaxUploadBytes)
			if err := r.ParseMultipartForm(assets.MaxUploadBytes); err != nil {
				writeError(w, errors.New("invalid_or_oversize_multipart"))
				return
			}
			file, header, err := r.FormFile("file")
			if err != nil {
				writeError(w, errors.New("file_required"))
				return
			}
			defer file.Close()
			data, err := io.ReadAll(io.LimitReader(file, assets.MaxUploadBytes+1))
			if err != nil {
				writeError(w, errors.New("file_read_failed"))
				return
			}

			created, err := assets.CreateReferencedImageAsset(
				ctx, pool, storageRoot,
				userID, producerUserID, locationID, "storyboard_card",
				header.Filename, header.Header.Get("Content-Type"), data, assets.MaxUploadBytes,
			)
			if err != nil {
				writeError(w, err)
				return
			}
			card, err := SetCardImage(ctx, pool, userID, boardID, cardID, created.AssetID)
			if err != nil {
				writeError(w, err)
				return
			}
			broadcastCardEvent(ctx, pool, hub, boardID, EventCardUpdated, card)
			writeOK(w, map[string]any{"card": card})
			return
		default:
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
		}
	}
}
