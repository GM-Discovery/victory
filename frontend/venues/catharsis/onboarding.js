(function () {
  const introKey = "victory:catharsis:onboarding:intro-seen";
  const socioKey = "victory:catharsis:onboarding:socio-seen";

  const overlay = document.getElementById("catharsis-onboarding");
  const introPanel = document.getElementById("catharsis-onboarding-intro");
  const socioPanel = document.getElementById("catharsis-onboarding-socio");
  const buildPanel = document.getElementById("catharsis-onboarding-build");
  const chapterPanel = document.getElementById("catharsis-onboarding-chapter");
  const childhoodPanel = document.getElementById("catharsis-onboarding-childhood");
  const introButton = document.getElementById("catharsis-onboarding-continue");
  const dismissButton = document.getElementById("catharsis-onboarding-dismiss");
  const socioStatus = document.getElementById("catharsis-onboarding-socio-status");
  const chapterContinueButton = document.getElementById("catharsis-onboarding-chapter-continue");
  const childhoodStatus = document.getElementById("catharsis-onboarding-childhood-status");
  const rollButtons = [
    document.getElementById("catharsis-roll-1"),
    document.getElementById("catharsis-roll-2"),
    document.getElementById("catharsis-roll-3"),
  ];
  const rollAllButton = document.getElementById("catharsis-roll-all");
  const dieResultNodes = [
    document.getElementById("catharsis-die-1"),
    document.getElementById("catharsis-die-2"),
    document.getElementById("catharsis-die-3"),
  ];
  const rollTotalNode = document.getElementById("catharsis-roll-total");
  const buildTitle = document.getElementById("catharsis-build-title");
  const buildCopy = document.getElementById("catharsis-build-copy");
  const buildRoll = document.getElementById("catharsis-build-roll");
  const buildClass = document.getElementById("catharsis-build-class");
  const buildCredit = document.getElementById("catharsis-build-credit");
  const buildMeter = document.getElementById("catharsis-build-meter");
  const buildStatus = document.getElementById("catharsis-build-status");
  const buildWheelWindow = document.getElementById("catharsis-wheel-window");
  const buildLockButton = document.getElementById("catharsis-build-lock-in");
  const buildRerollButton = document.getElementById("catharsis-build-reroll");
  const childhoodContinueButton = document.getElementById("catharsis-onboarding-childhood-continue");
  const characterTrayButton = document.getElementById("character-tray-button");
  const characterTrayLabel = document.getElementById("character-tray-label");

  if (
    !overlay ||
    !introPanel ||
    !socioPanel ||
    !buildPanel ||
    !chapterPanel ||
    !childhoodPanel ||
    !introButton ||
    !dismissButton ||
    !chapterContinueButton ||
    !rollButtons[0] ||
    !rollButtons[1] ||
    !rollButtons[2] ||
    !rollAllButton ||
    !dieResultNodes[0] ||
    !dieResultNodes[1] ||
    !dieResultNodes[2] ||
    !rollTotalNode ||
    !buildTitle ||
    !buildCopy ||
    !buildRoll ||
    !buildClass ||
    !buildCredit ||
    !buildMeter ||
    !buildStatus ||
    !buildWheelWindow ||
    !buildLockButton ||
    !buildRerollButton ||
    !childhoodContinueButton
  ) {
    return;
  }

  const state = {
    canDraft: false,
    cards: [],
    activeCharacter: null,
    parentageChart: [],
    parentageChartVersion: "",
    phase: 1,
    parentRolls: [null, null],
    dieTotals: [null, null, null],
    totalRoll: null,
    revealEntry: null,
    workbookCardId: "",
    workbookContext: {},
    parentageRows: [],
    startingWealth: 0,
    draftToken: "",
    donorRolls: { egg_donor: null, sperm_donor: null },
  };

  let buildSequenceToken = 0;

  const draftTokenKey = "victory:catharsis:onboarding:draft-token";

  const ensureDraftToken = () => {
    if (state.draftToken) return state.draftToken;
    let token = "";
    try {
      token = localStorage.getItem(draftTokenKey) || "";
    } catch (error) {
      token = "";
    }
    if (!token) {
      token = window.crypto?.randomUUID ? window.crypto.randomUUID() : `${Date.now()}-${Math.random().toString(16).slice(2)}`;
      try {
        localStorage.setItem(draftTokenKey, token);
      } catch (error) {
        // localStorage unavailable; the token still works for this session.
      }
    }
    state.draftToken = token;
    return token;
  };

  const donorEventKeyForPhase = () => (state.phase === 1 ? "egg_donor" : "sperm_donor");

  const fetchDonorRoll = async () => {
    const eventKey = donorEventKeyForPhase();
    if (state.donorRolls[eventKey]) return state.donorRolls[eventKey];

    const response = await fetch("/api/character-cards/parentage-roll", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ draft_token: ensureDraftToken(), event_key: eventKey }),
    });
    const payload = await response.json().catch(() => null);
    if (!response.ok || !payload?.ok || !payload?.data) {
      throw new Error(payload?.data?.error || payload?.error || `Parentage roll failed (${response.status}).`);
    }
    state.donorRolls[eventKey] = payload.data;
    return payload.data;
  };

  const escapeHtml = (value) => String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll("\"", "&quot;")
    .replaceAll("'", "&#39;");

  const sleep = (delay) => new Promise((resolve) => window.setTimeout(resolve, delay));

  const randomDie = (sides) => {
    const max = Math.max(1, Number(sides) || 1);
    if (window.crypto?.getRandomValues) {
      const buffer = new Uint32Array(1);
      window.crypto.getRandomValues(buffer);
      return 1 + (buffer[0] % max);
    }
    return 1 + Math.floor(Math.random() * max);
  };

  const closeOverlay = () => {
    overlay.hidden = true;
    document.body.classList.remove("modal-open");
  };

  const setSocioStatus = (text) => {
    if (socioStatus) socioStatus.textContent = text || "";
  };

  const setBuildStatus = (text) => {
    if (buildStatus) buildStatus.textContent = text || "";
  };

  const setChildhoodStatus = (text) => {
    if (childhoodStatus) childhoodStatus.textContent = text || "";
  };

  const phaseLabel = () => (state.phase === 1 ? "first parent" : "other parent");

  const phasePrompt = () => (state.phase === 1
    ? "Roll the dice for the first parent."
    : "You have two parents so we're doing the process a second time to see about your other parent.");

  const phaseLockLabel = () => (state.phase === 1 ? "Lock first parent" : "Lock it in");

  const formatRollRange = (entry) => {
    if (!entry) return "Unknown";
    return entry.roll_min === entry.roll_max ? String(entry.roll_min) : `${entry.roll_min}-${entry.roll_max}`;
  };

  const findParentageEntry = (roll) => {
    return state.parentageChart.find((entry) => Number(entry.roll_min) <= roll && Number(entry.roll_max) >= roll) || null;
  };

  const parentageWindowForRoll = (roll) => {
    if (!state.parentageChart.length) return [];
    const index = state.parentageChart.findIndex((entry) => Number(entry.roll_min) <= roll && Number(entry.roll_max) >= roll);
    const safeIndex = index >= 0 ? index : 0;
    return state.parentageChart.slice(Math.max(0, safeIndex - 2), Math.min(state.parentageChart.length, safeIndex + 3));
  };

  const renderWheelWindow = (roll) => {
    const entries = parentageWindowForRoll(roll);
    if (!entries.length) {
      buildWheelWindow.innerHTML = '<div class="catharsis-wheel-window__slot"><span>Wheel</span><strong>Roll all three dice to reveal your place.</strong></div>';
      return;
    }

    buildWheelWindow.innerHTML = entries.map((entry) => {
      const current = Number(entry.roll_min) <= roll && Number(entry.roll_max) >= roll;
      return `
        <div class="catharsis-wheel-window__slot${current ? " catharsis-wheel-window__slot--current" : ""}">
          <span>Roll ${escapeHtml(formatRollRange(entry))}</span>
          <strong>${escapeHtml(entry.social_class || "Unknown")}</strong>
          <div>${escapeHtml(entry.description || "")}</div>
        </div>
      `;
    }).join("");
  };

  const showChapterPanel = () => {
    introPanel.hidden = true;
    socioPanel.hidden = true;
    buildPanel.hidden = true;
    chapterPanel.hidden = false;
    childhoodPanel.hidden = true;
    overlay.hidden = false;
    buildSequenceToken += 1;
    document.body.classList.add("modal-open");
    chapterContinueButton.focus();
  };

  const showChildhoodPanel = () => {
    introPanel.hidden = true;
    socioPanel.hidden = true;
    buildPanel.hidden = true;
    chapterPanel.hidden = true;
    childhoodPanel.hidden = false;
    overlay.hidden = false;
    document.body.classList.add("modal-open");
    setChildhoodStatus("Stage 2 is parked as a shell only. Return to the workbook when you are ready.");
    childhoodContinueButton.disabled = false;
    childhoodContinueButton.focus();
  };

  const getCurrentWorkbookContext = () => {
    const workbookContext = state.workbookContext && typeof state.workbookContext === "object" ? state.workbookContext : {};
    return { ...workbookContext };
  };

  const stringifyMaybe = (value) => {
    if (value == null) return "";
    if (Array.isArray(value) || typeof value === "object") {
      return JSON.stringify(value);
    }
    return String(value);
  };

  const buildParentageBundle = () => {
    const firstRoll = Number(state.parentRolls[0]) || 0;
    const secondRoll = Number(state.parentRolls[1]) || 0;
    const parentRolls = [firstRoll, secondRoll];
    const parentRows = parentRolls.map((roll, index) => {
      const entry = findParentageEntry(roll);
      const startingCredit = Number(entry?.starting_credit) || 0;
      return {
        parent_index: index + 1,
        roll_total: roll,
        roll_range: entry ? formatRollRange(entry) : "",
        social_class: entry?.social_class || "Unknown",
        wealth_kind: entry?.wealth_kind || "",
        starting_credit: startingCredit,
        description: entry?.description || "",
        coin_flip_eligible: startingCredit > 50,
        coin_flip_result: "pending",
        inheritance_passed: false,
        inherited_wealth: 0,
      };
    });

    return {
      parentRows,
      workbookContext: {
        source: "catharsis",
        creation_key: "catharsis:onboarding",
        ruleset_key: "socio",
        ruleset_version: "1.1",
        socio_parentage_chart_version: state.parentageChartVersion,
        socio_parentage_roll: firstRoll + secondRoll,
        socio_parentage_first_roll: firstRoll,
        socio_parentage_second_roll: secondRoll,
        socio_parentage_total_roll: firstRoll + secondRoll,
        socio_parentage_passes: 2,
        socio_parentage_parents: parentRows,
        socio_wealth_eligible_parents: parentRows.filter((parent) => parent.coin_flip_eligible).map((parent) => parent.parent_index),
        socio_starting_wealth: 0,
        socio_starting_wealth_source_parent_index: 0,
        socio_starting_wealth_source_roll: 0,
        socio_starting_wealth_source_class: "",
        active_page: "face",
        current_stage: 1,
        current_event: "egg_donor_parentage",
      },
    };
  };

  const summarizeParentRows = (parentRows) => parentRows.map((parent) => {
    const parts = [
      `Parent ${parent.parent_index}`,
      `roll ${parent.roll_total}`,
      parent.social_class,
      `credit ${parent.starting_credit}`,
      `coin flip ${parent.coin_flip_result}`,
      `inheritance ${parent.inherited_wealth}`,
    ].filter(Boolean);
    return parts.join(" · ");
  }).join("\n");

  const postWorkbookEvents = async (cardId, request) => {
    const response = await fetch(`/api/character-workbooks/${encodeURIComponent(cardId)}/events`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(request),
    });
    const payload = await response.json().catch(() => null);
    if (!response.ok || !payload?.ok) {
      throw new Error(payload?.data?.error || payload?.error || `Workbook event write failed (${response.status}).`);
    }
    return payload.data || {};
  };

  const resetRollState = (editable = true) => {
    state.dieTotals = [null, null, null];
    state.totalRoll = null;
    state.revealEntry = null;
    buildSequenceToken += 1;
    rollButtons.forEach((button, index) => {
      button.disabled = !editable || !state.canDraft;
      button.classList.remove("is-rolled");
      button.textContent = `Die ${index + 1}`;
    });
    dieResultNodes.forEach((node) => {
      node.textContent = "--";
    });
    rollTotalNode.textContent = "--";
    rollAllButton.disabled = !editable || !state.canDraft;
    rollAllButton.textContent = "All dice";
    buildLockButton.disabled = true;
    buildLockButton.textContent = phaseLockLabel();
    buildRerollButton.disabled = true;
    buildRerollButton.textContent = state.phase === 1 ? "Reroll first parent" : "Reroll second parent";
    buildTitle.textContent = "The wheel is revealing your place";
    buildCopy.textContent = "The curtain stays open while Catharsis binds your parentage roll into a first-pass identity.";
    buildRoll.textContent = "--";
    buildClass.textContent = "--";
    buildCredit.textContent = "--";
    buildMeter.style.width = "18%";
    buildStatus.textContent = "";
    buildWheelWindow.innerHTML = '<div class="catharsis-wheel-window__slot"><span>Wheel</span><strong>Roll all three dice to reveal your place.</strong></div>';
  };

  const renderRollState = () => {
    const rolledCount = state.dieTotals.filter((value) => Number.isFinite(value)).length;
    const total = state.dieTotals.reduce((sum, value) => sum + (Number(value) || 0), 0);

    state.dieTotals.forEach((value, index) => {
      const button = rollButtons[index];
      const resultNode = dieResultNodes[index];
      if (value == null) {
        button.disabled = !state.canDraft;
        button.classList.remove("is-rolled");
        button.textContent = `Die ${index + 1}`;
        resultNode.textContent = "--";
        return;
      }

      button.classList.add("is-rolled");
      button.disabled = true;
      button.textContent = `Die ${index + 1}: ${value}`;
      resultNode.textContent = String(value);
    });

    rollAllButton.disabled = !state.canDraft || rolledCount === 3;
    rollAllButton.textContent = rolledCount === 3 ? "All dice rolled" : "All dice";
    rollTotalNode.textContent = rolledCount ? String(total) : "--";

    if (rolledCount < 3) {
      buildLockButton.disabled = true;
      buildRerollButton.disabled = rolledCount === 0;
      setBuildStatus(rolledCount ? `${rolledCount} of 3 dice are locked in.` : "Roll one die at a time, or all three together.");
      if (rolledCount) {
        setSocioStatus(`${rolledCount} of 3 dice are locked in.`);
      }
      return;
    }

    state.totalRoll = total;
    state.revealEntry = findParentageEntry(total);
    buildTitle.textContent = state.phase === 1 ? "First parent revealed" : "Second parent revealed";
    buildRoll.textContent = String(total);
    buildClass.textContent = state.revealEntry?.social_class || "Unknown";
    buildCredit.textContent = String(state.revealEntry?.starting_credit ?? "--");
    buildCopy.textContent = state.revealEntry
      ? `${state.revealEntry.description} Nearby slots are shown below so you can see the strata around you.`
      : "The chart is ready, but the roll did not resolve to a known slot.";
    buildMeter.style.width = `${Math.max(18, Math.min(100, Math.round(18 + (total / 120) * 62)))}%`;
    renderWheelWindow(total);
    buildLockButton.disabled = false;
    buildRerollButton.disabled = false;
    buildLockButton.textContent = phaseLockLabel();
    buildRerollButton.textContent = state.phase === 1 ? "Reroll first parent" : "Reroll second parent";
    setBuildStatus("The wheel is ready.");
    setSocioStatus("");
  };

  const showIntro = () => {
    introPanel.hidden = false;
    socioPanel.hidden = true;
    buildPanel.hidden = true;
    chapterPanel.hidden = true;
    childhoodPanel.hidden = true;
    overlay.hidden = false;
    resetRollState(false);
    setSocioStatus("");
    document.body.classList.add("modal-open");
  };

  const showSocioPrompt = () => {
    introPanel.hidden = true;
    socioPanel.hidden = false;
    buildPanel.hidden = true;
    chapterPanel.hidden = true;
    childhoodPanel.hidden = true;
    overlay.hidden = false;
    resetRollState(true);
    setSocioStatus(phasePrompt());
    document.body.classList.add("modal-open");
  };

  window.VictoryCatharsisOnboarding = {
    begin: () => {
      showIntro();
    },
    continue: () => {
      showSocioPrompt();
    },
  };

  const showBuildPanel = () => {
    introPanel.hidden = true;
    socioPanel.hidden = true;
    buildPanel.hidden = false;
    chapterPanel.hidden = true;
    childhoodPanel.hidden = true;
    overlay.hidden = false;
    buildTitle.textContent = state.phase === 1 ? "First parent revealed" : "Second parent revealed";
    buildCopy.textContent = state.phase === 1
      ? "Lock it in to continue to the second parent."
      : "Lock it in and Catharsis will stamp both parent rolls into your draft.";
    buildLockButton.textContent = phaseLockLabel();
    buildRerollButton.textContent = state.phase === 1 ? "Reroll first parent" : "Reroll second parent";
    document.body.classList.add("modal-open");
  };

  const loadParentageChart = async () => {
    if (state.parentageChart.length) {
      return state.parentageChart;
    }

    try {
      const response = await fetch("/api/characters/parentage-chart", { credentials: "include" });
      const payload = await response.json().catch(() => null);
      if (response.ok && payload?.ok && payload?.data) {
        state.parentageChart = Array.isArray(payload.data.entries) ? payload.data.entries : [];
        state.parentageChartVersion = String(payload.data.version || "");
      }
    } catch (error) {
      console.error("parentage chart load failed", error);
    }

    return state.parentageChart;
  };

  // Animates the same flicker reveal as before, but always lands on the
  // server-determined canonical face. The flicker frames are decorative
  // only; they never decide the outcome.
  const animateButtonRollToValue = async (button, label, token, duration, finalFace) => {
    const startedAt = Date.now();
    while (Date.now() - startedAt < duration) {
      if (token !== buildSequenceToken) return null;
      button.textContent = `${label}: ${randomDie(20)}`;
      await sleep(58);
    }
    if (token !== buildSequenceToken) return null;
    button.textContent = `${label}: ${finalFace}`;
    return finalFace;
  };

  const rollDie = async (index) => {
    if (state.dieTotals[index] != null || !state.canDraft) return;

    const button = rollButtons[index];
    const token = buildSequenceToken;
    const label = `Die ${index + 1}`;
    button.disabled = true;
    setSocioStatus(`${label} is rolling.`);
    setBuildStatus(`${label} is rolling.`);

    let donorRoll;
    try {
      donorRoll = await fetchDonorRoll();
    } catch (error) {
      console.error("catharsis parentage roll request failed", error);
      if (token !== buildSequenceToken) return;
      setSocioStatus("Could not reach the dice. Try again.");
      setBuildStatus("Could not reach the dice. Try again.");
      button.disabled = false;
      return;
    }
    if (token !== buildSequenceToken) return;

    const chain = donorRoll.dice?.[index]?.chain || [donorRoll.dice?.[index]?.subtotal ?? 1];
    const firstRoll = await animateButtonRollToValue(button, label, token, 720, chain[0]);
    if (firstRoll == null || token !== buildSequenceToken) return;

    let dieTotal = firstRoll;
    if (chain.length > 1) {
      setSocioStatus(`${label} exploded on 20.`);
      setBuildStatus(`${label} exploded on 20. Rolling it once more.`);
      await sleep(220);
      const secondRoll = await animateButtonRollToValue(button, `${label} +`, token, 520, chain[1]);
      if (secondRoll == null || token !== buildSequenceToken) return;
      dieTotal += secondRoll;
      button.textContent = `${label}: 20 + ${secondRoll} = ${dieTotal}`;
    } else {
      button.textContent = `${label}: ${dieTotal}`;
    }

    state.dieTotals[index] = dieTotal;
    renderRollState();

    if (state.dieTotals.every((value) => Number.isFinite(value))) {
      showBuildPanel();
      renderRollState();
      setBuildStatus("The wheel is settling.");
      await sleep(260);
      if (token !== buildSequenceToken) return;
      setBuildStatus(`${phaseLabel()[0].toUpperCase()}${phaseLabel().slice(1)} revealed.`);
      buildLockButton.focus();
      localStorage.setItem(introKey, "1");
      localStorage.setItem(socioKey, "1");
      return;
    }

    const rolledCount = state.dieTotals.filter((value) => Number.isFinite(value)).length;
    setBuildStatus(`${rolledCount} of 3 dice are locked in.`);
  };

  const rollAllDice = async () => {
    for (let index = 0; index < rollButtons.length; index += 1) {
      if (state.dieTotals[index] == null) {
        await rollDie(index);
      }
      if (state.dieTotals.every((value) => Number.isFinite(value))) {
        return;
      }
    }
  };

  const advanceToSecondParent = async () => {
    const firstRoll = state.totalRoll ?? state.dieTotals.reduce((sum, value) => sum + Number(value || 0), 0);
    state.parentRolls[0] = firstRoll;
    state.phase = 2;
    resetRollState(true);
    showSocioPrompt();
    setSocioStatus("You have two parents so we're doing the process a second time to see about your other parent.");
  };

  const createCharacterFromRoll = async () => {
    if (!state.canDraft) {
      setSocioStatus("Character drafting is not granted yet.");
      return;
    }
    if (!state.dieTotals.every((value) => Number.isFinite(value))) {
      setSocioStatus("Roll all three dice first.");
      return;
    }

    const currentRoll = state.totalRoll ?? state.dieTotals.reduce((sum, value) => sum + Number(value || 0), 0);
    state.parentRolls[1] = currentRoll;
    const sequenceToken = ++buildSequenceToken;
    showBuildPanel();
    setBuildStatus("Binding the workbook...");
    buildCopy.textContent = "Catharsis is stamping both parent rolls into a workbook that Greenroom can open.";
    buildMeter.style.width = "100%";

    try {
      const bundle = buildParentageBundle();
      const total = bundle.parentRows[0].roll_total + bundle.parentRows[1].roll_total;
      const cardContext = {
        source: "catharsis",
        creation_key: "catharsis:onboarding",
        draft_token: ensureDraftToken(),
        ruleset_key: "socio",
        ruleset_version: "1.1",
        socio_parentage_roll: total,
        socio_parentage_first_roll: Number(state.parentRolls[0]) || 0,
        socio_parentage_second_roll: currentRoll,
        socio_parentage_total_roll: total,
        socio_parentage_passes: 2,
        socio_parentage_chart_version: state.parentageChartVersion,
        socio_parentage_parents: bundle.parentRows,
        socio_wealth_eligible_parents: bundle.parentRows.filter((parent) => parent.coin_flip_eligible).map((parent) => parent.parent_index),
        socio_starting_wealth: 0,
        socio_starting_wealth_source_parent_index: 0,
        socio_starting_wealth_source_roll: 0,
        socio_starting_wealth_source_class: "",
        active_page: "face",
        current_stage: 1,
        current_event: "egg_donor_parentage",
      };

      const response = await fetch("/api/character-cards", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          workbook_status: "draft",
          workbook_context: cardContext,
        }),
      });
      const payload = await response.json().catch(() => null);
      if (!response.ok || !payload?.ok || !payload?.data?.id) {
        throw new Error(payload?.data?.error || payload?.error || `Character creation failed (${response.status}).`);
      }

      if (sequenceToken !== buildSequenceToken) return;
      const card = payload.data;
      state.workbookCardId = String(card.id || "");
      state.workbookContext = { ...cardContext, ...(card.workbook_context || {}) };
      state.activeCharacter = { ...card, character_card_id: String(card.id || card.character_card_id || "") };
      refreshCharacterTray();
      try {
        localStorage.removeItem(draftTokenKey);
      } catch (error) {
        // ignore; a stale token only affects a future fresh build.
      }
      state.draftToken = "";
      state.donorRolls = { egg_donor: null, sperm_donor: null };

      const serverParentRows = Array.isArray(state.workbookContext.socio_parentage_parents)
        ? state.workbookContext.socio_parentage_parents
        : bundle.parentRows;
      const serverStartingWealth = Number(state.workbookContext.socio_starting_wealth) || 0;
      const firstParentRow = serverParentRows[0] || bundle.parentRows[0];
      const secondParentRow = serverParentRows[1] || bundle.parentRows[1];
      state.parentageRows = serverParentRows;
      state.startingWealth = serverStartingWealth;

      const historyEntries = [
        {
          page_key: "history",
          entry_type: "parentage_roll",
          title: "Parent 1 roll",
          body: summarizeParentRows([firstParentRow]),
          stage_number: 1,
          sort_order: 1,
          payload: {
            ...firstParentRow,
            source: "catharsis",
            ruleset_key: "socio",
            ruleset_version: "1.1",
          },
        },
      ];

      if (Number(firstParentRow?.starting_credit || 0) > 50) {
        historyEntries.push({
          page_key: "history",
          entry_type: "coin_flip",
          title: "Parent 1 coin flip",
          body: `Parent 1 retention resolved to ${firstParentRow.coin_flip_result || "pending"} and ${firstParentRow.inherited_wealth || 0} starting wealth.`,
          stage_number: 1,
          sort_order: 2,
          payload: {
            ...firstParentRow,
            source: "catharsis",
            ruleset_key: "socio",
            ruleset_version: "1.1",
          },
        });
      }

      historyEntries.push({
        page_key: "history",
        entry_type: "parentage_roll",
        title: "Parent 2 roll",
        body: summarizeParentRows([secondParentRow]),
        stage_number: 1,
        sort_order: historyEntries.length + 1,
        payload: {
          ...secondParentRow,
          source: "catharsis",
          ruleset_key: "socio",
          ruleset_version: "1.1",
        },
      });

      if (Number(secondParentRow?.starting_credit || 0) > 50) {
        historyEntries.push({
          page_key: "history",
          entry_type: "coin_flip",
          title: "Parent 2 coin flip",
          body: `Parent 2 retention resolved to ${secondParentRow.coin_flip_result || "pending"} and ${secondParentRow.inherited_wealth || 0} starting wealth.`,
          stage_number: 1,
          sort_order: historyEntries.length + 1,
          payload: {
            ...secondParentRow,
            source: "catharsis",
            ruleset_key: "socio",
            ruleset_version: "1.1",
          },
        });
      }

      historyEntries.push({
        page_key: "creation_progress",
        entry_type: "workbook_seed",
        title: "Workbook initialized",
        body: `Socio parentage is bound to the workbook. Total roll ${total}. Starting wealth ${serverStartingWealth}.`,
        stage_number: 1,
        sort_order: historyEntries.length + 1,
        payload: {
          source: "catharsis",
          ruleset_key: "socio",
          ruleset_version: "1.1",
          total_roll: total,
          parentage_first_roll: Number(state.parentRolls[0]) || 0,
          parentage_second_roll: currentRoll,
          parentage_rows: serverParentRows,
          starting_wealth: serverStartingWealth,
        },
      });

      await postWorkbookEvents(card.id, {
        character_card_id: card.id,
        module_key: "socio",
        module_status: "draft",
        current_stage: 1,
        current_event: "egg_donor_parentage",
        workbook_status: "draft",
        workbook_context: {
          ...state.workbookContext,
          active_page: "face",
          current_stage: 1,
          current_event: "egg_donor_parentage",
          socio_parentage_history: historyEntries,
        },
        entries: historyEntries,
      });

      if (sequenceToken !== buildSequenceToken) return;
      buildTitle.textContent = String(card?.name || firstParentRow.social_class || "Socio Candidate").trim();
      buildCopy.textContent = serverStartingWealth > 0
        ? `Catharsis finished the two-pass parentage. Starting wealth resolved to ${serverStartingWealth}.`
        : "Catharsis finished the two-pass parentage. No wealth inheritance passed the gate.";
      buildRoll.textContent = String(total);
      buildClass.textContent = String(secondParentRow.social_class || card?.name || "Unknown").trim();
      buildCredit.textContent = String(secondParentRow.starting_credit || "--");
      renderWheelWindow(total);
      setBuildStatus(`Ready: ${String(card?.name || secondParentRow.social_class || "Socio Candidate").trim()}`);
      localStorage.setItem(introKey, "1");
      localStorage.setItem(socioKey, "1");
      await sleep(620);
      if (sequenceToken !== buildSequenceToken) return;
      showChapterPanel();
      setBuildStatus("Chapter II is ready.");
    } catch (error) {
      if (sequenceToken !== buildSequenceToken) return;
      console.error("catharsis create character failed", error);
      setBuildStatus("Character creation failed.");
      setSocioStatus("Character creation failed. Try again once your Catharsis approval settles.");
      buildCopy.textContent = "The stage is still open. Try again once the approval settles.";
      buildMeter.style.width = "18%";
      buildLockButton.disabled = false;
      buildRerollButton.disabled = false;
    }
  };

  const continueFromChapter = async () => {
    if (!state.workbookCardId) {
      showChildhoodPanel();
      setChildhoodStatus("Stage 2 is parked as a shell only. Return to the workbook when you are ready.");
      return;
    }

    const sequenceToken = ++buildSequenceToken;
    chapterContinueButton.disabled = true;
    setBuildStatus("Saving Chapter II handoff...");
    try {
      await postWorkbookEvents(state.workbookCardId, {
        character_card_id: state.workbookCardId,
        module_key: "socio",
        module_status: "draft",
        current_stage: 2,
        current_event: "childhood_stages",
        workbook_status: "draft",
        workbook_context: {
          ...getCurrentWorkbookContext(),
          current_stage: 2,
          current_event: "childhood_stages",
          active_page: "face",
        },
        entries: [
          {
            page_key: "creation_progress",
            entry_type: "chapter_handoff",
            title: "Chapter II: Childhood Stages",
            body: "Catharsis opened the next chapter and handed the workbook into the childhood stage.",
            stage_number: 2,
            sort_order: 1,
            payload: {
              source: "catharsis",
              ruleset_key: "socio",
              ruleset_version: "1.1",
            },
          },
        ],
      });
      if (sequenceToken !== buildSequenceToken) return;
      showChildhoodPanel();
      setChildhoodStatus("Stage 2 is parked as a shell only. Return to the workbook when you are ready.");
    } catch (error) {
      if (sequenceToken !== buildSequenceToken) return;
      console.error("catharsis chapter handoff failed", error);
      chapterContinueButton.disabled = false;
      setBuildStatus("Could not save Chapter II handoff.");
    }
  };

  const continueFromChildhood = async () => {
    buildSequenceToken += 1;
    closeOverlay();
  };

  const romanNumeralNamePattern = /^M{0,4}(CM|CD|D?C{0,3})(XC|XL|L?X{0,3})(IX|IV|V?I{0,3})$/;
  const isAutoNamedCharacter = (name) => {
    const trimmed = String(name || "").trim();
    return trimmed !== "" && romanNumeralNamePattern.test(trimmed);
  };

  const refreshCharacterTray = () => {
    const activeCharacterName = String(state.activeCharacter?.display_name || state.activeCharacter?.name || "").trim();
    if (characterTrayLabel) {
      characterTrayLabel.textContent = activeCharacterName || "New Character";
      characterTrayLabel.title = isAutoNamedCharacter(activeCharacterName)
        ? "Auto-named — click to open the workbook and rename"
        : "";
      characterTrayLabel.classList.toggle("character-chip__auto-name", isAutoNamedCharacter(activeCharacterName));
    }
    if (characterTrayButton) {
      characterTrayButton.title = activeCharacterName
        ? (isAutoNamedCharacter(activeCharacterName) ? `Open the workbook for ${activeCharacterName} (auto-named)` : `Open the workbook for ${activeCharacterName}`)
        : "Open the workbook";
    }
  };

  characterTrayButton?.addEventListener("click", () => {
    const activeCharacterId = String(state.activeCharacter?.character_card_id || state.activeCharacter?.id || "").trim();
    if (activeCharacterId) {
      window.location.href = `/venues/greenroom/?character_id=${encodeURIComponent(activeCharacterId)}`;
      return;
    }
    showIntro();
  });

  introButton.addEventListener("click", async () => {
    localStorage.setItem(introKey, "1");
    await loadParentageChart();
    showSocioPrompt();
  });

  dismissButton.addEventListener("click", () => {
    localStorage.setItem(introKey, "1");
    localStorage.setItem(socioKey, "1");
    buildSequenceToken += 1;
    closeOverlay();
  });

  rollButtons.forEach((button, index) => {
    button.addEventListener("click", () => {
      void rollDie(index);
    });
  });

  rollAllButton.addEventListener("click", () => {
    void rollAllDice();
  });

  buildLockButton.addEventListener("click", () => {
    if (state.phase === 1) {
      void advanceToSecondParent();
      return;
    }
    void createCharacterFromRoll();
  });

  buildRerollButton.addEventListener("click", () => {
    buildSequenceToken += 1;
    resetRollState(true);
    showSocioPrompt();
  });

  chapterContinueButton.addEventListener("click", () => {
    void continueFromChapter();
  });

  childhoodContinueButton.addEventListener("click", () => {
    void continueFromChildhood();
  });

  void (async () => {
    await loadParentageChart();
    state.phase = 1;
    state.parentRolls = [null, null];
    state.workbookCardId = "";
    state.workbookContext = {};
    state.parentageRows = [];
    state.startingWealth = 0;
    resetRollState(false);

    try {
      const response = await fetch("/api/character-cards/me", { credentials: "include" });
      const payload = await response.json().catch(() => null);
      const data = payload?.data || {};
      state.canDraft = Boolean(data.can_draft);
      state.cards = Array.isArray(data.cards) ? data.cards : [];
      state.activeCharacter = data.active_character || null;
      if (state.activeCharacter?.character_card_id) {
        state.workbookCardId = String(state.activeCharacter.character_card_id || "");
        state.workbookContext = { ...(state.activeCharacter.workbook_context || {}) };
      }
      refreshCharacterTray();

      const requestedNewCharacter = new URLSearchParams(window.location.search).get("new_character") === "1";
      if (requestedNewCharacter) {
        const url = new URL(window.location.href);
        url.searchParams.delete("new_character");
        window.history.replaceState({}, "", url.toString());
      }

      const shouldShowOnboarding = state.canDraft && state.cards.length === 0;
      const hasSeenIntro = localStorage.getItem(introKey) === "1";
      const hasSeenSocio = localStorage.getItem(socioKey) === "1";
      if (requestedNewCharacter || shouldShowOnboarding || (!hasSeenIntro && !hasSeenSocio)) {
        showIntro();
        return;
      }
    } catch (error) {
      console.error("catharsis onboarding state failed", error);
      refreshCharacterTray();
    }

    overlay.hidden = true;
  })();
})();
