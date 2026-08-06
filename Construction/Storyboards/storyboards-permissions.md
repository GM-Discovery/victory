# Storyboards Permissions Matrix (Kernel 80)

Implementation: `backend/internal/storyboards/authority.go`. Modeled on
`ewrite/authority.go` and `showruns/authority.go`'s Operator-short-circuit
shape, but with **no location-role floor**: Storyboard authority is
purely per-board (ownership or an explicit `storyboard_grants` row),
never location-scoped. Client-supplied role claims are never authority —
every check re-derives the tier from the database via
`resolveViewerTier`, which is the one function every other authority and
projection function calls. See `TestServerResolvedTierIgnoresClientClaims`
and `TestRoleForgeryOverHTTPRejected` for the enforced proof of this.

## Tiers

`owner` and `operator` are not `location_role` values; every other tier
name is a direct enum value. Resolution order in `resolveViewerTier`:
Operator (env-based superuser, `access.IsOperatorUser`) → board owner
(`board.OwnerUserID == userID`) → `storyboard_grants` lookup → `TierNone`
(no access at all).

## Matrix

| Capability | Audience | Cast | Crew | Director/Producer | Owner | Operator |
|---|---|---|---|---|---|---|
| View board / cards (non-hidden) | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| See hidden-from-audience cards | — | — | ✓ | ✓ | ✓ | ✓ |
| Create/edit/move/reorder/delete cards | — | — | ✓ | ✓ | ✓ | ✓ |
| Edit an **unlocked** band's label | — | — | ✓ | ✓ | ✓ | ✓ |
| Set a card's hidden-from-audience flag | — | — | — | ✓ | ✓ | ✓ |
| Add/remove/reorder columns, rows, bands | — | — | — | ✓ | ✓ | ✓ |
| Move a row between bands | — | — | — | ✓ | ✓ | ✓ |
| Lock/unlock bands and cards | — | — | — | ✓ | ✓ | ✓ |
| Edit board metadata (title/description) | — | — | — | ✓ | ✓ | ✓ |
| Export board JSON | — | — | — | ✓ | ✓ | ✓ |
| Grant/revoke access, choose granted role | — | — | — | — | ✓ | ✓ |
| Archive/unarchive/delete the board | — | — | — | — | ✓ | ✓ |

Crew is deliberately narrower than Director+ on two points the spec
calls out explicitly: Crew never sees or sets `hidden_from_audience`
(spec 1.9's role table), and Crew never exports (spec 5.3-5.5 list export
only under Director+/Owner).

## Lock behavior

A locked card or a card whose row's band is locked additionally requires
`CanEditStructure` (Director+/owner/Operator) to mutate — lock only ever
restricts Crew, never Director+/owner, matching how `UpdateBandLabel`
treats band locks. `cards.go`'s `canMutateCard` is the single place this
is decided; `SetCardLock`/`SetBandLock` themselves require
`CanEditStructure` unconditionally (only Director+ may lock/unlock at
all, spec 5.4).

## Owner vs. grant — why ownership is never a grant row

See `storyboards-domain-model.md`'s "Ownership vs. grants" section for
the full rationale. In authority terms: `CanManageSharing` and
`CanArchiveOrDeleteBoard` both reduce to "owner or Operator" — there is
no `granted_role` value that reaches this tier, by design.

## Sharing is deliberate, never implicit

`grants.go`'s `AddGrant` looks up the target user by **handle**
(`SELECT id FROM users WHERE handle = $1`), mirroring `ewrite/editors.go`'s
`AddEditor` — there is no generic user-search endpoint in this repo,
handle-paste is the established mechanism. A My People relationship
(`playerrelationships`) never itself grants anything; `TestMyPeopleRelationshipAloneGrantsNothing`
proves a friend with a relationship but no explicit grant has zero board
access. The frontend's sharing panel shows My People names as
informational suggestions only (see `storyboards-ui-contract.md`).

## Revocation

`RemoveGrant` deletes the `storyboard_grants` row immediately;
`TestRevokeGrantRemovesAccessImmediately` proves the next `CanViewBoard`
check fails right after. Over the WebSocket, revocation is enforced on
the next mutation attempt (HTTP 403, since every mutation re-derives the
tier) and the next `watch_board`/reconnect (rejected) — an already-open
watch is not proactively kicked mid-session; see
`storyboards-live-events.md` for the recorded scope boundary.
