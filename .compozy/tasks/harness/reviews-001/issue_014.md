---
status: resolved
file: internal/daemon/harness_reentry_bridge.go
line: 168
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-dMy,comment:PRRC_kwDOR5y4QM65IPEG
---

# Issue 014: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don't let a full reentry queue block task state transitions.**

`OnTaskEvent` sends synchronously into a bounded channel. Once `b.events` fills up, the caller blocks here until the bridge drains it, so detached reentry throughput can start stalling terminal task transitions.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/harness_reentry_bridge.go` around lines 164 - 168,
OnTaskEvent currently does a blocking send into the bounded channel b.events
(select on b.ctx.Done / case b.events <- record) which can stall callers when
the queue is full; change the send to a non-blocking send so task state
transitions never block: replace the blocking branch with a select that attempts
case b.events <- record and falls back to default (optionally logging or
metric-ing the dropped record) while still respecting b.ctx.Done. Target the
send logic around b.events in the OnTaskEvent / harness_reentry_bridge.go bridge
code and ensure you do not spin or leak goroutines when dropping events.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `OnTaskEvent` currently performs a blocking send into a bounded `b.events` channel, so a saturated bridge queue can stall the caller that is trying to record a terminal task transition.
  - The bridge should decouple task-state progression from reentry processing while still preserving eventual handling of the terminal run.
  - I will replace the blocking-only handoff with non-blocking overflow signaling backed by a coalesced rescan of durable terminal runs, so task transitions do not block and reentry work is not silently lost.
