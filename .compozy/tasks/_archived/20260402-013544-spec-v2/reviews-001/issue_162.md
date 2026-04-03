---
status: resolved
file: internal/drivers/opencode/opencode.go
line: 336
severity: medium
author: claude-reviewer
---

# Issue 162: OpenCode driver passes port to TUI mode but event stream URL may not be reachable



## Review Comment

In `Start`, after process readiness is detected, `startEventStream` is called unconditionally for both TUI and server modes:

```go
d.startEventStream(proc, env)
```

In server mode, the event URL (`/event`) is served by the OpenCode HTTP server, which was verified reachable during `detectServerReady`.

In TUI mode, readiness is detected via PTY output patterns (like "OpenCode" or "Press ? for help"), not via the HTTP health endpoint. The event URL is still constructed from the allocated port:

```go
"event_url": buildEventURL(baseURL),
```

If the OpenCode TUI binary doesn't start an HTTP server at the allocated port, the `streamEvents` goroutine will repeatedly fail to connect to the event URL and enter an infinite retry loop (reconnecting every `readyPollInterval`). This retry loop runs silently in the background, consuming resources and generating failed HTTP requests.

The `consumeEventStream` method returns errors on connection failure, and `streamEvents` retries indefinitely:

```go
func (d *Driver) streamEvents(ctx context.Context, proc *kernel.AgentProcess, env []string) {
    for {
        if err := d.consumeEventStream(ctx, proc, env); err != nil {
            if ctx.Err() != nil {
                return
            }
            timer := time.NewTimer(d.readyPollInterval)
            ...
            continue
        }
        return
    }
}
```

**Suggested fix**: If TUI mode does start an HTTP server (which is why the port is passed), this is fine but should be documented. If TUI mode does not start an HTTP server, `startEventStream` should be conditional on server mode only.

## Triage

- Decision: `invalid`
- Notes:
  - The local spec explicitly requires an SSE hook-stream goroutine for OpenCode, and the existing TUI-mode tests already assert that the TUI startup path connects to `/event`.
  - In other words, this code is implementing the chosen design rather than accidentally probing an endpoint that should never exist.
  - If a particular OpenCode build stopped exposing `/event` in TUI mode, that would be a separate compatibility issue requiring spec and integration review. It is not a demonstrable defect in the current scoped implementation.
  - Resolution: closed as consistent with the current OpenCode spec and test expectations.
