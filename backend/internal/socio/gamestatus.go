package socio

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/cohorts"
)

// BuildGameStatusForCohort assembles the Game Status panel's content for
// one cohort (kernel-85 S1.10, S1.11): one compact block per Character
// currently in that cohort, each carrying live-read canonical HP pools and
// active statuses. cohortID may be the literal string "ungrouped" to view
// Ungrouped participants instead of a real cohort -- Ungrouped is
// legitimately selectable per kernel-85 S1.11. Director+/Producer/Operator
// authority only, enforced by cohorts.ListRosterForShow.
func BuildGameStatusForCohort(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, cohortID string) ([]CharacterBlock, error) {
	roster, err := cohorts.ListRosterForShow(ctx, pool, actorUserID, showID)
	if err != nil {
		return nil, err
	}

	var participants []cohorts.Participant
	var resolvedCohortID string
	if cohortID == "ungrouped" {
		participants = roster.Ungrouped
	} else {
		found := false
		for _, c := range roster.Cohorts {
			if c.ID == cohortID {
				participants = c.Members
				resolvedCohortID = c.ID
				found = true
				break
			}
		}
		if !found {
			return nil, errors.New("cohort_not_found")
		}
	}

	out := []CharacterBlock{}
	for _, p := range participants {
		if p.CharacterCardID == "" {
			// No Character selected yet -- nothing canonical to show a
			// Game Status block for (kernel-85 S2: Victory never invents
			// Character state).
			continue
		}
		state, err := GetState(ctx, pool, p.CharacterCardID)
		if err != nil {
			return nil, err
		}
		activeStatuses, err := ListActiveStatuses(ctx, pool, p.CharacterCardID)
		if err != nil {
			return nil, err
		}
		out = append(out, CharacterBlock{
			CharacterCardID: p.CharacterCardID,
			CharacterName:   p.CharacterName,
			UserID:          p.UserID,
			CohortID:        resolvedCohortID,
			Pools:           state.Pools,
			ActiveStatuses:  activeStatuses,
		})
	}
	return out, nil
}
