(function (root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  }
  root.VictoryStageLogic = factory();
})(typeof globalThis !== "undefined" ? globalThis : window, function () {
  function clonePlain(value) {
    if (value === null || value === undefined) return value;
    if (Array.isArray(value) || typeof value === "object") {
      return JSON.parse(JSON.stringify(value));
    }
    return value;
  }

  function venueConfigFlag(config, key, fallback = false) {
    const source = config && typeof config === "object" ? config : {};
    const value = source[key];
    if (typeof value === "boolean") return value;
    if (typeof value === "string") {
      const lowered = value.trim().toLowerCase();
      if (["true", "1", "yes", "on"].includes(lowered)) return true;
      if (["false", "0", "no", "off"].includes(lowered)) return false;
    }
    return fallback;
  }

  function objectKind(model) {
    const kind = String(model?.kind || model?.contextClass || model?.elementType || "").trim().toLowerCase();
    if (kind === "token") return "token";
    const assetID = String(model?.assetID || model?.asset_id || model?.source?.data?.asset_id || "").trim();
    const assetURL = String(model?.assetContentURL || model?.assetThumbnailURL || model?.source?.data?.asset_content_url || model?.source?.data?.asset_thumbnail_url || "").trim();
    if (assetID || assetURL) return "token";
    if (kind === "index_card") return "card";
    return kind;
  }

  function objectState(model) {
    const source = model?.source || {};
    const state = model?.state || source?.state || {};
    const visibility = model?.visibility || source?.visibility || {};
    const nameplateVisible = state.nameplate_visible ?? state.nameplateVisible ?? visibility.nameplate_visible ?? visibility.nameplateVisible ?? true;
    return {
      locked: Boolean(state.locked ?? visibility.locked ?? false),
      nameplateVisible: Boolean(nameplateVisible),
      visible: Boolean(state.visible ?? visibility.visible ?? true),
    };
  }

  function updateObjectMatches(a, b) {
    if (!a || !b) return false;
    const aKey = String(a.key || "");
    const bKey = String(b.key || "");
    const aElementId = String(a.elementId || "");
    const bElementId = String(b.elementId || "");
    const aElementSlug = String(a.elementSlug || "");
    const bElementSlug = String(b.elementSlug || "");
    return (aKey && aKey === bKey) || (aElementId && aElementId === bElementId) || (aElementSlug && aElementSlug === bElementSlug);
  }

  function viewerCanSeeHiddenCards(role) {
    return String(role || "").trim().toLowerCase() !== "audience";
  }

  function isCardObject(model) {
    return objectKind(model) === "card";
  }

  function isTokenObject(model) {
    return objectKind(model) === "token";
  }

  function isLiveStageObject(model) {
    return !!model?.live && ["card", "prop", "fire", "token"].includes(objectKind(model));
  }

  function canActorRevealHideStageObjects(role, config) {
    const normalizedRole = String(role || "").trim().toLowerCase();
    if (normalizedRole === "producer" || normalizedRole === "director") {
      return true;
    }
    return (normalizedRole === "cast" || normalizedRole === "actor") && venueConfigFlag(config, "actors_can_reveal", false);
  }

  function canEditLiveCard(model, canManageIndexCards) {
    return isCardObject(model) && model?.live && !objectState(model).locked && Boolean(canManageIndexCards);
  }

  function canMoveLiveStageObject(model, canManageIndexCards) {
    return isLiveStageObject(model) && !objectState(model).locked && Boolean(canManageIndexCards);
  }

  function canDuplicateLiveStageObject(model, canManageIndexCards) {
    return isLiveStageObject(model) && !objectState(model).locked && Boolean(canManageIndexCards);
  }

  function canRemoveLiveStageObject(model, canManageIndexCards) {
    return isLiveStageObject(model) && !objectState(model).locked && Boolean(canManageIndexCards);
  }

  function canDeleteLiveCard(model, canManageIndexCards) {
    return isCardObject(model) && model?.live && !objectState(model).locked && Boolean(canManageIndexCards);
  }

  function cardFaceForModel(model) {
    const cardKey = String(model?.elementId || model?.elementSlug || model?.key || "");
    return String(model?.cardFace || model?.source?.data?.face || model?.source?.data?.card_face || model?.face || "front").toLowerCase() === "back" ? "back" : "front";
  }

  function cardPinData(model) {
    const data = model?.source?.data || model?.data || {};
    const position = model?.position || {};
    const readNumber = (value) => {
      if (value === null || value === undefined || value === "") {
        return Number.NaN;
      }
      const parsed = Number(value);
      return Number.isFinite(parsed) ? parsed : Number.NaN;
    };
    const pinMode = String(data.pin_mode || position.frame || "").trim().toLowerCase();
    return {
      mode: pinMode === "world" ? "world" : "overlay",
      worldX: readNumber(data.world_x),
      worldY: readNumber(data.world_y),
      screenX: readNumber(data.screen_x),
      screenY: readNumber(data.screen_y),
    };
  }

  function cardDisplayMode(model) {
    return cardPinData(model).mode;
  }

  function normalizeOverlayPoint(point, bounds) {
    const stageBounds = bounds || { x: 0, y: 0, width: 1, height: 1 };
    const x = Number(point?.x || 0);
    const y = Number(point?.y || 0);
    return {
      screen_x: clampNumber((x - stageBounds.x) / stageBounds.width, 0, 1, 0.5),
      screen_y: clampNumber((y - stageBounds.y) / stageBounds.height, 0, 1, 0.5),
    };
  }

  function clampOverlayPoint(point, bounds) {
    const stageBounds = bounds || { x: 0, y: 0, width: 0, height: 0 };
    return {
      x: Math.max(stageBounds.x, Math.min(stageBounds.x + stageBounds.width, Math.round(Number(point?.x || 0)))),
      y: Math.max(stageBounds.y, Math.min(stageBounds.y + stageBounds.height, Math.round(Number(point?.y || 0)))),
    };
  }

  function offsetPoint(point, dx = 24, dy = 16) {
    return {
      x: Math.round(Number(point?.x || 0) + Number(dx || 0)),
      y: Math.round(Number(point?.y || 0) + Number(dy || 0)),
    };
  }

  function tokenSnapModeForModel(model, gridConfig) {
    const data = model?.source?.data || {};
    const snapMode = String(data.snap_mode || "").trim().toLowerCase();
    if (snapMode === "grid" || snapMode === "free") {
      return snapMode;
    }
    return gridConfig && gridConfig.grid_type !== "none" ? "grid" : "free";
  }

  function tokenLayerForModel(model) {
    const data = model?.source?.data || {};
    const tokenLayer = String(data.token_layer || "").trim().toLowerCase();
    return tokenLayer === "director" ? "director" : "public";
  }

  function tokenScaleForModel(model) {
    const data = model?.source?.data || {};
    const scale = Number(data.scale ?? 100);
    if (!Number.isFinite(scale) || scale <= 0) return 100;
    return Math.max(25, Math.min(500, scale));
  }

  function tokenDisplayNameForModel(model) {
    const source = model?.source || {};
    const data = source?.data || {};
    const candidates = [
      model?.displayName,
      model?.label,
      data.display_name,
      data.asset_display_name,
      data.asset_name,
      model?.assetName,
      source?.name,
    ];
    for (const candidate of candidates) {
      const text = String(candidate || "").trim();
      if (text) return text;
    }
    return "Token";
  }

  function tokenStatusPercentForModel(model) {
    const data = model?.source?.data || {};
    const visibility = model?.visibility || model?.source?.visibility || {};
    const candidates = [
      model?.statusPercent,
      model?.status_percent,
      data.status_percent,
      data.health_percent,
      data.counter_percent,
      data.resource_percent,
      data.bar_percent,
      visibility.status_percent,
    ];
    for (const candidate of candidates) {
      const value = Number(candidate);
      if (Number.isFinite(value)) {
        return Math.max(0, Math.min(100, value));
      }
    }
    return null;
  }

  function tokenFootprintForModel(model) {
    const data = model?.source?.data || {};
    const width = Number(data.default_grid_width ?? model?.defaultGridWidth ?? 1);
    const height = Number(data.default_grid_height ?? model?.defaultGridHeight ?? 1);
    return {
      width: Math.max(1, Math.round(Number.isFinite(width) ? width : 1)),
      height: Math.max(1, Math.round(Number.isFinite(height) ? height : 1)),
    };
  }

  function tokenPlacementBaseSize(model, gridConfig) {
    const snapMode = tokenSnapModeForModel(model, gridConfig);
    const hasGrid = Boolean(gridConfig && gridConfig.grid_type && gridConfig.grid_type !== "none");
    return hasGrid && snapMode === "grid" ? Number(gridConfig.cell_size || 50) : 64;
  }

  function tokenDisplaySizeForModel(model, gridConfig) {
    const footprint = tokenFootprintForModel(model);
    const base = tokenPlacementBaseSize(model, gridConfig);
    const scale = tokenScaleForModel(model) / 100;
    return {
      width: Math.max(8, Math.round(base * footprint.width * scale)),
      height: Math.max(8, Math.round(base * footprint.height * scale)),
    };
  }

  function cardStatusBadgeText(model) {
    const state = objectState(model);
    if (!state.nameplateVisible) {
      return "";
    }
    if (!state.visible) {
      return "HIDDEN";
    }
    if (state.locked) {
      return "LOCKED";
    }
    if (isTokenObject(model)) {
      return tokenLayerForModel(model) === "director" ? "DIR" : "TOK";
    }
    if (isCardObject(model)) {
      return cardDisplayMode(model) === "world" ? "MAP" : "SCREEN";
    }
    return "";
  }

  function canToggleLiveVisibility(model, canManageIndexCards, canRevealHide) {
    return isLiveStageObject(model) && !objectState(model).locked && Boolean(canManageIndexCards) && Boolean(canRevealHide);
  }

  function canToggleNameplate(model, canManageIndexCards) {
    return isLiveStageObject(model) && !objectState(model).locked && Boolean(canManageIndexCards);
  }

  function canToggleLock(model, canManageIndexCards) {
    return isLiveStageObject(model) && Boolean(canManageIndexCards);
  }

  function canTogglePinState(model, canManageIndexCards) {
    return isCardObject(model) && model?.live && !objectState(model).locked && Boolean(canManageIndexCards);
  }

  function truncateCardText(text, maxChars = 24) {
    const value = String(text || "").replace(/\s+/g, " ").trim();
    if (!value) return "";
    if (value.length <= maxChars) return value;
    return `${value.slice(0, Math.max(1, maxChars - 1)).trimEnd()}…`;
  }

  function cardEditorDraftFromModel(model) {
    const source = model?.source || {};
    const data = source?.data || {};
    return {
      elementId: String(model?.elementId || source?.element_id || ""),
      elementSlug: String(model?.elementSlug || source?.slug || ""),
      label: String(model?.label || source?.name || data.front_text || "Index card"),
      frontText: String(model?.frontText || data.front_text || source?.name || "").trim(),
      backText: String(model?.backText || data.back_text || "").trim(),
      color: String(model?.color || data.color || "#d9c7a6").trim() || "#d9c7a6",
    };
  }

  function stageContextMenuModel() {
    return {
      key: "stage:canvas",
      kind: "stage",
      live: false,
      elementId: "",
      elementSlug: "",
      label: "Stage",
      position: { x: 0, y: 0, order: 0 },
    };
  }

  function resolveStageObjectActions(objectModel, context = {}) {
    const kind = objectKind(objectModel);
    const state = objectState(objectModel);
    const actions = [];
    const canManageIndexCards = Boolean(context.canManageIndexCards);
    const canManageStageTokens = Boolean(context.canManageStageTokens);
    const canReveal = Boolean(context.canActorRevealHideStageObjects);
    const hasSelection = Boolean(context.hasSelection);
    const cardFace = typeof context.cardFaceForModel === "function"
      ? context.cardFaceForModel
      : cardFaceForModel;
    const cardDisplay = typeof context.cardDisplayMode === "function"
      ? context.cardDisplayMode
      : cardDisplayMode;
    const tokenScale = typeof context.tokenScaleForModel === "function"
      ? context.tokenScaleForModel
      : tokenScaleForModel;
    const tokenSnapMode = typeof context.tokenSnapModeForModel === "function"
      ? context.tokenSnapModeForModel
      : tokenSnapModeForModel;
    const tokenLayer = typeof context.tokenLayerForModel === "function"
      ? context.tokenLayerForModel
      : tokenLayerForModel;

    const push = (action, label, group, options = {}) => {
      actions.push({
        action,
        label,
        group,
        requiresPoint: Boolean(options.requiresPoint),
        disabled: Boolean(options.disabled),
      });
    };

    if (kind === "stage") {
      push("set-map", "Add / Replace Map", "create", { disabled: !canManageIndexCards });
      push("configure-grid", "Configure Grid", "create", { disabled: !canManageIndexCards });
      push("add-token", "Add Token", "create", { disabled: !canManageStageTokens });
      push("add-index-card", "Create Index Card", "create", { disabled: !canManageIndexCards, requiresPoint: true });
      push("inspect", "Inspect Stage", "info");
      if (hasSelection) {
        push("clear", "Clear selection", "clear");
      }
      return actions;
    }

    if (kind === "fire") {
      push("select", "Select", "info");
      push("info", "Info", "info");
      if (canRemoveLiveStageObject(objectModel, canManageIndexCards)) {
        push("remove", "Remove from Stage", "remove");
      }
      if (hasSelection) {
        push("clear", "Clear selection", "clear");
      }
      return actions;
    }

    if (!objectModel?.live) {
      push("select", "Select", "info");
      push("info", "Info", "info");
      // Kernel 73A bound interactions (e.g. Kessa's token -> "Speak with
      // Kessa") previously only surfaced in the selected-object action
      // panel after a plain click (see runtime.js's syncSelectedActions),
      // not here in the right-click context menu -- the more discoverable
      // path a Director/Player naturally reaches for first. Both paths call
      // the same openInteraction controller, so this is additive, not a
      // second source of truth.
      const binding = objectModel?.source?.data?.binding;
      if (binding?.participant_interaction_id && binding?.enabled) {
        push(`open-bound-interaction:${binding.participant_interaction_id}`, binding.stage_button_label || "Interact", "interact");
      }
      if (hasSelection) {
        push("clear", "Clear selection", "clear");
      }
      return actions;
    }

    if (state.locked) {
      if (canToggleLock(objectModel, canManageIndexCards)) {
        push("unlock", "Unlock", "lock");
      }
      push("info", "Info", "info");
      return actions;
    }

    push("info", "Info", "info");

    if (kind === "token" && canManageStageTokens) {
      push("scale", `Scale (${Math.round(tokenScale(objectModel, context.gridConfig))}%)`, "edit");
      push(tokenSnapMode(objectModel, context.gridConfig) === "grid" ? "free-placement" : "snap-to-grid", tokenSnapMode(objectModel, context.gridConfig) === "grid" ? "Free Placement" : "Snap to Grid", "pin");
      push("replace-asset", "Replace Asset", "edit");
      push(tokenLayer(objectModel) === "director" ? "move-public-layer" : "move-director-layer", tokenLayer(objectModel) === "director" ? "Move to Public Layer" : "Move to Director Layer", "visibility");
    }

    if (kind === "card" && canEditLiveCard(objectModel, canManageIndexCards)) {
      push("inspect", "Inspect", "edit");
      push("edit", "Edit", "edit");
      push("flip", cardFace(objectModel) === "back" ? "Show Front" : "Flip", "face");
    }

    if (kind === "card" && canTogglePinState(objectModel, canManageIndexCards)) {
      push(cardDisplay(objectModel) === "world" ? "unpin" : "pin", cardDisplay(objectModel) === "world" ? "Pin to Screen" : "Attach to Map", "pin");
    }

    if (canMoveLiveStageObject(objectModel, canManageIndexCards)) {
      push("move-here", "Move here", "move", { requiresPoint: true });
    }

    if (canToggleLiveVisibility(objectModel, canManageIndexCards, canReveal)) {
      push(state.visible ? "hide" : "show", state.visible ? "Hide Audience" : "Show Audience", "visibility");
    }

    if (kind === "card" && canToggleNameplate(objectModel, canManageIndexCards)) {
      push(state.nameplateVisible ? "hide-nameplate" : "show-nameplate", state.nameplateVisible ? "Hide Nameplate" : "Show Nameplate", "visibility");
    }

    if (kind === "token" && canToggleNameplate(objectModel, canManageIndexCards)) {
      push(state.nameplateVisible ? "hide-nameplate" : "show-nameplate", state.nameplateVisible ? "Hide Nameplate" : "Show Nameplate", "visibility");
    }

    if (kind === "card" && canDuplicateLiveStageObject(objectModel, canManageIndexCards)) {
      push("duplicate", "Duplicate Card", "duplicate");
    }

    if ((kind === "card" || kind === "prop" || kind === "fire" || kind === "token") && canRemoveLiveStageObject(objectModel, canManageIndexCards)) {
      push("remove", "Remove from Stage", "remove");
    }

    if (kind === "card" && canDeleteLiveCard(objectModel, canManageIndexCards)) {
      push("delete-card", "Delete Card", "remove");
    }

    if ((kind === "card" || kind === "fire" || kind === "token") && canToggleLock(objectModel, canManageIndexCards)) {
      push("lock", "Lock", "lock");
    }

    if (kind === "token" && canDuplicateLiveStageObject(objectModel, canManageIndexCards)) {
      push("duplicate", "Duplicate Token", "duplicate");
    }

    return actions;
  }

  function clampNumber(value, min, max, fallback) {
    const parsed = Number(value);
    if (!Number.isFinite(parsed)) return fallback;
    return Math.max(min, Math.min(max, parsed));
  }

  return {
    venueConfigFlag,
    objectKind,
    objectState,
    updateObjectMatches,
    viewerCanSeeHiddenCards,
    isCardObject,
    isTokenObject,
    isLiveStageObject,
    canActorRevealHideStageObjects,
    canEditLiveCard,
    canMoveLiveStageObject,
    canDuplicateLiveStageObject,
    canRemoveLiveStageObject,
    canDeleteLiveCard,
    cardFaceForModel,
    cardPinData,
    cardDisplayMode,
    normalizeOverlayPoint,
    clampOverlayPoint,
    offsetPoint,
    tokenSnapModeForModel,
    tokenLayerForModel,
    tokenScaleForModel,
    tokenDisplayNameForModel,
    tokenStatusPercentForModel,
    tokenFootprintForModel,
    tokenPlacementBaseSize,
    tokenDisplaySizeForModel,
    cardStatusBadgeText,
    canToggleLiveVisibility,
    canToggleNameplate,
    canToggleLock,
    canTogglePinState,
    truncateCardText,
    cardEditorDraftFromModel,
    stageContextMenuModel,
    resolveStageObjectActions,
  };
});
