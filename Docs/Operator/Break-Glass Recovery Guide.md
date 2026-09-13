# Break-Glass Recovery Guide

This is the advanced, terminal-based path for getting back into Victory when ordinary login is unavailable. It exists so that losing access to your own account is never the same thing as losing Victory itself.

If you're comfortable with a command line, this document is safe to follow directly. If you're not, this is the point to ask someone who is, or to open a support request — see the Troubleshooting Guide first for anything short of "I am completely locked out."

---

## 1. What this is

`victory-recover` is a small command-line tool that runs directly against your database from the server's own shell. It needs no working Victory login, no working web server, and no working session — only a working database connection.

It deliberately **cannot**:

- create a hidden administrator account (Operator authority is tied to one specific handle you control, so any change it makes is visible in your own `users` table);
- set a password for you directly (it always routes through the same password-reset flow a real user would use);
- read anyone's private content.

It is not exposed through the web at all — every subcommand runs from a trusted shell with direct database access, not a public endpoint, so there is no route for anyone without that shell access to reach it.

---

## 2. Before you use it

Gather this first — it saves time and avoids guessing:

- Your database connection string (`DATABASE_URL`).
- The handle your Operator account is supposed to use (`OPERATOR_HANDLE`).
- If applicable: the email or Discord identity you normally sign in with.

Start with the read-only command, always:

```bash
victory-recover whoami --handle YOUR_HANDLE
```

This prints the account's ID, display name, email, linked Discord identity, whether it holds Operator authority, whether it has a password set, how many active sessions it has, and every Location membership with its role — without changing anything. Run this before any mutating command so you know exactly what state you're actually in.

---

## 3. Common situations

**The Operator account doesn't exist yet** (a freshly built database with no accounts):

```bash
victory-recover bootstrap --operator-handle YOUR_HANDLE --email you@example.com
```

Creates the account and prints a one-time recovery link. The account has no password and no session until that link is used — an unclaimed bootstrap account is not a standing back door.

**You signed in through Discord and it created a new, non-Operator account** (this happens because Discord sign-in always creates a fresh account first; Operator authority has to be explicitly moved onto it):

```bash
victory-recover claim --discord-id YOUR_DISCORD_ID --operator-handle YOUR_HANDLE
victory-recover grant --handle YOUR_HANDLE --location YOUR_LOT_SLUG --role producer
```

The account that previously held the handle is renamed, not deleted — anything it owned (Characters, journals) survives under the renamed account.

**You can't log in and can't use Discord either:**

```bash
victory-recover recover --handle YOUR_HANDLE --base-url https://your-install.example
```

Prints a one-time, one-hour password-reset link. Only its hash is ever stored; the real link is shown once in your terminal and nowhere else.

**Everything is fine except a session should be forcibly ended** (a lost device, a shared machine):

```bash
victory-recover revoke --handle YOUR_HANDLE
```

Add `--all-sessions` only if you mean every user on the install, not just one account.

---

## 4. What gets logged

Every mutating subcommand prints exactly what it changed as it runs — there is no silent write. Nothing it does is hidden from the `whoami` view afterward: a claimed handle, a granted role, or a revoked session all show up the next time you check.

---

## 5. Before you escalate to support

If break-glass recovery itself isn't working — the tool errors, the database is unreachable, or something looks wrong beyond "I forgot my password" — gather a support bundle (see the Troubleshooting Guide) before reaching out. It captures logs without exposing your database credentials, and saves a round trip.

---

## 6. What not to do

- Don't hand-edit the `users` table directly to fix a login problem. Every one of the situations above has a supported command; direct SQL surgery risks leaving the account in a state the application doesn't expect.
- Don't share the output of `recover` or `bootstrap` with anyone — it's a live, redeemable credential, not a log line.
- Don't run `revoke --all-sessions` to fix a single account's problem — it signs out every user on the install.
