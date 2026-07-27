BEGIN;

-- Kernel 74: the Player-controlled tail of the Locked Courtyard tutorial.
-- Kessa completion -> door hotspot reveal -> freeform door intention ->
-- Ra's guided dialogue -> a participant-local tutorial-handoff projection,
-- with no Director GO anywhere inside the sequence.
--
-- Every table here is bounded to that sequence on purpose. This is NOT a
-- quest tracker, NOT a dialogue-graph engine, and NOT a second shared Show
-- current Scene -- see the per-table notes for exactly where each boundary
-- is drawn and why.

-- participant_tutorial_progress: durable, bounded milestone records for one
-- Player's participation context (kernel-74 S5.1). Deliberately a dedicated
-- table with a CHECK-constrained milestone_key rather than a generic
-- key/value progress store: S15 excludes general quest tracking, and an
-- enumerated column makes "which milestones exist" a schema fact a reviewer
-- can read rather than a convention buried in Go.
--
-- Scoped to character_card_id as well as user_id (S5.2): switching the
-- selected Character must not transfer the previous Character's tutorial
-- completion or door intention, and prior records stay associated with the
-- Character who actually performed them. The UNIQUE constraint is the
-- idempotency mechanism (S5.3) -- double-clicks, retries, reconnects and
-- replayed requests all collide into one row via ON CONFLICT DO NOTHING.
CREATE TABLE IF NOT EXISTS participant_tutorial_progress (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  character_card_id UUID NOT NULL REFERENCES character_cards(id) ON DELETE CASCADE,
  show_id UUID NOT NULL REFERENCES shows(id) ON DELETE CASCADE,
  show_run_id UUID REFERENCES show_runs(id) ON DELETE SET NULL,
  show_scene_placement_id UUID REFERENCES show_scene_placements(id) ON DELETE SET NULL,
  milestone_key TEXT NOT NULL,
  payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT participant_tutorial_progress_milestone_check CHECK (milestone_key IN (
    'kessa_intro_completed',
    'door_intention_submitted',
    'ra_intro_started',
    'ra_intro_completed',
    'tutorial_handoff_entered'
  )),
  UNIQUE (user_id, character_card_id, show_id, milestone_key)
);

CREATE INDEX IF NOT EXISTS idx_participant_tutorial_progress_lookup
  ON participant_tutorial_progress(show_id, user_id, character_card_id);

-- participant_freeform_submissions: the Player's own words at the door
-- (S7.2). Stored verbatim as plain text -- never HTML, never rewritten, and
-- never prepended to authored narration server-side (S1.4: Players may type
-- a complete sentence, so "You start to <input>" is unsafe grammar).
-- Escaping is a render-time concern at every surface that displays it.
--
-- idempotency_key mirrors character_inventory_purchase_attempts (migration
-- 059): a retried submission returns the original row rather than creating
-- a second intention or a second Directors+ note.
CREATE TABLE IF NOT EXISTS participant_freeform_submissions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  participant_interaction_id UUID NOT NULL REFERENCES participant_interactions(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  character_card_id UUID NOT NULL REFERENCES character_cards(id) ON DELETE CASCADE,
  show_id UUID NOT NULL REFERENCES shows(id) ON DELETE CASCADE,
  session_id UUID REFERENCES sessions(id) ON DELETE SET NULL,
  show_scene_placement_id UUID REFERENCES show_scene_placements(id) ON DELETE SET NULL,
  submitted_text TEXT NOT NULL,
  idempotency_key TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (participant_interaction_id, user_id, character_card_id, idempotency_key)
);

-- A second, key-independent uniqueness floor: even a client that generates
-- a fresh idempotency_key on every retry cannot produce two door intentions
-- for the same Character in the same Show. S7.2 makes the submission
-- one-shot ("cannot be edited after Ra's interruption begins"), so this is
-- the honest constraint -- the idempotency key alone would not enforce it.
CREATE UNIQUE INDEX IF NOT EXISTS uq_participant_freeform_submissions_once
  ON participant_freeform_submissions(participant_interaction_id, user_id, character_card_id);

CREATE INDEX IF NOT EXISTS idx_participant_freeform_submissions_show
  ON participant_freeform_submissions(show_id, created_at DESC);

