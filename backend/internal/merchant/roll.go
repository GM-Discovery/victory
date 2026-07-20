package merchant

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dice"
)

// SkillGatedRoll is the bounded new foundation the kernel-73 audit asked
// for (spec S3/S13.1 deliverable #4): NOT a generalized challenge/TV-
// comparison engine, just "does this Character have the named skill,
// therefore which die size" plus one canonical server-authoritative roll.
// Reused by both stance attempts (result only picks a response-text tier;
// disposition itself stays authored/fixed per spec S2.6) and Haggle (result
// IS compared to a target value, real pass/fail) -- callers do their own
// comparison, this only resolves skill possession + rolls the die.
type SkillGatedRoll struct {
	HasSkill bool
	Die      string
	Result   dice.Result
}

// hasCharacterSkill replicates characters.FindCharacterSkillByName's exact-
// or-unambiguous-prefix matching, but queries character_skills directly
// instead of going through characters.ListCharacterSkills/CanEditCard.
// CanEditCard additionally requires CanDraftCharacter (a location_memberships
// role of producer/director/cast/crew) -- a gate built for the character
// workbook editing surface, not for a ticket-only Show Run Player who has
// already been authorized by ResolveEligibleContext's own ownership check
// (characterActiveAndOwnedBy) before this is ever called. Reusing
// FindCharacterSkillByName here would incorrectly deny exactly the Player
// this whole feature is for.
func hasCharacterSkill(ctx context.Context, pool *pgxpool.Pool, characterCardID, skillName string) (bool, error) {
	skillName = strings.TrimSpace(skillName)
	if skillName == "" {
		return false, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT skill_name FROM character_skills WHERE character_card_id = $1
	`, characterCardID)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	lowered := strings.ToLower(skillName)
	matchCount := 0
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return false, err
		}
		if strings.EqualFold(name, skillName) {
			return true, nil
		}
		if strings.HasPrefix(strings.ToLower(name), lowered) {
			matchCount++
		}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	// An ambiguous prefix (multiple skills sharing it) is treated as "no
	// exact skill" rather than an error -- Haggle-gating only needs a
	// boolean, unlike /char advance's need to name exactly one skill.
	return matchCount == 1, nil
}

// RollSkillGatedDie resolves whether characterCardID has skillName, rolls
// skilledDie if so and unskilledDie otherwise, and returns both the die
// used and the canonical server-side roll. Callers are expected to have
// already resolved eligibility (ownership, roster role, enablement) before
// calling this.
func RollSkillGatedDie(ctx context.Context, pool *pgxpool.Pool, actorUserID, characterCardID, skillName, skilledDie, unskilledDie string) (SkillGatedRoll, error) {
	_ = actorUserID // ownership already verified by the caller's ResolveEligibleContext
	hasSkill, err := hasCharacterSkill(ctx, pool, characterCardID, skillName)
	if err != nil {
		return SkillGatedRoll{}, err
	}

	die := unskilledDie
	if hasSkill {
		die = skilledDie
	}

	result, err := dice.RollExpression(ctx, die, dice.CryptoSource{})
	if err != nil {
		return SkillGatedRoll{}, err
	}

	return SkillGatedRoll{HasSkill: hasSkill, Die: die, Result: result}, nil
}
