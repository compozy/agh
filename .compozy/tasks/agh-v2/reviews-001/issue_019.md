---
status: resolved
file: internal/httpapi/prompt.go
line: 108
severity: medium
author: claude-code
provider_ref:
---

# Issue 019: SSE prompt errors silently swallowed, events channel not drained

## Review Comment

In `promptSession` (line 108), when the request context is cancelled or the server shuts down, the handler returns immediately without: (1) sending an error SSE event to the client, (2) draining the `events` channel, or (3) calling `state.finish()`. The client receives a truncated SSE stream with no indication of what happened.

If the session manager's `Prompt` spawns a goroutine writing to the channel, that goroutine will block on a send to the undrained channel and leak. The same pattern exists in `internal/udsapi/handlers.go` at line ~317.

**Suggested fix:** Add a deferred drain loop (`for range events {}`) and attempt to send a best-effort error event before returning. Ensure `state.finish()` is always called.

## Triage

- Decision: `invalid`
- Notes: The claimed leak does not occur in the current implementation. `Manager.pumpPrompt()` keeps draining the driver source channel even after the request context is canceled because its send to the client channel is guarded by `select { case out <- ...; case <-ctx.Done(): }`. That means the ACP prompt source is still consumed and the handler does not need an explicit drain loop to avoid blocking the producer. A best-effort terminal SSE on disconnect/shutdown would be cosmetic, not a correctness fix.
