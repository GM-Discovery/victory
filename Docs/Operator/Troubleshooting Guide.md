# Troubleshooting Guide

Start from what you're actually seeing. Each symptom below lists the safest first check before anything more involved.

---

## "Victory will not start"

1. Check whether the process is even running (on Windows: is the Victory app open and not showing an error dialog; on a server: `systemctl status victory` or however you launch it).
2. Check the most recent log output — a startup failure almost always prints why (missing configuration, a port already in use, a database it can't reach).
3. If the log mentions the database specifically, see "database unhealthy" below before anything else.

---

## "I can't reach the local page"

1. Confirm Victory is actually running (see above) before assuming the browser side is the problem.
2. Hit the health endpoint directly: `curl http://localhost:PORT/health` (or open it in a browser). A response like `{"ok":true,...}` means the web server is up — note that this only confirms the server process is answering requests, not that the database behind it is reachable.
3. If the health check itself fails to connect, the process isn't listening on the port you expect — check what port it's actually configured for.

---

## "The remote/share URL isn't reachable"

1. Confirm the local page works first (above) — a remote-access problem on top of a local-start problem is really just the local-start problem.
2. Check whether the remote-access component (the tunnel/relay) shows as connected in Victory's own status area.
3. If it shows disconnected, restarting Victory typically re-establishes it; if it stays disconnected, the relay service itself may be unreachable from your network — check outbound internet access.

---

## "A friend can't connect with the link I gave them"

1. Confirm the link is current — a remote URL can rotate if the tunnel reconnects with a new address; always copy the current one from Victory's own display of it rather than reusing an old one from memory or a saved message.
2. Confirm the Showing/venue they're trying to reach hasn't changed its access settings since you shared the link.
3. Ask what error they actually see — "the page won't load" and "it loads but says forbidden" are different problems (the second is an access/role issue, not a connectivity one).

---

## "Login fails"

1. Check the specific error shown — "wrong password" and "no account found" point in different directions.
2. If it's a forgotten password or a locked-out Operator account, that's the Break-Glass Recovery Guide, not this document.
3. If Discord sign-in specifically fails, see "Discord integration fails" below.

---

## "An update failed"

1. Check the update/launcher log for the specific failure — most update failures are a network interruption mid-download, which simply retries on next launch.
2. Confirm you're not trying to update while a Show is actively live — Victory deliberately avoids applying backend-changing updates during one.
3. If it keeps failing across multiple attempts, generate a support bundle (below) before doing anything more invasive — don't manually delete application files to "force" a reinstall.

---

## "The database seems unhealthy"

1. Check whether PostgreSQL itself is running and reachable independent of Victory (`psql "$DATABASE_URL" -c 'select 1;'`).
2. If Postgres isn't reachable, that's an infrastructure problem outside Victory's own code — check the database service/container status directly.
3. If Postgres is reachable but Victory still reports errors, check Victory's own logs for the actual query/connection error rather than guessing.

---

## "Assets/images are missing"

1. Confirm the storage directory Victory is configured to use actually exists and is readable by the process running Victory.
2. Check whether the asset was ever actually uploaded successfully (an interrupted upload can leave a database reference with no matching file) versus a working asset that stopped rendering (more likely a path/permissions change).

---

## "Live play / WebSocket keeps disconnecting"

1. A single reconnect is normal and expected — Victory's client automatically retries with a backoff and re-syncs state on reconnect, so a brief blip shouldn't lose anything.
2. If it disconnects repeatedly and doesn't stay connected, check for a restrictive network (aggressive proxy/firewall closing long-lived connections) between the client and Victory, especially over a remote/share link rather than a local connection.
3. If it's happening for everyone at once, that points at the server side (a restart, or the remote-access tunnel dropping) rather than any one person's network.

---

## "Discord integration fails"

1. Confirm you actually want Discord integration — Victory works fully without it; if you didn't intentionally configure it, this isn't something to troubleshoot, just leave it off.
2. If you did configure it, check that the callback URL registered in Discord's developer portal still matches Victory's actual public URL — this is the most common cause of a working setup breaking after a URL change (e.g. after a remote-access link rotates).
3. Check Victory's logs for the specific Discord API error rather than assuming it's a Victory-side bug — Discord's own outages and rate limits look identical to a misconfiguration from Victory's side.

---

## Generating a support bundle

If none of the above resolves it, or you're about to ask someone else for help, generate a support bundle first — it saves back-and-forth.

On the Windows build, this is a single action in the application (produces one zip file on your Desktop). It contains recent logs — including every update-check outcome, not just ones that showed a notification — useful for diagnosing exactly what happened and when.

**What it deliberately does not contain:** your database contents, your `.env`/secrets, or credentials of any kind. It's safe to attach to a support request as-is.

**What you should still do before sending it:** a quick skim if you're on a shared or public install — logs can contain venue/Show names and similar in-product context even though they never contain secrets.

**Never** send your `.env` file or database password to anyone claiming to need it for support — no legitimate troubleshooting path requires either, and the support bundle exists specifically so you never have to.

---

## When to stop and use Break-Glass Recovery instead

If the actual problem is "I cannot get into my own Operator account" and none of the login troubleshooting above applies, stop here and go to the Break-Glass Recovery Guide — it's a more direct, more powerful tool than anything in this document, and using it correctly matters more than working through unrelated troubleshooting steps first.
