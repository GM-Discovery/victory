# Victory User Data Export Format

**Status:** implemented, Kernel 77 (2026-08-01).
**Code:** `backend/internal/identity/account_export.go`.

---

## 1. Request lifecycle

```
POST   /api/account/export           start a new export (409 if one is already pending/running)
GET    /api/account/export/status    poll for progress
GET    /api/account/export/download  download the ready archive (session-authenticated, no token)
DELETE /api/account/export           delete the current export early
```

An export runs as a background goroutine kicked off from the request handler (a fresh
`context.Background()`, not the request's own context, which is cancelled the moment the HTTP
handler returns) — this satisfies "background-safe execution without requiring an asynchronous
product system" without adding a queue or worker process. `account_export_jobs` tracks status:
`pending → running → ready` (or `failed`). A completed export expires 48 hours after
completion; an hourly sweep (started from `main()`) removes expired archives from disk even if
no one ever checks status again, and the download/status endpoints also lazily expire on access.

Download is resolved entirely from the session — no token, no id, no predictable URL segment.
This is the "authenticated download" alternative Kernel 77 §7.5 explicitly permits as
equivalent to an unguessable token.

## 2. Archive layout

```
manifest.json
README.md
account/
  profile.json           handle, display_name, email, user_id
  memberships.json        every Location membership, role, active flag
characters/
  index.json               id, name, folder for each owned Character
  <name>-<id8>/
    character.json          name, pronouns, tagline, descriptions, workbook_status
    journal.md               every journal entry the user authored for this Character, as Markdown
    mechanics.json            workbook entries (page_key, entry_type, title, body, payload)
relationships/
  relationships.json        the user's own observations only (who it's about, by handle,
                             never the subject's email/Discord id)
  notes/<about-handle>.md    the user's own relationship journal entries about that person
messages/
  messages.json              every message sent or received, with direction
activity/
  authored-actions.json     up to 2000 most recent Actions the user authored (id, type,
                             timestamp only -- a reference list, not a full Production dump)
uploads/
  manifest.json              one entry per owned/uploaded asset; "skipped" noted if the
                              500 MB export size ceiling was reached
  files/                     the actual files, named "<asset-id>-<original-filename>"
```

Only folders that actually apply to the account are created.

## 3. What's deliberately excluded

Password hashes, session tokens, password-reset tokens, OAuth tokens/state, server secrets, and
any other user's private content (their email, Discord id, relationship notes, journals, or
unpublished profile material) — enforced structurally: every query in `buildExportArchive` is
scoped to the requesting `user_id`, and relationship export includes only the *subject's public
handle* as minimal shared context, never their private fields.

## 4. Security properties

- **No client-selected target.** Every query resolves the account from the session; nothing
  accepts a `user_id`, `email`, or `handle` parameter as authority (Kernel 77 §7.3).
- **Bounded size.** A 500 MB ceiling on copied upload bytes; assets beyond it are listed in the
  manifest as skipped rather than silently omitted.
- **No archive path traversal.** The zip writer normalizes every path with `filepath.Rel` under
  the staging directory and explicitly refuses anything that would escape it (defense in depth;
  `filepath.Walk` under a known root cannot actually produce such a path, but the check exists
  rather than assuming that).
- **Safe filenames.** Uploaded-file names inside the archive are sanitized
  (`exportSafeName`) before being used as a path component.
- **Rate limited.** `POST /api/account/export` shares the credential rate limiter with
  `/api/auth/*` and `/api/account/email` — this is the most expensive endpoint Victory has.

## 5. Test coverage

`backend/internal/identity/account_export_test.go`: a full request → poll → download round
trip against a real fixture account (profile, membership, Character, private journal entry),
verifying the archive reopens as a valid zip, contains the expected top-level files, the
private journal entry is present, manifest counts match, and no secret-shaped string (password
hash fragment, the raw session token) appears anywhere in the archive bytes. Separately: a
second user cannot download or see the status of someone else's export (404, not 403 — the
export simply doesn't exist from their perspective), and an expired export is refused (410) and
its file removed from disk.
