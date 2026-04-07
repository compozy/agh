---
status: resolved
file: internal/memory/consolidation/runtime.go
line: 392
severity: medium
author: claude-code
provider_ref:
---

# Issue 010: Dream session deferred Stop uses same ctx that may be canceled

## Review Comment

In `spawnSession`, the deferred cleanup calls `sessions.Stop(ctx, ...)` using the same context passed to the function. If the context was canceled (the reason the prompt stream ended), the `Stop` call will also fail immediately, leaving the dream session running. The `cleanupFailedStart` in `session/manager_lifecycle.go` correctly creates a fresh timeout context for stop operations.

```go
defer func() {
    stopErr := sessions.Stop(ctx, dreamSession.ID) // ctx may already be canceled
    // ...
}()
```

**Fix:** Use a fresh background context with a timeout:

```go
defer func() {
    stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    stopErr := sessions.Stop(stopCtx, dreamSession.ID)
    // ...
}()
```

## Triage

- Decision: `valid`
- Root cause: Deferred dream-session cleanup reuses the request context passed into `spawnSession`. That context is often canceled precisely when prompt streaming ends, which can cause the stop call to fail immediately and leave the dream session running longer than intended.
- Fix approach: Use a fresh background timeout context for deferred shutdown, mirroring the session lifecycle cleanup pattern already used elsewhere.
- Resolution: Implemented and covered by consolidation runtime tests; full repository verification passed.
