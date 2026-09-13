# Kernel 90 Report Back — Canonical Stage Object State, Visibility & Cue Control

**Kernel spec:** `Kernel_90_Canonical_Stage_Object_State_Visibility_and_Cue_Control.md`
**Status:** PASS, with one evidence gap honestly stated below (browser rendering)
**Date:** 2026-08-17
**Deployed:** not yet — uncommitted, awaiting Grant's review (the pattern since Kernel 86)
**Operator guide:** `Construction/Domains/Operations/stage-object-visibility.md`

---

## 1. What this kernel is

Before today, "who can currently perceive or interact with this thing on
stage?" had three unrelated answers in Victory:

1. `act/reveal_element` / `act/hide_element` — a per-session action-log replay
   in `world/snapshot.go`, deriving an actor/audience layer binary for live
   warehouse tokens only.
2. Four Cue actions (`reveal_object`, `hide_object`, `enable_interaction`,
   `disable_interaction`) — deferred at Kernel 70 for want of a canonical
   object identity to target.
3. `participant_interactions.enabled` — a global authoring boolean, unrelated
   to either of the above.

Kernel 90 replaces all three with one canonical model:

- **One object identity** (`stageobjects.Ref{kind, id}`) across the four
  durable object kinds Victory actually has: live warehouse elements
  (`venue_layout_element`), Scene-authored composition elements
  (`scene_stage_element`), Kernel 87 drawings (`drawing_object`), and bound
  participant interactions (`participant_interaction`).
- **One state store** — `stage_object_states` (visibility + interaction, one
  row per non-default object) and `stage_object_scope_grants` (the exception
  list that reveals a hidden object to named audiences), migration 106.
- **One mutation path** — `stageobjects.ApplyMutation`. Manual Director
  controls and the four now-real Cue actions both call it. §24's "manual and
  Cue must produce identical state" is not a tested coincidence; it is the
  same function call.
- **One projector** — `stageobjects.Projector`, which every viewer's snapshot
  (live venue, drawing list) is filtered through before anything is sent over
  the wire. A hidden object is **omitted**, never shipped with a flag.

The old mechanism (1) is deleted, not kept alongside the new one — the layer
replay in `world/snapshot.go` (`deriveElementLayerVisibility`,
`effectiveAudienceVisible`, `elementVisibilityForLayer`) is gone. Mechanism
(3) keeps its distinct meaning as a global kill-switch, bridged to (not
merged with) the new Show-scoped dimension. `act/show_overlay` /
`act/hide_overlay` were left alone — genuine Cave-era presentation, not a
durable object concern, per §19's own instruction.

---

## 2. Deviations Grant explicitly approved

**No backfill.** Anything hidden under the old per-session mechanism before
this migration reads as visible after it. The old mechanism never recorded
*who* something was hidden from — only a bare actor/audience binary — so
there was nothing honest to convert into a scoped grant. Grant's instruction,
2026-08-17: "don't worry about backfilling anything… I'll add it again under
the new schema."

