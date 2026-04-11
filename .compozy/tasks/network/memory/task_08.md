# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Expose the existing daemon-owned network runtime through shared contracts, UDS handlers, CLI commands, and observability/status surfaces for `status`, `peers`, `spaces`, `send`, and `inbox`.
- Keep the control plane on top of `core.NetworkService`; do not bypass daemon validation by calling transport/router internals directly.
- Preserve optional correlation and AGH workflow/handoff `ext` metadata in surfaced payloads without making them required for v0.

## Important Decisions
- Treat the PRD/techspec as the approved design baseline for this execution task, so implementation can proceed directly after repository/task grounding.
- Build the contract/core conversion layer first so UDS and CLI can share one canonical payload model.
- Keep the control plane on the existing `core.NetworkService` seam; network CLI, HTTP, and UDS layers stay daemon-owned and never call transport/router internals directly.

## Learnings
- `internal/network.Manager` already implements the runtime service surface (`Send`, `ListPeers`, `ListSpaces`, `Status`, `Inbox`), so task 08 is primarily transport/contract/observability work.
- The current codebase already exposes safe network diagnostics in daemon status/info, but not the task-required network-specific CLI/UDS control-plane endpoints.
- Contract changes also require refreshing generated API artifacts; `make verify` enforces this through the stale-OpenAPI check, so `make codegen` must run before the final full gate when the network DTOs change.
- The final touched-package coverage stayed at or above the task target: `internal/api/core` 80.6%, `internal/api/udsapi` 81.7%, `internal/api/httpapi` 82.0%, `internal/cli` 80.0%, and `internal/network` 80.8%.

## Files / Surfaces
- `internal/api/contract/contract.go`
- `internal/api/contract/responses.go`
- `internal/api/core/handlers.go`
- `internal/api/core/network.go`
- `internal/api/udsapi/routes.go`
- `internal/api/httpapi/routes.go`
- `internal/cli/client.go`
- `internal/cli/network.go`
- `internal/daemon/info.go`
- `internal/cli/daemon.go`
- `internal/network/manager.go`
- `internal/network/stats.go`
- `openapi/agh.json`
- `web/src/generated/agh-openapi.d.ts`

## Errors / Corrections
- Initial full verification failed because `openapi/agh.json` was stale after the contract changes; corrected by running `make codegen` and then rerunning `make verify` to a clean pass.

## Ready for Next Run
- Task 08 is implementation-complete and freshly verified. Later tasks can rely on the stable `agh network {status,peers,spaces,send,inbox}` surface, the matching `/api/network/*` daemon endpoints, and the richer runtime status/log metadata for workflow-aware debugging.
