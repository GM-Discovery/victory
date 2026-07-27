package merchant

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
)

// ShowParticipation is the roster-and-Character half of an eligibility
// context, for surfaces keyed on a SHOW rather than on an interaction.
type ShowParticipation struct {
	ActorUserID     string
	ShowID          string
	ShowRunID       string
	LocationID      string
	CharacterCardID string
	// SessionID is best-effort and may be empty. See the note in
	// ResolveShowParticipation about why this surface does not require a
	// live Session.
	SessionID string
}

// ResolveShowParticipation answers "is this user an active Player with an
// owned Character in this Show".
//
// It exists because Aftercare is Show-keyed, not interaction-keyed: S8 lets
// a Player open Aftercare later from the Greenroom, when no Program is on
// screen and possibly no Session is running. ResolveEligibleContext cannot
// serve that -- it requires a current Scene placement and an active Session,
// both correct for a stage interaction and both wrong for a reflection
// written the next morning.
//
// The roster and Character checks below are lifted VERBATIM from
// ResolveEligibleContext, which now calls this function, so there remains
// exactly one definition of who counts as a Player here. Writing a second
// copy is the drift tutorial_flow.go's header argues against, and an
// authority check that exists twice is an authority check that will
// eventually disagree with itself.
//
// TWO DELIBERATE RELAXATIONS relative to ResolveEligibleContext, neither of
// which widens who may act:
//
//   - No current-Scene check. Aftercare is not a stage action; requiring the
//     Courtyard to still be the current Scene would make reflection
//     impossible the moment the table moved on.
//   - No active-Session requirement. SessionID is looked up if one happens
//     to exist, and left empty otherwise.
//
// Everything that decides AUTHORITY -- authenticated, active roster row,
// role player, Character selected, Character owned and not deleted -- is
// unchanged.
func ResolveShowParticipation(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID string) (ShowParticipation, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return ShowParticipation{}, errors.New("not_authenticated")
	}
	showID = strings.TrimSpace(showID)
	if showID == "" {
		return ShowParticipation{}, errors.New("no_active_show")
	}

	show, err := shows.LoadShowByID(ctx, pool, showID)
	if err != nil {
		return ShowParticipation{}, err
	}
	run, err := showruns.LoadShowRunByID(ctx, pool, show.ShowRunID)
	if err != nil {
		return ShowParticipation{}, err
	}

	member, err := showruns.LoadMyRosterMember(ctx, pool, actorUserID, show.ShowRunID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || strings.TrimSpace(err.Error()) == "roster_member_not_found" {
			return ShowParticipation{}, errors.New("not_a_roster_member")
		}
		return ShowParticipation{}, err
	}
	if strings.ToLower(strings.TrimSpace(member.Role)) != "player" {
		return ShowParticipation{}, errors.New("insufficient_role")
	}
	if strings.TrimSpace(member.CharacterCardID) == "" {
		return ShowParticipation{}, errors.New("no_character_selected")
	}

	allowed, err := characterActiveAndOwnedBy(ctx, pool, actorUserID, member.CharacterCardID)
	if err != nil {
		return ShowParticipation{}, err
	}
	if !allowed {
		return ShowParticipation{}, errors.New("character_not_owned")
	}

	// Best-effort only -- see the doc comment.
	var sessionID string
	_ = pool.QueryRow(ctx, `
		SELECT id::text FROM sessions
		WHERE show_id = $1
		ORDER BY started_at DESC
		LIMIT 1
	`, showID).Scan(&sessionID)

	return ShowParticipation{
		ActorUserID:     actorUserID,
		ShowID:          showID,
		ShowRunID:       show.ShowRunID,
		LocationID:      run.LocationID,
		CharacterCardID: member.CharacterCardID,
		SessionID:       sessionID,
	}, nil
}
