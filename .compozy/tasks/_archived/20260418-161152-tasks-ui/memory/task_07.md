# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Define the authoritative shared contract and OpenAPI surface for the expanded tasks feature: enriched task list/detail reads, draft publication, task timeline/stream/tree/run detail, observer-backed dashboard/inbox, and approval/triage mutations.
- Regenerate `web/src/generated/agh-openapi.d.ts` from the updated spec and add regression coverage so frontend work starts from generated types instead of manual DTOs.

## Important Decisions
- Keep one shared task contract vocabulary for both HTTP and UDS. Do not fork transport-specific or frontend-only DTOs.
- Extend the OpenAPI spec builder to support explicit non-JSON response media types so the task stream can be documented as `text/event-stream`.
- Define inbox/approval/triage contract types now even though handler and transport wiring land in later tasks.
- Regenerate both frontend and SDK codegen artifacts from the spec so operation IDs and payload shapes stay authoritative outside the Go transport layer.

## Learnings
- Current public contracts only cover the legacy task CRUD/run payloads; none of the task-native live or observer-backed task surfaces are documented yet.
- `internal/task` already provides enriched list/detail and live run/timeline/tree models, while `internal/observe` already provides dashboard aggregates. Inbox/approval transport contracts still need to be defined ahead of implementation.
- Expanding `contract.TaskDetailPayload` is enough to trip `gocritic` `hugeParam` warnings in existing helper code, so downstream CLI/daemon helpers that only render or search detail views should pass the payload by pointer.

## Files / Surfaces
- `internal/api/contract/tasks.go`
- `internal/api/contract/responses.go`
- `internal/api/contract/tasks_test.go`
- `internal/api/spec/spec.go`
- `internal/api/spec/spec_test.go`
- `internal/cli/task.go`
- `internal/cli/task_test.go`
- `internal/daemon/automation_task_e2e_assertions_test.go`
- `internal/daemon/daemon_automation_task_integration_test.go`
- `openapi/agh.json`
- `sdk/typescript/src/generated/contracts.ts`
- `web/src/generated/agh-openapi.d.ts`

## Errors / Corrections
- Pre-change `rg` across `internal/api/spec/spec.go`, `openapi/agh.json`, and `web/src/generated/agh-openapi.d.ts` returned no publish/timeline/stream/tree/run-detail/dashboard/inbox task routes, confirming the contract surface is still missing.
- Full verification surfaced one remaining `unparam` lint on the dedicated `after_sequence` query helper; the fix was to make it an explicit optional-sequence helper instead of carrying an unused `required` argument.
- The larger task detail contract triggered `gocritic hugeParam` findings in pre-existing CLI and daemon helpers; those helpers now accept `*TaskDetailPayload` without changing behavior.

## Ready for Next Run
- Implementation and verification are complete. Remaining closeout work is limited to task tracking, self-review, and the required local commit.
