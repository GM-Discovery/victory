package commands

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/characters"
)

// charSettableFields is the explicit allowlist guard that characters.UpdateCard
// itself does not have (UpdateCard will happily overwrite any column,
// including protected onboarding facts, if a caller assembles the input).
// Anything not in this map -- archetype, attributes, skill dice, ruleset,
// onboarding status, etc. -- is rejected with "protected_fact".
var charSettableFields = map[string]func(*characters.CharacterCardInput, string){
	"name":     func(in *characters.CharacterCardInput, v string) { in.Name = v },
	"pronouns": func(in *characters.CharacterCardInput, v string) { in.Pronouns = v },
	"aura":     func(in *characters.CharacterCardInput, v string) { in.TokenAura = v },
}

var charNavigatePages = map[string]bool{
	"face":      true,
	"mechanics": true,
	"history":   true,
	"journal":   true,
}

// ExecuteCharSet performs a read-modify-write against characters.UpdateCard:
// load the full current card, copy every field into a CharacterCardInput,
// patch only the targeted field, and write the whole row back. This is
// required because UpdateCard has no partial-patch path -- passing an input
// with only one field set would blank every other column.
func ExecuteCharSet(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID, field, value string) (characters.CharacterCard, error) {
	field = strings.ToLower(strings.TrimSpace(field))
	mutate, ok := charSettableFields[field]
	if !ok {
		return characters.CharacterCard{}, errors.New("protected_fact")
	}

	card, err := characters.LoadCardForActor(ctx, pool, actorUserID, cardID)
	if err != nil {
		return characters.CharacterCard{}, err
	}

	sheetLinks := append([]characters.SheetLink{}, card.SheetLinks...)
	input := characters.CharacterCardInput{
		Name:              card.Name,
		Pronouns:          card.Pronouns,
		PortraitURL:       card.PortraitURL,
		TokenAura:         card.TokenAura,
		Color:             card.Color,
		Tagline:           card.Tagline,
		PublicDescription: card.PublicDescription,
		PrivateNotes:      card.PrivateNotes,
		SheetLinks:        &sheetLinks,
		WorkbookStatus:    card.WorkbookStatus,
		WorkbookContext:   card.WorkbookContext,
	}
	mutate(&input, strings.TrimSpace(value))

	return characters.UpdateCard(ctx, pool, actorUserID, cardID, input)
}

// ExecuteCharNavigate validates the requested workbook page. It performs no
// mutation -- the HTTP layer turns this into a navigation result pointing at
// the resolved active character's workbook.
func ExecuteCharNavigate(page string) (string, error) {
	page = strings.ToLower(strings.TrimSpace(page))
	if page == "" || page == "char" {
		page = "face"
	}
	if !charNavigatePages[page] {
		return "", errors.New("unknown_command")
	}
	return page, nil
}
