---
provider: coderabbit
pr: "106"
round: 2
round_created_at: 2026-05-06T05:52:55.253953Z
status: resolved
file: internal/bridges/task_notifier.go
line: 237
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4233550358,nitpick_hash:f51a86b8ee8c
review_hash: f51a86b8ee8c
source_review_id: "4233550358"
source_review_submitted_at: "2026-05-06T05:52:14Z"
---

# Issue 001: Confusing function signature with two separate error return values.
## Review Comment

The return signature `(terminalTaskNotificationDecision, error, int64, error)` with two distinct error semantics (diagnostic at position 2, processing error at position 4) is non-idiomatic and error-prone for callers. Consider wrapping both into a single result struct:

```go
type processRecordResult struct {
decision terminalTaskNotificationDecision
diagnostic error
sequence int64
}
```

Then return `(processRecordResult, error)` where the error is the processing failure.

## Triage

- Decision: `VALID`
- Root cause: `processTaskNotificationRecord` currently returns a four-value tuple with two different error slots, so the call site has to remember positional semantics for diagnostic vs terminal failure.
- Fix approach: Replace the tuple with a small result struct plus one terminal error return, then update the notifier loop and add focused regression coverage in `internal/bridges/task_notifier_test.go`.
