package characters

import (
	"context"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/ewrite"
)

// applyFaceOverrides layers Director value overrides and owner/Director
// curation metadata onto the contract-inferred Face field list. It sets
// FaceVisible so callers can filter without re-deriving the mode string.
func applyFaceOverrides(fields []WorkbookPageField, overrides map[string]FaceOverride) []WorkbookPageField {
	out := make([]WorkbookPageField, len(fields))
	for i, f := range fields {
		o, hasOverride := overrides[f.Key]
		mode := faceVisibilityInferred
		if hasOverride && o.VisibilityMode != "" {
			mode = o.VisibilityMode
		}
		f.FaceVisibilityMode = mode
		f.FaceVisible = mode != faceVisibilityHidden
		if hasOverride {
			f.VisibilityLocked = o.VisibilityLocked
			f.ValueLocked = o.ValueLocked
			if o.ValueOverrideActive {
				f.Value = o.ValueOverride
				f.SourceKind = "director_override"
			}
			if o.PriorityMode == facePriorityManual {
				f.PriorityMode = facePriorityManual
				f.PriorityScore = o.PriorityScore
				f.PriorityBand = priorityBandForScore(o.PriorityScore)
				f.PriorityLocked = o.PriorityLocked
			}
		}
		out[i] = f
	}
	return out
}

// sortFaceFields orders fields by descending priority score, then a stable
// label tie-break (Kernel 59A §7.2 -- deterministic event-ID tie-breaking is
// deferred since nothing produces colliding scores with the same label yet).
func sortFaceFields(fields []WorkbookPageField) {
	sort.SliceStable(fields, func(i, j int) bool {
		if fields[i].PriorityScore != fields[j].PriorityScore {
			return fields[i].PriorityScore > fields[j].PriorityScore
		}
		return fields[i].Label < fields[j].Label
	})
}

// resolveEffectiveFaceFields is the pure half of the shared computation:
// apply overrides to an already-loaded eligible-field list, keep only what's
// visible, and sort for display. Split out from EffectiveFaceFields so a
// caller that already has the fields in hand (Greenroom, which already ran
// buildWorkbookPages for the rest of the page set) doesn't re-fetch and
// re-derive them a second time.
func resolveEffectiveFaceFields(fields []WorkbookPageField, overrides map[string]FaceOverride) []WorkbookPageField {
	fields = applyFaceOverrides(fields, overrides)

	visible := make([]WorkbookPageField, 0, len(fields))
	for _, f := range fields {
		if f.FaceVisible {
			visible = append(visible, f)
		}
	}
	sortFaceFields(visible)
	return visible
}

// EffectiveFaceFields is the one shared computation Greenroom and the venue
// sheet both call: contract-inferred Face fields with owner overrides
// applied, filtered to what's actually visible right now, sorted for
// display (Kernel 59A §3.6, §8.1 -- "one shared character-sheet projector...
// so both responses are derived" from it, instead of each maintaining its
// own Face-eligibility rules).
func EffectiveFaceFields(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID string) ([]WorkbookPageField, error) {
	fields, err := resolveFaceEligibility(ctx, pool, actorUserID, cardID)
	if err != nil {
		return nil, err
	}
	overrides, err := loadFaceOverrides(ctx, pool, cardID)
	if err != nil {
		return nil, err
	}
	return resolveEffectiveFaceFields(fields, overrides), nil
}

// CharacterSheetAtAGlanceLimit caps the "At a Glance" region so it stays a
// highlight rather than a duplicate of the whole sheet (Kernel 59A §7.3).
const CharacterSheetAtAGlanceLimit = 6

