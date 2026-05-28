const infoBoothState = {
  initialized: false,
  modal: null,
  closeButton: null,
};

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

async function loadVenues() {
  const statusEl = document.getElementById("status");
  const mapLayerEl = document.querySelector(".map-layer");
  const venueIconsEl = document.getElementById("venue-icons");

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

    const venues = Array.isArray(payload.data) ? payload.data : [];

    statusEl.textContent = `${venues.length} venue${venues.length === 1 ? "" : "s"} visible`;

    if (venues.length === 0) {
      return;
    }

      const fallbackPositions = {
        library: { x: 67, y: 87 },
        "first-theater": { x: 73, y: 66 },
        "middle-school-stage": { x: 87, y: 38 },
        "the-cave": { x: 22, y: 23 },
        "grants-cabin": { x: 28, y: 21 },
      "audition-hall": { x: 24, y: 37 },
      "producers-office": { x: 39, y: 14 },
      "directors-chair": { x: 51, y: 24 },
      greenroom: { x: 65, y: 14 },
      trailers: { x: 77, y: 53 },
      catharsis: { x: 37, y: 77 },
      "victory-theater": { x: 82, y: 78 },
      workshop: { x: 77, y: 38 },
      warehouse: { x: 76, y: 27 },
      "soil-experts": { x: 16, y: 72 },
      construction: { x: 29, y: 57 },
      "info-booth": { x: 50, y: 95 },
    };

    for (const venue of venues) {
      const pin = document.createElement("button");
      pin.className = "venue-pin";

      if (venue.slug === "victory-theater") {
        pin.classList.add("venue-pin--theater");
      }
      pin.type = "button";
      pin.setAttribute(
        "aria-label",
        venue.name || venue.slug || "Unknown Venue"
      );

      const fallback = fallbackPositions[venue.slug] || { x: 50, y: 50 };
      const x = typeof venue.map_x === "number" ? venue.map_x : fallback.x;
      const y = typeof venue.map_y === "number" ? venue.map_y : fallback.y;

      pin.style.left = `${x}%`;
      pin.style.top = `${y}%`;

      const icon = document.createElement("img");
      const venueIcons = {
        library: "/assets/librarycc.png",
        "first-theater": "/assets/default.png",
        "middle-school-stage": "/assets/default.png",
        "soil-experts": "/assets/mudfarm.png",
        "info-booth": "/assets/infobooth.png",
        "audition-hall": "/assets/audition-hall.png",
        "the-cave": "/assets/cave.png",
        "workshop": "/assets/workshop.png",
        "victory-theater": "/assets/victorytheater.png",
        "construction": "/assets/construction.png",
        "greenroom": "/assets/Greenroom.png",
        "trailers": "/assets/trailers.png",
        warehouse: "/assets/warehouse.png",
        "producers-office": "/assets/producersoffice.png",
        "directors-chair": "/assets/directorschair.png",
        "grants-cabin": "/assets/grantsoffice.png",
        catharsis: "/assets/catharsis.png",
      };

      icon.src = venue.icon_url || venueIcons[venue.slug] || "/assets/default.png";
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

      pin.addEventListener("click", () => {
        if (venue.slug === "info-booth") {
          if (typeof infoBoothState.open === "function") {
            infoBoothState.open();
          }
          return;
        }

        if (venue.slug === "library") {
          window.location.href = "/venues/library/";
          return;
        }

        if (venue.slug === "first-theater") {
          window.location.href = "/venues/first-theater/";
          return;
        }

        if (venue.slug === "middle-school-stage") {
          window.location.href = "/venues/middle-school-stage/";
          return;
        }

        if (venue.slug === "warehouse") {
          window.location.href = "/venues/warehouse/";
          return;
        }

        if (venue.slug === "soil-experts") {
          window.location.href = "/venues/soil-experts/";
          return;
        }

        if (venue.slug === "the-cave") {
          window.location.href = "/venues/the-cave/";
          return;
        }

        if (venue.slug === "audition-hall") {
          window.location.href = "/venues/audition-hall/";
          return;
        }

        if (venue.slug === "victory-theater") {
          window.location.href = "/venues/victory-theater/";
          return;
        }

        if (venue.slug === "construction") {
          window.location.href = "/venues/construction/";
          return;
        }

        if (venue.slug === "workshop") {
          window.location.href = "/venues/workshop/";
          return;
        }

        if (venue.slug === "greenroom") {
          window.location.href = "/venues/greenroom/";
          return;
        }

        if (venue.slug === "trailers") {
          window.location.href = "/venues/trailers/";
          return;
        }

        if (venue.slug === "producers-office") {
          window.location.href = "/venues/producers-office/";
          return;
        }

        if (venue.slug === "directors-chair") {
          window.location.href = "/venues/directors-chair/";
          return;
        }

        if (venue.slug === "grants-cabin") {
          window.location.href = "/venues/grants-cabin/";
          return;
        }

        if (venue.slug === "catharsis") {
          window.location.href = "/venues/catharsis/";
          return;
        }

        alert(`Venue "${venue.name || venue.slug}" is not built yet.`);
      });

      venueIconsEl.appendChild(pin);
    }
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
    }
    if (!mapLayerEl) return;
    mapLayerEl.classList.toggle("map-layer--signed-in", Boolean(signedIn));
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
    const response = await fetch("/api/session/me", {
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
    const session = payload?.data || {};

    if (payload && payload.signed_in) {
      const displayName = session.display_name || "Friend";
      accountLink.textContent = `Hello ${displayName}`;
      accountLink.href = "/account/";
      accountLink.title = "View your account";
      setMapSignedInState(true);
      if (accountMenuToggle) accountMenuToggle.hidden = false;
      if (accountProfileLink) accountProfileLink.href = "/account/";
      closeMenu();
      return;
    }

    accountLink.textContent = "Log In";
    accountLink.href = "/login/";
    accountLink.title = "Log in";
    setMapSignedInState(false);
    if (accountMenuToggle) accountMenuToggle.hidden = true;
    closeMenu();
  } catch (error) {
    accountLink.textContent = "Log In";
    accountLink.href = "/login/";
    accountLink.title = "Log in";
    setMapSignedInState(false);
    if (accountMenuToggle) accountMenuToggle.hidden = true;
    closeMenu();
  }
}

loadAccountLink();
loadVenues();
