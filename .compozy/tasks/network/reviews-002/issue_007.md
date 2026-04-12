---
status: resolved
file: internal/cli/daemon_wait_test.go
line: 136
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093888065,nitpick_hash:1386f266a6c4
review_hash: 1386f266a6c4
source_review_id: "4093888065"
source_review_submitted_at: "2026-04-11T14:57:02Z"
---

# Issue 007: Assert stop-state explicitly, not only Network == nil.
## Review Comment

At Line 140, this test only checks network clearing. Add a `status.Status == "stopped"` assertion (optionally PID too) so state-transition regressions are caught.

As per coding guidelines, "`**/*_test.go`: Focus on critical paths: workflow execution, state management, error handling" and "`**/*_test.go`: MUST test meaningful business logic, not trivial operations".

## Triage

- Decision: `valid`
- Root cause: The stale-network snapshot test only proves the nested network payload is cleared; it does not assert the daemon reached the stopped state.
- Fix plan: Strengthen the assertions to verify the returned status reflects a stopped daemon as well as a cleared network snapshot.
