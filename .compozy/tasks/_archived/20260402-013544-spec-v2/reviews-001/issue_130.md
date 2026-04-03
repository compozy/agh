---
status: resolved
file: internal/dashboard/websocket.go
line: 317
severity: high
author: claude-reviewer
---

# Issue 130: wsClient.enqueue adds oversized payload that exceeds maxQueuedByte limit



## Review Comment

In `wsClient.enqueue` (line 297), when a single payload exceeds `maxQueuedByte`, the method clears the queue and increments the drop counter but then unconditionally enqueues the oversized payload anyway:

```go
if len(payload) > c.maxQueuedByte {
    c.queue = nil       // clear everything
    c.queueBytes = 0
    c.dropped++
}

// This always runs -- the oversized payload is enqueued regardless
c.queue = append(c.queue, outboundMessage{typ: typ, data: payload})
c.queueBytes += len(payload)
```

After this code runs, `c.queueBytes` exceeds `c.maxQueuedByte`, violating the intended backpressure invariant. The task requirement states "buffer up to 64KB per client, drop oldest on overflow" -- but this allows a single message larger than 64KB to be queued.

With PTY binary output, it's plausible that a large burst of terminal output could produce a chunk exceeding 64KB, causing unbounded memory growth per WebSocket client.

**Suggested fix:** When a single payload exceeds the buffer limit, drop it entirely rather than enqueuing it:

```go
if len(payload) > c.maxQueuedByte {
    c.queue = nil
    c.queueBytes = 0
    c.dropped++
    c.cond.Signal()
    return nil  // Drop the oversized payload
}
```

Or alternatively, accept it as the sole message but document that the queue can temporarily exceed the limit by one message.

## Triage

- Decision: `valid`
- Notes: Confirmed in `wsClient.enqueue`: if `len(payload) > c.maxQueuedByte`, the queue is cleared and `dropped` is incremented, but the oversized payload is still appended immediately afterward. That breaks the queue-size invariant and is a direct correctness bug in the backpressure implementation.
