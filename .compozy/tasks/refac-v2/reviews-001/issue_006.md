---
status: resolved
file: internal/session/manager_lifecycle.go
line: 328
severity: high
author: claude-code
provider_ref:
---

# Issue 006: Nil dereference on processHandle().Stderr() in error path

## Review Comment

In `finalizeStopped`, when `waitErr != nil`, the code calls `session.processHandle().Stderr()` at line 328 without a nil guard. `processHandle()` can return `nil` if the process was already cleared by a concurrent stop path or never fully initialized. A few lines later (line 346), the same pattern is used *with* a nil guard (`if proc := session.processHandle(); proc != nil`), showing awareness of the problem. The first call lacks this guard and will panic.

```go
// line 328 — no nil guard, will panic
Text: session.processHandle().Stderr(),

// line 346 — correctly guarded
if proc := session.processHandle(); proc != nil {
    stopEvent.Text = proc.Stderr()
}
```

**Fix:** Guard the first `processHandle()` call the same way:

```go
var stderr string
if proc := session.processHandle(); proc != nil {
    stderr = proc.Stderr()
}
event := acp.AgentEvent{
    // ...
    Text: stderr,
}
```

## Triage

- Decision: `valid`
- Root cause: `finalizeStopped` dereferences `session.processHandle()` in the error-event path without checking whether another stop/finalization path has already cleared the process pointer.
- Fix approach: Capture stderr through a nil-guarded process lookup before building the error event, and add coverage for the wait-error path with a cleared process handle.
- Resolution: Implemented in the lifecycle finalization path and verified by the full repository gate.
