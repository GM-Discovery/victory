// Kernel 91: the shared guided-tour overlay engine. One implementation for
// every page with tour content (campus map, Catharsis, Director's Chair,
// ...) rather than a per-venue copy -- see kernel-91 S4/S6, which require
// the same dim/spotlight/arrow/click-gating behavior everywhere. This
// module is a dumb renderer: it never decides eligibility itself, it only
// shows what /api/tours/state or /api/tours/{key}/replay hand it
// (kernel-91 S46 -- the authority boundary lives entirely server-side).
(function () {
  const state = {
    targets: Object.create(null),
    active: null, // { definition, stepIndex, replay }
    els: null, // DOM refs for the currently-mounted overlay
    resizeObserver: null,
    mutationObserver: null,
    positionScheduled: false,
    targetClickHandler: null,
    targetClickEl: null,
  };

  function apiFetch(url, options) {
    return fetch(url, Object.assign({ credentials: "include" }, options || {})).then((res) =>
      res.json().then((payload) => ({ ok: res.ok, status: res.status, payload })),
    );
  }

  // A click-gated step's target is often a real navigation (Audition Hall,
  // Trailer) that starts the instant it's clicked. An ordinary fetch() can
  // be aborted mid-flight by that navigation, silently losing the write and
  // restarting the tour on the user's next visit -- a real bug in
  // production. navigator.sendBeacon is built for exactly this: the browser
  // guarantees delivery even if the page unloads immediately after the call
  // returns. credentials (the session cookie) are included automatically
  // for a same-origin beacon, same as any other same-origin request.
  function beaconPost(url, body) {
    const payload = JSON.stringify(body || {});
    if (navigator.sendBeacon) {
      const blob = new Blob([payload], { type: "application/json" });
      if (navigator.sendBeacon(url, blob)) return;
    }
    // Fallback for browsers without sendBeacon: keepalive asks the browser
    // to let the request outlive the document, best-effort.
    fetch(url, {
      method: "POST",
      credentials: "include",
      keepalive: true,
      headers: { "Content-Type": "application/json" },
      body: payload,
    }).catch(() => {});
  }

  function resolveTarget(ref) {
    const resolver = state.targets[ref];
    if (typeof resolver !== "function") return null;
    try {
      return resolver() || null;
    } catch (error) {
      console.error("VictoryTourEngine: target resolver threw", ref, error);
      return null;
    }
  }

  // The dimmed background is four rectangles framing the target (not one
  // full-viewport div): a full-screen overlay would physically sit on top
  // of the page and swallow every click before it ever reached the real
  // element underneath, no matter what click-gating logic ran afterward.
  // Each mask only covers the area OUTSIDE the target rect, so a click
  // inside the spotlighted hole lands on the real page element directly --
  // no synthetic re-dispatch needed.
  function ensureMounted() {
    if (state.els) return state.els;

    const scrim = document.createElement("div");
    scrim.className = "tour-scrim";
    scrim.setAttribute("role", "presentation");

    const maskTop = document.createElement("div");
    maskTop.className = "tour-mask";
    const maskBottom = document.createElement("div");
    maskBottom.className = "tour-mask";
    const maskLeft = document.createElement("div");
    maskLeft.className = "tour-mask";
    const maskRight = document.createElement("div");
    maskRight.className = "tour-mask";

    const spotlight = document.createElement("div");
    spotlight.className = "tour-spotlight";
    spotlight.hidden = true;

    const arrow = document.createElement("div");
    arrow.className = "tour-arrow";
    arrow.hidden = true;

    const card = document.createElement("div");
    card.className = "tour-card";
    card.setAttribute("role", "dialog");
    card.setAttribute("aria-modal", "true");
    card.setAttribute("aria-labelledby", "tour-card-title");

    const eyebrow = document.createElement("p");
    eyebrow.className = "tour-card-eyebrow";
    const title = document.createElement("h2");
    title.id = "tour-card-title";
    const body = document.createElement("p");
    body.className = "tour-card-body";
    const hint = document.createElement("p");
    hint.className = "tour-card-hint";
    hint.hidden = true;

    const actions = document.createElement("div");
    actions.className = "tour-card-actions";
    const backButton = document.createElement("button");
    backButton.type = "button";
    backButton.className = "tour-button tour-button--ghost";
    backButton.textContent = "Back";
    const skipButton = document.createElement("button");
    skipButton.type = "button";
    skipButton.className = "tour-button tour-button--ghost";
    skipButton.textContent = "Skip";
    const nextButton = document.createElement("button");
    nextButton.type = "button";
    nextButton.className = "tour-button tour-button--primary";
    nextButton.textContent = "Next";

    actions.appendChild(backButton);
    actions.appendChild(skipButton);
    actions.appendChild(nextButton);
    card.appendChild(eyebrow);
    card.appendChild(title);
    card.appendChild(body);
    card.appendChild(hint);
    card.appendChild(actions);

    scrim.appendChild(maskTop);
    scrim.appendChild(maskBottom);
    scrim.appendChild(maskLeft);
    scrim.appendChild(maskRight);
    scrim.appendChild(spotlight);
    scrim.appendChild(arrow);
    scrim.appendChild(card);
    document.body.appendChild(scrim);

    backButton.addEventListener("click", () => step(-1));
    nextButton.addEventListener("click", () => step(1));
    skipButton.addEventListener("click", () => skipCurrent());

    state.els = {
      scrim, maskTop, maskBottom, maskLeft, maskRight, spotlight, arrow, card,
      eyebrow, title, body, hint, backButton, skipButton, nextButton,
    };
    return state.els;
  }

  function detachTargetClickHandler() {
    if (state.targetClickEl && state.targetClickHandler) {
      state.targetClickEl.removeEventListener("click", state.targetClickHandler, true);
    }
    state.targetClickEl = null;
    state.targetClickHandler = null;
  }

  function teardown() {
    if (!state.els) return;
    if (state.resizeObserver) {
      state.resizeObserver.disconnect();
      state.resizeObserver = null;
    }
    if (state.mutationObserver) {
      state.mutationObserver.disconnect();
      state.mutationObserver = null;
    }
    detachTargetClickHandler();
    document.removeEventListener("keydown", onKeydown, true);
    state.els.scrim.remove();
    state.els = null;
    document.body.classList.remove("modal-open");
    state.active = null;
  }

  function currentStep() {
    if (!state.active) return null;
    return state.active.definition.steps[state.active.stepIndex] || null;
  }

  function schedulePosition() {
    if (state.positionScheduled) return;
    state.positionScheduled = true;
    window.requestAnimationFrame(() => {
      state.positionScheduled = false;
      positionOverlay();
    });
  }

  function positionOverlay() {
    const els = state.els;
    const activeStep = currentStep();
    if (!els || !activeStep) return;

    const targetEl = activeStep.target ? resolveTarget(activeStep.target) : null;
    const viewportW = window.innerWidth;
    const viewportH = window.innerHeight;

    if (!targetEl) {
      // Missing target (kernel-91 S33): degrade to dim-only, centered card,
      // no spotlight/arrow -- never trap the user behind an impossible step.
      els.spotlight.hidden = true;
      els.arrow.hidden = true;
      setRect(els.maskTop, 0, 0, viewportW, viewportH);
      setRect(els.maskBottom, 0, 0, 0, 0);
      setRect(els.maskLeft, 0, 0, 0, 0);
      setRect(els.maskRight, 0, 0, 0, 0);
      els.card.style.top = "50%";
      els.card.style.left = "50%";
      els.card.style.bottom = "";
      els.card.style.width = `${Math.min(360, viewportW - 32)}px`;
      els.card.style.transform = "translate(-50%, -50%)";
      detachTargetClickHandler();
      return;
    }

    const rect = targetEl.getBoundingClientRect();
    const pad = 10;
    const holeTop = Math.max(rect.top - pad, 0);
    const holeLeft = Math.max(rect.left - pad, 0);
    const holeRight = Math.min(rect.right + pad, viewportW);
    const holeBottom = Math.min(rect.bottom + pad, viewportH);

    setRect(els.maskTop, 0, 0, viewportW, holeTop);
    setRect(els.maskBottom, 0, holeBottom, viewportW, Math.max(viewportH - holeBottom, 0));
    setRect(els.maskLeft, 0, holeTop, holeLeft, holeBottom - holeTop);
    setRect(els.maskRight, holeRight, holeTop, Math.max(viewportW - holeRight, 0), holeBottom - holeTop);

    els.spotlight.hidden = false;
    els.spotlight.style.top = `${holeTop}px`;
    els.spotlight.style.left = `${holeLeft}px`;
    els.spotlight.style.width = `${holeRight - holeLeft}px`;
    els.spotlight.style.height = `${holeBottom - holeTop}px`;

    const cardW = Math.min(360, viewportW - 32);
    const spaceBelow = viewportH - rect.bottom;
    const placeBelow = spaceBelow > 220 || spaceBelow > rect.top;

    els.card.style.transform = "";
    els.card.style.width = `${cardW}px`;
    const cardLeft = Math.min(Math.max(rect.left, 16), Math.max(viewportW - cardW - 16, 16));
    els.card.style.left = `${cardLeft}px`;

    if (placeBelow) {
      els.card.style.top = `${Math.min(rect.bottom + 24, viewportH - 40)}px`;
      els.card.style.bottom = "";
      els.arrow.className = "tour-arrow tour-arrow--up";
      els.arrow.style.top = `${rect.bottom + 6}px`;
    } else {
      els.card.style.bottom = `${viewportH - rect.top + 24}px`;
      els.card.style.top = "";
      els.arrow.className = "tour-arrow tour-arrow--down";
      els.arrow.style.top = `${rect.top - 22}px`;
    }
    els.arrow.hidden = false;
    els.arrow.style.left = `${Math.min(Math.max(rect.left + rect.width / 2 - 10, 16), viewportW - 26)}px`;

    attachTargetClickHandlerIfNeeded(targetEl, activeStep);
  }

  function setRect(el, left, top, width, height) {
    el.style.left = `${left}px`;
    el.style.top = `${top}px`;
    el.style.width = `${Math.max(width, 0)}px`;
    el.style.height = `${Math.max(height, 0)}px`;
  }

  // Click-gating for an action-required step: rather than intercepting
  // every click in the document and trying to guess whether it landed on
  // the target (fragile once a real overlay could sit on top), listen on
  // the resolved target element itself. A real click on the real element
  // both does whatever it normally does AND advances the tour.
  function attachTargetClickHandlerIfNeeded(targetEl, activeStep) {
    if (state.targetClickEl === targetEl && Boolean(state.targetClickHandler) === Boolean(activeStep.action_required)) {
      return;
    }
    detachTargetClickHandler();
    if (!activeStep.action_required) return;
    const handler = () => {
      // Record via beacon *before* calling step(1) -- if this click also
      // triggers real navigation (Audition Hall, Trailer), the page may
      // unload before step(1)'s own render or finish()'s ordinary fetch
      // completes. The beacon is what actually persists the user's place.
      const def = state.active && state.active.definition;
      if (def && !state.active.replay) {
        const isLastStep = state.active.stepIndex === def.steps.length - 1;
        const path = isLastStep
          ? `/api/tours/${encodeURIComponent(def.key)}/complete`
          : `/api/tours/${encodeURIComponent(def.key)}/progress`;
        beaconPost(path, { step_reached: activeStep.key });
      }
      step(1);
    };
    targetEl.addEventListener("click", handler, true);
    state.targetClickEl = targetEl;
    state.targetClickHandler = handler;
  }

  function renderStep() {
    const els = ensureMounted();
    const activeStep = currentStep();
    if (!state.active || !activeStep) return;

    document.body.classList.add("modal-open");
    els.eyebrow.textContent = state.active.definition.mandatory ? "Campus orientation" : "Guided tour";
    els.title.textContent = activeStep.title || "";
    els.body.textContent = activeStep.body || "";

    const targetResolvable = Boolean(activeStep.target && resolveTarget(activeStep.target));
    const actionGated = Boolean(activeStep.action_required && targetResolvable);
    els.hint.hidden = !actionGated;
    els.hint.textContent = "Click the highlighted spot to continue.";

    els.backButton.hidden = state.active.stepIndex === 0;
    els.skipButton.hidden = state.active.definition.mandatory || state.active.replay;
    // Kernel 91 S33: never trap the user behind an impossible click -- if
    // the target can't be resolved, fall back to a normal Next button even
    // on an action-required step.
    els.nextButton.hidden = actionGated;

    positionOverlay();
    document.addEventListener("keydown", onKeydown, true);
  }

  function onKeydown(event) {
    if (event.key !== "Escape") return;
    if (!state.active) return;
    if (state.active.definition.mandatory) {
      // Mandatory tour: Escape is deliberate but does not exit the tour
      // (kernel-91 S15/S35 -- no keyboard trap, but no silent bypass
      // either). It simply does nothing here.
      event.preventDefault();
      return;
    }
    event.preventDefault();
    skipCurrent();
  }

  function step(delta) {
    if (!state.active) return;
    const next = state.active.stepIndex + delta;
    if (next < 0) return;
    if (next >= state.active.definition.steps.length) {
      finish(false);
      return;
    }
    state.active.stepIndex = next;
    renderStep();
  }

  function skipCurrent() {
    if (!state.active) return;
    if (state.active.definition.mandatory) return;
    finish(true);
  }

  function finish(skipped) {
    if (!state.active) return;
    const def = state.active.definition;
    const isReplay = state.active.replay;
    teardown();
    if (isReplay) return; // Replay never writes completion state.
    const path = `/api/tours/${encodeURIComponent(def.key)}/${skipped ? "skip" : "complete"}`;
    apiFetch(path, { method: "POST" }).catch((error) => {
      console.error("VictoryTourEngine: failed to record tour state", error);
    });
  }

  function beginDefinition(definition, options) {
    if (!definition || !Array.isArray(definition.steps) || definition.steps.length === 0) return;
    if (state.active) teardown();
    state.active = { definition, stepIndex: 0, replay: Boolean(options && options.replay) };
    renderStep();

    if (!state.resizeObserver) {
      state.resizeObserver = new ResizeObserver(() => schedulePosition());
      state.resizeObserver.observe(document.body);
    }
    if (!state.mutationObserver) {
      // childList/subtree only -- deliberately NOT attributes. positionOverlay
      // writes inline style attributes onto elements inside this same
      // subtree (the scrim is a child of body), so watching attributes here
      // would make the observer fire on its own writes: an infinite
      // reposition loop that pins a CPU core and looks like exactly the
      // kind of runaway script activity an ad/tracker blocker flags.
      // Structural (childList) changes elsewhere on the page are still
      // caught, coalesced onto one rAF tick via schedulePosition.
      state.mutationObserver = new MutationObserver((mutations) => {
        // Ignore mutations inside our own overlay (e.g. text content swaps
        // on Next/Back visibility) so normal tour interaction never
        // re-triggers itself either.
        const relevant = mutations.some((m) => !state.els || !state.els.scrim.contains(m.target));
        if (relevant) schedulePosition();
      });
      state.mutationObserver.observe(document.body, { childList: true, subtree: true });
    }
  }

  async function fetchState(venueSlug) {
    const query = venueSlug ? `?venue=${encodeURIComponent(venueSlug)}` : "";
    const { ok, payload } = await apiFetch(`/api/tours/state${query}`);
    if (!ok || !payload || !payload.ok) return null;
    return payload.data;
  }

  const Engine = {
    registerTargets(targets) {
      Object.assign(state.targets, targets || {});
    },

    async autostart(options) {
      const opts = options || {};
      const venueSlug = opts.venueSlug || "";
      const data = await fetchState(venueSlug);
      if (!data) return;
      if (data.mandatory) {
        beginDefinition(data.mandatory, { replay: false });
        return;
      }
      const eligible = Array.isArray(data.eligible) ? data.eligible : [];
      if (eligible.length > 0) {
        beginDefinition(eligible[0], { replay: false });
      }
    },

    async start(tourKey, options) {
      const opts = options || {};
      const { ok, payload } = await apiFetch(`/api/tours/${encodeURIComponent(tourKey)}/replay`);
      if (!ok || !payload || !payload.ok) return;
      beginDefinition(payload.data, { replay: Boolean(opts.replay) });
    },

    isActive() {
      return Boolean(state.active);
    },
  };

  window.VictoryTourEngine = Engine;
})();
