---
provider: coderabbit
pr: "88"
round: 2
round_created_at: 2026-05-02T22:54:45.308545Z
status: resolved
file: internal/automation/manager.go
line: 1468
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4215360648,nitpick_hash:10a342450dc4
review_hash: 10a342450dc4
source_review_id: "4215360648"
source_review_submitted_at: "2026-05-02T18:22:08Z"
---

# Issue 013: Remove the dead desiredTriggerSecrets parameter instead of keeping a hard-fail compatibility path.
## Review Comment

Every non-empty use now errors immediately, so the signature still advertises an obsolete mutation style that callers have to reason about even though it is no longer supported.

As per coding guidelines, "Never sacrifice code quality for backward compatibility in greenfield alpha; delete obsolete code instead of working around it."

## Triage

- Decision: `valid`
- Notes:
  - `SyncManagedDefinitions` still advertises the obsolete `desiredTriggerSecrets` input even though any non-empty use now hard-fails immediately.
  - That left a dead compatibility path in a greenfield-alpha surface; I removed the `desiredTriggerSecrets` parameter from `internal/automation/manager.go` and updated the minimal affected tests in `internal/automation/resource_test.go` and `internal/daemon/daemon_test.go`.
  - Verification: `make verify` passed after the signature cleanup and caller updates.
