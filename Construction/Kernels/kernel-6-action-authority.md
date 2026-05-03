# Kernel 6 Action Authority Notes

## What is enforced

- `act/reveal_element` and `act/hide_element` are validated server-side before they are written to `actions`.
- The client still shows/hides buttons for convenience, but the server is the source of truth.
- Current scope is the Cave fire only. The seam is centralized in `backend/internal/actions/authority.go`.

## Where `actors_can_reveal` lives

- Stored on `venues.config` as JSONB.
- The Cave seed sets `actors_can_reveal` to `false` by default.
- The authority check reads the flag from the database on each action attempt.

## How to toggle it in dev

Use SQL against the local database:

```sql
UPDATE venues
SET config = jsonb_set(
  COALESCE(config, '{}'::jsonb),
  '{actors_can_reveal}',
  'true'::jsonb,
  TRUE
)
WHERE slug = 'the-cave';
```

To turn it back off, replace `'true'::jsonb` with `'false'::jsonb`.

## Test commands

- Allowed Producer/Director path: join The Cave as a producer or director, then click Reveal/Hide on the first fire.
- Denied audience path: open the Cave without a privileged role, then send `act/reveal_element` or `act/hide_element` from devtools or websocket payloads.
- Denied cast path with policy off: set `actors_can_reveal` to `false`, then try the same action as cast.
- Allowed cast path with policy on: set `actors_can_reveal` to `true`, then retry as cast.

Example database check:

```sql
SELECT id, moment_id, actor_id, type, target, payload
FROM actions
WHERE session_id = '<cave-session-id>'
ORDER BY moment_id DESC;
```

Denied actions should not appear in this table.

## Rebuild / restart notes

- No backend environment variable is required for this policy.
- No backend restart is needed for a pure DB toggle.
- A backend rebuild/restart is only needed when code changes are deployed.

## Design seam for future grants

The final decision is currently:

- authenticated user
- joined session participant
- cave target check
- role default
- `actors_can_reveal` policy

Future grants, relationships, and licenses should attach inside `backend/internal/actions/authority.go` without changing the client contract.
