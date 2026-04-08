# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement `internal/skills/hooks.go` with `HookRunner`, JSON stdin/stdout subprocess dispatch, timeout handling, fail-open logging, and source/name ordering; cover the task matrix with unit tests and `internal/skills` coverage >=80%.

## Important Decisions
- Kept hook ordering separate from `orderSkillsBySource()` so MCP resolver precedence stays unchanged while hooks add same-source alphabetical ordering.
- Used a single POSIX shell fixture at `internal/skills/testdata/hooks/driver.sh` to exercise payload echo, env propagation, non-zero exit, and timeout behavior without adding test-only production seams.
- `RunHooks()` treats the method `event` argument as authoritative and overwrites `HookPayload.Event` before JSON marshaling.

## Learnings
- A deferred duration assignment on a returned struct value does not update the returned copy; `HookResult.Duration` must be assigned before each return path.
- A shell busy loop is sufficient for deterministic timeout coverage because `exec.CommandContext` kills the shell process directly; using `sleep` inside a shell script risks leaving orphaned children.

## Files / Surfaces
- `internal/skills/hooks.go`
- `internal/skills/hooks_test.go`
- `internal/skills/testdata/hooks/driver.sh`

## Errors / Corrections
- Initial test run caught `HookResult.Duration` staying at `0s` on successful hooks; fixed the production code to stamp duration before returning instead of relying on a deferred update.

## Ready for Next Run
- Task-specific checks passed: `go test ./internal/skills -count=1` and `go test ./internal/skills -cover -count=1` (`81.9%`).
- Repo gate passed: `make verify`.
- Local code-only commit created: `a3c36ea` (`feat: implement skills hook runner`).
