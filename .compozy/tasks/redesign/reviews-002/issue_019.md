---
status: resolved
file: packages/ui/src/components/dropdown-menu.test.tsx
line: 56
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4135497854,nitpick_hash:3e48c6076221
review_hash: 3e48c6076221
source_review_id: "4135497854"
source_review_submitted_at: "2026-04-19T05:12:06Z"
---

# Issue 019: Tighten mock assertions to catch duplicate/unexpected event emissions.
## Review Comment

`toHaveBeenCalledWith(...)` can still pass if callbacks fire multiple times unexpectedly. Consider asserting call counts alongside payload checks.

## Triage

- Decision: `valid`
- Reasoning: The dropdown callback tests assert payloads but not invocation counts, so duplicate or unexpected extra emissions could still pass unnoticed.
- Root cause: The tests verify arguments without constraining the number of calls.
- Fix plan: Add call-count assertions alongside the existing payload checks.

## Resolution

- Tightened the dropdown callback tests in `packages/ui/src/components/dropdown-menu.test.tsx` with explicit call-count assertions.
- Verified with `make verify` after all batch changes.
