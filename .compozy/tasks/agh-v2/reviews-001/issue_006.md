---
status: resolved
file: internal/udsapi/handlers.go
line: 219
severity: medium
author: claude-code
provider_ref:
---

# Issue 006: Data race on UDS Handlers.streamDone field

## Review Comment

Same pattern as httpapi (issue_004). `setStreamDone()` at line 219 writes to `Handlers.streamDone` without synchronization, but the field is concurrently read by handler goroutines in `streamSession` (line 441), `streamObserveEvents` (line 590), and `promptSession` (line 321).

**Suggested fix:** Pass `streamDone` as a constructor argument to `newHandlers`, or use an `atomic.Value`.

## Triage

- Decision: `invalid`
- Notes: This is the same publication pattern as the HTTP handlers. `streamDone` is set before the UDS server starts serving requests and is not modified after startup, so there is no actual concurrent write/read race in the current code path.
