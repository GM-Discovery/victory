# Kernel 24 - Greenroom Character Dressing Room

## Summary
Kernel 24 moves character drafting out of The Cave and into The Greenroom. The Cave keeps only live-session persona use: select an existing character, put it on, take it off.

## Implemented Shape
- Greenroom now has a Character Dressing Room card with character selection, New, Edit, read-only display, and sheet link display.
- Greenroom reveals the left tool column only when Edit/New or the profile Edit button is used.
- Character editor is hidden until New or Edit is clicked.
- Character cards can save story-first fields plus `sheet_links`.
- Sheet links are metadata only: `id`, `ruleset`, `sheet_type`, `label`, `url`, `created_at`.
- Producer/director character-drafting grant controls moved to Greenroom tooling.
- The Cave character panel no longer exposes full card creation/editing controls.
- The Cave still uses existing `persona/equip`, `persona/unequip`, and `presence/update`.

## Deferred
- Playable character sheet records.
- Sheet page rendering.
- Ruleset macro/tool loading.
- Dice, stats, HP, initiative, grid, tokens, and fog.
- Greenroom decompression flow.
