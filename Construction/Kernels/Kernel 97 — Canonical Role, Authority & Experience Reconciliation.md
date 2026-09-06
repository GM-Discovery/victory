# Kernel 97 — Canonical Role, Authority & Experience Reconciliation

**Status:** DRAFT — ready for implementation  
**Type:** Role-resolution audit + authority reconciliation + experience consistency  
**Sequence position:** After Kernel 96  
**Primary proof:** Every Victory role resolves consistently across venues, Shows, Showings, stage state, and shared services, with no contradictory authority paths  
**Core doctrine:** One person may occupy many contexts, but Victory should have one canonical answer for what that person may see and do in each context.

---

## 0. Kernel mode

Kernel 97 is not a redesign of roles.

It is not a new permission system.

It is not another dress rehearsal.

It is a reconciliation kernel.

The goal is to eliminate contradictory role/authority behavior that accumulated across many kernels and venue-specific implementations.

Work from the repository outward:

1. inventory every role/authority resolver;
2. identify duplicate and conflicting truth;
3. define the canonical source for each authority question;
4. migrate callers toward the canonical source;
5. preserve intended role behavior;
6. prove cross-role consistency mechanically;
7. perform only a small human experience spot-check.

Do not ask Grant questions the repository can answer.

Do not broaden capabilities merely because a resolver is being cleaned up.

---

## 1. Canonical role set

Victory’s current human-facing role hierarchy remains:

- Audience
- Cast / Player
- Crew
- Director
- Producer
- Operator / Owner

Do not invent new top-level roles in K97.

If repository terminology includes synonyms such as Player/Cast or Owner/Operator, reconcile their meaning without forcing unnecessary renames through every historical table.

The product-facing vocabulary should remain coherent.

---

## 2. Context matters

Authority may depend on:

- account;
- Location / lot;
- venue;
- Show;
- Showing;
- Show Run;
- Scene;
- Character selection;
- cohort;
- explicit grant;
- participation state;
- production role;
- operator status.

K97 must distinguish context-specific authority from global identity.

Do not collapse every role into one sitewide rank.

A user may be Director in one Show and Audience in another.

A Producer may have broad production authority without being the canonical Director of every live action.

An Operator is infrastructure/product authority, not a reason to bypass every product model invisibly.

---

## 3. Selected Character is canonical for Show participation

Preserve the established rule:

> **The selected Character on the Show roster is canonical for Show participation.**

Do not reintroduce sitewide active Character as the authority source.

Do not use presence as durable participation truth.

Do not restore `current_session_personas` as canonical Show authority.

Audit any remaining call sites that still do.

---

## 4. Show / Showing / Scene boundaries

K97 must make role resolution coherent across:

- Show;
- Showing;
- Showtime/live wrapper;
- Scene;
- venue access;
- Audience admission;
- live stage projection.

A role resolved for Show A must not silently grant authority in Show B.

A Showing-specific Audience admission must not be treated as global Audience membership.

A user’s venue access must not silently imply production authority.

---

## 5. Known carried findings

Explicitly investigate previously observed authority inconsistencies, including:

- roster Player being misclassified as Audience when the participation resolver lacks the correct Show Run / Showing hint;
- pure Audience venue-entry inconsistencies;
- duplicated Director authority checks;
- venue-role lookup versus production-role lookup disagreement;
- Producer/Director/Operator fallback behavior;
- backstage-role cohort overrides;
- characterless users falling into unintended participation paths;
- any legacy access-grant/membership logic still acting as primary authority where newer production models should control.

These are known leads, not the complete audit.

---

## 6. Authority questions must have named answers

For every major action, Victory should be able to answer:

> “Why is this user allowed?”

and

> “Why is this user denied?”

through a canonical resolver path.

Avoid ad hoc condition chains scattered through handlers.

Prefer small explicit domain functions such as:

- can view this venue;
- can enter this Showing;
- can edit this Scene;
- can control this stage object;
- can manage this Show;
- can configure Audience projection;
- can alter this Character;
- can manage this publication;
- can see this private relationship/message.

Do not build a generic policy DSL.

---

## 7. Resolve duplicate authority logic

Search for:

- direct role-string comparisons;
- repeated `producer || director || operator` checks;
- venue membership checks;
- location membership checks;
- Show role checks;
- Show Run participant checks;
- audience admission checks;
- operator-handle checks;
- user-ID special cases;
- client-side-only control gating.

Where duplicate logic answers the same authority question, move callers toward one canonical domain function.

