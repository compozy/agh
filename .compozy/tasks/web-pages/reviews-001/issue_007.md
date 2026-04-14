---
status: resolved
file: internal/api/udsapi/bridges_test.go
line: 127
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4103023844,nitpick_hash:ce2962199d83
review_hash: ce2962199d83
source_review_id: "4103023844"
source_review_submitted_at: "2026-04-14T02:37:32Z"
---

# Issue 007: Align this new UDS handler test with table-driven subtests.
## Review Comment

Please wrap this in `t.Run("Should...")` table-driven structure for consistency with the repository’s Go test policy.

As per coding guidelines, `**/*_test.go`: "Use table-driven tests with subtests (t.Run) as default in Go tests" and "MUST use t.Run("Should...") pattern for ALL test cases".

## Triage

- Decision: `valid`
- Root cause: the new UDS bridge-provider test is written as a standalone body instead of the repo-standard `t.Run("Should...")` shape.
- Fix approach: convert it to the table-driven subtest pattern used elsewhere in Go tests.
