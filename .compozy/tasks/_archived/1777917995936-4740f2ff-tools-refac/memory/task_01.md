# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implemented dynamic tool policy resolution for task_01: projections and dispatch now resolve `PolicyInputs` through a scoped resolver at evaluation time instead of relying on daemon boot-time static inputs.

## Important Decisions
- Added `tools.PolicyInputResolver` with a static adapter to preserve existing registry composition tests and callers.
- Kept the existing `PolicyEvaluator` as the only decision engine; the resolver only feeds current inputs into the shipped evaluation path.
- Applied `agh__bootstrap` and `agh__catalog` as a runtime default discovery overlay only when the caller has a session/agent subject and the resolved agent did not already declare an allowlist. Explicit agent allowlists, denies, and session lineage remain stronger.
- Wired daemon policy resolution to session status, workspace config snapshots, resource-backed agent definitions, agent permissions, and session lineage; availability and hook denial remain in the existing registry dispatch/projection layers.

## Learnings
- `RuntimeRegistry` already shared projection and dispatch evaluation through `evaluatorFor`; passing `context` and `Scope` into that seam was enough to avoid a parallel policy engine.
- Session lineage should enforce an empty child tool list as deny-all, while root sessions with no lineage tool policy remain unrestricted before default discovery overlay.
- Parsed `AgentDef` fixtures need a prompt body to satisfy production validation.

## Files / Surfaces
- `internal/tools/policy_resolver.go`
- `internal/tools/registry.go`
- `internal/tools/dispatch.go`
- `internal/tools/registry_test.go`
- `internal/daemon/tool_policy_resolver.go`
- `internal/daemon/native_tools.go`
- `internal/daemon/native_tools_test.go`

## Errors / Corrections
- Initial daemon integration fixture failed with `agent prompt is required`; corrected the fixture to use a valid production-shaped `AgentDef` instead of weakening validation.

## Ready for Next Run
- Focused verification passed:
  - `go test ./internal/tools -run 'TestRuntimeRegistry(DynamicPolicyResolver|ResolverRevalidatesProjectionAndDispatch|Projections|CallReturnsPolicyDenialsBeforeDispatch)'`
  - `go test ./internal/daemon -run 'TestDaemonNativeRuntimePolicyResolver|TestDaemonBootToolRegistry|TestDaemonNativeTools'`
  - `go test -cover ./internal/tools` => 80.7% statements
  - `go test ./internal/tools ./internal/daemon`
- Full `make verify` passed after the lint correction for `bootToolRegistry` funlen and range value copies.
- Task tracking updated in `task_01.md` and `_tasks.md`.
- Self-review completed with no follow-up code changes required.
- Local commit created: `a4601294 feat: add dynamic tool policy resolver`.
- Post-commit `make verify` passed: frontend checks completed, Go lint reported `0 issues.`, Go tests reported `DONE 7008 tests in 10.381s`, and package boundaries reported `OK: all package boundaries respected`.
