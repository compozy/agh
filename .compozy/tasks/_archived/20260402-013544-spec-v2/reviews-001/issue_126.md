---
status: resolved
file: internal/dashboard/websocket.go
line: 229
severity: medium
author: claude-reviewer
---

# Issue 126: wsClient writeLoop goroutine is fire-and-forget with no ownership tracking



## Review Comment

The `newWSClient` constructor spawns a goroutine for the write loop at line 229:

```go
func newWSClient(conn *websocket.Conn, maxQueuedBytes int) *wsClient {
    // ...
    go client.writeLoop()
    return client
}
```

This goroutine has no explicit ownership via `sync.WaitGroup` or context cancellation -- it is a fire-and-forget goroutine. While the goroutine does eventually terminate (when `Close` or `CloseWithControl` is called, which sets `c.closing` and signals the condition variable), the calling code has no way to synchronize with the goroutine's completion other than reading from the `c.done` channel.

The `Done()` channel provides a way to wait, but the project coding style states: "No fire-and-forget goroutines; track all goroutines with `sync.WaitGroup` or equivalent." The `done` channel is a partial implementation of this but the goroutine is spawned in a constructor rather than being explicitly started, making the lifecycle less visible.

This is a minor architectural concern rather than a bug, since the `Done()` channel is used in callers. But it deviates from the explicit ownership pattern the project requires.

**Suggested fix:** Either document in the constructor that the caller is responsible for calling `Close` and waiting on `Done()`, or restructure so the write loop is started explicitly:

```go
client := newWSClient(conn, bufferSize)
// Start is explicit, making ownership visible
client.Start()
defer func() {
    _ = client.Close(...)
    <-client.Done()
}()
```

## Triage

- Decision: `invalid`
- Notes: The constructor-spawned write loop is already paired with explicit shutdown primitives: callers are required to call `Close`/`CloseWithControl` and can wait on `Done()`, which the current handlers do. The concern here is about API shape and lifecycle visibility, not an observed leak or correctness failure in the scoped code, so it is not an actionable bug for this batch.
