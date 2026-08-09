package cohorts

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/shows"
	"victory/backend/internal/showruns"
)

const cohortColumns = `
	id::text, show_id::text, serial_number, slug, name,
	current_show_scene_placement_id::text, created_by_user_id::text,
	created_at, updated_at, archived_at
`

func scanCohort(row pgx.Row) (Cohort, error) {
	var c Cohort
	var placementID *string
	if err := row.Scan(
		&c.ID, &c.ShowID, &c.SerialNumber, &c.Slug, &c.Name,
		&placementID, &c.CreatedByUserID, &c.CreatedAt, &c.UpdatedAt, &c.ArchivedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Cohort{}, errors.New("cohort_not_found")
		}
		return Cohort{}, err
	}
	c.CurrentShowScenePlacementID = placementID
	return c, nil
}

// showRunLocationForShow resolves the Show Run location a Show's cohorts
// authority is checked against, the same two-step lookup
// scenes.showRunForShow and shows.SetCurrentScenePlacement already perform.
func showRunLocationForShow(ctx context.Context, pool *pgxpool.Pool, showID string) (shows.Show, string, error) {
	s, err := shows.LoadShowByID(ctx, pool, showID)
	if err != nil {
		return shows.Show{}, "", err
	}
	sr, err := showruns.LoadShowRunByID(ctx, pool, s.ShowRunID)
	if err != nil {
		return shows.Show{}, "", err
	}
	return s, sr.LocationID, nil
}

func requireManage(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID string) (shows.Show, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return shows.Show{}, errors.New("not_authenticated")
	}
	s, locationID, err := showRunLocationForShow(ctx, pool, showID)
	if err != nil {
		return shows.Show{}, err
	}
	ok, err := showruns.CanManageShowRun(ctx, pool, actorUserID, locationID)
	if err != nil {
		return shows.Show{}, err
	}
	if !ok {
		return shows.Show{}, errors.New("not_authorized")
	}
	return s, nil
}

