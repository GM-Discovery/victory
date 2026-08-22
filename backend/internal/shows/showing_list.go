package shows

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showruns"
)

// ShowingListItem is one row in the Kernel 92 Showtime popup's Today /
// Upcoming / Historical browser.
type ShowingListItem struct {
	ShowID           string     `json:"show_id"`
	ShowRunID        string     `json:"show_run_id"`
	ShowRunTitle     string     `json:"show_run_title"`
	Nickname         string     `json:"nickname"`
	ShortCode        string     `json:"short_code,omitempty"`
	Status           string     `json:"status"`
	ScheduledStartAt *time.Time `json:"scheduled_start_at,omitempty"`
	ActualStartAt    *time.Time `json:"actual_start_at,omitempty"`
	ActualEndAt      *time.Time `json:"actual_end_at,omitempty"`
	IsLive           bool       `json:"is_live"`
}

// Historical age-band keys, kernel doc §9/§30, in oldest-last iteration
// order for the frontend to render sections top-to-bottom.
const (
	BandLessThan7Days = "lt_7d"
	Band7To30Days     = "d7_30"
	Band31To90Days    = "d31_90"
	Band91To180Days   = "d91_180"
	Band181To365Days  = "d181_365"
	Band366DaysTo36Mo = "d366_36mo"
	BandOlder         = "older"
)

// HistoricalBandOrder is the display order of the age bands above.
var HistoricalBandOrder = []string{
	BandLessThan7Days, Band7To30Days, Band31To90Days,
	Band91To180Days, Band181To365Days, Band366DaysTo36Mo, BandOlder,
}

// ShowingListResult is the bucketed shape the Showtime popup renders
// directly: Today, then Upcoming, then Historical grouped into age bands.
// Liveness (IsLive) is derived per-row from `sessions` at read time, never
// stored -- kernel doc §25 explicitly forbids a second, redundant
// liveness/state model.
type ShowingListResult struct {
	Today      []ShowingListItem            `json:"today"`
	Upcoming   []ShowingListItem            `json:"upcoming"`
	Historical map[string][]ShowingListItem `json:"historical"`
}

// ListShowingsForViewer returns every Showing (shows row) under a Show Run
// the viewer can manage, bucketed and sorted per kernel doc §7-9/§30. `now`
// is threaded in explicitly (rather than time.Now() inline) so tests can
// pin the bucket boundaries deterministically.
//
// "Today" is computed as a UTC calendar day -- the server has no per-viewer
// timezone to hand, and the rest of the codebase already stores/derives
// timestamps in UTC (see shows.UpdateShow's ScheduledStartAt handling).
func ListShowingsForViewer(ctx context.Context, pool *pgxpool.Pool, viewerUserID string, now time.Time) (ShowingListResult, error) {
	viewerUserID = strings.TrimSpace(viewerUserID)
	if viewerUserID == "" {
		return ShowingListResult{}, errors.New("not_authenticated")
	}

	runs, err := showruns.ListShowRunsVisibleToUser(ctx, pool, viewerUserID)
	if err != nil {
		return ShowingListResult{}, err
	}
	var runIDs []string
	runTitles := map[string]string{}
	for _, r := range runs {
		if !r.CanManage {
			continue
		}
		runIDs = append(runIDs, r.ID)
		runTitles[r.ID] = r.Title
	}
	result := ShowingListResult{Historical: map[string][]ShowingListItem{}}
	for _, band := range HistoricalBandOrder {
		result.Historical[band] = nil
	}
	if len(runIDs) == 0 {
		return result, nil
	}

	rows, err := pool.Query(ctx, `
		SELECT `+showColumns+`
		FROM shows
		WHERE show_run_id = ANY($1) AND archived_at IS NULL
	`, runIDs)
	if err != nil {
		return ShowingListResult{}, err
	}
	defer rows.Close()

	var all []Show
	var showIDs []string
	for rows.Next() {
		s, err := scanShow(rows)
		if err != nil {
			return ShowingListResult{}, err
		}
		all = append(all, s)
		showIDs = append(showIDs, s.ID)
	}
	if err := rows.Err(); err != nil {
		return ShowingListResult{}, err
	}

	liveShowIDs, err := loadLiveShowIDs(ctx, pool, showIDs)
	if err != nil {
		return ShowingListResult{}, err
	}

	todayStart := now.UTC().Truncate(24 * time.Hour)
	todayEnd := todayStart.Add(24 * time.Hour)

	var today, upcoming []ShowingListItem
	historical := map[string][]ShowingListItem{}

	for _, s := range all {
		item := ShowingListItem{
			ShowID:           s.ID,
			ShowRunID:        s.ShowRunID,
			ShowRunTitle:     runTitles[s.ShowRunID],
			Nickname:         s.Nickname,
			ShortCode:        s.ShortCode,
			Status:           s.Status,
			ScheduledStartAt: s.ScheduledStartAt,
			ActualStartAt:    s.ActualStartAt,
			ActualEndAt:      s.ActualEndAt,
			IsLive:           liveShowIDs[s.ID],
		}

		isPast := s.Status == "completed" || s.Status == "cancelled"
		anchor := s.ScheduledStartAt
		if s.ActualStartAt != nil {
			anchor = s.ActualStartAt
		}

		switch {
		case !isPast && anchor != nil && !anchor.Before(todayStart) && anchor.Before(todayEnd):
			today = append(today, item)
		case !isPast && (anchor == nil || !anchor.Before(todayEnd)):
			upcoming = append(upcoming, item)
		default:
			age := now.Sub(coalesceTime(anchor, s.CreatedAt))
			band := historicalBand(age)
			historical[band] = append(historical[band], item)
		}
	}

	sortByNickname(today)
	sortByNickname(upcoming)
	for _, band := range HistoricalBandOrder {
		sortByNickname(historical[band])
		result.Historical[band] = historical[band]
	}
	result.Today = today
	result.Upcoming = upcoming

	return result, nil
}

func coalesceTime(t *time.Time, fallback time.Time) time.Time {
	if t != nil {
		return *t
	}
	return fallback
}

func historicalBand(age time.Duration) string {
	switch {
	case age < 7*24*time.Hour:
		return BandLessThan7Days
	case age < 30*24*time.Hour:
		return Band7To30Days
	case age < 90*24*time.Hour:
		return Band31To90Days
	case age < 180*24*time.Hour:
		return Band91To180Days
	case age < 365*24*time.Hour:
		return Band181To365Days
	case age < 36*30*24*time.Hour:
		return Band366DaysTo36Mo
	default:
		return BandOlder
	}
}

func sortByNickname(items []ShowingListItem) {
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Nickname) < strings.ToLower(items[j].Nickname)
	})
}

// loadLiveShowIDs batch-derives which of the given Shows currently have a
// rehearsal/live session, rather than joining `sessions` per-row.
func loadLiveShowIDs(ctx context.Context, pool *pgxpool.Pool, showIDs []string) (map[string]bool, error) {
	out := map[string]bool{}
	if len(showIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT DISTINCT show_id::text FROM sessions
		WHERE show_id = ANY($1) AND status IN ('rehearsal', 'live')
	`, showIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}
