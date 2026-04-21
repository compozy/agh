---
status: resolved
file: internal/task/manager.go
line: 2582
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-dM_,comment:PRRC_kwDOR5y4QM65IPEU
---

# Issue 028: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Decouple post-commit notifications from the request context.**

At this point `CreateTaskEvent` has already succeeded, so this read/observer/live path is best-effort follow-up work. Reusing the caller `ctx` means a disconnect or deadline right after the write can make `GetTaskEventRecord` fail and silently skip the detached-harness observer even though the event is already committed. Run this notification path on a non-cancelable post-commit context instead.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/task/manager.go` around lines 2574 - 2582, The post-commit
notification code uses the caller ctx; change it to run on a non-cancelable
post-commit context so transient caller cancellations or deadlines don't prevent
best-effort follow-up work. After CreateTaskEvent returns success, create a
detached context (e.g. context.Background() or a derived background ctx) and use
that when calling m.store.GetTaskEventRecord, m.eventObserver.OnTaskEvent,
m.emitTaskLiveRecordBestEffort and m.emitTaskLiveEventBestEffort; run this
notification path asynchronously (goroutine) or sequentially but using the
detached ctx and keep the same error handling/semantics so failures remain
best-effort.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: after `CreateTaskEvent` commits successfully, `recordTaskEvent` still reuses the caller context for `GetTaskEventRecord`, observer fan-out, and live emission. A caller cancellation at that point can suppress the best-effort post-commit notifications even though the write already succeeded.
- Fix approach: switch the post-commit read/observer/live path to a non-cancelable context derived after the durable write so the follow-up notification path is detached from transient caller cancellation.
