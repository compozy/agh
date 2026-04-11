# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement task_06 by adding the extension lifecycle manager, subprocess handshake/recovery, hook/resource registration, and the required unit/integration coverage.
- Finish only after `internal/extension` coverage is >=80%, the extension integration suite passes, and `make verify` succeeds.

## Important Decisions
- The manager keeps extension runtime state in-memory and exposes additive accessors (`HookDeclarations`, `AgentDefinitions`, `MCPServers`, `Statuses`, `Get`) instead of mutating unrelated registries directly.
- Extension skills register through `skills.Registry.RegisterExternal` overlays keyed by extension name so stop/disable flows can remove them cleanly without filesystem writes.
- Capability grants are registered during VALIDATE and explicitly unregistered during stop/disable so Host API authorization always follows the currently active extension set.

## Learnings
- The first coverage pass landed at `76.3%`; targeted helper/option/health-path tests raised `internal/extension` coverage to `83.3%`.
- `make verify` initially failed on a `staticcheck` clone warning in `cloneExtension`; replacing the manual skill-copy loop with `append(clone.Skills, ext.skills...)` cleared the final gate.

## Files / Surfaces
- `internal/extension/manager.go`
- `internal/extension/manager_test.go`
- `internal/extension/manager_integration_test.go`
- `internal/extension/capability.go`
- `internal/skills/loader.go`
- `internal/skills/registry.go`
- `internal/skills/registry_external.go`
- `.compozy/tasks/ext-architecture/task_06.md`
- `.compozy/tasks/ext-architecture/_tasks.md`

## Errors / Corrections
- Coverage initially missed the task target; fixed by adding direct tests for manager options, helper branches, non-subprocess activation, and unhealthy-process supervision.
- The full verification gate found one lint issue in `cloneExtension`; fixed before rerunning `make verify`.

## Ready for Next Run
- Verification evidence:
- `go test -cover ./internal/extension` → `83.3%`
- `go test ./internal/extension ./internal/skills`
- `go test -tags integration ./internal/extension`
- `make verify`
