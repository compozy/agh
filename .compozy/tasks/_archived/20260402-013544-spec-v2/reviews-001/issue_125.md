---
status: resolved
file: internal/dashboard/websocket.go
line: 99
severity: medium
author: claude-reviewer
---

# Issue 125: PTY WebSocket handler does not wait for read goroutine on all exit paths



## Review Comment

In `handlePTYWebSocket`, a read goroutine is spawned at lines 93-97:

```go
readDone := make(chan struct{})
go func() {
    defer close(readDone)
    s.handlePTYReads(c.Request.Context(), conn, client, subscription)
}()
```

However, when the handler exits via `<-client.Done()` (line 101-102), the read goroutine may still be blocked inside `conn.Read()`. The handler returns without waiting for `readDone`. While `conn.Close()` from the write loop will eventually unblock the read, the handler has already returned and the deferred `subscription.Unsubscribe()` has already run. This creates a window where the read goroutine could attempt to use the subscription (specifically `subscription.SnapshotFn()` on line 143) after it has been unsubscribed.

In practice this is mitigated because `conn.Close()` causes `conn.Read()` to return an error, and the read goroutine exits on any read error. But the ordering is not guaranteed -- the read goroutine could be processing a message (replay request) when the defer chain runs.

**Suggested fix:** Wait for the read goroutine to finish on all exit paths:

```go
case <-client.Done():
    <-readDone  // Ensure read goroutine has exited
    return
```

Or cancel a context that the read goroutine's `conn.Read` call uses, ensuring it exits promptly.

## Triage

- Decision: `valid`
- Notes: Confirmed in `handlePTYWebSocket`: on the `client.Done()` exit path the handler returns immediately without waiting for the read goroutine, even though other exit paths do synchronize after closing the websocket. The read goroutine can still be inside `conn.Read` or replay handling while deferred unsubscription runs, so the handler should close/cancel and wait consistently on all exits.
