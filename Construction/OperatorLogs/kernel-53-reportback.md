# Kernel Report Back - Kernel 53

## 1. Status
PARTIAL PASS

Kernel 53 now has the workbook-first character foundation in place, and the mandatory parentage threshold correction is implemented. The canonical root remains `character_cards`, Catharsis now writes append-forward workbook entries and workbook context, Greenroom renders workbook pages, and the Socio draft flow captures richer parentage metadata for later compiler work.

This is still not a completed kernel report because browser-level evidence has not been captured in this environment. The code paths and focused tests are in place, but the live browser smoke remains pending.

## 2. What Was Built

- Kept `character_cards` as the canonical root and extended it with workbook metadata.
- Kept `character_workbook_modules` and `character_workbook_entries` as the workbook runtime tables.
- Kept `character_journals` as editable private notes instead of canonical history.
- Rewired Catharsis to create or resume workbook drafts from workbook context.
- Recorded Stage 1 parentage as structured history entries instead of a single flattened result.
- Corrected donor retention eligibility to strictly `> 50`.
- Recorded coin-flip retention only for qualifying parents on the server side.
- Persisted richer Socio metadata into `workbook_context` so later compilers can surface:
  - parent rolls
  - chart version
  - inheritance metadata
  - current stage and event
  - starting wealth resolution
- Added workbook face-page summary fields so the Socio metadata is readable from Greenroom.
- Reworked Greenroom so the face page renders workbook summaries and the module page stays separate from the old card editor assumptions.
- Added a full-screen `Chapter II: Childhood Stages` interstitial after parentage resolves.
- Disabled Stage 2 mechanics in Kernel 53 so the flow stops at the Stage 2 shell.
- Wired the Catharsis tray control so a fresh workbook can actually start the onboarding flow instead of bouncing straight to Greenroom.

## 3. Evidence

Checks run:

- `cd backend && GOCACHE=/tmp/victory-gocache go test ./internal/characters ./cmd/victory`
- `node --check frontend/venues/catharsis/onboarding.js`
- `node --check frontend/venues/catharsis/runtime.js`
- `docker compose up -d --build backend`
- `docker exec victory-backend wget -qO- http://127.0.0.1:8081/health`
- `git diff --check`

Result summary:

- Backend character workbook tests passed.
- Backend command package tests passed.
- Catharsis onboarding and runtime JavaScript syntax checks passed.
- The backend container rebuilt successfully and came back healthy.
- The live backend health probe returned `ok:true`.
- The workspace diff is clean.

## 4. Verified Kernel Points

- Canonical workbook root: `character_cards`.
- Schema/bootstrap path: the workbook tables and `workbook_context` column are created in the runtime bootstrap path, so no new additive migration was required for this kernel slice.
- Create Character idempotence: Catharsis resumes the existing draft path through workbook context instead of creating a separate card path.
- First workbook ownership: the active workbook is surfaced in the Catharsis tray, and the tray button now starts onboarding when no workbook exists.
- Incomplete character load: the workbook view path loads draft workbook state, journals, and module state rather than a one-shot card dump.
- `/journal`: private, player-authored, module-scoped notes; it does not feed venue chat.
- Parentage chart v1.1: the chart validator enforces continuous coverage from 3 through 120 with no gaps.
- Parentage donor rolls: server-side `3d20` parentage still resolves through the canonical chart path, with the later coin-flip retention step now separated and corrected.
- Donor retention flip: unique per qualifying parent and resolved server-side.
- Retained Starting Credit: only qualifying parents above 50 can retain; 49 and 50 do not flip.
- Stage 1 summary / Parentage Face: workbook summary fields are written onto the face page for later rendering.
- Escape / refresh / resume: the workbook context is preserved in the draft path and the onboarding flow resumes from stored state when reopened.
- Stage 2 handoff: Chapter II opens as a shell-only transition to `Stages of Childhood → Conception` and does not execute childhood mechanics.

## 5. Still Pending

- Live browser smoke.
- In-browser verification of the surfaces below:
  - portrait upload using the existing Workshop constraints
  - initial sparse Face
  - initial Face widgets
  - active-character control near presence
  - character dashboard opening from that control
- A full visual confirmation that the new-workbook trigger appears and launches the Catharsis onboarding flow exactly where expected.

## 6. Operator Notes

- The canonical root remains `character_cards` for compatibility.
- The workbook-first surface is now the user-facing model.
- Catharsis now records metadata that later compilers can read without reconstructing parentage from a single flattened total.
- The starting-wealth logic is stored as metadata, not compiled into a finished rules engine yet.
- The Chapter II handoff is intentionally a full-screen transition card.
- Stage 2 is shell-only in Kernel 53.
- Greenroom now reads workbook state; it should not regress back to a one-card editor path.

## 7. Files Changed / Created

- `backend/internal/characters/parentage_chart_test.go`
- `backend/internal/characters/socio_build.go`
- `backend/internal/characters/socio_build_test.go`
- `backend/internal/characters/workbook_pages.go`
- `backend/internal/characters/workbook_pages_test.go`
- `frontend/venues/catharsis/index.html`
- `frontend/venues/catharsis/onboarding.js`
- `frontend/venues/catharsis/runtime.js`
- `frontend/venues/greenroom/index.html`
- `Construction/OperatorLogs/kernel-53-reportback.md`
