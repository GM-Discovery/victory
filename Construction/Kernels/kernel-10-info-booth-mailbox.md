# Kernel 10 Info Booth + Mailbox Foundation

## What is enforced

- `Info Booth` is a public map modal, not a full venue page.
- `Mailbox` is durable, authenticated-only message delivery.
- Mailbox is bound to the authenticated user on the server.
- Client identity claims are never trusted for message ownership.

## Mailbox model

- One inbox per user.
- Messages are stored in a dedicated `messages` table.
- Message fields:
  - `id`
  - `to_user_id`
  - `from_user_id` nullable
  - `subject`
  - `body`
  - `created_at`
  - `read`
- Message bodies are capped around 250 characters in the API.

## Access rules

- `Info Booth` is visible to everyone.
- `Mailbox` requires authentication.
- Users can only read their own inbox.
- `POST /api/messages` is operator/dev use only.

## Backend surfaces

- table bootstrap and handlers: `backend/internal/messages/messages.go`
- routes: `backend/cmd/victory/main.go`
- migration: `database/migrations/008_kernel10_info_booth_mailbox.sql`

## Frontend surfaces

- map modal: `frontend/index.html`
- map interaction: `frontend/app.js`
- mailbox page: `frontend/mailbox/index.html`

## Operator notes

- No new environment variables were added.
- No new dependencies were added.
- Backend restart is required after code changes.
- Test insert example:

```sql
INSERT INTO messages (to_user_id, from_user_id, subject, body)
VALUES (
  '<recipient-user-id>',
  NULL,
  'System message',
  'Mailbox foundation is live.'
);
```

- The live server also bootstraps the table on startup now.
