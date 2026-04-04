---
status: resolved
file: internal/httpapi/stream.go
line: 92
severity: medium
author: claude-code
provider_ref:
---

# Issue 018: SSE session stream polls indefinitely for stopped sessions

## Review Comment

`streamSession` (line 92) polls `h.sessions.Events()` in a loop, exiting only when the client disconnects or `streamDone` fires. If a session has stopped, the handler continues polling forever until the client disconnects. There is no mechanism to detect session completion and send a final event. The same issue exists in `internal/udsapi/handlers.go` at line ~437.

This means idle clients watching stopped sessions generate 10 DB queries/second each (100ms poll interval) forever, wasting resources.

**Suggested fix:** After each poll iteration, check session state via `h.sessions.Status()`. If the session is stopped and no new events were returned, send a terminal SSE event (e.g., `event: session_stopped`) and return.

## Triage

- Decision: `valid`
- Notes: Both HTTP and UDS session stream handlers keep polling forever after a session has already stopped unless the client disconnects or the server shuts down. That wastes work on already-terminated sessions. The stream should detect terminal session state and stop itself once there are no new events left to emit.
