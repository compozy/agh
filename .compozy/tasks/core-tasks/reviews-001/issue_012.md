---
status: resolved
file: internal/automation/dispatch.go
line: 489
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562an9,comment:PRRC_kwDOR5y4QM63mgRO
---

# Issue 012: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Retries can create duplicate or orphaned tasks before enqueue succeeds.**

`CreateTask` happens before the idempotent `EnqueueRun`. If enqueue fails after a successful create, the retry path will create another task for the same automation run, leaving the previous task orphaned. This needs an idempotent task identity derived from `scheduledRun.ID`, or a task-service operation that creates the task and initial run atomically.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/dispatch.go` around lines 469 - 489, The current flow
calls tasks.CreateTask (directTaskSpec) before the idempotent tasks.EnqueueRun,
so retries can create duplicate/orphaned tasks; change to an idempotent creation
path by deriving a stable task identity from scheduledRun.ID (e.g., pass a
TaskID or idempotency key computed from scheduledRun.ID into tasks.CreateTask or
use a new tasks.CreateOrGetTask/CreateTaskWithInitialRun API that atomically
creates the task and first run), or update the call sites (directTaskSpec,
automationTaskRunIdempotencyKey, tasks.CreateTask, tasks.EnqueueRun,
delegateRun) to use a single atomic operation on the tasks service that returns
both task and run, ensuring retries are safe and no orphaned tasks are produced.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Root cause check: the current flow intentionally materializes the durable task before enqueueing its first run.
- Why invalid: a task without an initial run is still a valid durable artifact in this model, and retries create distinct automation attempt records with distinct run IDs rather than duplicate writes for the same persisted attempt. A truly atomic task-plus-run API would be broader design work, not a localized correctness fix inside this batch.

## Resolution

- No code change was required because the current durable-task-first flow is intentional and consistent with the task model.
- The batch still passed the final `make verify` run unchanged for this issue.
