---
status: resolved
file: internal/drivers/codex/codex.go
line: 474
severity: medium
author: claude-reviewer
---

# Issue 151: copyOutput goroutine has no context-based cancellation



## Review Comment

The `copyOutput` goroutine in all three drivers runs in an infinite loop with no context-based cancellation:

```go
func (d *CodexDriver) copyOutput(proc *kernel.AgentProcess) {
    buf := make([]byte, 4096)
    for {
        n, err := proc.Read(buf)
        if n > 0 {
            _, _ = proc.OutputBuffer.Write(buf[:n])
        }
        if err != nil {
            proc.NotifyEOF(err)
            if errors.Is(err, io.EOF) || errors.Is(err, os.ErrClosed) {
                return
            }
            return
        }
    }
}
```

This goroutine is launched at line 228 (codex), 314 (opencode), and 241 (pi) with `go d.copyOutput(proc)` and has no way to be cancelled via context. It relies entirely on `proc.Read` returning an error (EOF or ErrClosed) to terminate.

Per the project's concurrency discipline: "Every goroutine must have explicit ownership and shutdown via `context.Context` cancellation" and "Use `select` with `ctx.Done()` in all long-running goroutine loops."

While the PTY close during `Stop` will eventually cause `Read` to return an error, if the PTY close fails or is delayed, this goroutine will hang. The goroutine also has no tracking via `sync.WaitGroup` or equivalent.

This pattern is duplicated in all three drivers:
- `internal/drivers/codex/codex.go:474`
- `internal/drivers/opencode/opencode.go:933`
- `internal/drivers/pi/pi.go:496`

**Suggested fix**: Accept a context parameter and use a select-based read pattern, or at minimum track the goroutine with a WaitGroup so Stop can wait for it to complete.

## Triage

- Decision: `invalid`
- Notes:
  - The goroutine does not accept an explicit context, but shutdown ownership is still explicit: `Stop` closes the PTY first, and `Process.ClosePTY()` unblocks the blocking `Read`, which then calls `NotifyEOF` and exits.
  - I did not find a reproducible leak or hung shutdown path in the current `internal/pty.Process` implementation. `MockPty.Read` and the real PTY both unblock on close, which is the actual lifecycle signal for this loop.
  - Making the read loop context-aware would require a broader runtime API change because `Process.Read` is a blocking PTY read without cancellable semantics. That is architecture work beyond this scoped bug-fix batch.
  - Resolution: closed as not actionable within the current driver/runtime API.
