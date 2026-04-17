---
status: resolved
file: internal/testutil/acpmock/fixture_test.go
line: 274
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4129384275,nitpick_hash:2c1b4edccc8a
review_hash: 2c1b4edccc8a
source_review_id: "4129384275"
source_review_submitted_at: "2026-04-17T13:54:50Z"
---

# Issue 022: Consider adding t.Parallel() to the parent test function.
## Review Comment

`TestRegistrationHelperOverridesAndDiagnosticsErrors` is missing `t.Parallel()` on the parent function. Note that the subtest using `t.Setenv` (line 280) correctly omits `t.Parallel()` since `t.Setenv` is incompatible with parallel execution.

## Triage

- Decision: `invalid`
- Notes:
  The parent test cannot safely call `t.Parallel()` because one of its subtests
  uses `t.Setenv`. Go forbids `t.Setenv` in a parallel test or under a parallel
  ancestor, so parallelizing the parent would violate the testing contract and
  make the existing env override coverage invalid.

## Resolution

- No code change. Analysis confirmed the parent test must remain serial because
  a child subtest uses `t.Setenv`.