Do not create abstraction for unrelated checks merely because both use roles.

---

## 8. Operator authority

Operator remains the highest product/installation authority.

But K97 should make Operator behavior explicit.

Required:

- Operator identity comes from canonical installation authority established after K96;
- no Grant-specific handle dependency;
- Operator can recover/manage the installation;
- Operator privilege is server-authoritative;
- Operator does not accidentally distort ordinary role resolution for test users;
- ordinary role previews should still be truthful when Operator uses “view as” or equivalent tools.

Do not create hidden “if Grant then allow” paths.

---

## 9. Producer authority

Producer is production-level authority.

Audit:

- Show creation/configuration;
- production administration;
- Showing scheduling;
- Producer Office;
- Show roster management;
- Director assignment;
- appropriate backstage access.

Producer should not automatically become the active Director in live interaction unless the product explicitly intends that fallback.

Where Producer fallback exists, document and reconcile it.

---

## 10. Director authority

Director is live/show creative authority.

Audit:

- Scene configuration;
- stage objects;
- Audience projection controls;
- cues;
- prepared play;
- announcements where currently supported;
- drawing/cartography authority;
- live show controls;
- Storyboards/editor surfaces where applicable;
- visibility state.

Do not use K97 to add missing Director capabilities to venues.

Only reconcile who is allowed to use capabilities that already exist.

---

## 11. Crew authority

Crew should remain meaningful but bounded.

Audit:

- content creation;
- card/band creation;
- backstage participation;
- venue tools;
- show preparation;
- assets;
- Storyboards;
- drawing tools if current rules permit;
- stage mutation where explicitly granted.

Crew must not inherit Director authority through broad “backstage” checks.

---

## 12. Cast / Player authority

Audit:

- selected Character;
- Show participation;
- cohort membership;
- current turn/group leader affordances;
- player-initiated rolls;
- Character-owned state;
- chat/reactions;
- scene visibility;
- interactions;
- stage object manipulation where allowed;
- personal notes/workbook.

Cast authority should follow the canonical Show roster / selected Character model.

A Cast member must not become Audience merely because a legacy resolver fails to find a session hint.

---

## 13. Audience authority

Audience is intentionally minimal.

Audit:

- single-Showing admission;
- venue entry;
- Audience projection;
- chat/reactions where allowed;
- public dice;
- public health status;
- public Scene/stage objects;
- fanmail;
- post-show surfaces.

Audience must not gain Cast, Crew, Director, or backstage authority through venue membership, generic Location membership, or presence.

---

## 14. Venue access versus production authority

These must remain distinct.

Being able to enter a venue does not automatically mean:

- edit it;
- control its stage;
- manage its Show;
- see private production data;
- see private Characters;
- become Crew/Director.

Audit every place where venue membership is used as a shortcut for production authority.

If venue membership is the correct source for a venue-specific action, keep it.

If not, reconcile.

---

## 15. Location / lot authority

Location membership may govern access to the overall lot/campus and specific shared spaces.

It must not silently grant unrelated Show authority.

Audit:

- location roles;
- public/front-door surfaces;
- Producer/Operator defaults;
- Third Place;
- venue discovery;
- access grants.

Make sure fresh-install Location behavior from K96 remains compatible.

---

## 16. Presence is not durable authority

Presence indicates who is currently there.

It must not become the source of truth for:

- Show participation;
- role;
- Character ownership;
- venue permission;
- Director authority.

Audit any remaining code where presence is used as a permission proxy.

---

## 17. Cohorts

Cohorts are contextual grouping, not role.

Audit:

- cohort-scoped projection;
- scene placement;
- dice visibility;
- private/public health;
- interactions;
- audience boundaries;
- backstage users.

A Producer/Director/Operator should not accidentally be forced into player cohort semantics unless intentionally participating as a Character.

Preserve prior fix patterns for backstage-role cohort override issues.

---

## 18. Character ownership and control

Separate:

- who owns a Character;
- who has selected a Character for a Show;
- who may view the Character;
- who may edit Character mechanics;
- who may direct the Character’s stage token;
- who may inspect private Character data.

Do not use one boolean to represent all of these.

K97 should reconcile existing authority checks, not redesign Character architecture.

---

## 19. “View as” / preview authority

Where Director or Operator can preview another role’s experience:

- preview must not mutate actual role;
- preview must not grant hidden data to the projected client beyond the privileged viewer’s current context;
- preview should reflect canonical role resolution;
- preview should not be accepted as the only proof of Audience/Cast behavior where real-account proof already exists elsewhere.

