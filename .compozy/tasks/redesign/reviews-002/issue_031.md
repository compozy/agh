---
status: resolved
file: packages/ui/src/components/section.tsx
line: 23
severity: minor
author: coderabbitai[bot]
provider_ref: review:4135497854,nitpick_hash:cdefa16abdfb
review_hash: cdefa16abdfb
source_review_id: "4135497854"
source_review_submitted_at: "2026-04-19T05:12:06Z"
---

# Issue 031: Header visibility check treats null/false as present content.
## Review Comment

Line 23 can render an empty header when consumers pass `label={condition && ...}` (false) or `null`. That creates unintended chrome (border + spacing).

## Triage

- Decision: `valid`
- Notes:
  - `Section` uses `!== undefined` to decide whether to render header chrome, so `label={false}` or `label={null}` still renders an empty header row and spacing.
  - Fix by treating `null` and `false` as absent content while preserving legitimate values like `0`.
  - Regression coverage requires touching adjacent existing test file `packages/ui/src/components/section.test.tsx`, which is outside the listed batch code files but is the colocated test surface for this component.
