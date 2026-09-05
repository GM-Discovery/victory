const infoBoothState = {
  initialized: false,
  modal: null,
  closeButton: null,
};

const mapMenuState = {
  initialized: false,
  toggle: null,
  panel: null,
  closeButton: null,
  outline: null,
};

const soilExpertsState = {
  initialized: false,
  modal: null,
  closeButton: null,
};

const mapState = {
  venues: [],
  pinsReady: false,
};

const hiddenMainMapVenueSlugs = new Set([
  "gateway-thread",
  "gateway-thread-fixture",
  "gateway-thread-venue",
]);

function isHiddenMainMapVenueSlug(slug) {
  return hiddenMainMapVenueSlugs.has(String(slug || "").trim().toLowerCase());
}

function initInfoBoothModal() {
  if (infoBoothState.initialized) return;

  infoBoothState.modal = document.getElementById("info-booth-modal");
  infoBoothState.closeButton = document.getElementById("info-booth-close");

  const closeInfoBooth = () => {
    if (!infoBoothState.modal) return;
    infoBoothState.modal.hidden = true;
    document.body.classList.remove("modal-open");
  };

  const openInfoBooth = () => {
    if (!infoBoothState.modal) return;
    infoBoothState.modal.hidden = false;
    document.body.classList.add("modal-open");
  };

  if (infoBoothState.closeButton) {
    infoBoothState.closeButton.addEventListener("click", closeInfoBooth);
  }

  if (infoBoothState.modal) {
    infoBoothState.modal.addEventListener("click", (event) => {
      if (event.target === infoBoothState.modal) {
        closeInfoBooth();
      }
    });
  }

  infoBoothState.initialized = true;
  infoBoothState.open = openInfoBooth;
  infoBoothState.close = closeInfoBooth;
}

function initMapMenu() {
  if (mapMenuState.initialized) return;

  mapMenuState.toggle = document.getElementById("map-menu-toggle");
  mapMenuState.panel = document.getElementById("map-menu");
  mapMenuState.closeButton = document.getElementById("map-menu-close");
  mapMenuState.outline = document.getElementById("map-menu-outline");

  const closeMapMenu = () => {
    if (!mapMenuState.panel || !mapMenuState.toggle) return;
    mapMenuState.panel.hidden = true;
    mapMenuState.toggle.setAttribute("aria-expanded", "false");
  };

  const openMapMenu = () => {
    if (!mapMenuState.panel || !mapMenuState.toggle) return;
    mapMenuState.panel.hidden = false;
    mapMenuState.toggle.setAttribute("aria-expanded", "true");
  };

  if (mapMenuState.toggle) {
    mapMenuState.toggle.addEventListener("click", (event) => {
      event.preventDefault();
      if (mapMenuState.panel?.hidden) {
        openMapMenu();
      } else {
        closeMapMenu();
      }
    });
  }

  if (mapMenuState.closeButton) {
    mapMenuState.closeButton.addEventListener("click", closeMapMenu);
  }

  if (mapMenuState.panel) {
    mapMenuState.panel.addEventListener("click", (event) => {
      if (event.target === mapMenuState.panel) {
        closeMapMenu();
      }
    });
  }

  mapMenuState.initialized = true;
  mapMenuState.open = openMapMenu;
  mapMenuState.close = closeMapMenu;
}

function initSoilExpertsModal() {
  if (soilExpertsState.initialized) return;

  soilExpertsState.modal = document.getElementById("soil-experts-modal");
  soilExpertsState.closeButton = document.getElementById("soil-experts-close");

  const closeSoilExperts = () => {
    if (!soilExpertsState.modal) return;
    soilExpertsState.modal.hidden = true;
    document.body.classList.remove("modal-open");
  };

  const openSoilExperts = () => {
    if (!soilExpertsState.modal) return;
    soilExpertsState.modal.hidden = false;
    document.body.classList.add("modal-open");
  };

  if (soilExpertsState.closeButton) {
    soilExpertsState.closeButton.addEventListener("click", closeSoilExperts);
  }

  if (soilExpertsState.modal) {
    soilExpertsState.modal.addEventListener("click", (event) => {
      if (event.target === soilExpertsState.modal) {
        closeSoilExperts();
      }
    });
  }

  soilExpertsState.initialized = true;
  soilExpertsState.open = openSoilExperts;
  soilExpertsState.close = closeSoilExperts;
}

// User-facing label overrides (Kernel 68 §3.5): internal slug/routes stay
// "show-runs" to avoid churn across the API and existing links -- only the
// map tile/menu label changes to "Stage Management".
const venueDisplayNameOverrides = {
  "show-runs": "Stage Management",
};

function venueDisplayName(venue) {
  if (!venue) return "Unknown Venue";
  return venueDisplayNameOverrides[venue.slug] || venue.name || venue.slug || "Unknown Venue";
}

