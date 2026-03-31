async function loadVenues() {
  const statusEl = document.getElementById("status");
  const venueIconsEl = document.getElementById("venue-icons");

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
      };

      icon.src = venue.icon_url || venueIcons[venue.slug] || "/assets/default.png";
      icon.alt = "";

      const label = document.createElement("span");
      label.className = "venue-label";
      label.textContent = venue.name || venue.slug || "Unknown Venue";

      pin.appendChild(icon);
      pin.appendChild(label);

      pin.addEventListener("click", () => {
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
  if (!accountLink) return;

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

    if (payload && payload.signed_in) {
      const displayName = payload.display_name || "Friend";
      accountLink.textContent = `Hello ${displayName}`;
      accountLink.href = "/account/";
      return;
    }

    accountLink.textContent = "Log In";
    accountLink.href = "/login/";
  } catch (error) {
    accountLink.textContent = "Log In";
    accountLink.href = "/login/";
  }
}

loadAccountLink();
loadVenues();