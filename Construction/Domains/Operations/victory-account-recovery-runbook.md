# Victory Account Recovery Runbook

**Purpose.** Get the operator back into Victory when normal login is unavailable, without
rolling back the application and without a standing back door.

**Tool.** `backend/cmd/victory-recover`, added in Kernel 76 (§2.2).
**Where it runs.** The trusted server shell on the Victory host. It talks to PostgreSQL
directly and needs no Victory session.

This document contains no secrets. Every command below is safe to paste into a ticket; the
*output* of `recover` and `bootstrap` is not — it contains a live one-time credential.

---

## 0. Setup

```bash
cd /opt/victory/backend
set -a; . /opt/victory/.env; set +a
export DATABASE_URL="postgres://victory:${POSTGRES_PASSWORD}@127.0.0.1:5432/victory?sslmode=disable"
export GOCACHE=/tmp/victory-gocache
```

Run any subcommand with `go run ./cmd/victory-recover <subcommand>`, or build it once with
`go build -o /usr/local/bin/victory-recover ./cmd/victory-recover`.

`victory-recover help` prints usage.

---

## 1. Find out what is actually true

Always start here. It answers "which account am I even talking about" before you change
anything.

```bash
go run ./cmd/victory-recover whoami --handle straturli
go run ./cmd/victory-recover whoami --discord-id 694402257331945562
go run ./cmd/victory-recover whoami --email grant@amurray.family
```

Prints the user id, handle, display name, email, linked Discord id, whether the account holds
operator authority, whether a password is set, how many live sessions exist, and every
Location membership with its role and active flag.

A target that matches more than one account is an error, not a guess.

---

## 2. The operator account does not exist at all

This is the state right after a database rebuild: migrations seed Locations and Venues but no
human accounts.

```bash
go run ./cmd/victory-recover bootstrap \
  --operator-handle straturli \
  --email grant@amurray.family
```

Creates `straturli` with `producer` at `amurray-family`, then prints a one-time recovery link.
The account has **no password and no session** until that link is redeemed — a bootstrapped
account nobody claims is not a usable back door.

Re-running is safe: if the handle already exists it reports so and creates nothing.

---

## 3. You signed in with Discord and landed in the wrong account

Discord signup creates a *new* account with handle `discord_<id>`. Operator authority in
Victory is "your handle equals `OPERATOR_HANDLE`" (see `access.IsOperatorUser`), so that new
account is **not** the operator, even though it is you.

```bash
go run ./cmd/victory-recover claim --discord-id 694402257331945562 --operator-handle straturli
go run ./cmd/victory-recover grant --handle straturli --location amurray-family --role producer
```

`claim` moves the operator handle onto the Discord-linked account. Any account previously
holding the handle is renamed to `straturli_released_<timestamp>` rather than deleted — it may
still own Characters and journals.

---

## 4. You cannot log in and Discord is unavailable

This is the path that keeps account ownership from depending on Discord staying up.

```bash
go run ./cmd/victory-recover recover --handle straturli
```

Prints `https://victory.amurray.family/login/reset.html?token=…`, valid for one hour, single
use. Open it in a browser and set a password; redeeming it signs you in immediately.

Only the SHA-256 hash is stored, matching what `/api/auth/password-reset/confirm` verifies.
The raw token is printed once to your terminal and never written to a log.

> The redemption page lives at `/login/reset.html`, not `/auth/…`, because Caddy
> reverse-proxies `/auth/*` to the backend.

---

## 4a. Self-service recovery now exists (Kernel 77)

As of Kernel 77, an ordinary user who is locked out no longer needs the operator at all, as
long as they previously verified a recovery email address:

1. `/login/forgot.html` — enter the email on file.
2. If — and only if — that address matches an account **and** the account's
   `users.email_verified_at` is set, Victory emails a one-time link through the configured
   SMTP relay (Brevo). The response is identical either way, so this page never reveals
   whether an address exists or is verified.
3. The link opens `/login/reset.html?token=…` — the same redemption page break-glass links
   use, because both mint the identical token shape into `auth.password_reset_tokens`.
4. Redeeming it sets a password, signs the user in, and revokes every other session on the
   account.

**Verifying a recovery email** happens from the Account page (`/account/`) — "Verify Email for
Recovery" sends a link to `/account/verify-email.html?token=…`, valid 24 hours, via
`auth.email_verification_tokens`. Changing the email on file always clears verification;
Discord-adopted emails are not verified automatically.

This does not replace anything above. `victory-recover` remains the operator's own path and
is unaffected — it mints tokens directly rather than emailing them, so it works even if SMTP
is down or the user's email was never verified. If `RECOVERY_EMAIL_ENABLED` is unset or SMTP
is not fully configured, `/api/auth/password-reset/request` stays closed (`410`) exactly as
Kernel 76 left it, and `victory-recover` is the only path — see
`Construction/Domains/Operations/victory-backup-secret-recovery.md` for where the SMTP credentials
themselves are kept.

---

## 5. Memberships were lost but the account is fine

```bash
go run ./cmd/victory-recover grant --handle straturli --location amurray-family --role producer
```

Idempotent — re-running reactivates an existing membership rather than failing on the unique
constraint. Valid roles: `producer`, `director`, `cast`, `crew`, `audience`.

---

## 6. A session may be compromised

```bash
go run ./cmd/victory-recover revoke --handle straturli     # one account
go run ./cmd/victory-recover revoke --all-sessions          # everyone, including you
```

`--all-sessions` refuses to run if you also pass a target, so you cannot ask for one and get
the other. After it, everyone signs in again — make sure you can complete §4 first.

---

## 7. Full recovery from a rebuilt database

The sequence proven end-to-end during Kernel 76 on the live host:

```bash
# 1. backend boots, applies migrations, seeds Locations and Venues
docker compose up -d backend
docker logs victory-backend 2>&1 | grep migrate:

# 2. create the operator account and get a claim link
go run ./cmd/victory-recover bootstrap --operator-handle straturli --email grant@amurray.family

# 3. open the printed link, set a password  -> signed in

# 4. confirm
go run ./cmd/victory-recover whoami --handle straturli
```

Verified result: `is_operator: true`, `producer` at `amurray-family`, and all sixteen Venues
visible including `grants-cabin`.

---

## 8. What this tool deliberately cannot do

- Create a hidden superuser. Operator authority is the `OPERATOR_HANDLE` handle and nothing
  else, so every claim is visible in the `users` table.
- Set a password directly. Recovery goes through the ordinary reset-confirm endpoint.
- Read private content — journals, messages, relationship notes, or uploads.
- Run without an explicit target. Every subcommand refuses to guess.

---

## 9. If the tool itself will not run

- **`DATABASE_URL is not set`** — you skipped §0, or `.env` lacks `POSTGRES_PASSWORD`.
- **`database connect failed`** — check `docker ps` for `victory-postgres`. The database
  listens on `127.0.0.1:5432` only; there is no remote path to it by design.
- **`no Location with slug "amurray-family"`** — the database has not been migrated and
  seeded. Boot the backend once, then retry.
- **Password lost entirely** — the credential lives only in `/opt/victory/.env`. If that file
  is lost, reset the role from the postgres superuser inside the container
  (`ALTER USER victory WITH PASSWORD '…'`) and write the new value back to `.env`.