Audit existing Audience preview and role-preview tools.

---

## 20. Frontend gating is not authority

Buttons may hide based on role for usability.

Server handlers must still enforce authority.

Search for actions where:

- the frontend hides a control;
- but the backend route does not check the same canonical permission.

Fix those.

Do not duplicate full policy logic into the frontend.

---

## 21. WebSocket authority

Realtime messages must respect the same role/Show/venue boundaries as HTTP.

Audit:

- subscriptions;
- broadcasts;
- Show stage invalidation;
- dice;
- chat;
- reactions;
- stage object updates;
- drawing updates;
- Scene changes;
- presence;
- Audience projection changes.

A user should not receive a realtime event they would be forbidden to fetch through the canonical HTTP/read path.

---

## 22. Command authority

Slash commands must obey the same server authority as GUI actions.

Audit:

- `/showtime`;
- Scene controls;
- dice;
- stage actions;
- chat bridge;
- role-sensitive commands;
- any operator/admin commands.

Do not let the command path become a privileged bypass.

---

## 23. Background/automation authority

If Victory has background jobs, scheduled actions, or asynchronous operations:

- they must execute under explicit recorded authority;
- they must not assume Operator simply because no user is present;
- they should preserve the initiating user/context where practical.

Do not create new automation architecture in K97.

---

## 24. Access grants and memberships

Legacy access grants/memberships may still be useful.

But they should not remain accidental primary truth for newer Show/Showing authority.

Audit whether each is being used for:

- campus access;
- venue access;
- production role;
- temporary invitation;
- historical compatibility.

Document the intended meaning.

Retire only clearly superseded runtime paths.

Do not delete historical data merely to simplify code.

---

## 25. Role hierarchy is not always inheritance

Do not assume every higher role automatically inherits every lower-role experience.

Examples:

- Audience has no trays;
- Cast has role-specific live UI;
- Producer has Production Room;
- Operator has Cabin;
- Director has top-right live tools.

Higher authority may permit actions while still using a different experience.

K97 must preserve product experience distinctions.

---

## 26. Experience consistency

The human-facing role label and the actual authority should agree.

Audit visible labels such as:

- Audience;
- Cast;
- Crew;
- Director;
- Producer;
- Operator.

If the UI says Cast but the backend resolves Audience, that is a defect.

If the UI says Director but the action is denied due to a conflicting resolver, that is a defect.

If a user has multiple roles, the active context should make the current experience understandable.

---

## 27. Role transition

Audit legitimate transitions:

- invited visitor → Audience;
- Audience → Cast;
- Cast → Crew;
- Crew → Director;
- Director assignment/revocation;
- Producer assignment;
- Operator bootstrap/recovery;
- Character selection/change within a Show.

Verify:

- new authority appears when expected;
- revoked authority disappears;
- stale sessions/WebSockets do not retain old authority;
- old UI state does not misrepresent the new role.

---

## 28. Cross-Show isolation

Create at least two Shows with overlapping and non-overlapping users.

Prove:

- Director of Show A is not Director of Show B;
- Cast of Show A is not Cast of Show B unless explicitly rostered;
- Audience admission for Showing A does not authorize Showing B;
- cohort state does not cross Shows;
- Scene/stage authority does not cross Shows;
- Character selection remains Show-contextual.

---

## 29. Cross-Location isolation

If the current model supports multiple Locations/lots, create at least two.

Prove:

- location membership does not cross;
- venue discovery/access is scoped;
- Operator semantics are installation-wide only where intended;
- Producer/Director roles do not silently cross Locations;
- public surfaces remain intentionally public.

Do not expand multi-tenant architecture beyond the existing model.

---

## 30. Canonical resolver documentation

Create/update one concise internal authority map.

Suggested location:

`Construction/Identity/Canonical Role and Authority Resolution.md`

It should answer:

- what determines Operator;
- what determines Producer;
- what determines Director;
- what determines Crew;
- what determines Cast;
- what determines Audience;
- how Show/Showing/venue/Location scope modifies those;
- what Character selection means;
- what presence does **not** mean;
- where audience admissions fit;
- where access grants/memberships fit.

This is implementation doctrine, not user-facing policy text.

---

## 31. Repository audit

Search at minimum:

