---
status: resolved
file: internal/dashboard/websocket.go
line: 402
severity: medium
author: claude-reviewer
---

# Issue 123: wsClient.write uses context.Background() instead of propagating parent context



## Review Comment

The `wsClient.write` method (line 402) and the standalone `writeWSMessage` helper (line 422) both create timeouts from `context.Background()`:

```go
func (c *wsClient) write(typ websocket.MessageType, payload []byte) error {
    ctx, cancel := context.WithTimeout(context.Background(), c.writeTimeout)
    defer cancel()
    return c.conn.Write(ctx, typ, payload)
}

func writeWSMessage(conn *websocket.Conn, typ websocket.MessageType, payload []byte) error {
    ctx, cancel := context.WithTimeout(context.Background(), defaultWSWriteTimeout)
    defer cancel()
    return conn.Write(ctx, typ, payload)
}
```

Using `context.Background()` means these writes are not cancellable when the server is shutting down. If the server begins graceful shutdown, existing write operations will not be interrupted and will continue until their individual 5-second timeout expires. This violates the project's coding style: "Pass `context.Context` as the first argument to all functions crossing runtime boundaries; avoid `context.Background()` outside `main` and focused tests."

**Suggested fix:** Store a parent context (e.g., the server or session context) in the `wsClient` struct and derive write timeouts from it. This ensures that when the parent context is cancelled during shutdown, all pending writes are immediately interrupted:

```go
func (c *wsClient) write(typ websocket.MessageType, payload []byte) error {
    ctx, cancel := context.WithTimeout(c.parentCtx, c.writeTimeout)
    defer cancel()
    return c.conn.Write(ctx, typ, payload)
}
```

## Triage

- Decision: `valid`
- Notes: Confirmed in `internal/dashboard/websocket.go`: both `wsClient.write` and `writeWSMessage` derive deadlines from `context.Background()`, so websocket writes ignore request/session cancellation. This is a concrete lifecycle bug during shutdown and should be fixed by threading the parent context into the write helpers.
