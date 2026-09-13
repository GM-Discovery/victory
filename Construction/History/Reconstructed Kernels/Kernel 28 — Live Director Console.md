# Kernel 28 — Live Director Console

**Provenance:** RECONSTRUCTED — PARTIAL
**Original date/window:** Unrecoverable precisely; immediately follows Kernel 27, same pre-Kernel-32 window
**Implementation status:** IMPLEMENTED — status not recorded (no surviving spec or reportback), but unlike Kernel 27, its output is directly traceable in live code today
**Evidence sources:** `Construction/kernel-maker-field-guide.md:48` ("Kernel 28 added the live Director Console for current-showing control"); `backend/internal/network/director_console.go:355` — a live code comment reading `StartShowingNote: "Kernel 28 defers new-showing startup."`

## Reconstructed purpose

Add a live Director Console surface for controlling the *currently running* Showing (as opposed to Kernel 27's review of *closed* Showings) — explicitly scoped to not also handle starting a new Showing, per the surviving code comment.

## What evidence proves was built

`director_console.go` exists today as a substantial, actively-maintained file (subject of real authority fixes in Kernel 97 this same session — see `Construction/Identity/Canonical Role and Authority Resolution.md` §2, §6). The `StartShowingNote` field and its literal string citing "Kernel 28" is direct, unambiguous evidence that this kernel's original scope boundary (control current Showing, defer new-Showing startup) is still legible in the codebase 70+ kernels later.

## Files/systems affected

`backend/internal/network/director_console.go` (created here in some early form; heavily amended by many later kernels — it is one of the files Kernel 97 migrated off the unscoped `access.CurrentLocationRole` this same session).

## Known deviations / later corrections

The file has clearly grown far beyond its Kernel 28 origin (Director Console access authority was itself the subject of a real bug fix in tonight's Kernel 97 pass). The original "defers new-showing startup" boundary appears to have persisted as a real, still-honored design decision rather than being quietly reversed — no evidence was found of new-Showing-startup being added to this console later, though that absence-of-evidence was not exhaustively confirmed.

## What this kernel handed to the next kernel

The Director Console surface itself, which every subsequent Director-facing kernel (up through Kernel 97's authority reconciliation this session) has built on rather than replaced.

## Confidence / unresolved gaps

PARTIAL — the kernel's purpose and a literal surviving fragment of its original scope boundary are confirmed with high confidence via the live code comment, but no spec, reportback, or commit could be attributed to its original introduction, so its full original scope and any contemporaneous acceptance criteria are unrecoverable.