1. all role strings;
2. all `IsOperator*` helpers;
3. all Director/Producer checks;
4. venue-role lookup;
5. participation resolvers;
6. Show roster resolution;
7. selected Character resolution;
8. Show Run/session persona remnants;
9. location membership checks;
10. venue membership checks;
11. access-grant checks;
12. audience admission checks;
13. cohort overrides;
14. WebSocket authorization;
15. command authorization;
16. stage-object authorization;
17. Storyboards permissions;
18. eWrite permissions;
19. message/fanmail permissions;
20. asset permissions;
21. Director prep permissions;
22. Showtime/Showing permissions;
23. frontend role labels;
24. preview/view-as logic.

Do not ask Grant to identify these paths manually.

---

## 32. Mechanical test matrix

Build a compact test matrix.

Rows should include representative actions such as:

- enter venue;
- view Show;
- join Showing;
- view Scene;
- mutate Scene;
- view stage object;
- mutate stage object;
- roll dice;
- view private roll;
- configure Audience;
- manage roster;
- create Storyboard card;
- edit Storyboard card;
- view Character;
- edit Character;
- send message;
- view relationship;
- access Director prep;
- start/end Showtime.

Columns:

- Anonymous
- Audience
- Cast
- Crew
- Director
- Producer
- Operator

Do not expect every cell to be simple inheritance.

Record context assumptions.

---

## 33. Adversarial role tests

Attempt:

- payload role spoofing;
- URL/query role spoofing;
- client-side state spoofing;
- Character-ID swapping;
- Show-ID swapping;
- cohort-ID swapping;
- admission-ID swapping;
- stale role after revocation;
- unauthorized WebSocket command;
- direct endpoint call with hidden GUI control.

K96 may already have security coverage for some of these.

K97 focuses on **semantic consistency of role resolution**, not broad internet hardening.

Reuse prior tests where useful.

---

## 34. No capability expansion

K97 may expose missing capability parity.

Do not implement it unless the missing behavior is actually caused by broken role resolution.

Examples deferred:

- First Theater gaining Catharsis announcements;
- new Producer tools;
- new Crew tools;
- new Audience features;
- new role types.

Record them for 102+.

---

## 35. Human spot-check

This kernel does **not** require another full role-by-role dress rehearsal.

After mechanical proof, Grant should spot-check a small set of historically confusing paths:

- Cast in an active Show;
- pure Audience in admitted Showing;
- Director in live venue;
- Producer outside active Director role;
- Operator using ordinary product surfaces.

The purpose is to catch labels/experience disagreement.

Do not turn this into another K93.

---

## 36. Pass criteria

K97 passes when:

- each major role has one documented canonical resolution path;
- no Grant-specific operator shortcut survives;
- selected Character remains canonical for Show participation;
- sitewide active Character is not reused as Show authority;
- presence is not used as durable authority;
- venue access and production authority are distinct;
- audience admissions remain Showing-scoped;
- role resolution is Show/Location-contextual where intended;
- duplicate contradictory role checks are materially reduced;
- Cast is not misclassified as Audience through missing context hints;
- Producer/Director fallback semantics are explicit;
- backstage users are not incorrectly subjected to player cohort semantics;
- HTTP, WebSocket, and command paths agree on authority;
- frontend role labels agree with backend resolution;
- cross-Show isolation passes;
- cross-Location isolation passes where applicable;
- revocation/role transition updates authority correctly;
- the mechanical role/action matrix passes;
- a small human spot-check finds no major experience contradiction.

---

## 37. Reportback

Report:

1. all role/authority resolvers found;
2. canonical resolver chosen for each major authority question;
3. duplicate/conflicting paths removed or deprecated;
4. known carried bugs and their outcomes;
5. selected Character / Show participation proof;
6. venue versus production authority proof;
7. Producer/Director/Operator semantics;
8. Audience admission semantics;
9. cohort/backstage behavior;
10. HTTP/WebSocket/command consistency;
11. cross-Show proof;
12. cross-Location proof;
13. role-transition/revocation proof;
14. test matrix;
15. residual ambiguities;
16. deferred capability ideas;
17. human spot-check result.

Final status:

- **PASS**
- **PARTIAL**
- **FAIL**

Do not call PASS while contradictory role resolvers remain active for the same product action.

---

## 38. Completion condition

Kernel 97 passes when Victory can answer:

> **Who is this person here, and what may they do here?**

with one consistent answer across the product.

The answer may change by Show, Showing, venue, Character, or Location.

It may not change merely because two different code paths happened to be written in different kernels.