// Kernel 95 Pass 6: this was called (loadAccountLink, below) but never
// defined -- a ReferenceError thrown on every single signed-in campus page
// load, silently caught by loadAccountLink's try/catch and papered over by
// resetting the whole account UI back to its signed-out state. Nobody
// visiting the campus map while signed in has ever seen their real name,
// a working account menu, or (as of this kernel) the Mailbox pip -- same
// "generic account name falls back to handle" convention already used in
// stage-runtime/runtime.js's updateShellMetaPresentation, for consistency.
function identityDisplayName(user) {
  const displayName = String(user?.display_name || "").trim();
  const handle = String(user?.handle || "").trim();
  const genericDisplayNames = new Set(["web user", "webuser", "browser user", "browser", "account", "user"]);
  if (displayName && !genericDisplayNames.has(displayName.toLowerCase())) return displayName;
  return handle || "Account";
}

// Kernel 95 Pass 6: roleLabel was also called (loadAccountLink, below) but
// never defined -- the same copy-the-call-without-the-helper gap as
// identityDisplayName above, and the same normalizeRole/roleLabel pair
// already used in stage-runtime/runtime.js, ported here for the exact same
// role vocabulary.
function normalizeRole(role) {
  const value = String(role || "audience").trim().toLowerCase();
  if (["producer", "director", "operator", "cast", "crew", "audience", "actor"].includes(value)) {
    return value;
  }
  return "audience";
}

function roleLabel(role) {
  switch (normalizeRole(role)) {
    case "producer":
      return "Producer";
    case "director":
      return "Director";
    case "operator":
      return "Operator";
    case "cast":
    case "actor":
      return "Cast";
    case "crew":
      return "Crew";
    default:
      return "Audience";
  }
}

function venueHref(slug) {
  switch (slug) {
    case "library":
      return "/venues/library/";
    case "writers-room":
      return "/venues/writers-room/";
    case "first-theater":
      return "/venues/first-theater/";
    case "middle-school-stage":
      return "/venues/middle-school-stage/";
    case "warehouse":
      return "/venues/warehouse/";
    case "the-cave":
      return "/venues/the-cave/";
    case "audition-hall":
      return "/venues/audition-hall/?venue=catharsis&role=cast";
    case "victory-theater":
      return "/venues/victory-theater/";
    case "construction":
      return "/venues/construction/";
    case "workshop":
      return "/venues/workshop/";
    case "greenroom":
      return "/venues/greenroom/";
    case "trailers":
      return "/venues/trailers/";
    case "third-place":
      return "/venues/third-place/";
    case "show-runs":
      return "/venues/show-runs/";
    case "producers-office":
      return "/venues/producers-office/";
    case "directors-chair":
      return "/venues/directors-chair/";
    case "grants-cabin":
      return "/venues/grants-cabin/";
    case "catharsis":
      return "/venues/catharsis/";
    case "storyboards":
      return "/venues/storyboards/";
    default:
      return "";
  }
}

function openVisibleVenue(venue) {
  if (!venue) return;
  if (venue.slug === "info-booth") {
    initInfoBoothModal();
    if (typeof infoBoothState.open === "function") {
      infoBoothState.open();
    }
    return;
  }
  if (venue.slug === "soil-experts") {
    initSoilExpertsModal();
    if (typeof soilExpertsState.open === "function") {
      soilExpertsState.open();
    }
    return;
  }
  const href = venueHref(venue.slug);
  if (href) {
    window.location.href = href;
    return;
  }
  alert(`Venue "${venueDisplayName(venue)}" is not built yet.`);
}

