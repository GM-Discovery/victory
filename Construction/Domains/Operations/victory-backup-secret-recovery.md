# Victory Backup Secret Recovery

**Purpose.** List exactly what Grant must preserve *outside* the Victory host for a backup to
still be restorable if the host itself is lost. This document names what to keep and where —
it does not contain the secrets themselves (Kernel 77 §9.2, §9.5: never commit them, and don't
make the encrypted backup the only copy of the key that unlocks it).

If the host disk dies today, everything below that is **not** independently preserved is gone
with it.

---

## 1. What must survive independently of this host

| Secret | Where it currently lives | Preserve it in |
|---|---|---|
| `POSTGRES_PASSWORD` | `/opt/victory/.env` (0600, gitignored) | a password manager, not just this host |
| `rclone` crypt password + password2 | generated during setup, stored only in obscured form inside `/root/.config/rclone/rclone.conf` | Bitwarden (already done — see below) |
| Google Drive OAuth token for `victoryvtt.ops@gmail.com` | `/root/.config/rclone/rclone.conf` | the Google account's own credentials (email + password + any 2FA), separately in a password manager; the token itself can be re-minted by re-running `rclone config` against that account |
| Discord `DISCORD_CLIENT_SECRET`, `DISCORD_BOT_TOKEN` | `/opt/victory/.env` | a password manager |
| Brevo SMTP credentials (`SMTP_USERNAME`/`SMTP_PASSWORD`) | `/opt/victory/.env` | the Brevo account's own login, separately preserved |
| This host's SSH access | wherever Grant's SSH keys/host access already live | unchanged by Kernel 77 — not newly introduced |

**Status as of Kernel 77 (2026-08-01):** the rclone crypt password and password2 were generated
with `openssl rand`, shown to Grant once in-session, and confirmed saved to Bitwarden. The
Google account credentials for `victoryvtt.ops@gmail.com` and the Brevo account credentials are
Grant's to store the same way — not captured in this session beyond the account creation itself.

---

## 2. If `.env` is lost but the host disk survives

`POSTGRES_PASSWORD` in `.env` is the live PostgreSQL role password. If `.env` is deleted but
the container and volume still exist, the database itself is not lost — only the password
Victory uses to reach it.

```bash
docker exec -it victory-postgres psql -U postgres -c "ALTER USER victory WITH PASSWORD '<new password>';"
```

then write the new value into a recreated `.env` (see `.env.example` for the full variable
list) and `docker compose up -d`.

## 3. If the whole host is lost

1. Provision a new host with Docker and `docker compose`.
2. Clone the Victory git repository.
3. Recreate `/opt/victory/.env` from the password-manager copies in §1.
4. Install `rclone`; recreate `/root/.config/rclone/rclone.conf` by re-running `rclone config`
   against the `victoryvtt.ops@gmail.com` Google account (interactive OAuth) and re-entering the
   preserved crypt password/password2 for the `victoryvtt-crypt` remote — using the **same**
   crypt password reconnects to the **same** encrypted files; a different password produces an
   empty-looking remote, not an error, so get this right before assuming data is gone.
5. Follow `victory-restore-runbook.md` §8 (restoring over a genuinely empty live database)
   using the most recent `full-*` archive from `victoryvtt-crypt:VictoryBackups/full/`.
6. Follow `victory-account-recovery-runbook.md` to re-establish operator access if needed.

## 4. What Victory itself never needs

The application (`victory-backend` container) has no Google Drive credentials at all and never
requires one to start, run, or serve traffic — backup and restore are host/operator operations
layered outside the application, exactly as Kernel 77 §1.4 and §4.2 require. Losing or rotating
the rclone config does not affect Victory's ability to run; it only affects whether new backups
can leave the host until the config is fixed, which `backup-status.json` will show as a
failure, not a crash.