// CreateCohort allocates the next safe serial number for this Show inside
// one transaction (row-locked counter, kernel-85 S1.2/S3.2/S3.3) and inserts
// the new cohort. Director+/Producer/Operator authority only.
func CreateCohort(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID string) (Cohort, error) {
	if _, err := requireManage(ctx, pool, actorUserID, showID); err != nil {
		return Cohort{}, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return Cohort{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO show_cohort_serial_counters (show_id, next_serial)
		VALUES ($1, 1)
		ON CONFLICT (show_id) DO NOTHING
	`, showID); err != nil {
		return Cohort{}, err
	}

	var serial int
	if err := tx.QueryRow(ctx, `
		SELECT next_serial FROM show_cohort_serial_counters WHERE show_id = $1 FOR UPDATE
	`, showID).Scan(&serial); err != nil {
		return Cohort{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE show_cohort_serial_counters SET next_serial = next_serial + 1 WHERE show_id = $1
	`, showID); err != nil {
		return Cohort{}, err
	}

	slug := cohortSlug(serial)
	name := cohortName(serial)
	row := tx.QueryRow(ctx, `
		INSERT INTO show_cohorts (show_id, serial_number, slug, name, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+cohortColumns,
		showID, serial, slug, name, actorUserID)
	c, err := scanCohort(row)
	if err != nil {
		return Cohort{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Cohort{}, err
	}
	return c, nil
}

func cohortSlug(serial int) string {
	return "cohort-" + itoa(serial)
}

func cohortName(serial int) string {
	return "Cohort " + itoa(serial)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// LoadCohortByID returns the raw row with no authority check.
func LoadCohortByID(ctx context.Context, pool *pgxpool.Pool, cohortID string) (Cohort, error) {
	cohortID = strings.TrimSpace(cohortID)
	if cohortID == "" {
		return Cohort{}, errors.New("cohort_not_found")
	}
	row := pool.QueryRow(ctx, `SELECT `+cohortColumns+` FROM show_cohorts WHERE id = $1`, cohortID)
	return scanCohort(row)
}

// ArchiveCohort retires a cohort and returns its members to Ungrouped
// (deleting their assignment rows) in one transaction. The cohort's serial
// number is never reissued (the counter only ever increments, S1.2).
func ArchiveCohort(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, cohortID string) error {
	if _, err := requireManage(ctx, pool, actorUserID, showID); err != nil {
		return err
	}
	c, err := LoadCohortByID(ctx, pool, cohortID)
	if err != nil {
		return err
	}
	if c.ShowID != showID {
		return errors.New("cohort_show_mismatch")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM show_cohort_assignments WHERE cohort_id = $1`, cohortID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE show_cohorts SET archived_at = NOW(), updated_at = NOW() WHERE id = $1
	`, cohortID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// AssignParticipant moves a Show's Player-role roster member into a cohort
// -- from Ungrouped or from another cohort. UPSERT on the assignment
// table's (show_id, user_id) primary key is what makes "at most one active
// cohort per participant" true by construction (kernel-85 S1.3): a second
// assignment for the same user simply overwrites the first, it can never
// create a second active row.
func AssignParticipant(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, cohortID, targetUserID string) error {
	if _, err := requireManage(ctx, pool, actorUserID, showID); err != nil {
		return err
	}
	targetUserID = strings.TrimSpace(targetUserID)
	if targetUserID == "" {
		return errors.New("user_id_required")
	}
	c, err := LoadCohortByID(ctx, pool, cohortID)
	if err != nil {
		return err
	}
	if c.ShowID != showID {
		return errors.New("cohort_show_mismatch")
	}
	if c.ArchivedAt != nil {
		return errors.New("cohort_archived")
	}

	isParticipant, err := isActivePlayerOnShow(ctx, pool, showID, targetUserID)
	if err != nil {
		return err
	}
	if !isParticipant {
		return errors.New("user_not_a_show_participant")
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO show_cohort_assignments (show_id, user_id, cohort_id, assigned_by_user_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (show_id, user_id) DO UPDATE
		  SET cohort_id = EXCLUDED.cohort_id, assigned_by_user_id = EXCLUDED.assigned_by_user_id, assigned_at = NOW()
	`, showID, targetUserID, cohortID, actorUserID)
	return err
}

// UnassignParticipant returns a participant to Ungrouped by deleting their
// assignment row -- Ungrouped is never a stored state (S3.1).
func UnassignParticipant(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, targetUserID string) error {
	if _, err := requireManage(ctx, pool, actorUserID, showID); err != nil {
		return err
	}
	targetUserID = strings.TrimSpace(targetUserID)
	if targetUserID == "" {
		return errors.New("user_id_required")
	}
	_, err := pool.Exec(ctx, `
		DELETE FROM show_cohort_assignments WHERE show_id = $1 AND user_id = $2
	`, showID, targetUserID)
	return err
}

func isActivePlayerOnShow(ctx context.Context, pool *pgxpool.Pool, showID, userID string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM show_run_roster_members rm
			JOIN shows s ON s.show_run_id = rm.show_run_id
			WHERE s.id = $1 AND rm.user_id = $2 AND rm.role = 'player' AND rm.removed_at IS NULL
		)
	`, showID, userID).Scan(&exists)
	return exists, err
}

// ListRosterForShow returns every cohort (with members) and Ungrouped for a
// Show. Director+/Producer/Operator authority only -- matches every other
// cohort-management read/write in this package (kernel-85 S4: "Ordinary
// participants do not gain cohort-management controls").
func ListRosterForShow(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID string) (Roster, error) {
	s, err := requireManage(ctx, pool, actorUserID, showID)
	if err != nil {
		return Roster{}, err
	}

	rows, err := pool.Query(ctx, `SELECT `+cohortColumns+` FROM show_cohorts WHERE show_id = $1 AND archived_at IS NULL ORDER BY serial_number ASC`, showID)
	if err != nil {
		return Roster{}, err
	}
	cohortList := []Cohort{}
	for rows.Next() {
		c, err := scanCohort(rows)
		if err != nil {
			rows.Close()
			return Roster{}, err
		}
		cohortList = append(cohortList, c)
	}
	if err := rows.Err(); err != nil {
		return Roster{}, err
	}
	rows.Close()

	participants, err := listPlayerParticipants(ctx, pool, s.ShowRunID)
	if err != nil {
		return Roster{}, err
	}

	assignments, err := loadAssignments(ctx, pool, showID)
	if err != nil {
		return Roster{}, err
	}

	byCohort := map[string][]Participant{}
	ungrouped := []Participant{}
	for _, p := range participants {
		if cid, ok := assignments[p.UserID]; ok {
			byCohort[cid] = append(byCohort[cid], p)
		} else {
			ungrouped = append(ungrouped, p)
		}
	}

	out := Roster{Cohorts: []CohortWithMembers{}, Ungrouped: ungrouped}
	for _, c := range cohortList {
		members := byCohort[c.ID]
		if members == nil {
			members = []Participant{}
		}
		out.Cohorts = append(out.Cohorts, CohortWithMembers{Cohort: c, Members: members})
	}
	return out, nil
}

func loadAssignments(ctx context.Context, pool *pgxpool.Pool, showID string) (map[string]string, error) {
	rows, err := pool.Query(ctx, `SELECT user_id::text, cohort_id::text FROM show_cohort_assignments WHERE show_id = $1`, showID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var userID, cohortID string
		if err := rows.Scan(&userID, &cohortID); err != nil {
			return nil, err
		}
		out[userID] = cohortID
	}
	return out, rows.Err()
}

// listPlayerParticipants returns every active Player-role roster member of
// a Show Run, display-projected (stage name + selected Character, never
// handle/raw account facts -- matching this codebase's established
// people-display convention).
func listPlayerParticipants(ctx context.Context, pool *pgxpool.Pool, showRunID string) ([]Participant, error) {
	rows, err := pool.Query(ctx, `
		SELECT rm.user_id::text, COALESCE(sn.stage_name, ''),
		       COALESCE(rm.character_card_id::text, ''), COALESCE(cc.name, '')
		FROM show_run_roster_members rm
		LEFT JOIN player_stage_name_history sn ON sn.user_id = rm.user_id AND sn.ended_at IS NULL
		LEFT JOIN character_cards cc ON cc.id = rm.character_card_id
		WHERE rm.show_run_id = $1 AND rm.role = 'player' AND rm.removed_at IS NULL
		ORDER BY rm.added_at ASC
	`, showRunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Participant{}
	for rows.Next() {
		var p Participant
		if err := rows.Scan(&p.UserID, &p.DisplayName, &p.CharacterCardID, &p.CharacterName); err != nil {
			return nil, err
		}
		if p.DisplayName == "" {
			p.DisplayName = "Participant"
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
