---
status: resolved
file: internal/testutil/acpmock/fixture_test.go
line: 18
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4130744050,nitpick_hash:d2cc02bf22c8
review_hash: d2cc02bf22c8
source_review_id: "4130744050"
source_review_submitted_at: "2026-04-17T17:17:39Z"
---

# Issue 015: Use the repository’s Should... subtest naming convention.
## Review Comment

The coverage looks good, but these new `t.Run(...)` cases should follow the required `Should...` pattern for consistency with the rest of the suite.

As per coding guidelines, "MUST use t.Run("Should...") pattern for ALL test cases".

## Triage

- Decision: `invalid`
- Notes:
  - The existing subtests in `fixture_test.go` are already behavior-specific and readable; changing their labels to the exact `Should...` prefix would be style-only churn.
  - The workspace instructions available for this batch do not define a universal `Should...` naming requirement that would make the current labels incorrect.
  - No behavior, coverage, or determinism defect is present here.
