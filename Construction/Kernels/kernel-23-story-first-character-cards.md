# Kernel 23 Story-First Character Cards

## What is enforced

- Character cards are personas, not alternate accounts.
- Account identity remains the accountable actor identity.
- Producers and directors can draft character cards implicitly.
- Producers inherit director-level character drafting authority.
- Cast and crew can draft only after receiving `character_card:draft`.
- Equipping a character updates The Cave presence roster and future action actor projection.
- Unequipping returns the user to account-only projection.

## Data model

- `character_cards`
  - owner user
  - location / optional production scope
  - name, pronouns, portrait URL, color, tagline, public description, private notes
- `permission_grants`
  - generic capability grants
  - Kernel 23 capability: `character_card:draft`
- `current_session_personas`
  - one active character card per user per session

## Live actions

- `persona/equip`
- `persona/unequip`

Persona actions are recorded in the append-only action log and attach to the active showing.

## Interfaces

- `GET /api/character-cards/me`
- `POST /api/character-cards`
- `PATCH /api/character-cards/{id}`
- `POST /api/character-card-permissions`
- `POST /api/character-card-permissions/revoke`
- WebSocket `persona/equip`
- WebSocket `persona/unequip`
- WebSocket `presence/update`

## Frontend surfaces

- The Cave has a compact Character panel.
- Users with drafting authority can create or edit their own character cards.
- Users with character cards can put on or take off a character in the current Cave session.
- Producer/director grant controls are intentionally minimal in v1.

## Not in this kernel

- Dice
- Grid
- Tokens
- Stats
- HP
- Initiative
- Fog of war
- Greenroom decompression flow
