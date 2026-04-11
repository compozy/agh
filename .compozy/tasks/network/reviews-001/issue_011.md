---
status: resolved
file: internal/daemon/daemon.go
line: 520
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBZf,comment:PRRC_kwDOR5y4QM623eZw
---

# Issue 011: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Shut down the API listeners before the network runtime.**

`httpServer` and `udsServer` stay live until Lines 524-533, but the shared network service is torn down here. That creates a shutdown window where `/api/network/*` can race a half-closed runtime and return transient failures instead of cleanly rejecting new work.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/daemon.go` around lines 516 - 520, Move the API listener
shutdowns to occur before tearing down the shared network runtime: call the
shutdown/close logic for httpServer and udsServer (the existing httpServer and
udsServer shutdown/Close calls) and append their errors to errs first, then
invoke network.Shutdown(ctx) and append its error as currently done; ensure you
don't remove the existing error-wrapping (fmt.Errorf("daemon: shutdown network
runtime: %w", err)) so all failures are still collected.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `Daemon.Shutdown` currently tears down the shared network runtime before shutting down the HTTP and UDS listeners, leaving a brief window where API handlers can race a half-closed runtime.
- Fix approach: reorder shutdown so the listeners stop accepting new requests before the network runtime is drained, and update the shutdown-order test accordingly.
