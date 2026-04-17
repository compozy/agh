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

# Issue 031: Use t.Run("Should...") style for subtest names
## Review Comment

The current case labels work, but they miss the enforced test naming convention used in this repo.

As per coding guidelines: `MUST use t.Run("Should...") pattern for ALL test cases`.

Also applies to: 33-34, 40-41, 49-50

## Triage

- Decision: `VALID`
- Notes: The table-driven subtests in `agent_resource_test.go` still use descriptive lowercase labels instead of the repo’s required `Should...` pattern. This is a style-only change, but it is an explicit project test convention and should be brought into compliance while the file is open for other review fixes.
