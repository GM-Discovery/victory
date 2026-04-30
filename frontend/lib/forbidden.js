function isForbiddenAccessError(payload) {
  return (
    payload?.error === "forbidden" ||
    payload?.error === "not_authenticated" ||
    payload?.error === "ticket_or_auth_required"
  );
}

function isForbiddenAccessResponse(response, payload) {
  return (
    response?.status === 401 ||
    response?.status === 403 ||
    isForbiddenAccessError(payload)
  );
}

function showForbiddenScreen() {
  const forbiddenScreen = document.getElementById("forbidden-screen");
  const shell =
    document.querySelector(".shell") ||
    document.querySelector(".app-shell") ||
    document.querySelector("main");

  if (forbiddenScreen) forbiddenScreen.hidden = false;
  if (shell) shell.hidden = true;

  const connectionStatus = document.getElementById("connection-status");
  if (connectionStatus) connectionStatus.textContent = "Forbidden";
}
