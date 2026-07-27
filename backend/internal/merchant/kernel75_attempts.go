package merchant

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Kernel 75 S5.1: durable, Character-keyed records of what a Player actually
// tried at a merchant's stall.
//
// Kernel 73 recorded stance and Haggle attempts only as ephemeral actions
// rows -- keyed to a user rather than a Character, and belonging to Showing
// Review rather than to Character history. Story So Far needs to say "at
// Kessa's stall, you approached through Insight" truthfully, months later,
// after the Session is long closed. That requires a row that outlives the
// Session and names the Character.
//
// These writes are ADDITIVE. The actions rows in AttemptStance and
// AttemptHaggle stay exactly as they were; Showing Review still reads them.

// attemptRecord is the union of the stance and Haggle shapes. Fields that do
// not apply to a kind are left zero and stored NULL, never as a sentinel --
// see the migration's note on success being NULL for stance rows.
type attemptRecord struct {
	AttemptKind  string
	PacketSlug   string
	StanceKey    string
	Disposition  string
	ResponseTier *int
	SkillKey     string
	HasSkill     *bool
	Die          string
	Total        *int
	TargetValue  *int
	Success      *bool
}

// recordInteractionAttempt writes the durable attempt row for an already
// authorized eligibility context.
//
// Callers must treat a returned error as fatal to the whole attempt, and
// must call this BEFORE writing the ephemeral actions row. The ordering
// matters: if the durable write fails, no actions row should exist claiming
// an attempt that Character history cannot corroborate. The reverse ordering
// would make the two logs silently disagree.
//
// No authority check here. eligible came from ResolveEligibleContext, which
// is the one Player-eligibility gate; re-checking would duplicate it and
// eventually drift from it.
func recordInteractionAttempt(ctx context.Context, pool *pgxpool.Pool, eligible EligibleContext, rec attemptRecord) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO character_interaction_attempts (
			character_card_id, actor_user_id, participant_interaction_id,
			show_run_id, show_id, session_id, show_scene_placement_id,
			packet_slug, attempt_kind,
			stance_key, disposition, response_tier,
			skill_key, has_skill, die, total, target_value, success
		)
		VALUES (
			$1, $2, $3::uuid,
			$4::uuid, $5::uuid, $6::uuid, $7::uuid,
			$8, $9,
			$10, $11, $12,
			$13, $14, $15, $16, $17, $18
		)
	`,
		eligible.CharacterCardID,
		eligible.ActorUserID,
		attemptNullableID(eligible.Interaction.ID),
		attemptNullableID(eligible.ShowRunID),
		attemptNullableID(eligible.ShowID),
		attemptNullableID(eligible.SessionID),
		attemptNullableID(eligible.PlacementID),
		strings.TrimSpace(rec.PacketSlug),
		rec.AttemptKind,
		strings.TrimSpace(rec.StanceKey),
		strings.TrimSpace(rec.Disposition),
		rec.ResponseTier,
		strings.TrimSpace(rec.SkillKey),
		rec.HasSkill,
		strings.TrimSpace(rec.Die),
		rec.Total,
		rec.TargetValue,
		rec.Success,
	)
	return err
}

// attemptNullableID mirrors tutorial.nullableID: an empty optional foreign
// key must reach Postgres as NULL, not as the empty string, which would fail
// the uuid cast.
func attemptNullableID(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return v
}

func intPtr(v int) *int    { return &v }
func boolPtr(v bool) *bool { return &v }
