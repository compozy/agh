---
status: resolved
file: internal/extension/host_api.go
line: 625
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QAae,comment:PRRC_kwDOR5y4QM62zlsp
---

# Issue 022: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Wait for prompt completion before looking up the new turn.**

`Prompt` is asynchronous, so the immediate `Events(...AfterSequence...)` query can race and return no user-message yet, producing a false `"prompt turn id not found"` error under load. The detached `go drainAgentEvents(...)` also has no ownership/cancellation path if the producer never closes. As per coding guidelines, "Every goroutine must have explicit ownership and shutdown via `context.Context` cancellation".



Also applies to: 626-637, 963-966

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/host_api.go` around lines 619 - 625, The prompt is started
with context.WithoutCancel and a detached goroutine draining events, causing
race and leak; change promptCtx to context.WithCancel(ctx), start the goroutine
that calls drainAgentEvents(eventsCh) but have it accept/observe ctx and signal
completion (e.g., close a done channel or use a WaitGroup) when eventsCh is
drained, and ensure you cancel the promptCtx on parent ctx.Done to stop the
goroutine if needed; then block (select on done or ctx.Done) to wait for prompt
completion before issuing the Events(...AfterSequence...) lookup so you don't
race and get "prompt turn id not found". Ensure drainAgentEvents is invoked in a
cancellable/owned goroutine and that promptCtx is cancelled when finished.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  `session.Manager.Prompt` persists the user-message event before it returns the event channel, so the immediate `Events(...AfterSequence...)` lookup in `submitPrompt` is not racing the turn-id persistence path. Blocking until prompt completion would also change `sessions/prompt` from an asynchronous submission API into a synchronous run-to-completion API, which is not the current contract.
  The detached drain goroutine is intentional here: once the Host API request returns, someone still needs to consume the prompt stream so the session prompt can continue. Under the current driver contract that channel closes on prompt completion, so this is owned by the prompt lifecycle rather than an unbounded background task.
