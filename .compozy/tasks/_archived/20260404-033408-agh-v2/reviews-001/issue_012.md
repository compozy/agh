---
status: resolved
file: internal/daemon/daemon.go
line: 679
severity: medium
author: claude-code
provider_ref:
---

# Issue 012: No delay between SIGTERM and SIGKILL in orphan cleanup

## Review Comment

In `cleanupOrphans` (line 664), after sending `SIGTERM` at line 679, the code immediately calls `processAlive` at line 683. Since SIGTERM is asynchronous, the process is almost certainly still alive at that instant, so SIGKILL will nearly always be sent at line 684, defeating the purpose of the graceful SIGTERM.

```go
if err := d.signalProcess(proc.PID, syscall.SIGTERM); err != nil {
    continue
}
if d.processAlive(proc.PID) {  // always true right after SIGTERM
    if err := d.signalProcess(proc.PID, syscall.SIGKILL); err != nil {
```

**Suggested fix:** Add a short polling loop (e.g., up to 2 seconds with 100ms ticks) checking `processAlive` before escalating to SIGKILL. Use context for cancellation to respect shutdown deadlines.

## Triage

- Decision: `valid`
- Notes: `cleanupOrphans()` sends `SIGTERM` and immediately checks `processAlive()`, which makes the graceful signal effectively useless because process exit is asynchronous. A short wait/poll period is required before escalating to `SIGKILL`.
