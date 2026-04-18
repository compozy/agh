---
status: resolved
file: internal/daemon/harness_reentry_bridge.go
line: 706
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-dM1,comment:PRRC_kwDOR5y4QM65IPEJ
---

# Issue 016: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Make the synthetic wait loop cancellable.**

`awaitSyntheticWake` waits forever for the ACP channel to close and never checks `b.ctx.Done()`. If the synthetic stream hangs during shutdown, this goroutine never exits and the run stays unfinalized in the processing set.

As per coding guidelines, "Every goroutine must have explicit ownership and shutdown via context.Context cancellation" and "Use select with ctx.Done() in all long-running goroutine loops".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/harness_reentry_bridge.go` around lines 667 - 706, The
awaitSyntheticWake method in harnessReentryBridge currently loops over events
without observing b.ctx.Done(), so make the loop cancellable by replacing the
range over events with a select that listens for b.ctx.Done() and the events
channel; on ctx cancellation exit the loop and treat as a dispatch failure (set
sawError or choose harnessReentryOutcomeDropped with
harnessReentryReasonDispatchFailed) before calling syntheticEventExists and
finalizeRunOutcome; retain the existing logic around detecting EventTypeError,
call syntheticEventExists(item.targetSessionID, item.runID) and then
finalizeRunOutcome(item.runID, item.targetSessionID, item.targetAgentName, ...)
as before so the run is always finalized even if the context is canceled.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `awaitSyntheticWake` ranges the event channel without observing `b.ctx.Done()`, so a hung synthetic stream can keep the goroutine alive through shutdown.
  - In that state the run may never be finalized as dropped, which leaves reentry bookkeeping inconsistent.
  - I will make the wait loop cancellable and use shutdown-safe finalization so in-flight synthetic wake handling still records a dropped outcome when the bridge is canceled.
