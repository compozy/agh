---
status: resolved
file: packages/ui/src/components/popover.test.tsx
line: 54
severity: minor
author: coderabbitai[bot]
provider_ref: review:4135497854,nitpick_hash:feee6d334255
review_hash: feee6d334255
source_review_id: "4135497854"
source_review_submitted_at: "2026-04-19T05:12:06Z"
---

# Issue 028: Ensure console.error is always restored in the throw-path test.
## Review Comment

Current mutation on Lines 55–64 can leak if the expectation fails before restoration. Wrap mutation in `try/finally`.

## Triage

- Decision: `valid`
- Notes:
  - The orphan-render throw test replaces `console.error` without `try/finally`, so a failed assertion would leak the stubbed console into later tests.
  - Fix by wrapping the mutation/restoration in `try/finally`.