-- dialogue_packets / dialogue_topics / dialogue_topic_prerequisites: the
-- smallest reusable guided-dialogue model (S9.2). Location-scoped by slug,
-- mirroring merchant_packets (migration 060) exactly.
--
-- Bounded on purpose (S9.2, S15): topics are a flat authored list with
-- prerequisite edges, not a cyclic graph, not a script, not a chatbot. The
-- prerequisite table can express a DAG; Go validates acyclicity on load so
-- a bad seed fails loudly instead of hanging a Player mid-conversation.
--
-- portrait_url is a plain static path (e.g. /assets/ra.png) rather than an
-- assets(id) FK: the assets table requires a real uploaded file with a
-- checksum and owner, which a migration cannot honestly synthesize.
-- portrait_asset_id is here for when real art is uploaded through the
-- normal path; readers prefer it over portrait_url when set.
CREATE TABLE IF NOT EXISTS dialogue_packets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  slug TEXT NOT NULL,
  npc_name TEXT NOT NULL,
  portrait_asset_id UUID REFERENCES assets(id) ON DELETE SET NULL,
  portrait_url TEXT NOT NULL DEFAULT '',
  opening_narration TEXT NOT NULL DEFAULT '',
  opening_line TEXT NOT NULL DEFAULT '',
  closing_narration TEXT NOT NULL DEFAULT '',
  leave_label TEXT NOT NULL DEFAULT 'Leave',
  destination_scene_slug TEXT NOT NULL DEFAULT '',
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (location_id, slug)
);

CREATE TABLE IF NOT EXISTS dialogue_topics (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  packet_id UUID NOT NULL REFERENCES dialogue_packets(id) ON DELETE CASCADE,
  topic_key TEXT NOT NULL,
  label TEXT NOT NULL,
  response_text TEXT NOT NULL,
  required_for_completion BOOLEAN NOT NULL DEFAULT FALSE,
  sort_order INTEGER NOT NULL DEFAULT 0,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (packet_id, topic_key)
);

CREATE INDEX IF NOT EXISTS idx_dialogue_topics_packet
  ON dialogue_topics(packet_id, sort_order);

CREATE TABLE IF NOT EXISTS dialogue_topic_prerequisites (
  topic_id UUID NOT NULL REFERENCES dialogue_topics(id) ON DELETE CASCADE,
  requires_topic_id UUID NOT NULL REFERENCES dialogue_topics(id) ON DELETE CASCADE,
  PRIMARY KEY (topic_id, requires_topic_id),
  CONSTRAINT dialogue_topic_prerequisites_no_self CHECK (topic_id <> requires_topic_id)
);

