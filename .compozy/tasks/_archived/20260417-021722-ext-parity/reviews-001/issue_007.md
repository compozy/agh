---
status: resolved
file: internal/acp/handlers.go
line: 390
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57fTQA,comment:PRRC_kwDOR5y4QM64dqGG
---

# Issue 007: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Thread the inbound request context through terminal creation.**

This path hardcodes `context.Background()` for both local and injected tool hosts, so `terminal/create` cannot honor request cancellation or deadlines. `handleCreateTerminal` should take the inbound `ctx` and pass it through here.


As per coding guidelines, `Use context.Context as first argument to functions crossing runtime boundaries — avoid context.Background() outside main and focused tests`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/acp/handlers.go` around lines 381 - 390, The code uses
context.Background() when creating terminals which prevents honoring request
cancellations/deadlines; update the function that contains this snippet (and its
callers, e.g., handleCreateTerminal) to accept a context.Context parameter (use
ctx as the first argument) and replace both context.Background() calls with that
inbound ctx when invoking localToolHost.createTerminal and host.CreateTerminal;
ensure you preserve the existing ownership flow and keep
p.recordTerminalOwnership(response.TerminalId, ownership) unchanged.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `handleCreateTerminal` still uses `context.Background()` for both local and injected tool hosts, even though the inbound ACP request already carries a context. That drops request cancellation and deadlines on terminal creation. The fix is to thread the inbound `ctx` through `handleInbound` into `handleCreateTerminal`.
