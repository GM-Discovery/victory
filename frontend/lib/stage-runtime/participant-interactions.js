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
          <button type="button" class="is-primary" data-action="finish">Leave Kessa's Stall</button>
        </div>
      `);
      bindActions({
        stance: (btn) => runStance(btn.getAttribute("data-stance")),
        haggle: () => runHagglePreview(),
        equipment: () => renderEquipment(),
        close: () => panel.close(),
        finish: () => runLeaveKessa(),
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

    // --- Kernel 74: Kessa completion ---------------------------------------

    // "Leave Kessa's Stall" is a real server call, distinct from the panel's
    // close "×". Closing the panel is a UI dismissal that records nothing;
    // this records kessa_intro_completed, which is what reveals the door
    // hotspot for this Player and this Character only.
    //
    // No purchase, no Haggle, and no successful stance is required (S6.1).
    async function runLeaveKessa() {
      panel.setLoading(true);
      try {
        await fetchJSON(`/api/participant-interactions/${encodeURIComponent(currentInteractionId)}/complete`, { method: "POST" });
        panel.close();
        setStageStatus("You step away from Kessa's stall.");
        deps.onTutorialProgress?.();
      } catch (error) {
        panel.setError(error.message || String(error));
      }
    }

    // --- Kernel 74: freeform door intention --------------------------------

    function renderFreeform(config) {
      const title = config.title || "";
      const description = config.description || "";
      const prompt = config.prompt || "What does your Character try?";
      const maxLength = Number(config.max_length) || 1000;
      panel.setBody(`
        ${title ? `<p><strong>${escapeHtml(title)}</strong></p>` : ""}
        ${description ? `<p>${escapeHtml(description)}</p>` : ""}
        <p>${escapeHtml(prompt)}</p>
        <textarea data-field="freeform" rows="4" maxlength="${maxLength}"
          style="width:100%;box-sizing:border-box;padding:8px;border-radius:8px;"
          aria-label="${escapeHtml(prompt)}"></textarea>
        <div class="victory-program-panel__buttons">
          <button type="button" class="is-primary" data-action="submit">${escapeHtml(config.submit_label || "Make the Attempt")}</button>
        </div>
      `);
      // No suggested actions, choices, skills, stances, or rolls are offered
      // (S1.3). One neutral field, one button.
      bindActions({ submit: () => runFreeformSubmit() });
      const field = document.querySelector('[data-field="freeform"]');
      field?.focus();
    }

    async function runFreeformSubmit() {
      const field = document.querySelector('[data-field="freeform"]');
      const text = String(field?.value || "").trim();
      if (!text) {
        setStageStatus("Write what your Character tries first.");
        field?.focus();
        return;
      }
      panel.setLoading(true);
      try {
        const data = await fetchJSON(`/api/participant-interactions/${encodeURIComponent(currentInteractionId)}/submit`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ text, idempotency_key: newIdempotencyKey() }),
        });
        renderFreeformResult(data.submission);
      } catch (error) {
        panel.setError(error.message || String(error));
      }
    }

    // renderFreeformResult quotes the Player's own words back to them.
    //
    // The quoted text is inserted with textContent, never interpolated into
    // the HTML string above it (S1.4: "must be escaped and treated as plain
    // text, never trusted HTML"). The authored lead-in and interruption are
    // separate strings placed AROUND the quote rather than concatenated onto
    // it, so a Player who typed a complete sentence reads correctly.
    function renderFreeformResult(submission) {
      panel.setBody(`
        <p>${escapeHtml(submission.narration || "You make your move:")}</p>
        <blockquote style="margin:8px 0;padding-left:12px;border-left:3px solid rgba(255,233,197,0.4);font-style:italic;"><span data-field="submitted"></span></blockquote>
        <p>${escapeHtml(submission.interruption || "")}</p>
      `);
      const slot = document.querySelector('[data-field="submitted"]');
      if (slot) slot.textContent = `“${submission.submitted_text}”`;
      deps.onTutorialProgress?.();

      // S7.3: Ra begins automatically. No Director GO, no waiting.
      if (submission.next_interaction_id) {
        setTimeout(() => openInteraction(submission.next_interaction_id, "Ra", "guided_dialogue"), 900);
      }
    }

    // --- Kernel 74: Ra's guided dialogue -----------------------------------

    function renderDialogue(state, options = {}) {
      // Topic buttons are rendered from server-computed seen/unlocked flags.
      // The client never derives either one -- a locked topic is disabled
      // here for clarity, and refused again server-side if posted anyway.
      const topics = Array.isArray(state.topics) ? state.topics : [];
      const topicButtons = topics.map((topic) => {
        const attrs = topic.unlocked ? "" : "disabled";
        const seenMark = topic.seen ? " ✓" : "";
        const hint = topic.unlocked ? "" : " (not yet)";
        return `<button type="button" data-action="topic" data-topic="${escapeHtml(topic.topic_key)}" ${attrs}
          aria-label="${escapeHtml(topic.label)}${topic.seen ? ", already asked" : ""}"
          style="text-align:left;${topic.seen ? "opacity:0.72;" : ""}">${escapeHtml(topic.label)}${seenMark}${escapeHtml(hint)}</button>`;
      }).join("");

      const leaveAttrs = state.can_leave ? 'class="is-primary"' : "disabled";
      const leaveHint = state.can_leave ? "" : `<p style="opacity:0.75;font-size:0.85rem;">There is more you should hear before you go.</p>`;

      panel.setBody(`
        ${options.showOpening && state.opening_narration ? `<p><em>${escapeHtml(state.opening_narration)}</em></p>` : ""}
        ${state.current_response ? `<p><strong>${escapeHtml(state.npc_name)}:</strong> “${escapeHtml(state.current_response)}”</p>` : ""}
        <div class="victory-program-panel__buttons" style="flex-direction:column;align-items:stretch;">
          ${topicButtons}
        </div>
        ${leaveHint}
        <div class="victory-program-panel__buttons">
          <button type="button" data-action="leave" ${leaveAttrs}>${escapeHtml(state.leave_label || "Leave")}</button>
        </div>
      `);
      bindActions({
        topic: (btn) => runDialogueTopic(btn.getAttribute("data-topic")),
        leave: () => runDialogueLeave(),
      });
    }

    async function runDialogueTopic(topicKey) {
      panel.setLoading(true);
      try {
        const data = await fetchJSON(`/api/participant-interactions/${encodeURIComponent(currentInteractionId)}/dialogue/topic`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ topic_key: topicKey }),
        });
        currentContext = data.dialogue;
        renderDialogue(data.dialogue);
      } catch (error) {
        panel.setError(error.message || String(error));
      }
    }

    async function runDialogueLeave() {
      panel.setLoading(true);
      try {
        const data = await fetchJSON(`/api/participant-interactions/${encodeURIComponent(currentInteractionId)}/dialogue/leave`, { method: "POST" });
        // The closing narration is the lock reveal (S10.1) -- Ra operating a
        // concealed courtyard-side mechanism. Shown before the transition so
        // the Player actually reads how the door opened.
        panel.setBody(`
          <p><em>${escapeHtml(data.dialogue.closing_narration || "")}</em></p>
          <div class="victory-program-panel__buttons">
            <button type="button" class="is-primary" data-action="continue">Step Through</button>
          </div>
        `);
        bindActions({
          continue: () => {
            panel.close();
            deps.onTutorialProgress?.();
          },
        });
      } catch (error) {
        panel.setError(error.message || String(error));
      }
    }

    // --- Dispatch ------------------------------------------------------------

    async function openInteraction(interactionId, buttonLabel, interactionType) {
      currentInteractionId = interactionId;
      const type = String(interactionType || "open_equip_mode");
      panel.open({
        title: buttonLabel || "Program",
        onClose: () => { currentInteractionId = null; currentContext = null; },
      });
      panel.setLoading(true);
      try {
        // One open verb for all three Program types; the server dispatches on
        // the interaction's own stored type, so `type` here only decides how
        // to render the response, never what to request.
        const data = await fetchJSON(`/api/participant-interactions/${encodeURIComponent(interactionId)}/open`, { method: "POST" });

        if (type === "freeform_submission") {
          panel.updateHeader({ title: buttonLabel || "The Locked Door", subtitle: "" });
          // Re-opening after submitting shows the committed words rather
          // than an empty field: the intention cannot be edited once Ra's
          // interruption has begun.
          if (data.submitted) {
            renderFreeformResult(data.submitted);
          } else {
            renderFreeform(data.config || {});
          }
          return;
        }
        if (type === "guided_dialogue") {
          currentContext = data.dialogue;
          panel.updateHeader({
            title: data.dialogue.npc_name,
            subtitle: buttonLabel || "",
            imageUrl: data.dialogue.portrait_asset_id
              ? `/api/assets/${encodeURIComponent(data.dialogue.portrait_asset_id)}/content?variant=thumbnail`
              : (data.dialogue.portrait_url || ""),
          });
          renderDialogue(data.dialogue, { showOpening: true });
          deps.onTutorialProgress?.();
          return;
        }

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
