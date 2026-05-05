# Kernel 18 Context Menu + Move Tool v1

## What is enforced

- Right-clicking a placed stage element opens a cursor-anchored context menu.
- The menu is role-aware.
- Audience has no stage context menu.
- Move, reveal, hide, remove, info, and create-card actions all flow through the existing server-authoritative action spine.

## Scope

- Stage elements only.
- Fire and index cards use the same context menu path where relevant.
- Move v1 is prompt-based and persists through the existing `act/place_element` action.
- Remove from Stage is implemented as a stage-to-tray placement.
- Info is a small popup with basic metadata/content.
- Create Index Card reuses the existing Cave card editor.

## Role rules

- Audience: no stage context menu
- Cast: permitted commands only
- Director: broad controls
- Producer: full suite

## Backend surfaces

- reveal/hide target rules: `backend/internal/actions/reveal.go`
- stage placement: `backend/internal/actions/place.go`
- authority checks: `backend/internal/actions/authority.go`

## Frontend surfaces

- Cave stage and context menu: `frontend/venues/the-cave/index.html`

## Operator notes

- No new dependencies were added.
- No new environment variables were added.
- Backend restart is required after code changes.
- Move is currently prompt-based, not drag/drop.
