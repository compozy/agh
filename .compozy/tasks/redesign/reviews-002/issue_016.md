---
status: resolved
file: packages/ui/src/components/dialog.tsx
line: 30
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4135497854,nitpick_hash:67ef969f1f4b
review_hash: 67ef969f1f4b
source_review_id: "4135497854"
source_review_submitted_at: "2026-04-19T05:12:06Z"
---

# Issue 016: Consider extracting shared “overlay root + motion context” wrapper logic.
## Review Comment

The controlled/uncontrolled open handling + `actionsRef` + provider setup is very similar to the new `Sheet` (and likely `Popover`) implementation. A shared helper/hook would reduce drift and make future fixes safer.

## Triage

- Decision: `invalid`
- Reasoning: This is a refactor suggestion about extracting shared infrastructure with other components, not a defect in `Dialog` behavior itself. The scoped batch is for concrete review fixes, not cross-component abstraction work.
- Notes: No behavior or test gap in the scoped files requires this extraction to satisfy the current review round.

## Resolution

- Marked invalid after triage; no code changes were required for this issue.
- Batch verification still completed successfully with `make verify`.
