# Kernel 22 - Showing Model

## Expected Purpose
Introduce the showing model that links session activity to a reviewable production-run record.

## Likely Delivered Behavior
- `showings` table and bootstrap exist
- sessions are associated with showings
- action authority checks can consult showing state
- showing/session linkage is available to snapshot and action code

## Evidence Source
- [kernel-22-showing-model-v1.md](/opt/victory/Construction/Kernels/kernel-22-showing-model-v1.md)
- [showings.go](/opt/victory/backend/internal/showings/showings.go)
- [main.go](/opt/victory/backend/cmd/victory/main.go)

## TODO
- A dedicated reportback artifact is not clearly separated yet.
- Showing Review UI still does not exist.
