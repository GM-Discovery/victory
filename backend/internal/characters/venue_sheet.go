package characters

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// VenueSkillEntry is one row of the right-tray sheet's skill list (Kernel 60
// §8). Expression is the exploding form -- clicking a skill rolls this.
type VenueSkillEntry struct {
	SkillID          string `json:"skill_id"`
	SkillName        string `json:"skill_name"`
	AttributeName    string `json:"attribute_name"`
	LadderStep       int    `json:"ladder_step"`
	Expression       string `json:"expression"`
	ImprovementCount int    `json:"improvement_count"`
}

// VenueFactEntry is a compact, editing-metadata-free narrative fact for the
// venue "At a Glance" section -- the venue trims Greenroom's
// WorkbookPageField (which carries Editable/InputType/Placeholder, all
// meaningless in a read-only venue context) down to just what's worth
// showing (Kernel 59A §8.3).
type VenueFactEntry struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Value string `json:"value"`
}

// VenueCharacterSheet is the compact, public-safe projection the venue
// right-tray reads (Kernel 60 §8, Kernel 59A §8.3: derived from the same
// shared ProjectCharacterSheet Greenroom uses, "not a second source of
// truth. No private fields (journal, private notes) in the venue payload").
type VenueCharacterSheet struct {
	CharacterCardID   string            `json:"character_card_id"`
	ProjectionVersion string            `json:"projection_version"`
	Name              string            `json:"name"`
	Pronouns          string            `json:"pronouns"`
	PortraitURL       string            `json:"portrait_url"`
	TokenAura         string            `json:"token_aura"`
	Color             string            `json:"color"`
	AtAGlance         []VenueFactEntry  `json:"at_a_glance"`
	Attributes        map[string]int    `json:"attributes"`
	Skills            []VenueSkillEntry `json:"skills"`
}

// BuildVenueCharacterSheet assembles the right-tray sheet for a character
// from the shared ProjectCharacterSheet projection (Kernel 59A §3.6, §8.1),
// so a fact hidden or reprioritized in Greenroom resolves the same way here.
func BuildVenueCharacterSheet(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID string) (VenueCharacterSheet, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	cardID = strings.TrimSpace(cardID)
	if actorUserID == "" {
		return VenueCharacterSheet{}, errors.New("not_authenticated")
	}
	if cardID == "" {
		return VenueCharacterSheet{}, errors.New("character_card_id_required")
	}

	card, err := LoadCardForActor(ctx, pool, actorUserID, cardID)
	if err != nil {
		return VenueCharacterSheet{}, err
	}

	proj, err := ProjectCharacterSheet(ctx, pool, actorUserID, cardID)
	if err != nil {
		return VenueCharacterSheet{}, err
	}

	glance := make([]VenueFactEntry, 0, len(proj.AtAGlance))
	for _, f := range proj.AtAGlance {
		if strings.TrimSpace(f.Value) == "" {
			continue
		}
		glance = append(glance, VenueFactEntry{Key: f.Key, Label: f.Label, Value: f.Value})
	}

	skillEntries := make([]VenueSkillEntry, 0, len(proj.Skills))
	for _, s := range proj.Skills {
		expr, err := StepExpression(s.LadderStep, true)
		if err != nil {
			return VenueCharacterSheet{}, err
		}
		skillEntries = append(skillEntries, VenueSkillEntry{
			SkillID:          s.SkillID,
			SkillName:        s.SkillName,
			AttributeName:    s.AttributeName,
			LadderStep:       s.LadderStep,
			Expression:       expr,
			ImprovementCount: s.ImprovementCount,
		})
	}

	return VenueCharacterSheet{
		CharacterCardID:   card.ID,
		ProjectionVersion: proj.ProjectionVersion,
		Name:              card.Name,
		Pronouns:          card.Pronouns,
		PortraitURL:       card.PortraitURL,
		TokenAura:         card.TokenAura,
		Color:             card.Color,
		AtAGlance:         glance,
		Attributes:        proj.Attributes,
		Skills:            skillEntries,
	}, nil
}
