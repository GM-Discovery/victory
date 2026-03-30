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
      "the-cave": { x: 70, y: 39 },
    };

    for (const venue of venues) {
      const pin = document.createElement("button");
      pin.className = "venue-pin";
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

        alert(`Venue "${venue.name || venue.slug}" is not built yet.`);
      });

      venueIconsEl.appendChild(pin);
    }
  } catch (error) {
    statusEl.textContent = "Failed to load venues.";
    console.error(error);
  }
}

loadVenues();