**Legacy code deleted rather than preserved.** The layer-replay functions in
`world/snapshot.go` are gone outright, per the same instruction ("delete the
old stuff if it doesn't fit/breaks something").

---

## 3. What was built

**Database** — `backend/migrations/106_kernel90_stage_object_state.sql`:
`stage_object_states` (CHECK-constrained `object_kind`/`visibility`,
show-scoped, `NULL` `interaction_enabled` for non-interactable kinds) and
`stage_object_scope_grants` (four scope kinds: `cast`, `audience`, `cohort`,
`character`; `NULLS NOT DISTINCT` unique constraint so a duplicate tier grant
collides rather than silently duplicating).

**Backend** — `backend/internal/stageobjects/` (new package, deliberately a
leaf: it takes its Director-authority gate and its WS notifier by
*injection*, because `world` imports it for projection while
`shows → network → world`, so importing `showruns` or `network` here would
close a real cycle):

- `types.go` — `Ref`, `State`, `Scope`, the four object kinds, the two
  visibility states, the four scope kinds.
- `resolve.go` — `ResolveRef`, one resolver per object kind. This is where
  §3's map boundary is enforced: `resolveSceneStageElement` refuses
  `map_backdrop`/`grid_config` by stored kind.
- `store.go` — `ApplyMutation` (the single write path), `LoadShowStates`.
- `projection.go` — `Projector`, `ResolveViewer`. Cast is resolved from Show
  Run roster participation, not from the role string handed in — see §7's
  found-not-fixed note below for why that matters.
- `audit.go` — best-effort `actions` row per mutation (evidence, not a
  precondition: a Director preparing between sessions with no live session
  can still set state), plus `InteractionInvocable`, the invoke-time guard.
- `http.go` — `GET/POST /api/shows/{id}/stage-object-states`,
  `GET /api/shows/{id}/stage-object-scope-targets`. Director+ only, including
  the read.

**Backend, existing packages touched:**

- `cues/types.go`, `cues/cues.go`, `cues/execute.go` — the four deferred
  actions are real; `executeStageObjectState` calls
  `stageobjects.ApplyMutation` directly.
- `world/snapshot.go` — canonical projection replaces the legacy layer
  replay; `canonicalRefForElement` maps a snapshot element onto its
  canonical ref (reusing the pre-existing `scene:` id prefix as the kind
  discriminator); `applyInteractionState` bridges global and Show-scoped
  interaction flags.
- `drawing/objects.go`, `drawing/types.go` — drawing list projection filtered
  through the same `Projector`; `Object.HiddenBackstageOnly`.
- `merchant/interactions.go` — `ResolveEligibleContext` now calls
  `InteractionInvocable`, so a Director-disabled interaction is refused at
  invoke time, not merely un-offered.
- `network/ws.go` + new `network/kernel90_stage_object_state.go` — the legacy
  `act/reveal_element`/`act/hide_element` WS control now writes canonical
  state too (`applyLegacyRevealToCanonicalState`), so the Cave-era control and
  the new Director menu are two doors into one room rather than two competing
  mechanisms.

**Frontend:**

- `kernel90-visibility-tools.js` (new) — the four canonical operations plus
  the scope panel. No optimistic local state: a mutation is followed by a
  server-broadcast stage invalidation and a fresh snapshot, because a hidden
  object is omitted from payloads rather than flagged.
- `logic.js` — `canonicalStageObjectRef`, `boundInteractionRef`,
  `stageObjectHidden`; the flat "Hide Audience"/"Show Audience" toggle is
  replaced by nested `Visibility`/`Interaction` families on **every** durable
  object branch, including Scene-authored composition elements, which had no
  visibility control of any kind before this kernel.
- `action-router.js` — routes the four `k90-*` menu actions.
- `scene-nodes.js` — `markHiddenForDirector`: dimming + dashed outline for
  objects the Director keeps on their working stage while ordinary viewers
  do not receive them (§14/§45).
- `runtime.js` — `VictoryStageKernel88Bridge.setStageStatus` exposed so the
  tool module reports through the engine's own status line.

**Fixtures/scripts:** `backend/cmd/k90fixture/` (two Cohorts, three Players,
one Audience viewer, a dedicated Scene with a token, a map_backdrop, a bound
interaction, and a drawing object); `scripts/smoke/kernel90-run.sh`,
`scripts/smoke/kernel90-stage-object-visibility.js` (acceptance),
`scripts/smoke/kernel90-visibility-browser.js` (UI).

---

## 4. Evidence

### Backend tests

```
go build ./... && go vet ./... && go test ./...     # green, full suite
```

New: 18 pure unit tests (`internal/stageobjects/projection_test.go`, no
database — semantic claims like "hidden ≠ deleted", "Cohort A cannot infer
Cohort B's grants", "an Ungrouped viewer must not match a cohort grant via
empty-string collision"); 20 dbtests
(`internal/stageobjects/stageobjects_dbtest_test.go` — storage, persistence,
scope validation against real Cohorts/Characters, the map refused against a
*real* `map_backdrop` row, drawing objects, manual/Cue row-identity parity);
6 Cue dbtests (`internal/cues/kernel90_stage_object_dbtest_test.go` — real
Cues fired through the real `ExecuteCue` GO-press path, including a target
deleted between authoring and firing, and a cross-Show target refusal); 3
world dbtests (`internal/world/kernel90_stage_object_projection_dbtest_test.go`
— the §36 payload-omission proof against a real `LoadVenueSnapshot` call, the
scope-metadata-withholding proof, and an explicit regression guard proving the
deleted legacy replay stays deleted).

### Frontend tests

```
node --test tests/    # 169 pass / 9 pre-existing dice.test.js failures, unchanged
```

New: 12 tests in `tests/stage-runtime/kernel90-visibility-tools.test.js` —
identity is read from server-sent state and never synthesized from a label or
key; a Player gets no visibility/interaction controls even holding forged
identity; the family is nested, not flat; the legacy toggle is gone; Interaction
appears only when there is something to interact with; state ticks reflect
current state, not the offered verb.

### Acceptance proof — real HTTP, five real viewers, zero mocking

```
bash scripts/smoke/kernel90-run.sh
```

**52/52 checks pass.** Director, two Cohort-assigned Players, one Ungrouped
Player, and a genuine Audience-tier viewer, each reading their own
`/api/world/catharsis` projection from a real compiled backend. Covers: default
visibility, hide/reveal, all five scope kinds (Director-only, Cohort, Character,
Audience, Cast) including two grants on one object at once, disabled-not-hidden
and hidden-not-disabled independence with the invoke-time refusal proven (a
Player calling a disabled interaction anyway gets `403 interaction_disabled`),
persistence across repeated fresh reads, the map refused (`422
stage_object_not_supported`), drawing objects, and the full `§35` authority
matrix (nine distinct refusals, each on the actual boundary named).

### Browser proof — partial, environment-blocked

```
bash scripts/smoke/kernel90-run.sh --ui
```

**Honest gap, not a product defect.** This host's headless Chromium cannot
sustain a live PixiJS WebGL render: every WebGL-capable launch configuration
tried (software swiftshader via `--use-gl`, via ANGLE, via
`LIBGL_ALWAYS_SOFTWARE`, the full Chromium binary instead of
`chromium_headless_shell`) crashed the GPU/renderer process, confirmed via
`dmesg` as a real trap (`int3`), not an OOM. Disabling WebGL avoids the crash
but leaves Pixi unable to initialize at all in this bundle (`Unable to
auto-detect a suitable renderer` — no canvas-renderer fallback is bundled), so
no canvas ever exists to test against.

What the browser proof **did** prove, over real HTTP through the Kernel 87 dev
proxy against the real backend, before hitting the rendering wall: the scope
panel opens and lists real Cohorts and real Characters from the actual
database (`Construction/OperatorLogs/evidence/kernel-90/k90-08-scope-panel.png`
— note the onboarding overlay still visible behind it in that capture, a
cosmetic ordering issue in the harness, not a functional one), and a real Cue
fired end-to-end through `/api/cues/{id}/go` in an earlier partial run before
the environment gave out.

The claims that specifically need a rendered canvas — real right-click opening
the grouped context menu, and the dimmed/dashed backstage treatment on a
hidden object — are **not independently browser-proven** in this
environment. Everything else those claims depend on IS proven: the menu shape
by the frontend unit tests reading real server-shaped data (12/12 pass), and
the dimming/outline code itself
(`scene-nodes.js`'s `markHiddenForDirector`) is straightforward Pixi drawing
gated on the exact `hidden_backstage_only` flag the acceptance proof already
confirms the server sends correctly to backstage viewers only. The script is
left in place, correctly configured for a healthy host, with the failure mode
documented in its own header comment so a future run either succeeds cleanly
or fails with an immediately diagnosable message rather than a mystery hang.

---

## 5. Known issues carried forward

**Found, not fixed — a real pre-existing role-resolution bug.**
`cmd/victory`'s `lookupVenueRole` calls
`participation.ResolveParticipationContext` with no Show Run hint, so that
resolver's roster step never runs and every Show Run roster Player resolves to
`"audience"` on the live venue snapshot path. `stageobjects.ResolveViewer`
works around this correctly by deriving Cast from Show Run roster
participation directly rather than trusting the role string — proven in
`TestResolveViewerDerivesCastFromShowParticipationNotTheRoleString` — but
anything else in the product trusting that role string on this path is still
getting the wrong answer. Recorded here rather than fixed, because the shared
resolver is well outside this kernel's object-identity scope.

**The Audience genuinely cannot open Catharsis.** `access.ResolveVisibleVenues`
gates the venue on a `producer/director/cast/crew` membership role; a plain
`audience` role membership cannot pass that gate at all. The acceptance proof's
Audience viewer is real and its scoping is genuinely proven — it reaches the
venue by holding a `cast` location membership with no Show Run roster row,
which is what the participation resolver's fallback step correctly reads as
Audience. Whether Victory should let a pure Audience account through the venue
gate is a separate product decision, the same shape as Kernel 89's First
Theater finding.

**Not pursued, per the kernel's own §50:** Kernel 89's announcement
Showing-Review persistence question, First Theater participant access,
per-merchant Haggle mechanics.

---

## 6. Next recommended step

Grant's own walkthrough of `Construction/Domains/Operations/stage-object-visibility.md`
§9, on a real install (which will also settle whether this host's WebGL
limitation is specific to this container or reproducible on the deployed
box — worth checking before assuming the browser proof will pass there
unmodified).
