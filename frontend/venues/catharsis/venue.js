// Catharsis venue config for the shared stage engine (Kernel 72).
// The engine (/lib/stage-runtime/) owns all generic stage behavior; anything
// Socio/Catharsis-specific belongs here or in onboarding.js — never in the
// engine. Future games follow this same pattern: one venue.js + game layer.
window.VictoryStageVenue = {
  slug: "catharsis",
  name: "Catharsis",
  // The character tray invites a brand-new workbook here (onboarding flow),
  // instead of the template's plain "Open the workbook".
  workbookOpenLabel: "Begin a new workbook",
  // Card editor button restarts the in-venue new-character flow rather than
  // navigating to the Greenroom.
  onCardEditor: () => {
    const url = new URL(window.location.href);
    url.searchParams.set("new_character", "1");
    window.location.href = url.toString();
  },
  // Right-tray help button re-enters guided onboarding (loaded after the
  // engine, hence the lazy lookup).
  onHelp: () => {
    window.VictoryCatharsisOnboarding?.begin?.();
  },
};
