# Character Workbook

## Canonical Root
- The canonical character root is `character_cards`.
- Kernel 53 extends that root instead of introducing a parallel workbook table.
- Workbook metadata now lives on the card row through `workbook_status` and `workbook_context`.

## Ownership And Active State
- A workbook belongs to its creator.
- The current active workbook for a user is tracked in `active_user_characters`.
- Catharsis creation now writes the new workbook into the active slot so the shell can project it immediately.

## Workbook Modules
- Each workbook can have module instances in `character_workbook_modules`.
- Kernel 53 seeds a Catharsis `socio` module with draft status and Stage 1 parentage metadata.
- Later productions can add their own module rows without changing the canonical root.

## History And Journal
- Append-only workbook facts belong in character-entry style records later.
- Private player notes belong in `character_journals`.
- Journals do not become canonical history automatically.

## Greenroom
- Greenroom becomes visible once the user owns at least one workbook.
- Draft workbooks stay visible and resumable.

## Active Character Surface
- Catharsis and First Theater now show the active workbook in shell chrome.
- Clicking the shell chip opens Greenroom on the loaded workbook when one is active.

## Catharsis Draft Flow
- The Catharsis start button creates or resumes the onboarding draft.
- The draft is keyed by workbook context so repeated clicks do not duplicate the draft.
- The flow stays resumable instead of requiring a finished character.

## Journal Command
- `/journal <text>` bypasses venue chat.
- The command writes to the private workbook journal for the active character.

## Deferred Work
- The Stage 1 Parentage chart and full workbook compiler remain deferred until the versioned chart data is supplied.
- Stage 2 mechanics are still out of scope for this kernel.
