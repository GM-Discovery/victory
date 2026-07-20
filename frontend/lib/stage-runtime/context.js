(function (root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  }
  root.VictoryStageContext = factory();
})(typeof globalThis !== "undefined" ? globalThis : window, function () {
  function createContextInteractionHelpers(deps = {}) {
    const getCurrentObjects = typeof deps.getCurrentObjects === "function" ? deps.getCurrentObjects : () => [];
    const getCurrentNodeMap = typeof deps.getCurrentNodeMap === "function" ? deps.getCurrentNodeMap : () => new Map();
    const getStageShell = typeof deps.getStageShell === "function" ? deps.getStageShell : () => null;
    const getStageHost = typeof deps.getStageHost === "function" ? deps.getStageHost : () => null;
    const getPixiApp = typeof deps.getPixiApp === "function" ? deps.getPixiApp : () => null;
    const getStageCamera = typeof deps.getStageCamera === "function" ? deps.getStageCamera : () => null;
    const getLastStagePoint = typeof deps.getLastStagePoint === "function" ? deps.getLastStagePoint : () => null;
    const setLastStagePoint = typeof deps.setLastStagePoint === "function" ? deps.setLastStagePoint : () => {};
    const setPointerLine = typeof deps.setPointerLine === "function" ? deps.setPointerLine : () => {};
    const setSmokeLine = typeof deps.setSmokeLine === "function" ? deps.setSmokeLine : () => {};
    const currentSelectionSummary = typeof deps.currentSelectionSummary === "function" ? deps.currentSelectionSummary : () => "none";
    const stageContextMenuModel = typeof deps.stageContextMenuModel === "function" ? deps.stageContextMenuModel : () => null;
    const objectKind = typeof deps.objectKind === "function" ? deps.objectKind : () => "unknown";
    const stageScreenPointFromClient = typeof deps.stageScreenPointFromClient === "function" ? deps.stageScreenPointFromClient : (clientX, clientY) => ({ x: Number(clientX || 0), y: Number(clientY || 0) });
    const stagePointFromClient = typeof deps.stagePointFromClient === "function" ? deps.stagePointFromClient : (clientX, clientY) => ({ x: Number(clientX || 0), y: Number(clientY || 0) });
    const setStagePlacementCandidate = typeof deps.setStagePlacementCandidate === "function" ? deps.setStagePlacementCandidate : () => {};
    const setContextMenuTarget = typeof deps.setContextMenuTarget === "function" ? deps.setContextMenuTarget : () => {};
    const closeContextMenuUI = typeof deps.closeContextMenuUI === "function" ? deps.closeContextMenuUI : () => {};
    const smokeMode = Boolean(deps.smokeMode);

    function isSecondaryPointerEvent(event) {
      const button = Number(event?.button ?? event?.data?.button ?? -1);
      const buttons = Number(event?.buttons ?? event?.data?.buttons ?? 0);
      return button === 2 || (buttons & 2) === 2;
    }

    function markContextMenuHandled(event) {
      if (!event) return;
      event.__stageContextMenuHandled = true;
      if (event.data?.originalEvent) {
        event.data.originalEvent.__stageContextMenuHandled = true;
      }
    }

    function wasContextMenuHandled(event) {
      return !!(event?.__stageContextMenuHandled || event?.data?.originalEvent?.__stageContextMenuHandled);
    }

    function cancelContextMenuEvent(event) {
      event?.stopPropagation?.();
      event?.preventDefault?.();
      event?.stopImmediatePropagation?.();
      event?.data?.originalEvent?.preventDefault?.();
      event?.data?.originalEvent?.stopPropagation?.();
      event?.data?.originalEvent?.stopImmediatePropagation?.();
    }

    function eventClientPoint(event) {
      if (!event) {
        return { clientX: 0, clientY: 0 };
      }

      const originalClientX = Number(event.data?.originalEvent?.clientX ?? event.originalEvent?.clientX ?? Number.NaN);
      const originalClientY = Number(event.data?.originalEvent?.clientY ?? event.originalEvent?.clientY ?? Number.NaN);
      if (Number.isFinite(originalClientX) && Number.isFinite(originalClientY)) {
        return { clientX: originalClientX, clientY: originalClientY };
      }

      const clientX = Number(event.clientX ?? event.x ?? 0);
      const clientY = Number(event.clientY ?? event.y ?? 0);
      if (Number.isFinite(clientX) && Number.isFinite(clientY) && (clientX !== 0 || clientY !== 0)) {
        return { clientX, clientY };
      }

      const stagePoint = event.data?.global || event.global || null;
      const pixiApp = getPixiApp();
      const stageHost = getStageHost();
      if (stagePoint && pixiApp?.renderer) {
        const rect = stageHost?.getBoundingClientRect();
        const screenWidth = Number(pixiApp.renderer.screen?.width || rect?.width || 1);
        const screenHeight = Number(pixiApp.renderer.screen?.height || rect?.height || 1);
        const domX = rect ? rect.left + (Number(stagePoint.x || 0) * (rect.width / screenWidth)) : Number(stagePoint.x || 0);
        const domY = rect ? rect.top + (Number(stagePoint.y || 0) * (rect.height / screenHeight)) : Number(stagePoint.y || 0);
        return { clientX: domX, clientY: domY };
      }

      return { clientX: 0, clientY: 0 };
    }

    function updatePointerReadout(screenX, screenY, stagePoint, note = "") {
      if (stagePoint) {
        setLastStagePoint(stagePoint);
      }
      const resolved = stagePoint || getLastStagePoint() || { x: 0, y: 0 };
      const screenLabel = `Screen ${Math.round(screenX || 0)}, ${Math.round(screenY || 0)}`;
      const stageLabel = `Stage ${resolved.x}, ${resolved.y}`;
      setPointerLine(`${screenLabel} | ${stageLabel}`);
      if (smokeMode) {
        const prefix = note ? `${note} · ` : "";
        setSmokeLine(`${prefix}${stageLabel} · selection ${currentSelectionSummary()}`);
      }
    }

    function stageScreenPointFromClientLocal(clientX, clientY) {
      const stageShell = getStageShell();
      const rect = stageShell?.getBoundingClientRect?.();
      if (!rect) {
        return { x: Number(clientX || 0), y: Number(clientY || 0) };
      }
      return {
        x: Math.round(Number(clientX || 0) - Number(rect.left || 0)),
        y: Math.round(Number(clientY || 0) - Number(rect.top || 0)),
      };
    }

    function stagePointFromClientLocal(clientX, clientY) {
      const stageCamera = getStageCamera();
      const screenPoint = stageScreenPointFromClientLocal(clientX, clientY);
      if (stageCamera?.screenToWorld) {
        const worldPoint = stageCamera.screenToWorld(screenPoint.x, screenPoint.y);
        if (worldPoint) {
          return {
            x: Math.round(worldPoint.x),
            y: Math.round(worldPoint.y),
          };
        }
      }
      return {
        x: Math.round(Number(screenPoint.x || 0)),
        y: Math.round(Number(screenPoint.y || 0)),
      };
    }

    function hitTestContextMenuTarget(screenPoint) {
      const currentObjects = getCurrentObjects();
      const currentNodeMap = getCurrentNodeMap();
      for (let index = currentObjects.length - 1; index >= 0; index -= 1) {
        const model = currentObjects[index];
        const node = currentNodeMap.get(model.key);
        const bounds = node?.container?.getBounds?.();
        if (bounds?.contains?.(screenPoint.x, screenPoint.y)) {
          return model;
        }

        const localPoint = node?.container?.toLocal && typeof PIXI !== "undefined"
          ? node.container.toLocal(new PIXI.Point(screenPoint.x, screenPoint.y))
          : null;
        if (!localPoint) continue;

        const kind = objectKind(model);
        const width = kind === "fire" ? 124 : 200;
        const height = kind === "fire" ? 150 : 140;
        const hitSlop = 8;
        if (
          localPoint.x >= -hitSlop &&
          localPoint.x <= width + hitSlop &&
          localPoint.y >= -hitSlop &&
          localPoint.y <= height + hitSlop
        ) {
          return model;
        }
      }
      return null;
    }

    function resolveContextMenuTargetFromClient(clientX, clientY) {
      const screenPoint = stageScreenPointFromClientLocal(clientX, clientY);
      const stagePoint = stagePointFromClientLocal(clientX, clientY);
      const objectModel = hitTestContextMenuTarget(screenPoint, stagePoint) || stageContextMenuModel();
      return { objectModel, stagePoint, screenPoint };
    }

    function openContextMenuTarget(event, objectModel) {
      if (!objectModel) return;
      setContextMenuTarget(objectModel);
      const clientPoint = eventClientPoint(event);
      const stagePoint = stagePointFromClientLocal(clientPoint.clientX, clientPoint.clientY);
      const screenPoint = stageScreenPointFromClientLocal(clientPoint.clientX, clientPoint.clientY);
      if (objectModel.kind === "stage" || objectModel.kind === "fire" || (objectModel.live && objectKind(objectModel) !== "unknown")) {
        setStagePlacementCandidate(stagePoint, screenPoint);
      }
      return { clientPoint, stagePoint, screenPoint };
    }

    function closeContextMenu() {
      closeContextMenuUI();
    }

    function isStageEvent(event) {
      const target = event?.target;
      const composedPath = typeof event?.composedPath === "function" ? event.composedPath() : [];
      const stageShell = getStageShell();
      const stageHost = getStageHost();
      const pixiApp = getPixiApp();
      return target === stageHost || target === stageShell || stageShell?.contains?.(target) || stageHost?.contains?.(target) || composedPath.includes(stageHost) || composedPath.includes(stageShell) || composedPath.includes(pixiApp?.view);
    }

    function handleNativeStageContextMenu(event) {
      if (!event) return;
      if (wasContextMenuHandled(event)) return;
      if (!isStageEvent(event)) return;
      if (event.type !== "contextmenu" && !isSecondaryPointerEvent(event)) {
        return;
      }
      event.preventDefault();
      event.stopPropagation();
      event.stopImmediatePropagation?.();
      const clientPoint = eventClientPoint(event);
      const resolved = resolveContextMenuTargetFromClient(clientPoint.clientX, clientPoint.clientY);
      const nativeEvent = event?.originalEvent || event?.data?.originalEvent || event;
      markContextMenuHandled(nativeEvent);
      openContextMenuTarget(nativeEvent, resolved.objectModel);
    }

    function openResolvedContextMenu(event) {
      if (!event || !getStageShell()) return;
      const clientPoint = eventClientPoint(event);
      const resolved = resolveContextMenuTargetFromClient(clientPoint.clientX, clientPoint.clientY);
      const nativeEvent = event?.data?.originalEvent || event?.originalEvent || event;
      markContextMenuHandled(nativeEvent);
      openContextMenuTarget(nativeEvent, resolved.objectModel);
    }

    return {
      isSecondaryPointerEvent,
      markContextMenuHandled,
      wasContextMenuHandled,
      cancelContextMenuEvent,
      eventClientPoint,
      updatePointerReadout,
      stageScreenPointFromClient: stageScreenPointFromClientLocal,
      stagePointFromClient: stagePointFromClientLocal,
      hitTestContextMenuTarget,
      resolveContextMenuTargetFromClient,
      openContextMenuTarget,
      closeContextMenu,
      handleNativeStageContextMenu,
      openResolvedContextMenu,
      isStageEvent,
    };
  }

  return { createContextInteractionHelpers };
});
