---
status: resolved
file: internal/session/stop_reason.go
line: 111
severity: major
author: coderabbitai[bot]
provider_ref: review:4130502052,nitpick_hash:c88129ce6aa4
review_hash: c88129ce6aa4
source_review_id: "4130502052"
source_review_submitted_at: "2026-04-17T16:38:53Z"
---

# Issue 015: TOCTOU race in error suppression logic at lines 120–121.
## Review Comment

If the process exits between the snapshot at line 111 and the `Stop()` call at line 112, a benign `proc.Wait()` error can be returned as a hard failure. The error check at line 120 uses only the pre-call `doneBeforeStop` snapshot; it should also verify the current state.

Change line 120 from:
```go
if stopErr != nil && !doneBeforeStop {
```
to:
```go
if stopErr != nil && !doneBeforeStop && !isProcessDone(proc) {
```

This ensures errors are suppressed only when the process was not done before, during, and after the `Stop()` call.

## Triage

- Decision: `VALID`
- Notes:
  `StopWithCause` decides whether to surface `driver.Stop` errors using only the
  pre-call `doneBeforeStop` snapshot. If the process exits during `Stop`, a
  benign shutdown race can still be reported as a hard failure. Plan: re-check
  the current process state after `Stop` returns and add a lifecycle test that
  covers the race.
