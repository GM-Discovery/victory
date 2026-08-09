package world

// Kernel 85: resolve a viewer's Show Cohort current-Scene placement, if
// any. This is a narrow, deliberately duplicated query against
// show_cohort_assignments/show_cohorts rather than an import of
// backend/internal/cohorts -- that package's http.go imports
// backend/internal/network for its broadcast call, and network already
// imports world (director_console.go), so importing cohorts here would
// create world -> cohorts -> network -> world. The same avoidance is why
// this file's sibling, snapshot.go, already reads shows.
// current_show_scene_placement_id with raw SQL instead of importing
// backend/internal/shows.

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// resolveCohortPlacementForViewer returns "" when the viewer is Ungrouped,
// belongs to an archived cohort, or their cohort has no current Scene set
// yet -- every one of those cases falls through to the unchanged
// Show-global current-Scene path in LoadVenueSnapshot's caller (kernel-85
// S5.6: Ungrouped never reaches sustained-play Scene progression, because
// there is never an assignment row to find).
func resolveCohortPlacementForViewer(ctx context.Context, pool *pgxpool.Pool, showID, userID string) (string, error) {
	showID = strings.TrimSpace(showID)
	userID = strings.TrimSpace(userID)
	if showID == "" || userID == "" {
		return "", nil
	}
	var placementID *string
	err := pool.QueryRow(ctx, `
		SELECT c.current_show_scene_placement_id::text
		FROM show_cohort_assignments a
		JOIN show_cohorts c ON c.id = a.cohort_id
		WHERE a.show_id = $1 AND a.user_id = $2 AND c.archived_at IS NULL
	`, showID, userID).Scan(&placementID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	if placementID == nil {
		return "", nil
	}
	return *placementID, nil
}
