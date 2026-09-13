# Remote Access Guide

This explains how your players can reach Victory from outside your own home network — no port forwarding, no router configuration, no certificate to buy or install.

---

## 1. How it actually works

Victory uses **Cloudflare Tunnel** to expose your installation securely. By default this is a **Quick Tunnel** — a fresh, unique `https://*.trycloudflare.com` address generated automatically each time Victory starts, requiring no Cloudflare account of your own. If you already have your own Cloudflare account and domain, Victory also supports a **Named Tunnel** with a stable address you control instead.

Either way, the tunnel is outbound-only from your machine — it reaches out to Cloudflare, rather than Cloudflare (or anyone else) reaching in. That's what lets this work with no router configuration: there's nothing incoming to forward.

Your Victory backend itself never listens on anything but your own machine (`127.0.0.1`) — the tunnel software is the only thing that ever talks to it directly. This is also why Windows Firewall never prompts you about Victory: there's genuinely nothing exposed for it to ask about.

---

## 2. Finding and sharing your URL

Once remote access is on, Victory's Status screen shows your current public URL. Copy it and share it with whoever you're inviting — an invite link generated from Victory already uses this real, reachable address automatically, not `localhost`.

A Quick Tunnel's address changes each time Victory restarts. If you restart Victory between sessions, check Status for the current URL before sending out a new invite link — an old Quick Tunnel address won't still work after a restart. A Named Tunnel's address stays the same across restarts, if you've set one up.

---

## 3. What your remote players experience

From a remote player's side, it's an ordinary `https://` link — no certificate warning, no special setup, no app to install beyond a browser. HTTPS is handled entirely by Cloudflare before traffic ever reaches your machine; you never need to obtain or manage a certificate yourself.

---

## 4. If the tunnel is briefly unavailable

Local play is never affected. Victory itself keeps running on your own network regardless of tunnel state — remote access is a separate layer on top of, not a requirement for, Victory working at all. If the tunnel connection drops, Status reflects that plainly rather than silently masking it; anyone already connected locally is unaffected, and remote players will need the tunnel to reconnect (which happens automatically) or a fresh URL if it doesn't recover on its own.

---

## 5. Local/LAN access

Regardless of whether remote access is on, Victory is always reachable at `http://localhost:<port>` from the same machine, and from other devices on your own local network using that machine's local network address. You don't need remote access turned on at all if everyone's playing from the same house or the same network.

---

## 6. What Victory does not do

Victory does not operate its own broker or relay service, and does not collect any metadata about your sessions beyond what Cloudflare's tunnel service necessarily sees to route the traffic (the same as any HTTPS traffic passing through any reverse proxy). There is no "phone home" beyond that, and no requirement to create any account with Victory itself to use remote access.

Do not configure port forwarding for Victory — it's neither required nor expected by this architecture, and doing so would expose your machine directly to the internet unnecessarily.
