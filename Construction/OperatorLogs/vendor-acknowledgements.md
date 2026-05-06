# Vendor Acknowledgements

## Historical Note
This file is now a historical snapshot.

Use [Construction/vendor-acknowledgements.md](/opt/victory/Construction/vendor-acknowledgements.md) for the current dependency list.

## Purpose

This document records all third-party software, libraries, and infrastructure components used in the system.

This is maintained for:
- transparency
- operational awareness
- future licensing review
- security auditing

Only **actual dependencies in use** are listed here.

---

## Infrastructure

### Docker
- **Type:** Container runtime
- **Use:** Service orchestration and environment isolation
- **Scope:** All services (backend, database)
- **Notes:** Enables reproducible environments and controlled deployments

---

### Docker Compose
- **Type:** Service orchestration tool
- **Use:** Defines multi-container application stack
- **Scope:** Local development and VPS deployment
- **Notes:** Used to manage backend + database lifecycle

---

### PostgreSQL 16 (Alpine)
- **Type:** Relational database
- **Use:** Primary data store
- **Scope:** All persistent system data
- **Notes:**
  - Uses `pgcrypto` extension for secure random UUID generation
  - Enforces constraints, relationships, and transactional integrity

---

## Backend Runtime

### Go (Golang)
- **Type:** Programming language / runtime
- **Use:** Backend application server
- **Scope:** Entire API layer
- **Notes:**
  - Chosen for simplicity, performance, and static compilation
  - Minimal external dependency footprint

---

## Go Libraries

### github.com/jackc/pgx/v5
- **Type:** PostgreSQL driver and connection pool
- **Use:** Database access layer
- **Scope:** All DB queries and transactions
- **Notes:**
  - Uses `pgxpool` for connection management
  - Supports prepared statements and context-aware queries

---

### golang.org/x/crypto/argon2
- **Type:** Cryptographic library
- **Use:** Password hashing (Argon2id)
- **Scope:** Authentication system
- **Notes:**
  - Memory-hard hashing function
  - Resistant to GPU and ASIC cracking
  - Industry-standard for password storage

---

### golang.org/x/sys (indirect)
- **Type:** Low-level system utilities
- **Use:** CPU feature detection for cryptographic operations
- **Scope:** Dependency of `argon2`
- **Notes:**
  - Not directly used in application code
  - Required for optimized crypto performance

---

## Database Extensions

### pgcrypto
- **Type:** PostgreSQL extension
- **Use:** Cryptographic functions
- **Scope:** UUID generation (`gen_random_uuid`)
- **Notes:**
  - Used for primary keys and secure identifiers
  - Avoids predictable ID sequences

---

## Security-Related Dependencies Summary

| Component | Purpose |
|----------|--------|
| Argon2 (golang.org/x/crypto) | Password hashing |
| SHA-256 (stdlib) | Token hashing |
| pgcrypto | UUID generation |
| pgx | Secure DB access |

---

## Current Philosophy on Dependencies

- Prefer **minimal external dependencies**
- Use **standard library where possible**
- Add libraries only when:
  - security requires it
  - complexity would otherwise increase risk
- Avoid large frameworks

---

## Not Yet Introduced (Planned)

These are expected future additions but are **not yet in use**:

- Image processing library (for workshop asset pipeline)
- Rate limiting middleware
- Structured logging framework
- Metrics/observability tooling

These will be added only when required and documented here.

---

## Licensing Note

All listed dependencies should be reviewed for:
- permissive licensing (MIT, BSD, Apache preferred)
- compatibility with project distribution goals

No license conflicts identified at current stage.

---

## Status

This list reflects the system at:

**Kernel 2 — Identity, Auth, Invites, Access Foundation**

It should be updated whenever:
- a new library is added
- infrastructure changes
- security-relevant components are introduced

fire.jpg = tobias-rademacher-wnF27F85ZKw-unsplash
