# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Expose the daemon-owned automation manager through shared API contracts and handlers, wire HTTP/UDS automation routes with HTTP-only webhook delivery, add automation health data to `GET /api/observe/health`, and regenerate OpenAPI/web type artifacts with required tests.

## Important Decisions
- Use the task spec + techspec as the approved design baseline for this PRD execution task.
- Reuse the existing daemon-owned automation manager from task 06; keep ownership enforcement, decoding, and webhook rejection inside `internal/api/core` rather than per transport.
- Preserve config-backed ownership rules in handlers: config-sourced jobs/triggers may toggle `enabled`, but definition edits and deletes remain rejected.
- Dynamic webhook trigger creation now backfills a stable `webhook_id` inside `internal/automation.Manager` before runtime registration, so API callers may create webhook triggers with `endpoint_slug` + secret and still receive a stable endpoint suffix.

## Learnings
- The current `core.AutomationManager` surface is too narrow for task 07: it exposes list/status/overlay/webhook methods only, while the store and runtime already have enough primitives to support CRUD/get/run-history APIs.
- HTTP and UDS transports currently accept sessions/observer/workspaces/memory/dream dependencies but do not yet receive automation, so server constructors and daemon server factories must be extended.
- `observe.Health` does not currently carry automation status; the additive automation block will need to be composed in the API layer response.
- Runtime webhook registration requires both a secret and a stable `webhook_id`; leaving `webhook_id` empty on dynamic webhook triggers caused a real 500 path that had to be fixed in production code rather than papered over in tests.
- Session lifecycle trigger matching uses automation scope plus exact filter paths (`kind`, `scope`, `workspace_id`, `source`, `data.*`). Workspace-bound session triggers in transport tests need the canonical workspace ID and `data.session_type`, not ad hoc filter keys.

## Files / Surfaces
- `internal/api/contract/*`
- `internal/api/core/*`
- `internal/api/httpapi/*`
- `internal/api/udsapi/*`
- `internal/api/spec/*`
- `internal/automation/manager.go`
- `internal/daemon/*`
- `openapi/agh.json`
- `sdk/typescript/src/generated/contracts.ts`
- `web/src/generated/agh-openapi.d.ts`

## Errors / Corrections
- Fixed a manager/runtime gap where creating a dynamic webhook trigger with only `endpoint_slug` returned HTTP 500 because runtime registration required `webhook_id`.
- Corrected the UDS trigger integration path to use a workspace-scoped trigger with `data.session_type = "user"` after resolving the canonical workspace ID.

## Ready for Next Run
- Task 07 implementation is complete. Verification evidence:
  - `go test ./internal/api/core ./internal/api/contract ./internal/api/spec -cover`
  - `go test -tags integration ./internal/api/httpapi ./internal/api/udsapi -count=1`
  - task-scoped coverage: `internal/api/core/automation.go` 80.3% via `-coverpkg`, `internal/api/spec` 86.2%, `internal/api/contract` 100%
  - `make verify`
- Follow-on work is in downstream tasks only: task 08 can consume the new HTTP/UDS automation management surface, and task 10 can build on the generated automation OpenAPI/web types.
