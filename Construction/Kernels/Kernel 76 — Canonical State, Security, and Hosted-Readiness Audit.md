# Kernel 76 — Canonical State, Security, and Hosted-Readiness Audit

**Status:** COMPLETE — PASS
**Type:** Audit with bounded critical repair
**Executed:** 2026-07-30 / 2026-07-31
**Baseline commit:** `97c7169` on `main`
**Achieved readiness:** Level 3 — Invited strangers
**Required target:** Level 4 — Private paying clients (blocked by K76-M04, K76-M05, backup)

---

## 1. What this kernel was asked to determine

Whether Victory can responsibly accept strangers and their private data — established by
inspecting the running system and reproducing real behaviour, not by reviewing documents.

The governing specification is the Kernel 76 brief supplied by the operator, reproduced in
intent here. Planning authority: `Victory_Canonical_Roadmap_v2.md`. Implementation authority:
the repository, database, and deployment as they actually were.

---

## 2. Product decisions taken

Two decisions were referred to the operator before work began, because §13.6 permits asking
only product questions:

| Question | Decision |
|---|---|
| Disposition of current test data | **Wipe early** — rebuild from migrations |
| Password credentials after Discord becomes the account path | **Discord only**, build a recovery mechanism |

Both were carried out. A third question — execution style — was answered "run straight
through", so the kernel proceeded autonomously with the destructive step gated on proven
recovery.

---

## 3. Method

1. Preflight: environment, runtime mode, kernel numbering (§4).
2. Rollback point: 5.5 MB database dump retained before any change (§2.4).
3. Route enumeration from `main.go` — 221 registrations — then live negative testing with
   four identities against the public host.
4. WebSocket testing with a real `gorilla/websocket` client.
5. Bounded repair of every Critical and High finding, with regression tests.
6. Break-glass recovery tool built **and proven end-to-end before** the destructive step.
7. Database rebuild from migrations; operator access restored and verified.
8. Artifacts and reportback.

The ordering in steps 5–7 is the anti-lockout contract (§2.1): the replacement path was proven
from a clean context before anything was removed.

---

## 4. Outcome

**Seven findings repaired and verified live:**

| ID | Severity | Finding |
|---|---|---|
| K76-C01 | Critical | Raw password-reset tokens written to the production log |
| K76-H01 | High | Open public registration granting Location membership |
| K76-H02 | High | Third Place commons served real names to any account |
| K76-H03 | High | Committed default database password in production use |
| K76-M01 | Medium | Production list disclosed to every Location member |
| K76-M02 | Medium | Unauthenticated infrastructure status endpoint |
| K76-M03 | Medium | No transport limits — bodies, headers, idle, WebSocket frames |

**Two blockers deferred to Kernel 77:** account deletion (K76-M04) and data export (K76-M05),
plus the backup posture.

**Built:** `backend/cmd/victory-recover`, a break-glass recovery tool with `whoami`,
`bootstrap`, `claim`, `grant`, `revoke`, and `recover`; and `frontend/login/reset.html`, the
page that redeems a recovery link.

---

## 5. Artifacts

| Artifact | Path |
|---|---|
| Finding ledger | `Construction/Domains/Security/kernel-76-findings.md` |
| Current-state inventory | `Construction/Domains/Security/kernel-76-current-state-inventory.md` |
| Architecture and exposure map | `Construction/Domains/Security/kernel-76-architecture-exposure-map.md` |
| HTTP route matrix | `Construction/Domains/Security/kernel-76-route-matrix.md` |
| WebSocket matrix | `Construction/Domains/Security/kernel-76-websocket-matrix.md` |
| Data classification, deletion, export, Markdown pre-audit | `Construction/Domains/Security/kernel-76-data-classification.md` |
| Recovery and anti-lockout runbook | `Construction/Domains/Operations/victory-account-recovery-runbook.md` |
| Backup and restore assessment | `Construction/Domains/Operations/victory-backup-restore-assessment.md` |
| Updated security notes | `Construction/OperatorLogs/Security-notes.md` |
| Kernel 77 proposed scope | `Construction/Kernels/Kernel 77 — Proposed Hosted-User Stabilization Scope.md` |
| Reportback | `Construction/OperatorLogs/kernel-76-reportback.md` |

---

## 6. Why PASS, and why not Level 4

Kernel 76 §11 permits PASS when the audit covers the required surfaces, artifacts exist,
critical bounded repairs are complete, anti-lockout proof is complete, readiness is honestly
classified, and Kernel 77 scope is explicit. All are satisfied.

PASS does not mean hosted readiness, and this kernel does not claim it. Level 4 requires a
usable account-deletion and export path and a credible backup implementation. Deletion is not
merely missing — it is structurally blocked by a foreign key, proven by attempting it. That is
a capability gap, not an exposure, which is why the readiness classification is Level 3 rather
than a failure.

---

## 7. Deviation from the specification

The brief anticipated that Kernel 76 might disable password login entirely. It did not:
signup is closed but login is retained, because §2.1 forbids removing a working access path
before the replacement is proven from a fresh browser, and password login is also the redemption
path for break-glass recovery tokens. Removing it would have made Victory's recovery depend
entirely on Discord, which §5.4 explicitly forbids.
