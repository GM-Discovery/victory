(function (root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  }
  root.VictoryStageState = factory();
})(typeof globalThis !== "undefined" ? globalThis : window, function () {
  function clonePlain(value) {
    if (value === null || value === undefined) return value;
    if (Array.isArray(value) || typeof value === "object") {
      return JSON.parse(JSON.stringify(value));
    }
    return value;
  }

  function clampNumber(value, min, max, fallback) {
    const parsed = Number(value);
    if (!Number.isFinite(parsed)) return fallback;
    return Math.max(min, Math.min(max, parsed));
  }

  function normalizeVisibility(value) {
    const source = value && typeof value === "object" ? value : {};
    const visible = source.visible ?? source.is_visible ?? true;
    const nameplateVisible = source.nameplate_visible ?? source.nameplateVisible ?? true;
    return {
      ...clonePlain(source),
      visible: Boolean(visible),
      nameplate_visible: Boolean(nameplateVisible),
      nameplateVisible: Boolean(nameplateVisible),
    };
  }

  function normalizeState(value) {
    const source = value && typeof value === "object" ? value : {};
    const state = {
      ...clonePlain(source),
    };
    if (state.nameplateVisible === undefined && state.nameplate_visible !== undefined) {
      state.nameplateVisible = Boolean(state.nameplate_visible);
    }
    if (state.nameplate_visible === undefined && state.nameplateVisible !== undefined) {
      state.nameplate_visible = Boolean(state.nameplateVisible);
    }
    return state;
  }

  function readFirst(value, keys) {
    for (const key of keys) {
      const candidate = value && typeof value === "object" ? value[key] : undefined;
      const text = String(candidate ?? "").trim();
      if (text) return text;
    }
    return "";
  }

  function normalizePosition(value) {
    const source = value && typeof value === "object" ? value : {};
    return {
      ...clonePlain(source),
      x: Number.isFinite(Number(source.x)) ? Number(source.x) : 0,
      y: Number.isFinite(Number(source.y)) ? Number(source.y) : 0,
      order: Number.isFinite(Number(source.order)) ? Math.round(Number(source.order)) : 0,
    };
  }

  function makeKey(element) {
    return readFirst(element, [
      "element_id",
      "elementId",
      "slug",
      "element_slug",
      "elementSlug",
      "key",
      "id",
    ]);
  }

  // Kernel 73A: a "scene_composition" context_class marks a synthetic
  // element backend/internal/world/snapshot.go merged in from a Scene's
  // own scene_stage_elements (Base + Show layer authored composition),
  // not a warehouse-asset-backed element. Its data.kind is the authored
  // composition kind (token/index_card/map_backdrop/grid_config), which
  // this maps onto the SAME "token"/"card" node kinds the rest of this
  // pipeline already knows how to render -- reusing the real PIXI token/
  // card node factories (scene-nodes.js) rather than a bolt-on renderer,
  // so a Scene's tokens and index cards appear on the actual Catharsis
  // stage exactly like any other live object. map_backdrop/grid_config
  // have no equivalent generic per-element node in this pipeline (the
  // venue map/backdrop and grid are rendered as a separate background
  // layer, not a per-element node) -- composition rows of those two kinds
  // are intentionally not converted into stage objects here yet, not
  // silently mis-rendered as a token/card. Kernel 74 reads map_backdrop
  // server-side instead (world/kernel74_local_projection.go), which is why
  // they still produce no node here.
  //
  // Kernel 74 adds "interaction_hotspot": a positioned, resizable region
  // aligned over artwork that already exists in the map (the Courtyard's
  // door). It gets its own node kind rather than reusing "token" precisely
  // because it must NOT draw token art over the door it sits on -- the
  // whole point is that the door is already painted into the map.
  function sceneCompositionNodeKind(dataKind) {
    const kind = String(dataKind || "").trim().toLowerCase();
    if (kind === "token") return "token";
    if (kind === "index_card") return "card";
    if (kind === "interaction_hotspot") return "hotspot";
    return ""; // map_backdrop, grid_config, or unknown -- no node yet.
  }

  function kindForElement(element) {
    const source = element && typeof element === "object" ? element : {};
    const data = source.data && typeof source.data === "object" ? source.data : {};
    if (String(source.context_class || "").trim().toLowerCase() === "scene_composition") {
      return sceneCompositionNodeKind(data.kind);
    }
    const kind = String(source.kind || source.element_type || data.type || "").trim().toLowerCase();
    if (kind === "token") return "token";
    if (kind === "card" || kind === "index_card") return "card";
    if (kind === "fire") return "fire";
    if (kind === "prop") return "prop";
    if (kind === "stage") return "stage";
    if (kind === "map") return "map";
    return kind || "unknown";
  }

  function resolveAssetURL(data, fallbackKind) {
    const contentURL = readFirst(data, ["asset_content_url", "assetContentURL"]);
    if (contentURL) return contentURL;
    const thumbnailURL = readFirst(data, ["asset_thumbnail_url", "assetThumbnailURL"]);
    if (thumbnailURL) return thumbnailURL;
    if (fallbackKind === "token") return "/assets/construction.png";
    return "";
  }

  function normalizeElement(element, options = {}) {
    const source = element && typeof element === "object" ? element : {};
    const data = source.data && typeof source.data === "object" ? source.data : {};
    const visibility = normalizeVisibility(source.visibility);
    const state = normalizeState(source.state);
    const isSceneComposition = String(source.context_class || "").trim().toLowerCase() === "scene_composition";
    const kind = kindForElement(source);
    if (isSceneComposition && !kind) {
      // map_backdrop / grid_config / an unrecognized composition kind --
      // no stage-object node exists for these yet; drop rather than
      // mis-render as a token or card (see sceneCompositionNodeKind).
      return null;
    }
    const key = makeKey(source);
    const role = String(options.viewerRole || options.role || "").trim().toLowerCase();
    const tokenLayer = String(
      data.token_layer ??
      data.tokenLayer ??
      source.token_layer ??
      source.tokenLayer ??
      ""
    ).trim().toLowerCase();
    if (kind === "token" && role === "audience" && tokenLayer === "director") {
      return null;
    }
    const label = readFirst(source, [
      "label",
      "name",
      "display_name",
      "asset_name",
    ]) || "Token";

    const assetID = readFirst(data, ["asset_id", "assetID"]);
    const assetName = readFirst(data, ["asset_name", "assetName"]) || label;
    const position = normalizePosition(source.position);
    const snapshotRevision = Number(source.revision ?? source.moment_id ?? data.revision ?? 0);

    return {
      key,
      kind,
      // Kernel 73A composition elements are never live/1persistable stage
      // objects -- they have no elements-table row for update/token or
      // act/place_element to target, so drag/scale/lock actions must not
      // attempt to send a persistence request (makeTokenNode/makeCardNode's
      // own `if (!model.live) return;` guard after selectObject already
      // covers this: selection and the bound "Talk to X" action below both
      // still work with live=false, only drag-persistence is suppressed).
      live: isSceneComposition ? false : Boolean(source.live ?? true),
      elementId: readFirst(source, ["element_id", "elementId"]),
      elementSlug: readFirst(source, ["slug", "element_slug", "elementSlug"]),
      label,
      displayName: readFirst(source, ["display_name", "displayName"]) || label,
      assetID,
      assetName,
      assetShape: readFirst(data, ["asset_shape", "assetShape"]),
      assetContentURL: resolveAssetURL(data, kind),
      assetThumbnailURL: readFirst(data, ["asset_thumbnail_url", "assetThumbnailURL"]),
      defaultGridWidth: clampNumber(data.default_grid_width ?? data.defaultGridWidth ?? 1, 1, 64, 1),
      defaultGridHeight: clampNumber(data.default_grid_height ?? data.defaultGridHeight ?? 1, 1, 64, 1),
      retainOriginal: Boolean(data.retain_original ?? data.retainOriginal ?? false),
      snapMode: String(data.snap_mode ?? data.snapMode ?? "").trim().toLowerCase(),
      tokenLayer: tokenLayer === "director" ? "director" : "public",
      scale: clampNumber(data.scale ?? 100, 25, 500, 100),
      // Kernel 74 normalized 0-1 size. Only interaction_hotspot authors it
      // today; every other kind leaves these null and keeps its intrinsic
      // size. The defaults are a visible-but-modest box so a hotspot whose
      // width/height never got authored is still findable and draggable in
      // Scene Setup rather than being a zero-area invisible target.
      hotspotWidth: kind === "hotspot" ? clampNumber(data.width ?? 0.1, 0.01, 1, 0.1) : null,
      hotspotHeight: kind === "hotspot" ? clampNumber(data.height ?? 0.1, 0.01, 1, 0.1) : null,
      nameplateVisible: data.nameplate_visible !== false,
      position,
      state,
      visibility,
      source: {
        ...clonePlain(source),
        name: label,
        data: clonePlain(data),
        state,
        visibility,
      },
      revision: Number.isFinite(snapshotRevision) ? snapshotRevision : 0,
      deleted: Boolean(source.deleted ?? false),
    };
  }

  function normalizeSnapshot(snapshot, options = {}) {
    const source = snapshot && typeof snapshot === "object" ? snapshot : {};
    const elements = Array.isArray(source.elements) ? source.elements : [];
    const normalized = [];
    for (const element of elements) {
      const model = normalizeElement(element, options);
      if (model) normalized.push(model);
    }
    return {
      ...clonePlain(source),
      elements: normalized,
    };
  }

  function actionSequence(action) {
    const candidate = [
      action?.sequence,
      action?.moment_id,
      action?.momentId,
      action?.payload?.moment_id,
      action?.payload?.momentId,
      action?.ts,
    ].find((value) => value !== undefined && value !== null && value !== "");
    const parsed = Number(candidate);
    return Number.isFinite(parsed) ? parsed : 0;
  }

  function actionKey(action) {
    const target = action?.target && typeof action.target === "object" ? action.target : {};
    const payload = action?.payload && typeof action.payload === "object" ? action.payload : {};
    return readFirst(target, [
      "element_id",
      "elementId",
      "slug",
      "element_slug",
      "elementSlug",
      "id",
      "asset_id",
      "assetId",
    ]) || readFirst(payload, [
      "element_id",
      "elementId",
      "slug",
      "element_slug",
      "elementSlug",
      "id",
      "asset_id",
      "assetId",
    ]);
  }

  function actionKind(action) {
    const type = String(action?.type || "").trim().toLowerCase();
    if (type === "create/token" || type === "update/token" || type === "duplicate/token") return "token";
    if (type === "create/index_card" || type === "update/index_card" || type === "delete/index_card") return "card";
    if (type.includes("element")) return "element";
    return "unknown";
  }

  function mergeVisibility(previous, next) {
    return normalizeVisibility({
      ...(previous || {}),
      ...(next || {}),
    });
  }

  function mergeState(previous, next) {
    return normalizeState({
      ...(previous || {}),
      ...(next || {}),
    });
  }

  function createProjectedState(options = {}) {
    let viewerRole = String(options.viewerRole || "audience").trim().toLowerCase();
    let state = {
      snapshot: null,
      objects: [],
      objectsByKey: new Map(),
      version: 0,
      lastEvent: null,
      viewerRole,
    };
    const subscribers = new Set();

    function emit(changedTopics, payload = {}) {
      const topics = Array.isArray(changedTopics) ? changedTopics : [changedTopics];
      for (const subscriber of subscribers) {
        try {
          if (subscriber.predicate(state, payload, topics)) {
            subscriber.callback(state, payload);
          }
        } catch (error) {
          console.warn("stage state subscriber failed", error);
        }
      }
    }

    function upsertObject(model, options = {}) {
      if (!model || !model.key) return null;
      const existing = state.objectsByKey.get(model.key) || null;
      const nextRevision = Number.isFinite(Number(options.sequence)) ? Number(options.sequence) : Number(model.revision || 0);
      if (existing && existing.deleted && options.allowResurrection !== true) {
        return null;
      }
      if (existing && Number(existing.revision || 0) > nextRevision) {
        return null;
      }
      const merged = existing ? {
        ...existing,
        ...model,
        position: normalizePosition({
          ...(existing.position || {}),
          ...(model.position || {}),
          x: model.position && model.position.x !== undefined ? model.position.x : existing.position?.x,
          y: model.position && model.position.y !== undefined ? model.position.y : existing.position?.y,
          order: model.position && model.position.order !== undefined ? model.position.order : existing.position?.order,
        }),
        state: mergeState(existing.state, model.state),
        visibility: mergeVisibility(existing.visibility, model.visibility),
        source: {
          ...(existing.source || {}),
          ...(model.source || {}),
          data: {
            ...((existing.source && existing.source.data) || {}),
            ...((model.source && model.source.data) || {}),
          },
          state: mergeState(existing.source?.state, model.source?.state || model.state),
          visibility: mergeVisibility(existing.source?.visibility, model.source?.visibility || model.visibility),
        },
        deleted: false,
        revision: nextRevision,
      } : {
        ...model,
        position: normalizePosition(model.position),
        state: mergeState(model.state, model.state),
        visibility: mergeVisibility(model.visibility, model.visibility),
        source: {
          ...(model.source || {}),
          data: { ...((model.source && model.source.data) || {}) },
          state: mergeState(model.source?.state, model.state),
          visibility: mergeVisibility(model.source?.visibility, model.visibility),
        },
        deleted: false,
        revision: nextRevision,
      };

      if (!merged.assetContentURL && merged.kind === "token") {
        merged.assetContentURL = "/assets/construction.png";
      }

      state.objectsByKey.set(model.key, merged);
      state.objects = state.objects.filter((item) => item.key !== model.key);
      state.objects.push(merged);
      state.version += 1;
      state.lastEvent = options.event || null;
      emit([merged.kind, merged.key, "objects"], { changedKey: merged.key, kind: merged.kind, object: merged, event: options.event || null });
      return merged;
    }

    function removeObject(action, options = {}) {
      const key = actionKey(action);
      if (!key) return null;
      const existing = state.objectsByKey.get(key) || null;
      if (!existing) return null;
      const seq = Number.isFinite(Number(options.sequence)) ? Number(options.sequence) : actionSequence(action);
      if (existing.revision && Number(existing.revision) > seq) return null;
      existing.deleted = true;
      existing.revision = Math.max(Number(existing.revision || 0), seq);
      state.objects = state.objects.filter((item) => item.key !== key);
      state.version += 1;
      state.lastEvent = options.event || null;
      emit([existing.kind, key, "objects"], { changedKey: key, kind: existing.kind, deleted: true, event: options.event || null });
      return existing;
    }

    function applyAction(action, options = {}) {
      const type = String(action?.type || "").trim().toLowerCase();
      const sequence = actionSequence(action);
      const key = actionKey(action);
      const payload = action?.payload && typeof action.payload === "object" ? action.payload : {};
      const target = action?.target && typeof action.target === "object" ? action.target : {};
      if (type === "remove_element" || type === "act/remove_element" || type === "delete/index_card") {
        return removeObject(action, { ...options, sequence });
      }
      if (type === "create/token" || type === "update/token" || type === "create/index_card" || type === "update/index_card" || type === "duplicate") {
        const existing = key ? state.objectsByKey.get(key) : null;
        if (existing && existing.deleted) {
          return null;
        }
        const base = normalizeElement({
          element_id: key || readFirst(target, ["element_id", "elementId"]) || readFirst(payload, ["element_id", "elementId"]),
          element_slug: readFirst(target, ["element_slug", "elementSlug"]) || readFirst(payload, ["element_slug", "elementSlug"]),
          name: readFirst(payload, ["asset_name", "name", "label"]),
          display_name: readFirst(payload, ["display_name", "displayName"]),
          element_type: type.includes("token") ? "token" : "card",
          data: {
            ...(existing?.source?.data || {}),
            ...payload,
          },
          position: {
            ...(existing?.position || {}),
            ...(payload.position || {}),
            x: payload.x ?? existing?.position?.x,
            y: payload.y ?? existing?.position?.y,
            order: payload.order ?? existing?.position?.order ?? 0,
          },
          state: {
            ...(existing?.state || {}),
            ...(payload.state || {}),
          },
          visibility: {
            ...(existing?.visibility || {}),
            ...(payload.visibility || {}),
            nameplate_visible:
              payload.nameplate_visible ?? payload.nameplateVisible ?? existing?.visibility?.nameplate_visible ?? existing?.visibility?.nameplateVisible ?? true,
            visible:
              payload.visible ?? existing?.visibility?.visible ?? true,
          },
          live: existing?.live ?? true,
          revision: sequence,
        }, { viewerRole });
        if (!base) return null;
        if (existing && existing.deleted) return null;
        if (existing && Number(existing.revision || 0) > sequence) return null;
        const next = upsertObject(base, { sequence, event: action });
        if (next && base.kind === "token" && !next.assetContentURL) {
          next.assetContentURL = "/assets/construction.png";
          next.source.data.asset_content_url = next.assetContentURL;
        }
        if (next && next.kind === "card") {
          // normalizeElement has no frontText/backText/color fields at all
          // (only nested under source.data) and falls back label to the
          // literal string "Token" when payload carries no name -- fine for
          // the actor's own tab, which editors.js/runtime.js's
          // saveCardEditor patches model.frontText/backText directly on
          // save, but a receiving client applying this same broadcast (e.g.
          // Audience) only ever goes through this path and would otherwise
          // render stale (or "Token"-labeled) card content forever despite
          // source.data having the correct new text underneath.
          next.frontText = readFirst(payload, ["front_text", "frontText"]) ?? next.source?.data?.front_text ?? next.frontText ?? "";
          next.backText = readFirst(payload, ["back_text", "backText"]) ?? next.source?.data?.back_text ?? next.backText ?? "";
          next.color = readFirst(payload, ["color"]) ?? next.source?.data?.color ?? next.color;
          next.label = next.frontText || next.label;
          next.source.name = next.label;
          const nextFace = readFirst(payload, ["face"]);
          if (nextFace) {
            next.cardFace = String(nextFace).toLowerCase() === "back" ? "back" : "front";
            next.source.data.face = next.cardFace;
          }
        }
        return next;
      }
      if (type === "set_nameplate_visibility" || type === "act/set_nameplate_visibility") {
        const existing = key ? state.objectsByKey.get(key) : null;
        if (!existing || existing.deleted) return null;
        if (Number(existing.revision || 0) > sequence) return null;
        existing.visibility = mergeVisibility(existing.visibility, {
          nameplate_visible: payload.visible ?? payload.nameplate_visible ?? payload.nameplateVisible ?? true,
        });
        existing.source.visibility = mergeVisibility(existing.source.visibility, existing.visibility);
        existing.revision = sequence;
        state.version += 1;
        state.lastEvent = options.event || null;
        emit([existing.kind, key, "objects"], { changedKey: key, kind: existing.kind, object: existing, event: options.event || null });
        return existing;
      }
      if (type === "act/set_element_lock" || type === "set_element_lock") {
        const existing = key ? state.objectsByKey.get(key) : null;
        if (!existing || existing.deleted) return null;
        if (Number(existing.revision || 0) > sequence) return null;
        existing.state = mergeState(existing.state, { locked: Boolean(payload.locked) });
        existing.source.state = mergeState(existing.source.state, existing.state);
        existing.revision = sequence;
        state.version += 1;
        state.lastEvent = options.event || null;
        emit([existing.kind, key, "objects"], { changedKey: key, kind: existing.kind, object: existing, event: options.event || null });
        return existing;
      }
      if (type === "act/hide_element" || type === "hide_element" || type === "act/reveal_element" || type === "reveal_element") {
        const existing = key ? state.objectsByKey.get(key) : null;
        if (!existing || existing.deleted) return null;
        if (Number(existing.revision || 0) > sequence) return null;
        const visible = type === "act/reveal_element" || type === "reveal_element";
        existing.state = mergeState(existing.state, { visible });
        existing.visibility = mergeVisibility(existing.visibility, { visible });
        existing.source.state = mergeState(existing.source.state, existing.state);
        existing.source.visibility = mergeVisibility(existing.source.visibility, existing.visibility);
        existing.revision = sequence;
        state.version += 1;
        state.lastEvent = options.event || null;
        emit([existing.kind, key, "objects"], { changedKey: key, kind: existing.kind, object: existing, event: options.event || null });
        return existing;
      }
      return null;
    }

    function replaceFromSnapshot(snapshot, options = {}) {
      viewerRole = String(options.viewerRole || viewerRole || "audience").trim().toLowerCase();
      state.viewerRole = viewerRole;
      const normalized = normalizeSnapshot(snapshot, { viewerRole });
      const nextObjects = [];
      const nextObjectsByKey = new Map();
      for (const element of normalized.elements) {
        if (!element || !element.key) continue;
        element.revision = Number.isFinite(Number(element.revision)) ? Number(element.revision) : 0;
        element.deleted = false;
        nextObjects.push(element);
        nextObjectsByKey.set(element.key, element);
      }
      state = {
        ...state,
        snapshot: normalized,
        objects: nextObjects,
        objectsByKey: nextObjectsByKey,
        version: state.version + 1,
        lastEvent: options.event || null,
      };
      emit(["snapshot", "objects"], { snapshot: normalized, event: options.event || null });
      return state;
    }

    function applyEvent(event, options = {}) {
      const action = event && typeof event === "object" ? event : null;
      if (!action) return null;
      if (String(action.type || "").trim().toLowerCase() === "snapshot") {
        return replaceFromSnapshot(action.data || action.snapshot || action.payload || {}, { ...options, event: action });
      }
      return applyAction(action, { ...options, event: action });
    }

    function subscribe(selectorOrTopic, callback) {
      const topic = typeof selectorOrTopic === "string" ? selectorOrTopic.trim() : "";
      const predicate = typeof selectorOrTopic === "function"
        ? selectorOrTopic
        : (currentState, payload, topics) => {
            if (!topic) return true;
            return topics.includes(topic) || payload?.changedKey === topic || payload?.kind === topic;
          };
      const entry = {
        predicate,
        callback: typeof callback === "function" ? callback : () => {},
      };
      subscribers.add(entry);
      return () => {
        subscribers.delete(entry);
      };
    }

    function getState() {
      return state;
    }

    return {
      getState,
      replaceFromSnapshot,
      applyEvent,
      subscribe,
    };
  }

  return {
    createProjectedState,
    normalizeSnapshot,
    normalizeElement,
    normalizeVisibility,
    normalizeState,
  };
});
