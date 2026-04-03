---
status: resolved
file: internal/session/manager.go
line: 0
severity: high
author: claude-code
provider_ref:
---

# Issue 002: Manager has no Shutdown method -- resource leak on daemon exit

## Review Comment

The `Manager` has no `Shutdown(ctx)` or `StopAll(ctx)` method. When the daemon exits, running agent subprocesses are orphaned, open SQLite databases are not flushed/closed, and `watchProcess` goroutines (spawned at line ~769 as `go func()` calling `proc.Wait()`) have no cancellation mechanism -- they lack a `ctx.Done()` select case and no root context is propagated.

Additionally, `AgentProcess` contexts use `context.Background()` (in `acp/client.go:112`), so they are disconnected from the daemon lifecycle. Without an explicit shutdown path, a daemon stop relies entirely on post-mortem orphan cleanup on next boot.

**Suggested fix:** Add a `Shutdown(ctx context.Context) error` method that:
1. Iterates all active sessions and calls `Stop` on each
2. Waits for all `watchProcess` goroutines to complete (or context deadline)
3. Optionally accept a root context in the Manager constructor that is cancelled to signal all watchers

## Triage

- Decision: `invalid`
- Notes: The current runtime already has an explicit daemon shutdown path. `daemon.Shutdown()` calls `stopSessions()`, which calls `Manager.Stop()` for every live session, and `Driver.Stop()` waits for subprocess exit before `finalizeStopped()` closes the recorder and removes the session. The manager does not expose a separate `Shutdown()` method, but the claimed orphaning/leak path is not present in the current daemon-owned lifecycle.
