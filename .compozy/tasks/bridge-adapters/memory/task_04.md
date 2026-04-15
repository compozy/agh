# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement provider-scoped bridge Host API instance management and ownership-based authorization for `task_04`.
- Completion evidence: focused Go/TypeScript tests are green and `make verify` passed after the Host API contract, handler, and SDK updates.

## Important Decisions
- Added `bridges/instances/list` for provider-owned instance discovery instead of keeping all lookup flows on implicit single-instance runtime state.
- Changed `bridges/instances/get` and `bridges/instances/report_state` to require `bridge_instance_id`; bridge ingest already required it and now shares the same ownership model.
- Extended bridge state reporting to carry optional structured degradation and `clear_degradation`, with the daemon registry applying lifecycle validation and auto-clearing degradation when statuses recover.

## Learnings
- The protocol wire-order test in `internal/extension/protocol/host_api_test.go` must be updated whenever a new Host API method is added, or `make verify` fails late in the Go test phase.
- `internal/extension` package coverage improved with provider-scoped unit tests but still measures below 80% package-wide because the package contains substantial unrelated Host API and manager surface; `make verify` does not enforce a coverage floor.

## Files / Surfaces
- `internal/extension/protocol/host_api.go`
- `internal/extension/contract/host_api.go`
- `internal/extension/host_api.go`
- `internal/extension/host_api_bridges.go`
- `internal/extension/capability.go`
- `internal/bridges/registry.go`
- `internal/extension/host_api_test.go`
- `internal/extension/host_api_integration_test.go`
- `internal/extension/protocol/host_api_test.go`
- `internal/bridges/registry_test.go`
- `sdk/typescript/src/host-api.ts`
- `sdk/typescript/src/host-api.test.ts`
- `sdk/typescript/src/generated/contracts.ts`
- `openapi/agh.json`

## Errors / Corrections
- `make verify` initially failed because `internal/extension/protocol/host_api_test.go` still expected the old Host API wire-order length after adding `bridges/instances/list`; updating the expected method list fixed the failure.
- A new recovery test initially created an `auth_required` bridge instance with `Enabled` left at the zero value; setting `Enabled: true` fixed the invalid lifecycle setup.

## Ready for Next Run
- Task implementation is complete and verified; next run only needs tracking updates and commit handling.
