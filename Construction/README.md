# Construction — Index

This folder is Victory's build history and living design record: every kernel spec that shaped the product, the reportbacks proving what was actually built, and the reference documentation that keeps a future agent or operator from having to rediscover it all. Reorganized for navigability by Kernel 99's own follow-up pass (2026-09-13) — see `History/Kernel Index.md` if a path below doesn't match what an older doc expects.

## Where things live

- **`Kernels/`** — every kernel spec (the actual work orders), numbered 1 through 100. This is the spine of Victory's construction history. Read a kernel's own spec before touching the area it covers.
- **`OperatorLogs/`** — every reportback and closure ledger proving what a kernel actually built, plus the chronological `operator-log.md` and durable `operator-notes.md`. This is the evidence layer beneath `Kernels/`.
- **`History/`** — the Kernel 99 archive: `Kernel Index.md` (start here for "what happened with kernel N"), `Kernel Lineage.md` (the narrative version), `Architecture Milestones.md`, `Superseded Doctrine Map.md` ("do not resurrect" list), `Migration Chronology.md`, `Kernel Debt Ledger.md`, `Kernel-to-Feature Map.md`, `Kernel-to-Commit Map.md`, `Historical Uncertainty Report.md`, `Current Architecture Manifest.md`, and the machine-readable `kernels.json`. Also `Reconstructed Kernels/` (spec-less kernels rebuilt from other evidence) and `Archived Original Docs/`.
- **`Canon/`** — living product/world-state truth: `current-state.md` (current routes/venues/domain state — read this before assuming something exists), `roadmap.md` and `roadmaps/` (priority), `Dictionary.txt` (in-world terminology), `character-workbook.md`, `Everything Implemented.md`, `Recovered Work Reconciliation.md`.
- **`Process/`** — how construction itself works: `kernel-maker-field-guide.md` (read this first if you're a new agent about to work a kernel), `reportbacktemplate.txt`, `Construction — Contracts.txt`, `vendor-acknowledgements.md`, plus `workflow/` (dev workflow), `templates/`, and `deployment/` (fresh-install proof docs).
- **`Domains/`** — per-subsystem design notes and contracts, one folder per area: `Chat/`, `Design/`, `Dice/`, `eWrite/`, `Identity/` (includes the Kernel 97 canonical role/authority resolver doc), `Network/`, `Operations/` (includes the break-glass recovery runbook), `Privacy/`, `Scenes/`, `Security/`, `Shows/`, `Socio/`, `Stage/`, `Storyboards/`, `Venues/`.

## Quick answers

- **"What did Kernel N actually build?"** → `History/Kernel Index.md`, then the cited reportback in `OperatorLogs/`.
- **"Is this old assumption still true?"** → `History/Superseded Doctrine Map.md`.
- **"What's still open/broken?"** → `History/Kernel Debt Ledger.md`.
- **"Where's the current authority model?"** → `Domains/Identity/Canonical Role and Authority Resolution.md`.
- **"How do I work a new kernel?"** → `Process/kernel-maker-field-guide.md`.
- **"What actually exists right now?"** → `Canon/current-state.md`.

User-facing product/operator documentation (install, Discord, backup, glossary, role guides) lives outside this folder, in `Docs/` at the repository root — this folder is construction history and internal design reference, not the consumer-facing manual.
