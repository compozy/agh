---
status: resolved
file: internal/daemon/task_runtime_test.go
line: 1889
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4135189365,nitpick_hash:5eaeaaf77889
review_hash: 5eaeaaf77889
source_review_id: "4135189365"
source_review_submitted_at: "2026-04-18T22:38:10Z"
---

# Issue 021: Replace the sleep-poll helper with synchronization.
## Review Comment

This helper makes the detached runtime tests timing-sensitive under load and under `-race`. Prefer waiting on an explicit signal or a ticker/context-based helper instead of a fixed `time.Sleep` loop.

As per coding guidelines, "Never use time.Sleep() in orchestration — use proper synchronization primitives".

## Triage

- Decision: `valid`
- Root cause: `waitForTaskRuntimeCondition` still uses a fixed `time.Sleep(10 * time.Millisecond)` loop in the shared detached-runtime test helper. That introduces timing jitter under load and under `-race`, which is exactly the orchestration pattern the repo forbids.
- Fix approach: Replace the fixed-sleep polling with timer/ticker synchronization so the helper waits on explicit timeout and polling signals without arbitrary sleeps.
