# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Task 03 by extending the existing `internal/hooks` taxonomy and dispatch runtime with typed autonomy events, then adding a narrow `internal/task` hook bridge consumed by task-run transitions.
- Required proof points: taxonomy/introspection, typed payloads/patch guards, task no-op path, daemon adapter/resource-backed dispatch, post-commit ordering after task audit writes, hooks coverage >=80%, and `make verify`.

## Important Decisions
- Preserve task-domain audit events as immutable `internal/task` records; hook dispatch is co-emitted at manager call sites and never derived by tailing task event tables.
- Treat scheduler wake/no-match/recovery as observability only for this task; do not add scheduler hook events.
- No `internal/api/contract` DTO changes are expected for Task 03, so generated contract/web/docs updates should remain out of scope unless implementation proves otherwise.
- The actual agent-facing safe spawn call site is scheduled for later spawn tasks; Task 03 adds typed spawn payloads/patches/guards and daemon bridge dispatch so later behavior has a first-class extension point.
- Unsafe autonomy hook patches use the existing hook patch rejection path: pre-claim criteria lowering/blank capabilities and spawn permission widening/non-positive TTL are ignored without applying the mutation.
- Generated hook catalog contracts are in scope for Task 03 because the hook enum, matcher shape, and payload/patch schemas are SDK-visible; `make codegen` must be kept in sync with hook taxonomy changes.
- To keep hook declarations cheap to copy and avoid lint suppressions, autonomy-specific matcher fields should be grouped under a nested matcher struct, and spawn permission payloads should avoid embedding large permission sets by value.

## Learnings
- Current hook package already has typed dispatch, patch guards, matchers, introspection descriptors, resource-backed hook binding, and native test executors; autonomy should extend those patterns directly.
- Baseline search shows no existing `HookCoordinator*`, `HookTaskRun*`, `HookSpawn*`, `TaskRunPreClaim`, or `task.run.pre_claim` implementation.
- Baseline `go test ./internal/hooks -cover` passed with 81.5% statement coverage before Task 03 changes.
- Focused hook coverage after final lint corrections is 80.0% (`go test ./internal/hooks -cover`).
- Generated hook contracts now include autonomy payload/patch schemas and nested `HookMatcher.autonomy` correlation fields.

## Files / Surfaces
- Touched: `internal/hooks/events.go`, `internal/hooks/payloads.go`, `internal/hooks/dispatch.go`, `internal/hooks/introspection.go`, `internal/hooks/matcher.go`, `internal/hooks/types.go`, `internal/hooks/async_clone.go`.
- Touched: `internal/task/manager.go`, new `internal/task/hooks.go`, task unit/integration tests.
- Touched: `internal/daemon/hooks_bridge.go`, `internal/daemon/task_runtime.go`, daemon hook binding resource integration tests and fake hook runtime stubs.

## Errors / Corrections
- Initial daemon boot test exposed a typed-nil adapter issue: `bootTasks` now appends `WithTaskRunHooks(state.notifier)` only when the notifier exists.
- Initial hook coverage was 79.4%; added observation dispatch coverage for the new autonomy no-op methods to reach 80.2%.
- Full `make verify` first failed on stale generated contracts; adding autonomy hook contract types to the extension SDK registry allowed `make codegen` and `make codegen-check` to pass.
- Full `make verify` then reached Go lint and failed on `HookDecl`/spawn payload copy size, one `TaskRunHookDispatcher` stutter warning, one `bootTasks` length warning, and two long lines. The correction is structural: shrink copied hook types and rename/refactor, not suppress lint.
- Follow-up lint found `HookDecl` at the exact 512-byte threshold; compacting `HookSource` storage and packing the small `HookDecl` fields fixed remaining `hugeParam` warnings without changing JSON field names.
- Refactoring task manager options briefly reintroduced the typed-nil notifier issue by accepting the hook adapter as an interface; the helper now accepts `*hooksNotifier` and checks nil before appending `WithTaskRunHooks`.

## Ready for Next Run
- Implementation, generated contracts, focused checks, integration checks, hook coverage, `make codegen-check`, `make lint`, `git diff --check`, and full `make verify` are clean.
- Task tracking was updated locally (`task_03.md` status/checklists and `_tasks.md` row), but tracking/memory files were intentionally left out of the automatic code commit per workflow instruction.
- Local code commit created: `57227473 feat: add autonomy hook taxonomy`.