function createVenuePin(venue, assetURL, fallbackPositions) {
  const pin = document.createElement("button");
  pin.className = "venue-pin";

  if (venue.slug === "victory-theater") {
    pin.classList.add("venue-pin--theater");
  }
  if (venue.slug === "third-place") {
    pin.classList.add("venue-pin--third-place");
  }
  if (venue.slug === "audition-hall") {
    pin.classList.add("venue-pin--audition-hall");
  }
  if (venue.slug === "info-booth") {
    pin.classList.add("venue-pin--info-booth");
  }
  if (venue.slug === "first-theater") {
    pin.classList.add("venue-pin--first-theater");
  }
  if (venue.slug === "trailers") {
    // Kernel 95 Pass 6: Mailbox has no map icon of its own -- this pin is
    // the closest thing to a discoverable "you have new messages" surface
    // on the map itself, even though Mailbox is technically a separate
    // feature from the Trailers profile/showcase venue this pin actually
    // opens. Pip state resolved after all pins are in the DOM (see the
    // refreshPips() call in loadVenues below).
    pin.dataset.mailboxPipTarget = "";
  }
  pin.type = "button";
  pin.dataset.venueSlug = venue.slug;
  pin.setAttribute("aria-label", venueDisplayName(venue));

  const fallback = fallbackPositions[venue.slug] || { x: 50, y: 50 };
  const x = typeof venue.map_x === "number" ? venue.map_x : fallback.x;
  const y = typeof venue.map_y === "number" ? venue.map_y : fallback.y;

  pin.style.left = `${x}%`;
  pin.style.top = `${y}%`;

  const icon = document.createElement("img");
  const venueIcons = {
    library: "/assets/librarycc.png",
    "writers-room": "/assets/writers-room.png",
    "first-theater": "/assets/firsttheater.png",
    "middle-school-stage": "/assets/default.png",
    "soil-experts": "/assets/mudfarm.png",
    "info-booth": "/assets/infobooth.png",
    "audition-hall": "/assets/audition-hall.png",
    "the-cave": "/assets/cave.png",
    workshop: "/assets/workshop.png",
    "victory-theater": "/assets/victorytheater.png",
    construction: "/assets/construction.png",
    greenroom: "/assets/Greenroom.png",
    trailers: "/assets/trailers.png",
    "third-place": "/assets/third-place.png",
    "show-runs": "/assets/show-runs.png",
    warehouse: "/assets/warehouse.png",
    "producers-office": "/assets/producersoffice.png",
    "directors-chair": "/assets/directorschair.png",
    "grants-cabin": "/assets/grantsoffice.png",
    catharsis: "/assets/catharsis.png",
    storyboards: "/assets/storyboard.png",
  };

  icon.src = assetURL(venue.icon_url || venueIcons[venue.slug] || "/assets/default.png");
  icon.alt = "";

  const label = document.createElement("span");
  label.className = "venue-label";
  label.textContent = venueDisplayName(venue);

  if (Number(venue.notification_count || 0) > 0) {
    const badge = document.createElement("span");
    badge.className = "venue-notification";
    badge.textContent = String(venue.notification_count);
    pin.appendChild(badge);
  }

  pin.appendChild(icon);
  pin.appendChild(label);
  pin.addEventListener("click", () => openVisibleVenue(venue));
  return pin;
}

function renderMapMenu() {
  initMapMenu();
  const outlineEl = mapMenuState.outline;
  if (!outlineEl) return;

  const signedIn = Boolean(document.querySelector(".app-shell")?.classList.contains("app-shell--signed-in"));
  const venues = Array.isArray(mapState.venues) ? mapState.venues.filter((venue) => !isHiddenMainMapVenueSlug(venue?.slug)) : [];
  const sorted = venues.sort((a, b) => {
    const aLabel = String(a.visible_because || "");
    const bLabel = String(b.visible_because || "");
    return aLabel.localeCompare(bLabel) || String(a.name || a.slug || "").localeCompare(String(b.name || b.slug || ""));
  });

  const groupLabels = {
    public: "Public front door",
    authenticated_surface: "First-time access",
    producer_surface: "Producer surface",
    director_surface: "Director surface",
    performer_surface: "Performer surface",
    approved_performer_surface: "Approved performer access",
    venue_membership: "Venue membership",
    grant: "Granted access",
    location_role: "Location role",
    production_role: "Production role",
    delayed_lot_surface: "Delayed surface",
    backstage_authority_surface: "Backstage authority",
    show_run_crew_surface: "Show Run crew",
    trailer_face_ready: "Trailer Face ready",
    ewrite_author_surface: "Writer's Room access",
  };

  const groups = new Map();
  for (const venue of sorted) {
    const key = venue.visible_because || "public";
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key).push(venue);
  }

  outlineEl.innerHTML = "";

  const actions = document.createElement("div");
  actions.className = "map-menu-actions";
  actions.innerHTML = "";
  if (signedIn) {
    const account = document.createElement("a");
    account.href = "/account/";
    account.textContent = "Profile";
    actions.appendChild(account);
  } else {
    const register = document.createElement("a");
    register.href = "/signup/";
    register.textContent = "REGISTER";
    actions.appendChild(register);
    const login = document.createElement("a");
    login.href = "/login/";
    login.textContent = "Log In";
    actions.appendChild(login);
  }
  outlineEl.appendChild(actions);

  const frontDoor = document.createElement("section");
  frontDoor.className = "map-menu-group";
  const frontDoorHeading = document.createElement("h3");
  frontDoorHeading.textContent = "Public front door";
  frontDoor.appendChild(frontDoorHeading);
  const frontDoorList = document.createElement("ul");
  const infoDoorItem = document.createElement("li");
  const infoButton = document.createElement("button");
  infoButton.type = "button";
  infoButton.textContent = "Info Booth";
  infoButton.addEventListener("click", () => {
    mapMenuState.close?.();
    openVisibleVenue({ slug: "info-booth", name: "Info Booth" });
  });
  infoDoorItem.appendChild(infoButton);
  const infoMeta = document.createElement("small");
  infoMeta.textContent = "public front door";
  infoDoorItem.appendChild(infoMeta);
  frontDoorList.appendChild(infoDoorItem);
  frontDoor.appendChild(frontDoorList);
  outlineEl.appendChild(frontDoor);

  if (!sorted.length) {
    const empty = document.createElement("div");
    empty.className = "map-menu-group";
    empty.textContent = "No visible venues yet.";
    outlineEl.appendChild(empty);
    return;
  }

  for (const [groupKey, venuesInGroup] of groups.entries()) {
    const group = document.createElement("section");
    group.className = "map-menu-group";

    const heading = document.createElement("h3");
    heading.textContent = groupLabels[groupKey] || groupKey.replaceAll("_", " ");
    group.appendChild(heading);

    const list = document.createElement("ul");
    for (const venue of venuesInGroup) {
      const item = document.createElement("li");
      const target = venue.slug === "soil-experts" ? "#" : venueHref(venue.slug);

      if (target && target !== "#") {
        const link = document.createElement("a");
        link.href = target;
        link.textContent = venueDisplayName(venue);
        link.addEventListener("click", () => mapMenuState.close?.());
        item.appendChild(link);
      } else {
        const button = document.createElement("button");
        button.type = "button";
        button.textContent = venueDisplayName(venue);
        button.addEventListener("click", () => {
          mapMenuState.close?.();
          openVisibleVenue(venue);
        });
        item.appendChild(button);
      }

      const meta = document.createElement("small");
      meta.textContent = venue.visible_because ? venue.visible_because.replaceAll("_", " ") : "";
      item.appendChild(meta);
      list.appendChild(item);
    }
    group.appendChild(list);
    outlineEl.appendChild(group);
  }
}

