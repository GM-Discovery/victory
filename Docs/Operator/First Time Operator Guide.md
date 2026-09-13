# First Time Operator Guide

Welcome. This guide explains what running Victory means, in plain terms — not Go, not Docker, not Postgres, just the product and the decisions you'll actually make.

---

## 1. What an Operator is

An **Operator** is whoever installed and runs this particular copy of Victory — your own "lot." You control the installation itself: who's invited, where the data lives, when it updates, and how it's backed up.

An Operator is different from a **Producer**, who runs productions *inside* Victory (Shows, Showings, casting, scenes). You can be both — most people running Victory for their own group are — but they're separate hats. See [Operator versus Producer](#2-operator-versus-producer) below.

Nobody but you needs to think about any of this. Your players just open a link and play.

---

## 2. Operator versus Producer

| | Operator | Producer |
|---|---|---|
| Authority over | The installation itself — the server, updates, backups, who can access it at all | A production inside Victory — a Show, its cast, its Showings |
| Typical concerns | "Is Victory running? Who's invited? Is it backed up?" | "Who's playing what Character? When's the next Showing?" |
| Needs infrastructure knowledge? | A little — covered by this guide | No — ordinary production work never requires it |

If you're running Victory just for your own table, you'll wear both hats, but you don't need infrastructure knowledge to do the Producer half. Nothing in Directing, casting, or running a Show requires knowing anything in this guide.

---

## 3. Naming and orienting your lot

The first time you run Victory, you'll be asked to name your **lot** — the human name for this installation ("The Murray Family Theater," "Thursday Night Table," anything). This name is what your players see; it's not a technical setting and can't be confused with a server address.

---

## 4. Your Operator account and recovery

Your first-run setup creates a real Operator account on this installation — a handle and password specific to *this* machine, not a shared default. If you ever forget your password or lock yourself out, see the break-glass recovery guide (`Docs/Operator/Break-Glass Recovery Guide.md`) for the terminal-based recovery path. It requires access to the machine itself — nobody can recover your Operator account remotely without that.

---

## 5. Where your data lives

Depends on how you installed Victory:

- **Windows (the consumer installer):** everything persistent — your database, uploaded images, backups, exports, logs, and the generated configuration file — lives under `%LocalAppData%\Victory`. The application program files live somewhere else entirely and get replaced wholesale on every update; only `%LocalAppData%\Victory` is yours and is never touched by an update or uninstall.
- **Linux/server install:** your database lives in Postgres's own data directory, and uploaded files live under whatever `STORAGE_ROOT` your `.env` sets (defaults to `/opt/victory/storage`).

Either way: your data is *your* data, on *your* machine. Nothing about running Victory sends your campaign's content anywhere else. (See the privacy disclosure in §9 for what "your machine" itself means for privacy.)

---

## 6. Your remote/share URL

If you've turned on remote access, Victory gives you a real, shareable `https://` link your players can use from anywhere — no port forwarding, no router configuration, no certificate to manage. See `Docs/Operator/Remote Access Guide.md` for the full explanation of how this works and what to expect if it's ever briefly unavailable.

If you haven't turned on remote access, Victory is still fully usable on your own local network at `http://localhost:<port>` — remote access is optional, not required to play.

---

## 7. Inviting your first people

Once your lot exists, invite people directly — an invite link works whether they'll ultimately play, direct, or produce. They'll set their own password when they accept it. Nothing about accepting an invite requires them to understand Discord, hosting, or anything in this guide.

---

## 8. Your first Producer or production

If you intend to run a Show yourself, you're your own first Producer — there's no separate infrastructure step to "become" one. Create your first production from inside Victory the same way any Producer would; this guide's job stops at getting you and your players *into* Victory, not running a Show once you're there.

---

## 9. Privacy: what "self-hosted" actually means

Be plain with the people you invite about this:

> **The server Operator may have technical access to the installation's database, files, backups, and administrative recovery tools.**

Victory's application-level permissions (roles, visibility, private messages) protect people from *each other* — from another ordinary user seeing something they shouldn't. They do not, and cannot, protect anyone from the person who actually administers the machine the software runs on. That's true of every self-hosted application, not a Victory-specific limitation, and it's worth saying out loud to the people you invite rather than leaving it implied.

This is current architectural truth, verified during Kernel 96's security hardening pass — not a claim of any formal certification.

---

## 10. Optional: Discord integration

Victory works completely without Discord — nothing in first-run setup requires it. If you'd like presence/voice integration with a Discord server your group already uses, see `Docs/Operator/Discord Integration Guide.md` once you're up and running. It's entirely optional and can be turned on or off at any time.

---

## 11. Updates

By default, Victory checks for updates automatically and applies them at a quiet time rather than interrupting a live Show. See `Docs/Operator/Update Guide.md` for exactly how this works and how to check your current version.

---

## 12. Backups

See `Docs/Operator/Backup Guide.md` for what's backed up automatically (if anything, on your install type), how to take a manual backup, and how to restore one. Don't assume backups exist until you've confirmed it for your own installation — the guide is explicit about what's real today versus what still requires your own manual step.

---

## 13. Diagnostics and support

If something's wrong, start with `Docs/Operator/Troubleshooting Guide.md` — it's organized by symptom, not by infrastructure component. If you need to ask someone else for help, the built-in support-bundle feature (where available) collects the right diagnostic information into one file without you needing to hunt through logs by hand, and without including your password or database credentials.

---

## 14. What you can technically access

As Operator, you have the same access described in §9: the real database, the real files, the real backups. Victory doesn't hide this from you or pretend otherwise — you're the administrator of your own installation, the same as you'd be for any other self-hosted software.
