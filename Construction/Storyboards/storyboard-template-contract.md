# Storyboard Template/Instantiation Contract (Kernel 82)

## The contract, verbatim from spec §4.2

```text
instantiate template ≠ edit template
```

## Implementation choice: code-defined seed data, not a DB row

`backend/internal/storyboards/timeline_template.go` defines `timelineDefaultColumns`, `timelineDefaultBandLabel`, `timelineDefaultRowLabel`, `timelineDefaultReferenceFields`, and `TimelineTemplateVersion` as plain Go constants/vars. There is no `storyboard_templates` table and no template row a saved board could reference back into. `CreateTimelineBoard` (`timeline.go`) reads these constants once, inside the same transaction that creates the board, and never touches them again.

This was the deciding factor over a DB-row-backed template (also explicitly permitted by the spec): a Go constant is *structurally* incapable of being mutated by anything a saved board's own API surface can reach — there is no `UPDATE storyboard_templates` anywhere in this codebase, and no route exists that could add one by accident. "Instantiate template ≠ edit template" is true because there is nothing in the database an edit could target, not because of an access-control check that could later be loosened or bypassed.

## Instance metadata (`storyboards.mode`, `storyboards.template_version`)

Every board (Blank or Timeline) carries `mode TEXT NOT NULL DEFAULT 'blank'` and a nullable `template_version INTEGER`. For Timeline boards, `template_version` is set once at creation to `TimelineTemplateVersion` (currently `1`) — a historical fact ("this board was instantiated from Timeline template version 1"), never a live foreign key. Bumping `TimelineTemplateVersion` in a future kernel changes what the *next* "New Timeline" click produces; it has no effect on any board that already exists, because nothing ever re-reads the template after creation.

Blank boards leave `template_version` `NULL` — there is no "Blank template version" concept, since Blank has never had a seeded default structure beyond the single column/band/row every board needs at minimum (unchanged since Kernel 80).

## Independence, proven directly

`TestTwoNewTimelinesAreIndependent` (`kernel82_timeline_dbtest_test.go`) creates two Timelines, edits one's Reference Panel field, and asserts the other's corresponding field is untouched — then creates a *third* Timeline and asserts it starts with the same empty defaults as the first two, proving the template itself was never mutated by editing an earlier instance. This is the literal spec requirement from §14.1, not an inferred property.

## Extensibility for future personal templates (spec 1.3)

Kernel 82 deliberately does not build "Save as Template." The chosen architecture doesn't foreclose it either: a future "user template" would be another row somewhere (or another code-defined set, if templates stay code-only) that `CreateBoard`-family functions read from once at creation — the same shape `CreateTimelineBoard` already establishes. No schema decision here assumes Timeline is the only template that will ever exist; `storyboards.mode` is a `CHECK (mode IN ('blank', 'timeline'))` constraint specifically so a future mode requires an explicit, deliberate migration to add a third value, not an accidental one.
