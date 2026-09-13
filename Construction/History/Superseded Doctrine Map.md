# Superseded Doctrine Map — Do Not Resurrect

**Produced by:** Kernel 99, 2026-09-12. Each entry below was a real assumption or architecture Victory once used, later deliberately replaced. Every entry is verified against repository history, not assumed from current doctrine — a future agent reading old code, an old comment, or an old kernel spec may encounter one of these and should recognize it as historical, not as something to rebuild.

---

### Raw Cave/session as product center
**Retired by:** Kernel 22 (Showing model), fully displaced by Kernels 66/70A/71/92.
Early Victory (Kernels 1-21) treated the-cave and its session as the product's actual center. Show and Showing are now the canonical product center; a Session is a live connection, not a durable product entity. See `Kernel Lineage.md` Era 2-4 and Era 8's "Session is not the canonical Show owner" note.

### `current_session_personas` as canonical Show authority
**Retired by:** confirmed retired directly during Kernel 97 (2026-09-12).
`current_session_personas` and sitewide "active Character" tracking were confirmed, by direct code audit, to resolve *display* (which name/portrait to show) only — never participation authority. The canonical authority path is the Show Run roster (`show_run_roster_members`) plus location membership, resolved through `participation.ResolveParticipationContext`.

### Sitewide active Character as Show authority
**Retired by:** same Kernel 97 audit as above. Selected Character is canonical for *display*; roster membership is canonical for *participation*. Do not conflate the two.

### Access grants / memberships as primary Show authority
**Retired by:** Kernel 66 (Show Run roster introduced as the real authority source), confirmed as boundary by Kernel 97.
`access_grants` is a boolean venue/production *reach* grant carrying no role of its own — explicitly used as an additive upgrade layered on top of the real role/roster model, never as primary truth. The older `memberships` table is under-populated by any code path written after Kernel 66 (a Producer/Director bootstrapped post-Kernel-66 normally has no row there at all — closed as a real bug in Kernel 70A). Kernel 93 separately found and fixed a live case where `access_grants` could leak visibility that the correct, Showing-scoped `audience_admissions` table should have gated instead.

### Presence as durable participation
**Retired by:** confirmed non-canonical by both Kernel 97 and Kernel 98 (2026-09-12), independently.
Presence (who is currently connected via WebSocket) indicates connection state only. It has never been found gating any real authority decision in the current codebase.

### Session as canonical Show owner
**Retired by:** Kernel 70A (2026-07-16), which fixed a real Audience rehearsal-banner leak caused by treating session state as if it were Show state; fully reconciled by Kernel 97.
A Session is a live connection; a Show Run (via its roster) and a Location (via membership) are the actual owners of authority. Do not resolve "can this user do X in this Show" by inspecting session state directly.

### Global, unscoped Location role resolution
**Retired by:** Kernel 97 (2026-09-12) — `access.CurrentLocationRole` (which ignored `location_id` entirely and returned a user's single globally-best role across every Location they belonged to) was found live at 11 call sites, two gating real privileged actions, and was deleted outright rather than merely deprecated. Replaced by `access.CurrentDefaultLocationRole`/`access.CurrentLocationRoleForLocation`. **The old function no longer exists in the codebase at all** — do not reintroduce an unscoped location-role lookup.

### Operator's client-side role display overriding the server's real resolved role
**Retired by:** Kernel 97 (2026-09-12) — `frontend/lib/stage-runtime/session-sync.js` used to silently rewrite a server-resolved "audience" role to "producer" whenever the joining account was an Operator. Removed outright. The client must always display the server's real resolved role; if an Operator needs elevated tooling, that must come from a real, separate authority check, never a display-layer override.

### DOM-era stage math (where superseded by Pixi)
**Retired by:** Kernel 29 (PixiJS Stage Spike) through Kernel 30 (Cave Organization + Renderer Containment).
Early Cave rendering used direct DOM positioning math for stage objects. This was superseded venue-by-venue as Pixi adoption spread — check the specific venue's own history before assuming DOM-era math still applies anywhere; as of Kernel 94, Storyboards uses Vue+CSS Grid rather than either DOM-math or Pixi, a third, deliberately different rendering approach for a non-spatial venue.

### Universal Index Card abstraction as a one-size-fits-all primitive
**Status: partially superseded, not fully retired.** The index card (Kernel 12) remains a real, live primitive (note-card delivery, Workshop placement, First Theater card pinning). It has NOT been generalized into every content type Victory now has — Storyboards cards (Kernel 80+), Scene elements (Kernel 69+), and stage objects (Kernel 90) are distinct models with their own authority and persistence, not index-card variants. Do not assume "index card" is the universal content primitive across the whole product; check which specific model a given venue actually uses.

### Storyboards' pre-Vue plain-HTML/table rendering
**Retired by:** Kernel 81 (CSS Grid replacing the HTML table) and fully by Kernel 94 (Vue rebuild). Do not resurrect table-based Storyboards layout.

### Manually-placed `scene_stage_elements` as governing "current composition" for Update/Save-as-New-Scene
**Retired by:** Kernel 93 (2026-08-23), confirmed and documented during Kernel 98 (2026-09-12).
`UpdateCurrentScene`/`SaveArrangementAsNewScene` were changed by Kernel 93 to capture composition from the live venue's actual stage state (`venue_layout_elements`), not from placement-level `scene_stage_elements` rows. Two tests in `backend/internal/scenes/capture_test.go` still exercise the old model and currently fail — this is a stale-test problem (routed to K101), not evidence that the old model is still real. Do not "fix" those two Update/Save-as-New-Scene functions to match the old tests.
