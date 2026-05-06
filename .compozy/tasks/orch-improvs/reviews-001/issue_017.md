---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/daemon/task_runtime.go
line: 320
severity: major
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:1f6cc5975189
review_hash: 1f6cc5975189
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 017: Don’t make missing reentry support a daemon-wide boot failure.
## Review Comment

`bootTasks` already treats a missing `taskStore` as a feature downgrade, but this new path makes the weaker `harnessReentryStore` / `harnessReentrySessionManager` checks fatal. That means a registry or session manager that can still execute tasks will now stop the whole daemon at startup just because synthetic reentry is unavailable. Consider treating “reentry unsupported” as an opt-out and only returning an error for real construction failures.

Also applies to: 374-392

## Triage

- Decision: `valid`
- Notes:
  - `bootHarnessReentryBridge` still treats missing `harnessReentrySessionManager` / `harnessReentryStore` support as fatal errors.
  - `bootTasks` already tolerates a weaker registry surface for the broader task runtime, so this path currently upgrades an optional feature into a daemon-wide boot blocker.
  - Planned fix: downgrade unsupported synthetic reentry support to a logged no-op, while preserving hard failures for actual bridge construction errors.
  - Resolved: unsupported reentry support is now downgraded to warnings/no-op, `bootTasks` skips recovery when reentry is absent, and runtime tests cover the degraded boot path.
