# Victory Product Glossary

Canonical current meaning of Victory's core vocabulary. Where an older document uses a term differently, this glossary — not that document — is current truth (see `Construction/Domains/Identity/Canonical Role and Authority Resolution.md` for the authority-resolution detail behind the role terms, and `Construction/History/Architecture Milestones.md` for when a term's meaning changed).

**Location (a.k.a. "the lot")** — one Victory installation's top-level container: a campus of venues, its own set of people, and its own Operator. Most installs have exactly one. Location membership governs lot-wide access (who's on the campus at all), not what any individual can do inside a specific Show — see Show Run roster below.

**Venue** — a distinct place/room inside Victory with its own purpose (Catharsis, Storyboards, Greenroom, etc. — see `Docs/Product/Venue Index.md`). Being able to open a venue ("venue access") is a separate question from having authority to do anything privileged inside it.

**Show** — the enduring production: a title, a roster, a body of Scenes and Storyboards, existing whether or not anyone is currently playing it. A Show is not tied to any single session or browser tab.

**Show Run** — the internal/database identity a Show's roster and authority are actually scoped to (`show_run_roster_members`, etc.). In ordinary product language this is just "the Show" — the distinction only matters when reading resolver code.

**Showing** — one scheduled/ticketed audience-facing performance of a Show. A Show can have many Showings. Audience admission is scoped to a specific Showing, not to the Show as a whole.

**Showtime** — the live-performance mode/state a Show enters when a Showing is actually running.

**Scene** — a saved, reusable unit of stage composition (tokens, layout, cues) that belongs to a Show's Scene Library and can be placed into a live session.

**Character** — the persistent identity a person plays inside a Show. **Selected Character is canonical for Show participation** — which Character you have selected determines what you can do and see as Cast, not your account or your currently-active persona elsewhere.

**Cohort** — a contextual grouping used to scope what an Audience member can see (e.g. splitting an audience into small groups with different projected content). Cohorts are grouping, not role or authority — a backstage-visibility role (Director/Producer/Operator/Crew) always bypasses cohort scoping and perceives everything.

**Audience** — the default/spectator role. Governed by admission to a specific Showing (`audienceadmission`, keyed by Showing + the admitted user), not by location membership or Show-Run roster.

**Cast** *(a.k.a. Player)* — a person rostered onto a specific Show Run with an active-participation role, resolved via `show_run_roster_members`. Distinct from Audience: a Cast member actively plays a Character in the Show rather than watching it.

**Crew** — a bounded, contract-based backstage role. Crew can see full backstage state (same visibility tier as Director/Producer/Operator) but cannot manage the Show — only perform a short, explicitly named list of non-destructive actions (things like merchant dialogue authoring, equipment, cue triggering, scene composition edits). Crew is always scoped to a specific task, never a general management role.

**Director** — has live production-management authority for a Show Run (`CanManageShowRun`), plus cue-triggering authority. Treated as management-equivalent to Producer for most purposes; Directors are the ones actually running a live session moment-to-moment.

**Producer** — has the same management authority as Director (`CanManageShowRun`) at the Location level, for the business/administrative side of running productions: scheduling, roster, Showings. A person can be both Director and Producer.

**Operator** — the installation's own infrastructure authority (identified by `OPERATOR_USER_ID`/`OPERATOR_HANDLE`, not a role anyone assigns in-product). Full authority everywhere in Victory by construction. Distinct from Producer: Operator is "who runs the install," Producer is "who runs a production inside it." See the Operator role guide for the important privacy disclosure that comes with this.

**Stage object** — any individual placeable/interactive element on a Scene's composition (a token, a drawing, an index card, etc.), each with its own per-object visibility state.

**Visibility** — an explicit, per-object projection state (who can currently perceive a given stage object), not an implicit consequence of role alone. Resolved through the same canonical projection path for every viewer, including the narrow "Preview as Player" tool.

**Storyboards** — the spatial creative worktable for structuring a Show's cards, columns, rows, and bands outside live play — planning and authoring space, not the live stage itself.

**Timeline** — a specific, code-defined Storyboards mode/template (immutable structure with protected boundary columns) for laying out a Show's events in sequence, distinct from Storyboards' general free-form board mode.

**eWrite** — Victory's Markdown-based document/rules-authoring system (manuscripts, Rulesets, Skill Directories) used for Socio and other rules-native content, with its own visibility and revision model.

**Guide / chat pod** — Victory's contextual in-product help and communication surface; "Guide" refers to the help/onboarding content itself, appearing where relevant rather than as one giant manual.
