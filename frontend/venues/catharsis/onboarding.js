(function () {
  const introKey = "victory:catharsis:onboarding:intro-seen";
  const socioKey = "victory:catharsis:onboarding:socio-seen";

  const overlay = document.getElementById("catharsis-onboarding");
  const introPanel = document.getElementById("catharsis-onboarding-intro");
  const socioPanel = document.getElementById("catharsis-onboarding-socio");
  const buildPanel = document.getElementById("catharsis-onboarding-build");
  const chapterPanel = document.getElementById("catharsis-onboarding-chapter");
  const introButton = document.getElementById("catharsis-onboarding-continue");
  const dismissButton = document.getElementById("catharsis-onboarding-dismiss");
  const socioStatus = document.getElementById("catharsis-onboarding-socio-status");
  const chapterContinueButton = document.getElementById("catharsis-onboarding-chapter-continue");
  const chapter2Panel = document.getElementById("catharsis-onboarding-chapter2");
  const chapter3ArchetypePanel = document.getElementById("catharsis-onboarding-chapter3-archetype");
  const chapter3Content = document.getElementById("catharsis-chapter3-content");
  const chapter4Panel = document.getElementById("catharsis-onboarding-chapter4");
  const chapter4ContinueButton = document.getElementById("catharsis-onboarding-chapter4-continue");
  const coinFlipPanel = document.getElementById("catharsis-onboarding-coinflip");
  const coinFlipTitle = document.getElementById("catharsis-coinflip-title");
  const coinFlipCopy = document.getElementById("catharsis-coinflip-copy");
  const coinFlipFace = document.getElementById("catharsis-coinflip-face");
  const coinFlipButton = document.getElementById("catharsis-coinflip-button");
  const coinFlipStatus = document.getElementById("catharsis-coinflip-status");
  const coinFlipContinueButton = document.getElementById("catharsis-coinflip-continue");
  const ch2StageName = document.getElementById("catharsis-chapter2-stage-name");
  const ch2Intro = document.getElementById("catharsis-chapter2-intro");
  const ch2Fp = document.getElementById("catharsis-chapter2-fp");
  const ch2StageNumber = document.getElementById("catharsis-chapter2-stage-number");
  const ch2PrimaryAttrLabel = document.getElementById("catharsis-chapter2-primary-attr-label");
  const ch2PrimaryAttrTotal = document.getElementById("catharsis-chapter2-primary-attr-total");
  const ch2PrimaryLabel = document.getElementById("catharsis-chapter2-primary-label");
  const ch2D4 = document.getElementById("catharsis-chapter2-d4");
  const ch2D6 = document.getElementById("catharsis-chapter2-d6");
  const ch2Final = document.getElementById("catharsis-chapter2-final");
  const ch2RollD4Button = document.getElementById("catharsis-chapter2-roll-d4");
  const ch2RollD6Button = document.getElementById("catharsis-chapter2-roll-d6");
  const ch2AllocationWrap = document.getElementById("catharsis-chapter2-allocation");
  const ch2AllocationRows = document.getElementById("catharsis-chapter2-allocation-rows");
  const ch2BonusChoices = document.getElementById("catharsis-chapter2-bonus-choices");
  const ch2RepresentationWrap = document.getElementById("catharsis-chapter2-representation-wrap");
  const ch2Representation = document.getElementById("catharsis-chapter2-representation");
  const ch2CompanionWrap = document.getElementById("catharsis-chapter2-companion-wrap");
  const ch2CompanionToggle = document.getElementById("catharsis-chapter2-companion-toggle");
  const ch2CompanionLabel = document.getElementById("catharsis-chapter2-companion-label");
  const ch2CompanionName = document.getElementById("catharsis-chapter2-companion-name");
  const ch2Traits = document.getElementById("catharsis-chapter2-traits");
  const ch2Status = document.getElementById("catharsis-chapter2-status");
  const ch2CompleteButton = document.getElementById("catharsis-chapter2-complete");
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
  const characterTrayButton = document.getElementById("character-tray-button");
  const characterTrayLabel = document.getElementById("character-tray-label");

  if (
    !overlay ||
    !introPanel ||
    !socioPanel ||
    !buildPanel ||
    !chapterPanel ||
    !chapter2Panel ||
    !chapter3ArchetypePanel ||
    !chapter3Content ||
    !chapter4Panel ||
    !chapter4ContinueButton ||
    !coinFlipPanel ||
    !coinFlipTitle ||
    !coinFlipCopy ||
    !coinFlipFace ||
    !coinFlipButton ||
    !coinFlipStatus ||
    !coinFlipContinueButton ||
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
    !ch2StageName ||
    !ch2Intro ||
    !ch2Fp ||
    !ch2StageNumber ||
    !ch2PrimaryAttrLabel ||
    !ch2PrimaryAttrTotal ||
    !ch2PrimaryLabel ||
    !ch2D4 ||
    !ch2D6 ||
    !ch2Final ||
    !ch2RollD4Button ||
    !ch2RollD6Button ||
    !ch2AllocationWrap ||
    !ch2AllocationRows ||
    !ch2BonusChoices ||
    !ch2RepresentationWrap ||
    !ch2Representation ||
    !ch2CompanionWrap ||
    !ch2CompanionToggle ||
    !ch2CompanionLabel ||
    !ch2CompanionName ||
    !ch2Traits ||
    !ch2Status ||
    !ch2CompleteButton
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
    chapter2Rules: null,
    chapter2: null,
    coinFlipQueue: [],
    coinFlipResults: {},
    chapter3Catalog: null,
    chapter3: {
      mode: "intro",
      selectedKey: "",
      pendingConfirmKey: "",
      confirmSource: "direct",
      quiz: {
        currentQuestion: 0,
        scores: {},
        selectedAnswers: [],
        selectedThisQuestion: [],
      },
    },
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

  const setCh2Status = (text) => {
    if (ch2Status) ch2Status.textContent = text || "";
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

  const hideAllOnboardingPanels = () => {
    introPanel.hidden = true;
    socioPanel.hidden = true;
    buildPanel.hidden = true;
    chapterPanel.hidden = true;
    chapter2Panel.hidden = true;
    chapter3ArchetypePanel.hidden = true;
    chapter4Panel.hidden = true;
    coinFlipPanel.hidden = true;
  };

  const showChapterPanel = () => {
    hideAllOnboardingPanels();
    chapterPanel.hidden = false;
    overlay.hidden = false;
    buildSequenceToken += 1;
    document.body.classList.add("modal-open");
    chapterContinueButton.focus();
  };

  // --- Chapter 2: Lifepath (Kernel 54) ---

  const loadChapter2Rules = async () => {
    if (state.chapter2Rules) return state.chapter2Rules;
    const response = await fetch("/api/characters/chapter2-rules", { credentials: "include" });
    const payload = await response.json().catch(() => null);
    if (!response.ok || !payload?.ok || !payload?.data) {
      throw new Error(payload?.data?.error || payload?.error || `Chapter 2 rules load failed (${response.status}).`);
    }
    state.chapter2Rules = payload.data;
    return state.chapter2Rules;
  };

  const chapter2StageRules = (stageNumber) => {
    const stages = state.chapter2Rules?.stages || [];
    return stages.find((stage) => Number(stage.stage_number) === Number(stageNumber)) || null;
  };

  const chapter2Ctx = () => {
    const wb = getCurrentWorkbookContext();
    return wb.chapter2 && typeof wb.chapter2 === "object" ? wb.chapter2 : null;
  };

  const ch2DefaultState = () => ({
    version: state.chapter2Rules?.version || "",
    starting_fp: Number(state.chapter2Rules?.starting_fp) || 12,
    fp_balance: Number(state.chapter2Rules?.starting_fp) || 12,
    current_stage: 1,
    attributes: {},
    stages: {},
  });

  const ch2Local = {
    stageNumber: 1,
    stageRules: null,
    rolls: null,
    useEnhancement: false,
    bonusChoiceId: "",
    traitIds: [],
    companionPurchased: false,
    companionTarget: "",
    representationText: "",
    allocation: {},
  };

  const ch2ResetLocalSelections = () => {
    ch2Local.useEnhancement = false;
    ch2Local.bonusChoiceId = "";
    ch2Local.traitIds = [];
    ch2Local.companionPurchased = false;
    ch2Local.companionTarget = "";
    ch2Local.representationText = "";
    ch2Local.allocation = {};
  };

  const ch2FinalRoll = () => {
    if (!ch2Local.rolls) return null;
    const d4 = Number(ch2Local.rolls.d4);
    const d6 = ch2Local.useEnhancement && ch2Local.rolls.d6 != null ? Number(ch2Local.rolls.d6) : null;
    if (d6 == null) return d4;
    return Math.max(d4, d6);
  };

  const ch2RenderBonusChoices = () => {
    const stage = ch2Local.stageRules;
    ch2BonusChoices.innerHTML = (stage.bonus_choices || []).map((choice) => `
      <div class="catharsis-chapter2-bonus-choice">
        <input type="radio" name="ch2-bonus" id="ch2-bonus-${escapeHtml(choice.id)}" value="${escapeHtml(choice.id)}" ${ch2Local.bonusChoiceId === choice.id ? "checked" : ""} />
        <label for="ch2-bonus-${escapeHtml(choice.id)}">
          <strong>${escapeHtml(choice.name)} (+${escapeHtml(choice.modifier)} ${escapeHtml(choice.target_attribute)})</strong>
        </label>
      </div>
    `).join("");
    ch2BonusChoices.querySelectorAll("input[name=ch2-bonus]").forEach((input) => {
      input.addEventListener("change", () => {
        ch2Local.bonusChoiceId = input.value;
        ch2RenderCompleteGate();
      });
    });
  };

  const ch2RenderTraits = () => {
    const stage = ch2Local.stageRules;
    const traits = stage.traits || [];
    if (!traits.length) {
      ch2Traits.innerHTML = "";
      return;
    }
    ch2Traits.innerHTML = `<p class="catharsis-onboarding__hint">Optional traits (0-2, descriptive only):</p>` + traits.map((trait) => `
      <div class="catharsis-chapter2-trait">
        <input type="checkbox" id="ch2-trait-${escapeHtml(trait.id)}" value="${escapeHtml(trait.id)}" ${ch2Local.traitIds.includes(trait.id) ? "checked" : ""} />
        <label for="ch2-trait-${escapeHtml(trait.id)}">
          <strong>${escapeHtml(trait.name)} <span class="catharsis-chapter2-trait-cost">(${escapeHtml(trait.cost_fp)} FP)</span></strong>
          <span>${escapeHtml(trait.description)}</span>
        </label>
      </div>
    `).join("");
    ch2Traits.querySelectorAll("input[type=checkbox]").forEach((input) => {
      input.addEventListener("change", () => {
        if (input.checked) {
          if (ch2Local.traitIds.length >= 2) {
            input.checked = false;
            setCh2Status("At most 2 traits per stage.");
            return;
          }
          ch2Local.traitIds.push(input.value);
        } else {
          ch2Local.traitIds = ch2Local.traitIds.filter((id) => id !== input.value);
        }
        ch2RenderCompleteGate();
      });
    });
  };

  const ch2RenderAllocation = () => {
    const stage = ch2Local.stageRules;
    if (!stage.has_special_allocation) {
      ch2AllocationWrap.hidden = true;
      return;
    }
    ch2AllocationWrap.hidden = false;
    const finalRoll = ch2FinalRoll();
    const attributes = state.chapter2Rules?.attributes || ["Spirit", "Might", "Empathy", "Grace", "Awareness", "Intellect", "Lore", "Presence", "Craft", "Resolve"];
    ch2AllocationRows.innerHTML = attributes.map((attr) => `
      <div class="catharsis-chapter2-allocation-row">
        <span>${escapeHtml(attr)}</span>
        <input type="number" min="0" max="${finalRoll || 0}" value="${ch2Local.allocation[attr] || 0}" data-attr="${escapeHtml(attr)}" />
      </div>
    `).join("");
    ch2AllocationRows.querySelectorAll("input[type=number]").forEach((input) => {
      input.addEventListener("input", () => {
        const attr = input.dataset.attr;
        const value = Math.max(0, Number(input.value) || 0);
        ch2Local.allocation[attr] = value;
        ch2RenderCompleteGate();
      });
    });
  };

  const ch2RenderCompleteGate = () => {
    const stage = ch2Local.stageRules;
    const finalRoll = ch2FinalRoll();
    let ready = finalRoll != null && Boolean(ch2Local.bonusChoiceId);

    if (stage.has_special_allocation && ready) {
      const total = Object.values(ch2Local.allocation).reduce((sum, v) => sum + (Number(v) || 0), 0);
      const craft = Number(ch2Local.allocation.Craft) || 0;
      ready = total === finalRoll && craft >= 1;
    }

    ch2CompleteButton.disabled = !ready;
  };

  const ch2RenderStagePanel = () => {
    const stage = ch2Local.stageRules;
    const ctx = chapter2Ctx() || ch2DefaultState();

    ch2StageName.textContent = `Stage ${stage.stage_number}: ${stage.name}`;
    ch2Intro.textContent = stage.canonical_intro || stage.purpose || "";
    ch2Fp.textContent = String(ctx.fp_balance ?? "--");
    ch2StageNumber.textContent = `${stage.stage_number} / 10`;
    ch2PrimaryAttrLabel.textContent = stage.has_special_allocation ? "Craft (min)" : stage.primary_attribute;
    ch2PrimaryAttrTotal.textContent = String((ctx.attributes || {})[stage.has_special_allocation ? "Craft" : stage.primary_attribute] ?? 0);
    ch2PrimaryLabel.textContent = `Roll for ${stage.name}`;

    ch2D4.textContent = ch2Local.rolls ? String(ch2Local.rolls.d4) : "--";
    ch2D6.textContent = ch2Local.rolls?.d6 != null ? String(ch2Local.rolls.d6) : "--";
    ch2Final.textContent = ch2FinalRoll() != null ? String(ch2FinalRoll()) : "--";

    ch2RollD4Button.disabled = Boolean(ch2Local.rolls);
    ch2RollD6Button.disabled = !ch2Local.rolls || ch2Local.rolls.d6 != null || Number(ctx.fp_balance) < 3;
    ch2RollD6Button.hidden = false;

    ch2RepresentationWrap.hidden = !stage.has_representation_field;
    if (stage.has_representation_field) {
      ch2Representation.value = ch2Local.representationText;
    }

    ch2CompanionWrap.hidden = !stage.companion;
    if (stage.companion) {
      ch2CompanionLabel.textContent = `${stage.companion.name} (${stage.companion.cost_fp} FP) — ${stage.companion.description}`;
      ch2CompanionToggle.checked = ch2Local.companionPurchased;
      ch2CompanionName.hidden = !ch2Local.companionPurchased;
      ch2CompanionName.value = ch2Local.companionTarget;
    }

    ch2RenderBonusChoices();
    ch2RenderTraits();
    ch2RenderAllocation();
    ch2RenderCompleteGate();
    setCh2Status("");
  };

  const showChapter2Panel = async (stageNumber) => {
    hideAllOnboardingPanels();
    overlay.hidden = false;
    document.body.classList.add("modal-open");
    setCh2Status("Loading stage...");
    chapter2Panel.hidden = false;

    try {
      await loadChapter2Rules();
    } catch (error) {
      console.error("chapter2 rules load failed", error);
      setCh2Status("Could not load Chapter II rules. Try again.");
      return;
    }

    const stage = chapter2StageRules(stageNumber);
    if (!stage) {
      setCh2Status(`Unknown stage ${stageNumber}.`);
      return;
    }

    ch2Local.stageNumber = stageNumber;
    ch2Local.stageRules = stage;
    ch2Local.rolls = null;
    ch2ResetLocalSelections();

    const ctx = chapter2Ctx();
    const existingStage = ctx?.stages?.[String(stageNumber)];
    if (existingStage?.completed) {
      // Already committed: show as read-only completed summary and advance.
      ch2Local.rolls = { d4: existingStage.final_roll, d6: null };
      ch2Local.useEnhancement = Boolean(existingStage.enhancement_used);
      ch2RenderStagePanel();
      setCh2Status(`Stage ${stageNumber} is already complete.`);
      ch2RollD4Button.disabled = true;
      ch2RollD6Button.disabled = true;
      ch2CompleteButton.disabled = false;
      ch2CompleteButton.textContent = stageNumber >= 10 ? "Continue to Chapter III" : "Continue to next stage";
      return;
    }

    ch2CompleteButton.textContent = "Complete stage";

    // The stage isn't committed yet, but a d4 (and maybe d6) may already be
    // locked server-side from a prior visit. Restore it instead of showing a
    // blank roll, so a refresh never looks like the roll was lost or rerolled.
    try {
      const params = new URLSearchParams({
        character_card_id: state.workbookCardId,
        stage_number: String(stageNumber),
      });
      const response = await fetch(`/api/character-cards/chapter2-roll?${params.toString()}`, { credentials: "include" });
      const payload = await response.json().catch(() => null);
      const rolls = payload?.ok ? payload?.data?.rolls : null;
      if (rolls) {
        ch2Local.rolls = { d4: Number(rolls.d4), d6: rolls.d6 != null ? Number(rolls.d6) : null };
        ch2Local.useEnhancement = rolls.d6 != null;
      }
    } catch (error) {
      console.error("chapter2 roll status load failed", error);
    }

    ch2RenderStagePanel();
  };

  const ch2RequestRoll = async (dieType) => {
    const response = await fetch("/api/character-cards/chapter2-roll", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        character_card_id: state.workbookCardId,
        stage_number: ch2Local.stageNumber,
        die_type: dieType,
      }),
    });
    const payload = await response.json().catch(() => null);
    if (!response.ok || !payload?.ok || !payload?.data) {
      throw new Error(payload?.data?.error || payload?.error || `Chapter 2 roll failed (${response.status}).`);
    }
    return payload.data;
  };

  const ch2RollD4 = async () => {
    if (ch2Local.rolls) return;
    ch2RollD4Button.disabled = true;
    setCh2Status("Rolling...");
    const token = buildSequenceToken;
    try {
      const startedAt = Date.now();
      while (Date.now() - startedAt < 480) {
        if (token !== buildSequenceToken) return;
        ch2D4.textContent = String(randomDie(4));
        await sleep(58);
      }
      const result = await ch2RequestRoll("d4");
      if (token !== buildSequenceToken) return;
      ch2Local.rolls = { d4: Number(result.value), d6: null };
      ch2RenderStagePanel();
      setCh2Status("Roll locked.");
    } catch (error) {
      console.error("chapter2 d4 roll failed", error);
      setCh2Status("Could not roll. Try again.");
      ch2RollD4Button.disabled = false;
    }
  };

  const ch2RollD6 = async () => {
    if (!ch2Local.rolls || ch2Local.rolls.d6 != null) return;
    ch2RollD6Button.disabled = true;
    setCh2Status("Enhancing...");
    const token = buildSequenceToken;
    try {
      const startedAt = Date.now();
      while (Date.now() - startedAt < 480) {
        if (token !== buildSequenceToken) return;
        ch2D6.textContent = String(randomDie(6));
        await sleep(58);
      }
      const result = await ch2RequestRoll("d6");
      if (token !== buildSequenceToken) return;
      ch2Local.rolls.d6 = Number(result.value);
      ch2Local.useEnhancement = true;
      ch2RenderStagePanel();
      setCh2Status("Enhancement locked.");
    } catch (error) {
      console.error("chapter2 d6 roll failed", error);
      setCh2Status("Could not enhance. Try again.");
      ch2RollD6Button.disabled = false;
    }
  };

  const ch2CompleteStage = async () => {
    const ctx = chapter2Ctx();
    const stageKey = String(ch2Local.stageNumber);
    if (ctx?.stages?.[stageKey]?.completed) {
      const nextStage = ch2Local.stageNumber + 1;
      if (ch2Local.stageNumber >= 10) {
        showChapter3Panel();
        return;
      }
      await showChapter2Panel(nextStage);
      return;
    }

    ch2CompleteButton.disabled = true;
    setCh2Status("Saving stage...");
    try {
      const response = await fetch("/api/character-cards/chapter2-stage", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          character_card_id: state.workbookCardId,
          stage_number: ch2Local.stageNumber,
          use_enhancement: ch2Local.useEnhancement,
          bonus_choice_id: ch2Local.bonusChoiceId,
          companion_purchased: ch2Local.companionPurchased,
          companion_target: ch2Local.companionTarget,
          trait_ids: ch2Local.traitIds,
          allocation: ch2Local.stageRules.has_special_allocation ? ch2Local.allocation : undefined,
          representation_text: ch2Local.representationText,
        }),
      });
      const payload = await response.json().catch(() => null);
      if (!response.ok || !payload?.ok || !payload?.data) {
        throw new Error(payload?.data?.error || payload?.error || `Stage commit failed (${response.status}).`);
      }

      // The stage endpoint already persisted workbook_context (including the
      // chapter2 sub-object) server-side. Only the flat stage result comes
      // back here; refresh state.workbookContext afterward instead of
      // patching it from this response, so we never clobber it with a stale
      // client-side copy.
      const result = payload.data;

      const stageEntry = {
        page_key: "history",
        entry_type: "chapter2_stage",
        title: `Stage ${ch2Local.stageNumber}: ${ch2Local.stageRules.name}`,
        body: `Final roll ${result.final_roll}. Bonus: ${ch2Local.bonusChoiceId}.`,
        stage_number: ch2Local.stageNumber,
        sort_order: ch2Local.stageNumber,
        payload: {
          source: "catharsis",
          ruleset_key: "socio",
          ruleset_version: "1.1",
          chapter2_result: result,
        },
      };

      await postWorkbookEvents(state.workbookCardId, {
        character_card_id: state.workbookCardId,
        module_key: "socio",
        module_status: result.completed && ch2Local.stageNumber >= 10 ? "complete" : "draft",
        current_stage: result.next_stage ? 2 : 3,
        current_event: result.next_stage ? `chapter2_stage_${result.next_stage}` : "chapter2_complete",
        workbook_status: "draft",
        entries: [stageEntry],
      });

      await refreshWorkbookContextFromServer();

      if (result.next_stage) {
        await showChapter2Panel(result.next_stage);
      } else {
        await showChapter3ArchetypePanel();
      }
    } catch (error) {
      console.error("chapter2 stage commit failed", error);
      setCh2Status(String(error.message || "Could not save this stage."));
      ch2CompleteButton.disabled = false;
    }
  };

  const showChapter4Panel = () => {
    hideAllOnboardingPanels();
    chapter4Panel.hidden = false;
    overlay.hidden = false;
    document.body.classList.add("modal-open");
    chapter4ContinueButton.focus();
  };

  // --- Chapter 3: Character Archetypes (direct selection + quiz) ---

  const DISPLAY_PERCENT_BUMP = 12;

  const loadChapter3Catalog = async () => {
    if (state.chapter3Catalog) return state.chapter3Catalog;
    const response = await fetch("/api/characters/chapter3-archetypes", { credentials: "include" });
    const payload = await response.json().catch(() => null);
    if (!response.ok || !payload?.ok || !payload?.data) {
      throw new Error(payload?.data?.error || payload?.error || `Chapter 3 archetype catalog load failed (${response.status}).`);
    }
    const list = Array.isArray(payload.data.archetypes) ? payload.data.archetypes : [];
    list.sort((a, b) => Number(a.display_order) - Number(b.display_order));
    state.chapter3Catalog = list;
    return list;
  };

  const ch3ArchetypeByKey = (key) => (state.chapter3Catalog || []).find((a) => a.key === key) || null;

  const ch3CalculateMaxPossibleScores = () => {
    const maxScores = {};
    (state.chapter3Catalog || []).forEach((a) => { maxScores[a.key] = 0; });
    const questions = window.VictoryChapter3QuizData.questions;
    questions.forEach((question) => {
      const maxSelections = question.maxSelections || 1;
      Object.keys(maxScores).forEach((key) => {
        const possible = question.answers
          .map((answer) => (answer.scores && answer.scores[key]) ? answer.scores[key] : 0)
          .sort((a, b) => b - a)
          .slice(0, maxSelections)
          .reduce((sum, value) => sum + value, 0);
        maxScores[key] += possible;
      });
    });
    return maxScores;
  };

  const ch3GetAllResults = () => {
    const maxScores = ch3CalculateMaxPossibleScores();
    const scores = state.chapter3.quiz.scores;
    return Object.keys(maxScores)
      .map((key) => {
        const points = scores[key] || 0;
        const max = maxScores[key] || 1;
        const rawPercent = Math.round((points / max) * 100);
        return { key, points, max, rawPercent, percent: rawPercent + DISPLAY_PERCENT_BUMP };
      })
      .sort((a, b) => b.rawPercent - a.rawPercent || b.points - a.points || a.key.localeCompare(b.key));
  };

  const ch3GetConfidenceLabel = (primaryPercent, secondPercent) => {
    const gap = primaryPercent - secondPercent;
    if (primaryPercent >= 95 && gap >= 15) return "Very strong";
    if (primaryPercent >= 82 && gap >= 8) return "Strong";
    if (primaryPercent >= 65) return "Moderate";
    return "Emerging";
  };

  const ch3GetBlendedProfileItems = (primary, second, third, field, total = 5) => {
    const primaryItems = Array.isArray(primary?.[field]) ? primary[field] : [];
    const secondItems = Array.isArray(second?.[field]) ? second[field] : [];
    const thirdItems = Array.isArray(third?.[field]) ? third[field] : [];
    const chosen = [];
    const addItem = (item) => { if (item && !chosen.includes(item)) chosen.push(item); };
    primaryItems.slice(0, 3).forEach(addItem);
    addItem(secondItems[0]);
    addItem(thirdItems[0]);
    [...primaryItems.slice(3), ...secondItems.slice(1), ...thirdItems.slice(1)].forEach((item) => {
      if (chosen.length < total) addItem(item);
    });
    return chosen.slice(0, total);
  };

  const ch3GetSelfGuess = () => {
    const questions = window.VictoryChapter3QuizData.questions;
    const finalIndex = questions.length - 1;
    const finalSelection = state.chapter3.quiz.selectedAnswers.find((item) => item.question === finalIndex);
    if (!finalSelection) return null;
    const answer = questions[finalIndex].answers[finalSelection.answer];
    return answer?.selfGuess || null;
  };

  const ch3RenderList = (items) => (items || []).map((item) => `<li>${escapeHtml(item)}</li>`).join("");

  const ch3RenderPills = (items) => (items || []).map((item) => `<span class="ch3-pill">${escapeHtml(item)}</span>`).join("");

  const ch3RenderBars = (allResults, options = {}) => {
    const limit = options.limit || null;
    const results = limit ? allResults.slice(0, limit) : allResults;
    return `
      <div class="ch3-profile-section">
        <h3>${escapeHtml(options.title || "Archetype Breakdown")}</h3>
        ${options.description ? `<p class="small">${escapeHtml(options.description)}</p>` : ""}
        ${results.map((result) => {
          const archetype = ch3ArchetypeByKey(result.key);
          const title = archetype?.title || result.key;
          const barWidth = Math.max(3, Math.min(100, result.percent));
          return `
            <div class="ch3-bar-row">
              <div class="ch3-bar-label"><span>${escapeHtml(title)}</span><strong>${result.percent}%</strong></div>
              <div class="ch3-bar-track"><div class="ch3-bar-fill" style="width:${barWidth}%"></div></div>
            </div>
          `;
        }).join("")}
      </div>
    `;
  };

  const ch3RenderPrediction = (primaryKey) => {
    const selfGuess = ch3GetSelfGuess();
    if (!selfGuess) return "";
    const guessed = ch3ArchetypeByKey(selfGuess);
    const primary = ch3ArchetypeByKey(primaryKey);
    if (!guessed || !primary) return "";
    if (selfGuess === primaryKey) {
      return `
        <div class="ch3-profile-section">
          <h3>How You See Yourself</h3>
          <p><strong>You called it.</strong> The value you picked lines up with your strongest result: <strong>${escapeHtml(primary.title)}</strong>.</p>
          <p>That usually means the way you like to see yourself is pretty close to the pattern your answers showed.</p>
        </div>
      `;
    }
    const guessedNoun = guessed.title.replace(/^The\s+/i, "");
    const primaryNoun = primary.title.replace(/^The\s+/i, "");
    return `
      <div class="ch3-profile-section">
        <h3>How You See Yourself</h3>
        <p>You chose a value connected to <strong>${escapeHtml(guessed.title)}</strong>, but your answers pointed more strongly toward <strong>${escapeHtml(primary.title)}</strong>.</p>
        <p>That does not mean you were wrong. It may mean <strong>${escapeHtml(guessedNoun)}</strong> is how you like to think of yourself, while <strong>${escapeHtml(primaryNoun)}</strong> is closer to the pattern your habits reveal.</p>
        <p><strong>The ${escapeHtml(guessedNoun)} in you:</strong> ${escapeHtml(guessed.short_description || guessed.primary?.intro || "")}</p>
      </div>
    `;
  };

  const ch3ArchetypeOptionsHtml = (selectedKey) => (state.chapter3Catalog || [])
    .map((a) => `<option value="${escapeHtml(a.key)}" ${a.key === selectedKey ? "selected" : ""}>${escapeHtml(a.title)}</option>`)
    .join("");

  const ch3RenderIntro = () => {
    const selectedKey = state.chapter3.selectedKey || (state.chapter3Catalog || [])[0]?.key || "";
    state.chapter3.selectedKey = selectedKey;
    const selected = ch3ArchetypeByKey(selectedKey);
    chapter3Content.innerHTML = `
      <p class="eyebrow">Chapter III</p>
      <h2>Character Archetypes</h2>
      <p>Answer as your character. What would this character do, notice, value, fear, or choose in each situation?</p>
      <div class="ch3-select">
        <label for="ch3-archetype-select">Choose an Archetype</label>
        <select id="ch3-archetype-select">${ch3ArchetypeOptionsHtml(selectedKey)}</select>
      </div>
      ${selected ? `<div class="ch3-echo-card"><p>${escapeHtml(selected.echo)}</p></div>` : ""}
      <p id="ch3-status" class="catharsis-onboarding__status" aria-live="polite"></p>
      <div class="catharsis-onboarding__actions">
        <button id="ch3-confirm-direct" type="button">Confirm Archetype</button>
        <button id="ch3-take-quiz" type="button">Take the Archetype Quiz</button>
      </div>
    `;

    document.getElementById("ch3-archetype-select").addEventListener("change", (event) => {
      state.chapter3.selectedKey = event.target.value;
      ch3RenderIntro();
    });
    document.getElementById("ch3-confirm-direct").addEventListener("click", () => {
      state.chapter3.pendingConfirmKey = state.chapter3.selectedKey;
      state.chapter3.confirmSource = "direct";
      state.chapter3.mode = "confirm";
      renderChapter3Content();
    });
    document.getElementById("ch3-take-quiz").addEventListener("click", () => {
      state.chapter3.quiz = { currentQuestion: 0, scores: {}, selectedAnswers: [], selectedThisQuestion: [] };
      (state.chapter3Catalog || []).forEach((a) => { state.chapter3.quiz.scores[a.key] = 0; });
      state.chapter3.mode = "quiz-question";
      renderChapter3Content();
    });
  };

  const ch3RenderQuizQuestion = () => {
    const questions = window.VictoryChapter3QuizData.questions;
    const index = state.chapter3.quiz.currentQuestion;
    const q = questions[index];
    const maxSelections = q.maxSelections || 1;
    state.chapter3.quiz.selectedThisQuestion = [];

    chapter3Content.innerHTML = `
      <p class="eyebrow">Chapter III: Archetype Quiz</p>
      <p class="small">Question ${index + 1} of ${questions.length}</p>
      <h2>${escapeHtml(q.text)}</h2>
      <p class="catharsis-onboarding__hint">${maxSelections === 1 ? "Choose one answer." : `Choose up to ${maxSelections} answers.`}</p>
      <div class="ch3-answers" id="ch3-answers"></div>
      <div class="catharsis-onboarding__actions">
        <button id="ch3-quiz-continue" type="button" disabled>Continue</button>
      </div>
    `;

    const answersDiv = document.getElementById("ch3-answers");
    q.answers.forEach((answer, answerIndex) => {
      const button = document.createElement("button");
      button.type = "button";
      button.className = "ch3-answer-btn";
      button.textContent = answer.text;
      button.addEventListener("click", () => {
        const selected = state.chapter3.quiz.selectedThisQuestion;
        const already = selected.includes(answerIndex);
        if (already) {
          state.chapter3.quiz.selectedThisQuestion = selected.filter((i) => i !== answerIndex);
          button.classList.remove("is-selected");
        } else if (maxSelections === 1) {
          state.chapter3.quiz.selectedThisQuestion = [answerIndex];
          answersDiv.querySelectorAll(".ch3-answer-btn").forEach((btn) => btn.classList.remove("is-selected"));
          button.classList.add("is-selected");
        } else if (selected.length < maxSelections) {
          state.chapter3.quiz.selectedThisQuestion.push(answerIndex);
          button.classList.add("is-selected");
        }
        document.getElementById("ch3-quiz-continue").disabled = state.chapter3.quiz.selectedThisQuestion.length === 0;
      });
      answersDiv.appendChild(button);
    });

    document.getElementById("ch3-quiz-continue").addEventListener("click", () => {
      state.chapter3.quiz.selectedThisQuestion.forEach((answerIndex) => {
        const answer = q.answers[answerIndex];
        state.chapter3.quiz.selectedAnswers.push({ question: index, answer: answerIndex });
        Object.entries(answer.scores || {}).forEach(([key, points]) => {
          state.chapter3.quiz.scores[key] = (state.chapter3.quiz.scores[key] || 0) + points;
        });
      });
      state.chapter3.quiz.currentQuestion += 1;
      if (state.chapter3.quiz.currentQuestion >= questions.length) {
        state.chapter3.mode = "quiz-result";
      }
      renderChapter3Content();
    });
  };

  const ch3RenderQuizResult = () => {
    const allResults = ch3GetAllResults();
    const top = allResults.slice(0, 3);
    const primaryResult = top[0];
    const secondResult = top[1];
    const thirdResult = top[2];
    const primary = ch3ArchetypeByKey(primaryResult.key);
    const second = ch3ArchetypeByKey(secondResult.key);
    const third = ch3ArchetypeByKey(thirdResult.key);
    const confidence = ch3GetConfidenceLabel(primaryResult.percent, secondResult.percent);

    chapter3Content.innerHTML = `
      <p class="eyebrow">Chapter III: Archetype Quiz Result</p>
      <p class="small">Your strongest Socio-Archetype pattern is</p>
      <h2>${escapeHtml(primary.title)}</h2>
      <p><strong>${primaryResult.percent}% Resonance</strong> &middot; Result confidence: ${escapeHtml(confidence)}</p>
      <div class="ch3-echo-card">
        <blockquote>${escapeHtml(primary.motto || "")}</blockquote>
        <p>${escapeHtml(primary.short_description || "")}</p>
        <div>${ch3RenderPills(primary.core_drives)}</div>
      </div>

      ${ch3RenderBars(allResults, { limit: 5, title: "Top Patterns", description: "Your five strongest normalized archetype signals." })}

      <div class="ch3-profile-section">
        <h3>Your Echoes</h3>
        <div class="ch3-echo-grid">
          <div class="ch3-echo-card"><h4>${escapeHtml(second.title)} &mdash; ${secondResult.percent}%</h4><p>${escapeHtml(second.echo)}</p></div>
          <div class="ch3-echo-card"><h4>${escapeHtml(third.title)} &mdash; ${thirdResult.percent}%</h4><p>${escapeHtml(third.echo)}</p></div>
        </div>
      </div>

      <div class="ch3-profile-grid">
        <div class="ch3-profile-section">
          <h3>Primary Profile</h3>
          <p>${escapeHtml(primary.primary?.intro || "")}</p>
          <p>${escapeHtml(primary.primary?.strengths || "")}</p>
          <p>${escapeHtml(primary.primary?.challenges || "")}</p>
          <p>${escapeHtml(primary.primary?.socio || "")}</p>
        </div>
        <div class="ch3-profile-section">
          <h3>Personality Pattern</h3>
          <p><strong>${escapeHtml(primary.hexaco || "")}</strong></p>
          <p>${escapeHtml(primary.hexaco_plain || "")}</p>
          <h3>What People Notice First</h3>
          <p>What people first notice about you is your <strong>${escapeHtml(primary.primary_attribute || "")}</strong>, followed by your <strong>${escapeHtml(primary.secondary_attribute || "")}</strong>.</p>
          <p>Your key skill is your ability to use <strong>${escapeHtml(primary.key_skill || "")}</strong>.</p>
        </div>
      </div>

      <div class="ch3-profile-grid">
        <div class="ch3-profile-section"><h3>What You Notice</h3><p>${escapeHtml(primary.notices || "")}</p></div>
        <div class="ch3-profile-section"><h3>In A Group</h3><p>${escapeHtml(primary.group_role || "")}</p></div>
      </div>

      <div class="ch3-profile-grid">
        <div class="ch3-profile-section"><h3>Natural Strengths</h3><ul>${ch3RenderList(ch3GetBlendedProfileItems(primary, second, third, "strengths_list"))}</ul></div>
        <div class="ch3-profile-section"><h3>Growth Edges</h3><ul>${ch3RenderList(ch3GetBlendedProfileItems(primary, second, third, "growth_edges"))}</ul></div>
      </div>

      <div class="ch3-profile-grid">
        <div class="ch3-profile-section"><h3>Questions To Ask Yourself</h3><ul>${ch3RenderList(primary.questions_to_ask)}</ul></div>
        <div class="ch3-profile-section"><h3>How To Help Yourself</h3><ul>${ch3RenderList(primary.how_to_help_yourself)}</ul></div>
      </div>

      <div class="ch3-profile-grid">
        <div class="ch3-profile-section"><h3>Under Stress</h3><p>${escapeHtml(primary.under_stress || "")}</p></div>
        <div class="ch3-profile-section"><h3>In Socio-</h3><ul>${ch3RenderList(primary.play_suggestions)}</ul></div>
      </div>

      ${ch3RenderPrediction(primary.key)}

      ${ch3RenderBars(allResults, { title: "Full Archetype Breakdown", description: "All fourteen archetype signals from this result." })}

      <div class="ch3-select">
        <label for="ch3-archetype-select">Confirm your archetype</label>
        <select id="ch3-archetype-select">${ch3ArchetypeOptionsHtml(primary.key)}</select>
      </div>
      <div class="catharsis-onboarding__actions">
        <button id="ch3-take-quiz-again" type="button">Take the Quiz Again</button>
        <button id="ch3-confirm-from-quiz" type="button">Confirm and Continue</button>
      </div>
    `;

    state.chapter3.selectedKey = primary.key;
    document.getElementById("ch3-archetype-select").addEventListener("change", (event) => {
      state.chapter3.selectedKey = event.target.value;
    });
    document.getElementById("ch3-take-quiz-again").addEventListener("click", () => {
      state.chapter3.mode = "intro";
      renderChapter3Content();
    });
    document.getElementById("ch3-confirm-from-quiz").addEventListener("click", () => {
      state.chapter3.pendingConfirmKey = state.chapter3.selectedKey;
      state.chapter3.confirmSource = "quiz";
      state.chapter3.mode = "confirm";
      renderChapter3Content();
    });
  };

  const ch3RenderConfirm = () => {
    const archetype = ch3ArchetypeByKey(state.chapter3.pendingConfirmKey);
    chapter3Content.innerHTML = `
      <p class="eyebrow">Chapter III</p>
      <h2>Select ${escapeHtml(archetype?.title || "")} as this character's archetype?</h2>
      ${archetype ? `<div class="ch3-echo-card"><p>${escapeHtml(archetype.echo)}</p></div>` : ""}
      <p id="ch3-status" class="catharsis-onboarding__status" aria-live="polite"></p>
      <div class="catharsis-onboarding__actions">
        <button id="ch3-confirm-back" type="button">Back</button>
        <button id="ch3-confirm-final" type="button">Confirm and Continue</button>
      </div>
    `;

    document.getElementById("ch3-confirm-back").addEventListener("click", () => {
      state.chapter3.mode = state.chapter3.confirmSource === "quiz" ? "quiz-result" : "intro";
      renderChapter3Content();
    });
    document.getElementById("ch3-confirm-final").addEventListener("click", () => {
      void ch3SubmitConfirmation();
    });
  };

  const ch3SubmitConfirmation = async () => {
    const button = document.getElementById("ch3-confirm-final");
    const status = document.getElementById("ch3-status");
    if (button) button.disabled = true;
    if (status) status.textContent = "Saving...";
    try {
      const response = await fetch("/api/character-cards/chapter3-confirm", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          character_card_id: state.workbookCardId,
          archetype_key: state.chapter3.pendingConfirmKey,
        }),
      });
      const payload = await response.json().catch(() => null);
      if (!response.ok || !payload?.ok || !payload?.data) {
        throw new Error(payload?.data?.error || payload?.error || `Archetype confirmation failed (${response.status}).`);
      }
      await refreshWorkbookContextFromServer();
      showChapter4Panel();
    } catch (error) {
      console.error("chapter3 archetype confirm failed", error);
      if (status) status.textContent = String(error.message || "Could not save the archetype. Try again.");
      if (button) button.disabled = false;
    }
  };

  const renderChapter3Content = () => {
    if (state.chapter3.mode === "quiz-question") {
      ch3RenderQuizQuestion();
    } else if (state.chapter3.mode === "quiz-result") {
      ch3RenderQuizResult();
    } else if (state.chapter3.mode === "confirm") {
      ch3RenderConfirm();
    } else {
      ch3RenderIntro();
    }
  };

  const showChapter3ArchetypePanel = async () => {
    hideAllOnboardingPanels();
    chapter3ArchetypePanel.hidden = false;
    overlay.hidden = false;
    document.body.classList.add("modal-open");
    chapter3Content.innerHTML = "<p>Loading archetypes...</p>";

    try {
      await loadChapter3Catalog();
    } catch (error) {
      console.error("chapter3 catalog load failed", error);
      chapter3Content.innerHTML = "<p>Could not load Chapter III archetypes. Try again.</p>";
      return;
    }

    state.chapter3.mode = "intro";
    state.chapter3.selectedKey = "";
    renderChapter3Content();
  };

  // --- Inheritance coin flip ---

  const goToChapterTwoFromBuildPanel = async () => {
    hideAllOnboardingPanels();
    buildPanel.hidden = false;
    overlay.hidden = false;
    document.body.classList.add("modal-open");
    const card = state.activeCharacter;
    const lastParent = state.parentageRows[state.parentageRows.length - 1] || {};
    buildTitle.textContent = String(card?.name || lastParent.social_class || "Socio Candidate").trim();
    buildCopy.textContent = state.startingWealth > 0
      ? `Catharsis finished the two-pass parentage. Starting wealth resolved to ${state.startingWealth}.`
      : "Catharsis finished the two-pass parentage. No wealth inheritance passed the gate.";
    buildClass.textContent = String(lastParent.social_class || card?.name || "Unknown").trim();
    buildCredit.textContent = String(lastParent.starting_credit || "--");
    setBuildStatus(`Ready: ${String(card?.name || lastParent.social_class || "Socio Candidate").trim()}`);
    await sleep(620);
    showChapterPanel();
    setBuildStatus("Chapter II is ready.");
  };

  const showCoinFlipPanel = () => {
    hideAllOnboardingPanels();
    coinFlipPanel.hidden = false;
    overlay.hidden = false;
    document.body.classList.add("modal-open");

    const parentIndex = state.coinFlipQueue[0];
    const row = state.parentageRows.find((parent) => Number(parent.parent_index) === Number(parentIndex)) || {};
    coinFlipTitle.textContent = `Parent ${parentIndex} may pass down wealth`;
    coinFlipCopy.textContent = `Starting credit ${row.starting_credit ?? "--"} (${row.social_class || "Unknown"}) qualifies for retention. Flip the coin to see if it passes to you.`;
    coinFlipFace.textContent = "?";
    coinFlipButton.disabled = false;
    coinFlipButton.textContent = "Flip the coin";
    coinFlipStatus.textContent = "";
    coinFlipContinueButton.disabled = true;
    coinFlipButton.focus();
  };

  const runCoinFlip = async () => {
    const parentIndex = state.coinFlipQueue[0];
    if (parentIndex == null) return;
    coinFlipButton.disabled = true;
    coinFlipStatus.textContent = "Flipping...";
    const token = ++buildSequenceToken;

    const startedAt = Date.now();
    while (Date.now() - startedAt < 900) {
      if (token !== buildSequenceToken) return;
      coinFlipFace.textContent = Math.random() < 0.5 ? "Heads" : "Tails";
      await sleep(90);
    }
    if (token !== buildSequenceToken) return;

    let result;
    try {
      const response = await fetch("/api/character-cards/coin-flip", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ character_card_id: state.workbookCardId, parent_index: parentIndex }),
      });
      const payload = await response.json().catch(() => null);
      if (!response.ok || !payload?.ok || !payload?.data) {
        throw new Error(payload?.data?.error || payload?.error || `Coin flip failed (${response.status}).`);
      }
      result = payload.data;
    } catch (error) {
      console.error("catharsis coin flip failed", error);
      if (token !== buildSequenceToken) return;
      coinFlipStatus.textContent = "Could not flip the coin. Try again.";
      coinFlipButton.disabled = false;
      return;
    }
    if (token !== buildSequenceToken) return;

    state.coinFlipResults[parentIndex] = result;
    const row = state.parentageRows.find((parent) => Number(parent.parent_index) === Number(parentIndex));
    if (row) {
      row.coin_flip_roll = result.coin_flip_roll;
      row.coin_flip_result = result.coin_flip_result;
      row.inheritance_passed = result.inheritance_passed;
      row.inherited_wealth = result.inherited_wealth;
    }
    state.startingWealth = Number(result.starting_wealth) || 0;

    const won = result.coin_flip_result === "retain";
    coinFlipFace.textContent = won ? "Retained!" : "Lost.";
    coinFlipStatus.textContent = won
      ? `Parent ${parentIndex}'s wealth passes down: +${result.inherited_wealth} starting credit.`
      : `Parent ${parentIndex}'s wealth does not pass down.`;
    coinFlipContinueButton.disabled = false;
    coinFlipContinueButton.focus();
  };

  const advanceCoinFlipQueue = async () => {
    const finishedParentIndex = state.coinFlipQueue.shift();
    const result = state.coinFlipResults[finishedParentIndex];

    if (result && state.workbookCardId) {
      try {
        await postWorkbookEvents(state.workbookCardId, {
          character_card_id: state.workbookCardId,
          module_key: "socio",
          module_status: "draft",
          workbook_status: "draft",
          entries: [
            {
              page_key: "history",
              entry_type: "coin_flip",
              title: `Parent ${finishedParentIndex} coin flip`,
              body: `Parent ${finishedParentIndex} retention resolved to ${result.coin_flip_result} and ${result.inherited_wealth} starting wealth.`,
              stage_number: 1,
              sort_order: 1,
              payload: {
                ...result,
                source: "catharsis",
                ruleset_key: "socio",
                ruleset_version: "1.1",
              },
            },
          ],
        });
      } catch (error) {
        console.error("catharsis coin flip history write failed", error);
      }
    }

    if (state.coinFlipQueue.length > 0) {
      showCoinFlipPanel();
      return;
    }

    await goToChapterTwoFromBuildPanel();
  };

  coinFlipButton.addEventListener("click", () => {
    void runCoinFlip();
  });

  coinFlipContinueButton.addEventListener("click", () => {
    void advanceCoinFlipQueue();
  });

  ch2RollD4Button.addEventListener("click", () => {
    void ch2RollD4();
  });

  ch2RollD6Button.addEventListener("click", () => {
    void ch2RollD6();
  });

  ch2Representation.addEventListener("input", () => {
    ch2Local.representationText = ch2Representation.value;
  });

  ch2CompanionToggle.addEventListener("change", () => {
    ch2Local.companionPurchased = ch2CompanionToggle.checked;
    ch2CompanionName.hidden = !ch2Local.companionPurchased;
    ch2RenderCompleteGate();
  });

  ch2CompanionName.addEventListener("input", () => {
    ch2Local.companionTarget = ch2CompanionName.value;
  });

  ch2CompleteButton.addEventListener("click", () => {
    void ch2CompleteStage();
  });

  chapter4ContinueButton.addEventListener("click", () => {
    closeOverlay();
  });

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

  const refreshWorkbookContextFromServer = async () => {
    const response = await fetch("/api/character-cards/me", { credentials: "include" });
    const payload = await response.json().catch(() => null);
    const data = payload?.data || {};
    const activeCharacterId = String(state.activeCharacter?.character_card_id || state.workbookCardId || "");
    const cards = Array.isArray(data.cards) ? data.cards : [];
    const refreshed = data.active_character?.character_card_id === activeCharacterId
      ? data.active_character
      : cards.find((card) => String(card.id || card.character_card_id || "") === activeCharacterId);
    if (refreshed) {
      state.workbookContext = { ...(refreshed.workbook_context || {}) };
      state.activeCharacter = { ...state.activeCharacter, ...refreshed };
    }
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
    hideAllOnboardingPanels();
    introPanel.hidden = false;
    overlay.hidden = false;
    resetRollState(false);
    setSocioStatus("");
    document.body.classList.add("modal-open");
  };

  const showSocioPrompt = () => {
    hideAllOnboardingPanels();
    socioPanel.hidden = false;
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
    hideAllOnboardingPanels();
    buildPanel.hidden = false;
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

      // Coin-flip history entries are written later, once each flip is
      // actually resolved by the player (see runCoinFlip/advanceCoinFlipQueue)
      // rather than here, since the result is not known yet at this point.
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
        {
          page_key: "history",
          entry_type: "parentage_roll",
          title: "Parent 2 roll",
          body: summarizeParentRows([secondParentRow]),
          stage_number: 1,
          sort_order: 2,
          payload: {
            ...secondParentRow,
            source: "catharsis",
            ruleset_key: "socio",
            ruleset_version: "1.1",
          },
        },
      ];

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
      buildRoll.textContent = String(total);
      buildClass.textContent = String(secondParentRow.social_class || card?.name || "Unknown").trim();
      buildCredit.textContent = String(secondParentRow.starting_credit || "--");
      renderWheelWindow(total);
      localStorage.setItem(introKey, "1");
      localStorage.setItem(socioKey, "1");

      state.coinFlipQueue = serverParentRows
        .filter((parent) => Boolean(parent.coin_flip_eligible))
        .map((parent) => Number(parent.parent_index))
        .filter((index) => Number.isFinite(index));
      state.coinFlipResults = {};

      if (state.coinFlipQueue.length > 0) {
        buildCopy.textContent = "Catharsis finished the two-pass parentage. At least one parent qualifies for inheritance retention — flip the coin to find out.";
        setBuildStatus("Inheritance coin flip is ready.");
        await sleep(620);
        if (sequenceToken !== buildSequenceToken) return;
        showCoinFlipPanel();
        return;
      }

      buildCopy.textContent = "Catharsis finished the two-pass parentage. No wealth inheritance passed the gate.";
      setBuildStatus(`Ready: ${String(card?.name || secondParentRow.social_class || "Socio Candidate").trim()}`);
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
      await showChapter2Panel(1);
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
        current_event: "chapter2_stage_1",
        workbook_status: "draft",
        workbook_context: {
          ...getCurrentWorkbookContext(),
          current_stage: 2,
          current_event: "chapter2_stage_1",
          active_page: "face",
        },
        entries: [
          {
            page_key: "creation_progress",
            entry_type: "chapter_handoff",
            title: "Chapter II: Lifepath",
            body: "Catharsis opened the next chapter and handed the workbook into the Lifepath stages.",
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
      await showChapter2Panel(1);
    } catch (error) {
      if (sequenceToken !== buildSequenceToken) return;
      console.error("catharsis chapter handoff failed", error);
      chapterContinueButton.disabled = false;
      setBuildStatus("Could not save Chapter II handoff.");
    }
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

      // An in-progress Chapter 2 draft always takes priority over the
      // "first time in this browser" intro gate below — otherwise a fresh
      // browser/profile/incognito session (no localStorage flags yet) would
      // drop a resuming player back into Stage 1 character creation instead
      // of their already-existing draft.
      if (!requestedNewCharacter && state.workbookCardId && Number(state.workbookContext?.current_stage) === 2) {
        const ctx = chapter2Ctx();
        const resumeStage = Math.min(10, Math.max(1, Number(ctx?.current_stage) || 1));
        await showChapter2Panel(resumeStage);
        return;
      }

      // Chapter 3 is never restored mid-quiz (quiz state is intentionally
      // client-only and not canonical) — an unconfirmed Chapter 3 always
      // resumes at the title screen. A confirmed Chapter 3 has already
      // advanced current_stage to 4, so it falls through to the normal
      // "no onboarding needed" path below.
      if (!requestedNewCharacter && state.workbookCardId && Number(state.workbookContext?.current_stage) === 3) {
        await showChapter3ArchetypePanel();
        return;
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
