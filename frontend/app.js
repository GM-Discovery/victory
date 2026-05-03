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
        "workshop": { x: 62, y: 47 },
        "info-booth": { x: 50, y: 88 },
        "audition-hall": { x: 30, y: 27 },
        "the-cave": { x: 20, y: 17 },
        "victory-theater": { x: 50, y: 70 },
        "construction": { x: 37, y: 50 },
        "greenroom": { x: 60, y: 30 },
        "trailers": { x: 67, y: 38 },
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
        "info-booth": "/assets/infobooth.png",
        "audition-hall": "/assets/audition-hall.png",
        "the-cave": "/assets/cave.png",
        "workshop": "/assets/workshop.png",
        "victory-theater": "/assets/victorytheater.png",
        "construction": "/assets/construction.png",
        "greenroom": "/assets/Greenroom.png",
        "trailers": "/assets/trailers.png",
      };

      icon.src = venue.icon_url || venueIcons[venue.slug] || "/assets/default.png";
      icon.alt = "";

      const label = document.createElement("span");
      label.className = "venue-label";
      label.textContent = venue.name || venue.slug || "Unknown Venue";

      pin.appendChild(icon);
      pin.appendChild(label);

      pin.addEventListener("click", () => {
        if (venue.slug === "info-booth") {
          if (typeof infoBoothState.open === "function") {
            infoBoothState.open();
          }
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
  if (!accountLink) return;

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
      return;
    }

    const payload = await response.json();
    const session = payload?.data || {};

    if (payload && payload.signed_in) {
      const displayName = session.display_name || "Friend";
      accountLink.textContent = `Hello ${displayName}`;
      accountLink.href = "/account/";
      accountLink.title = "View your account";
      if (accountMenuToggle) accountMenuToggle.hidden = false;
      if (accountProfileLink) accountProfileLink.href = "/account/";
      closeMenu();
      return;
    }

    accountLink.textContent = "Log In";
    accountLink.href = "/login/";
    accountLink.title = "Log in";
    if (accountMenuToggle) accountMenuToggle.hidden = true;
    closeMenu();
  } catch (error) {
    accountLink.textContent = "Log In";
    accountLink.href = "/login/";
    if (accountMenuToggle) accountMenuToggle.hidden = true;
    closeMenu();
  }
}

loadAccountLink();
loadVenues();
