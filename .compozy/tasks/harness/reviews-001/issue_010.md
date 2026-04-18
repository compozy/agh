---
status: resolved
file: internal/daemon/daemon_integration_test.go
line: 646
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-dMu,comment:PRRC_kwDOR5y4QM65IPEC
---

# Issue 010: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Replace the FIFO sleep with an observable barrier.**

Line 645 uses `time.Sleep(20 * time.Millisecond)` to order the two completions. That makes this integration test timing-sensitive under CI load and can still reorder intermittently. Wait for the first wake/reentry signal to be recorded before completing the second run instead of sleeping.

As per coding guidelines, "Never use time.Sleep() in orchestration — use proper synchronization primitives".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/daemon_integration_test.go` around lines 644 - 646, Replace
the brittle time.Sleep by waiting for an observable signal that the first run
has been processed: after calling completeDetachedHarnessRunForTest(t,
daemonInstance.tasks, first.Run.ID, "sess-owner"), block until the daemon
records the wake/reentry for first.Run.ID (e.g., poll or select on the task
event channel, inspect daemonInstance.tasks for a recorded wake/reentry event,
or use a wait channel/WaitGroup you add to the harness) and only then call
completeDetachedHarnessRunForTest for second.Run.ID; reference the existing
symbols completeDetachedHarnessRunForTest, daemonInstance.tasks, first.Run.ID
and second.Run.ID to locate where to replace the sleep with the synchronization
check.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The integration test currently uses `time.Sleep(20 * time.Millisecond)` to create ordering between two detached-run completions.
  - That is explicitly brittle under CI load and the repo guidance forbids sleep-based orchestration when an observable synchronization point exists.
  - The test already exposes a concrete barrier via `sessions.syntheticPromptCount()`, so I will replace the sleep with condition-based waiting for the first wake dispatch before completing the second run.
