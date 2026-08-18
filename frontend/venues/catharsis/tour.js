// Kernel 91: the Cast venue tour for Catharsis. Kept separate from
// onboarding.js on purpose -- onboarding.js is the Kernel 74/75
// character-creation wizard, a different concern from orienting a Cast
// member to the venue's live controls. This file only registers targets and
// autostarts; VictoryTourEngine (a resolved role check) makes the actual
// eligibility decision server-side.
(function () {
  function wizardOpen() {
    const overlay = document.getElementById("catharsis-onboarding");
    return Boolean(overlay && overlay.hidden === false);
  }

  function registerTargets() {
    if (!window.VictoryTourEngine) return;
    window.VictoryTourEngine.registerTargets({
      "control:catharsis-character-tray": () => document.getElementById("character-tray-button"),
      "control:catharsis-socio-hud": () => document.getElementById("kernel88-player-hud"),
      "control:catharsis-dice-tray": () => document.getElementById("catharsis-dice-tray"),
    });
  }

  async function maybeStart() {
    if (!window.VictoryTourEngine || wizardOpen()) return;
    try {
      const response = await fetch("/api/session/me", { credentials: "include" });
      const payload = await response.json().catch(() => null);
      if (!payload || !payload.signed_in) return;
    } catch (error) {
      return;
    }
    registerTargets();

    const replayKey = new URLSearchParams(window.location.search).get("replay-tour");
    if (replayKey === "catharsis_cast") {
      window.VictoryTourEngine.start(replayKey, { replay: true });
      return;
    }
    window.VictoryTourEngine.autostart({ venueSlug: "catharsis" });
  }

  // The character-creation wizard may open shortly after load for a
  // brand-new character; give it a beat to claim the screen first rather
  // than racing it.
  window.setTimeout(maybeStart, 400);
})();
