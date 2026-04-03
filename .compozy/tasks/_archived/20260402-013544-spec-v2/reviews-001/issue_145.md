---
status: resolved
file: internal/drivers/opencode/opencode.go
line: 774
severity: high
author: claude-reviewer
---

# Issue 145: OpenCode startEventStream uses context.Background instead of parent context



## Review Comment

The `startEventStream` method creates a new context from `context.Background()` rather than using the parent context from `Start`:

```go
func (d *Driver) startEventStream(proc *kernel.AgentProcess, env []string) {
    if proc == nil || d.hookForwarder == nil {
        return
    }

    ctx, cancel := context.WithCancel(context.Background())

    go func() {
        defer cancel()
        _ = proc.Wait()
    }()

    go func() {
        select {
        case <-proc.EOF():
            cancel()
        case <-ctx.Done():
        }
    }()

    go d.streamEvents(ctx, proc, env)
}
```

This means the SSE event stream goroutine is not tied to the lifecycle of the caller's context. If the caller cancels the context passed to `Start`, the event stream will continue running independently until the process exits or its EOF channel signals.

While the goroutine does have shutdown paths (proc.Wait() and proc.EOF()), it violates the project's concurrency discipline rule: "Every goroutine must have explicit ownership and shutdown via `context.Context` cancellation." The parent context should flow through to the event stream.

Additionally, the `Start` method itself doesn't store the cancel function anywhere accessible, which means there's no way to explicitly stop the event stream from outside without stopping the entire process.

**Suggested fix**: Pass the parent context (or a derived context stored on the driver/process) into `startEventStream` so that the event stream respects the caller's lifecycle. Store the cancel function so it can be called during `Stop`.

## Triage

- Decision: `valid`
- Notes: Confirmed in `internal/drivers/opencode/opencode.go`: `startEventStream` creates its own cancellation tree from `context.Background()`, so event streaming ignores the caller context used by `Start`. That breaks goroutine ownership and should be fixed by deriving the stream context from the parent startup context.
