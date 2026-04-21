# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Expose workspace-scoped provider options directly on `WorkspaceDetailPayload` so task_06 can render a provider picker without re-deriving config state in the web client.
- Keep non-interactive/internal session creation deterministic by making every automatic creator pass `Provider: ""` explicitly.

## Important Decisions
- Build provider options from `workspace.ResolvedWorkspace.Config`, because that is the merged workspace config already used by runtime session creation.
- Publish UI-ready options as sorted `contract.SessionProviderOptionPayload` entries on `WorkspaceDetailPayload.Providers`.
- Treat automatic creator defaults as an explicit contract: internal session creators now pass `Provider: ""` instead of relying on omitted zero values.
- Use changed-surface coverage for the task coverage target because `internal/daemon` package coverage includes large unrelated runtime areas; touched implementation files measure `83.4%` coverage (`1787/2142` statements).

## Learnings
- The resolved workspace config can expose builtin and overlay providers together; the backend can hand the web client a ready-to-render provider list without a separate discovery endpoint.
- `session.Session.Info()` cannot produce a non-nil session with a nil info snapshot, so the `created.Info() == nil` guard in `taskSessionBridge.StartTaskSession` is effectively defensive-only.
- Shared transport test alias files (`internal/api/httpapi/shared_test.go`, `internal/api/udsapi/shared_test.go`) can fall out of sync with contract usage and break the strict lint gate even when production logic is correct.

## Files / Surfaces
- `internal/api/contract/contract.go`
- `internal/api/core/{conversions.go,workspaces.go,memory_workspace_test.go,network_details.go,network_test.go,session_workspace_internal_test.go}`
- `internal/api/{httpapi/handlers_test.go,httpapi/shared_test.go,udsapi/handlers_test.go,udsapi/shared_test.go}`
- `internal/automation/{dispatch.go,dispatch_test.go}`
- `internal/daemon/{task_runtime.go,task_runtime_test.go,daemon_integration_test.go}`
- `internal/extension/{host_api_bridges.go,host_api_test.go}`
- `internal/memory/consolidation/{runtime.go,runtime_test.go}`
- `openapi/agh.json`
- `web/src/generated/agh-openapi.d.ts`

## Errors / Corrections
- `make verify` initially failed on two unused test aliases in `internal/api/httpapi/shared_test.go` and `internal/api/udsapi/shared_test.go`; removed those aliases and reran the full gate successfully.

## Ready for Next Run
- Task_06 can consume `WorkspaceDetailPayload.providers` directly from the workspace detail response; no frontend-side provider inference should be added.
- Automatic internal creators across automation, daemon task runtime, memory consolidation, network details, and host bridges now intentionally request the agent default provider with `Provider: ""`.
- Fresh verification evidence:
  - `make codegen-check`
  - `make verify`
  - touched implementation coverage: `83.4%` (`1787/2142` statements)
