---
provider: coderabbit
pr: "105"
round: 5
round_created_at: 2026-05-06T02:28:33.373448Z
status: resolved
file: internal/network/helpers_test.go
line: 307
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4232600854,nitpick_hash:7d859d8a87a1
review_hash: 7d859d8a87a1
source_review_id: "4232600854"
source_review_submitted_at: "2026-05-06T01:29:19Z"
---

# Issue 010: Variable name blankDirect is misleading after KindDirect removal.
## Review Comment

This test uses `KindSay` but the variable is named `blankDirect`, which is confusing given `KindDirect` was removed from the valid kinds. Consider renaming to reflect what's actually being tested.

## Triage

- Decision: `valid`
- Notes:
  - The variable named `blankDirect` now decodes a `KindSay` body, so the identifier no longer describes what the test is asserting after `KindDirect` was removed from the valid-kind path.
  - This is a readability and maintenance issue in the test only.
  - Fix plan: rename the variable to reflect the real scenario, such as a second blank-say decode case, without changing the assertion semantics.

## Resolution

- Renamed the misleading local from `blankDirect` to `blankSayNewline` so it matches the actual `KindSay` case being tested.
- Verified with fresh full `make verify` (passed).
