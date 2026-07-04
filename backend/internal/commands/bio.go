package commands

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/characters"
)

// ExecuteBioSet replaces the active character's biography
// (character_cards.public_description, labeled "Biography" on the Face/Bio
// workbook page -- distinct from the "tagline"/"Featured Quote" field that
// /quote controls). Same read-modify-write requirement as /char set.
func ExecuteBioSet(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID, text string) (characters.CharacterCard, error) {
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
		PublicDescription: strings.TrimSpace(text),
		PrivateNotes:      card.PrivateNotes,
		SheetLinks:        &sheetLinks,
		WorkbookStatus:    card.WorkbookStatus,
		WorkbookContext:   card.WorkbookContext,
	}
	return characters.UpdateCard(ctx, pool, actorUserID, cardID, input)
}
