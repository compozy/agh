---
provider: coderabbit
pr: "85"
round: 1
round_created_at: 2026-04-30T14:00:14.99254Z
status: resolved
file: internal/cli/task_test.go
line: 623
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4204955814,nitpick_hash:b158307b8320
review_hash: b158307b8320
source_review_id: "4204955814"
source_review_submitted_at: "2026-04-30T12:11:10Z"
---

# Issue 022: Normalize these subtest names to the Should ... pattern.
## Review Comment

The changed subtests here use short labels like `"next"` and `"release"`, which drifts from the repository’s required Go test naming convention.

As per coding guidelines, "MUST use t.Run("Should...") pattern for ALL test cases".

## Triage

- Decision: `VALID`
- Notes: `TestAgentTaskCommandsMapLeaseRequests` uses subtest names such as `next`, `next no work`, `heartbeat`, `complete`, `fail`, and `release`. These are changed test cases and they do not follow the required `Should ...` naming pattern. The fix is to rename the subtests/table `name` fields without changing behavior.
- Resolution: Renamed affected `task_test.go` subtests/table cases to `Should ...` names and verified with the AGH test-shape checker plus `make verify`.
