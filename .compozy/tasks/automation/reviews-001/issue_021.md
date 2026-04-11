---
status: resolved
file: internal/cli/automation_test.go
line: 467
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093724766,nitpick_hash:0b7071e55645
review_hash: 0b7071e55645
source_review_id: "4093724766"
source_review_submitted_at: "2026-04-11T12:31:10Z"
---

# Issue 021: Assert the webhook fields on trigger update.
## Review Comment

This case passes `--event webhook`, `--webhook-id`, and `--endpoint-slug`, but the test only checks `Retry`, `Filter`, and `WebhookSecret`. If any of those new flags stopped being wired into `updateTriggerRequest`, this test would still pass.

As per coding guidelines, "Focus on critical paths: workflow execution, state management, error handling".

## Triage

- Decision: `valid`
- Notes: The trigger-update CLI test passes webhook-specific flags but only asserts a subset of the parsed request, so wiring regressions for `event`, `webhook_id`, or `endpoint_slug` would go unnoticed. I will extend the assertions to cover the webhook fields that command path is supposed to populate.
