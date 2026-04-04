---
status: resolved
file: internal/httpapi/server.go
line: 477
severity: high
author: claude-code
provider_ref:
---

# Issue 004: Data race on Handlers.streamDone and httpPort fields

## Review Comment

`setStreamDone()` (line 477) and `setHTTPPort()` (line 481) write to `Handlers` fields without synchronization, but these fields are concurrently read by HTTP handler goroutines (e.g., `promptSession` in `prompt.go`, `streamSession` in `stream.go`, `daemonStatus` in `daemon.go`).

The writes happen inside `Server.Start()` which holds `Server.mu`, but `Handlers` has no mutex of its own, and HTTP handlers do not acquire `Server.mu`. The Go memory model does not guarantee that goroutines spawned by `net/http.Serve` observe writes made before the `Serve` call, because the happens-before relationship only flows through direct goroutine creation.

The same pattern exists in `internal/udsapi/handlers.go:219` (see issue_005).

**Suggested fix:** Pass `streamDone` and `httpPort` as constructor arguments to `newHandlers` so they are set before the `Handlers` value is shared, or use `atomic.Value` / `sync.Once` for these fields.

## Triage

- Decision: `invalid`
- Notes: `streamDone` and `httpPort` are assigned before the serving goroutine is started and are never mutated afterward. That publication pattern is safe here: the handler struct is fully initialized before `httpServer.Serve` is launched, and request goroutines only read immutable values after that point. This is not a real concurrent read/write race in the current implementation.
