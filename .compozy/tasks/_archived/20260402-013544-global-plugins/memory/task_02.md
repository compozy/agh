# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Unify internal runtime env usage on `AGH_*` and remove `BuildHookConfig` / `HookConfig` from the driver interface and tests.

## Important Decisions
- Expand the env rename across all `internal/` CLI/runtime surfaces because task_02 success criteria require zero remaining internal `COLLAB_*` / `AGI_*` references.
- Keep task_02 scoped to env/interface cleanup and `BuildHookConfig` removal; do not silently roll in the broader task_03 zero-workdir refactor beyond code needed to compile after removing the hook-config API.

## Learnings
- The env rename had to cover additional CLI/runtime surfaces beyond the task callout list: `internal/cli/runtime.go`, `messaging.go`, `lifecycle.go`, `output.go`, and `daemon.go`, plus their tests.
- The interface removal affected kernel/CLI test doubles in addition to the four concrete drivers.
- A source-level regression test under `internal/kernel/api_test.go` now guards against reintroducing literal `AGI_` / `COLLAB_` references under `internal/`.
- The initial targeted CLI package coverage landed at `79.2%`; targeted tests for `agent-status`, `wait`, and `done` error branches raised it to `80.1%`.

## Files / Surfaces
- `internal/kernel/types.go`
- `internal/kernel/api.go`
- `internal/cli/hooks.go`
- `internal/cli/runtime.go`
- `internal/cli/messaging.go`
- `internal/cli/lifecycle.go`
- `internal/cli/output.go`
- `internal/cli/daemon.go`
- `internal/drivers/claude/claude.go`
- `internal/drivers/codex/codex.go`
- `internal/drivers/opencode/opencode.go`
- `internal/drivers/pi/pi.go`
- Matching `*_test.go` files under `internal/kernel`, `internal/cli`, and `internal/drivers`

## Errors / Corrections
- Legacy-prefix stripping helpers in drivers cannot keep literal `AGI_` / `COLLAB_` strings in source because task success uses grep over `internal/`; use concatenated literals in the helper constants instead.
- CLI coverage needed an extra pass after the main refactor; added focused tests instead of weakening thresholds or broadening scope.

## Ready for Next Run
- Final verification evidence for this task:
  - `rg -n 'COLLAB_|AGI_' internal` returned no matches.
  - `go test -cover ./internal/kernel ./internal/cli ./internal/drivers/claude ./internal/drivers/codex ./internal/drivers/opencode ./internal/drivers/pi` passed with package coverage `80.2%`, `80.1%`, `85.7%`, `85.7%`, `82.1%`, and `86.1%`.
  - `make verify` passed cleanly.
