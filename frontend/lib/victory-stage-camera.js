(function () {
  function clamp(value, min, max, fallback) {
    const parsed = Number(value);
    if (!Number.isFinite(parsed)) return fallback;
    return Math.max(min, Math.min(max, parsed));
  }

  function normalizeBounds(bounds, fallbackWidth = 1, fallbackHeight = 1) {
    const source = bounds || {};
    return {
      x: Number.isFinite(Number(source.x)) ? Number(source.x) : 0,
      y: Number.isFinite(Number(source.y)) ? Number(source.y) : 0,
      width: Math.max(1, Number(source.width || fallbackWidth || 1)),
      height: Math.max(1, Number(source.height || fallbackHeight || 1)),
    };
  }

  function safeStorageKey(value) {
    return String(value || "")
      .trim()
      .toLowerCase()
      .replace(/[^a-z0-9._-]+/g, "-")
      .replace(/^-+|-+$/g, "")
      || "browser";
  }

  function readJSON(key) {
    try {
      const raw = window.localStorage.getItem(key);
      if (!raw) return null;
      return JSON.parse(raw);
    } catch (error) {
      return null;
    }
  }

  function writeJSON(key, value) {
    try {
      window.localStorage.setItem(key, JSON.stringify(value));
      return true;
    } catch (error) {
      return false;
    }
  }

  function mount(options = {}) {
    const stageElement = options.stageElement instanceof Element ? options.stageElement : null;
    const worldLayer = options.worldLayer && typeof options.worldLayer.addChild === "function" ? options.worldLayer : null;
    const getPlayableBounds = typeof options.getPlayableBounds === "function" ? options.getPlayableBounds : () => ({ x: 0, y: 0, width: 1, height: 1 });
    // Bug fix: worldBounds used to be a one-time snapshot (options.worldBounds,
    // captured once at mount and only ever changed by an explicit setWorldBounds
    // call that nothing in the codebase actually makes) while playableBounds is
    // re-read live via getPlayableBounds() on every clampState(). Whenever the
    // live value drifted from that frozen snapshot (window resize, layout
    // settling after mount), clampState()'s pan clamp fought the caller's
    // requested pan against a stale boundary -- invisible at normal zoom, but a
    // visible X/Y drift for Kernel 87 Detail View's near-max-zoom single-cell
    // regions. getWorldBounds mirrors getPlayableBounds's live-function pattern
    // so both boundaries always agree.
    const getWorldBoundsOption = typeof options.getWorldBounds === "function" ? options.getWorldBounds : null;
    const onChange = typeof options.onChange === "function" ? options.onChange : () => {};
    const minZoom = clamp(options.minZoom, 0.01, 10, 0.7);
    const maxZoom = clamp(options.maxZoom, 0.01, 10, 4);
    const edgeBand = clamp(options.edgeBand, 8, 120, 32);
    const edgeMaxSpeed = clamp(options.edgeMaxSpeed, 1, 2000, 18);
    const wheelZoomFactor = clamp(options.wheelZoomFactor, 0.001, 0.02, 0.0018);
    const scope = safeStorageKey(options.userScope || options.userId || options.scope || "browser");
    const venueSlug = safeStorageKey(options.venueSlug || "first-theater");
    const storageKey = `victory.camera.v1.${scope}.${venueSlug}`;

    const state = {
      panX: 0,
      panY: 0,
      zoomRelativeToFit: 1,
      activeMapId: "",
      updatedAt: "",
    };

    let worldBounds = normalizeBounds(options.worldBounds, 1, 1);
    let playableBounds = normalizeBounds(getPlayableBounds(), 1, 1);
    let pointer = null;
    let middleDragging = false;
    let middleDragPointerId = null;
    let lastDragPoint = null;
    let edgeLoopHandle = null;
    let animationFrameHandle = null;
    let animationToken = 0;
    let disposed = false;
    let interactionLocked = false;

    function emitChange() {
      onChange(getView());
    }

    function persist() {
      const payload = {
        venue_slug: venueSlug,
        active_map_id: state.activeMapId || "",
        pan_x: state.panX,
        pan_y: state.panY,
        zoom_relative_to_fit: state.zoomRelativeToFit,
        updated_at: new Date().toISOString(),
      };
      state.updatedAt = payload.updated_at;
      writeJSON(storageKey, payload);
    }

    function applyTransform() {
      if (!worldLayer) return;
      worldLayer.position.set(state.panX, state.panY);
      worldLayer.scale.set(state.zoomRelativeToFit);
    }

    function clampState() {
      playableBounds = normalizeBounds(getPlayableBounds(), playableBounds.width, playableBounds.height);
      worldBounds = normalizeBounds(getWorldBoundsOption ? getWorldBoundsOption() : worldBounds, worldBounds.width, worldBounds.height);

      const scaledWidth = worldBounds.width * state.zoomRelativeToFit;
      const scaledHeight = worldBounds.height * state.zoomRelativeToFit;
      const viewLeft = playableBounds.x;
      const viewTop = playableBounds.y;
      const viewRight = playableBounds.x + playableBounds.width;
      const viewBottom = playableBounds.y + playableBounds.height;

      const leftAtPanZero = worldBounds.x * state.zoomRelativeToFit;
      const topAtPanZero = worldBounds.y * state.zoomRelativeToFit;
      const minPanX = viewRight - ((worldBounds.x + worldBounds.width) * state.zoomRelativeToFit);
      const maxPanX = viewLeft - leftAtPanZero;
      const minPanY = viewBottom - ((worldBounds.y + worldBounds.height) * state.zoomRelativeToFit);
      const maxPanY = viewTop - topAtPanZero;

      if (scaledWidth <= playableBounds.width) {
        const centeredLeft = viewLeft + ((playableBounds.width - scaledWidth) / 2);
        state.panX = centeredLeft - leftAtPanZero;
      } else {
        state.panX = clamp(state.panX, minPanX, maxPanX, state.panX);
      }

      if (scaledHeight <= playableBounds.height) {
        const centeredTop = viewTop + ((playableBounds.height - scaledHeight) / 2);
        state.panY = centeredTop - topAtPanZero;
      } else {
        state.panY = clamp(state.panY, minPanY, maxPanY, state.panY);
      }
    }

    function sync() {
      if (disposed) return;
      clampState();
      applyTransform();
      emitChange();
      persist();
    }

    function cancelAnimation() {
      if (animationFrameHandle !== null) {
        window.cancelAnimationFrame(animationFrameHandle);
        animationFrameHandle = null;
      }
      animationToken += 1;
    }

    function restore() {
      const stored = readJSON(storageKey);
      if (!stored) {
        sync();
        return;
      }

      state.activeMapId = String(stored.active_map_id || "");
      state.panX = Number.isFinite(Number(stored.pan_x)) ? Number(stored.pan_x) : 0;
      state.panY = Number.isFinite(Number(stored.pan_y)) ? Number(stored.pan_y) : 0;
      state.zoomRelativeToFit = clamp(stored.zoom_relative_to_fit, minZoom, maxZoom, 1);
      state.updatedAt = String(stored.updated_at || "");
      sync();
    }

    function scheduleEdgeScroll() {
      if (edgeLoopHandle || disposed) return;
      edgeLoopHandle = window.requestAnimationFrame(stepEdgeScroll);
    }

    function stopEdgeScroll() {
      if (edgeLoopHandle) {
        window.cancelAnimationFrame(edgeLoopHandle);
        edgeLoopHandle = null;
      }
    }

    function edgeVelocity() {
      if (!pointer) return { x: 0, y: 0 };
      const x = pointer.x;
      const y = pointer.y;
      const left = playableBounds.x;
      const top = playableBounds.y;
      const right = playableBounds.x + playableBounds.width;
      const bottom = playableBounds.y + playableBounds.height;

      if (x < left || x > right || y < top || y > bottom) {
        return { x: 0, y: 0 };
      }

      let vx = 0;
      let vy = 0;
      const leftDist = x - left;
      const rightDist = right - x;
      const topDist = y - top;
      const bottomDist = bottom - y;

      if (leftDist <= edgeBand) {
        const amount = Math.max(0, 1 - (leftDist / edgeBand));
        vx += edgeMaxSpeed * amount * amount;
      }
      if (rightDist <= edgeBand) {
        const amount = Math.max(0, 1 - (rightDist / edgeBand));
        vx -= edgeMaxSpeed * amount * amount;
      }
      if (topDist <= edgeBand) {
        const amount = Math.max(0, 1 - (topDist / edgeBand));
        vy += edgeMaxSpeed * amount * amount;
      }
      if (bottomDist <= edgeBand) {
        const amount = Math.max(0, 1 - (bottomDist / edgeBand));
        vy -= edgeMaxSpeed * amount * amount;
      }

      return { x: vx, y: vy };
    }

    function stepEdgeScroll() {
      edgeLoopHandle = null;
      if (disposed || middleDragging) {
        stopEdgeScroll();
        return;
      }

      const velocity = edgeVelocity();
      if (!velocity.x && !velocity.y) {
        stopEdgeScroll();
        return;
      }

      state.panX += velocity.x;
      state.panY += velocity.y;
      clampState();
      applyTransform();
      emitChange();
      persist();
      scheduleEdgeScroll();
    }

    function setPointerFromEvent(event) {
      if (!event) return;
      const rect = stageElement?.getBoundingClientRect?.();
      if (!rect) return;
      pointer = {
        x: Number(event.clientX || 0),
        y: Number(event.clientY || 0),
        stageX: Number(event.clientX || 0) - rect.left,
        stageY: Number(event.clientY || 0) - rect.top,
      };
    }

    function screenToWorld(screenX, screenY) {
      return {
        x: ((Number(screenX || 0) - state.panX) / state.zoomRelativeToFit),
        y: ((Number(screenY || 0) - state.panY) / state.zoomRelativeToFit),
      };
    }

    function worldToScreen(worldX, worldY) {
      return {
        x: (Number(worldX || 0) * state.zoomRelativeToFit) + state.panX,
        y: (Number(worldY || 0) * state.zoomRelativeToFit) + state.panY,
      };
    }

    function zoomAt(screenX, screenY, nextZoom) {
      const before = screenToWorld(screenX, screenY);
      state.zoomRelativeToFit = clamp(nextZoom, minZoom, maxZoom, state.zoomRelativeToFit);
      state.panX = Number(screenX || 0) - (before.x * state.zoomRelativeToFit);
      state.panY = Number(screenY || 0) - (before.y * state.zoomRelativeToFit);
      sync();
    }

    function fit(activeMapId = state.activeMapId) {
      cancelAnimation();
      state.activeMapId = String(activeMapId || "");
      state.panX = 0;
      state.panY = 0;
      state.zoomRelativeToFit = 1;
      sync();
    }

    function setView(next = {}, options = {}) {
      cancelAnimation();
      if (next.activeMapId !== undefined) {
        state.activeMapId = String(next.activeMapId || "");
      }
      if (next.zoomRelativeToFit !== undefined) {
        state.zoomRelativeToFit = clamp(next.zoomRelativeToFit, minZoom, maxZoom, 1);
      }
      if (next.panX !== undefined) {
        state.panX = Number(next.panX || 0);
      }
      if (next.panY !== undefined) {
        state.panY = Number(next.panY || 0);
      }
      if (options.fit === true) {
        state.panX = 0;
        state.panY = 0;
        state.zoomRelativeToFit = 1;
      }
      sync();
    }

    function animateToView(next = {}, options = {}) {
      cancelAnimation();
      const duration = clamp(options.duration, 0, 5000, 250);
      if (duration <= 0) {
        setView(next, options);
        return;
      }

      const start = getView();
      const target = {
        activeMapId: next.activeMapId !== undefined ? String(next.activeMapId || "") : start.activeMapId,
        panX: next.panX !== undefined ? Number(next.panX || 0) : start.panX,
        panY: next.panY !== undefined ? Number(next.panY || 0) : start.panY,
        zoomRelativeToFit: next.zoomRelativeToFit !== undefined ? clamp(next.zoomRelativeToFit, minZoom, maxZoom, start.zoomRelativeToFit) : start.zoomRelativeToFit,
      };
      const token = animationToken + 1;
      animationToken = token;
      const startedAt = performance.now();

      function frame(now) {
        if (disposed || token !== animationToken) {
          return;
        }

        const progress = clamp((now - startedAt) / duration, 0, 1, 0);
        const eased = 1 - Math.pow(1 - progress, 3);
        state.activeMapId = target.activeMapId;
        state.panX = start.panX + ((target.panX - start.panX) * eased);
        state.panY = start.panY + ((target.panY - start.panY) * eased);
        state.zoomRelativeToFit = start.zoomRelativeToFit + ((target.zoomRelativeToFit - start.zoomRelativeToFit) * eased);
        clampState();
        applyTransform();
        emitChange();

        if (progress < 1) {
          animationFrameHandle = window.requestAnimationFrame(frame);
          return;
        }

        animationFrameHandle = null;
        persist();
      }

      animationFrameHandle = window.requestAnimationFrame(frame);
    }

    function setWorldBounds(nextBounds) {
      worldBounds = normalizeBounds(nextBounds, worldBounds.width, worldBounds.height);
      sync();
    }

    function setPlayableBounds(nextBounds) {
      playableBounds = normalizeBounds(nextBounds, playableBounds.width, playableBounds.height);
      sync();
    }

    function setActiveMapId(nextActiveMapId, options = {}) {
      const next = String(nextActiveMapId || "");
      if (state.activeMapId === next && !options.reset) {
        return;
      }
      state.activeMapId = next;
      if (options.reset !== false) {
        state.panX = 0;
        state.panY = 0;
        state.zoomRelativeToFit = 1;
      }
      sync();
    }

    function getView() {
      return {
        activeMapId: state.activeMapId,
        panX: state.panX,
        panY: state.panY,
        zoomRelativeToFit: state.zoomRelativeToFit,
        updatedAt: state.updatedAt,
      };
    }

    function serialize() {
      return {
        venue_slug: venueSlug,
        active_map_id: state.activeMapId,
        pan_x: state.panX,
        pan_y: state.panY,
        zoom_relative_to_fit: state.zoomRelativeToFit,
        updated_at: state.updatedAt,
      };
    }

    function handleWheel(event) {
      if (disposed) return;
      if (interactionLocked) return;
      if (!stageElement || !stageElement.contains(event.target)) return;
      event.preventDefault();
      const delta = Number(event.deltaY || 0);
      const factor = Math.exp(-delta * wheelZoomFactor);
      zoomAt(event.clientX, event.clientY, state.zoomRelativeToFit * factor);
    }

    function beginMiddleDrag(event) {
      if (interactionLocked) return;
      if (!stageElement || event.button !== 1) return;
      event.preventDefault();
      event.stopPropagation();
      setPointerFromEvent(event);
      middleDragging = true;
      middleDragPointerId = event.pointerId;
      lastDragPoint = { x: Number(event.clientX || 0), y: Number(event.clientY || 0) };
      stageElement.style.cursor = "grabbing";
      stopEdgeScroll();
    }

    function movePointer(event) {
      if (disposed) return;
      setPointerFromEvent(event);

      if (interactionLocked) {
        stopEdgeScroll();
        return;
      }

      if (middleDragging) {
        const nextPoint = { x: Number(event.clientX || 0), y: Number(event.clientY || 0) };
        const deltaX = nextPoint.x - lastDragPoint.x;
        const deltaY = nextPoint.y - lastDragPoint.y;
        lastDragPoint = nextPoint;
        state.panX += deltaX;
        state.panY += deltaY;
        clampState();
        applyTransform();
        emitChange();
        persist();
        return;
      }

      const velocity = edgeVelocity();
      if (velocity.x || velocity.y) {
        scheduleEdgeScroll();
      } else {
        stopEdgeScroll();
      }
    }

    function endMiddleDrag() {
      if (!middleDragging) return;
      middleDragging = false;
      middleDragPointerId = null;
      lastDragPoint = null;
      if (stageElement) {
        stageElement.style.cursor = "";
      }
      clampState();
      applyTransform();
      emitChange();
      persist();
    }

    function handlePointerLeave(event) {
      if (event?.pointerId !== undefined && middleDragPointerId !== null && event.pointerId !== middleDragPointerId) {
        return;
      }
      pointer = null;
      stopEdgeScroll();
    }

    function handleBlur() {
      endMiddleDrag();
      pointer = null;
      stopEdgeScroll();
    }

    if (stageElement) {
      stageElement.addEventListener("wheel", handleWheel, { passive: false });
      stageElement.addEventListener("pointerdown", beginMiddleDrag, { passive: false });
      stageElement.addEventListener("pointermove", movePointer, { passive: true });
      stageElement.addEventListener("pointerup", endMiddleDrag, { passive: true });
      stageElement.addEventListener("pointercancel", endMiddleDrag, { passive: true });
      stageElement.addEventListener("pointerleave", handlePointerLeave, { passive: true });
      stageElement.addEventListener("contextmenu", (event) => {
        if (event.button === 1 || event.pointerType === "mouse") {
          event.preventDefault();
        }
      });
      window.addEventListener("blur", handleBlur);
      window.addEventListener("pointerup", endMiddleDrag, true);
      window.addEventListener("pointercancel", endMiddleDrag, true);
    }

    function setInteractionLocked(nextLocked) {
      interactionLocked = Boolean(nextLocked);
      if (interactionLocked) {
        endMiddleDrag();
        stopEdgeScroll();
        if (stageElement) {
          stageElement.style.cursor = "";
        }
      }
    }

    restore();

    return {
      fit,
      getView,
      serialize,
      setView,
      setWorldBounds,
      setPlayableBounds,
      setActiveMapId,
      setInteractionLocked,
      screenToWorld,
      worldToScreen,
      zoomAt,
      destroy() {
        disposed = true;
        stopEdgeScroll();
        if (stageElement) {
          stageElement.removeEventListener("wheel", handleWheel);
          stageElement.removeEventListener("pointerdown", beginMiddleDrag);
          stageElement.removeEventListener("pointermove", movePointer);
          stageElement.removeEventListener("pointerup", endMiddleDrag);
          stageElement.removeEventListener("pointercancel", endMiddleDrag);
          stageElement.removeEventListener("pointerleave", handlePointerLeave);
        }
        window.removeEventListener("blur", handleBlur);
        window.removeEventListener("pointerup", endMiddleDrag, true);
        window.removeEventListener("pointercancel", endMiddleDrag, true);
        cancelAnimation();
      },
    };
  }

  window.VictoryStageCamera = {
    mount,
  };
})();
