package storysofar

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LoadInputs gathers exactly S5.1's permitted source data for one
// (Character, Show) participation.
//
// Every query here is scoped to the Character, and the Show-scoped ones are
// scoped to the Show as well. A Character who played the same tutorial twice
// in two Shows gets two independent reflections, and Character A never sees
// Character B's data even for the same user -- the same (user, Character,
// Show) discipline Kernel 74 established.
//
// Missing data is normal, not exceptional. A Character created before
// Kernel 75 has no character_interaction_attempts rows; one who never opened
// their workbook has no stage lines. Every one of those simply yields an
// empty slice, and the template library omits the corresponding clause.
func LoadInputs(ctx context.Context, pool *pgxpool.Pool, ref Ref, rules Rules, completedAt time.Time) (Inputs, error) {
	if err := ref.valid(); err != nil {
		return Inputs{}, err
	}
	in := Inputs{
		CharacterCardID: ref.CharacterCardID,
		OwnerUserID:     ref.OwnerUserID,
		ShowRunID:       ref.ShowRunID,
		ShowID:          ref.ShowID,
		SessionID:       ref.SessionID,
		PlacementID:     ref.PlacementID,
		Attributes:      map[string]int{},
		Milestones:      map[string]bool{},
		CompletedAt:     completedAt,
	}

	// --- Character identity, archetype, attributes -------------------------
	var pronouns string
	var workbookContext []byte
	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(NULLIF(name, ''), ''), COALESCE(pronouns, ''), workbook_context
		FROM character_cards
		WHERE id = $1 AND is_deleted = FALSE
	`, ref.CharacterCardID).Scan(&in.CharacterName, &pronouns, &workbookContext); err != nil {
		return Inputs{}, err
	}
	in.Pronouns = pronouns

	var wb map[string]any
	if len(workbookContext) > 0 {
		_ = json.Unmarshal(workbookContext, &wb)
	}
	applyChapter3(&in, wb, rules)
	applyChapter2Attributes(&in, wb)

	// --- Face Sheet life-stage lines ---------------------------------------
	stageRows, err := pool.Query(ctx, `
		SELECT id::text, COALESCE(title, ''), COALESCE(body, ''),
		       COALESCE(stage_number, 0), COALESCE(sort_order, 0)
		FROM character_workbook_entries
		WHERE character_card_id = $1
		  AND page_key = 'history'
		  AND entry_type = 'chapter2_stage'
		ORDER BY stage_number ASC, sort_order ASC, id ASC
	`, ref.CharacterCardID)
	if err != nil {
		return Inputs{}, err
	}
	for stageRows.Next() {
		var s StageLine
		if err := stageRows.Scan(&s.ID, &s.Title, &s.Body, &s.StageNumber, &s.SortOrder); err != nil {
			stageRows.Close()
			return Inputs{}, err
		}
		in.StageLines = append(in.StageLines, s)
	}
	stageRows.Close()
	if err := stageRows.Err(); err != nil {
		return Inputs{}, err
	}

	// --- Tutorial milestones ------------------------------------------------
	if ref.ShowID != "" {
		mRows, err := pool.Query(ctx, `
			SELECT milestone_key
			FROM participant_tutorial_progress
			WHERE user_id = $1 AND character_card_id = $2 AND show_id = $3
		`, ref.OwnerUserID, ref.CharacterCardID, ref.ShowID)
		if err != nil {
			return Inputs{}, err
		}
		for mRows.Next() {
			var key string
			if err := mRows.Scan(&key); err != nil {
				mRows.Close()
				return Inputs{}, err
			}
			in.Milestones[key] = true
		}
		mRows.Close()
		if err := mRows.Err(); err != nil {
			return Inputs{}, err
		}
	}

	// --- Equipment acquired in THIS Show ------------------------------------
	//
	// Scoped by source_show_id, not by "everything this Character owns": the
	// reflection is about what happened here, and a Character arriving with
	// inherited equipment did not acquire it at Kessa's stall.
	if ref.ShowID != "" {
		invRows, err := pool.Query(ctx, `
			SELECT ci.id::text, COALESCE(NULLIF(ei.name, ''), 'an unnamed item'),
			       ci.quantity, ci.first_acquired_at
			FROM character_inventory_items ci
			JOIN equipment_items ei ON ei.id = ci.equipment_item_id
			WHERE ci.character_card_id = $1 AND ci.source_show_id = $2
			ORDER BY ci.first_acquired_at ASC, ei.name ASC, ci.id ASC
		`, ref.CharacterCardID, ref.ShowID)
		if err != nil {
			return Inputs{}, err
		}
		for invRows.Next() {
			var line InventoryLine
			if err := invRows.Scan(&line.InventoryItemID, &line.ItemName, &line.Quantity, &line.AcquiredAt); err != nil {
				invRows.Close()
				return Inputs{}, err
			}
			in.Inventory = append(in.Inventory, line)
		}
		invRows.Close()
		if err := invRows.Err(); err != nil {
			return Inputs{}, err
		}
	}

	// --- Door intention -----------------------------------------------------
	//
	// Read verbatim. Never trimmed of internal content, never rewritten,
	// never wrapped here -- see tmplDoorIntention.
	if ref.ShowID != "" {
		var intention string
		err := pool.QueryRow(ctx, `
			SELECT submitted_text
			FROM participant_freeform_submissions
			WHERE character_card_id = $1 AND show_id = $2 AND user_id = $3
			ORDER BY created_at ASC
			LIMIT 1
		`, ref.CharacterCardID, ref.ShowID, ref.OwnerUserID).Scan(&intention)
		if err == nil {
			in.DoorIntention = intention
		}
		// A missing row is the ordinary "never reached the door" case, not
		// an error. pgx returns ErrNoRows; every other error would also
		// leave DoorIntention empty and omit the clause, which is the safe
		// direction to fail.
	}

	// --- Kessa stance / Haggle attempts -------------------------------------
	if ref.ShowID != "" {
		aRows, err := pool.Query(ctx, `
			SELECT attempt_kind, packet_slug, stance_key, disposition,
			       COALESCE(response_tier, 0), skill_key, COALESCE(has_skill, FALSE),
			       die, COALESCE(total, 0), COALESCE(target_value, 0),
			       COALESCE(success, FALSE), created_at
			FROM character_interaction_attempts
			WHERE character_card_id = $1 AND show_id = $2
			ORDER BY created_at ASC, id ASC
		`, ref.CharacterCardID, ref.ShowID)
		if err != nil {
			return Inputs{}, err
		}
		for aRows.Next() {
			var a Attempt
			if err := aRows.Scan(&a.Kind, &a.PacketSlug, &a.StanceKey, &a.Disposition,
				&a.ResponseTier, &a.SkillKey, &a.HasSkill, &a.Die, &a.Total,
				&a.TargetValue, &a.Success, &a.CreatedAt); err != nil {
				aRows.Close()
				return Inputs{}, err
			}
			in.Attempts = append(in.Attempts, a)
		}
		aRows.Close()
		if err := aRows.Err(); err != nil {
			return Inputs{}, err
		}
	}

	// --- Ra topics actually read --------------------------------------------
	if ref.ShowID != "" {
		tRows, err := pool.Query(ctx, `
			SELECT dt.id::text, dt.topic_key, dt.label,
			       COALESCE(dt.required_for_completion, FALSE), v.viewed_at
			FROM participant_dialogue_topic_views v
			JOIN dialogue_topics dt ON dt.id = v.topic_id
			WHERE v.character_card_id = $1 AND v.show_id = $2 AND v.user_id = $3
			ORDER BY v.viewed_at ASC, dt.id ASC
		`, ref.CharacterCardID, ref.ShowID, ref.OwnerUserID)
		if err != nil {
			return Inputs{}, err
		}
		for tRows.Next() {
			var t DialogueTopicSeen
			if err := tRows.Scan(&t.TopicID, &t.TopicKey, &t.Label, &t.Required, &t.ViewedAt); err != nil {
				tRows.Close()
				return Inputs{}, err
			}
			in.Topics = append(in.Topics, t)
		}
		tRows.Close()
		if err := tRows.Err(); err != nil {
			return Inputs{}, err
		}
	}

	return in, nil
}

// applyChapter3 reads the archetype fact from workbook_context.
//
// The stored fact already carries primary_attribute / secondary_attribute /
// key_skill, copied from the catalog at confirmation time. Rules.Archetype
// is the repair path for a fact written before those fields existed, or one
// naming an archetype whose catalog entry has since been corrected -- the
// catalog stays authoritative, and this package still owns no copy of it.
func applyChapter3(in *Inputs, wb map[string]any, rules Rules) {
	raw, ok := wb["chapter3"]
	if !ok {
		return
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return
	}
	var fact struct {
		ArchetypeKey       string `json:"archetype_key"`
		ArchetypeTitle     string `json:"archetype_title"`
		PrimaryAttribute   string `json:"primary_attribute"`
		SecondaryAttribute string `json:"secondary_attribute"`
		KeySkill           string `json:"key_skill"`
		Confirmed          bool   `json:"confirmed"`
	}
	if err := json.Unmarshal(encoded, &fact); err != nil {
		return
	}
	in.ArchetypeKey = strings.TrimSpace(fact.ArchetypeKey)
	in.ArchetypeTitle = strings.TrimSpace(fact.ArchetypeTitle)
	in.PrimaryAttribute = strings.TrimSpace(fact.PrimaryAttribute)
	in.SecondaryAttribute = strings.TrimSpace(fact.SecondaryAttribute)
	in.KeySkill = strings.TrimSpace(fact.KeySkill)
	in.ArchetypeConfirmed = fact.Confirmed

	if in.ArchetypeKey != "" && rules.Archetype != nil {
		if a, ok := rules.Archetype(in.ArchetypeKey); ok {
			if in.ArchetypeTitle == "" {
				in.ArchetypeTitle = a.Title
			}
			if in.PrimaryAttribute == "" {
				in.PrimaryAttribute = a.PrimaryAttribute
			}
			if in.SecondaryAttribute == "" {
				in.SecondaryAttribute = a.SecondaryAttribute
			}
			if in.KeySkill == "" {
				in.KeySkill = a.KeySkill
			}
		}
	}
}

// applyChapter2Attributes mirrors characters.loadChapter2Attributes, which is
// unexported. The parse is deliberately tolerant: the Catharsis onboarding
// writes JSON numbers, but a hand-repaired workbook may hold strings, and an
// attribute this cannot parse is simply absent rather than zero -- a zero
// would be a claim ("Defeated") the data does not make.
func applyChapter2Attributes(in *Inputs, wb map[string]any) {
	raw, ok := wb["chapter2"].(map[string]any)
	if !ok {
		return
	}
	attrs, ok := raw["attributes"].(map[string]any)
	if !ok {
		return
	}
	for name, value := range attrs {
		switch v := value.(type) {
		case float64:
			in.Attributes[name] = int(v)
		case int:
			in.Attributes[name] = v
		case json.Number:
			if n, err := v.Int64(); err == nil {
				in.Attributes[name] = int(n)
			}
		}
	}
}
