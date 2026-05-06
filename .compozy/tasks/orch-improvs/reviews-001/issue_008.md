---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/cli/task.go
line: 658
severity: minor
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:9d8d8fcc1d42
review_hash: 9d8d8fcc1d42
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 008: Validate --last for notification listing.
## Review Comment

This builder forwards negative limits to the API, while the other list commands in this file reject them locally. `agh task notification list --last -1` should fail fast in the CLI instead of producing downstream behavior that depends on the server implementation.

## Triage

- Decision: `valid`
- Notes:
  - `buildTaskBridgeNotificationSubscriptionListQuery` still forwards `last` directly into the query without the negative-value guard used by sibling list builders.
  - That pushes an obvious CLI validation error downstream into server behavior.
  - Planned fix: reject negative `--last` values in the CLI builder and cover the contract with command tests.
  - Resolved: the task notification list builder now rejects negative `--last`, and CLI tests assert the validation error.
