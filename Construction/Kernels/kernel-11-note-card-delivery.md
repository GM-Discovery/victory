# Kernel 11 Note Card Delivery System

## What is enforced

- Note cards are durable mailbox messages, not live chat.
- Senders must be authenticated and currently anchored in the Cave session.
- The server resolves both sender and recipient identity.
- Client-provided sender identity is ignored.

## Delivery model

- Note cards are stored in the shared `messages` table.
- Note cards use `message_type = note_card`.
- Note cards carry Cave context when available:
  - `venue_slug`
  - `session_id`
- Recipient mailboxes show note cards on refresh and after open.

## Recipient rules

- Preferred path:
  - send to a visible/current Cave participant by `to_user_id`
- Alternate path:
  - send to a specific session participant by `to_participant_id`
- Fallback:
  - deliver to the current director mailbox

## Limits

- Note card body is capped at 250 characters in the API.
- No attachments.
- No threading.
- No realtime delivery.

## Backend surfaces

- note card route: `POST /api/note-cards`
- mailbox routes:
  - `GET /api/messages`
  - `GET /api/messages/{id}`
  - `POST /api/messages`
- handler + message storage: `backend/internal/messages/messages.go`
- routes: `backend/cmd/victory/main.go`

## Frontend surfaces

- Cave sender form: `frontend/venues/the-cave/index.html`
- mailbox view: `frontend/mailbox/index.html`
- mailbox link: `frontend/venues/trailers/index.html`

## Presence-scoped recipient rule

- The Cave sender form is populated from the current presence roster.
- Audience-visible recipients are filtered to:
  - Director
  - Cast
  - Crew
- Other audience members are never shown as note-card recipients.

## Operator notes

- No new environment variables were added.
- No new dependencies were added.
- Backend restart is required after code changes.
- Test insert example for mailbox:

```sql
INSERT INTO messages (message_type, to_user_id, from_user_id, subject, body, venue_slug, session_id)
VALUES (
  'note_card',
  '<recipient-user-id>',
  NULL,
  'Stage hello',
  'This is a test note card.',
  'the-cave',
  NULL
);
```

- The server now bootstraps `messages.message_type`, `messages.venue_slug`, and `messages.session_id` if the table already exists.