// CharacterSheetProjection is the canonical shape both Greenroom (full) and
// the venue right tray (trimmed further in venue_sheet.go) derive from
// (Kernel 59A §8.1). UrgentState is intentionally absent: no fact contract
// produces an urgent-eligible fact yet, so there is nothing to project --
// add it here first when one exists, rather than inventing an always-empty
// field now.
type CharacterSheetProjection struct {
	ProjectionVersion string                         `json:"projection_version"`
	Identity          []WorkbookPageField            `json:"identity"`
	AtAGlance         []WorkbookPageField            `json:"at_a_glance"`
	DetailSections    map[string][]WorkbookPageField `json:"detail_sections"`
	Attributes        map[string]int                 `json:"attributes"`
	Skills            []CharacterSkill               `json:"skills"`
	// SkillRuleLinks is the Skill Directory deep-link metadata for Skills,
	// keyed by skill_id (Kernel 79A goal 6.1) -- lets the venue right tray
	// offer "View rule" during actual play, not just the Greenroom.
	SkillRuleLinks map[string]ewrite.RuleLink `json:"skill_rule_links,omitempty"`
}

// CharacterProjectionVersion is a durable, comparable server version for the
// effective projected sheet. It advances when the card, Face overrides, or
// Mechanics skills change, without minting a separate cache/revision table.
func CharacterProjectionVersion(ctx context.Context, pool *pgxpool.Pool, cardID string) (string, error) {
	var version time.Time
	if err := pool.QueryRow(ctx, `
		SELECT GREATEST(
			cc.updated_at,
			COALESCE((SELECT MAX(cfo.updated_at) FROM character_face_overrides cfo WHERE cfo.character_card_id = cc.id), cc.updated_at),
			COALESCE((SELECT MAX(cs.updated_at) FROM character_skills cs WHERE cs.character_card_id = cc.id), cc.updated_at)
		)
		FROM character_cards cc
		WHERE cc.id = $1
		  AND cc.is_deleted = FALSE
	`, cardID).Scan(&version); err != nil {
		return "", err
	}
	return version.UTC().Format(time.RFC3339Nano), nil
}

// ProjectCharacterSheet computes the shared projection for a character:
// effective (override-applied, visible-only) Face fields grouped into fixed
// regions, plus Mechanics data (attributes + Kernel 60 skills). Both
// LoadCharacterWorkbookView (Greenroom) and BuildVenueCharacterSheet (venue
// right tray) call this rather than each re-deriving Face eligibility,
// visibility, and the skill list on their own (Kernel 59A §3.6, §8.1).
func ProjectCharacterSheet(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID string) (CharacterSheetProjection, error) {
	visible, err := EffectiveFaceFields(ctx, pool, actorUserID, cardID)
	if err != nil {
		return CharacterSheetProjection{}, err
	}

	proj := CharacterSheetProjection{
		DetailSections: map[string][]WorkbookPageField{},
	}
	glance := make([]WorkbookPageField, 0, len(visible))
	for _, f := range visible {
		switch f.Region {
		case "identity":
			proj.Identity = append(proj.Identity, f)
		case "glance":
			glance = append(glance, f)
		default:
			region := f.Region
			if region == "" {
				region = "additional"
			}
			proj.DetailSections[region] = append(proj.DetailSections[region], f)
		}
	}
	if len(glance) > CharacterSheetAtAGlanceLimit {
		glance = glance[:CharacterSheetAtAGlanceLimit]
	}
	proj.AtAGlance = glance

	var wbContextRaw []byte
	if err := pool.QueryRow(ctx, `
		SELECT workbook_context FROM character_cards WHERE id = $1 AND is_deleted = FALSE
	`, cardID).Scan(&wbContextRaw); err != nil {
		return CharacterSheetProjection{}, err
	}
	proj.Attributes = loadChapter2Attributes(decodeJSONMap(wbContextRaw))

	skills, err := ListCharacterSkills(ctx, pool, actorUserID, cardID)
	if err != nil {
		return CharacterSheetProjection{}, err
	}
	proj.Skills = skills
	skillIDs := make([]string, len(skills))
	for i, s := range skills {
		skillIDs[i] = s.SkillID
	}
	skillRuleLinks, err := ewrite.RuleLinksForCharacterSkills(ctx, pool, skillIDs)
	if err != nil {
		return CharacterSheetProjection{}, err
	}
	proj.SkillRuleLinks = skillRuleLinks

	version, err := CharacterProjectionVersion(ctx, pool, cardID)
	if err != nil {
		return CharacterSheetProjection{}, err
	}
	proj.ProjectionVersion = version

	return proj, nil
}
