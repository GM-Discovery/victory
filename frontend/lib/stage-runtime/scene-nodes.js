(function (root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  }
  root.VictoryStageSceneNodes = factory();
})(typeof globalThis !== "undefined" ? globalThis : window, function () {
  function createSceneNodeFactory(deps) {
    const PIXI = deps.PIXI;
    const objectState = deps.objectState;
    const canEditLiveCard = deps.canEditLiveCard;
    const canToggleLock = deps.canToggleLock;
    const canManageIndexCards = deps.canManageIndexCards;
    const currentRole = () => deps.currentRole?.() || "audience";
    const cardFaceForModel = deps.cardFaceForModel;
    const cardStatusBadgeText = deps.cardStatusBadgeText;
    const viewerCanSeeHiddenCards = deps.viewerCanSeeHiddenCards;
    const tokenDisplayNameForModel = deps.tokenDisplayNameForModel;
    const tokenStatusPercentForModel = deps.tokenStatusPercentForModel;
    const tokenLayerForModel = deps.tokenLayerForModel;
    const tokenDisplaySizeForModel = deps.tokenDisplaySizeForModel;
    const truncateCardText = deps.truncateCardText;
    const isTokenObject = deps.isTokenObject;
    const selectObject = deps.selectObject;
    const setStageStatus = deps.setStageStatus;
    const setMovementReport = deps.setMovementReport;
    const openCardEditor = deps.openCardEditor;
    const wasContextMenuHandled = deps.wasContextMenuHandled;
    const cancelContextMenuEvent = deps.cancelContextMenuEvent;
    const openResolvedContextMenu = deps.openResolvedContextMenu;
    const eventClientPoint = deps.eventClientPoint;
    const stageScreenPointFromClient = deps.stageScreenPointFromClient;
    const cardDisplayMode = deps.cardDisplayMode;
    const tokenSnapModeForModel = deps.tokenSnapModeForModel;
    const renderPixiScene = deps.renderPixiScene;
    const stageState = deps.stageState;

    // Kernel 90 §14/§45: mark an object that is hidden from ordinary viewers
    // but present on the Director's working stage.
    //
    // The Director's stage must NOT lose the object (§14 is explicit that
    // hidden objects stay visible while directing, absent a dedicated
    // preview-as-audience mode), so the treatment has to be legible without
    // being loud: reduced opacity plus a dashed outline, reusing the same
    // faded-backstage idiom makeHotspotNode already uses for unusable nodes.
    // No debug label and no state inspector -- §45 rules both out, and the
    // existing "HIDDEN" status badge already carries the words for objects
    // that show a nameplate.
    //
    // This can only ever fire for a backstage viewer, because an ordinary
    // viewer does not receive a hidden object at all (world/snapshot.go omits
    // it). The role check is therefore belt-and-braces rather than the
    // security boundary -- the boundary is server-side, per §36.
    function markHiddenForDirector(container, model) {
      const st = model?.state || model?.source?.state || {};
      if (st.hidden_backstage_only !== true) return;
      if (!["producer", "director", "operator", "crew"].includes(currentRole())) return;

      container.alpha = Math.min(container.alpha ?? 1, 0.45);

      const outline = new PIXI.Graphics();
      const bounds = container.getLocalBounds();
      outline.lineStyle(2, 0x7fb2ff, 0.85);
      // Dashed by hand: PIXI v7 has no dash support on Graphics, and a solid
      // outline reads as "selected" in this renderer, which would collide
      // with the real selection treatment.
      const step = 9;
      const x0 = bounds.x - 3;
      const y0 = bounds.y - 3;
      const x1 = bounds.x + bounds.width + 3;
      const y1 = bounds.y + bounds.height + 3;
      for (let x = x0; x < x1; x += step * 2) {
        outline.moveTo(x, y0); outline.lineTo(Math.min(x + step, x1), y0);
        outline.moveTo(x, y1); outline.lineTo(Math.min(x + step, x1), y1);
      }
      for (let y = y0; y < y1; y += step * 2) {
        outline.moveTo(x0, y); outline.lineTo(x0, Math.min(y + step, y1));
        outline.moveTo(x1, y); outline.lineTo(x1, Math.min(y + step, y1));
      }
      outline.eventMode = "none";
      outline.zIndex = 9999;
      container.addChild(outline);
    }

    // removeChildren() only detaches DisplayObjects from the tree -- it does
    // not release their GPU-side geometry/mesh data, so a full scene rebuild
    // (fires on every snapshot replace) leaked every prior token/card node
    // for the lifetime of the page. destroy({children:true, texture:false})
    // frees each node's own resources while leaving textures alone (they're
    // cached/shared by URL via PIXI.Assets, not owned by any one node).
    function destroyLayerChildren(layer) {
      if (!layer) return;
      const removed = layer.removeChildren();
      for (const child of removed) {
        child.destroy?.({ children: true, texture: false, baseTexture: false });
      }
    }

    function clearSceneNodes() {
      stageState.currentNodeMap = new Map();
      destroyLayerChildren(stageState.pinnedObjectLayer);
      destroyLayerChildren(stageState.overlayObjectLayer);
      destroyLayerChildren(stageState.floatingObjectLayer);
      destroyLayerChildren(stageState.facadeLayer);
    }

    function refreshNodeSelection() {
      for (const [key, node] of stageState.currentNodeMap.entries()) {
        node.updateSelected?.(stageState.currentSelection?.key === key);
      }
    }

    function makeCardNode(model) {
      const container = new PIXI.Container();
      container.sortableChildren = true;
      const state = objectState(model);
      const cardKey = String(model.elementId || model.elementSlug || model.key || "");
      const face = cardFaceForModel(model);
      const titleText = face === "back"
        ? truncateCardText(model.backText || "Back", 22) || "Back"
        : truncateCardText(model.frontText || model.label || "Untitled card", 22) || "Untitled card";
      const shadow = new PIXI.Graphics();
      shadow.beginFill(0x000000, 0.28);
      shadow.drawRoundedRect(8, 10, 192, 132, 16);
      shadow.endFill();
      const body = new PIXI.Graphics();
      body.beginFill(PIXI.utils.string2hex(model.color || "#d9c7a6"), 1);
      body.lineStyle(2, 0x5d4327, 0.52);
      body.drawRoundedRect(0, 0, 192, 132, 16);
      body.endFill();
      const titleStyle = new PIXI.TextStyle({ fontFamily: "Arial", fontSize: 14, fontWeight: "700", fill: 0x2f2112, wordWrap: true, wordWrapWidth: 144 });
      const title = new PIXI.Text(titleText, titleStyle);
      title.position.set(18, 18);
      title.alpha = 0.98;
      title.eventMode = "static";
      title.cursor = "text";
      const faceButtonBg = new PIXI.Graphics();
      faceButtonBg.beginFill(0x000000, 0.18);
      faceButtonBg.drawRoundedRect(0, 0, 26, 26, 13);
      faceButtonBg.endFill();
      faceButtonBg.position.set(160, 104);
      faceButtonBg.alpha = (canEditLiveCard(model) || canToggleLock(model)) && state.visible ? 1 : 0.34;
      const faceButton = new PIXI.Text(face === "back" ? "↻" : "↺", new PIXI.TextStyle({ fontFamily: "Arial", fontSize: 15, fontWeight: "700", fill: 0x5d4327 }));
      faceButton.position.set(168, 108);
      faceButton.eventMode = "static";
      faceButton.cursor = "pointer";
      faceButton.on("pointerdown", (event) => {
        event.stopPropagation();
        if (objectState(model).locked) {
          selectObject(model, `${model.label} is locked.`);
          return;
        }
        if (!canEditLiveCard(model) && !canManageIndexCards(currentRole())) return;
        deps.cardFaceState.set(cardKey, face === "back" ? "front" : "back");
        renderPixiScene();
      });
      const statusBadgeTextValue = cardStatusBadgeText(model);
      const statusBadge = new PIXI.Container();
      const statusBadgeStyle = new PIXI.TextStyle({ fontFamily: "Arial", fontSize: 10, fontWeight: "700", fill: 0xf0d49b });
      const statusBadgeLabel = new PIXI.Text(statusBadgeTextValue, statusBadgeStyle);
      statusBadgeLabel.position.set(10, 3);
      const statusBadgeBg = new PIXI.Graphics();
      statusBadgeBg.beginFill(0x1a0f0f, 0.84);
      statusBadgeBg.lineStyle(1, 0xf0d49b, 0.22);
      const statusBadgeWidth = Math.max(54, Math.ceil(statusBadgeLabel.width + 20));
      statusBadgeBg.drawRoundedRect(0, 0, statusBadgeWidth, 18, 9);
      statusBadgeBg.endFill();
      statusBadge.addChild(statusBadgeBg, statusBadgeLabel);
      statusBadge.position.set(Math.max(8, 192 - statusBadgeWidth - 10), 10);
      statusBadge.visible = Boolean(statusBadgeTextValue);
      const focus = new PIXI.Graphics();
      focus.lineStyle(0, 0x000000, 0);
      focus.drawRoundedRect(0, 0, 200, 140, 18);
      container.addChild(shadow, body, title, faceButtonBg, faceButton, statusBadge, focus);
      container.eventMode = "static";
      container.cursor = "pointer";
      container.interactive = true;
      container.visible = state.visible || viewerCanSeeHiddenCards();
      title.visible = true;
      faceButtonBg.visible = true;
      faceButton.visible = true;
      const node = { model, container, updateSelected(selected) { focus.clear(); if (selected) { focus.lineStyle(3, 0xf0d49b, 0.9); focus.drawRoundedRect(0, 0, 200, 140, 18); container.zIndex = 50; } else { focus.lineStyle(0, 0x000000, 0); focus.drawRoundedRect(0, 0, 200, 140, 18); container.zIndex = 10; } } };
      const openCardContextMenu = (event) => { if (wasContextMenuHandled(event)) return; cancelContextMenuEvent(event); openResolvedContextMenu(event); };
      container.on("pointerdown", (event) => {
        if (event?.data?.button === 2 || event.button === 2 || (event.buttons & 2) === 2) return;
        event.stopPropagation();
        selectObject(model, `${model.label} selected.`);
        if (model.live) {
          if (objectState(model).locked) {
            setStageStatus(`${model.label} is locked.`);
            setMovementReport("Locked objects cannot be dragged.");
            return;
          }
          const clientPoint = eventClientPoint(event);
          const screenPoint = stageScreenPointFromClient(clientPoint.clientX, clientPoint.clientY);
          const space = cardDisplayMode(model) === "world" ? "world" : "screen";
          const renderedPoint = { x: container.x, y: container.y };
          const floatingPoint = { x: Math.round(Number(screenPoint.x || 0)), y: Math.round(Number(screenPoint.y || 0)) };
          deps.setDragState?.({
            node: container, model, space, originalPinMode: space, originalData: { ...(model.source?.data || {}) }, floatingPoint, offsetX: floatingPoint.x - renderedPoint.x, offsetY: floatingPoint.y - renderedPoint.y,
          });
          setMovementReport("Dragging live object...");
          setStageStatus("Release to update the card.");
        }
      });
      const inspectFromText = (event) => { event?.stopPropagation?.(); event?.stopImmediatePropagation?.(); selectObject(model, `${model.label} selected.`); openCardEditor(model); };
      title.on("click", inspectFromText);
      title.on("contextmenu", openCardContextMenu);
      faceButton.on("contextmenu", openCardContextMenu);
      container.on("contextmenu", openCardContextMenu);
      markHiddenForDirector(container, model);
      return node;
    }

    function makeFireNode(model) {
      const container = new PIXI.Container();
      const glow = new PIXI.Graphics();
      glow.beginFill(0xff8b36, 0.14);
      glow.drawCircle(62, 62, 78);
      glow.endFill();
      const halo = new PIXI.Graphics();
      halo.beginFill(0xffc16f, 0.22);
      halo.drawCircle(62, 62, 40);
      halo.endFill();
      const sprite = PIXI.Sprite.from("/assets/fire.jpg");
      sprite.anchor.set(0, 0);
      sprite.position.set(0, 0);
      sprite.width = 124;
      sprite.height = 124;
      const base = new PIXI.Graphics();
      base.beginFill(0x000000, 0.2);
      base.drawEllipse(62, 118, 54, 14);
      base.endFill();
      const label = new PIXI.Text(model.label || "Fire", new PIXI.TextStyle({ fontFamily: "Arial", fontSize: 12, fontWeight: "700", fill: 0xf4dfb4 }));
      label.anchor.set(0, 0);
      label.position.set(0, 130);
      container.addChild(glow, halo, base, sprite, label);
      container.eventMode = "static";
      container.cursor = "pointer";
      container.interactive = true;
      const focus = new PIXI.Graphics();
      focus.lineStyle(0, 0x000000, 0);
      focus.drawRoundedRect(0, 0, 124, 150, 18);
      container.addChild(focus);
      const node = { model, container, updateSelected(selected) { focus.clear(); if (selected) { focus.lineStyle(3, 0xf0d49b, 0.95); focus.drawRoundedRect(0, 0, 124, 150, 18); } else { focus.lineStyle(0, 0x000000, 0); focus.drawRoundedRect(0, 0, 124, 150, 18); } } };
      container.on("pointerdown", (event) => { if (event?.data?.button === 2 || event.button === 2 || (event.buttons & 2) === 2) return; event.stopPropagation(); selectObject(model, "Fire selected."); });
      container.on("contextmenu", (event) => { if (wasContextMenuHandled(event)) return; cancelContextMenuEvent(event); openResolvedContextMenu(event); });
      return node;
    }

    function makeTokenNode(model) {
      const container = new PIXI.Container();
      container.sortableChildren = true;
      const size = tokenDisplaySizeForModel(model);
      const legacyColor = String(model.color || model.source?.data?.color || "").trim().toLowerCase();
      const auraHex = String(model.tokenAura || model.source?.data?.token_aura || "").trim().toLowerCase() || (legacyColor && legacyColor !== "#d9c7a6" ? legacyColor : "");
      const auraColor = /^#[0-9a-f]{6}$/.test(auraHex) ? PIXI.utils.string2hex(auraHex) : null;
      const assetURL = String(model.assetContentURL || model.source?.data?.asset_content_url || "").trim();
      const thumbnailURL = String(model.assetThumbnailURL || model.source?.data?.asset_thumbnail_url || "").trim();
      const textureSource = assetURL || thumbnailURL || "/assets/construction.png";
      const texture = PIXI.Texture.from(textureSource);
      if (texture?.baseTexture && !texture.baseTexture.valid) {
        const rerender = () => renderPixiScene?.();
        texture.baseTexture.once?.("loaded", rerender);
        texture.baseTexture.once?.("error", rerender);
      }
      const aura = new PIXI.Graphics();
      if (auraColor !== null) {
        aura.beginFill(auraColor, 0.14);
        aura.drawRoundedRect(-size.width / 2 - 16, -size.height / 2 - 16, size.width + 32, size.height + 32, 24);
        aura.endFill();
      }
      const sprite = new PIXI.Sprite(texture);
      sprite.anchor.set(0.5);
      sprite.width = size.width;
      sprite.height = size.height;
      sprite.eventMode = "static";
      sprite.cursor = "pointer";
      const layerTag = new PIXI.Text(tokenLayerForModel(model) === "director" ? "DIR" : "TOK", new PIXI.TextStyle({ fontFamily: "Arial", fontSize: 10, fontWeight: "700", fill: tokenLayerForModel(model) === "director" ? 0x9dd0ff : 0xe9f5ff }));
      layerTag.anchor.set(0.5);
      layerTag.position.set(size.width / 2 - 12, -size.height / 2 + 12);
      layerTag.visible = false;
      const label = new PIXI.Text(truncateCardText(tokenDisplayNameForModel(model), 24) || "Token", new PIXI.TextStyle({ fontFamily: "Arial", fontSize: 12, fontWeight: "700", fill: 0xf4e3c1, align: "center", wordWrap: true, wordWrapWidth: Math.max(92, size.width + 24) }));
      label.anchor.set(0.5, 0);
      label.position.set(0, size.height / 2 + 8);
      const statusPercent = tokenStatusPercentForModel(model);
      const hasStatus = Number.isFinite(statusPercent);
      const statusTrack = hasStatus ? new PIXI.Graphics() : null;
      const statusWidth = Math.max(44, Math.min(96, Math.round(Math.max(48, size.width * 0.72))));
      const statusHeight = 4;
      const statusY = size.height / 2 + 18;
      const statusGlyph = new PIXI.Graphics();
      statusGlyph.position.set(-Math.max(18, Math.round(size.width / 2 + 2)), size.height / 2 + 9);
      if (statusTrack) {
        statusTrack.beginFill(0x14110e, 0.82);
        statusTrack.drawRoundedRect(-statusWidth / 2, statusY, statusWidth, statusHeight, 999);
        statusTrack.endFill();
        statusTrack.lineStyle(1, 0xffffff, 0.06);
        statusTrack.drawRoundedRect(-statusWidth / 2, statusY, statusWidth, statusHeight, 999);
        statusGlyph.beginFill(statusPercent <= 25 ? 0xc85a4d : statusPercent <= 60 ? 0xe0b04c : 0x8dcf83, 0.92);
        statusGlyph.drawCircle(0, 0, 4);
        statusGlyph.endFill();
      }
      const statusFill = hasStatus ? new PIXI.Graphics() : null;
      if (statusFill) {
        const fillWidth = Math.max(2, Math.round((statusWidth - 2) * (statusPercent / 100)));
        const fillColor = statusPercent <= 25 ? 0xc85a4d : statusPercent <= 60 ? 0xe0b04c : 0x8dcf83;
        statusFill.beginFill(fillColor, 0.96);
        statusFill.drawRoundedRect(-statusWidth / 2 + 1, statusY + 1, fillWidth, Math.max(1, statusHeight - 2), 999);
        statusFill.endFill();
      }
      const focus = new PIXI.Graphics();
      focus.lineStyle(0, 0x000000, 0);
      focus.drawRoundedRect(-size.width / 2 - 4, -size.height / 2 - 4, size.width + 8, size.height + 8, 14);
      container.addChild(aura, sprite, focus);
      container.eventMode = "static";
      container.cursor = "pointer";
      container.interactive = true;
      const applyNameplateVisibility = () => {
        const show = Boolean(objectState(model).nameplateVisible);
        const parts = [layerTag, label];
        if (hasStatus) parts.push(statusGlyph);
        if (hasStatus && statusTrack) parts.push(statusTrack);
        if (hasStatus && statusFill) parts.push(statusFill);
        for (const part of parts) {
          const attached = part.parent === container;
          if (show && !attached) container.addChild(part);
          if (!show && attached) container.removeChild(part);
        }
      };
      const updateFocus = (selected) => {
        focus.clear();
        const focusHeight = size.height + 8;
        if (selected) {
          focus.lineStyle(3, 0x8fb7da, 0.92);
          focus.drawRoundedRect(-size.width / 2 - 4, -size.height / 2 - 4, size.width + 8, focusHeight, 14);
          if (auraColor !== null) {
            aura.clear();
            aura.beginFill(auraColor, 0.22);
            aura.drawRoundedRect(-size.width / 2 - 18, -size.height / 2 - 18, size.width + 36, size.height + 36, 26);
            aura.endFill();
          }
          container.zIndex = 60;
        } else {
          focus.lineStyle(0, 0x000000, 0);
          focus.drawRoundedRect(-size.width / 2 - 4, -size.height / 2 - 4, size.width + 8, focusHeight, 14);
          if (auraColor !== null) {
            aura.clear();
            aura.beginFill(auraColor, 0.14);
            aura.drawRoundedRect(-size.width / 2 - 16, -size.height / 2 - 16, size.width + 32, size.height + 32, 24);
            aura.endFill();
          }
          container.zIndex = tokenLayerForModel(model) === "director" ? 35 : 25;
        }
      };
      const beginTokenDrag = (event) => {
        if (event?.data?.button === 2 || event.button === 2 || (event.buttons & 2) === 2) return;
        event.stopPropagation();
        selectObject(model, `${model.label} selected.`);
        if (!model.live) return;
        if (objectState(model).locked) {
          setStageStatus(`${model.label} is locked.`);
          setMovementReport("Locked objects cannot be dragged.");
          return;
        }
        const clientPoint = eventClientPoint(event);
        const screenPoint = stageScreenPointFromClient(clientPoint.clientX, clientPoint.clientY);
        const renderedPoint = { x: container.x, y: container.y };
        const floatingPoint = { x: Math.round(Number(screenPoint.x || 0)), y: Math.round(Number(screenPoint.y || 0)) };
        deps.setDragState?.({
          node: container,
          model,
          space: "screen",
          originalPinMode: tokenSnapModeForModel(model),
          originalData: { ...(model.source?.data || {}) },
          floatingPoint,
          offsetX: floatingPoint.x - renderedPoint.x,
          offsetY: floatingPoint.y - renderedPoint.y,
        });
        setMovementReport("Dragging live token...");
        setStageStatus("Release to update the token.");
      };
      const node = { model, container, updateSelected(selected) { updateFocus(selected); applyNameplateVisibility(); } };
      applyNameplateVisibility();
      const openTokenContextMenu = (event) => {
        if (event?.data?.button === 2 || event.button === 2 || (event.buttons & 2) === 2) {
          if (wasContextMenuHandled(event)) return;
          cancelContextMenuEvent(event);
          openResolvedContextMenu(event);
        }
      };
      container.on("pointerdown", (event) => {
        openTokenContextMenu(event);
        beginTokenDrag(event);
      });
      sprite.on("pointerdown", (event) => {
        openTokenContextMenu(event);
        beginTokenDrag(event);
      });
      container.on("rightdown", openTokenContextMenu);
      sprite.on("rightdown", openTokenContextMenu);
      container.on("rightclick", openTokenContextMenu);
      sprite.on("rightclick", openTokenContextMenu);
      container.on("contextmenu", (event) => { if (wasContextMenuHandled(event)) return; cancelContextMenuEvent(event); openResolvedContextMenu(event); });
      sprite.on("contextmenu", (event) => { if (wasContextMenuHandled(event)) return; cancelContextMenuEvent(event); openResolvedContextMenu(event); });
      markHiddenForDirector(container, model);
      return node;
    }

    // Kernel 74: an interaction hotspot aligned over map artwork that
    // already exists (the Courtyard's ironbound oak door).
    //
    // It draws no art of its own -- deliberately. S1.1 rejects "a separate
    // visible door token merely to duplicate the artwork", so what a Player
    // sees is the painted door, a nameplate, and a faint outline that
    // strengthens on hover/focus. The node exists to be a hit target and an
    // accessible control, not a picture.
    function makeHotspotNode(model) {
      const container = new PIXI.Container();
      container.sortableChildren = true;

      // An unbound (or disabled) hotspot is inert: it renders a faint
      // authoring outline for backstage viewers so a Director can still see
      // where it sits, and NOTHING for anyone else -- and in neither case
      // does it accept pointer events.
      //
      // This was a real bug. A hotspot left unbound rendered a full-size,
      // pointer-capturing box that sat over Kessa's token and swallowed
      // every click meant for her, then answered with "cannot be used right
      // now". A control that cannot do anything must not be able to block
      // anything either.
      const binding = model?.source?.data?.binding;
      const usable = Boolean(binding?.participant_interaction_id) && Boolean(binding?.enabled);

      const bounds = deps.hotspotPixelSize?.(model) || { width: 96, height: 220 };
      const width = Math.max(24, Number(bounds.width) || 96);
      const height = Math.max(24, Number(bounds.height) || 220);

      const region = new PIXI.Graphics();
      const paint = (strength) => {
        region.clear();
        // A near-transparent fill rather than zero alpha: PIXI needs a filled
        // area for hit-testing, and a hotspot the Player cannot click is
        // worse than one they can faintly see.
        region.beginFill(0xffffff, 0.001 + strength * 0.06);
        region.lineStyle(strength > 0 ? 2 : 1, 0xd9c7a6, strength > 0 ? 0.85 : 0.25);
        region.drawRoundedRect(-width / 2, -height / 2, width, height, 8);
        region.endFill();
      };
      paint(0);
      region.eventMode = usable ? "static" : "none";
      region.cursor = usable ? "pointer" : "default";
      if (usable) {
        region.hitArea = new PIXI.Rectangle(-width / 2, -height / 2, width, height);
      }
      container.addChild(region);

      const nameplate = new PIXI.Text(model.label || "Hotspot", new PIXI.TextStyle({
        fontFamily: "Arial",
        fontSize: 13,
        fontWeight: "700",
        fill: 0xf2e8d5,
        stroke: 0x1a1512,
        strokeThickness: 3,
        align: "center",
      }));
      nameplate.anchor.set(0.5, 0.5);
      nameplate.position.set(0, height / 2 + 14);
      nameplate.visible = usable && model.nameplateVisible !== false;
      nameplate.eventMode = usable ? "static" : "none";
      nameplate.cursor = usable ? "pointer" : "default";
      container.addChild(nameplate);

      container.eventMode = usable ? "static" : "none";
      container.cursor = usable ? "pointer" : "default";

      if (!usable) {
        // Backstage keeps a faint outline to author against; everyone else
        // gets nothing at all.
        const backstage = ["producer", "director", "operator", "crew"].includes(currentRole());
        container.visible = backstage;
        container.alpha = 0.35;
        return { model, container, activate() {}, updateSelected() {} };
      }

      // S6.3 requires keyboard focus. PIXI has no focus model, so the node
      // publishes an activation callback and runtime.js mirrors every
      // hotspot into a real focusable DOM button in the stage's accessible
      // control strip -- the same pattern the Cue stage buttons already use.
      // Click, Enter/Space, and touch all land on this one function.
      const activate = () => {
        selectObject(model, `${model.label} selected.`);
        deps.activateHotspot?.(model);
      };

      const onOver = () => paint(1);
      const onOut = () => paint(stageState.currentSelection?.key === model.key ? 1 : 0);
      for (const target of [container, region, nameplate]) {
        target.on("pointerover", onOver);
        target.on("pointerout", onOut);
        target.on("pointertap", (event) => {
          if (event?.data?.button === 2 || event.button === 2) return;
          event.stopPropagation?.();
          activate();
        });
        target.on("rightdown", (event) => {
          if (wasContextMenuHandled(event)) return;
          cancelContextMenuEvent(event);
          selectObject(model, `${model.label} selected.`);
          openResolvedContextMenu(event);
        });
      }

      return {
        model,
        container,
        activate,
        updateSelected(selected) {
          paint(selected ? 1 : 0);
          nameplate.visible = model.nameplateVisible !== false;
        },
      };
    }

    return { clearSceneNodes, refreshNodeSelection, makeCardNode, makeFireNode, makeTokenNode, makeHotspotNode };
  }

  return { createSceneNodeFactory };
});
