---
status: resolved
file: internal/api/httpapi/prompt.go
line: 99
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58iyxw,comment:PRRC_kwDOR5y4QM654NoP
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Detaching the prompt context needs a replacement cancellation path.**

`Prompt` no longer sees request cancellation, but this handler still returns immediately on disconnect/shutdown and stops draining `events`. If the session manager is still producing events, that goroutine can block forever or keep spending work with no consumer. Please cancel or explicitly stop the prompt on every early-exit path here.



Also applies to: 121-137

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/httpapi/prompt.go` around lines 98 - 99, Replace the unmanaged
promptCtx created by context.WithoutCancel with an explicitly cancellable
context and ensure cancel() is invoked on every early-exit path: create ctx,
cancel := context.WithCancel(context.Background()) (or
context.WithCancel(c.Request.Context()) if you want baseline cancellation), pass
ctx into h.Sessions.Prompt instead of promptCtx, defer cancel() in the handler
and call cancel() before any early returns (including the disconnect/shutdown
branches and the goroutine that drains events) so the Prompt invocation is
always signalled to stop when the handler stops consuming events.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the handler detaches prompt execution from request cancellation with `context.WithoutCancel`, but it never installs its own cancellation hook. If the client disconnects or stream setup fails, the handler stops consuming events while the prompt producer can keep running.
- Fix plan: wrap the detached context with `context.WithCancel`, `defer cancel()`, and let every handler exit path cancel the prompt. Update the existing prompt-handler coverage in `internal/api/httpapi/handlers_test.go` to assert the prompt context is explicitly canceled when the request ends.
- Resolution: implemented and verified through targeted Go tests and a clean `make verify` run.
