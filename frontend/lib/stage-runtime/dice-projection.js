// Kernel 86A: Spatial Dice Landing & Explosion-Safe Projection. Consumes
// already-canonical Stage Effect payloads (backend/internal/stageeffects)
// pushed over the venue websocket -- this module never determines a die's
// final value or explodes anything; it only animates toward what the
// server already resolved (Kernel 86A spec 2.1). A bounded sibling module
// of dice.js (the roll REQUEST/history UI) rather than an addition to
// runtime.js, matching this directory's established "small UMD module,
// deps-injected, one concern" shape (dice.js, kernel85-cohort-tools.js,
// geometry.js, ...).
//
// Kernel 86 rendered a roll as one grouped screen-space (HUD) node. Kernel
// 86A splits that into two independently-lifecycled pieces:
//   - a WORLD-space cluster of individual die nodes, one per base die,
//     each landing at its own map-relative coordinate (mounted into the
//     caller's world/map layer, e.g. pinnedObjectLayer, so pan/zoom carries
//     them naturally via the existing stageCamera -> worldLayer transform);
//   - a screen-space (HUD) announcement banner -- the preserved Kernel 86
//     caption/total summary -- that only appears once every die in the
//     roll has settled, and fades independently of pinned dice (spec 1.10).
// If no active map is loaded, there is no world-space surface to land
// dice on; the module falls back to the original Kernel 86 grouped-HUD
// presentation rather than placing dice at a meaningless (0,0).
(function (root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  }
  root.VictoryStageDiceProjection = factory();
})(typeof globalThis !== "undefined" ? globalThis : window, function () {
  const DEFAULT_TOKEN_SIZE = 64;
  const DEFAULT_DURATION_MS = 8500; // matches network.stageEffectDefaultDurationMs
  const ROLL_PHASE_MS = 650;
  const ROLL_TICK_MS = 70;
  const STAGGER_MS = 90;
  const FADE_MS = 300;
  const MAX_QUEUE_LENGTH = 12;
  const MAX_DICE_SHOWN = 12;
  const LANDING_ATTEMPTS = 8;

  function clampNumber(value, min, max) {
    const n = Number(value);
    if (!Number.isFinite(n)) return min;
    return Math.max(min, Math.min(max, n));
  }

  // normalizeEffect accepts the raw Stage Effect JSON (snake_case, as
  // sent by the server: effect_id, source_action_id, cohort_id, actor_id,
  // duration_ms, ...) and produces the shape this module works with
  // internally, mirroring dice.js's normalizeRollAction convention of
  // doing this translation once at the boundary.
  function normalizeEffect(raw) {
    const source = raw && typeof raw === "object" ? raw : null;
    if (!source) return null;
    const payload = source.payload && typeof source.payload === "object" ? source.payload : {};
    const actor = payload.actor && typeof payload.actor === "object" ? payload.actor : {};
    const dice = Array.isArray(payload.dice) ? payload.dice : [];
    return {
      id: String(source.effect_id || source.id || ""),
      type: String(source.type || "dice_roll"),
      sourceActionId: String(source.source_action_id || ""),
      sessionId: String(source.session_id || ""),
      showId: String(source.show_id || ""),
      cohortId: String(source.cohort_id || ""),
      audience: String(source.audience || "show"),
      actorId: String(source.actor_id || actor.user_id || ""),
      actorDisplayName: String(actor.display_name || actor.displayName || actor.handle || "Unknown"),
      label: String(source.label || payload.label || ""),
      expression: String(payload.expression || ""),
      dice: dice.map((die, index) => ({
        index: Number(die?.index ?? index) || 0,
        chain: Array.isArray(die?.chain) ? die.chain.map((v) => Number(v)).filter((v) => Number.isFinite(v)) : [],
        subtotal: Number(die?.subtotal ?? 0) || 0,
      })),
      modifier: Number(payload.modifier ?? 0) || 0,
      total: Number(payload.total ?? 0) || 0,
      explosionCount: Number(payload.explosion_count ?? 0) || 0,
      durationMs: Number(source.duration_ms ?? DEFAULT_DURATION_MS) || DEFAULT_DURATION_MS,
      pinned: Boolean(source.pinned),
      raw: source,
    };
  }

  // --- deterministic per-die placement -------------------------------
  // Landing coordinates must be identical for every viewer and stable
  // across reconnect (spec 4.3, 9): a future Cartograph step may reference
  // these positions, so two players can never see the same die in two
  // different map spots. The server does not track any viewer's camera,
  // so positions are a pure function of (effect id, die index, the
  // session's shared map bounds, token size) -- every client recomputes
  // the identical answer independently, with no new backend storage.

  function hashSeed(str) {
    let h = 2166136261 >>> 0; // FNV-1a
    for (let i = 0; i < str.length; i++) {
      h ^= str.charCodeAt(i);
      h = Math.imul(h, 16777619) >>> 0;
    }
    return h >>> 0;
  }

  function mulberry32(seed) {
    let a = seed >>> 0;
    return function () {
      a |= 0;
      a = (a + 0x6d2b79f5) | 0;
      let t = Math.imul(a ^ (a >>> 15), 1 | a);
      t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
      return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
    };
  }

  function deterministicUnitPoint(effectId, dieIndex, attempt) {
    const rand = mulberry32(hashSeed(`${effectId}:${dieIndex}:${attempt}`));
    return { u: rand(), v: rand() };
  }

  // computeLandingPositions picks `count` distinguishable points (spec
  // 1.6) inside a safe inset rectangle of the active map's world bounds
  // (spec 1.5). No physics engine (spec 6): a bounded deterministic
  // retry loop is enough -- try a few deterministic candidates per die,
  // keep the one farthest from already-placed dice.
  function computeLandingPositions(effectId, count, mapBounds, tokenSize) {
    const inset = tokenSize * 0.75;
    const rectWidth = Math.max(1, mapBounds.width - inset * 2);
    const rectHeight = Math.max(1, mapBounds.height - inset * 2);
    const rectX = mapBounds.x + Math.min(inset, mapBounds.width / 2);
    const rectY = mapBounds.y + Math.min(inset, mapBounds.height / 2);
    const minSeparation = tokenSize * 1.15;

    const positions = [];
    for (let i = 0; i < count; i++) {
      let best = null;
      let bestScore = -Infinity;
      for (let attempt = 0; attempt < LANDING_ATTEMPTS; attempt++) {
        const { u, v } = deterministicUnitPoint(effectId, i, attempt);
        const candidate = { x: rectX + u * rectWidth, y: rectY + v * rectHeight };
        let minDist = Infinity;
        for (const p of positions) {
          minDist = Math.min(minDist, Math.hypot(p.x - candidate.x, p.y - candidate.y));
        }
        if (minDist >= minSeparation) {
          best = candidate;
          break;
        }
        if (minDist > bestScore) {
          bestScore = minDist;
          best = candidate;
        }
      }
      positions.push(best || { x: rectX, y: rectY });
    }
    return positions;
  }

  function entryOffsetFor(effectId, dieIndex) {
    const { u, v } = deterministicUnitPoint(effectId, dieIndex, 99);
    const angle = u * Math.PI * 2;
    const distance = 90 + v * 60;
    return { dx: Math.cos(angle) * distance, dy: Math.sin(angle) * distance };
  }

  function centroid(positions) {
    if (!positions.length) return { x: 0, y: 0 };
    const sum = positions.reduce((acc, p) => ({ x: acc.x + p.x, y: acc.y + p.y }), { x: 0, y: 0 });
    return { x: sum.x / positions.length, y: sum.y / positions.length };
  }

  function normalizeMapBounds(raw) {
    if (!raw || typeof raw !== "object") return null;
    const x = Number(raw.x) || 0;
    const y = Number(raw.y) || 0;
    const width = Number(raw.width);
    const height = Number(raw.height);
    if (!Number.isFinite(width) || !Number.isFinite(height) || width <= 0 || height <= 0) return null;
    return { x, y, width, height };
  }

  function createDiceProjectionController(deps = {}) {
    const PIXI = deps.PIXI || (typeof window !== "undefined" ? window.PIXI : null);
    const setTimeoutFn = deps.setTimeout || ((fn, ms) => setTimeout(fn, ms));
    const clearTimeoutFn = deps.clearTimeout || ((id) => clearTimeout(id));
    const getStageSize = typeof deps.getStageSize === "function" ? deps.getStageSize : () => ({ width: 960, height: 640 });
    const getDefaultTokenSize = typeof deps.getDefaultTokenSize === "function" ? deps.getDefaultTokenSize : () => DEFAULT_TOKEN_SIZE;
    const getMapWorldBounds = typeof deps.getMapWorldBounds === "function" ? deps.getMapWorldBounds : () => null;
    const sendAction = typeof deps.sendAction === "function" ? deps.sendAction : () => false;
    const onEffectChange = typeof deps.onEffectChange === "function" ? deps.onEffectChange : () => {};

    let hudContainer = null;
    let worldContainer = null;
    let queue = [];
    let current = null;
    const pinned = new Map(); // effectId -> { effect, clusterNode }
    let destroyed = false;

    function mount(hudLayer, worldLayer) {
      if (!PIXI || !hudLayer) return null;
      if (!hudContainer) {
        hudContainer = new PIXI.Container();
        hudContainer.sortableChildren = true;
      }
      hudLayer.addChild(hudContainer);

      const targetWorldLayer = worldLayer || hudLayer;
      if (!worldContainer) {
        worldContainer = new PIXI.Container();
        worldContainer.sortableChildren = true;
      }
      targetWorldLayer.addChild(worldContainer);

      // The server sends stage_effects/pinned right after "snapshot",
      // often before the Pixi scene finishes constructing (containers
      // don't exist yet) AND before the map texture/bounds have settled
      // to their final value -- runtime.js's renderVenueMapLayer returns
      // a screen-sized PLACEHOLDER rectangle while the texture is still
      // loading, not null, so a single "is mapBounds non-null yet" check
      // can lock in a placement computed against that placeholder rather
      // than the real fitted-image bounds. Force a few unconditional
      // recomputes over ~2.4s (comfortably covers a normal map image
      // load) so the final attempt lands on the settled value regardless
      // of what any earlier one saw; a currently-correct entry just gets
      // recomputed to the identical answer each time (harmless).
      forceReconcileAllPinned(6, 400);
      return { hud: hudContainer, world: worldContainer };
    }

    // reconcilePinnedPlacements (flag-gated, cheap) handles the ordinary
    // "next roll happens long after load" case. forceReconcileAllPinned
    // (unconditional, time-boxed) exists only for the narrow post-connect
    // window where the map's own bounds computation is itself still
    // settling -- see mount()'s comment.
    function forceReconcileAllPinned(attemptsLeft, delayMs) {
      if (attemptsLeft <= 0 || destroyed) return;
      setTimeoutFn(() => {
        if (destroyed || pinned.size === 0) return;
        for (const entry of Array.from(pinned.values())) {
          applyPinned(entry.effect.raw || entry.effect);
        }
        forceReconcileAllPinned(attemptsLeft - 1, delayMs);
      }, delayMs);
    }

    function clearTimers(entry) {
      if (!entry) return;
      for (const t of entry.timers || []) {
        clearTimeoutFn(t);
      }
      entry.timers = [];
    }

    // PIXI v7 API (frontend/lib/pixi.min.js is pinned at 7.4.3, not the v8
    // fill()/stroke()/roundRect() chain) -- beginFill/lineStyle before
    // drawRoundedRect, matching every other Graphics user in this
    // directory (e.g. runtime.js's proscenium/curtain chrome).
    function buildDieFace(size) {
      const g = new PIXI.Graphics();
      g.lineStyle(Math.max(2, size * 0.04), 0xd9c46a, 0.9);
      g.beginFill(0x1c1424, 0.92);
      g.drawRoundedRect(-size / 2, -size / 2, size, size, Math.round(size * 0.18));
      g.endFill();
      return g;
    }

    function makeText(text, style) {
      return new PIXI.Text(text, new PIXI.TextStyle(Object.assign({ fontFamily: "Arial" }, style)));
    }

    function buildDieWorldNode(size, faceValue) {
      const node = new PIXI.Container();
      const face = buildDieFace(size);
      const text = makeText(String(faceValue ?? "?"), { fill: 0xf3ecd8, fontSize: Math.round(size * 0.42), fontWeight: "700" });
      text.anchor.set(0.5);
      node.addChild(face, text);
      node.__face = face;
      node.__text = text;
      return node;
    }

    function randomFace(die) {
      const maxKnown = die.chain.length ? Math.max(...die.chain) : 20;
      return 1 + Math.floor(Math.random() * Math.max(2, maxKnown));
    }

    function finalFace(die) {
      if (die.subtotal) return die.subtotal;
      return die.chain.length ? die.chain[die.chain.length - 1] : 0;
    }

    function captionLine(effect) {
      const who = effect.actorDisplayName || "Unknown";
      const what = effect.label || effect.expression || "a roll";
      return `${who} rolled ${what}`;
    }

    function totalLine(effect) {
      const modifierText = effect.modifier > 0 ? ` +${effect.modifier}` : effect.modifier < 0 ? ` ${effect.modifier}` : "";
      return `= ${effect.total}${modifierText ? ` (mod${modifierText})` : ""}`;
    }

    // buildActionButton creates a small interactive label (Pin / Dismiss).
    // Kernel 86 authority note (still true under 86A): the server is the
    // actual authority (stageEffectAuthorized), so this button is offered
    // to every viewer who can see the projection at all rather than
    // duplicating that role/cohort logic client-side; an unauthorized
    // click is simply rejected server-side with no visible effect.
    function buildActionButton() {
      const text = makeText("", { fill: 0x8fb3ff, fontSize: 13, fontWeight: "600" });
      text.anchor.set(0.5, 0);
      text.alpha = 0;
      text.eventMode = "none";
      text.cursor = "pointer";
      return text;
    }

    function setActionButton(button, label, onClick) {
      if (!button) return;
      button.removeAllListeners?.("pointertap");
      if (!label) {
        button.alpha = 0;
        button.eventMode = "none";
        return;
      }
      button.text = label;
      button.alpha = 1;
      button.eventMode = "static";
      button.on("pointertap", onClick);
    }

    // --- HUD announcement (preserved Kernel 86 summary, spec 1.1) ------

    function placeAtTransientSlot(node, width) {
      const size = getStageSize();
      const w = Number(size?.width) || 960;
      const h = Number(size?.height) || 640;
      const halfWidth = Math.max(80, width / 2);
      const x = clampNumber(w / 2, halfWidth, Math.max(halfWidth, w - halfWidth));
      const y = clampNumber(h * 0.14, 40, Math.max(40, h - 40));
      node.position.set(x, y);
    }

    function buildAnnouncementNode(effect) {
      const node = new PIXI.Container();
      const caption = makeText(captionLine(effect), { fill: 0xf3ecd8, fontSize: 15, fontWeight: "600", align: "center" });
      caption.anchor.set(0.5, 0);
      node.addChild(caption);

      const totalText = makeText(totalLine(effect), { fill: 0xd9c46a, fontSize: 20, fontWeight: "800", align: "center" });
      totalText.anchor.set(0.5, 0);
      totalText.position.set(0, 22);
      node.addChild(totalText);

      const pinButton = buildActionButton();
      pinButton.position.set(0, 48);
      node.addChild(pinButton);
      node.__pinButton = pinButton;

      const width = Math.max(caption.width, totalText.width, 160);
      placeAtTransientSlot(node, width);
      return node;
    }

    // --- world-space dice cluster (spec 1.3-1.7) ------------------------

    function buildClusterNode() {
      const node = new PIXI.Container();
      return node;
    }

    function animateDie(entry, dieEntry, effect) {
      if (destroyed || current !== entry) return;
      const totalTicks = Math.max(1, Math.round(ROLL_PHASE_MS / ROLL_TICK_MS));
      let tick = 0;
      const node = dieEntry.node;
      node.alpha = 1;
      const tickOnce = () => {
        if (destroyed || current !== entry) return;
        tick += 1;
        const t = clampNumber(tick / totalTicks, 0, 1);
        const eased = t * t * (3 - 2 * t);
        node.position.set(
          dieEntry.start.x + (dieEntry.final.x - dieEntry.start.x) * eased,
          dieEntry.start.y + (dieEntry.final.y - dieEntry.start.y) * eased,
        );
        node.rotation = (Math.random() - 0.5) * 0.6 * (1 - eased);
        node.scale.set(clampNumber(0.4 + eased * 0.6, 0.4, 1));
        node.__text.text = String(randomFace(dieEntry.die));
        if (tick < totalTicks) {
          entry.timers.push(setTimeoutFn(tickOnce, ROLL_TICK_MS));
        } else {
          node.position.set(dieEntry.final.x, dieEntry.final.y);
          node.rotation = 0;
          node.scale.set(1);
          node.__text.text = String(finalFace(dieEntry.die));
          dieEntry.settled = true;
          entry.settledCount += 1;
          if (entry.settledCount === entry.dieEntries.length) {
            onAllDiceSettled(entry, effect);
          }
        }
      };
      entry.timers.push(setTimeoutFn(tickOnce, ROLL_TICK_MS));
    }

    // onAllDiceSettled is the sequencing barrier required by spec 6.2 /
    // 7: the announcement must never enter while any die is still moving.
    function onAllDiceSettled(entry, effect) {
      if (destroyed || current !== entry) return;
      const node = buildAnnouncementNode(effect);
      hudContainer.addChild(node);
      entry.announcementNode = node;
      setActionButton(node.__pinButton, "Pin", () => pin(effect.id));
      onEffectChange({ kind: "settled", effect });

      const holdTimer = setTimeoutFn(() => {
        if (destroyed) return;
        if (entry.pinnedMidFlight) {
          settleIntoPinned(entry);
          return;
        }
        fadeOutEntry(entry, () => {
          if (current === entry) current = null;
          onEffectChange({ kind: "expired", effect });
          startNext();
        });
      }, Math.max(500, effect.durationMs));
      entry.timers.push(holdTimer);
    }

    function fadeNode(node, totalSteps, onDone) {
      if (!PIXI || !node) {
        onDone?.();
        return;
      }
      const startAlpha = node.alpha ?? 1;
      let step = 0;
      const tick = () => {
        if (destroyed) return;
        step += 1;
        const t = clampNumber(step / totalSteps, 0, 1);
        node.alpha = startAlpha * (1 - t);
        if (t >= 1) {
          node.destroy({ children: true });
          onDone?.();
          return;
        }
        setTimeoutFn(tick, 30);
      };
      setTimeoutFn(tick, 30);
    }

    // fadeOutEntry fades the dice cluster and the announcement together --
    // used only for the fully-transient path (never pinned).
    function fadeOutEntry(entry, onDone) {
      const totalSteps = Math.max(1, Math.round(FADE_MS / 30));
      let pending = 0;
      let done = false;
      const complete = () => {
        if (done) return;
        pending -= 1;
        if (pending <= 0) {
          done = true;
          onDone?.();
        }
      };
      if (entry.clusterNode) {
        pending += 1;
        fadeNode(entry.clusterNode, totalSteps, complete);
      }
      if (entry.announcementNode) {
        pending += 1;
        fadeNode(entry.announcementNode, totalSteps, complete);
      }
      if (entry.legacyNode) {
        pending += 1;
        fadeNode(entry.legacyNode, totalSteps, complete);
      }
      if (pending === 0) onDone?.();
    }

    // settleIntoPinned keeps the dice cluster in place permanently (spec
    // 1.9: static dice remain, exact map-relative coordinates, until
    // explicitly cleared). Live testing (2026-08-28): the original design
    // faded the announcement's "Pin" button away and handed dismiss
    // authority to a separate control down on the dice cluster -- clicking
    // Pin and having its replacement appear somewhere else on screen read
    // as broken, not intentional. The same banner node now stays put and
    // its own button just relabels Pin -> Dismiss in place. Known
    // limitation, not solved here: every transient banner shares one fixed
    // HUD slot (placeAtTransientSlot), so pinning more than one roll at a
    // time will stack their banners on top of each other -- narrower
    // problem than the one reported, left for if it actually comes up.
    function settleIntoPinned(entry) {
      clearTimers(entry);
      if (current === entry) current = null;

      if (entry.legacy) {
        setActionButton(entry.legacyNode.__actionButton, "Dismiss", () => dismiss(entry.effect.id));
        pinned.set(entry.effect.id, { effect: { ...entry.effect, pinned: true }, clusterNode: entry.legacyNode });
        onEffectChange({ kind: "pinned", effect: { ...entry.effect, pinned: true } });
        startNext();
        return;
      }

      if (entry.announcementNode) {
        setActionButton(entry.announcementNode.__pinButton, "Dismiss", () => dismiss(entry.effect.id));
      }
      pinned.set(entry.effect.id, {
        effect: { ...entry.effect, pinned: true },
        clusterNode: entry.clusterNode,
        announcementNode: entry.announcementNode,
      });
      onEffectChange({ kind: "pinned", effect: { ...entry.effect, pinned: true } });
      startNext();
    }

    // --- legacy (no active map) fallback ---------------------------------
    // Preserves the original Kernel 86 grouped-HUD presentation for the
    // rare case where there is no map surface to land dice on.

    function buildLegacyGroupedNode(effect, tokenSize, settledImmediately) {
      const node = new PIXI.Container();
      const shownDice = effect.dice.slice(0, MAX_DICE_SHOWN);
      const gap = Math.round(tokenSize * 0.18);
      const poolWidth = shownDice.length > 0 ? shownDice.length * tokenSize + (shownDice.length - 1) * gap : tokenSize;

      const dieNodes = shownDice.map((die, i) => {
        const dieNode = buildDieWorldNode(tokenSize, settledImmediately ? finalFace(die) : randomFace(die));
        dieNode.position.set(-poolWidth / 2 + tokenSize / 2 + i * (tokenSize + gap), 0);
        dieNode.scale.set(settledImmediately ? 1 : 0.4);
        dieNode.alpha = settledImmediately ? 1 : 0;
        node.addChild(dieNode);
        return dieNode;
      });

      const caption = makeText(captionLine(effect), { fill: 0xf3ecd8, fontSize: 15, fontWeight: "600", align: "center" });
      caption.anchor.set(0.5, 0);
      caption.position.set(0, tokenSize / 2 + 10);
      caption.alpha = settledImmediately ? 1 : 0;
      node.addChild(caption);

      const totalText = makeText(settledImmediately ? totalLine(effect) : "", { fill: 0xd9c46a, fontSize: 20, fontWeight: "800", align: "center" });
      totalText.anchor.set(0.5, 0);
      totalText.position.set(0, tokenSize / 2 + 34);
      totalText.alpha = settledImmediately ? 1 : 0;
      node.addChild(totalText);

      node.__dieNodes = dieNodes;
      node.__caption = caption;
      node.__totalText = totalText;
      node.__poolWidth = Math.max(poolWidth, caption.width, 160);
      node.__actionButton = buildActionButton();
      node.__actionButton.position.set(0, tokenSize / 2 + 58);
      node.addChild(node.__actionButton);
      return node;
    }

    function animateLegacyRoll(node, effect, tokenSize, onSettled) {
      const timers = [];
      for (const dieNode of node.__dieNodes) dieNode.alpha = 1;
      const totalTicks = Math.max(1, Math.round(ROLL_PHASE_MS / ROLL_TICK_MS));
      let tick = 0;
      const tickOnce = () => {
        if (destroyed) return;
        tick += 1;
        node.__dieNodes.forEach((dieNode, i) => {
          const die = effect.dice[i];
          if (!die) return;
          dieNode.__text.text = String(randomFace(die));
          dieNode.rotation = (Math.random() - 0.5) * 0.6;
          dieNode.scale.set(clampNumber(0.4 + tick / totalTicks, 0.4, 1));
        });
        if (tick < totalTicks) {
          timers.push(setTimeoutFn(tickOnce, ROLL_TICK_MS));
        } else {
          node.__dieNodes.forEach((dieNode, i) => {
            const die = effect.dice[i];
            if (!die) return;
            dieNode.__text.text = String(finalFace(die));
            dieNode.rotation = 0;
            dieNode.scale.set(1);
          });
          node.__caption.alpha = 1;
          node.__totalText.alpha = 1;
          node.__totalText.text = totalLine(effect);
          onSettled(timers);
        }
      };
      timers.push(setTimeoutFn(tickOnce, ROLL_TICK_MS));
      return timers;
    }

    function startLegacyHudOnly(effect, tokenSize) {
      const node = buildLegacyGroupedNode(effect, tokenSize, false);
      node.alpha = 1;
      placeAtTransientSlot(node, node.__poolWidth || 200);
      hudContainer.addChild(node);

      const entry = { effect, legacyNode: node, timers: [], pinnedMidFlight: false, legacy: true };
      current = entry;
      onEffectChange({ kind: "entered", effect });

      entry.timers.push(...animateLegacyRoll(node, effect, tokenSize, (rollTimers) => {
        entry.timers.push(...rollTimers);
        onEffectChange({ kind: "settled", effect });
        setActionButton(node.__actionButton, "Pin", () => pin(effect.id));

        const holdTimer = setTimeoutFn(() => {
          if (destroyed) return;
          if (entry.pinnedMidFlight) {
            settleIntoPinned(entry);
            return;
          }
          fadeOutEntry(entry, () => {
            if (current === entry) current = null;
            onEffectChange({ kind: "expired", effect });
            startNext();
          });
        }, Math.max(500, effect.durationMs));
        entry.timers.push(holdTimer);
      }));
    }

    // --- queue / lifecycle entry points ---------------------------------

    function startNext() {
      if (current || destroyed) return;
      const effect = queue.shift();
      if (!effect) return;

      // By the time any roll happens, the map has almost always finished
      // loading -- a natural, frequent point to upgrade any pinned effect
      // that had to fall back to a placeholder/legacy placement earlier
      // (see reconcilePinnedPlacements).
      reconcilePinnedPlacements();

      if (!worldContainer || !hudContainer || !PIXI) {
        onEffectChange({ kind: "settled", effect });
        startNext();
        return;
      }

      const tokenSize = clampNumber(getDefaultTokenSize(), 32, 200);
      const mapBounds = normalizeMapBounds(getMapWorldBounds());

      if (!mapBounds) {
        startLegacyHudOnly(effect, tokenSize);
        return;
      }

      const shown = effect.dice.slice(0, MAX_DICE_SHOWN);
      const positions = computeLandingPositions(effect.id, shown.length, mapBounds, tokenSize);

      const clusterNode = buildClusterNode();
      worldContainer.addChild(clusterNode);

      const dieEntries = shown.map((die, i) => {
        const node = buildDieWorldNode(tokenSize, randomFace(die));
        const final = positions[i];
        const offset = entryOffsetFor(effect.id, i);
        const start = { x: final.x + offset.dx, y: final.y + offset.dy };
        node.position.set(start.x, start.y);
        node.scale.set(0.4);
        node.alpha = 0;
        clusterNode.addChild(node);
        return { die, node, start, final, settled: false };
      });
      clusterNode.__dieNodes = dieEntries.map((d) => d.node);

      const dismissButton = buildActionButton();
      const center = centroid(positions);
      dismissButton.position.set(center.x, center.y + tokenSize / 2 + 26);
      clusterNode.addChild(dismissButton);
      clusterNode.__dismissButton = dismissButton;

      const entry = {
        effect,
        clusterNode,
        dieEntries,
        dismissButton,
        announcementNode: null,
        timers: [],
        settledCount: 0,
        pinnedMidFlight: false,
      };
      current = entry;
      onEffectChange({ kind: "entered", effect });

      dieEntries.forEach((dieEntry, i) => {
        entry.timers.push(setTimeoutFn(() => animateDie(entry, dieEntry, effect), i * STAGGER_MS));
      });
    }

    function enqueue(rawEffect) {
      const effect = normalizeEffect(rawEffect);
      if (!effect || !effect.id) return;
      if (pinned.has(effect.id)) return; // already-pinned effects don't re-enter the transient queue
      if (current && current.effect.id === effect.id) return;
      if (queue.some((e) => e.id === effect.id)) return;

      if (queue.length >= MAX_QUEUE_LENGTH) {
        queue.shift(); // drop the oldest still-queued roll rather than grow unbounded (kernel 86 §16)
      }
      queue.push(effect);
      startNext();
    }

    function applyPinned(rawEffect) {
      const effect = normalizeEffect(rawEffect);
      if (!effect || !effect.id) return;

      if (current && current.effect.id === effect.id) {
        current.pinnedMidFlight = true;
        return;
      }
      queue = queue.filter((e) => e.id !== effect.id);

      if (!worldContainer || !hudContainer || !PIXI) {
        // Containers don't exist yet -- mount() hasn't run. Record a
        // placeholder flagged for a real rebuild the moment containers
        // (and hopefully real map bounds) exist; see reconcilePinnedPlacements.
        pinned.set(effect.id, { effect, clusterNode: null, needsSpatialUpgrade: true });
        onEffectChange({ kind: "pinned", effect });
        return;
      }

      const tokenSize = clampNumber(getDefaultTokenSize(), 32, 200);
      const mapBounds = normalizeMapBounds(getMapWorldBounds());

      if (!mapBounds) {
        // No active map bounds *right now*. This is the deliberate,
        // permanent fallback for a genuinely map-less venue (spec's own
        // no-map contract) -- but it's indistinguishable, at this instant,
        // from "the map hasn't finished loading yet" (stage_effects/pinned
        // hydration races the map texture/bounds computation on connect,
        // same root cause as the containers-not-ready case above). Flag it
        // for a cheap idempotent re-check rather than assuming permanence:
        // if this really is a map-less venue, the re-check just rebuilds
        // the identical legacy node again; if the map was simply still
        // loading, it upgrades to real spatial placement once bounds exist.
        const node = buildLegacyGroupedNode(effect, tokenSize, true);
        placeAtTransientSlot(node, node.__poolWidth || 200);
        setActionButton(node.__actionButton, "Dismiss", () => dismiss(effect.id));
        hudContainer.addChild(node);
        pinned.set(effect.id, { effect, clusterNode: node, needsSpatialUpgrade: true });
        onEffectChange({ kind: "pinned", effect });
        return;
      }

      const shown = effect.dice.slice(0, MAX_DICE_SHOWN);
      const positions = computeLandingPositions(effect.id, shown.length, mapBounds, tokenSize);

      const clusterNode = buildClusterNode();
      const dieNodes = shown.map((die, i) => {
        const node = buildDieWorldNode(tokenSize, finalFace(die));
        node.position.set(positions[i].x, positions[i].y);
        clusterNode.addChild(node);
        return node;
      });
      clusterNode.__dieNodes = dieNodes;
      const dismissButton = buildActionButton();
      const center = centroid(positions);
      dismissButton.position.set(center.x, center.y + tokenSize / 2 + 26);
      setActionButton(dismissButton, "Dismiss", () => dismiss(effect.id));
      clusterNode.addChild(dismissButton);
      clusterNode.__dismissButton = dismissButton;
      worldContainer.addChild(clusterNode);

      pinned.set(effect.id, { effect, clusterNode, positions });
      onEffectChange({ kind: "pinned", effect });
    }

    // reconcilePinnedPlacements re-attempts applyPinned for any pinned
    // entry that was built without real containers/map bounds available
    // (see the two flagged branches in applyPinned above). Cheap and
    // idempotent -- if the underlying condition still holds it just
    // rebuilds the same placeholder/legacy representation again. Called
    // from mount() (covers hydration racing initial scene construction)
    // and at the start of startNext() (covers hydration racing the map
    // texture load specifically -- by the time any roll happens, the map
    // has almost always finished loading).
    function reconcilePinnedPlacements() {
      if (!worldContainer || !hudContainer || !PIXI) return;
      for (const entry of Array.from(pinned.values())) {
        if (entry.needsSpatialUpgrade) {
          applyPinned(entry.effect.raw || entry.effect);
        }
      }
    }

    function removePinned(effectId) {
      const id = String(effectId || "");
      const entry = pinned.get(id);
      if (!entry) return;
      pinned.delete(id);
      entry.clusterNode?.destroy?.({ children: true });
      entry.announcementNode?.destroy?.({ children: true });
      onEffectChange({ kind: "dismissed", effectId: id });
    }

    function hydratePinned(effects) {
      for (const id of Array.from(pinned.keys())) {
        removePinned(id);
      }
      for (const raw of Array.isArray(effects) ? effects : []) {
        applyPinned(raw);
      }
    }

    function pin(effectId) {
      return sendAction("stage_effect/pin", { effect_id: String(effectId || "") });
    }

    function dismiss(effectId) {
      return sendAction("stage_effect/dismiss", { effect_id: String(effectId || "") });
    }

    function getQueueLength() {
      return queue.length;
    }

    function getPinnedIds() {
      return Array.from(pinned.keys());
    }

    // getDebugState is a read-only introspection hook for browser-proof
    // tooling (spec 14) -- there is no way to observe individual landed
    // die coordinates, the sequencing barrier, or world/hud separation
    // from outside the module otherwise. Not used by production code.
    function getDebugState() {
      const worldDiceCount = worldContainer
        ? worldContainer.children.reduce((sum, c) => sum + (c.__dieNodes ? c.__dieNodes.length : 0), 0)
        : 0;
      // x/y are the die's own LOCAL (world-space) coordinates -- these are
      // deliberately constant regardless of camera pan/zoom (that's what
      // "map-relative" means: computeLandingPositions never changes once
      // set). screenX/screenY are the RENDERED global position (through
      // every ancestor transform, including stageCamera's worldLayer
      // pan/scale) -- use these, not x/y, to prove a die visibly moved
      // on screen when the camera moved.
      const globalOf = (n) => (typeof n.getGlobalPosition === "function" ? n.getGlobalPosition() : { x: n.x, y: n.y });
      return {
        queueLength: queue.length,
        pinnedIds: getPinnedIds(),
        current: current ? {
          effectId: current.effect.id,
          legacy: Boolean(current.legacy),
          settledCount: current.legacy ? null : current.settledCount,
          diceCount: current.legacy ? (current.legacyNode?.__dieNodes?.length || 0) : (current.dieEntries?.length || 0),
          hasAnnouncement: current.legacy ? Boolean(current.legacyNode?.__caption?.alpha) : Boolean(current.announcementNode),
          dicePositions: current.legacy ? [] : (current.dieEntries || []).map((d) => {
            const g = globalOf(d.node);
            return { x: d.node.x, y: d.node.y, screenX: g.x, screenY: g.y, settled: d.settled, text: d.node.__text.text };
          }),
        } : null,
        worldDiceCount,
        pinnedDice: Array.from(pinned.entries()).map(([id, entry]) => ({
          id,
          positions: (entry.clusterNode?.__dieNodes || []).map((n) => {
            const g = globalOf(n);
            return { x: n.x, y: n.y, screenX: g.x, screenY: g.y, text: n.__text?.text };
          }),
        })),
      };
    }

    function destroy() {
      if (destroyed) return;
      destroyed = true;
      clearTimers(current);
      current = null;
      queue = [];
      for (const entry of pinned.values()) {
        entry.clusterNode?.destroy?.({ children: true });
      }
      pinned.clear();
      if (hudContainer) {
        hudContainer.destroy({ children: true });
        hudContainer = null;
      }
      if (worldContainer) {
        worldContainer.destroy({ children: true });
        worldContainer = null;
      }
    }

    return {
      mount,
      enqueue,
      applyPinned,
      removePinned,
      hydratePinned,
      pin,
      dismiss,
      getQueueLength,
      getPinnedIds,
      getDebugState,
      destroy,
    };
  }

  return {
    createDiceProjectionController,
    normalizeEffect,
    // exported for tests: deterministic placement is a documented,
    // independently-testable contract (spec 11).
    computeLandingPositions,
    normalizeMapBounds,
  };
});