-- participant_dialogue_topic_views: per-Player/Character topic progress
-- (S9.5). Survives refresh and reconnect because it is a row, not browser
-- state (S5.1's explicit "do not hide critical progress only in browser
-- storage"). UNIQUE makes a re-read of an already-seen topic a no-op rather
-- than a duplicate record.
CREATE TABLE IF NOT EXISTS participant_dialogue_topic_views (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  character_card_id UUID NOT NULL REFERENCES character_cards(id) ON DELETE CASCADE,
  show_id UUID NOT NULL REFERENCES shows(id) ON DELETE CASCADE,
  topic_id UUID NOT NULL REFERENCES dialogue_topics(id) ON DELETE CASCADE,
  viewed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (user_id, character_card_id, show_id, topic_id)
);

CREATE INDEX IF NOT EXISTS idx_participant_dialogue_topic_views_lookup
  ON participant_dialogue_topic_views(show_id, user_id, character_card_id);

-- participant_local_projections: the participant-local stage projection
-- (S11.2). This is the one genuinely new architectural idea in Kernel 74,
-- so the boundary matters:
--
--   shows.current_show_scene_placement_id
--     -> the shared, authoritative Scene for the whole table. UNCHANGED by
--        anything in this kernel (S11.1, S16.15).
--   participant_local_projections
--     -> a temporary per-Player presentation layered over that shared stage,
--        scoped to tutorial handoff, cleared when a Director+ later flies a
--        shared Scene outside the tutorial sequence (S11.3).
--
-- Critically this does NOT fork the element/visibility model, which
-- cues/types.go:43-48 deliberately refused to do. It substitutes WHICH
-- Scene's composition one viewer resolves; every element inside that Scene
-- still goes through the single existing visibility filter in
-- world/snapshot.go. There is exactly one object model.
--
-- The destination is a Scene, not a free-form payload, so the handoff map
-- is authored through the existing Scene Library + Scene Setup composer and
-- rendered by the existing stage renderer -- no second authoring surface.
-- The Player never names it (S13): it comes from the dialogue packet.
CREATE TABLE IF NOT EXISTS participant_local_projections (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  character_card_id UUID NOT NULL REFERENCES character_cards(id) ON DELETE CASCADE,
  show_id UUID NOT NULL REFERENCES shows(id) ON DELETE CASCADE,
  destination_scene_id UUID NOT NULL REFERENCES scenes(id) ON DELETE CASCADE,
  origin_show_scene_placement_id UUID REFERENCES show_scene_placements(id) ON DELETE SET NULL,
  entered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  cleared_at TIMESTAMPTZ,
  cleared_reason TEXT NOT NULL DEFAULT ''
);

-- At most one active projection per (user, show) -- S5.3's "must not create
-- multiple local handoff projections" as a database fact, not a Go check.
CREATE UNIQUE INDEX IF NOT EXISTS uq_participant_local_projections_active
  ON participant_local_projections(user_id, show_id)
  WHERE cleared_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_participant_local_projections_show
  ON participant_local_projections(show_id) WHERE cleared_at IS NULL;

-- Two new participant interaction types. Kernel 73's CHECK allowed exactly
-- one value and its own comment anticipated more; this is that extension,
-- still an enumerated allowlist rather than a free-text column.
ALTER TABLE participant_interactions
  DROP CONSTRAINT IF EXISTS participant_interactions_type_check;
ALTER TABLE participant_interactions
  ADD CONSTRAINT participant_interactions_type_check CHECK (interaction_type IN (
    'open_equip_mode',
    'freeform_submission',
    'guided_dialogue'
  ));

-- interaction_hotspot: a positioned, resizable, invisible-or-lightly-
-- highlighted region with a Player-local nameplate, aligned over door
-- artwork that already exists in the Courtyard map (S1.1). Explicitly NOT a
-- second visible door token duplicating the art.
ALTER TABLE scene_stage_elements
  DROP CONSTRAINT IF EXISTS scene_stage_elements_kind_check;
ALTER TABLE scene_stage_elements
  ADD CONSTRAINT scene_stage_elements_kind_check CHECK (kind IN (
    'token', 'index_card', 'map_backdrop', 'grid_config', 'interaction_hotspot'
  ));

-- Normalized 0-1 size, matching the normalized 0-1 position already stored
-- in scene_stage_elements.position. Nullable and kind-agnostic: a hotspot
-- needs it to cover the door art, and nothing else is forced to carry it.
ALTER TABLE scene_stage_elements
  ADD COLUMN IF NOT EXISTS width DOUBLE PRECISION,
  ADD COLUMN IF NOT EXISTS height DOUBLE PRECISION;

-- requires_milestone: the server-authoritative reveal gate (S6.2). A
-- binding carrying a milestone key is resolved out of a Player's snapshot
-- entirely until that Player's participation has recorded it. It lives on
-- the binding rather than the element because it gates reachability of the
-- bound interaction, which is exactly what a binding is for.
ALTER TABLE stage_element_bindings
  ADD COLUMN IF NOT EXISTS requires_milestone TEXT NOT NULL DEFAULT '';

-- --- Seed: Ra's guided-dialogue packet -------------------------------------
--
-- Source grounding (kernel-74 S2): the Locked Courtyard tutorial --
-- "Ra - The Guard at the Gate" pp.13-14, "Beyond the Door Lies Niava" p.15,
-- "The Crown Bet: Ra's Challenge Explained" p.32.
--
-- Each response below is marked either "canonical source content" (the
-- substance comes from those sections) or "Victory onboarding adaptation"
-- (operator-directed wording or connective logic that is NOT a verbatim
-- book quotation). No permanent lore is invented here to make a test pass.
INSERT INTO dialogue_packets (
  location_id, slug, npc_name, portrait_url, opening_narration, opening_line,
  closing_narration, leave_label, destination_scene_slug
)
SELECT
  l.id, 'ra', 'Ra', '/assets/ra.png',
  -- canonical source content: Ra interrupts before the door attempt resolves.
  'A voice cuts across the courtyard behind you.',
  -- canonical source content (p.13): Ra enters in Command.
  'Hey! Stop that! What are you doing?',
  -- Victory onboarding adaptation: the source shows no lock on the courtyard
  -- side, and the operator requires Ra's unlocking to reveal a believable
  -- mechanism on THIS side (S1.8). Concealed, not an ordinary padlock -- this
  -- must not contradict the door's opening description.
  'Ra crouches and works his fingers beneath an iron plate you had taken for part of the door''s reinforcement. It lifts. Set into the stone behind it is a recessed lock, worn smooth with use. He turns a key in it, and somewhere inside the wall a heavy bolt withdraws.',
  'Leave Ra',
  'tutorial-handoff'
FROM locations l
WHERE l.slug = 'amurray-family'
  AND NOT EXISTS (SELECT 1 FROM dialogue_packets p WHERE p.location_id = l.id AND p.slug = 'ra');

INSERT INTO dialogue_topics (packet_id, topic_key, label, response_text, required_for_completion, sort_order)
SELECT p.id, v.topic_key, v.label, v.response_text, v.required, v.ord
FROM dialogue_packets p
JOIN locations l ON l.id = p.location_id AND l.slug = 'amurray-family'
JOIN (VALUES
  -- canonical source content (pp.13-14): Ra locked the door himself, and
  -- stepped away briefly. The "nature called" line is a Victory onboarding
  -- adaptation of that beat, not a verbatim quotation -- it exists because
  -- the operator wanted Ra's reason for inspecting what leaves the market to
  -- be concrete rather than officious.
  ('why-locked', 'Why is this door locked?',
   'He has the decency to look embarrassed. "That would be me. I locked it." A pause. "Sorry. Nature called, and I couldn''t let people leave before I got back and checked what was walking out of the market. It''s the only way out to the road, so -- one door, one lock, no arguments."',
   FALSE, 1),
  -- canonical source content (p.13): Ra's identity. He is earnest and
  -- underinformed; he does not know more than he says.
  ('who-are-you', 'Who are you?',
   '"Ra," he says, straightening up like the name should mean something and then visibly deciding it doesn''t. "I keep this gate. That''s most of it." He shrugs. "I''m not the one who decides things around here. I''m the one who stands here while they''re decided."',
   FALSE, 2),
  -- canonical source content (pp.13-14): Ra was sent to find or inform the
  -- characters -- this is why he is here at all.
  ('why-looking', 'Why were you looking for me?',
   '"Because someone sent me to find you." He says it plainly, as if the strangeness of it has not caught up with him either. "There''s an opportunity, and your name came up. They call it the Crown Bet. I was told to make sure you heard about it before you walked out that door and out of everyone''s reach."',
   TRUE, 3),
  -- canonical source content (p.32): the Crown Bet challenge -- return with
  -- the crown of a named king from the fractured Turtle continent.
  ('crown-bet', 'What is the Crown Bet?',
   '"It''s a wager, and it''s exactly as mad as it sounds." He counts it off on his fingers. "You go to the Turtle continent -- the broken one, the one that came apart and never got put back. You find a king there. A named one, a real one, not some hill with a flag on it. And you come back with his crown." He lets that sit. "That''s the whole bet. Nobody''s asking you to explain how."',
   TRUE, 4),
  -- canonical source content (p.32): the Games opportunity, fame, prize money.
  ('what-if-succeed', 'What happens if I succeed?',
   '"Then you stop being someone I had to go looking for." He grins, and for a moment he is not guarding anything. "A featured place in the Games. Your name on the bill, not in the crowd. And the purse -- I''ve heard the number and I don''t entirely believe it, so I''ll let someone else tell you. People have built whole lives on less than the losing share."',
   FALSE, 5),
  -- canonical source content (p.15): beyond the door lies the road toward
  -- shipping and the wider world. Deliberately does NOT name the Director's
  -- next Scene (S1.10) -- Ra may introduce Niava's Crown Bet, but Victory
  -- must not assume the campaign goes there.
  ('beyond-the-door', 'What lies beyond the door?',
   '"The road," he says. "It runs down out of the market and keeps going until it reaches water, and at the water there are ships, and the ships go everywhere." He nods at the oak. "That''s all a door is, really. This side is the market. That side is everywhere else."',
   FALSE, 6)
) AS v(topic_key, label, response_text, required, ord) ON TRUE
WHERE p.slug = 'ra'
  AND NOT EXISTS (SELECT 1 FROM dialogue_topics t WHERE t.packet_id = p.id AND t.topic_key = v.topic_key);

-- Prerequisite edges (S9.3's suggested dependency pattern):
--   why-looking -> unlocks crown-bet
--   crown-bet   -> unlocks what-if-succeed and beyond-the-door
-- why-locked and who-are-you are open from the start, so a Player who asks
-- in a non-default order is never stuck.
INSERT INTO dialogue_topic_prerequisites (topic_id, requires_topic_id)
SELECT t.id, req.id
FROM dialogue_packets p
JOIN locations l ON l.id = p.location_id AND l.slug = 'amurray-family'
JOIN (VALUES
  ('crown-bet', 'why-looking'),
  ('what-if-succeed', 'crown-bet'),
  ('beyond-the-door', 'crown-bet')
) AS v(topic_key, requires_key) ON TRUE
JOIN dialogue_topics t ON t.packet_id = p.id AND t.topic_key = v.topic_key
JOIN dialogue_topics req ON req.packet_id = p.id AND req.topic_key = v.requires_key
WHERE p.slug = 'ra'
ON CONFLICT DO NOTHING;

-- --- Seed: the setting-neutral tutorial-handoff Scene ----------------------
--
-- S1.10/S11.4: Ra may introduce Niava's Crown Bet, but the destination must
-- NOT hardcode the Director's next campaign Scene. This is a generic
-- outside/handoff projection with placeholder art and copy; Kernel 75
-- finalizes both. It is a normal Scene Library row, so a Director can edit
-- it in Scene Setup without a code change.
INSERT INTO scenes (location_id, slug, title, short_title, default_venue_id, audience_title, player_brief, status)
SELECT
  l.id, 'tutorial-handoff', 'Outside the Courtyard', 'Handoff',
  (SELECT v.id FROM venues v WHERE v.slug = 'catharsis'),
  'Outside the Courtyard',
  'Placeholder tutorial-handoff projection. Kernel 75 finalizes the art and copy.',
  'ready'
FROM locations l
WHERE l.slug = 'amurray-family'
  AND NOT EXISTS (SELECT 1 FROM scenes s WHERE s.location_id = l.id AND s.slug = 'tutorial-handoff');

INSERT INTO scene_stage_elements (scene_id, kind, label, data, position, sort_order)
SELECT s.id, 'map_backdrop', 'Outside the Courtyard',
  '{"content_url": "/assets/tutorial-handoff.png", "display_mode": "fullscreen", "fit": "cover", "grid_enabled": false}'::jsonb,
  '{}'::jsonb, 0
FROM scenes s
JOIN locations l ON l.id = s.location_id AND l.slug = 'amurray-family'
WHERE s.slug = 'tutorial-handoff'
  AND NOT EXISTS (
    SELECT 1 FROM scene_stage_elements e WHERE e.scene_id = s.id AND e.kind = 'map_backdrop'
  );

INSERT INTO scene_stage_elements (scene_id, kind, label, data, position, sort_order)
SELECT s.id, 'index_card', 'Tutorial complete',
  '{"text": "Tutorial complete. Your Character is equipped and ready. The Narrator will take it from here."}'::jsonb,
  '{"x": 0.5, "y": 0.72}'::jsonb, 1
FROM scenes s
JOIN locations l ON l.id = s.location_id AND l.slug = 'amurray-family'
WHERE s.slug = 'tutorial-handoff'
  AND NOT EXISTS (
    SELECT 1 FROM scene_stage_elements e WHERE e.scene_id = s.id AND e.kind = 'index_card'
  );

COMMIT;
