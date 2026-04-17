---
status: resolved
file: internal/acp/handlers.go
line: 501
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57fTQC,comment:PRRC_kwDOR5y4QM64dqGJ
---

# Issue 008: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**The lazy fallback tool host has no cancellable owner.**

Creating the fallback host on `context.Background()` means the terminal-manager shutdown goroutines it spawns have no context that ever closes on this path. That turns the lazy fallback into a goroutine/resource leak whenever terminals are created before a real host is injected.


As per coding guidelines, `Every goroutine must have explicit ownership and shutdown via context.Context cancellation`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/acp/handlers.go` around lines 487 - 501, The lazy fallback uses
context.Background() so its terminal-manager goroutines never get canceled;
change AgentProcess.toolHostOrDefault to derive a cancellable context from the
AgentProcess lifecycle (e.g., use an existing p.ctx or add a field like
p.ctx/p.cancel or p.toolHostCancel) and pass that context into
newLocalToolHostFromPolicy instead of context.Background(); store the cancel
function (e.g., p.toolHostCancel) when creating the host and ensure you call it
when replacing or shutting down p.toolHost so the spawned goroutines are
properly cancelled and no leak occurs.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `toolHostOrDefault` lazily creates the fallback local tool host on `context.Background()`. That host owns a terminal manager whose goroutines will never see process shutdown on this path. The fix is to bind fallback host creation to the `AgentProcess` lifecycle context and keep that context available on the process.
