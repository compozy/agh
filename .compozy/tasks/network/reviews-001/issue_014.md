---
status: resolved
file: internal/network/delivery.go
line: 325
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBZh,comment:PRRC_kwDOR5y4QM623eZy
---

# Issue 014: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Requeued deliveries can become permanently stuck after a prompt/render failure.**

Both failure branches put the envelope back at the front and then return. Once the deferred `deliveries.Delete()` runs, nothing retriggers a worker for that session, so the inbox can stay non-empty until some unrelated accept/turn-end happens. Please either keep the worker alive for retryable failures or explicitly retrigger after cleanup when the queue is still non-empty.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/delivery.go` around lines 302 - 325, The two failure
branches (formatNetworkMessage and c.prompter.PromptNetwork) requeue the
envelope with c.requeueFront and then return, but the deferred deliveries.Delete
can leave the session without a worker; update both branches to, after
c.clearInFlight(target) and c.requeueFront(target, item), explicitly retrigger
delivery processing instead of returning: call a new helper (e.g.
c.enqueueDeliveryWorker(target) or similar) that checks the deliveries queue for
the session and starts/schedules a worker if the queue is non-empty; add that
helper and use it from both failure branches so requeued envelopes won’t become
permanently stuck after the deferred deliveries.Delete runs.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: when rendering or prompting fails, the worker requeues the envelope and returns. The deferred `deliveries.Delete` removes the active worker state, but nothing retriggers delivery for that still-non-empty queue.
- Fix approach: add an explicit retry scheduling path that retriggers delivery once the current worker has exited and the queue still has work.
