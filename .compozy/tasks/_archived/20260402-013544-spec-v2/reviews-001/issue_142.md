---
status: resolved
file: internal/drivers/pi/pi.go
line: 676
severity: high
author: claude-reviewer
---

# Issue 142: Goroutine leak in waitWithTimeout when timeout fires before process exits (pi and opencode drivers)



## Review Comment

**Partially fixed:** The Codex driver (codex.go:662) and Claude driver (claude.go:738) have been rewritten to use `proc.WaitContext(waitCtx)` with a derived context and `context.WithTimeout`, eliminating the goroutine entirely. However, the Pi and OpenCode drivers still use the old pattern that spawns an orphaned goroutine.

In the Pi driver (pi.go:676) and OpenCode driver (opencode.go:1209), `waitWithTimeout` still spawns a goroutine to call `proc.Wait()`:

```go
func waitWithTimeout(ctx context.Context, proc *kernel.AgentProcess, timeout time.Duration) (bool, error) {
    waitCh := make(chan error, 1)
    go func() {
        waitCh <- proc.Wait()
    }()
    ...
}
```

When the timer fires (timeout case) or the context is cancelled, the function returns `(true, context.DeadlineExceeded)` but the goroutine calling `proc.Wait()` continues to block. This goroutine is orphaned.

In the `Stop` method, `waitWithTimeout` is called after sending SIGTERM. If the SIGTERM timeout fires, a SIGKILL is sent, followed by another `proc.WaitContext(ctx)`. This means there are now potentially two orphaned goroutines from the first `waitWithTimeout` call (one from DetectReady and one from Stop's first waitWithTimeout) all blocking on `proc.Wait()`.

The remaining affected locations are:
- `internal/drivers/pi/pi.go:676`
- `internal/drivers/opencode/opencode.go:1209`

**Suggested fix**: Apply the same context-based pattern already used by the Codex and Claude drivers, which uses `proc.WaitContext` with a derived timeout context instead of spawning a goroutine:

```go
func waitWithTimeout(ctx context.Context, proc *kernel.AgentProcess, timeout time.Duration) (bool, error) {
    if timeout <= 0 {
        return false, proc.WaitContext(ctx)
    }

    waitCtx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    err := proc.WaitContext(waitCtx)
    if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
        return true, context.DeadlineExceeded
    }
    return false, err
}
```

## Triage

- Decision: `valid`
- Notes: Confirmed in both scoped `waitWithTimeout` helpers: they spawn a goroutine that blocks on `proc.Wait()` and return on timeout or caller cancellation without cleaning it up. The Codex/Claude drivers have already moved to `proc.WaitContext`, and the same root-cause fix applies here.
