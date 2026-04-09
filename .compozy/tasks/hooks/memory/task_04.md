# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement task_04's generic sync pipeline, dispatch depth guard, permission deny-only enforcement, and unit tests in `internal/hooks`.
- Pre-change baseline: `internal/hooks` has ordering, matcher, normalization, payloads, and executors, but no `pipeline.go`, `depth.go`, or `permission.go`, and no tests for pipeline/guard behavior.

## Important Decisions
- The approved design is already captured by `_techspec.md` and ADR-005/006/007/009/012, so implementation can proceed directly without reopening design.
- Keep the pipeline centered on `ResolvedHook` plus the existing executor contract so task_06 can wrap it without back-importing other packages.
- Select and sort the hook snapshot once per dispatch, then run the sequential pipeline against that fixed order while each hook still sees the patched payload from earlier hooks.
- Treat permission deny→allow attempts as rejected patches instead of hook failures so the original deny stands and later hooks still see the unchanged denied payload.

## Learnings
- `PermissionRequestPayload` carries both `Decision` and `DecisionClass`, while `PermissionRequestPatch` can mutate both, so deny-only enforcement needs to validate the effective decision after every patch.
- Native executors currently still implement the byte-based `Executor` interface, so the pipeline will need an explicit typed bypass to satisfy the task requirement that native callbacks avoid serialization.
- Per-hook timeout enforcement needs to happen in the pipeline as well as inside subprocess execution so required native hooks can time out cleanly on `ctx.Done()`.

## Files / Surfaces
- `internal/hooks/types.go`
- `internal/hooks/payloads.go`
- `internal/hooks/ordering.go`
- `internal/hooks/executor.go`
- `internal/hooks/executor_native.go`
- `internal/hooks/executor_subprocess.go`
- `internal/hooks/pipeline.go`
- `internal/hooks/depth.go`
- `internal/hooks/permission.go`
- `internal/hooks/pipeline_test.go`

## Errors / Corrections
- Corrected the permission test helper so explicit deny takes precedence over any patched decision value, matching the production guard semantics.

## Ready for Next Run
- Task 04 is implemented and verified. Next dependency is task_05 (async worker pool) plus task_06 wiring the new pipeline into the `Hooks` struct and typed dispatch functions.
