# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Task 08: add five `sandbox.*` hook events/payloads/dispatchers/matchers, dispatch them from session sandbox prepare/sync/stop lifecycle points, add `sandbox/list`, `sandbox/info`, and `sandbox/exec` Host API methods with capability gating, and verify with tests plus `make verify`.

## Important Decisions
- `sandbox/exec` should run through the active session's environment `ToolHost` terminal interface, preserving provider abstraction for both local and Daytona environments.
- Sync hooks should propagate `exclude_patterns` and stats through provider-facing sync options/results rather than deriving behavior from logs.
- `sandbox/list` and `sandbox/info` require the manifest action grant only; `sandbox/exec` requires both the action grant and `sandbox.exec` security capability.

## Learnings
- Shared workflow memory says Task 04 already integrated session sandbox lifecycle and Task 07 completed boot-time sandbox reconciliation. Task 08 should build on those surfaces without adding a separate sandbox-specific extension registry.
- TechSpec step 13 requires routing extension delivery through the canonical hook runtime and registering Host APIs through the shared extension surfaces/grant model.
- Baseline implementation gap confirmed: `internal/` currently has no sandbox hook events/dispatchers and no `sandbox/list`, `sandbox/info`, `sandbox/exec`, or `sandbox.exec` Host API mapping.
- Hook runtime extensions require updating events, payloads, matcher, dispatch, introspection, and SDK contract registration together; missing introspection/SDK roots can leave runtime behavior working but extension contract discovery incomplete.
- Session sandbox sync now has to expose explicit options/results so `sandbox.sync.before` can pass `exclude_patterns` to providers and `sandbox.sync.after` can report files/bytes/error counts without parsing logs.

## Files / Surfaces
- Main touched surfaces: `internal/hooks`, `internal/session`, `internal/extension/contract`, `internal/extension/protocol`, `internal/extension/capability`, `internal/daemon`.
- Additional touched surfaces: `internal/sandbox` provider sync contracts/providers/tests and `internal/acp` because Host API `sandbox/exec` needs access to the prepared session `ToolHost`.

## Errors / Corrections
- Direct `EnvironmentHooks` implementations must be treated as authoritative too: `session.Manager` explicitly aborts on a returned prepare payload with `Denied=true`, not only on canonical hook runtime errors.
- `sandbox.stop` denial prevents provider destroy while still allowing session stop; tests assert the session reaches stopped state and the provider destroy count remains unchanged.
- The daemon Host API session adapter preserves the existing `api/core.SessionManager` surface and only exposes `ExecEnvironment` when the underlying manager implements it.
- Native-hook integration tests use a single async worker for deterministic event order while still exercising canonical async delivery.
- A coverage-instrumented session test exposed a scheduling-sensitive one-second timeout; widened the assertion timeout to the existing `defaultLifecycleTimeout` instead of weakening the behavior assertion.
- Targeted verification passed:
  - `go test ./internal/hooks ./internal/session ./internal/daemon`
  - `go test ./internal/hooks ./internal/session ./internal/extension/protocol ./internal/extension/contract ./internal/extension`
  - `go test -tags integration ./internal/session -run TestManagerIntegrationEnvironmentNativeHooksLifecycleOrder -count=1`
  - `go test ./internal/acp ./internal/sandbox/... ./internal/hooks ./internal/session ./internal/extension/... ./internal/daemon`
- Coverage gate passed for the task-critical packages: `go test -cover ./internal/hooks ./internal/session ./internal/extension` reported hooks `81.6%`, session `80.9%`, and extension `80.0%`.
- Final full gate passed after implementation and self-review: `make verify` completed with web tests, lint, Go race tests, build, and boundary checks green.
- Created local source-only commit `457d4d64` (`feat: add sandbox extension hooks`); task tracking and memory files remain unstaged per workflow rules.
- Post-commit verification passed:
  - `make verify`
  - `go test -cover ./internal/hooks ./internal/session ./internal/extension`
  - `go test -tags integration ./internal/session`

## Ready for Next Run
- Task 08 implementation, validation, self-review, tracking updates, and source commit are complete. Remaining worktree changes are unrelated prior sandbox/Daytona/tracking changes plus unstaged task memory/tracking updates.
