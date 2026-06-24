function shortUserID(userID) {
  if (!userID) return "Unknown Participant";
  return String(userID).slice(0, 8);
}

function personaDisplayName(persona) {
  if (persona && typeof persona === "object") {
    const displayName = typeof persona.display_name === "string" ? persona.display_name.trim() : "";
    if (displayName) return displayName;
    const handle = typeof persona.handle === "string" ? persona.handle.trim() : "";
    if (handle) return handle;
  }
  return "";
}

function identityDisplayName(identity) {
  if (!identity || typeof identity !== "object") return "Unknown Participant";

  const personaName = personaDisplayName(identity.persona);
  if (personaName) {
    const displayName = typeof identity.display_name === "string" ? identity.display_name.trim() : "";
    if (displayName && displayName !== personaName) {
      return `${personaName} (${displayName})`;
    }
    return personaName;
  }

  const displayName = typeof identity.display_name === "string" ? identity.display_name.trim() : "";
  const handle = typeof identity.handle === "string" ? identity.handle.trim() : "";
  const genericDisplayNames = new Set(["web user", "webuser", "browser user", "browser", "account", "user"]);
  if (displayName && !genericDisplayNames.has(displayName.toLowerCase())) return displayName;
  if (handle) return handle;

  return shortUserID(identity.user_id);
}

function roleLabel(role) {
  switch ((role || "").toLowerCase()) {
    case "producer":
      return "Producer";
    case "director":
      return "Director";
    case "cast":
    case "actor":
      return "Cast";
    case "crew":
      return "Crew";
    case "audience":
      return "Audience";
    default:
      return "Audience";
  }
}

function roleBadgeClass(role) {
  const normalized = (role || "").toLowerCase();
  return `role-badge--${normalized === "actor" ? "cast" : normalized || "audience"}`;
}
