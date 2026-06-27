(function (root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  }
  root.VictoryCatharsisActionRouter = factory();
})(typeof globalThis !== "undefined" ? globalThis : window, function () {
  function createActionRouter(deps) {
    const syncCurrentObjectsFromProjectedState = typeof deps.syncCurrentObjectsFromProjectedState === "function"
      ? deps.syncCurrentObjectsFromProjectedState
      : () => false;

    function performStageObjectAction(action, objectModel) {
      if (!objectModel || !action) return;
      const state = deps.objectState(objectModel);
      const point = deps.getStagePoint();
      const kind = deps.objectKind(objectModel);

      if (action === "select") {
        deps.selectObject(objectModel, `${objectModel.label} selected.`);
        deps.closeContextMenu();
        return;
      }

      if (action === "clear") {
        deps.selectObject(null, "Selection cleared.");
        deps.closeContextMenu();
        return;
      }

      if (action === "info" || action === "inspect") {
        if (deps.isTokenObject(objectModel) && typeof deps.openTokenEditor === "function") {
          deps.openTokenEditor(objectModel);
        } else if (kind === "card" && typeof deps.openCardEditor === "function") {
          deps.openCardEditor(objectModel);
        } else {
          deps.selectObject(objectModel, `${objectModel.label} selected.`);
        }
        deps.closeContextMenu();
        return;
      }

      if (action === "replace-asset" && deps.isTokenObject(objectModel)) {
        deps.selectObject(null, "Selection cleared.");
        deps.openTokenPicker("replace", objectModel);
        deps.closeContextMenu();
        return;
      }

      if (action === "hide-nameplate" || action === "show-nameplate") {
        const visible = action === "show-nameplate";
        const sent = deps.sendAction("act/set_nameplate_visibility", {
          element_id: objectModel.elementId || "",
          element_slug: objectModel.elementSlug || "",
          layer: "audience",
          visible,
        });
        if (sent) {
          deps.updateLocalObjectModel(objectModel, (model) => {
            model.state = { ...(model.state || {}), nameplate_visible: visible, nameplateVisible: visible };
            model.visibility = { ...(model.visibility || {}), nameplate_visible: visible };
          });
        }
        deps.setStageStatus(sent ? `${visible ? "Showing" : "Hiding"} nameplate for ${objectModel.label}...` : "Socket unavailable.");
        deps.closeContextMenu();
        return;
      }

      if (action === "move-here") {
        if (!point) {
          deps.setStageStatus("Move here needs a stage point.");
          return;
        }
        const moveTarget = deps.isTokenObject(objectModel)
          ? deps.tokenMoveTargetForModel(objectModel, point)
          : deps.cardMoveTargetForModel(objectModel, point);
        if (!moveTarget) {
          deps.setStageStatus("Move here needs a stage point.");
          deps.closeContextMenu();
          return;
        }
        const sent = deps.isTokenObject(objectModel)
          ? deps.sendAction("update/token", {
              element_id: objectModel.elementId || "",
              element_slug: objectModel.elementSlug || "",
              venue_slug: "the-cave",
              layer: "stage",
              x: point.x,
              y: point.y,
              order: Number(objectModel.position?.order ?? 0),
              snap_mode: moveTarget.snapMode,
              token_layer: deps.tokenLayerForModel(objectModel),
              scale: deps.tokenScaleForModel(objectModel),
            })
          : deps.sendAction("update/index_card", {
              element_id: objectModel.elementId || "",
              element_slug: objectModel.elementSlug || "",
              front_text: objectModel.frontText || "",
              back_text: objectModel.backText || "",
              color: objectModel.color || "#d9c7a6",
              pin_mode: moveTarget.pinMode,
              world_x: moveTarget.world_x,
              world_y: moveTarget.world_y,
              screen_x: moveTarget.screen_x,
              screen_y: moveTarget.screen_y,
            });
        const placementSent = deps.isTokenObject(objectModel)
          ? true
          : deps.sendAction("act/place_element", {
              element_id: objectModel.elementId || "",
              element_slug: objectModel.elementSlug || "",
              venue_slug: "the-cave",
              layer: "stage",
              x: point.x,
              y: point.y,
              order: Number(objectModel.position?.order ?? 0),
            });
        if (sent) {
          const updateModel = deps.isTokenObject(objectModel)
            ? deps.updateTokenLocalModel
            : deps.updateLocalCardPinModel;
          updateModel(objectModel, (model) => {
            if (deps.isTokenObject(model)) {
              model.source.data = {
                ...(model.source.data || {}),
                snap_mode: moveTarget.snapMode,
                token_layer: deps.tokenLayerForModel(model),
                scale: deps.tokenScaleForModel(model),
              };
              model.position = {
                ...(model.position || {}),
                x: point.x,
                y: point.y,
                order: Number(model.position?.order ?? 0),
                frame: "center",
              };
            } else {
              model.source.data = {
                ...(model.source.data || {}),
                pin_mode: moveTarget.pinMode,
                world_x: moveTarget.world_x,
                world_y: moveTarget.world_y,
                screen_x: moveTarget.screen_x,
                screen_y: moveTarget.screen_y,
              };
              model.position = deps.setLocalPositionOverrideForModel(model, point) || model.position;
            }
          });
        }
        const moveOk = deps.isTokenObject(objectModel) ? sent : (sent && placementSent);
        deps.setMovementLine(moveOk ? `Move sent for ${objectModel.label}.` : "Socket unavailable.");
        deps.setStageStatus(moveOk ? `Moving ${objectModel.label}.` : "Socket unavailable.");
        deps.closeContextMenu();
      }

      if (action === "set-map") {
        deps.openMapEditor();
        deps.closeContextMenu();
        return;
      }

      if (action === "configure-grid") {
        deps.openGridEditor();
        deps.closeContextMenu();
        return;
      }

      if (action === "add-token") {
        deps.selectObject(null, "Selection cleared.");
        deps.openTokenPicker("create", null);
        deps.closeContextMenu();
        return;
      }

      if (action === "scale" && deps.isTokenObject(objectModel)) {
        deps.openTokenEditor(objectModel);
        deps.closeContextMenu();
        return;
      }

      if (action === "hide" || action === "show") {
        const visible = action === "show";
        const type = visible ? "act/reveal_element" : "act/hide_element";
        const sent = deps.sendAction(type, {
          element_id: objectModel.elementId || "",
          element_slug: objectModel.elementSlug || "",
          layer: "audience",
        });
        if (sent) {
          deps.updateLocalObjectModel(objectModel, (model) => {
            model.state = { ...(model.state || {}), visible };
            model.visibility = { ...(model.visibility || {}), visible };
          });
        }
        deps.setStageStatus(sent ? `${visible ? "Showing" : "Hiding"} ${objectModel.label} for the audience...` : "Socket unavailable.");
        deps.closeContextMenu();
        return;
      }

      if (action === "edit" && kind === "card" && typeof deps.openCardEditor === "function") {
        deps.openCardEditor(objectModel);
        deps.closeContextMenu();
        return;
      }

      if (action === "flip" && kind === "card") {
        const nextFace = deps.cardFaceForModel(objectModel) === "back" ? "front" : "back";
        const sent = deps.sendAction("update/index_card", {
          element_id: objectModel.elementId || "",
          element_slug: objectModel.elementSlug || "",
          front_text: objectModel.frontText || "",
          back_text: objectModel.backText || "",
          color: objectModel.color || "#d9c7a6",
          face: nextFace,
        });
        if (sent) {
          deps.updateLocalObjectModel(objectModel, (model) => {
            model.cardFace = nextFace;
            model.source.data = { ...(model.source.data || {}), face: nextFace };
          });
        }
        deps.setStageStatus(sent ? `${objectModel.label} flipped.` : "Socket unavailable.");
        deps.closeContextMenu();
        return;
      }

      if (action === "pin" && kind === "card") {
        const sent = deps.sendAction("update/index_card", {
          element_id: objectModel.elementId || "",
          element_slug: objectModel.elementSlug || "",
          front_text: objectModel.frontText || "",
          back_text: objectModel.backText || "",
          color: objectModel.color || "#d9c7a6",
          pin_mode: "world",
          world_x: Number(objectModel.position?.world_x ?? objectModel.position?.x ?? 0),
          world_y: Number(objectModel.position?.world_y ?? objectModel.position?.y ?? 0),
          screen_x: Number(objectModel.position?.screen_x ?? 0),
          screen_y: Number(objectModel.position?.screen_y ?? 0),
        });
        if (sent) {
          deps.updateLocalCardPinModel(objectModel, (model) => {
            model.source.data = { ...(model.source.data || {}), pin_mode: "world" };
          });
        }
        deps.setStageStatus(sent ? `${objectModel.label} attached to map.` : "Socket unavailable.");
        deps.closeContextMenu();
        return;
      }

      if (action === "unpin" && kind === "card") {
        const sent = deps.sendAction("update/index_card", {
          element_id: objectModel.elementId || "",
          element_slug: objectModel.elementSlug || "",
          front_text: objectModel.frontText || "",
          back_text: objectModel.backText || "",
          color: objectModel.color || "#d9c7a6",
          pin_mode: "overlay",
          screen_x: Number(objectModel.position?.screen_x ?? 0),
          screen_y: Number(objectModel.position?.screen_y ?? 0),
        });
        if (sent) {
          deps.updateLocalCardPinModel(objectModel, (model) => {
            model.source.data = { ...(model.source.data || {}), pin_mode: "overlay" };
          });
        }
        deps.setStageStatus(sent ? `${objectModel.label} detached from the map.` : "Socket unavailable.");
        deps.closeContextMenu();
        return;
      }

      if (action === "lock" || action === "unlock") {
        const locked = action === "lock";
        const sent = deps.sendAction("act/set_element_lock", {
          element_id: objectModel.elementId || "",
          element_slug: objectModel.elementSlug || "",
          locked,
        });
        if (sent) {
          deps.updateLocalObjectModel(objectModel, (model) => {
            model.state = { ...(model.state || {}), locked };
            model.visibility = { ...(model.visibility || {}), locked };
          });
        }
        deps.setStageStatus(sent ? `${objectModel.label} ${locked ? "locked" : "unlocked"}.` : "Socket unavailable.");
        deps.closeContextMenu();
        return;
      }

      if ((action === "snap-to-grid" || action === "free-placement") && deps.isTokenObject(objectModel)) {
        const snapMode = action === "snap-to-grid" ? "grid" : "free";
        const snappedPoint = snapMode === "grid"
          ? deps.tokenPlacementPointForCreate(deps.currentDisplayedPointForModel(objectModel), objectModel, snapMode)
          : deps.currentDisplayedPointForModel(objectModel);
        const sent = deps.sendAction("update/token", {
          element_id: objectModel.elementId || "",
          element_slug: objectModel.elementSlug || "",
          venue_slug: "the-cave",
          layer: "stage",
          x: Number(snappedPoint?.x ?? objectModel.position?.x ?? 0),
          y: Number(snappedPoint?.y ?? objectModel.position?.y ?? 0),
          order: Number(objectModel.position?.order ?? 0),
          snap_mode: snapMode,
        });
        if (sent) {
          deps.updateTokenLocalModel(objectModel, (model) => {
            model.snapMode = snapMode;
            model.gridRelative = snapMode === "grid";
          });
        }
        deps.setStageStatus(sent ? `${objectModel.label} set to ${snapMode}.` : "Socket unavailable.");
        deps.closeContextMenu();
        return;
      }

      if ((action === "move-director-layer" || action === "move-public-layer") && deps.isTokenObject(objectModel)) {
        const tokenLayer = action === "move-director-layer" ? "director" : "public";
        const sent = deps.sendAction("update/token", {
          element_id: objectModel.elementId || "",
          element_slug: objectModel.elementSlug || "",
          venue_slug: "the-cave",
          layer: "stage",
          x: Number(objectModel.position?.x ?? 0),
          y: Number(objectModel.position?.y ?? 0),
          order: Number(objectModel.position?.order ?? 0),
          token_layer: tokenLayer,
        });
        if (sent) {
          deps.updateTokenLocalModel(objectModel, (model) => {
            model.tokenLayer = tokenLayer;
          });
        }
        deps.setStageStatus(sent ? `${objectModel.label} moved.` : "Socket unavailable.");
        deps.closeContextMenu();
        return;
      }

      if (action === "duplicate") {
        const duplicatePlacement = deps.isTokenObject(objectModel)
          ? deps.tokenDuplicatePlacementForModel(objectModel)
          : deps.cardDuplicatePlacementForModel(objectModel);
        if (!duplicatePlacement) {
          deps.setStageStatus("Socket unavailable.");
          deps.closeContextMenu();
          return;
        }
        const sent = deps.sendAction("act/duplicate_element", {
          element_id: objectModel.elementId || "",
          element_slug: objectModel.elementSlug || "",
          venue_slug: "the-cave",
          layer: "stage",
          x: duplicatePlacement.x,
          y: duplicatePlacement.y,
          order: Number(objectModel.position?.order ?? 0),
          pin_mode: duplicatePlacement.pinMode || "overlay",
          world_x: duplicatePlacement.world_x,
          world_y: duplicatePlacement.world_y,
          screen_x: duplicatePlacement.screen_x,
          screen_y: duplicatePlacement.screen_y,
        });
        deps.setStageStatus(sent ? `Duplicating ${objectModel.label}.` : "Socket unavailable.");
        deps.closeContextMenu();
      }

      if (action === "remove") {
        const sent = deps.sendAction("act/remove_element", {
          element_id: objectModel.elementId || "",
          element_slug: objectModel.elementSlug || "",
          venue_slug: "the-cave",
          layer: "stage",
        });
        if (sent) {
          deps.removeLocalObject(objectModel);
          deps.selectObject(null, "Selection cleared.");
          deps.renderPixiScene?.();
          deps.syncSelectedActions?.();
          deps.syncTokenEditorWithSelection?.();
          if (!syncCurrentObjectsFromProjectedState()) {
            deps.refreshWorld?.();
          }
        }
        deps.setStageStatus(sent ? `Removing ${objectModel.label} from the stage...` : "Socket unavailable.");
        deps.closeContextMenu();
      }
    }

    function resolveContextMenuTargetFromClient(clientX, clientY) {
      const screenPoint = deps.stageScreenPointFromClient(clientX, clientY);
      const stagePoint = deps.stagePointFromClient(clientX, clientY);
      const objectModel = deps.hitTestContextMenuTarget(screenPoint, stagePoint) || deps.stageContextMenuModel();
      return { objectModel, stagePoint, screenPoint };
    }

    function openContextMenu(event, objectModel) {
      if (!deps.contextMenu() || !objectModel) return;
      deps.setContextMenuTarget(objectModel);
      const clientPoint = deps.eventClientPoint(event);
      const stagePoint = deps.stagePlacementFromEvent(event);
      const screenPoint = deps.stageScreenPointFromClient(clientPoint.clientX, clientPoint.clientY);
      if (objectModel.kind === "stage" || objectModel.kind === "fire" || (objectModel.live && deps.isLiveStageObject(objectModel))) {
        deps.setStagePlacementCandidate(stagePoint, screenPoint);
      }
      const items = deps.resolveStageObjectActions(objectModel, deps.resolveStageObjectActionsContext());
      deps.renderContextMenu(items);
      deps.positionContextMenu(clientPoint.clientX || 0, clientPoint.clientY || 0);
    }

    function closeContextMenu() {
      deps.closeContextMenuUI();
    }

    function handleNativeStageContextMenu(event) {
      if (!event) return;
      if (deps.wasContextMenuHandled?.(event)) return;
      if (deps.cameraControlsContains?.(event.target)) return;
      if (!deps.isStageEvent(event)) return;
      if (event.type !== "contextmenu" && !deps.isSecondaryPointerEvent(event)) return;
      event.preventDefault();
      event.stopPropagation();
      event.stopImmediatePropagation?.();
      const clientPoint = deps.eventClientPoint(event);
      const resolved = resolveContextMenuTargetFromClient(clientPoint.clientX, clientPoint.clientY);
      const nativeEvent = event?.originalEvent || event?.data?.originalEvent || event;
      deps.markContextMenuHandled(nativeEvent);
      openContextMenu(nativeEvent, resolved.objectModel);
    }

    function openResolvedContextMenu(event) {
      if (!event || !deps.contextMenu() || !deps.pixiApp()) return;
      const clientPoint = deps.eventClientPoint(event);
      const resolved = resolveContextMenuTargetFromClient(clientPoint.clientX, clientPoint.clientY);
      const nativeEvent = event?.data?.originalEvent || event?.originalEvent || event;
      deps.markContextMenuHandled(nativeEvent);
      openContextMenu(nativeEvent, resolved.objectModel);
    }

    return {
      performStageObjectAction,
      resolveContextMenuTargetFromClient,
      openContextMenu,
      closeContextMenu,
      handleNativeStageContextMenu,
      openResolvedContextMenu,
    };
  }

  return { createActionRouter };
});
