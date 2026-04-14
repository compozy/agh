---
status: resolved
file: internal/automation/dispatch.go
line: 483
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM564LfM,comment:PRRC_kwDOR5y4QM63o2Oz
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Classify task-service cancellations as `RunCancelled`, not `RunFailed`.**

The session-backed path uses `classifyDispatchError`, but the task-backed path hard-codes `RunFailed` for `CreateTask`/`EnqueueRun` errors. A canceled or timed-out request will therefore be persisted as a failure and can trigger the wrong lifecycle hooks and retry decisions.


<details>
<summary>💡 Suggested fix</summary>

```diff
 	taskRecord, err := d.tasks.CreateTask(ctx, directTaskSpec(req.Job, preFirePrompt), actor)
 	if err != nil {
-		return d.finishRun(ctx, scheduledRun, RunFailed, err)
+		return d.finishRun(ctx, scheduledRun, classifyDispatchError(err), err)
 	}
@@
 	taskRun, err := d.tasks.EnqueueRun(ctx, taskpkg.EnqueueRun{
 		TaskID:         taskRecord.ID,
 		IdempotencyKey: automationTaskRunIdempotencyKey(scheduledRun.ID),
 		NetworkChannel: strings.TrimSpace(taskRecord.NetworkChannel),
 	}, actor)
 	if err != nil {
-		return d.finishRun(ctx, scheduledRun, RunFailed, err)
+		return d.finishRun(ctx, scheduledRun, classifyDispatchError(err), err)
 	}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/dispatch.go` around lines 469 - 483, The task-backed path
currently always records CreateTask/EnqueueRun errors as RunFailed; change it to
classify the error like the session-backed path by calling
classifyDispatchError(err) and passing that status to d.finishRun instead of
hard-coding RunFailed so cancellations/timeouts become RunCancelled; update both
error returns after d.tasks.CreateTask(ctx, ...) and d.tasks.EnqueueRun(ctx,
...) to compute status := classifyDispatchError(err) (or equivalent) and call
d.finishRun(ctx, scheduledRun, status, err) so lifecycle hooks and retry logic
behave correctly.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The task-backed dispatch path hard-codes `RunFailed` for `CreateTask` and `EnqueueRun` failures, while the session-backed path already classifies cancellations and deadlines through `classifyDispatchError`.
  Root cause: cancellation-aware status classification was only applied to the session runtime branch.
  Planned fix: classify both task-service failure sites with `classifyDispatchError(err)` and add regression tests covering cancellation on task creation and run enqueue.

## Resolution

- Updated `internal/automation/dispatch.go` so task-backed `CreateTask` and `EnqueueRun` failures use `classifyDispatchError(err)` instead of always recording `RunFailed`.
- Added regression coverage in `internal/automation/dispatch_test.go` for task-service cancellation and deadline-exceeded cases so canceled work is persisted as `RunCancelled`.
