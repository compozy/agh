---
status: resolved
file: packages/ui/src/components/collapsible.tsx
line: 15
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4135497854,nitpick_hash:22407d0ca1b3
review_hash: 22407d0ca1b3
source_review_id: "4135497854"
source_review_submitted_at: "2026-04-19T05:12:06Z"
---

# Issue 011: Export order alphabetized.
## Review Comment

The export order was changed to alphabetical (`CollapsibleContent` before `CollapsibleTrigger`). While this doesn't affect functionality, consider whether alphabetical ordering is the preferred convention for exports across the component library.

## Triage

- Decision: `invalid`
- Reasoning: This is a style preference about export ordering, not a correctness, accessibility, or maintainability defect that requires change within the scoped remediation batch.
- Notes: Leaving the current order intact avoids unrelated churn and stays aligned with the review-fix scope.

## Resolution

- Marked invalid after triage; no code changes were required for this issue.
- Batch verification still completed successfully with `make verify`.
