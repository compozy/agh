---
status: resolved
file: internal/session/manager.go
line: 790
severity: medium
author: claude-code
provider_ref:
---

# Issue 007: Double-finalize race between Stop() and watchProcess goroutine

## Review Comment

Both `Manager.Stop` and `Manager.handleProcessExit` (via the `watchProcess` goroutine) can call `finalizeStopped` concurrently for the same session. Scenario: user calls `Stop()`, which calls `driver.Stop()`. While in progress, the process exits naturally, triggering `handleProcessExit`, which also calls `finalizeStopped`.

There is no guard preventing double-execution. This can cause: closing an already-closed recorder, writing metadata twice, double-removing from the sessions map, and double-notifying `OnSessionStopped`.

**Suggested fix:** Use a `sync.Once` per session for finalization, or add a check-and-set guard in `finalizeStopped` that atomically checks if already finalized.

## Triage

- Decision: `valid`
- Notes: `Stop()` and `handleProcessExit()` can both reach `finalizeStopped()` for the same session if the process exits in the window between `Stop()` loading state/process information and its later finalization path. There is no manager-side claim/once guard today, so duplicate stop events, duplicate notifier calls, and recorder-close races are possible.
