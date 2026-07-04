package commands

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/characters"
)

// ExecuteQuoteSet replaces the active character's single "tagline"/"Featured
// Quote" field. There is no multi-quote collection store in this codebase
// today, so /quote add|feature|retire are explicitly out of scope -- this
// replaces the one existing singleton only.
func ExecuteQuoteSet(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID, text string) (characters.CharacterCard, error) {
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
		Tagline:           strings.TrimSpace(text),
		PublicDescription: card.PublicDescription,
		PrivateNotes:      card.PrivateNotes,
		SheetLinks:        &sheetLinks,
		WorkbookStatus:    card.WorkbookStatus,
		WorkbookContext:   card.WorkbookContext,
	}
	return characters.UpdateCard(ctx, pool, actorUserID, cardID, input)
}
