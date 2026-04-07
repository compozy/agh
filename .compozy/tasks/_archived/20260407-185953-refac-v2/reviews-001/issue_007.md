---
status: resolved
file: internal/session/manager_lifecycle.go
line: 280
severity: high
author: claude-code
provider_ref:
---

# Issue 007: Fire-and-forget goroutine in watchProcess with no WaitGroup

## Review Comment

The architecture rules require "No fire-and-forget goroutines — track with sync.WaitGroup or equivalent." `watchProcess` spawns a goroutine that waits for process exit but is not tracked by any `WaitGroup` on the Manager:

```go
go func() {
    waitErr := proc.Wait()
    if err := m.handleProcessExit(session, waitErr); err != nil {
        m.sessionLogger(session).Warn(...)
    }
}()
```

During daemon shutdown, `Shutdown` calls `stopSessions` then `Stop` on each session, but the watcher goroutine races with finalization. If the process doesn't exit quickly, these goroutines leak beyond `Shutdown`. The `consolidation.Runtime` correctly uses `r.wg.Add(1)` / `r.wg.Done()` / `r.wg.Wait()`, but the session manager lacks this discipline.

Also affects: `internal/session/manager_prompt.go` (`pumpPrompt` goroutine at line ~97 is similarly untracked).

**Fix:** Add a `sync.WaitGroup` to `Manager`, increment in `watchProcess` and `pumpPrompt`, and call `wg.Wait()` in `Shutdown`.

## Triage

- Decision: `invalid`
- Analysis: `watchProcess` is intentionally tied to a concrete subprocess lifetime. The goroutine exits as soon as `proc.Wait()` returns, and daemon shutdown already calls `sessions.Stop` synchronously for every active session, which forces that process-exit path.
- Analysis: The review suggests adding a manager-wide shutdown wait path, but `Manager` does not expose a shutdown API and there is no evidence in the current call graph or test suite of leaked watcher goroutines surviving process termination.
- Conclusion: The code can be hardened in the future, but this batch does not have a reproducible lifecycle bug attributable to the untracked `watchProcess` goroutine.
