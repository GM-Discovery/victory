package characters

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Chapter3Fact is the canonical, effective Chapter 3 archetype selection
// stored in workbook_context["chapter3"].
type Chapter3Fact struct {
	RulesetID               string `json:"ruleset_id"`
	RulesetVersion          string `json:"ruleset_version"`
	ArchetypeDatasetVersion string `json:"archetype_dataset_version"`
	ArchetypeStableID       string `json:"archetype_stable_id"`
	ArchetypeKey            string `json:"archetype_key"`
	ArchetypeTitle          string `json:"archetype_title"`
	ArchetypeCode           string `json:"archetype_code"`
	PrimaryAttribute        string `json:"primary_attribute"`
	SecondaryAttribute      string `json:"secondary_attribute"`
	KeySkill                string `json:"key_skill"`
	Confirmed               bool   `json:"confirmed"`
	ConfirmedAt             string `json:"confirmed_at"`
	SelectionCount          int    `json:"selection_count"`
}

// Chapter3SelectResult is returned to the client after a confirm request.
type Chapter3SelectResult struct {
	ArchetypeKey   string `json:"archetype_key"`
	ArchetypeTitle string `json:"archetype_title"`
	Completed      bool   `json:"completed"`
	NextStage      int    `json:"next_stage"`
	Overridden     bool   `json:"overridden"`
}

func chapter2Complete(wbContext map[string]any) bool {
	raw, ok := wbContext["chapter2"].(map[string]any)
	if !ok {
		return false
	}
	stages, ok := raw["stages"].(map[string]any)
	if !ok {
		return false
	}
	stage10, ok := stages["10"].(map[string]any)
	if !ok {
		return false
	}
	completed, _ := stage10["completed"].(bool)
	return completed
}

func loadChapter3Fact(wbContext map[string]any) (Chapter3Fact, bool) {
	raw, ok := wbContext["chapter3"]
	if !ok {
		return Chapter3Fact{}, false
	}
	rawBytes, err := json.Marshal(raw)
	if err != nil {
		return Chapter3Fact{}, false
	}
	var fact Chapter3Fact
	if err := json.Unmarshal(rawBytes, &fact); err != nil {
		return Chapter3Fact{}, false
	}
	return fact, fact.Confirmed
}

