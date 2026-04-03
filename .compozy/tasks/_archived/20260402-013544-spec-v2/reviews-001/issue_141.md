---
status: resolved
file: internal/drivers/pi/pi.go
line: 441
severity: high
author: claude-reviewer
---

# Issue 141: Goroutine leak in DetectReady when context is cancelled or timeout fires (pi and opencode drivers)



## Review Comment

**Partially fixed:** The Codex driver (codex.go:426) and Claude driver (claude.go:487) now use an `observeProcessExit` helper that returns a cancellable cleanup function (`stopWaiting`), properly preventing goroutine leaks. However, the Pi and OpenCode drivers still have the original problematic pattern.

In `DetectReady` in the Pi driver (pi.go:441) and `detectTUIReady`/`detectServerReady` in the OpenCode driver, a goroutine is launched to call `proc.Wait()` with no cleanup mechanism:

```go
waitCh := make(chan error, 1)
go func() {
    waitCh <- proc.Wait()
}()
```

If the function returns via the `ctx.Done()` or `timeout.C` case, this goroutine is never cleaned up. It will block forever on `proc.Wait()` because the process is still running (it hasn't been stopped yet).

The remaining affected locations are:
- `internal/drivers/pi/pi.go:441` (DetectReady)
- `internal/drivers/opencode/opencode.go:688` (detectTUIReady)
- `internal/drivers/opencode/opencode.go:726` (detectServerReady)

After `DetectReady` returns an error, the caller does call `Stop()`, which will eventually cause `proc.Wait()` to return. However, if `Stop()` is also subject to timeout and the process is truly hung, these goroutines accumulate. More importantly, calling `proc.Wait()` from multiple goroutines (DetectReady's goroutine and Stop's `waitWithTimeout` goroutine) could lead to undefined behavior depending on the Wait implementation.

**Suggested fix**: Apply the same `observeProcessExit` pattern already used by the Codex and Claude drivers. This helper creates a cancellable wait context and returns a cleanup function that cancels the context and waits for the goroutine to finish:

```go
waitCh, stopWaiting := observeProcessExit(proc)
defer stopWaiting()
```

## Triage

- Decision: `valid`
- Notes: Confirmed in both scoped drivers: `pi.DetectReady` and `opencode.detectTUIReady`/`detectServerReady` launch a goroutine that blocks on `proc.Wait()` with no cancellation or join path when readiness returns early on context cancellation or timeout. This is the same process-wait lifecycle bug already fixed in adjacent drivers and is actionable here.
