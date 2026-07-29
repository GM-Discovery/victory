# Delayed Features

## Mailbox pip for director invitations

Current state:
- Director invitations are created as show tickets and are visible in Audition Hall.
- The mailbox already supports unread message state and a visible count of unread messages.
- Invitations are not currently mirrored into the mailbox as messages, so the mailbox cannot reliably count them yet.

Why this is delayed:
- The map pip should reflect real unread mailbox items, not a separate ad hoc counter.
- To make director invitations show up in the mailbox and on the Info Booth badge, the backend needs a message-mirroring path when tickets are created or updated.

Suggested implementation:
- When a director creates an invitation ticket, also create a mailbox message for the invited user.
- When the ticket is accepted, declined, withdrawn, or withdrawn by the director, update or resolve the related mailbox message.
- Expose an unread message count for the signed-in user through the existing venue notification count path so the Info Booth can show a pip.

User-facing result:
- Director invitations remain visible in Audition Hall.
- The same invitations also appear in the mailbox.
- The Info Booth on the main map shows a small unread pip with the number of new mailbox items.

Status:
- Deferred until backend support is added.
