---
status: resolved
file: internal/testutil/e2e/config_seed_test.go
line: 85
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4172207861,nitpick_hash:61160659c827
review_hash: 61160659c827
source_review_id: "4172207861"
source_review_submitted_at: "2026-04-24T17:07:23Z"
---

# Issue 016: Please convert this new test to t.Run("Should...") style.
## Review Comment

Coverage is useful; only the test-case structure is out of line with the suite’s required pattern.

As per coding guidelines "Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests" and "MUST use t.Run(\"Should...\") pattern for ALL test cases".

## Triage

- Decision: `VALID`
- Notes:
  - `TestSeedConfigPersistsSessionSupervisionOverlay` runs the scenario directly in the top-level test body.
  - The fix is to wrap the scenario in a `Should...` subtest and keep the independent subtest parallel.
