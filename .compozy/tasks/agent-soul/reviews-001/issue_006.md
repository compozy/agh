---
provider: coderabbit
pr: "88"
round: 1
round_created_at: 2026-05-02T18:22:40.559088Z
status: pending
file: internal/api/core/automation_test.go
line: 490
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4215360648,nitpick_hash:aac6a196d8d1
review_hash: aac6a196d8d1
source_review_id: "4215360648"
source_review_submitted_at: "2026-05-02T18:22:08Z"
---

# Issue 006: Exercise the webhook_secret_ref path in this round-trip as well.
## Review Comment

This matrix now proves the inline `webhook_secret_value` flow, but it still never verifies that a `webhook_secret_ref` request survives handler parsing and reaches the manager. A regression in the new ref-backed path would pass this suite unchanged. Add a sibling create case that sends `webhook_secret_ref` and asserts the manager receives the ref instead of a value.

As per coding guidelines "Focus on critical paths: workflow execution, state management, error handling".

Also applies to: 661-668

## Triage

- Decision: `UNREVIEWED`
- Notes:
