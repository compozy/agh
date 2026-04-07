---
status: resolved
file: internal/store/sessiondb/session_db.go
line: 338
severity: medium
author: claude-code
provider_ref:
---

# Issue 009: SessionDB writerLoop has no ctx.Done() exit path

## Review Comment

The `writerLoop` goroutine only exits when it receives on `shutdownCh`. There is no `ctx.Done()` check — the goroutine started in `OpenSessionDB` uses no context for its lifecycle. If the caller loses all references to the `SessionDB` without calling `Close()`, the goroutine leaks forever.

```go
func (s *SessionDB) writerLoop() {
    for {
        select {
        case req := <-s.writeCh:
            req.result <- s.executeWrite(req)
        case shutdown := <-s.shutdownCh:
            shutdown.result <- s.drainWrites(shutdown.ctx)
            return
        // missing: case <-ctx.Done(): return
        }
    }
}
```

**Fix:** Accept a `context.Context` (or store a cancel func from construction) and add a `case <-ctx.Done()` branch to the select. Call the cancel in `Close()` as a belt-and-suspenders alongside the shutdown channel.

## Triage

- Decision: `valid`
- Root cause: `SessionDB` owns a dedicated writer goroutine but gives it no internal cancellation path beyond `Close`. If the owner abandons the instance, the goroutine has no way to observe lifecycle cancellation and remains blocked forever.
- Fix approach: Add an internal lifecycle context/cancel pair for the writer loop, cancel it from `Close`, and extend tests around close semantics so the goroutine has an explicit shutdown signal.
- Resolution: Implemented with internal writer cancellation and regression coverage; full repository verification passed.
