# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement executable `native_go` registry, skill, network, and bounded task tools for Task 05 through `tools.Registry.Call`.
- Success requires accurate descriptor/risk metadata, excluded lifecycle tools absent, real service wiring where practical, input validation before service calls, `make verify`, tracking updates, and one local commit.

## Important Decisions
- Keep `internal/tools` domain-neutral: it may own generic native provider support and static built-in descriptors/toolsets, but daemon-owned adapters must import and call `internal/skills`, `internal/network`, and `internal/task`.
- Do not expose task claim/release/complete/fail/run-start/run-complete/run-cancel or skill install/update/remove in this task.
- Wire native providers from daemon boot after hooks/network/tasks are available and store the registry on `RuntimeDeps.ToolRegistry` for later API/CLI tasks.
- Task create adapters only inherit caller `workspace_id` when the requested task scope is `workspace`; global task inputs must not silently inherit a workspace.

## Learnings
- Task 04 already provides central `RuntimeRegistry.Call` dispatch with schema validation, policy/availability recheck, result budgeting/redaction, and events.
- Pre-change signal: no native provider implementation exists yet; `go test ./internal/tools -run TestRuntimeRegistryCallDispatchesRegisteredTool -count=1` passes.
- Focused validation passed:
  - `go test ./internal/tools -count=1`
  - `go test ./internal/daemon -run 'TestDaemonNativeTools|TestDaemonBootToolRegistry' -count=1`
  - `go test ./internal/tools -coverprofile=/tmp/agh-task05-tools.cover -count=1` => 81.7%
  - `go test ./internal/tools ./internal/daemon -count=1`
- Self-review tightened tests so every native built-in has invalid-input coverage, all bounded task service methods are explicitly routed, and `network_peers` is covered alongside `network_send`.
- Final verification passed: `make verify` on 2026-04-28 completed frontend format/lint/typecheck/tests/build, Go lint, 6680 Go tests, build, and package boundaries.
- Post-commit verification also passed: `make verify` completed again after commit `bc8bd3a8` with 6680 Go tests and package boundaries clean.

## Files / Surfaces
- Added `internal/tools/native.go` generic `NativeProvider`.
- Added `internal/tools/builtin.go` native descriptor/toolset catalog for exactly the Task 05 MVP IDs.
- Added `internal/daemon/native_tools.go` daemon adapter handlers for registry, skills, network, and bounded task tools.
- Updated `internal/daemon/boot.go` and `internal/daemon/daemon.go` to compose and publish `RuntimeDeps.ToolRegistry`.
- Added focused tests in `internal/tools/native_test.go`, `internal/tools/builtin_test.go`, and `internal/daemon/native_tools_test.go`.

## Errors / Corrections
- Fixed a malformed intermediate edit that duplicated `taskCreateInput.spec`.
- Removed a duplicate `firstNonEmpty` helper in `native_tools.go`; the daemon package already owns one.
- Added daemon reset handling for `toolRegistry` so shutdown does not block future boots.

## Ready for Next Run
- Current phase: complete. Code-only local commit `bc8bd3a8` created after focused validation, final `make verify`, task tracking updates, self-review, and post-commit `make verify`.
