(function (root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  }
  root.VictoryStageParticipantInteractions = factory();
})(typeof globalThis !== "undefined" ? globalThis : window, function () {
  // Kernel 73: drives a participant-local interaction (currently only
  // interaction_type "open_equip_mode") inside a VictoryStageProgramPanel.
  // Venue-agnostic -- Catharsis supplies no logic here, only the button
  // label/placement via the Director authoring UI and Kessa's seeded
  // packet content server-side. A future bounded merchant/program packet
  // reuses this same module untouched.

  const STANCE_LABELS = {
    command: "Command",
    convince: "Convince",
    insight: "Insight",
    follow: "Follow",
    sympathize: "Sympathize",
  };
  const STANCE_ORDER = ["command", "convince", "insight", "follow", "sympathize"];

  function newIdempotencyKey() {
    if (typeof globalThis !== "undefined" && globalThis.crypto && globalThis.crypto.randomUUID) {
      return globalThis.crypto.randomUUID();
    }
    return "k" + Date.now() + "-" + Math.random().toString(16).slice(2);
  }

  async function fetchJSON(url, options) {
    const response = await fetch(url, Object.assign({ credentials: "include" }, options || {}));
    const payload = await response.json().catch(() => null);
    if (!response.ok || !payload || payload.ok === false) {
      const error = (payload && payload.data && payload.data.error) || `HTTP ${response.status}`;
      throw new Error(error);
    }
    return payload.data;
  }

  // createParticipantInteractionsController(deps): deps = { panel
  // (VictoryStageProgramPanel instance), escapeHtml, setStageStatus }.
  // Returns { openInteraction(interactionId, buttonLabel) }.
  function createParticipantInteractionsController(deps) {
    const panel = deps.panel;
    const escapeHtml = deps.escapeHtml || ((v) => String(v ?? ""));
    const setStageStatus = deps.setStageStatus || (() => {});

    let currentInteractionId = null;
    let currentContext = null; // last OpenEquipMode response

    function bindActions(handlers) {
      const bodyEl = panel.isOpen() ? document.querySelector(".victory-program-panel__body") : null;
      if (!bodyEl) return;
      bodyEl.onclick = (event) => {
        const target = event.target.closest("[data-action]");
        if (!target) return;
        const action = target.getAttribute("data-action");
        const handler = handlers[action];
        if (handler) handler(target);
      };
    }

    function renderMain() {
      const ctx = currentContext;
      const stanceButtons = STANCE_ORDER.map((key) => `
        <button type="button" data-action="stance" data-stance="${key}">${escapeHtml(STANCE_LABELS[key])}</button>
      `).join("");
      panel.setBody(`
        <p>${escapeHtml(ctx.packet.intro_text)}</p>
        <div class="victory-program-panel__buttons">
          ${stanceButtons}
          <button type="button" data-action="haggle">Haggle</button>
          <button type="button" data-action="equipment">Browse Equipment</button>
          <button type="button" data-action="close">${escapeHtml(ctx.packet.close_label || "Leave the Shop")}</button>
        </div>
      `);
      bindActions({
        stance: (btn) => runStance(btn.getAttribute("data-stance")),
        haggle: () => runHagglePreview(),
        equipment: () => renderEquipment(),
        close: () => panel.close(),
      });
    }

    async function runStance(stanceKey) {
      panel.setLoading(true);
      try {
        const data = await fetchJSON(`/api/participant-interactions/${encodeURIComponent(currentInteractionId)}/stance`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ stance: stanceKey }),
        });
        const result = data.result;
        panel.setBody(`
          <p><strong>${escapeHtml(STANCE_LABELS[stanceKey] || stanceKey)}</strong></p>
          <p>${escapeHtml(result.response)}</p>
          <div class="victory-program-panel__buttons">
            <button type="button" data-action="return">${escapeHtml((currentContext.packet.return_label) || "Return to Conversation")}</button>
          </div>
        `);
        bindActions({ return: () => renderMain() });
      } catch (error) {
        panel.setError(error.message || String(error));
      }
    }

    async function runHagglePreview() {
      panel.setLoading(true);
      try {
        const data = await fetchJSON(`/api/participant-interactions/${encodeURIComponent(currentInteractionId)}/haggle`, { method: "GET" });
        const preview = data.preview;
        const warning = preview.impossible_to_reach
          ? `<p><em>Your ${escapeHtml(preview.die)} cannot reach Target Value ${preview.target_value}. Success is impossible unless you attempt anyway.</em></p>`
          : "";
        const attemptButtons = preview.impossible_to_reach
          ? `<button type="button" data-action="attempt-anyway">Attempt Anyway</button>
             <button type="button" data-action="return">${escapeHtml(currentContext.packet.return_label || "Return to Conversation")}</button>`
          : `<button type="button" class="is-primary" data-action="attempt-anyway">Roll ${escapeHtml(preview.die)}</button>
             <button type="button" data-action="return">${escapeHtml(currentContext.packet.return_label || "Return to Conversation")}</button>`;
        panel.setBody(`
          <p>Kessa is an established merchant.</p>
          <p>Your Haggle die: <strong>${escapeHtml(preview.die)}</strong> ${preview.has_skill ? "(skilled)" : "(unskilled)"}</p>
          <p>Target Value: <strong>${preview.target_value}</strong></p>
          ${warning}
          <div class="victory-program-panel__buttons">${attemptButtons}</div>
        `);
        bindActions({
          "attempt-anyway": () => runHaggleAttempt(),
          return: () => renderMain(),
        });
      } catch (error) {
        panel.setError(error.message || String(error));
      }
    }

    async function runHaggleAttempt() {
      panel.setLoading(true);
      try {
        const data = await fetchJSON(`/api/participant-interactions/${encodeURIComponent(currentInteractionId)}/haggle`, { method: "POST" });
        const result = data.result;
        panel.setBody(`
          <p>${escapeHtml(result.die)}: rolled <strong>${result.total}</strong> vs Target Value ${result.target_value} — ${result.success ? "Success!" : "Not quite."}</p>
          <p>${escapeHtml(result.text)}</p>
          <div class="victory-program-panel__buttons">
            <button type="button" data-action="return">${escapeHtml(currentContext.packet.return_label || "Return to Conversation")}</button>
          </div>
        `);
        bindActions({ return: () => renderMain() });
      } catch (error) {
        panel.setError(error.message || String(error));
      }
    }

    function inventoryQuantityFor(equipmentItemId) {
      const entry = (currentContext.inventory || []).find((e) => e.equipment_item_id === equipmentItemId);
      return entry ? entry.quantity : 0;
    }

    function renderEquipment() {
      const stock = currentContext.packet.stock || [];
      const rows = stock.map((item) => {
        const owned = inventoryQuantityFor(item.id);
        const ownedNote = owned > 0 ? ` <em>(you have ${owned})</em>` : "";
        return `
          <div style="margin-bottom:10px;">
            <strong>${escapeHtml(item.name)}</strong>${ownedNote}<br/>
            <span style="opacity:0.8;">${escapeHtml(item.short_description)}</span><br/>
            <button type="button" data-action="purchase" data-item="${escapeHtml(item.id)}">Purchase</button>
          </div>
        `;
      }).join("") || "<p>Nothing in stock right now.</p>";
      panel.setBody(`
        ${rows}
        <div class="victory-program-panel__buttons">
          <button type="button" data-action="return">${escapeHtml(currentContext.packet.return_label || "Return to Conversation")}</button>
        </div>
      `);
      bindActions({
        purchase: (btn) => runPurchase(btn.getAttribute("data-item")),
        return: () => renderMain(),
      });
    }

    async function runPurchase(equipmentItemID) {
      panel.setLoading(true);
      try {
        const data = await fetchJSON(`/api/participant-interactions/${encodeURIComponent(currentInteractionId)}/purchase`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ equipment_item_id: equipmentItemID, idempotency_key: newIdempotencyKey() }),
        });
        const entry = data.inventory_item;
        // Refresh local inventory snapshot so subsequent "you have N" labels
        // are accurate without a full re-open.
        const existingIndex = (currentContext.inventory || []).findIndex((e) => e.equipment_item_id === equipmentItemID);
        if (existingIndex >= 0) {
          currentContext.inventory[existingIndex] = entry;
        } else {
          currentContext.inventory = (currentContext.inventory || []).concat([entry]);
        }
        panel.setBody(`
          <p>Added to your inventory. You now have <strong>${entry.quantity}</strong>.</p>
          <div class="victory-program-panel__buttons">
            <button type="button" data-action="equipment">Keep Browsing</button>
            <button type="button" data-action="return">${escapeHtml(currentContext.packet.return_label || "Return to Conversation")}</button>
          </div>
        `);
        bindActions({
          equipment: () => renderEquipment(),
          return: () => renderMain(),
        });
      } catch (error) {
        panel.setError(error.message || String(error));
      }
    }

    async function openInteraction(interactionId, buttonLabel) {
      currentInteractionId = interactionId;
      panel.open({
        title: buttonLabel || "Equip Mode",
        onClose: () => { currentInteractionId = null; currentContext = null; },
      });
      panel.setLoading(true);
      try {
        const data = await fetchJSON(`/api/participant-interactions/${encodeURIComponent(interactionId)}/open`, { method: "POST" });
        currentContext = data;
        panel.updateHeader({
          title: data.packet.display_name,
          subtitle: buttonLabel || "",
          imageUrl: data.packet.portrait_asset_id ? `/api/assets/${encodeURIComponent(data.packet.portrait_asset_id)}/content?variant=thumbnail` : "",
        });
        renderMain();
      } catch (error) {
        setStageStatus(error.message || String(error));
        panel.setError(error.message || String(error));
      }
    }

    return { openInteraction };
  }

  return { createParticipantInteractionsController };
});
