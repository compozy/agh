---
status: resolved
file: internal/daemon/restart_test.go
line: 533
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133264038,nitpick_hash:65351d634791
review_hash: 65351d634791
source_review_id: "4133264038"
source_review_submitted_at: "2026-04-18T02:14:16Z"
---

# Issue 020: Make this “not launched yet” assertion event-driven.
## Review Comment

Waiting a fixed 50ms only proves the helper did not launch *quickly*; a slower regression can still pass. Prefer an explicit sync point or probe channel over a wall-clock delay here.

As per coding guidelines, "Never use `time.Sleep()` in orchestration — use proper synchronization primitives."

## Triage

- Decision: `valid`
- Notes:
  Root cause confirmed in `internal/daemon/restart_test.go`: the pre-launch assertion relies on a fixed `time.After(50 * time.Millisecond)`, which can miss slower regressions and violates the repo’s orchestration rule. I will replace it with an explicit synchronization point that proves the helper observed the old daemon as still alive and still did not launch the replacement.
