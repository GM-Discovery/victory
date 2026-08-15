package socio

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
)

const (
	// TierOwner is the Player currently bound to this Character via their
	// own Show Run roster selection.
	TierOwner = "owner"
	// TierDirector is Director+/Producer/Operator authority on this Show.
	TierDirector = "director"
)

// QualitativePool is one HP pool shaped for non-exact display -- the
// spoiler-safe form every viewer gets by default (kernel-88 spec §6.2,
// §7).
type QualitativePool struct {
	Key       PoolKey `json:"key"`
	Label     string  `json:"label"`
	Condition string  `json:"condition"`
}

// FlagView is one active canonical status or Kernel 88 blank state, shaped
// for display -- no distinction in visibility between tiers (neither is
// Audience-facing to begin with; both Owner and Director may see a
// Character's own flags).
type FlagView struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	IsBlank bool   `json:"is_blank"`
}

// SocioProjection is one Character's Socio play state, shaped for the
// resolved viewer Tier. This is the single function every non-raw read path
// must go through (kernel-88 spec §24): a Player can never fetch another
// Character's exact HP numbers, and even for their own Character the
// default UI shape is qualitative, matching the hide-the-ball product rule.
// There is no Audience tier here -- Audience never queries Socio mechanical
// state at all; Audience-facing output is the existing stage/Cave
// projection pipeline (Kernel 81/86), untouched by this kernel.
type SocioProjection struct {
	CharacterCardID  string            `json:"character_card_id"`
	Tier             string            `json:"tier"`
	FateBalance      int               `json:"fate_balance"`
	FateCreationMode bool              `json:"fate_creation_mode"`
	StanceKey        string            `json:"stance_key,omitempty"`
	StanceLabel      string            `json:"stance_label,omitempty"`
	QualitativePools []QualitativePool `json:"qualitative_pools"`
	ExactPools       []Pool            `json:"exact_pools,omitempty"`
	Flags            []FlagView        `json:"flags"`
}

// ResolveTier determines whether viewerUserID may see characterCardID's
// Socio state at all, and at which tier. Returns "not_authorized" for
// anyone else -- there is deliberately no "view a teammate's mechanical
// state" surface in this kernel; a Player seeing anything about another
// Character during a scene comes from the stage/Cave channel, not this
// function.
func ResolveTier(ctx context.Context, pool *pgxpool.Pool, showID, characterCardID, viewerUserID string) (string, error) {
	if err := requireShowCharacterAuthority(ctx, pool, viewerUserID, showID, characterCardID); err == nil {
		return TierDirector, nil
	}

	s, err := shows.LoadShowByID(ctx, pool, showID)
	if err != nil {
		return "", err
	}
	member, err := showruns.LoadMyRosterMember(ctx, pool, viewerUserID, s.ShowRunID)
	if err != nil {
		return "", errors.New("not_authorized")
	}
	if member.CharacterCardID == "" || member.CharacterCardID != characterCardID {
		return "", errors.New("not_authorized")
	}
	return TierOwner, nil
}

// ProjectSocioState resolves viewerUserID's tier for characterCardID and
// returns their tier-shaped view of that Character's Fate, Stance, pools,
// and flags. Callers (HTTP handlers) must always call this instead of
// composing GetState/GetFate/GetStance/ListActiveStatuses directly for any
// non-Director-only surface -- that's what makes the security boundary
// structural (kernel-88 spec §24) rather than a per-handler judgment call.
func ProjectSocioState(ctx context.Context, pool *pgxpool.Pool, viewerUserID, showID, characterCardID string) (SocioProjection, error) {
	tier, err := ResolveTier(ctx, pool, showID, characterCardID, viewerUserID)
	if err != nil {
		return SocioProjection{}, err
	}

	state, err := GetState(ctx, pool, characterCardID)
	if err != nil {
		return SocioProjection{}, err
	}
	fate, err := GetFate(ctx, pool, characterCardID)
	if err != nil {
		return SocioProjection{}, err
	}
	stance, err := GetStance(ctx, pool, characterCardID)
	if err != nil {
		return SocioProjection{}, err
	}
	activeStatuses, err := ListActiveStatuses(ctx, pool, characterCardID)
	if err != nil {
		return SocioProjection{}, err
	}

	proj := SocioProjection{
		CharacterCardID:  characterCardID,
		Tier:             tier,
		FateBalance:      fate.Balance,
		FateCreationMode: fate.CreationMode,
		StanceKey:        stance.Key,
		StanceLabel:      stance.Label,
	}
	for _, p := range state.Pools {
		proj.QualitativePools = append(proj.QualitativePools, QualitativePool{
			Key: p.Key, Label: p.Label, Condition: QualitativeLabel(p.Current, p.Max),
		})
	}
	if tier == TierDirector {
		proj.ExactPools = state.Pools
	}
	for _, a := range activeStatuses {
		proj.Flags = append(proj.Flags, FlagView{ID: a.ID, Label: a.Label, IsBlank: a.IsBlank})
	}
	return proj, nil
}
