---
status: resolved
file: internal/acp/handlers.go
line: 424
severity: medium
author: claude-code
provider_ref:
---

# Issue 011: Terminal goroutines not tracked during shutdown

## Review Comment

`terminalManager.create` launches `go term.wait()` at line 424, but neither `closeAll()` nor `release()` waits for these goroutines to complete. `closeAll` kills processes but returns immediately. This violates the project rule: "No fire-and-forget goroutines — track with sync.WaitGroup or equivalent." During daemon shutdown, terminal wait goroutines may be orphaned, producing log noise or resource leaks.

```go
go term.wait() // not tracked by any WaitGroup
```

**Fix:** Add a `sync.WaitGroup` to `terminalManager`, increment in `create`, decrement at the end of `term.wait()`, and call `wg.Wait()` at the end of `closeAll()`.

## Triage

- Decision: `invalid`
- Analysis: Terminal wait goroutines are bound to individual child processes and `closeAll` forcibly terminates those processes before the owning ACP process is considered done. Once the kill completes, `cmd.Wait` returns and each terminal goroutine exits naturally.
- Analysis: There is no daemon-visible API that returns before the terminal processes have been asked to terminate, and the current tests do not expose a persistent resource leak after `closeAll`.
- Conclusion: Tracking the goroutines with a wait group would be stylistic hardening, but it is not a demonstrated correctness bug in the current shutdown flow.
