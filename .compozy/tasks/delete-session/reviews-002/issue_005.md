---
status: resolved
file: internal/store/globaldb/global_db_task_test.go
line: 245
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4167261241,nitpick_hash:58c2326585d3
review_hash: 58c2326585d3
source_review_id: "4167261241"
source_review_submitted_at: "2026-04-24T01:30:33Z"
---

# Issue 005: Adopt the required t.Run("Should...") pattern for this test case.
## Review Comment

This test is valuable, but it currently skips the repository-required subtest naming/structure convention.

As per coding guidelines, "MUST use t.Run("Should...") pattern for ALL test cases" and "Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests".

## Triage

- Decision: `valid`
- Notes:
  - `TestGlobalDBDeleteTaskMapsChildConstraintToValidationError` in `internal/store/globaldb/global_db_task_test.go` is written as a flat top-level test body instead of the required `t.Run("Should...")` structure used across this repo.
  - The underlying assertions are useful and should stay; only the test shape needs to change.
  - Planned fix: move the current body into a named subtest and keep the assertions intact.

## Resolution

- Wrapped `TestGlobalDBDeleteTaskMapsChildConstraintToValidationError` in a named `t.Run("ShouldMapChildConstraintFailuresToTaskValidationErrors", ...)` subtest.
- Kept the existing assertions and test data unchanged; this is a conformance fix for the repo's Go test structure requirements.
- Verified with `make verify` (exit `0`).
