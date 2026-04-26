---
status: resolved
file: web/src/components/app-sidebar.tsx
line: 138
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177411700,nitpick_hash:d83202abe6b3
review_hash: d83202abe6b3
source_review_id: "4177411700"
source_review_submitted_at: "2026-04-26T20:24:27Z"
---

# Issue 026: Consider extracting shared active-nav styles to avoid drift.
## Review Comment

`NavItem` and `FooterSlot` now duplicate the same active container and indicator classes. A small local constant/helper would reduce maintenance risk.

Also applies to: 484-494

## Triage

- Decision: `valid`
- Notes: `NavItem` and the settings footer link duplicate the same active row and active indicator class strings. A local constant for the shared row, active row, and indicator classes removes the drift risk without changing runtime behavior.
