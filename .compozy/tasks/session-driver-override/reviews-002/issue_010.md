---
status: resolved
file: internal/session/synthetic_prompt.go
line: 157
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59RcPP,comment:PRRC_kwDOR5y4QM6628D4
---

# Issue 010: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Make dispatch release atomic with requeue/drain.**

`finishQueuedSyntheticDispatch(target)` runs before either `failQueuedSyntheticPrompts` or `requeueSyntheticPromptFront`. That opens a race where another caller can claim the next queued item in the gap. On the error path, one item can escape the failure drain; on the prompting path, a newer item can overtake the older claimed item and break FIFO ordering.

<details>
<summary>Proposed direction</summary>

```diff
-		m.finishQueuedSyntheticDispatch(target)
-		m.emitQueuedSyntheticDispatchError(item, err)
-		m.failQueuedSyntheticPrompts(target, err)
+		failed := m.finishAndDrainQueuedSyntheticPrompts(target, item)
+		for _, queued := range failed {
+			m.emitQueuedSyntheticDispatchError(queued, err)
+		}
 		return
 	}
 	if session.IsPrompting() {
-		m.finishQueuedSyntheticDispatch(target)
-		m.requeueSyntheticPromptFront(target, item)
+		m.requeueSyntheticPromptFrontAndFinishDispatch(target, item)
 		return
 	}
```

Both helpers should hold `m.syntheticMu` for the full state transition so no other goroutine can claim the queue in between.
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/synthetic_prompt.go` around lines 150 - 157, The release of
a claimed synthetic dispatch is not atomic with the subsequent state transition,
causing races; ensure the entire sequence that finishes the queued dispatch and
then either drains failures or requeues (the calls to
finishQueuedSyntheticDispatch(target) combined with
failQueuedSyntheticPrompts(target, err) and requeueSyntheticPromptFront(target,
item)) is performed while holding m.syntheticMu so no other goroutine can claim
the next item in the gap; modify the callers to acquire m.syntheticMu before
calling finishQueuedSyntheticDispatch and keep it locked until after calling the
appropriate failQueuedSyntheticPrompts or requeueSyntheticPromptFront, then
release m.syntheticMu.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Decision: `valid`
- Notes: `startNextQueuedSyntheticPrompt` clears `syntheticDispatching` before it requeues or drains the queue, which creates an interleaving window where another goroutine can claim the same session queue and break FIFO/error-drain behavior. I will make the release-and-transition steps atomic under `syntheticMu` and add regression coverage in `internal/session/manager_integration_test.go`, since the scoped files do not currently contain a synthetic-queue race test.
