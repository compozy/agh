---
status: resolved
file: packages/ui/src/components/pills.tsx
line: 131
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4135497854,nitpick_hash:b8bd98c03f7b
review_hash: b8bd98c03f7b
source_review_id: "4135497854"
source_review_submitted_at: "2026-04-19T05:12:06Z"
---

# Issue 027: Avoid redundant onChange calls for already-active items.
## Review Comment

Even when value is unchanged, Line 131 still emits `onChange`, which can trigger unnecessary parent work/side effects.

## Triage

- Decision: `valid`
- Notes:
  - Clicking an already-active item currently calls `onChange` with the same value, which turns a selection-change callback into a generic click callback and can trigger redundant parent work or duplicate side effects.
  - Fix by treating clicks on the active item as a no-op.
  - Regression coverage requires touching adjacent existing test file `packages/ui/src/components/pills.test.tsx`, which is outside the listed batch code files but is the colocated test surface for this component.
