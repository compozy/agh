---
status: resolved
file: internal/memory/dream.go
line: 160
severity: medium
author: claude-code
provider_ref:
---

# Issue 003: ShouldRun() acquires lock as hidden side effect

## Review Comment

`ShouldRun()` (line 160) evaluates three gates and, when all pass, acquires the consolidation lock via `acquireLock()` — setting `s.pending = true` and creating the lock file on disk. The method name strongly implies a read-only check ("should this run?"), but it mutates state.

Current callers (`daemon.runDreamCheck` and `runtimeDreamTrigger.Trigger`) always follow `ShouldRun()` with `Run()`, so the lock is properly released. However:

1. A future caller using `ShouldRun()` as a pure query (e.g., for health/status reporting) would leak the lock permanently — `pending` stays true, blocking all subsequent consolidation attempts for the process lifetime.
2. The stale PID detection on the lock file would eventually reclaim it on daemon restart, but the in-memory `pending` flag has no recovery path.

**Suggested fix**: Split into two methods:
- `GatesPassed() (bool, error)` — pure read, evaluates time + session gates only
- `TryStart() (bool, error)` — evaluates all gates including lock acquisition

Alternatively, document the lock-acquiring behavior prominently and rename to `TryAcquireForRun()` or similar to signal the side effect.

## Triage

- Decision: `valid`
- Root cause: `ShouldRun()` is not a pure gate check. It mutates service state by acquiring the consolidation lock and setting `pending`, which makes the API easy to misuse and couples a read-style predicate to lifecycle side effects.
- Evidence: `internal/memory/dream.go` calls `acquireLock()` inside `ShouldRun()`, and the existing tests assert that `service.pending` is set after a successful `ShouldRun()`.
- Fix approach: Make `ShouldRun()` a pure gate evaluation and let `Run()` remain the only method that acquires/releases the lock. Update callers and tests so lock contention is handled at run time rather than as a side effect of the predicate.
- Resolution: `ShouldRun()` now evaluates only the time/session gates, `Run()` is the sole lock-acquiring method, and callers treat lock contention as a run-time skip instead of a predicate side effect.
