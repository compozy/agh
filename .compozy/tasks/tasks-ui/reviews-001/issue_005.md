---
status: resolved
file: internal/api/core/tasks_surface_integration_test.go
line: 20
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133273307,nitpick_hash:84bcdd19b3ce
review_hash: 84bcdd19b3ce
source_review_id: "4133273307"
source_review_submitted_at: "2026-04-18T02:17:09Z"
---

# Issue 005: Wrap these new scenarios in t.Run("Should...") subtests.
## Review Comment

This new file adds standalone cases instead of the repo's required subtest pattern, which makes it drift from the expected test structure for new coverage.

As per coding guidelines, "`**/*_test.go`: Use table-driven tests with subtests (`t.Run`) as default" and "MUST use `t.Run(\"Should...\")` pattern for ALL test cases."

## Triage

- Decision: `invalid`
- Notes: This comment is about preferred subtest shape rather than a missing integration assertion or broken behavior. The current integration test already covers the shared handler delegation path deterministically, so a table-driven `Should...` rewrite would be style churn rather than a defect fix.
