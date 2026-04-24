---
status: resolved
file: internal/api/udsapi/prompt.go
line: 78
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59dk0_,comment:PRRC_kwDOR5y4QM67HMWD
---

# Issue 007: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Give the detached drain goroutine an explicit shutdown bound.**

This is now a fire-and-forget goroutine, and `cancelPrompt()` is deferred until `events` closes. If the prompt stream wedges, the goroutine and detached prompt can survive indefinitely after the client disconnects. Tie the drain to a bounded context/timeout and track it with handler shutdown.

As per coding guidelines, "Every goroutine must have explicit ownership and shutdown via `context.Context` cancellation" and "No fire-and-forget goroutines — track with `sync.WaitGroup` or equivalent".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/udsapi/prompt.go` around lines 73 - 78, The detached goroutine
in Handlers.drainPromptEventsAsync currently fires-and-forgets and only defers
cancelPrompt(), which can hang if events never close; change it to create a
bounded child context (e.g., context.WithTimeout or context.WithCancel derived
from the handler's lifecycle context) and register the goroutine with the
handler's shutdown tracking (e.g., h.wg.Add(1) / h.wg.Done() or equivalent) so
the goroutine is owned and can be cancelled on handler shutdown; inside the
goroutine select on the child ctx.Done() and the events channel and ensure
cancelPrompt() is called and wg.Done() on exit, and remove any untracked
anonymous spawn so drainPromptEvents is not left running indefinitely.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - `internal/api/udsapi/prompt.go` starts a detached goroutine in `drainPromptEventsAsync` without local ownership tracking and without a timeout bound if the events channel wedges.
  - The fix is to make the drain goroutine owned by the UDS handlers with a wait group, tie it to stream shutdown, and add a bounded drain context that cancels the prompt on timeout or shutdown.
