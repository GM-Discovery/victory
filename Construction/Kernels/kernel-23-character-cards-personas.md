# Kernel 23 - Character Cards And Personas

## Expected Purpose
Add story-first character cards plus session persona equip/unequip behavior.

## Likely Delivered Behavior
- `character_cards` and `current_session_personas` exist
- users can own character cards
- Cave supports `persona/equip` and `persona/unequip`
- presence can update when persona changes
- action actor projection can carry persona state

## Evidence Source
- [kernel-23-story-first-character-cards.md](/opt/victory/Construction/Kernels/kernel-23-story-first-character-cards.md)
- [characters.go](/opt/victory/backend/internal/characters/characters.go)
- [persona.go](/opt/victory/backend/internal/actions/persona.go)
- [ws.go](/opt/victory/backend/internal/network/ws.go)

## TODO
- Older notes still mention draft grants as part of the main behavior, but current Greenroom drafting no longer depends on them.
