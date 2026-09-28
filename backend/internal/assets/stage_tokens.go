package assets

import (
	"context"
	"net/http"
	"strings"
	"time"

	"victory/backend/internal/access"
	"victory/backend/internal/sessions"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HandleStageTokenAssets is the stage token picker's own asset feed.
//
// Kernel 101 (101-24) let Cast create a brand-new token, but the picker they
// must pick that token from was reading /api/warehouse/assets, which
// requireWarehouseAccess gates at producer/director/operator -- so a Cast
// user got an enabled "Add Token" menu entry, an open picker, and a 403'd,
// permanently empty asset list. This endpoint is the narrow read that fixes
// that, deliberately separate from the Warehouse browser rather than a
// loosening of it:
//
//   - It is GET-only and lists nothing but ACTIVE TOKEN assets. The Warehouse
//     browser's other asset types, its deleted/tombstoned views, and
//     HandleWarehouseAssetByID's DELETE all stay behind requireWarehouseAccess
//     exactly as before.
//   - It grants no authority to place anything. The real create decision is
//     still canActCreateToken (actions/authority.go) on the socket action, and
//     editing a placed token is still canActUpdateToken, untouched and still
//     producer/director-only. Listing an asset here lets a Cast user name a
//     token; it does not let them modify one.
//   - Serving the actual image bytes needed no change at all:
//     userCanReadAsset already admits 'cast' location members (read.go), so
//     the thumbnails these rows point at were always fetchable.
func HandleStageTokenAssets(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		ok, err := requireStageTokenPickerAccess(ctx, pool, r)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "access_check_failed"})
			return
		}
		if !ok {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
			return
		}

		// asset_type and status are fixed here, not read from the query
		// string: this endpoint exists to serve the token picker, and a
		// caller must not be able to widen it into a general asset browser
		// by asking for another type or for tombstoned rows.
		items, errCode, err := queryWarehouseAssetList(ctx, pool, "token", "active", strings.TrimSpace(r.URL.Query().Get("search")))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": errCode})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": items})
	}
}

// requireStageTokenPickerAccess mirrors requireWarehouseAccess, plus 'cast'.
// Crew is deliberately NOT included: canActCreateToken denies Crew, and
// logic.js's canCreateStageObjects leaves Crew out of the Add Token menu
// entry, so listing tokens for them would grant a view with nothing behind
// it.
func requireStageTokenPickerAccess(ctx context.Context, pool *pgxpool.Pool, r *http.Request) (bool, error) {
	rawSession, err := sessions.ReadSessionCookie(r)
	if err != nil || strings.TrimSpace(rawSession) == "" {
		return false, nil
	}

	rec, err := sessions.GetSessionByRawToken(ctx, pool, rawSession)
	if err != nil || strings.TrimSpace(rec.UserID) == "" {
		return false, nil
	}

	userID := rec.UserID
	if ok, err := access.IsOperatorUser(ctx, pool, userID); err == nil && ok {
		return true, nil
	} else if err != nil {
		return false, err
	}

	role, err := access.CurrentDefaultLocationRole(ctx, pool, userID)
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "producer", "director", "cast":
		return true, nil
	default:
		return false, nil
	}
}
