package characters

import "testing"

func TestBuildWorkbookPagesIncludesCatharsisSummaryFields(t *testing.T) {
	card := CharacterCard{
		Name: "Test Character",
		WorkbookContext: map[string]any{
			"source":                        "catharsis",
			"socio_parentage_chart_version": ParentageChartVersionV11,
			"socio_parentage_roll":          72,
			"socio_parentage_parents": []any{
				map[string]any{
					"parent_index":       1,
					"roll_total":         22,
					"social_class":       "Tavern Keeper/Stable Hand",
					"starting_credit":    18,
					"coin_flip_result":   "not eligible",
					"inherited_wealth":   0,
					"coin_flip_eligible": false,
					"inheritance_passed": false,
				},
				map[string]any{
					"parent_index":       2,
					"roll_total":         52,
					"social_class":       "Banking Associate",
					"starting_credit":    260,
					"coin_flip_result":   "yes",
					"inherited_wealth":   260,
					"coin_flip_eligible": true,
					"inheritance_passed": true,
				},
			},
			"socio_starting_wealth":                     260,
			"socio_starting_wealth_source_parent_index": 2,
			"current_stage":                             2,
			"current_event":                             "childhood_stages",
		},
	}

	pages := buildWorkbookPages(card, nil, nil, nil)
	var face CharacterWorkbookPage
	for _, page := range pages {
		if page.Key == "face" {
			face = page
			break
		}
	}

	if face.Key != "face" {
		t.Fatalf("face page not found")
	}

	foundSummary := false
	foundWealth := false
	for _, field := range face.Fields {
		if field.Key == "socio_parentage_summary" {
			foundSummary = true
			if field.Value == "" {
				t.Fatalf("expected parentage summary to be populated")
			}
		}
		if field.Key == "socio_starting_wealth" {
			foundWealth = true
			if field.Value != "260" {
				t.Fatalf("starting wealth = %q, want 260", field.Value)
			}
		}
	}

	if !foundSummary {
		t.Fatalf("expected socio_parentage_summary field on face page")
	}
	if !foundWealth {
		t.Fatalf("expected socio_starting_wealth field on face page")
	}
}

func TestBuildWorkbookPagesIncludesChapter4SkillFields(t *testing.T) {
	card := CharacterCard{
		Name: "Test Character",
		WorkbookContext: map[string]any{
			"source": "catharsis",
			"chapter4": map[string]any{
				"skill_stable_id": "SKILL_CRAFT_ARTIFICE",
				"skill_name":      "Artifice",
				"attribute_id":    "ATTR_CRAFT",
				"attribute_name":  "Craft",
				"training_state":  "trained",
				"die_size":        "d6",
				"confirmed":       true,
			},
		},
	}

	pages := buildWorkbookPages(card, nil, nil, nil)
	var face, mechanics CharacterWorkbookPage
	for _, page := range pages {
		if page.Key == "face" {
			face = page
		}
		if page.Key == "mechanics" {
			mechanics = page
		}
	}

	faceHasSkill := false
	for _, field := range face.Fields {
		if field.Key == "chapter4_first_skill" && field.Value == "Artifice" {
			faceHasSkill = true
		}
	}
	if !faceHasSkill {
		t.Fatalf("expected chapter4_first_skill=Artifice on face page, got fields %+v", face.Fields)
	}

	mechanicsHasSkill := false
	mechanicsHasDie := false
	for _, field := range mechanics.Fields {
		if field.Key == "chapter4_skill_stable_id" && field.Value == "SKILL_CRAFT_ARTIFICE" {
			mechanicsHasSkill = true
		}
		if field.Key == "chapter4_die_size" && field.Value == "d6" {
			mechanicsHasDie = true
		}
	}
	if !mechanicsHasSkill {
		t.Fatalf("expected chapter4_skill_stable_id on mechanics page, got fields %+v", mechanics.Fields)
	}
	if !mechanicsHasDie {
		t.Fatalf("expected chapter4_die_size=d6 on mechanics page, got fields %+v", mechanics.Fields)
	}
}
