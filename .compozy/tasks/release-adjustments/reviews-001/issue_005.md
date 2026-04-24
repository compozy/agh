---
status: resolved
file: internal/api/core/coverage_helpers_test.go
line: 471
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4172207861,nitpick_hash:17d6b9529329
review_hash: 17d6b9529329
source_review_id: "4172207861"
source_review_submitted_at: "2026-04-24T17:07:23Z"
---

# Issue 005: Wrap this new test case in t.Run("Should...") to match test conventions.
## Review Comment

The assertions are good, but this new test path should follow the repo’s required subtest naming pattern.

As per coding guidelines "Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests" and "MUST use t.Run(\"Should...\") pattern for ALL test cases".

## Triage

- Decision: `VALID`
- Notes:
  - `TestObserveHealthPayloadIncludesRuntimeActivity` currently runs directly in the top-level test body.
  - The fix is to wrap the scenario in a `Should...` subtest and keep the independent subtest parallel.
