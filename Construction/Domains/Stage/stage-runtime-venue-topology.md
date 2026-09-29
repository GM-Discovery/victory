# Stage Runtime Venue Topology

Canon for which venues share stage code and which do not. Written 2026-09-29 after the question came up directly during Kernel 101 (101-30): *"the-cave doesn't run over the same runtime as first-theater anymore, correct?"* The answer is yes on the frontend and no on the backend, and that asymmetry is the whole point of this document.

Read this before assuming a change is venue-scoped. **The venue's name tells you nothing about your blast radius.** Only the directory the code lives in does.

## 1. The one-sentence version

**One backend serves all three stage venues; the frontend is split two ways.** `frontend/lib/stage-runtime/` is Catharsis and First Theater only. the-cave shares none of it, by design (§4).

## 2. Frontend: three implementations, two families

| Venue | Pixi stage | `/lib/stage-runtime/` modules | Stage logic lives in |
|---|---|---|---|
| Catharsis | yes | 28 | shared runtime |
| First Theater | yes | 24 | shared runtime |
| the-cave | **no** | **0** | inline, in its own `index.html` |

**Catharsis is a strict superset of First Theater.** It loads every module First Theater does, plus four: `drawing.js`, `kernel85-cohort-tools.js`, `kernel88-socio-player-hud.js`, `kernel93-audience-overlay.js`. First Theater loads nothing Catharsis doesn't. A consequence worth knowing: a `stage-runtime` change can only reach First Theater through a code path Catharsis also has, so Catharsis is the broader test surface of the two — but *not* a sufficient one, because the four extra modules can mask a break in a shared module's no-drawing/no-cohort path.

**the-cave loads no `stage-runtime` module, does not load `pixi.min.js`, and defines no `VictoryStage*` global.** Its stage behavior is inline in a ~5,900-line `index.html`. The only `/lib/` code it shares with the other two is generic chrome: command palette, mic chat, Discord audio, back-to-map, account badge. It does not load `venues/shared/venue-shell.js` either.

**There is a fourth consumer.** `frontend/venues/directors-chair/index.html` loads exactly one stage-runtime module standalone — `kernel92-showtime-panel.js`, which no stage venue loads at all. So "stage-runtime is Catharsis and First Theater" is true of the runtime engine but not of every file in the folder. Check `grep -rl` before assuming a module has only two callers.

`runtime.js` itself is loaded dynamically (`script.src = "/lib/stage-runtime/runtime.js?v=..."`) rather than via a `<script src>` tag, so a grep for `src="` will miss it in both venues. It lives near the bottom of each `index.html`.

## 3. Backend: one implementation, parameterized by slug

The route table reads as if there were two backends. There are not:

```go
// cmd/victory/main.go:1390-1392
mux.HandleFunc("/ws/the-cave",      network.ServeCaveWS(hub, pool, cfg))
mux.HandleFunc("/ws/catharsis",     network.ServeVenueWS(hub, pool, cfg, "catharsis"))
mux.HandleFunc("/ws/first-theater", network.ServeVenueWS(hub, pool, cfg, "first-theater"))
```

`ServeCaveWS` is a two-line alias (`internal/network/ws.go:249`):

```go
func ServeCaveWS(hub *Hub, pool *pgxpool.Pool, cfg identity.DiscordServerLinkConfig) http.HandlerFunc {
	return ServeVenueWS(hub, pool, cfg, "the-cave")
}
```

`world.LoadCaveSnapshot` is the same shape (`internal/world/snapshot.go:218`) — an alias for `LoadVenueSnapshot(..., "the-cave")`. The three `/api/world/*` handlers in `main.go` (lines 936, 1010, 1090) are near-identical copy-pasted blocks differing only in the slug string they pass to `access.UserCanAccessVenueSlug` and to the projector.

So all three venues share: **one WebSocket handler, one action authority gate (`actions.CanAct`), one world projector, one snapshot shape.** The Cave-named functions are historical skins from when the Cave was the only venue. Do not read them as evidence of a separate Cave backend; there has not been one for a long time.

### Practical consequence

- A change under `backend/internal/` reaches **all three** venues, whatever its name suggests.
- A change under `frontend/lib/stage-runtime/` reaches **Catharsis and First Theater** (plus `directors-chair` for the one module noted above).
- A change inside a venue's own `index.html` reaches **that venue only**.

These are three different blast radii and they do not line up with the venue list.

## 4. the-cave is deliberately not getting the stage runtime

**This is a design decision, not a migration that stalled.** the-cave is about raw controls, primitives and elements — not right-click context menus and the layered tool complexity the Catharsis/First Theater runtime exists to provide. It is not a candidate for "unify it with the shared runtime" work, and a future kernel proposing that should treat this section as a standing objection to be argued against explicitly rather than an oversight to be tidied up.

Concretely, the-cave sends no token actions at all. Its full action vocabulary:

```
create/index_card   update/index_card   delete/index_card
act/place_element   act/duplicate_element
act/reveal_element  act/hide_element
act/show_overlay    act/hide_overlay
act/set_element_lock
act/set_nameplate_visibility
```

No `create/token`, no `update/token`. Index cards and element placement/visibility, nothing more.

### The Token Workshop is not a token picker

the-cave's `refreshTokenWorkshopList` fetches `/api/warehouse/assets?asset_type=token&status=active` (`index.html:~4159`). This is an **asset authoring** surface — uploading and reviewing token assets in the Warehouse — not a stage placement picker. Its producer/director-only gate via `requireWarehouseAccess` is correct and should stay. Do not "fix" it to match the Catharsis token picker's Cast+ access; they are not the same feature. (This exact misreading was made and corrected during Kernel 101 101-30.)

### A live asymmetry to be aware of

Because the backend is shared, Cast in the-cave **already holds** `canActCreateToken` permission from Kernel 101 (101-24). There is simply no frontend in the-cave that would ever send `create/token`. The permission is real and the surface is absent. That is a stable, intended state given §4, not debt — but anyone reasoning about "who can create tokens" from the backend alone will over-count the-cave, and anyone reasoning from the-cave's UI alone will under-count it.

## 5. Where the stage token picker's authority actually lives

Recorded here because it spans all three layers and is easy to half-learn (Kernel 101 101-30):

- **Menu entry** — `logic.js`'s `resolveStageObjectActions`, gated on `canCreateStageObjects` (producer/director/operator/cast) for Add Token, and on `canManageStageTokens` (producer/director/operator) for every edit action.
- **Picker open + asset load** — `token-ui.js`'s `canUseTokenPickerMode(mode)`: create consults `canCreateStageObjects`, replace consults `canManageStageTokens`. Both gates ask the same question so they cannot drift.
- **Asset listing** — `GET /api/stage/token-assets` (`internal/assets/stage_tokens.go`), Cast+ but not Crew. Deliberately separate from `/api/warehouse/assets`, which stays producer/director-only.
- **Asset bytes** — `/api/assets/{id}/content`; `userCanReadAsset` already admits `cast` location members, so this needed no change.
- **The actual create** — `actions.CanAct` → `canActCreateToken`, reached with the asset id passed as `ElementID` (`internal/actions/token.go:90`). This is the only one of the five that is authoritative. The other four are UI truthfulness.
