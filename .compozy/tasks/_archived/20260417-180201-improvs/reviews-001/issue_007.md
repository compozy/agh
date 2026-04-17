---
status: resolved
file: internal/extension/host_api_tasks.go
line: 356
severity: major
author: coderabbitai[bot]
provider_ref: review:4130502052,nitpick_hash:0301330fb44d
review_hash: 0301330fb44d
source_review_id: "4130502052"
source_review_submitted_at: "2026-04-17T16:38:53Z"
---

# Issue 007: Wrap helper errors with operation context before returning.
## Review Comment

Lines 359 and 363 return raw errors from `h.taskManager()` and `h.taskActorContext(ctx)` respectively. Wrap these with descriptive context using `fmt.Errorf("context: %w", err)` so that callers can identify whether manager resolution or actor derivation failed. Per coding guidelines: "Use explicit error returns with wrapped context: `fmt.Errorf("context: %w", err)`".

## Triage

- Decision: `VALID`
- Notes:
  `taskManagerAndActor` returns raw helper errors from `taskManager()` and
  `taskActorContext(ctx)`, so callers cannot distinguish manager resolution from
  actor derivation failures. Plan: wrap both branches with precise operation
  context and add unit coverage for each failure.
