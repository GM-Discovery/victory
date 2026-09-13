# Discord Integration Guide

Discord integration is entirely optional. Victory installs, runs, and plays fully without it — nothing in first-run setup asks you to configure Discord, and you can turn it on, off, or never touch it at all.

---

## 1. What Discord integration adds

Three independent pieces, each optional on its own:

- **Sign in with Discord** — lets people join using their existing Discord account instead of a Victory-specific password.
- **Server link** — connects Victory to a Discord server (guild) your group already uses, for cross-referencing membership.
- **Voice presence** — shows who's currently in a voice channel on your linked Discord server inside Victory's Presence tray. Victory does not stream or relay audio itself; it only reflects who's present.

You can enable any subset of these. None of them are required for Victory to function.

---

## 2. Before you start: the Discord Developer Portal

You'll need a Discord application, created at Discord's own developer portal (discord.com/developers/applications). Create one there first — this guide assumes you already have:

- an **Application ID**
- a **Client Secret** (for sign-in)
- a **Bot Token** (only if you want server link/voice presence)
- a **Public Key** (only if you want server link)

Treat all of these as secrets. Never share them, commit them to a repository, or paste them somewhere public.

---

## 3. Configuring sign-in with Discord

Set the following in your installation's configuration:

| Setting | What it is |
|---|---|
| `DISCORD_CLIENT_ID` | Your application's Client ID |
| `DISCORD_CLIENT_SECRET` | Your application's Client Secret |
| `DISCORD_REDIRECT_URL` | Where Discord sends people back after they approve — must exactly match a redirect URL registered in the Developer Portal |
| `DISCORD_OAUTH_SCOPES` | Defaults to `identify email` — leave this as-is unless you have a specific reason to change it |
| `DISCORD_OAUTH_ENABLED` | Set to enable sign-in with Discord |

On the Windows installer, these are set through the installer's own configuration UI, not by hand-editing a file. On a Linux/server install, they go in your `.env` file.

All four of Client ID, Client Secret, and Redirect URL must be set and non-empty for sign-in with Discord to activate — Victory checks all three together, not just the "enabled" flag.

---

## 4. Configuring server link and voice presence

Additional settings, only needed if you want server link or voice presence:

| Setting | What it is |
|---|---|
| `DISCORD_APPLICATION_ID` | Same Application ID as above |
| `DISCORD_BOT_TOKEN` | Your bot's token |
| `DISCORD_BOT_PERMISSIONS` | Discord permission bitmask for the bot invite — keep the default unless you know you need to change it |
| `DISCORD_BOT_REDIRECT_URL` | Where Discord sends you after approving the bot's server install |
| `DISCORD_PUBLIC_KEY` | Your application's Public Key |
| `DISCORD_SERVER_LINK_ENABLED` | Set to enable server link |
| `DISCORD_GATEWAY_ENABLED` | Set to enable live voice-presence updates |
| `DISCORD_GATEWAY_INTENTS` | Defaults to a value that includes voice-state visibility — leave this as-is unless you understand Discord Gateway intents specifically |

Server link and voice presence are separate switches from sign-in — you can have people sign in with Discord without linking a server, or link a server without using Discord for sign-in at all.

---

## 5. How Victory stores this configuration

All of the above live in your installation's own configuration (the generated `.env` file, or the Windows installer's equivalent config store) — never in the database, never in a place a player account could read them. Bot tokens and client secrets are never exposed in ordinary logs or diagnostic output.

---

## 6. How to test it

After configuring sign-in, try signing in with Discord from a fresh browser session (or a private/incognito window) — you should land back in Victory already signed in. For server link, check the Operator status surface for a connected/linked confirmation. For voice presence, join a voice channel on the linked Discord server and confirm it appears in Victory's Presence tray.

If something doesn't work, double-check that your `DISCORD_REDIRECT_URL` (and `DISCORD_BOT_REDIRECT_URL`, if used) exactly match what's registered in the Discord Developer Portal — a mismatch here is the most common cause of a failed connection, and Discord's own error messages will usually say so directly.

---

## 7. How to disable it

Clear (or set to disabled) whichever of `DISCORD_OAUTH_ENABLED`, `DISCORD_SERVER_LINK_ENABLED`, or `DISCORD_GATEWAY_ENABLED` you no longer want active, and restart Victory. Accounts that signed in via Discord keep working through their existing Victory session; they just won't be able to use Discord sign-in again until it's re-enabled. Nothing about disabling Discord integration deletes any data.
