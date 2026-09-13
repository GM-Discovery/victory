# Historical Uncertainty Report

**Produced by:** Kernel 99, 2026-09-12, per spec §58. This is the honest accounting of what could not be fully recovered, after exhausting the evidence hierarchy (specs → reportbacks → commits → migrations → code comments → operator logs → adjacent-kernel references → current code). Nothing here was resolved by invention. See `Construction/History/Kernel Index.md` for the full per-kernel record this report qualifies.

---

## Full unknowns — no recoverable evidence

### Kernels 14, 17, 19, 20, 26

**What is known:** their numbers exist in the sequence (later kernels number contiguously around them); Kernel 25's own spec ("Current State Canon") implies routine sequential numbering was in effect through at least that point.

**What evidence is missing:** no spec file survives under any filename convention checked; no git commit (searched across all branches, by message and by content) mentions any of these numbers; no code comment anywhere in the current tree cites them; `operator-log.md` has no entries for them; the Kernel Maker Field Guide's own historical narrative jumps from Kernel 24 to Kernel 27 in prose, silently skipping 25 and 26 even though Kernel 25 has a surviving spec — suggesting the field guide's own author either didn't consider 26 a separate real kernel or simply didn't know about it either.

**Confidence:** none. These are marked UNKNOWN, not reconstructed.

**Does the gap matter to current maintenance?** No evidence found that any current, live code path depends on unrecovered work from this window — every downstream kernel's own citations trace to Kernels 16, 18, 21-25, and 27-30, never to 14/17/19/20/26 by name. The most likely explanations, in descending order of plausibility, are: (a) these numbers were reserved during informal early planning and never actually executed, (b) small work was done and folded silently into an adjacent kernel without a number change, or (c) evidence was lost during an early repository reorganization (the `780f312 Refactor: renamed file path parent folder kernels` commit is suggestive but not proof). No basis exists to choose between these.

---

## Partial reconstructions — some evidence, not enough for full confidence

### Kernel 27 — Closed-Showing Review Surface
Only one field-guide sentence attests to this kernel's existence and purpose. Reconstructed as PARTIAL. No code citation, migration, or test could be found to independently corroborate it. If a future agent finds the Director's Chair closed-Showing review feature in current code, that is corroborating (not certain) evidence this reconstruction is accurate.

### Kernel 28 — Live Director Console
Better-evidenced than 27 (a live code comment at `director_console.go:355` cites it by name, in addition to the field guide), but still short of the reportback-level evidence most other kernels have. Reconstructed as PARTIAL.

### Kernel 81A — Storyboard Structural Slugs
Real and live-deployed (migration 094, a dedicated test file, and a substantive `operator-log.md` entry all corroborate it), but has no filed spec and no standalone reportback — thinner documentation than every other kernel in its era. Not marked PARTIAL (the operator-log entry is detailed enough to support HIGH CONFIDENCE), but flagged here because a future agent searching for "the Kernel 81A spec" will not find one and should not conclude the kernel didn't happen.

### Kernels 88, 89, 90, 91, 92
Each has a detailed, self-consistent, contemporaneous reportback — but no committed original spec file exists in the repository for any of them (the operator-issued specs for this span were apparently never committed as separate documents). Reconstructed as HIGH CONFIDENCE on reportback strength, per the kernel's own evidence hierarchy (reportbacks rank above "current code only"), not PARTIAL — but the absence of the original spec itself is worth naming so a future agent doesn't assume one was lost rather than never committed.

---

## Confirmed, documented gaps that are NOT historical uncertainty (already explained by evidence)

These were investigated and resolved with a concrete explanation — listed here only so a future reader doesn't re-open them as new mysteries:

- **Migration `088` does not exist.** Confirmed harmless — the migration runner sorts by filename and tracks a checksum ledger, not contiguous numbers (see `Migration Chronology.md`).
- **Migration `093_fix_missing_permission_requests_table.sql` has no kernel citation.** Likely an out-of-band hotfix between Kernels 81 and 81A; no reportback claims it. Left unattributed rather than guessed.
- **`operator-log.md` has no entries for Kernels 75, 76, 77, or 77A** even though full reportbacks exist for all four — the reportbacks are the authoritative source for that span, not the log's absence.
- **Kernel 95 was never executed.** Its spec is real and complete (committed in the same batch as Kernels 96-100), but no reportback, ledger, or implementation evidence exists anywhere. This is not a historical-recovery gap — it is current, open, unbuilt scope, tracked in the Kernel Index as NOT EXECUTED rather than as an uncertainty.
- **"Kernel 50+", "51A/51B" are not independent kernel numbers.** They are same-arc hardening/follow-up passes documented within their base kernel's own reportback, folded into that kernel's row rather than given separate numbers.
- **Two Kernel 75 spec files exist.** One (`...aftercare-continuation-v0.1.md`) is self-marked SUPERSEDED in its own header; the other is the real, operator-issued, authoritative revision. Not a duplicate — a documented draft-to-revision pair, both preserved.

---

## What this means for the "all 1-100 represented" proof

Per spec §57 and §65: an honestly labeled UNKNOWN counts as representation; a missing number does not. All five true unknowns (14, 17, 19, 20, 26) appear explicitly in `Kernel Index.md` with the UNKNOWN provenance label and a one-line note on what was searched — they are represented, not hidden, and Kernel 99 does not call this a failure of the archive. It calls it an honest limit of what evidence survives.
