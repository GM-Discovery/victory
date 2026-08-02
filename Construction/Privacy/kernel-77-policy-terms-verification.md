# Kernel 77 — Privacy Policy & Terms of Service Verification

**Date:** 2026-08-01.
**Scope:** Kernel 77 §1.6, §10 — locate, verify reachability, check factual accuracy, correct
simple technical mismatches, flag anything requiring Grant's own wording decision. Not a
mandate to redraft the documents.

---

## 1. Location

| Document | Route | Source file | Actual content |
|---|---|---|---|
| Privacy Policy | `/legal/privacy/` | `frontend/legal/privacy/index.html` | iframes a published Google Doc |
| Terms of Service | `/legal/terms/` | `frontend/legal/terms/index.html` | iframes a published Google Doc |

Both pages are thin static wrappers; the actual legal text lives in Google Docs Grant owns and
edits directly, published via Google's "publish to web" feature. This repository does not and
should not contain the legal text itself.

**Reachability:** both return `200` unauthenticated (confirmed via `curl`). **Not linked from
anywhere in the app** as of Kernel 76/77 start — no footer, no signup page, no login page
referenced either document. This was a real gap; see §4.

## 2. Factual verification and corrections made

Effective date on both documents: **05/19/2026**, matching Grant's recollection exactly.

### Age boundary — corrected during this kernel

**Before:** Privacy Policy stated *"Victory is not intended for children under 13... Users
between 13 and 17 should use Victory only with permission from a parent or legal guardian."*
This is standard COPPA-style language, but contradicted Kernel 77's locked product decision
(§1.7): Victory is 17+, with no parental-consent mechanism of any kind, and Victory explicitly
must not imply one exists.

**After** (Grant's edit, verified in this session): Section 12, "Children's Privacy," now
reads: *"Victory is not intended for children under 17. Users under 17 may not create accounts
or use Victory. If we learn that we have collected personal information from a child under 17,
we will take reasonable steps to delete that information, including the account."* This
resolves the contradiction and, usefully, ties directly into the account-deletion capability
this kernel built.

### Contact method — corrected during this kernel

**Before:** Privacy Policy read *"To make a privacy request, contact: [EMAIL]"* — a literal
unfilled template placeholder. Terms of Service had no contact method anywhere.

**After:** Grant created `privacy@amurray.family` (a real mailbox on his existing Migadu-hosted
domain — not the `ops.amurray.family` subdomain used for transactional mail, which has no
inbound mail hosting; see the DNS note below) and reported adding it to the Privacy Policy's
contact line. **Operator action still open:** confirm the Terms of Service also gets a contact
line — it had none before or after this kernel's edits, and Kernel 77 did not draft one.

### DNS finding (informational, not a document defect)

While checking the intended `privacy@ops.amurray.family` address, its MX records were found to
point at Brevo's bounce-handling infrastructure (`brevosend.com`), not a real inbox — mail sent
there would not have reached anyone. The root domain `amurray.family` already has real mail
hosting via Migadu, which is why the contact address landed there instead. Not a policy-text
issue, but worth remembering if `ops.*` is ever used as a contact address again.

## 3. Not yet reflected in the documents (operator action required)

Kernel 77 shipped three capabilities the documents don't mention yet. Per §10.2, this kernel
corrects factual mismatches it caused but does not draft new legal language; these are flagged
for Grant's own wording:

1. **Account deletion.** The Privacy Policy currently only says data may be retained in
   "backups, logs, archives, security records, or append-only action history" and that users
   may "stop using the service" — it doesn't mention that self-service account deletion now
   exists, what it does (private data deleted, shared history anonymized), or how to request it.
2. **Data export.** Not mentioned at all; no portability language exists in either document.
3. **Backup storage location.** The Privacy Policy says Victory "is operated from the United
   States" and data "may be processed and stored in the United States or other locations where
   our service providers operate" — broad enough to technically cover an encrypted Google Drive
   backup, but doesn't specifically disclose it. Whether this needs more specific language is
   Grant's call, not a factual error as currently worded.
4. **Self-service password recovery via email.** Neither document currently addresses this
   (previously recovery was purely operator-mediated); the Terms/Policy language about
   authentication doesn't need to change, but could mention it exists.

## 4. Technical (non-legal-text) fix made in this kernel

Since the legal wording is Grant's to write but the *linking* is ordinary app code: this kernel
added links to `/legal/privacy/` and `/legal/terms/` on the account page (`/account/`) and the
login page (`/login/`), so the documents are now discoverable from two real, authenticated-and-
unauthenticated touchpoints rather than only reachable by a direct URL.

## 5. Age language elsewhere in the product

Per §10.3, no other public page was found suggesting the service targets children; the
correction in §2 above was the only instance found of language actually contradicting the 17+
boundary. Kernel 77 did not need to touch signup or account-creation copy beyond the Privacy
Policy itself, since password signup remains closed (Kernel 76) and Discord account creation
carries no age-specific copy of its own on Victory's side.