// Kernel 91: registers the campus map's tour targets (venue pins) and
// starts the mandatory/continuation campus tour. Replaces the old
// localStorage-only "Welcome to Victory" modal (S12: server-persisted
// completion, not browser storage). Waits for both venue pins to be
// rendered (loadVenues) and account state to resolve as signed-in
// (loadAccountLink) before autostarting, since a mandatory-tour target like
// venue:audition-hall must already exist in the DOM to be spotlighted.
function registerCampusTourTargets() {
  if (!window.VictoryTourEngine) return;
  window.VictoryTourEngine.registerTargets({
    "venue:audition-hall": () => document.querySelector('[data-venue-slug="audition-hall"]'),
    "venue:trailers": () => document.querySelector('[data-venue-slug="trailers"]'),
    "venue:catharsis": () => document.querySelector('[data-venue-slug="catharsis"]'),
    "venue:greenroom": () => document.querySelector('[data-venue-slug="greenroom"]'),
  });
}

function maybeStartCampusTour() {
  if (!window.VictoryTourEngine || !mapState.pinsReady) return;
  registerCampusTourTargets();

  const replayKey = new URLSearchParams(window.location.search).get("replay-tour");
  if (replayKey === "campus_mandatory" || replayKey === "campus_continuation" || replayKey === "greenroom_intro") {
    window.VictoryTourEngine.start(replayKey, { replay: true });
    return;
  }
  window.VictoryTourEngine.autostart({});
}

