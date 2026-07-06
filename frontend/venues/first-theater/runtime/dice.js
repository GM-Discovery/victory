(function (root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  }
  root.VictoryFirstTheaterDice = factory();
})(typeof globalThis !== "undefined" ? globalThis : window, function () {
  const DEFAULT_HISTORY_LIMIT = 8;

  function normalizePositiveInt(value, fallback) {
    const parsed = Number.parseInt(String(value ?? ""), 10);
    return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
  }

  function normalizeSignedInt(value, fallback = 0) {
    const parsed = Number.parseInt(String(value ?? ""), 10);
    return Number.isFinite(parsed) ? parsed : fallback;
  }

  function buildExpression({ count = 1, sides = 20, explode = false, modifier = 0 } = {}) {
    const normalizedCount = normalizePositiveInt(count, 1);
    const normalizedSides = normalizePositiveInt(sides, 20);
    const normalizedModifier = normalizeSignedInt(modifier, 0);
    const parts = [];
    if (normalizedCount !== 1) {
      parts.push(String(normalizedCount));
    }
    parts.push(`d${normalizedSides}`);
    if (explode) {
      parts.push("!");
    }
    if (normalizedModifier !== 0) {
      parts.push(normalizedModifier > 0 ? `+${normalizedModifier}` : `${normalizedModifier}`);
    }
    return parts.join("");
  }

  function rollSequenceProgressions(chain) {
    const values = Array.isArray(chain) ? chain.map((value) => Number(value)).filter((value) => Number.isFinite(value)) : [];
    const progressions = [];
    let current = "";
    for (const value of values) {
      current = current ? `${current} ↻ ${value}` : `${value}`;
      progressions.push(current);
    }
    return progressions;
  }

  function normalizeRollAction(action) {
    const source = action && typeof action === "object" ? action : null;
    if (!source) return null;
    if (String(source.type || "").trim().toLowerCase() !== "roll/dice") return null;
    const payload = source.payload && typeof source.payload === "object" ? source.payload : {};
    const target = source.target && typeof source.target === "object" ? source.target : {};
    const actor = source.actor && typeof source.actor === "object" ? source.actor : {};
    const dice = Array.isArray(payload.dice) ? payload.dice : [];
    return {
      id: String(source.id || ""),
      requestId: String(payload.request_id || payload.requestId || ""),
      momentId: Number(source.moment_id ?? source.momentId ?? 0) || 0,
      timestamp: String(source.ts || source.timestamp || ""),
      actorId: String(source.actor_id || source.actorId || actor.user_id || ""),
      actorHandle: String(source.actor_handle || source.actorHandle || actor.handle || ""),
      actorDisplayName: String(source.actor_display_name || source.actorDisplayName || actor.display_name || actor.displayName || actor.handle || "Unknown"),
      actorRole: String(source.actor_role || source.actorRole || actor.role || ""),
      label: String(payload.label || ""),
      expression: String(payload.expression || ""),
      visibilityMode: String(payload.visibility_mode || payload.visibilityMode || "public"),
      modifier: Number(payload.modifier ?? 0) || 0,
      total: Number(payload.total ?? 0) || 0,
      explosionCount: Number(payload.explosion_count ?? 0) || 0,
      rollVersion: Number(payload.roll_version ?? 1) || 1,
      spec: payload.spec && typeof payload.spec === "object" ? payload.spec : {},
      dice: dice.map((die, index) => ({
        index: Number(die?.index ?? index) || 0,
        chain: Array.isArray(die?.chain) ? die.chain.map((value) => Number(value)).filter((value) => Number.isFinite(value)) : [],
        subtotal: Number(die?.subtotal ?? 0) || 0,
      })),
      target,
      payload,
      actor,
      raw: source,
    };
  }

  function formatRollTitle(roll) {
    const actor = String(roll?.actorDisplayName || roll?.actorHandle || "Unknown").trim() || "Unknown";
    const label = String(roll?.label || "").trim();
    const expression = String(roll?.expression || "").trim();
    if (label) {
      return `${actor} rolled ${label}`;
    }
    if (expression) {
      return `${actor} rolled ${expression}`;
    }
    return `${actor} rolled`;
  }

  function formatRollSummary(roll) {
    return `${formatRollTitle(roll)} = ${Number(roll?.total ?? 0) || 0}`;
  }

  function formatModifier(modifier) {
    const value = Number(modifier ?? 0) || 0;
    if (value > 0) return `+${value}`;
    return String(value);
  }

  function formatRollEntryLines(roll) {
    const lines = [];
    const expression = String(roll?.expression || "").trim();
    const label = String(roll?.label || "").trim();

    if (label && expression) {
      lines.push(`Expression: ${expression}`);
    }

    for (const die of Array.isArray(roll?.dice) ? roll.dice : []) {
      const chain = Array.isArray(die?.chain) ? die.chain.join(" ↻ ") : "";
      lines.push(`Die ${Number(die?.index ?? 0) + 1}: ${chain || "0"}`);
    }

    lines.push(`Modifier: ${formatModifier(roll?.modifier)}`);
    lines.push(`Total: ${Number(roll?.total ?? 0) || 0}`);
    return lines;
  }

  function createDiceTrayController(deps = {}) {
    const doc = deps.document || (typeof document !== "undefined" ? document : null);
    const win = deps.window || (typeof window !== "undefined" ? window : null);
    const mountRoot = deps.mountRoot || null;
    const sendAction = typeof deps.sendAction === "function" ? deps.sendAction : () => false;
    const getCurrentSnapshot = typeof deps.getCurrentSnapshot === "function" ? deps.getCurrentSnapshot : () => null;
    const setStageStatus = typeof deps.setStageStatus === "function" ? deps.setStageStatus : () => {};
    const setMovementLine = typeof deps.setMovementLine === "function" ? deps.setMovementLine : () => {};
    const appendSystemChatNotice = typeof deps.appendSystemChatNotice === "function" ? deps.appendSystemChatNotice : () => {};
    const onAction = typeof deps.onAction === "function" ? deps.onAction : () => {};
    const onPendingChange = typeof deps.onPendingChange === "function" ? deps.onPendingChange : () => {};
    const onHistoryChange = typeof deps.onHistoryChange === "function" ? deps.onHistoryChange : () => {};
    const canSendAction = typeof deps.canSendAction === "function" ? deps.canSendAction : () => true;
    const refreshWorld = typeof deps.refreshWorld === "function" ? deps.refreshWorld : async () => {};
    const getReducedMotion = typeof deps.getReducedMotion === "function"
      ? deps.getReducedMotion
      : () => Boolean(win?.matchMedia?.("(prefers-reduced-motion: reduce)")?.matches);
    const setTimeoutFn = deps.setTimeoutFn || ((callback, delay) => win?.setTimeout?.(callback, delay) || setTimeout(callback, delay));
    const clearTimeoutFn = deps.clearTimeoutFn || ((id) => (win?.clearTimeout?.(id) || clearTimeout(id)));

    let root = mountRoot;
    let destroyed = false;
    let manualExpressionOverride = false;
    let recentRolls = [];
    let recentRollIds = new Set();
    let pendingRolls = new Map();
    let renderTimers = new Set();
    let elements = {};
    let listeners = [];
    let actionAvailable = Boolean(canSendAction());
    const historyLimit = Number(deps.historyLimit || DEFAULT_HISTORY_LIMIT);

    function listen(target, type, handler, options) {
      target?.addEventListener?.(type, handler, options);
      const cleanup = () => target?.removeEventListener?.(type, handler, options);
      listeners.push(cleanup);
      return cleanup;
    }

    function cleanupTimers() {
      for (const timer of renderTimers) {
        clearTimeoutFn(timer);
      }
      renderTimers.clear();
    }

    function rejectPendingRolls(reason) {
      const error = reason instanceof Error ? reason : new Error(String(reason || "Roll failed."));
      for (const [requestId, pending] of pendingRolls.entries()) {
        clearTimeoutFn(pending.timer);
        pending.reject(error);
        pendingRolls.delete(requestId);
      }
      onPendingChange(pendingRolls.size);
    }

    function rejectPendingRoll(requestId, reason) {
      const key = String(requestId || "").trim();
      if (!key || !pendingRolls.has(key)) return false;
      const pending = pendingRolls.get(key);
      clearTimeoutFn(pending.timer);
      pendingRolls.delete(key);
      pending.reject(reason instanceof Error ? reason : new Error(String(reason || "Roll failed.")));
      onPendingChange(pendingRolls.size);
      return true;
    }

    function resolvePendingRoll(requestId, roll) {
      const key = String(requestId || "").trim();
      if (!key || !pendingRolls.has(key)) return false;
      const pending = pendingRolls.get(key);
      clearTimeoutFn(pending.timer);
      pendingRolls.delete(key);
      pending.resolve(roll);
      onPendingChange(pendingRolls.size);
      return true;
    }

    function resolveAnyPendingRoll(roll) {
      if (pendingRolls.size !== 1) return false;
      const onlyEntry = pendingRolls.entries().next().value;
      if (!onlyEntry) return false;
      const [requestId, pending] = onlyEntry;
      clearTimeoutFn(pending.timer);
      pendingRolls.delete(requestId);
      pending.resolve(roll);
      onPendingChange(pendingRolls.size);
      return true;
    }

    function setStatus(text) {
      if (elements.status) {
        elements.status.textContent = text || "Ready.";
      }
    }

    function refreshActionAvailability() {
      actionAvailable = Boolean(canSendAction());
      if (elements.rollButton) {
        elements.rollButton.disabled = !actionAvailable || pendingRolls.size > 0;
      }
    }

    function generateRequestId() {
      if (win?.crypto?.randomUUID) {
        return win.crypto.randomUUID();
      }
      return `dice-${Date.now()}-${Math.random().toString(16).slice(2, 10)}`;
    }

    function getControlsExpression() {
      const expression = String(elements.expression?.value || "").trim();
      if (expression) {
        return expression;
      }
      return buildExpression({
        count: elements.count?.value,
        sides: elements.sides?.value,
        explode: Boolean(elements.explode?.checked),
        modifier: elements.modifier?.value,
      });
    }

    function syncExpressionFromControls(force = false) {
      if (!elements.expression) return;
      const built = buildExpression({
        count: elements.count?.value,
        sides: elements.sides?.value,
        explode: Boolean(elements.explode?.checked),
        modifier: elements.modifier?.value,
      });
      if (force || !manualExpressionOverride || !String(elements.expression.value || "").trim()) {
        elements.expression.value = built;
      }
      if (elements.preview) {
        elements.preview.textContent = built;
      }
    }

    function syncLabelFromState() {
      if (!elements.label) return;
      elements.label.placeholder = "Optional label";
    }

    function appendHistory(roll) {
      if (!roll || !roll.id || recentRollIds.has(roll.id)) return false;
      recentRollIds.add(roll.id);
      recentRolls = [roll, ...recentRolls].slice(0, historyLimit);
      onHistoryChange(recentRolls.slice());
      renderHistory();
      return true;
    }

    function replaceHistoryFromSnapshot(actions = []) {
      const next = [];
      const seen = new Set();
      for (const action of Array.isArray(actions) ? actions : []) {
        const roll = normalizeRollAction(action);
        if (!roll || !roll.id || seen.has(roll.id)) continue;
        seen.add(roll.id);
        next.push(roll);
      }
      next.sort((a, b) => (Number(b.momentId || 0) - Number(a.momentId || 0)) || String(b.timestamp || "").localeCompare(String(a.timestamp || "")));
      recentRolls = next.slice(0, historyLimit);
      recentRollIds = new Set(recentRolls.map((roll) => roll.id));
      onHistoryChange(recentRolls.slice());
      renderHistory();
      return recentRolls.slice();
    }

    function createRow(label, valueText, className = "") {
      const row = doc.createElement("div");
      row.className = className ? `dice-roll__row ${className}` : "dice-roll__row";
      const labelEl = doc.createElement("div");
      labelEl.className = "dice-roll__label";
      labelEl.textContent = label;
      const valueEl = doc.createElement("div");
      valueEl.className = "dice-roll__value";
      valueEl.textContent = valueText;
      row.append(labelEl, valueEl);
      return row;
    }

    function renderHistoryEntry(roll) {
      const article = doc.createElement("article");
      article.className = "dice-roll-entry";
      article.dataset.rollId = roll.id;

      const header = doc.createElement("strong");
      header.textContent = formatRollTitle(roll);
      article.appendChild(header);

      if (String(roll.label || "").trim()) {
        article.appendChild(createRow("Expression", String(roll.expression || "").trim()));
      }

      const reducedMotion = getReducedMotion();
      for (const die of Array.isArray(roll.dice) ? roll.dice : []) {
        const row = createRow(`Die ${Number(die.index ?? 0) + 1}`, "");
        const valueEl = row.querySelector(".dice-roll__value");
        const progressions = rollSequenceProgressions(die.chain);
        const finalText = progressions.length > 0 ? progressions[progressions.length - 1] : "0";
        if (valueEl) {
          valueEl.textContent = reducedMotion || progressions.length <= 1 ? finalText : progressions[0];
          if (!reducedMotion && progressions.length > 1) {
            progressions.slice(1).forEach((text, index) => {
              const timer = setTimeoutFn(() => {
                if (valueEl.isConnected) {
                  valueEl.textContent = text;
                }
                renderTimers.delete(timer);
              }, 140 * (index + 1));
              renderTimers.add(timer);
            });
          }
        }
        article.appendChild(row);
      }

      article.appendChild(createRow("Modifier", formatModifier(roll.modifier)));
      article.appendChild(createRow("Total", String(Number(roll.total || 0) || 0)));
      article.appendChild(createRow("Meta", roll.timestamp ? `#${roll.momentId || 0} · ${new Date(roll.timestamp).toUTCString()}` : `#${roll.momentId || 0}`));
      return article;
    }

    function renderHistory() {
      if (!elements.history) return;
      cleanupTimers();
      elements.history.innerHTML = "";
      if (recentRolls.length === 0) {
        const empty = doc.createElement("div");
        empty.className = "dice-roll-empty";
        empty.textContent = "Recent rolls will appear here.";
        elements.history.appendChild(empty);
        return;
      }
      for (const roll of recentRolls) {
        elements.history.appendChild(renderHistoryEntry(roll));
      }
    }

    function renderControls() {
      if (!doc || !root) return;
      root.innerHTML = "";

      const shell = doc.createElement("section");
      shell.className = "dice-tray";

      const header = doc.createElement("div");
      header.className = "dice-tray__header";
      const title = doc.createElement("strong");
      title.textContent = "Dice";
      const status = doc.createElement("div");
      status.className = "dice-tray__status";
      status.textContent = "Ready.";
      header.append(title, status);

      const quickRow = doc.createElement("div");
      quickRow.className = "dice-tray__quick";
      [4, 6, 8, 10, 12, 20, 100].forEach((sides) => {
        const button = doc.createElement("button");
        button.type = "button";
        button.textContent = `d${sides}`;
        listen(button, "click", () => {
          if (elements.sides) {
            elements.sides.value = String(sides);
          }
          manualExpressionOverride = false;
          syncExpressionFromControls(true);
        });
        quickRow.appendChild(button);
      });

      const controls = doc.createElement("div");
      controls.className = "dice-tray__controls";

      const countLabel = doc.createElement("label");
      countLabel.textContent = "Count";
      const count = doc.createElement("input");
      count.type = "number";
      count.min = "1";
      count.max = "100";
      count.step = "1";
      count.value = "1";
      countLabel.appendChild(count);

      const sidesLabel = doc.createElement("label");
      sidesLabel.textContent = "Sides";
      const sides = doc.createElement("input");
      sides.type = "number";
      sides.min = "2";
      sides.max = "1000000";
      sides.step = "1";
      sides.value = "20";
      sidesLabel.appendChild(sides);

      const explodeLabel = doc.createElement("label");
      explodeLabel.className = "dice-tray__toggle";
      const explode = doc.createElement("input");
      explode.type = "checkbox";
      const explodeText = doc.createElement("span");
      explodeText.textContent = "Explode";
      explodeLabel.append(explode, explodeText);

      const modifierLabel = doc.createElement("label");
      modifierLabel.textContent = "Modifier";
      const modifier = doc.createElement("input");
      modifier.type = "number";
      modifier.step = "1";
      modifier.value = "0";
      modifierLabel.appendChild(modifier);

      const label = doc.createElement("label");
      label.textContent = "Label";
      const labelInput = doc.createElement("input");
      labelInput.type = "text";
      labelInput.maxLength = 120;
      labelInput.value = "";
      label.appendChild(labelInput);

      controls.append(countLabel, sidesLabel, explodeLabel, modifierLabel, label);

      const expressionLabel = doc.createElement("label");
      expressionLabel.textContent = "Expression";
      const expression = doc.createElement("input");
      expression.type = "text";
      expression.placeholder = "d20";
      expression.autocomplete = "off";
      expression.spellcheck = false;
      expression.value = "d20";
      expressionLabel.appendChild(expression);

      const preview = doc.createElement("div");
      preview.className = "dice-tray__preview";
      preview.textContent = "d20";

      const actionRow = doc.createElement("div");
      actionRow.className = "dice-tray__actions";
      const rollButton = doc.createElement("button");
      rollButton.type = "button";
      rollButton.textContent = "Roll";
      actionRow.appendChild(rollButton);

      const body = doc.createElement("div");
      body.className = "dice-tray__body";
      const controlsPane = doc.createElement("div");
      controlsPane.className = "dice-tray__controls-pane";
      controlsPane.append(quickRow, controls, expressionLabel, preview, actionRow);
      const historyPane = doc.createElement("section");
      historyPane.className = "dice-tray__history-pane";
      historyPane.setAttribute("aria-label", "Recent rolls");
      const historyTitle = doc.createElement("strong");
      historyTitle.className = "dice-tray__history-title";
      historyTitle.textContent = "Recent Rolls";
      const history = doc.createElement("div");
      history.className = "dice-tray__history";
      historyPane.append(historyTitle, history);
      body.append(controlsPane, historyPane);

      shell.append(header, body);
      root.appendChild(shell);

      elements = {
        root,
        shell,
        header,
        body,
        status,
        controlsPane,
        quickRow,
        controls,
        count,
        sides,
        explode,
        modifier,
        label: labelInput,
        expression,
        preview,
        actionRow,
        rollButton,
        historyPane,
        history,
      };

      const syncFromInput = () => {
        manualExpressionOverride = true;
        syncExpressionFromControls();
      };
      listen(count, "input", syncExpressionFromControls);
      listen(sides, "input", syncExpressionFromControls);
      listen(explode, "change", syncExpressionFromControls);
      listen(modifier, "input", syncExpressionFromControls);
      listen(expression, "input", syncFromInput);
      listen(labelInput, "input", syncLabelFromState);
      listen(rollButton, "click", () => {
        void submitRoll();
      });

      syncLabelFromState();
      syncExpressionFromControls(true);
      refreshActionAvailability();
      renderHistory();
    }

    function clearAll() {
      cleanupTimers();
      rejectPendingRolls("Roll canceled.");
      recentRolls = [];
      recentRollIds = new Set();
      onHistoryChange(recentRolls.slice());
      if (elements.history) {
        elements.history.innerHTML = "";
      }
    }

    async function submitRoll() {
      const expression = getControlsExpression();
      const label = String(elements.label?.value || "").trim();
      const visibility = "public";
      try {
        const result = await roll({ expression, visibility, label });
        setStageStatus(formatRollSummary(result));
        appendSystemChatNotice(formatRollSummary(result));
        return result;
      } catch (error) {
        const message = String(error?.message || error || "Roll failed.");
        if (message === "roll_timeout") {
          setStatus("Refreshing roll history...");
          setStageStatus("Refreshing roll history...");
          try {
            await refreshWorld();
            const latest = recentRolls[0] || null;
            if (latest && String(latest.expression || "").trim() === expression) {
              setStageStatus(formatRollSummary(latest));
              appendSystemChatNotice(formatRollSummary(latest));
              return latest;
            }
          } catch (refreshError) {
            appendSystemChatNotice(String(refreshError?.message || refreshError || "Refresh failed."));
          }
        }
        setStatus(message === "socket_unavailable" ? "Socket unavailable." : `Roll failed: ${message}`);
        appendSystemChatNotice(message);
        throw error;
      }
    }

    function roll({ expression, visibility = "public", label = "", skillId = "" } = {}) {
      const normalizedExpression = String(expression || "").trim();
      const normalizedLabel = String(label || "").trim();
      const normalizedVisibility = String(visibility || "public").trim().toLowerCase() || "public";
      const normalizedSkillId = String(skillId || "").trim();
      if (!normalizedExpression) {
        return Promise.reject(new Error("expression_required"));
      }

      return new Promise((resolve, reject) => {
        const requestId = generateRequestId();
        const timer = setTimeoutFn(() => {
          pendingRolls.delete(requestId);
          onPendingChange(pendingRolls.size);
          reject(new Error("roll_timeout"));
        }, Number(deps.timeoutMs || 15000));
        pendingRolls.set(requestId, {
          timer,
          resolve: (rollResult) => resolve(rollResult),
          reject: (error) => reject(error),
        });
        onPendingChange(pendingRolls.size);
        refreshActionAvailability();
        setStageStatus(`Rolling ${normalizedExpression}...`);
        setMovementLine(`Rolling ${normalizedExpression}...`);

        const sent = sendAction("roll/dice", {
          request_id: requestId,
          expression: normalizedExpression,
          visibility: normalizedVisibility,
          label: normalizedLabel,
          skill_id: normalizedSkillId,
        });
        if (!sent) {
          clearTimeoutFn(timer);
          pendingRolls.delete(requestId);
          onPendingChange(pendingRolls.size);
          refreshActionAvailability();
          reject(new Error("socket_unavailable"));
          return;
        }
        setStatus(`Rolling ${normalizedExpression}...`);
      });
    }

    function handleAction(action) {
      const normalized = normalizeRollAction(action);
      if (!normalized) return null;
      appendHistory(normalized);
      const matched = normalized.requestId ? resolvePendingRoll(normalized.requestId, normalized) : false;
      if (!matched) {
        resolveAnyPendingRoll(normalized);
      }
      if (normalized.requestId || pendingRolls.size === 0) {
        refreshActionAvailability();
      }
      setStatus(formatRollSummary(normalized));
      setMovementLine(formatRollSummary(normalized));
      onAction(normalized);
      return normalized;
    }

    function handleError(errorText, message = {}) {
      const requestId = String(message?.request_id || message?.requestId || "").trim();
      let rejected = false;
      if (requestId) {
        rejected = rejectPendingRoll(requestId, new Error(String(errorText || "roll_failed")));
      } else if (pendingRolls.size === 1) {
        const onlyEntry = pendingRolls.entries().next().value;
        if (onlyEntry) {
          const [pendingRequestId] = onlyEntry;
          rejected = rejectPendingRoll(pendingRequestId, new Error(String(errorText || "roll_failed")));
        }
      }
      if (rejected) {
        setStatus(`Roll failed: ${errorText}`);
        refreshActionAvailability();
      }
      return rejected;
    }

    function handleSnapshot(snapshot) {
      const actions = Array.isArray(snapshot?.actions) ? snapshot.actions : [];
      replaceHistoryFromSnapshot(actions);
      return recentRolls.slice();
    }

    function destroy() {
      if (destroyed) return;
      destroyed = true;
      cleanupTimers();
      rejectPendingRolls("Roll canceled.");
      for (const cleanup of listeners) {
        cleanup();
      }
      listeners = [];
      if (root) {
        root.innerHTML = "";
      }
    }

    if (root && doc) {
      renderControls();
      handleSnapshot(getCurrentSnapshot());
    }

    return {
      buildExpression,
      formatRollSummary,
      formatRollTitle,
      rollSequenceProgressions,
      normalizeRollAction,
      handleAction,
      handleError,
      handleSnapshot,
      replaceHistoryFromSnapshot,
      getHistory: () => recentRolls.slice(),
      getPendingCount: () => pendingRolls.size,
      getStatus: () => String(elements.status?.textContent || ""),
      getExpression: () => String(elements.expression?.value || ""),
      setExpression: (value) => {
        if (elements.expression) {
          elements.expression.value = String(value || "");
          manualExpressionOverride = true;
          syncExpressionFromControls();
        }
      },
      setCount: (value) => {
        if (elements.count) {
          elements.count.value = String(value);
          manualExpressionOverride = false;
          syncExpressionFromControls(true);
        }
      },
      setSides: (value) => {
        if (elements.sides) {
          elements.sides.value = String(value);
          manualExpressionOverride = false;
          syncExpressionFromControls(true);
        }
      },
      setExplode: (value) => {
        if (elements.explode) {
          elements.explode.checked = Boolean(value);
          manualExpressionOverride = false;
          syncExpressionFromControls(true);
        }
      },
      setModifier: (value) => {
        if (elements.modifier) {
          elements.modifier.value = String(value);
          manualExpressionOverride = false;
          syncExpressionFromControls(true);
        }
      },
      setLabel: (value) => {
        if (elements.label) {
          elements.label.value = String(value || "");
        }
      },
      setManualExpressionOverride: (value) => {
        manualExpressionOverride = Boolean(value);
      },
      setActionAvailability: (value) => {
        actionAvailable = Boolean(value);
        if (elements.rollButton) {
          elements.rollButton.disabled = !actionAvailable || pendingRolls.size > 0;
        }
      },
      roll,
      submitRoll,
      rejectPendingRoll,
      rejectPendingRolls,
      destroy,
      mount: (nextRoot) => {
        root = nextRoot || root;
        if (root && doc) {
          renderControls();
          handleSnapshot(getCurrentSnapshot());
        }
        return root;
      },
    };
  }

  return {
    createDiceTrayController,
    buildExpression,
    rollSequenceProgressions,
    normalizeRollAction,
    formatRollSummary,
    formatRollTitle,
  };
});
