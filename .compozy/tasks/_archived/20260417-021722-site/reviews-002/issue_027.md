---
status: resolved
file: internal/config/agent_resource_test.go
line: 26
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:67a8e95f4668
review_hash: 67a8e95f4668
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 027: Use t.Run("Should...") style for subtest names
## Review Comment

The current case labels work, but they miss the enforced test naming convention used in this repo.

As per coding guidelines: `MUST use t.Run("Should...") pattern for ALL test cases`.

Also applies to: 33-34, 40-41, 49-50

## Triage

- Decision: `VALID`
- Notes:
  - The reviewed test file has moved; the current equivalent table-driven parser tests are in `internal/config/agent_test.go`.
  - Root cause: the subtest labels in that file still use lower-case descriptive names instead of the repo-preferred `Should...` naming convention.
  - Intended fix: rename the touched table cases while updating the same test file for issue 026.
  - Result: renamed the affected current subtest cases to `Should...` labels in `internal/config/agent_test.go`; verified with `go test ./internal/config` and `make verify`.
