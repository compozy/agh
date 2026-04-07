---
status: resolved
file: internal/api/core/handlers.go
line: 111
severity: medium
author: claude-code
provider_ref:
---

# Issue 014: Unsynchronized writes to BaseHandlers mutable fields

## Review Comment

`SetStreamDone()` and `SetHTTPPort()` write to fields on `BaseHandlers` with no synchronization. These fields are read concurrently by SSE handler goroutines in `select` statements (`StreamSession` at line 336, `StreamObserveEvents` at line 508, `DaemonStatus` at line 561). While current usage writes happen-before handler goroutines start (writes occur during `Server.Start` under the server mutex before `Serve` launches), `BaseHandlers` has no structural guarantee. If any caller invokes `SetStreamDone` after the server starts accepting connections, a data race occurs.

Additionally, if `SetStreamDone` is never called, `h.StreamDone` is `nil`. A `select` case on a nil channel never fires, so graceful server shutdown via stream cancellation silently fails.

**Fix:** Either make `StreamDone` and `HTTPPort` constructor-only values (remove setters), or use `atomic.Value`/`sync.RWMutex`. Initialize `StreamDone` to a default closed-never channel in `NewBaseHandlers`.

## Triage

- Decision: `valid`
- Root cause: `BaseHandlers` exposes mutable transport state through unsynchronized setters while the SSE handlers read those fields concurrently. The current startup path happens to set them early, but the type itself does not enforce that usage pattern.
- Fix approach: Move `StreamDone` and `HTTPPort` behind synchronized accessors, give `StreamDone` a non-nil default channel, and update tests to exercise the new concurrency-safe behavior.
- Resolution: Implemented with synchronized transport-state accessors and passing transport tests.
