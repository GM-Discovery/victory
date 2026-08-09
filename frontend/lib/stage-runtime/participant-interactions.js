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
  const KESSA_PORTRAIT_FALLBACK = "/assets/Kessa.png";
  const CHARACTER_MAKER_REQUIRED_MESSAGE = "You haven't finished making a character. Please select the character maker in the right tray first.";

  function interactionErrorMessage(error) {
    // Kernel 85 §8.1: the server now attaches a Player-facing "message"
    // alongside the machine-readable error code for the locked-door/Ra gate
    // states (no_character_selected / not_a_roster_member / milestone_required)
    // -- see merchant.tutorialGateMessage. Prefer it when present so the
    // door's copy stays authored in one place instead of drifting between
    // this file and the backend.
    if (error && error.friendlyMessage) {
      return error.friendlyMessage;
    }
    const code = String(error?.message || error || "").trim();
    // A Player who reaches the stage before Character Making has completed
    // may have neither a roster row nor a selected Character yet. Those are
    // distinct server-side authorization states, but they require the same
    // useful next step in the Player-facing Program panel. Fallback for
    // interaction types the server has no door-specific copy for.
    if (code === "not_a_roster_member" || code === "no_character_selected") {
      return CHARACTER_MAKER_REQUIRED_MESSAGE;
    }
    return code;
  }

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
      const code = (payload && payload.data && payload.data.error) || `HTTP ${response.status}`;
      const error = new Error(code);
      // Kernel 85 §8.1: an optional server-authored display string, distinct
      // from the machine-readable code above. See interactionErrorMessage.
      const message = payload && payload.data && payload.data.message;
      if (message) {
        error.friendlyMessage = message;
      }
      throw error;
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

    function portraitURL(assetId, portraitURLValue) {
      return assetId
        ? `/api/assets/${encodeURIComponent(assetId)}/content?variant=stage`
        : String(portraitURLValue || "");
    }

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
          <button type="button" class="is-primary" data-action="finish">${escapeHtml(ctx.packet.close_label || "Leave the Shop")}</button>
        </div>
      `);
      // ONE exit, not two. Kernel 74 first shipped "Leave Kessa's Stall"
      // beside the packet's existing "Leave the Shop", which read as two
      // different doors out of the same conversation -- the distinction
      // (dismiss the panel vs. record the milestone) is an implementation
      // detail no Player should have to parse. Leaving is leaving: it
      // records completion and closes. The "x" and Esc remain a pure
      // dismissal for someone who only wanted a look.
      bindActions({
        stance: (btn) => runStance(btn.getAttribute("data-stance")),
        haggle: () => runHagglePreview(),
        equipment: () => renderEquipment(),
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
      const ruleLinks = currentContext.rule_links || {};
      const rows = stock.map((item) => {
        const owned = inventoryQuantityFor(item.id);
        const ownedNote = owned > 0 ? ` <em>(you have ${owned})</em>` : "";
        // Kernel 78: published eWrite rule section for this item, if linked.
        const rule = ruleLinks[item.id];
        const ruleNote = rule
          ? ` <a href="/venues/library/read.html?pub=${encodeURIComponent(rule.publication_id)}${rule.section_anchor ? "#" + encodeURIComponent(rule.section_anchor) : ""}" target="_blank" rel="noopener" style="font-size:12px;opacity:0.85;">View rule${rule.section_title ? ": " + escapeHtml(rule.section_title) : ""} →</a>`
          : "";
        return `
          <div style="margin-bottom:10px;">
            <strong>${escapeHtml(item.name)}</strong>${ownedNote}${ruleNote}<br/>
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
      // Ra's opening beat is a deliberate single invitation. Once the
      // Player asks it, the server response is rendered without showOpening
      // and the rest of the authored topic tree appears normally.
      const firstBeat = options.showOpening && !topics.some((topic) => topic.seen);
      const visibleTopics = firstBeat
        ? topics.filter((topic) => topic.topic_key === "why-looking")
        : topics;
      const topicButtons = visibleTopics.map((topic) => {
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
        // Kernel 75 S3.1: the closing narration is the lock reveal -- Ra
        // exposing and operating a concealed courtyard-side mechanism. It is
        // now staged across ordered beats so the Player watches the gate
        // open rather than reading the whole ending at once.
        //
        // FALLBACK, load-bearing: a packet seeded before migration 072, or
        // one a Director left as a single paragraph, has no closing_beats.
        // Falling back to closing_narration is what lets the frontend and
        // backend deploy in either order.
        const beats = Array.isArray(data.dialogue.closing_beats) && data.dialogue.closing_beats.length
          ? data.dialogue.closing_beats
          : [data.dialogue.closing_narration || ""];
        renderClosingBeats(beats.filter((b) => String(b || "").trim() !== ""), 0);
      } catch (error) {
        panel.setError(error.message || String(error));
      }
    }

    // renderClosingBeats reveals one beat at a time. The final beat's button
    // is `continue`, which POSTs the Continue verb -- NOT a panel close.
    // S3.1 is explicit that the completion projection must not interrupt
    // Ra's final physical action, so the Program only opens after this.
    function renderClosingBeats(beats, index) {
      const shown = beats.slice(0, index + 1);
      const isLast = index >= beats.length - 1;
      const paragraphs = shown.map((beat, i) => {
        const dim = i < shown.length - 1 ? ' class="is-dim"' : "";
        return `<p${dim}><em>${escapeHtml(beat)}</em></p>`;
      }).join("");
      panel.setBody(`
        <div data-region="closing-beats">${paragraphs}</div>
        <div class="victory-program-panel__buttons">
          ${isLast
            ? '<button type="button" class="is-primary" data-action="continue">Continue</button>'
            : '<button type="button" class="is-primary" data-action="next-beat">…</button>'}
        </div>
      `);
      bindActions({
        "next-beat": () => renderClosingBeats(beats, index + 1),
        continue: () => runTutorialContinue(),
      });
    }

    async function runTutorialContinue() {
      panel.setLoading(true);
      try {
        const data = await fetchJSON(
          `/api/participant-interactions/${encodeURIComponent(currentInteractionId)}/tutorial/continue`,
          { method: "POST" });
        renderTutorialCompletion(data.completion);
        deps.onTutorialProgress?.();
      } catch (error) {
        panel.setError(error.message || String(error));
      }
    }

    // openTutorialCompletion is the reopen path (S4.3). A pure read: it
    // records nothing and re-awards nothing.
    async function openTutorialCompletion(interactionId) {
      currentInteractionId = interactionId;
      panel.open({
        title: "Tutorial Complete",
        onClose: () => { currentInteractionId = null; currentContext = null; },
      });
      panel.setLoading(true);
      try {
        const data = await fetchJSON(
          `/api/participant-interactions/${encodeURIComponent(interactionId)}/tutorial/completion`);
        renderTutorialCompletion(data.completion);
      } catch (error) {
        panel.setError(error.message || String(error));
      }
    }

    // renderTutorialCompletion draws the six required sections (S4.1).
    //
    // EVERY string of prose here comes from the server. The client must not
    // become a second place where this copy is decided, or the wording lives
    // in two files and drifts. The only text authored here is structural
    // labels and button captions.
    function renderTutorialCompletion(completion) {
      const c = completion || {};
      panel.updateHeader({ title: "Tutorial Complete", subtitle: c.character_name || "" });

      const body = (c.body || []).map((p) => `<p>${escapeHtml(p)}</p>`).join("");

      // Section 2 -- What You Did. Generated summaries, server-authored.
      const recap = (c.recap_lines || []).length
        ? `<ul class="victory-story-list">${(c.recap_lines || [])
            .map((line) => `<li>${escapeHtml(line)}</li>`).join("")}</ul>`
        : "<p>Your Character's history begins here.</p>";

      // Section 3 -- Story So Far. Private by default; say so plainly.
      const storyCount = (c.story_events || []).length;
      const story = `
        <p>${storyCount} moment${storyCount === 1 ? "" : "s"} ${storyCount === 1 ? "was" : "were"} recorded in your Character's Story So Far.</p>
        <p class="is-dim">These entries are private to you. You can share individual moments with your table later, from your Character's Story So Far page.</p>
        <div class="victory-program-panel__buttons">
          <a class="chip" href="/venues/greenroom/?character_id=${encodeURIComponent(c.character_card_id || "")}">Open Story So Far</a>
        </div>`;

      // Section 4 -- Progress Earned. S7.3's visible half: the one-time
      // Player recognition appears only on a genuine first completion; a
      // replay with a second Character shows the Character-level line.
      const recog = c.recognition || {};
      const progress = recog.newly_granted && recog.grant
        ? `<p><strong>${escapeHtml(recog.grant.label || "")}</strong></p>
           <p>You completed your first Socio Show.</p>`
        : `<p>${escapeHtml(c.character_name || "This Character")} completed the Locked Courtyard.</p>
           ${recog.grant ? '<p class="is-dim">You have already earned first-completion recognition on this account.</p>' : ""}`;

      const inventory = (c.inventory || []).length
        ? `<ul class="victory-story-list">${(c.inventory || [])
            .map((item) => `<li>${escapeHtml(item.name || item.item_name || "")}</li>`).join("")}</ul>`
        : "<p class=\"is-dim\">You carry nothing new from the Courtyard.</p>";

      const waiting = (c.waiting_copy || []).map((p) => `<p>${escapeHtml(p)}</p>`).join("");

      panel.setBody(`
        <section data-section="tutorial-complete">
          <h3>${escapeHtml(c.headline || "Tutorial Complete")}</h3>
          ${body}
        </section>
        <section data-section="what-you-did">
          <h4>What You Did</h4>
          ${recap}
        </section>
        <section data-section="story-so-far">
          <h4>Story So Far</h4>
          ${story}
        </section>
        <section data-section="progress-earned">
          <h4>Progress Earned</h4>
          ${progress}
          ${inventory}
        </section>
        <section data-section="aftercare" data-aftercare-slot="1">
          <h4>Aftercare</h4>
          <p>Take a moment, if you want to. It is optional and you can come back to it.</p>
          <div class="victory-program-panel__buttons">
            <button type="button" class="is-primary" data-action="aftercare-open">Write something</button>
            <button type="button" data-action="aftercare-skip">Skip</button>
          </div>
        </section>
        <section data-section="waiting">
          <h4>Waiting for Human Play</h4>
          ${waiting}
          <div class="victory-program-panel__buttons">
            <button type="button" data-action="close-completion">Explore Catharsis</button>
          </div>
        </section>
      `);

      bindActions({
        "close-completion": () => {
          // S1.3/S4.3: closing must not erase completion or return the
          // Player to the Courtyard fiction. It just puts the panel away;
          // the local projection and the reopen affordance both persist.
          panel.close();
          deps.onTutorialProgress?.();
        },
        "aftercare-open": () => openAftercare(c),
        "aftercare-skip": () => confirmAftercareSkip(c),
      });
    }

    // --- Aftercare (S8) -------------------------------------------------------
    //
    // Three qualitative prompts, all optional, with Save and Close / Skip.
    //
    // Drafts autosave SERVER-SIDE, not to localStorage. A Player may finish
    // on a different device, and reflection text must not linger in browser
    // storage after a logout on a shared machine.

    let aftercareDraftTimer = null;
    let aftercareShowID = "";

    function collectAftercareResponses() {
      const out = {};
      document.querySelectorAll("[data-aftercare-prompt]").forEach((el) => {
        out[el.getAttribute("data-aftercare-prompt")] = el.value || "";
      });
      return out;
    }

    function scheduleAftercareDraftSave() {
      if (aftercareDraftTimer) clearTimeout(aftercareDraftTimer);
      aftercareDraftTimer = setTimeout(saveAftercareDraft, 1200);
    }

    async function saveAftercareDraft() {
      if (!aftercareShowID) return;
      try {
        await fetchJSON(`/api/shows/${encodeURIComponent(aftercareShowID)}/aftercare/draft`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ responses: collectAftercareResponses() }),
        });
      } catch (error) {
        // A failed autosave must never interrupt writing. The Player still
        // has their text on screen, and Save and Close submits it directly.
        setStageStatus("Draft not saved: " + (error.message || String(error)));
      }
    }

    async function openAftercare(completion) {
      aftercareShowID = (completion && completion.show_id) || "";
      panel.setLoading(true);
      try {
        const data = await fetchJSON(`/api/shows/${encodeURIComponent(aftercareShowID)}/aftercare`);
        renderAftercareForm(completion, data.aftercare || {});
      } catch (error) {
        panel.setError(error.message || String(error));
      }
    }

    function renderAftercareForm(completion, state) {
      const prompts = state.prompts || [];
      const draft = (state.draft && state.draft.responses) || {};
      const fields = prompts.map((p) => `
        <label class="victory-aftercare-field">
          <span class="victory-aftercare-field__prompt">${escapeHtml(p.label)}</span>
          <textarea data-aftercare-prompt="${escapeHtml(p.key)}"
                    maxlength="${Number(p.max_length) || 2000}"
                    rows="3">${escapeHtml(draft[p.key] || "")}</textarea>
        </label>`).join("");

      panel.setBody(`
        <section data-section="aftercare-form">
          <h4>Aftercare</h4>
          <p class="victory-aftercare-notice" data-aftercare-notice="1">
            Your Director can read this.
          </p>
          <p class="is-dim">Every question is optional. You can save what you have and come back later.</p>
          <div class="victory-aftercare-fields">${fields}</div>
          <div class="victory-program-panel__buttons">
            <button type="button" class="is-primary" data-action="aftercare-save">Save and Close</button>
            <button type="button" data-action="aftercare-skip">Skip</button>
            <button type="button" data-action="aftercare-back">Back</button>
          </div>
        </section>
      `);

      document.querySelectorAll("[data-aftercare-prompt]").forEach((el) => {
        el.addEventListener("input", scheduleAftercareDraftSave);
      });

      bindActions({
        "aftercare-save": () => submitAftercare(completion),
        "aftercare-skip": () => confirmAftercareSkip(completion),
        "aftercare-back": () => renderTutorialCompletion(completion),
      });
    }

    async function submitAftercare(completion) {
      panel.setLoading(true);
      try {
        await fetchJSON(`/api/shows/${encodeURIComponent(aftercareShowID)}/aftercare`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ responses: collectAftercareResponses() }),
        });
        renderNotesReminder(completion);
      } catch (error) {
        panel.setError(error.message || String(error));
      }
    }

    // confirmAftercareSkip is S8.4/S1.14's two-step skip.
    //
    // The POST fires only on the SECOND press. Closing the panel, refreshing,
    // or pressing Go Back all leave the count untouched -- there is no
    // endpoint they could reach that would change it (S8.5).
    async function confirmAftercareSkip(completion) {
      aftercareShowID = (completion && completion.show_id) || aftercareShowID;
      let consecutive = 0;
      try {
        const data = await fetchJSON(`/api/shows/${encodeURIComponent(aftercareShowID)}/aftercare`);
        consecutive = Number((data.aftercare || {}).consecutive_skips) || 0;
      } catch (error) {
        // If we cannot read the count, still offer the choice -- just
        // without the factual context line.
        consecutive = 0;
      }

      // Factual, never punitive (S1.14).
      let context = "";
      if (consecutive === 1) {
        context = "<p>You skipped Aftercare last time.</p>";
      } else if (consecutive === 2) {
        context = "<p>You skipped the last two Aftercare check-ins.</p>";
      } else if (consecutive >= 3) {
        context = `<p>You have skipped the last ${consecutive} Aftercare check-ins.</p>`;
      }

      panel.setBody(`
        <section data-section="aftercare-skip-confirm">
          <h4>Skip Aftercare?</h4>
          ${context}
          <p>You can still write it later from the Greenroom. Skipping is recorded.</p>
          <div class="victory-program-panel__buttons">
            <button type="button" class="is-primary" data-action="skip-confirm">Continue Without Aftercare</button>
            <button type="button" data-action="skip-back">Go Back</button>
          </div>
        </section>
      `);
      bindActions({
        "skip-confirm": async () => {
          panel.setLoading(true);
          try {
            await fetchJSON(`/api/shows/${encodeURIComponent(aftercareShowID)}/aftercare/skip`, {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({ confirmed: true }),
            });
            renderNotesReminder(completion);
          } catch (error) {
            panel.setError(error.message || String(error));
          }
        },
        "skip-back": () => renderTutorialCompletion(completion),
      });
    }

    // S1.16/S8.6: My People is the default note follow-up.
    function renderNotesReminder(completion) {
      panel.setBody(`
        <section data-section="notes-reminder">
          <h4>Before you go</h4>
          <p>Did you take notes? Don't forget to update your notes on yourself and your fellow Players.</p>
          <div class="victory-program-panel__buttons">
            <a class="chip is-primary" href="/venues/trailers/people.html">Update My People</a>
            <button type="button" data-action="notes-next">Next</button>
          </div>
        </section>
      `);
      bindActions({
        "notes-next": () => renderTutorialCompletion(completion),
      });
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
          panel.hideSpeaker?.();
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
          const raPortrait = portraitURL(data.dialogue.portrait_asset_id, data.dialogue.portrait_url);
          panel.showSpeaker?.({ imageUrl: raPortrait, side: "right", label: data.dialogue.npc_name || "Ra" });
          panel.updateHeader({
            title: data.dialogue.npc_name,
            subtitle: buttonLabel || "",
            imageUrl: data.dialogue.portrait_asset_id
              ? portraitURL(data.dialogue.portrait_asset_id, data.dialogue.portrait_url)
              : (data.dialogue.portrait_url || ""),
          });
          renderDialogue(data.dialogue, { showOpening: true });
          deps.onTutorialProgress?.();
          return;
        }

        currentContext = data;
        const kessaPortrait = portraitURL(data.packet.portrait_asset_id, data.packet.portrait_url) || KESSA_PORTRAIT_FALLBACK;
        panel.showSpeaker?.({ imageUrl: kessaPortrait, side: "left", mirrored: true, label: data.packet.display_name || "Kessa" });
        panel.updateHeader({
          title: data.packet.display_name,
          subtitle: buttonLabel || "",
          imageUrl: kessaPortrait,
        });
        renderMain();
      } catch (error) {
        const message = interactionErrorMessage(error);
        setStageStatus(message);
        panel.setError(message);
      }
    }

    return { openInteraction, openTutorialCompletion };
  }

  return { createParticipantInteractionsController, interactionErrorMessage };
});