// Map positions are known even when a venue is hidden (fallbackPositions is
// fixed per slug), so a cloud puff can sit exactly over where the tile will
// appear once unlocked. Every venue gets one (info-booth excluded -- it's a
// permanent landmark, not a gateable venue). The base fog layer separately
// recedes with overall visible-venue count.
// Ambient puffs are purely decorative atmosphere, independent of venue
// visibility -- they never recede (that job belongs to the per-venue puffs
// in renderMapFog, which fully disappear once a venue unlocks). Fixed
// pseudo-random layout (seeded, not Math.random) so the sky doesn't reshuffle
// on every reload. Deliberately edge-weighted: the map's middle is already
// covered by venue puffs, so ambient atmosphere piles up along the border and
// thins out fast toward the center to avoid double-fogging the middle.
const AMBIENT_FOG_SEED_LAYOUT = [
  // Top edge
  { x: 2, y: 2, scale: 1.1, drift: "med" }, { x: 10, y: 4, scale: 0.9, drift: "slow" },
  { x: 20, y: 3, scale: 1.3, drift: "fast" }, { x: 30, y: 5, scale: 0.8, drift: "med" },
  { x: 40, y: 2, scale: 1.15, drift: "slow" }, { x: 50, y: 4, scale: 0.95, drift: "fast" },
  { x: 60, y: 3, scale: 1.2, drift: "med" }, { x: 70, y: 5, scale: 0.85, drift: "slow" },
  { x: 80, y: 2, scale: 1.1, drift: "fast" }, { x: 90, y: 4, scale: 1.3, drift: "med" },
  { x: 98, y: 2, scale: 0.9, drift: "slow" },
  // Bottom edge
  { x: 4, y: 96, scale: 1.0, drift: "fast" }, { x: 14, y: 92, scale: 1.2, drift: "med" },
  { x: 24, y: 95, scale: 0.85, drift: "slow" }, { x: 34, y: 91, scale: 1.15, drift: "fast" },
  { x: 44, y: 96, scale: 0.9, drift: "med" }, { x: 55, y: 92, scale: 1.25, drift: "slow" },
  { x: 65, y: 95, scale: 0.8, drift: "fast" }, { x: 75, y: 91, scale: 1.1, drift: "med" },
  { x: 85, y: 96, scale: 0.95, drift: "slow" }, { x: 96, y: 92, scale: 1.2, drift: "fast" },
  // Left edge -- lighter around (4, 65) where Soil Experts sits, so its icon
  // stays readable without going fully clear like the Info Booth.
  { x: 2, y: 15, scale: 1.1, drift: "slow" }, { x: 4, y: 28, scale: 0.9, drift: "med" },
  { x: 2, y: 40, scale: 1.2, drift: "fast" }, { x: 8, y: 55, scale: 0.55, drift: "slow" },
  { x: 9, y: 72, scale: 0.5, drift: "med" }, { x: 2, y: 88, scale: 0.9, drift: "slow" },
  // Right edge
  { x: 97, y: 15, scale: 1.05, drift: "med" }, { x: 95, y: 28, scale: 0.85, drift: "fast" },
  { x: 98, y: 40, scale: 1.2, drift: "slow" }, { x: 94, y: 52, scale: 0.95, drift: "med" },
  { x: 97, y: 64, scale: 1.1, drift: "fast" }, { x: 95, y: 76, scale: 0.8, drift: "slow" },
  { x: 98, y: 88, scale: 1.15, drift: "med" },
  // Sparse inner ring -- just enough to avoid a hard edge, nothing near the middle.
  { x: 16, y: 18, scale: 0.7, drift: "slow" }, { x: 84, y: 18, scale: 0.75, drift: "fast" },
  { x: 16, y: 82, scale: 0.7, drift: "med" }, { x: 84, y: 82, scale: 0.8, drift: "slow" },
];

// Info Booth sits at (50, 93) and must read clearly whether signed in or out --
// it's the entry point, not a gateable venue. Keep every ambient puff at least
// this far from it (in %, roughly circular) rather than hand-tuning coordinates.
const INFO_BOOTH_CLEAR_ZONE = { x: 50, y: 93, radius: 14 };

function isInInfoBoothClearZone(x, y) {
  const dx = x - INFO_BOOTH_CLEAR_ZONE.x;
  const dy = y - INFO_BOOTH_CLEAR_ZONE.y;
  return Math.sqrt(dx * dx + dy * dy) < INFO_BOOTH_CLEAR_ZONE.radius;
}

// Venue icon positions (mirrors the fallbackPositions built in loadVenues --
// duplicated here deliberately, since fog layout is decorative and shouldn't
// depend on venue-visibility data being loaded yet) used only to keep the new
// top-band/center-woods fog fill from sitting directly on top of an icon.
const VENUE_ICON_CLEAR_ZONES = [
  { x: 22, y: 23, r: 10 }, // the-cave
  { x: 28, y: 21, r: 10 }, // grants-cabin
  { x: 29, y: 37, r: 9 }, // audition-hall
  { x: 38, y: 14, r: 10 }, // producers-office
  { x: 55, y: 23, r: 10 }, // directors-chair
  { x: 52, y: 11, r: 10 }, // show-runs / Stage Management
  { x: 69, y: 13, r: 10 }, // trailers
  { x: 75, y: 33, r: 9 }, // workshop
  { x: 75, y: 27, r: 9 }, // warehouse
  { x: 88, y: 47, r: 9 }, // greenroom
  { x: 50, y: 50, r: 12 }, // third-place (icon is 2x size, give it more room)
  { x: 37, y: 69, r: 9 }, // catharsis
  { x: 29, y: 57, r: 9 }, // construction
  { x: 70, y: 62, r: 9 }, // first-theater
  { x: 70, y: 87, r: 9 }, // library
  { x: 29, y: 62, r: 9 }, // writers-room
  { x: 83, y: 77, r: 10 }, // victory-theater
  { x: 4, y: 65, r: 10 }, // soil-experts
  { x: 15, y: 50, r: 9 }, // storyboards
];

function isNearVenueIcon(x, y) {
  return VENUE_ICON_CLEAR_ZONES.some((zone) => {
    const dx = x - zone.x;
    const dy = y - zone.y;
    return Math.sqrt(dx * dx + dy * dy) < zone.r;
  });
}

