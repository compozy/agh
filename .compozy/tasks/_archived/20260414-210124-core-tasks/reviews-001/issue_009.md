---
status: resolved
file: internal/api/udsapi/routes.go
line: 113
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562anu,comment:PRRC_kwDOR5y4QM63mgQ9
---

# Issue 009: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**New task endpoints are not reflected in the canonical OpenAPI registry.**

`RegisterRoutes` now exposes `/api/tasks` and `/api/task-runs`, but the provided `internal/api/spec/spec.go` `Operations()` set does not include those paths. This creates API contract drift for generated docs/clients and schema-based validation.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/udsapi/routes.go` around lines 91 - 113, RegisterRoutes now
exposes /api/tasks and /api/task-runs but the OpenAPI registry returned by
internal/api/spec/spec.go::Operations() is missing those paths; update
Operations() to add path objects and operation entries for all task-related
endpoints (map HTTP methods and operationIds to CreateTask, ListTasks, GetTask,
UpdateTask, CancelTask, CreateChildTask, AddTaskDependency,
RemoveTaskDependency, EnqueueTaskRun, ListTaskRuns and for task-runs:
ClaimTaskRun, StartTaskRun, AttachTaskRunSession, CompleteTaskRun, FailTaskRun,
CancelTaskRun) so the canonical spec matches the handlers and methods defined in
RegisterRoutes.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `internal/api/udsapi/routes.go` now exposes task and task-run endpoints, but `internal/api/spec/spec.go::Operations()` does not register any of them, so the canonical OpenAPI document has drifted from the live transport.
- Fix approach: add the missing task/task-run operations to the spec registry. This requires a minimal out-of-scope edit in `internal/api/spec/spec.go` because there is no in-scope file that owns the canonical OpenAPI registry.

## Resolution

- Added the missing task and task-run operations plus task-domain enums to the canonical OpenAPI registry, and expanded the spec tests to assert the new paths and schemas.
- Regenerated the canonical outputs with `make codegen`, which updated the required out-of-scope generated artifacts `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts`.
- Verified in the final `make verify` run.
