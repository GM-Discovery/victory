# Venue Index

Why each venue exists, who uses it, and what related venue handles adjacent work. Not a control-by-control manual — the product teaches individual controls contextually.

**The Cave** — the original proving-ground stage: bare, role-filtered live-play surface. Still used as the reference implementation for new stage mechanics before they reach Catharsis/First Theater. Used by: whoever's testing a mechanic; not a normal Audience-facing venue.

**Catharsis** and **First Theater** — the two parallel live-performance venues where Shows are actually staged and Showings actually happen. Both share the same underlying stage runtime; they exist as separate venues rather than one, to keep separate productions' live state cleanly apart. Used by: Cast, Crew, Director, Audience — everyone, during a live session.

**Victory Theater** — a further live-performance venue alongside Catharsis/First Theater for the same purpose (staging Showings); consult current-state.md if you need the specific distinction that led to it existing separately.

**Storyboards** — the spatial creative worktable for structuring a Show's cards, columns, rows, and bands *outside* live play. This is where planning and authoring happen before something goes on stage. Used by: Producer, Director, Crew (task-scoped), Cast (where a production opens it up for collaborative planning).

**Library** — home for a Show's Scene Library and reference/eWrite material. Adjacent to Storyboards (planning) but focused on reusable saved content rather than active structuring.

**Writer's Room** — eWrite authoring surface: Rulesets, manuscripts, Skill Directories. Adjacent to Library (where finished eWrite content is read) but is where it's written and revised.

**Producer's Office** — administrative home for Producers: scheduling, roster, Showings, production-level configuration. Adjacent to Stage Management (day-to-day roster/session operations) and The Director's Chair (live-session prep).

**The Director's Chair** — Director's live-session preparation space: getting a Scene and cue plan ready before a Showtime. Adjacent to Producer's Office (administrative) and the live venues themselves (execution).

**Stage Management** *(`/venues/show-runs/`)* — operational home for Show Run administration day-to-day. Adjacent to Producer's Office; this is the working surface, the Office is the planning one.

**Greenroom** — Character home: creation, dressing, and management of the Characters people play. Used by anyone with a Character, most centrally Cast.

**Trailers** — profile/identity surface distinct from in-Show Characters — your own account-level presentation. Adjacent to Greenroom but answers a different question ("who are you," not "who are you playing").

**Third Place** — the informal social/commons space for the Location, independent of any specific Show. Used by anyone on the lot.

**Audition Hall** — where people are considered for roles/roster placement on a Show. Adjacent to Producer's Office (which actually manages the resulting roster).

**Workshop** — asset/placement authoring surface (the "workshop asset floor"). Adjacent to Storyboards/Library for where authored assets end up being used.

**Warehouse** — asset storage: uploaded files, their metadata, and permissions, distinct from Workshop's authoring focus.

**Grant's Cabin** — the Operator's own venue: infrastructure-facing tools and the Operator's contact point, distinct from any Producer/Director tooling. Not a venue ordinary Cast/Audience members have reason to visit.

**Middle School Stage** and **Stage Template** — an early stage-layout exploration and a reusable internal template for building new stage-type venues, respectively. Neither is a normal end-user destination; `stage-template` in particular is scaffolding for future venues, not a product surface of its own.

**Construction Site** — internal-facing venue tied to Victory's own build process, not part of the consumer product surface.