// Fills the top band (Trailers/Stage Management/Producer's Office row) and
// the central woods, which were left too clear by the edge-weighted layout
// above. Generated deterministically (not Math.random, same seeded-layout
// requirement as the rest of this file) across a jittered grid, then any
// point landing on a venue icon or the Info Booth is dropped.
const WOODS_FOG_LAYOUT = (() => {
  const points = [];
  const rows = [10, 18, 26, 34, 42, 50, 58, 66];
  let i = 0;
  for (const y of rows) {
    for (let x = 8; x <= 92; x += 10) {
      i += 1;
      const px = x + (((i * 37) % 7) - 3);
      const py = y + (((i * 53) % 7) - 3);
      if (isInInfoBoothClearZone(px, py)) continue;
      if (isNearVenueIcon(px, py)) continue;
      const scale = 0.75 + ((i * 13) % 6) * 0.08;
      const drift = ["slow", "med", "fast"][i % 3];
      points.push({ x: px, y: py, scale, drift });
    }
  }
  return points;
})();

// Anonymous-only extra atmosphere: signed-out visitors see a heavier center,
// which clears the instant they sign in (see .map-layer--signed-in rule in
// styles.css) rather than persisting like the edge/venue fog does.
const ANONYMOUS_CENTER_FOG_LAYOUT = [
  { x: 38, y: 48, scale: 1.1, drift: "slow" }, { x: 58, y: 42, scale: 0.95, drift: "med" },
  { x: 46, y: 58, scale: 1.2, drift: "fast" }, { x: 62, y: 60, scale: 0.9, drift: "slow" },
  { x: 34, y: 62, scale: 1.0, drift: "med" }, { x: 54, y: 34, scale: 1.05, drift: "fast" },
  { x: 24, y: 46, scale: 0.82, drift: "fast" }, { x: 76, y: 50, scale: 0.88, drift: "med" },
  { x: 43, y: 28, scale: 0.76, drift: "slow" }, { x: 68, y: 72, scale: 0.8, drift: "fast" },
];

const WIDE_VENUE_FOG = {
  catharsis: { scale: 1.35, shape: "wide-left" },
  "third-place": { scale: 1.5, shape: "wide-right" },
  "first-theater": { scale: 1.4, shape: "wide-left" },
};

function renderAmbientFog(fogLayer) {
  fogLayer.querySelectorAll(".map-fog-puff--ambient, .map-fog-puff--anon-center").forEach((el) => el.remove());
  for (const spot of AMBIENT_FOG_SEED_LAYOUT) {
    if (isInInfoBoothClearZone(spot.x, spot.y)) continue;
    const puff = document.createElement("div");
    puff.className = `map-fog-puff map-fog-puff--ambient map-fog-puff--drift-${spot.drift}`;
    puff.style.left = `${spot.x}%`;
    puff.style.top = `${spot.y}%`;
    puff.style.setProperty("--puff-scale", String(spot.scale));
    fogLayer.appendChild(puff);
  }
  for (const spot of WOODS_FOG_LAYOUT) {
    const puff = document.createElement("div");
    puff.className = `map-fog-puff map-fog-puff--ambient map-fog-puff--woods map-fog-puff--drift-${spot.drift}`;
    puff.style.left = `${spot.x}%`;
    puff.style.top = `${spot.y}%`;
    puff.style.setProperty("--puff-scale", String(spot.scale));
    fogLayer.appendChild(puff);
  }
  for (const spot of ANONYMOUS_CENTER_FOG_LAYOUT) {
    const puff = document.createElement("div");
    puff.className = `map-fog-puff map-fog-puff--anon-center map-fog-puff--drift-${spot.drift}`;
    puff.style.left = `${spot.x}%`;
    puff.style.top = `${spot.y}%`;
    puff.style.setProperty("--puff-scale", String(spot.scale));
    fogLayer.appendChild(puff);
  }
}

function renderMapFog(venues, fallbackPositions) {
  const fogLayer = document.getElementById("map-fog-layer");
  const fogBase = document.getElementById("map-fog-base");
  if (!fogLayer) return;

  const visibleSlugs = new Set(venues.map((v) => v.slug));

  if (fogBase) {
    const opacity = Math.max(0.08, 0.28 - venues.length * 0.012);
    fogBase.style.opacity = String(opacity);
  }

  fogLayer.querySelectorAll(".map-fog-puff:not(.map-fog-puff--ambient)").forEach((el) => el.remove());

  for (const slug of Object.keys(fallbackPositions)) {
    if (slug === "info-booth") continue;
    const pos = fallbackPositions[slug];
    if (!pos) continue;
    const puff = document.createElement("div");
    puff.className = "map-fog-puff";
    const fogStyle = WIDE_VENUE_FOG[slug];
    if (fogStyle) {
      puff.classList.add("map-fog-puff--venue-wide", `map-fog-puff--${fogStyle.shape}`);
      puff.style.setProperty("--puff-scale", String(fogStyle.scale));
    }
    if (visibleSlugs.has(slug)) {
      puff.classList.add("map-fog-puff--receded");
    }
    puff.style.left = `${pos.x}%`;
    puff.style.top = `${pos.y}%`;
    fogLayer.appendChild(puff);
  }

  if (!fogLayer.querySelector(".map-fog-puff--ambient")) {
    renderAmbientFog(fogLayer);
  }
}

