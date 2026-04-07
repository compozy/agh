# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Align `internal/config` with ADR-003 and the resolver contract: workspace config must load only from explicit `root_dir/.agh/config.toml`, agent discovery order must be reusable/shared for resolver consumption, and `os.Getwd()` must stop defining workspace identity.

## Important Decisions
- Treat missing `WithWorkspaceRoot` input as "global-only config load", not "derive workspace from current working directory".
- Move ordered workspace/additional/global agent discovery helpers into `internal/config` so tests and resolver use the same precedence logic.
- Keep skills discovery in resolver, but drive its root iteration from the shared `config.WorkspaceDiscoveryRoots` helper so agent and skill scans stay aligned without adding config merge layers.

## Learnings
- Current resolver tests already cover the ADR-003 asymmetry (`root` config wins, additional-dir config ignored), but the ordering helper is still private to `internal/workspace`.
- `config.Load()` no longer reads workspace `.env` or `.agh/config.toml` from the current process directory when no workspace root is supplied; callers must pass the resolved root explicitly if they want workspace overlays.
- `internal/config` coverage is now above the task target (`81.4%`) after adding explicit-root and multi-root agent precedence tests.

## Files / Surfaces
- `internal/config/config.go`
- `internal/config/agent.go`
- `internal/config/config_test.go`
- `internal/config/agent_test.go`
- `internal/workspace/resolver.go`
- `internal/workspace/resolver_test.go`

## Errors / Corrections
- None yet.

## Ready for Next Run
- Validation completed successfully:
  - `go test ./internal/config -count=1`
  - `go test ./internal/workspace -count=1`
  - `go test ./internal/config -cover -count=1` (`81.4%`)
  - `go test ./internal/workspace -cover -count=1` (`80.1%`)
  - `make verify`
- Remaining closeout step is the local code-only commit; workflow memory and tracking files should stay out of that commit.
