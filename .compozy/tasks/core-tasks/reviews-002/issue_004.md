---
status: resolved
file: internal/automation/dispatch.go
line: 144
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM564LfH,comment:PRRC_kwDOR5y4QM63o2Ot
---

# Issue 004: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Pass `context.Context` through the task-actor recorder boundary.**

This interface crosses out of the dispatcher into storage/provenance code, but callers cannot propagate cancellation or deadlines because both methods omit `context.Context`. It would be safer to shape this as `Record...(ctx, ...) error` / `Delete...(ctx, ...) error` so implementations do not have to invent background contexts or swallow shutdown signals.


As per coding guidelines, `Use context.Context as first argument to functions crossing runtime boundaries — avoid context.Background() outside main and focused tests`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/dispatch.go` around lines 139 - 144, The interface
AutomationSessionTaskActorRecorder must accept context propagation: change
RecordAutomationSessionTaskActor(sessionID string, actor taskpkg.ActorContext)
error to RecordAutomationSessionTaskActor(ctx context.Context, sessionID string,
actor taskpkg.ActorContext) error and change
DeleteAutomationSessionTaskActor(sessionID string) to
DeleteAutomationSessionTaskActor(ctx context.Context) error (make it return
error so callers can handle failures); then update all implementations and call
sites to pass the inbound ctx through (do not create background contexts) and
handle/propagate returned errors accordingly.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  The `AutomationSessionTaskActorRecorder` boundary is not currently a storage or blocking-runtime boundary. The concrete implementation in `internal/automation/manager.go` only mutates an in-memory map under a mutex, and `DeleteAutomationSessionTaskActor` cannot fail.
  Passing `context.Context` through this interface would add API churn without changing cancellation behavior or fixing a real bug in the current implementation.
  If this recorder later becomes persistent or performs I/O, adding context and error propagation would make sense, but that is not the case today.

## Resolution

- No code change was made. The finding is a design suggestion against an in-memory mutex-protected recorder, not a correctness bug in the current implementation.
