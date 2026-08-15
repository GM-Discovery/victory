package socio

// Kernel 88A: the Player HUD's "roll a mechanic" list.
//
// The HUD originally sourced its skill list from GET
// /api/characters/venue-sheet, which resolves the *equipped session persona*
// (current_session_personas, written by the persona/equip action). That is a
// third identity mechanism, unrelated to the Show-Run roster selection that
// both ProjectSocioState and the roll-authority path
// (actions.StorePlayerMechanicRoll -> showruns.LoadMyRosterMember) actually
// key off. A Player with a perfectly valid roster Character but no equipped
// persona therefore got a 404, an empty skill list, and -- because the HUD
// renders that section only when it has skills -- no roll buttons at all,
// with nothing on screen to explain why. That was the whole of the "rolling a
// skill doesn't roll the dice anywhere" report: there was nothing to click.
//
// This endpoint closes that gap by resolving the skill list from the same
// Character identity the roll itself will use, so a mechanic the HUD offers
// is by construction a mechanic the roll path will accept. It reuses
// characters.BuildVenueCharacterSheet (Kernel 59A/60's shared
// ProjectCharacterSheet projection) rather than reading skill rows directly
// -- no second source of truth for what a Character's mechanics are.
//
// Import direction note: socio -> characters is safe. characters
// deliberately does not import socio (see characters/
// character_creation_fate_handoff.go's header for why), and socio already
// sits downstream of it via cohorts -> network -> actions -> characters.

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/characters"
)

// MechanicEntry is one rollable mechanic offered to the owning Player.
type MechanicEntry struct {
	SkillID       string `json:"skill_id"`
	SkillName     string `json:"skill_name"`
	AttributeName string `json:"attribute_name"`
	Expression    string `json:"expression"`
	LadderStep    int    `json:"ladder_step"`
}

// CharacterDisplayName returns the roster Character's name for a viewer the
// caller has ALREADY authorized (see ListCharacterMechanics, its only
// caller). Read directly rather than via characters.LoadCardForActor for the
// same reason the skill read is: that path gates on CanEditCard. Without
// this the HUD fell back to the literal string "Your Character" for any
// Player without an equipped persona, since the name was coming from the
// same venue-sheet call that was 404ing.
func CharacterDisplayName(ctx context.Context, pool *pgxpool.Pool, characterCardID string) (string, error) {
	var name string
	err := pool.QueryRow(ctx, `
		SELECT name FROM character_cards WHERE id = $1 AND is_deleted = FALSE LIMIT 1
	`, characterCardID).Scan(&name)
	if err != nil {
		return "", err
	}
	return name, nil
}

// ListCharacterMechanics returns the rollable mechanics for characterCardID
// as seen by viewerUserID, along with that Character's display name. Authority
// is ResolveTier's, exactly as for every other Socio read: the owning Player
// or a Director+ on the Show, nobody else. It returns the tier alongside so a
// caller cannot accidentally offer roll controls to a Director looking at
// someone else's Character -- rolling your own mechanic is an Owner-tier act.
func ListCharacterMechanics(ctx context.Context, pool *pgxpool.Pool, viewerUserID, showID, characterCardID string) ([]MechanicEntry, string, error) {
	viewerUserID = strings.TrimSpace(viewerUserID)
	showID = strings.TrimSpace(showID)
	characterCardID = strings.TrimSpace(characterCardID)

	tier, err := ResolveTier(ctx, pool, showID, characterCardID, viewerUserID)
	if err != nil {
		return nil, "", err
	}

	// ResolveTier above is the authority check; the skill read below is
	// deliberately the un-gated variant, because the ordinary read path gates
	// on CanEditCard (an *edit* permission) and so denied Players their own
	// mechanics whenever they lacked location-level drafting rights. See that
	// function's contract comment.
	skills, err := characters.ListCharacterSkillsForAuthorizedReader(ctx, pool, characterCardID)
	if err != nil {
		return nil, "", err
	}

	out := make([]MechanicEntry, 0, len(skills))
	for _, s := range skills {
		// Exploding form -- the same expression Kernel 60's venue sheet shows
		// and the roll path evaluates, not a second rendering of the ladder.
		expr, err := characters.StepExpression(s.LadderStep, true)
		if err != nil {
			return nil, "", err
		}
		out = append(out, MechanicEntry{
			SkillID:       s.SkillID,
			SkillName:     s.SkillName,
			AttributeName: s.AttributeName,
			Expression:    expr,
			LadderStep:    s.LadderStep,
		})
	}
	return out, tier, nil
}