// CommitChapter3Archetype validates and records the player's (or an
// authorized override actor's) final Chapter 3 archetype selection. A
// normal confirm is idempotent: re-submitting the same already-confirmed
// archetype returns the existing fact without modification. An override
// (director/permitted-actor correction) always appends a new selection,
// which becomes the new effective archetype going forward.
func CommitChapter3Archetype(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID, archetypeKey string, override bool) (Chapter3SelectResult, map[string]any, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	cardID = strings.TrimSpace(cardID)
	archetypeKey = strings.TrimSpace(archetypeKey)

	if actorUserID == "" {
		return Chapter3SelectResult{}, nil, errors.New("not_authenticated")
	}
	if cardID == "" {
		return Chapter3SelectResult{}, nil, errors.New("character_card_id_required")
	}
	if archetypeKey == "" {
		return Chapter3SelectResult{}, nil, errors.New("archetype_key_required")
	}

	archetype, ok := Chapter3ArchetypeByKey(archetypeKey)
	if !ok {
		return Chapter3SelectResult{}, nil, errors.New("unknown_archetype_key")
	}

	allowed, err := CanEditCard(ctx, pool, actorUserID, cardID)
	if err != nil {
		return Chapter3SelectResult{}, nil, err
	}
	if !allowed {
		return Chapter3SelectResult{}, nil, errors.New("forbidden")
	}

	var wbContextRaw []byte
	var cardName string
	if err := pool.QueryRow(ctx, `
		SELECT workbook_context, name FROM character_cards WHERE id = $1 AND is_deleted = FALSE
	`, cardID).Scan(&wbContextRaw, &cardName); err != nil {
		return Chapter3SelectResult{}, nil, err
	}
	wbContext := decodeJSONMap(wbContextRaw)

	if !chapter2Complete(wbContext) {
		return Chapter3SelectResult{}, nil, errors.New("chapter2_not_complete")
	}

	existing, alreadyConfirmed := loadChapter3Fact(wbContext)
	if alreadyConfirmed && !override {
		return Chapter3SelectResult{
			ArchetypeKey:   existing.ArchetypeKey,
			ArchetypeTitle: existing.ArchetypeTitle,
			Completed:      true,
			NextStage:      4,
			Overridden:     false,
		}, wbContext, nil
	}

	selectionCount := existing.SelectionCount + 1
	fact := Chapter3Fact{
		RulesetID:               "socio",
		RulesetVersion:          Chapter3RulesetVersion,
		ArchetypeDatasetVersion: Chapter3ArchetypeDatasetVersion,
		ArchetypeStableID:       archetype.ID,
		ArchetypeKey:            archetype.Key,
		ArchetypeTitle:          archetype.Title,
		ArchetypeCode:           archetype.Code,
		PrimaryAttribute:        archetype.PrimaryAttribute,
		SecondaryAttribute:      archetype.SecondaryAttribute,
		KeySkill:                archetype.KeySkill,
		Confirmed:               true,
		ConfirmedAt:             time.Now().UTC().Format(time.RFC3339),
		SelectionCount:          selectionCount,
	}

	wbContext["chapter3"] = fact
	wbContext["current_stage"] = 4
	wbContext["current_event"] = "chapter3_archetype_confirmed"

	payloadJSON, err := json.Marshal(wbContext)
	if err != nil {
		return Chapter3SelectResult{}, nil, err
	}
	if _, err := pool.Exec(ctx, `
		UPDATE character_cards SET workbook_context = $2::jsonb, updated_at = NOW()
		WHERE id = $1 AND is_deleted = FALSE
	`, cardID, string(payloadJSON)); err != nil {
		return Chapter3SelectResult{}, nil, err
	}

	_, _ = pool.Exec(ctx, `
		UPDATE character_workbook_modules
		SET current_stage = 4, current_event = 'chapter3_archetype_confirmed', updated_at = NOW()
		WHERE character_card_id = $1 AND ruleset_key = 'socio'
	`, cardID)

	displayName := strings.TrimSpace(cardName)
	if displayName == "" {
		displayName = "The character"
	}
	archetypeNoun := strings.TrimSpace(strings.TrimPrefix(archetype.Title, "The"))
	historyBody := displayName + " selected " + archetypeNoun + " for their archetype."

	entryType := "archetype_selection"
	if alreadyConfirmed && override {
		entryType = "archetype_override"
	}

	if _, err := RecordWorkbookEvents(ctx, pool, actorUserID, WorkbookEventRequest{
		CharacterCardID: cardID,
		ModuleKey:       "socio",
		ModuleStatus:    "draft",
		CurrentStage:    4,
		CurrentEvent:    "chapter3_archetype_confirmed",
		Entries: []WorkbookEventInput{
			{
				PageKey:     "history",
				EntryType:   entryType,
				Title:       "Chapter III: Archetype selected",
				Body:        historyBody,
				StageNumber: 3,
				SortOrder:   selectionCount,
				Payload: map[string]any{
					"source":              "catharsis",
					"ruleset_key":         "socio",
					"ruleset_version":     Chapter3RulesetVersion,
					"archetype_key":       fact.ArchetypeKey,
					"archetype_stable_id": fact.ArchetypeStableID,
				},
			},
		},
	}); err != nil {
		return Chapter3SelectResult{}, nil, err
	}

	return Chapter3SelectResult{
		ArchetypeKey:   fact.ArchetypeKey,
		ArchetypeTitle: fact.ArchetypeTitle,
		Completed:      true,
		NextStage:      4,
		Overridden:     alreadyConfirmed && override,
	}, wbContext, nil
}
