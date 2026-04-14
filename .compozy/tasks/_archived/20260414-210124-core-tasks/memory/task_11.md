# Task Memory: task_11.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add capability-gated extension Host API methods for task list/create/get/update/cancel plus task-run list/enqueue/claim/start/attach/complete/fail/cancel, and preserve trusted extension-derived actor/origin metadata.
- Verification must include denied-access and trusted-identity unit coverage, dedicated-session execution integration coverage, package coverage at or above 80%, and clean repository verification.

## Important Decisions
- Reuse shared task request/response contracts and conversion helpers from `internal/api/contract` and `internal/api/core/tasks.go` instead of inventing extension-only task payload types.
- Use `task.DeriveExtensionActorContext` at the Host API ingress so payload-supplied identity fields remain ignored and immutable identity/origin come from extension context.
- Keep task lifecycle authority in `internal/task.Manager`; the Host API only delegates lifecycle requests and maps domain errors into RPC-safe responses.
- Keep scope to create/update/query/run flows from the task spec; do not add extension Host API dependency/child mutation methods that were not required by task 11 deliverables.

## Learnings
- The Host API test harness already boots real `globaldb`, `session.Manager`, `observe`, `automation`, and workspace resolution, so it can support true task/session integration once a task manager is wired in.
- Current Host API protocol/contract registries have no task methods yet; adding them requires touching protocol constants, contract specs, capability mapping, and handler method registration together.
- Adding Host API methods changes generated API artifacts; `make codegen` was required to refresh `openapi/agh.json` and `sdk/typescript/src/generated/contracts.ts`.
- Package-level coverage for `internal/extension` stayed below the workspace floor until `manager_test.go` covered option wiring and reload behavior alongside the new Host API task tests.

## Files / Surfaces
- `internal/extension/protocol/host_api.go`
- `internal/extension/contract/host_api.go`
- `internal/extension/capability.go`
- `internal/extension/host_api.go`
- `internal/extension/host_api_tasks.go`
- `internal/extension/host_api_test.go`
- `internal/extension/host_api_integration_test.go`
- `internal/extension/manager_test.go`
- `internal/daemon/daemon.go`
- `internal/daemon/boot.go`
- `openapi/agh.json`
- `sdk/typescript/src/generated/contracts.ts`

## Errors / Corrections
- Initial `make verify` hit a transient pre-existing automation test failure in `internal/automation` (`TestSchedulerAtJobUnregistersAfterFiringOnce`); isolated rerun passed and the final full `make verify` passed cleanly.
- An early task-memory note mentioned dependency management, but task 11 implementation intentionally stayed within the specified create/update/query/run scope.

## Ready for Next Run
- Task 11 implementation is complete: extension Host API now exposes capability-gated task create/query/update/run flows, derives actor/origin from trusted extension context, and keeps executable subtask starts on the dedicated-session TaskManager path.
- Verification evidence:
  - `go test ./internal/extension -count=1`
  - `go test ./internal/extension -cover -count=1` (`coverage: 80.0% of statements`)
  - `go test -tags integration ./internal/extension -count=1`
  - `make verify`
