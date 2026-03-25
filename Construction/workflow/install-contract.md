
---

# `Construction/workflow/install-contract.md`

```md
# Install Contract — VICTORY (Operator Standard)

## Purpose
Defines what “install from GitHub onto a new server” must achieve.

This is not optional. This is the standard for all future operators.

---

## Core Requirement

A fresh server must be able to:

1. run install
2. start system
3. produce a working cave
4. pass health checks
5. support a live session

No manual patching allowed.

---

## Minimum Install Steps (target)

Ideal form:

```bash
curl -fsSL <install.sh> | bash

or:

git clone ...
cd victory
bash install.sh
Required Outcomes

After install:

1. Services running
Postgres container running
Backend container running

Verify:

docker ps
2. Health endpoint
curl -s http://localhost:8081/health

Must return:

{"ok": true}
3. World snapshot
curl -s http://localhost:8081/api/world/the-cave

Must return:

location
lot
venue
session
fire element
4. Session exists
at least one live or rehearsal session for the-cave
5. Join works
POST /api/session/the-cave/join

Must:

create or update user
attach to session
6. WebSocket works
wscat -c ws://localhost:8081/ws/the-cave

Must:

connect
receive snapshot
respond to ping
7. Reaction works

Must:

accept react/emote
persist to DB
broadcast to observers
Required Components

Install must include:

Docker
docker
docker compose
Backend
built container image
Database
Postgres container
volume for persistence
Schema
migrations applied automatically
Seed
base world seeded:
location
lot
venue the-cave
fire element
session
Environment Configuration

Must be externalized:

.env should include:

DB credentials
ports
environment mode

Do not hardcode secrets in source.

Security Expectations

Minimum:

Postgres not publicly exposed
backend can be proxied (future)
no root-level credentials in app code
Update Model (future)

Install must support:

pull new version
restart containers
preserve DB data

Planned mechanism:

cron-based updater or operator-triggered update
Failure Conditions

Install is considered FAILED if:

manual DB setup required
missing session
missing fire element
WebSocket does not connect
reaction does not persist
system only works in dev mode
Non-Goals (for install phase)

Do NOT require:

UI polish
advanced permissions
scaling infrastructure
multi-venue orchestration

Install must prove:
the system lives, not that it is complete.

Operator Responsibility

Operator must:

provide server
run install
verify endpoints
ensure uptime

Operator does NOT:

manually edit DB
patch code per install
debug per-customer drift

Fix the repo, not individual installs.

Guiding Principle

If install fails:

do not fix the server
fix the repository