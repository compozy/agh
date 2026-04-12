# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Expose the daemon-owned channel runtime through shared API DTOs plus HTTP and UDS `/api/channels` endpoints for list, create, get, patch, enable, disable, restart, routes, and test-delivery.
- Update the generated OpenAPI spec so downstream clients can consume the new channel surface.
- Meet the task test plan with transport-focused unit/integration coverage and finish with `make verify`.

## Important Decisions
- Added the shared channel transport DTOs in `internal/api/contract/channels.go` and mapped them directly to `internal/channels` create/update/resolve request types instead of introducing transport-local business models.
- Added a transport-independent channel handler layer in `internal/api/core/channels.go` behind the new `core.ChannelService` seam so HTTP and UDS expose the same `/api/channels` behavior.
- Treated `POST /api/channels/:id/test-delivery` as dry-run typed target resolution that returns the resolved `DeliveryTarget`; it does not flatten delivery targets into platform-specific strings.
- Regenerated both `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts` after updating `internal/api/spec/spec.go` so downstream generated consumers stay aligned with the new channel surface.

## Learnings
- The UDS transport must pass `Channels: cfg.channels` into `core.NewBaseHandlers`; missing that wiring makes all channel endpoints return `503 channel service is not configured`.
- Task-surface coverage for `internal/api/{contract,core,httpapi,udsapi,spec}` plus `internal/channels` reached `80.2%` only after adding focused tests for channel error/status mapping and DTO edge cases; package-level totals in older API packages are lower because of unrelated pre-existing surfaces.

## Files / Surfaces
- `internal/api/contract/channels.go`
- `internal/api/contract/channels_test.go`
- `internal/api/core/channels.go`
- `internal/api/core/errors.go`
- `internal/api/core/errors_test.go`
- `internal/api/httpapi/{routes.go,server.go,helpers_test.go,handlers_test.go,channels_test.go,httpapi_integration_test.go,channels_integration_test.go}`
- `internal/api/udsapi/{routes.go,server.go,helpers_test.go,handlers_test.go,server_test.go,channels_test.go,udsapi_integration_test.go,channels_integration_test.go}`
- `internal/api/spec/{spec.go,spec_test.go}`
- `internal/api/testutil/apitest.go`
- `internal/channels/{registry.go,registry_test.go}`
- `internal/daemon/{boot.go,daemon.go}`
- `openapi/agh.json`
- `web/src/generated/agh-openapi.d.ts`

## Errors / Corrections
- Fixed a transport regression where `internal/api/udsapi/server.go:newHandlers()` failed to pass the channel service into `core.NewBaseHandlers`; UDS channel tests caught it immediately via `503` responses on create/get/routes.
- Corrected task-surface coverage by adding focused tests for `contract.ToResolveDeliveryTargetRequest`, `core.StatusForChannelError`, OpenAPI enum helpers, and UDS option wiring instead of loosening the coverage target.

## Ready for Next Run
- Task 09 can extend `internal/cli/client.go` and CLI commands against the shared `/api/channels` contract and generated OpenAPI surface without adding transport-specific channel DTOs.