async function loadVenues() {
  const statusEl = document.getElementById("status");
  const mapLayerEl = document.querySelector(".map-layer");
  const venueIconsEl = document.getElementById("venue-icons");
  const assetVersion = "map-icons-1";

  const assetURL = (url) => {
    if (!url || typeof url !== "string") return url;
    if (!url.startsWith("/assets/")) return url;
    return `${url}${url.includes("?") ? "&" : "?"}v=${assetVersion}`;
  };

  initInfoBoothModal();
  statusEl.textContent = "Loading venues...";
  venueIconsEl.innerHTML = "";

  try {
    const response = await fetch("/api/map/visibility", {
      credentials: "include",
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }

    const payload = await response.json();

    if (!payload.ok) {
      throw new Error(payload.error || "Request failed");
    }

    const venues = Array.isArray(payload.data)
      ? payload.data.filter((venue) => !isHiddenMainMapVenueSlug(venue?.slug))
      : [];
    mapState.venues = venues;

    statusEl.textContent = `${venues.length} venue${venues.length === 1 ? "" : "s"} visible`;

    const fallbackPositions = {
      library: { x: 70, y: 87 },
      "writers-room": { x: 29, y: 58 },
      "first-theater": { x: 69, y: 62 },
      "middle-school-stage": { x: 105, y: 50 },
      "the-cave": { x: 22, y: 23 },
      "grants-cabin": { x: 28, y: 21 },
      "audition-hall": { x: 29, y: 37 },
      "producers-office": { x: 38, y: 14 },
      "directors-chair": { x: 55, y: 23 },
      greenroom: { x: 88, y: 47 },
      trailers: { x: 71, y: 13 },
      "third-place": { x: 50, y: 50 },
      "show-runs": { x: 52, y: 11 },
      catharsis: { x: 37, y: 69 },
      "victory-theater": { x: 83, y: 77 },
      workshop: { x: 75, y: 33 },
      warehouse: { x: 75, y: 27 },
      "soil-experts": { x: 4, y: 65 },
      storyboards: { x: 81, y: 19 },
      construction: { x: -15, y: 57 },
      "info-booth": { x: 50, y: 93 },
    };

    venueIconsEl.innerHTML = "";
    for (const venue of venues) {
      if (venue.slug === "info-booth") {
        continue;
      }
      const pin = createVenuePin(venue, assetURL, fallbackPositions);
      venueIconsEl.appendChild(pin);
    }

    const infoBoothPin = createVenuePin(
      {
        slug: "info-booth",
        name: "Info Booth",
        icon_url: "/assets/infobooth.png",
      },
      assetURL,
      fallbackPositions,
    );
    venueIconsEl.appendChild(infoBoothPin);

    // Kernel 95 Pass 6: harmless no-op for a signed-out visitor (the
    // shared helper's fetch fails closed against an unauthenticated
    // /api/messages and just leaves the pip off).
    window.VictoryMailboxBadge?.refreshPips?.(venueIconsEl);

    renderMapFog(venues, fallbackPositions);
    renderMapMenu();
    mapState.pinsReady = true;
    maybeStartCampusTour();
  } catch (error) {
    statusEl.textContent = "Failed to load venues.";
    console.error(error);
  }
}

async function loadAccountLink() {
  const accountLink = document.getElementById("account-link");
  const accountMenu = document.getElementById("account-menu");
  const accountMenuToggle = document.getElementById("account-menu-toggle");
  const accountProfileLink = document.getElementById("account-profile-link");
  const accountLogoutButton = document.getElementById("account-logout-button");
  const infoBoothGuestActions = document.getElementById("info-booth-guest-actions");
  const infoBoothMailbox = document.getElementById("info-booth-mailbox");
  const appShell = document.querySelector(".app-shell");
  const mapLayerEl = document.querySelector(".map-layer");
  if (!accountLink) return;

  const setMapSignedInState = (signedIn) => {
    if (infoBoothGuestActions) {
      infoBoothGuestActions.hidden = Boolean(signedIn);
    }
    if (infoBoothMailbox) {
      infoBoothMailbox.hidden = !signedIn;
      // Kernel 95 Pass 6: only worth checking once the link is actually
      // shown -- window.VictoryMailboxBadge silently no-ops (returns
      // false) against an unauthenticated /api/messages anyway, but
      // there's no reason to make that request for a signed-out visitor.
      if (signedIn) window.VictoryMailboxBadge?.refreshPips?.();
    }
    if (appShell) {
      appShell.classList.toggle("app-shell--signed-in", Boolean(signedIn));
      appShell.classList.toggle("app-shell--anonymous", !signedIn);
    }
    if (!mapLayerEl) return;
    mapLayerEl.classList.toggle("map-layer--signed-in", Boolean(signedIn));
    mapLayerEl.classList.toggle("map-layer--anonymous", !signedIn);
    renderMapMenu();
  };

  setMapSignedInState(false);

  const closeMenu = () => {
    if (accountMenu) accountMenu.hidden = true;
    if (accountMenuToggle) {
      accountMenuToggle.setAttribute("aria-expanded", "false");
      accountMenuToggle.textContent = "▾";
    }
  };

  const openMenu = () => {
    if (!accountMenu || !accountMenuToggle) return;
    accountMenu.hidden = false;
    accountMenuToggle.setAttribute("aria-expanded", "true");
    accountMenuToggle.textContent = "▴";
  };

  if (accountMenuToggle) {
    accountMenuToggle.setAttribute("aria-haspopup", "menu");
    accountMenuToggle.addEventListener("click", (event) => {
      event.preventDefault();
      event.stopPropagation();
      if (accountMenu && accountMenu.hidden) {
        openMenu();
      } else {
        closeMenu();
      }
    });
  }

  accountLink.addEventListener("click", () => {
    closeMenu();
  });

  if (accountProfileLink) {
    accountProfileLink.addEventListener("click", () => {
      closeMenu();
    });
  }

  if (accountLogoutButton) {
    accountLogoutButton.addEventListener("click", async () => {
      try {
        await fetch("/api/auth/logout", {
          method: "POST",
          credentials: "include",
        });
      } catch (error) {
        console.error("logout failed", error);
      } finally {
        window.location.href = "/";
      }
    });
  }

  document.addEventListener("click", (event) => {
    if (!accountMenu || accountMenu.hidden) return;
    const target = event.target;
    if (
      accountMenu.contains(target) ||
      (accountMenuToggle && accountMenuToggle.contains(target)) ||
      accountLink.contains(target)
    ) {
      return;
    }
    closeMenu();
  });

  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      if (soilExpertsState.modal && !soilExpertsState.modal.hidden) {
        if (typeof soilExpertsState.close === "function") {
          soilExpertsState.close();
        }
        return;
      }
      if (mapMenuState.panel && !mapMenuState.panel.hidden) {
        if (typeof mapMenuState.close === "function") {
          mapMenuState.close();
        }
        return;
      }
      if (infoBoothState.modal && !infoBoothState.modal.hidden) {
        if (typeof infoBoothState.close === "function") {
          infoBoothState.close();
        }
        return;
      }
      closeMenu();
    }
  });

  try {
    const response = await fetch("/api/account/me", {
      credentials: "include",
    });

    if (!response.ok) {
      accountLink.textContent = "Log In";
      accountLink.href = "/login/";
      accountLink.title = "Log in";
      setMapSignedInState(false);
      return;
    }

    const payload = await response.json();
      const account = payload?.data || {};

    if (payload && payload.ok) {
      const user = account.user || {};
      const authority = account.authority || {};
      const displayName = identityDisplayName(user);
      const role = authority.is_producer ? "Producer" : roleLabel(authority.current_role);
      accountLink.textContent = `${displayName} · ${role}`;
      accountLink.href = "/account/";
      accountLink.title = "View your account";
      setMapSignedInState(true);
      if (accountMenuToggle) accountMenuToggle.hidden = false;
      if (accountProfileLink) accountProfileLink.href = "/account/";
      closeMenu();
      renderMapMenu();
      maybeStartCampusTour();
      return;
    }

    accountLink.textContent = "Log In";
    accountLink.href = "/login/";
    accountLink.title = "Log in";
    setMapSignedInState(false);
    if (accountMenuToggle) accountMenuToggle.hidden = true;
    closeMenu();
    renderMapMenu();
  } catch (error) {
    accountLink.textContent = "Log In";
    accountLink.href = "/login/";
    accountLink.title = "Log in";
    setMapSignedInState(false);
    if (accountMenuToggle) accountMenuToggle.hidden = true;
    closeMenu();
    renderMapMenu();
  }
}

const initialFogLayer = document.getElementById("map-fog-layer");
if (initialFogLayer) {
  renderAmbientFog(initialFogLayer);
}

loadAccountLink();
loadVenues();
