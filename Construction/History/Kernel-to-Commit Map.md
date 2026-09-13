# Kernel-to-Commit Map

**Produced by:** Kernel 99, 2026-09-12, per spec §18. Per the spec's own instruction: "A kernel may have one commit, many commits, shared commits, or no uniquely attributable commit. Record honestly. Do not rewrite Git history solely to create clean kernel boundaries."

**Honest limitation:** most of Victory's 100 kernels do not have a clean, uniquely-attributable commit range — many kernels' work is interleaved with adjacent fixes, documentation updates, and follow-on corrections in the same commit sequence, and the project's own primary record-of-truth for "what a kernel built" is its reportback, not its commit list (see the evidence hierarchy in spec §4, which ranks reportbacks above raw commits). This map records only what was found with high confidence; it is not a substitute for `Kernel Index.md`.

## Kernels with a confirmed specific commit citation (found directly in a reportback)

| Kernel | Commit(s) | Source |
|---|---|---|
| 59 | `f324901` | `kernel-59-reportback.md` |
| 70 | `920aeb7` | `kernel-70-reportback.md` |
| 70A | `2611840` | `kernel-70a-reportback.md` |
| 71 (Phase A) | `bef0a69` | `kernel-71-reportback.md` |
| 34 | `a2d77a5` | `operator-log.md` cross-reference |
| 35 | `adce59d` | migration + commit cross-reference |
| 36 | `adce59d`, `9fb08e4` | migration + commit cross-reference |
| 37 | `4a09abd` | migration + commit cross-reference |
| 38 | `dbae39d` | migration + commit cross-reference |
| 39 | `05c6894`, `1d9079d`, `89af9ba` | migration + commit cross-reference |

## Kernels 93-100 (this session's own work, exact commits known)

| Kernel | Commits |
|---|---|
| 94 (closure documentation) | `8d849a8` |
| 96/97 close-out and merges | `689afc6`, `b804d57`, `d52cdd9`, `a55a727`, `69faa13` (and windows-native merge commits) |
| 98 close-out | `df8a106` |

## Earliest repository history

| Commit | Subject | Significance |
|---|---|---|
| `a1ff569` | Add construction docs and kernel specs | Earliest construction-intent commit |
| `428c181` | Add system dictionary and initial DB migration | Same era |
| `c69fcd6` | Built Core - Implement initial backend setup with Docker, database migrations, and API endpoints | First real backend implementation commit |
| `684aa68` | Refactor backend: update Go version, add new dependencies, enhance WebSocket handling | Early WebSocket foundation |
| `ae8f7b5` | Add initial kernel specifications and role normalization for user joining | First explicit "kernel" language in a commit message |

## Everything else

Not individually mapped — commit-level archaeology for the remaining ~85 kernels was judged disproportionate to its value given that every one of them already has a reportback, spec, or operator-log entry (see `Kernel Index.md`) serving as stronger, more legible evidence than a bare commit hash. A future agent needing a specific kernel's commit range should start from its reportback's own date and cross-reference `git log --since=<date> --until=<date>` rather than expecting a pre-built table here.
