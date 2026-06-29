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

function venueHref(slug) {
  switch (slug) {
    case "library":
      return "/venues/library/";
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
    case "producers-office":
      return "/venues/producers-office/";
    case "directors-chair":
      return "/venues/directors-chair/";
    case "grants-cabin":
      return "/venues/grants-cabin/";
    case "catharsis":
      return "/venues/catharsis/";
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
  alert(`Venue "${venue.name || venue.slug}" is not built yet.`);
}

function createVenuePin(venue, assetURL, fallbackPositions) {
  const pin = document.createElement("button");
  pin.className = "venue-pin";

  if (venue.slug === "victory-theater") {
    pin.classList.add("venue-pin--theater");
  }
  pin.type = "button";
  pin.setAttribute("aria-label", venue.name || venue.slug || "Unknown Venue");

  const fallback = fallbackPositions[venue.slug] || { x: 50, y: 50 };
  const x = typeof venue.map_x === "number" ? venue.map_x : fallback.x;
  const y = typeof venue.map_y === "number" ? venue.map_y : fallback.y;

  pin.style.left = `${x}%`;
  pin.style.top = `${y}%`;

  const icon = document.createElement("img");
  const venueIcons = {
    library: "/assets/librarycc.png",
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
    warehouse: "/assets/warehouse.png",
    "producers-office": "/assets/producersoffice.png",
    "directors-chair": "/assets/directorschair.png",
    "grants-cabin": "/assets/grantsoffice.png",
    catharsis: "/assets/catharsis.png",
  };

  icon.src = assetURL(venue.icon_url || venueIcons[venue.slug] || "/assets/default.png");
  icon.alt = "";

  const label = document.createElement("span");
  label.className = "venue-label";
  label.textContent = venue.name || venue.slug || "Unknown Venue";

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
        link.textContent = venue.name || venue.slug || "Unknown Venue";
        link.addEventListener("click", () => mapMenuState.close?.());
        item.appendChild(link);
      } else {
        const button = document.createElement("button");
        button.type = "button";
        button.textContent = venue.name || venue.slug || "Unknown Venue";
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
      library: { x: 67, y: 87 },
      "first-theater": { x: 69, y: 62 },
      "middle-school-stage": { x: 101, y: 50 },
      "the-cave": { x: 22, y: 23 },
      "grants-cabin": { x: 28, y: 21 },
      "audition-hall": { x: 29, y: 37 },
      "producers-office": { x: 39, y: 14 },
      "directors-chair": { x: 51, y: 24 },
      greenroom: { x: 87, y: 48 },
      trailers: { x: 68, y: 15 },
      catharsis: { x: 36, y: 69 },
      "victory-theater": { x: 82, y: 75 },
      workshop: { x: 75, y: 33 },
      warehouse: { x: 73, y: 27 },
      "soil-experts": { x: 5, y: 66 },
      construction: { x: 26, y: 57 },
      "info-booth": { x: 48, y: 92 },
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

    renderMapMenu();
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
  const appShell = document.querySelector(".app-shell");
  const mapLayerEl = document.querySelector(".map-layer");
  if (!accountLink) return;

  const setMapSignedInState = (signedIn) => {
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

loadAccountLink();
loadVenues();
