# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement task 05 prompt provenance and ACP guardrails so network-originated turns are distinguishable from user turns and restricted to allowlisted control-plane commands.
- Completed: session prompt provenance, ACP guardrails, unit coverage, and integration coverage are in place and verified.

## Important Decisions
- Use session-owned runtime metadata for prompt provenance and terminal ownership; ACP handlers should depend on session/runtime state only, not `internal/network` imports.
- Skip the `brainstorming` skill for this run because the task already comes with an approved PRD/TechSpec/ADR design and the user asked for direct implementation.
- Implement the dedicated network prompt path as `Manager.PromptNetwork()` on top of `PromptWithOpts`, keeping the existing `Prompt()` API as the user-turn wrapper.
- Tag allowlisted `agh network {send,peers,spaces,status,inbox}` terminals as `network_owned` and require same-turn ownership for output/wait during network turns.

## Learnings
- Baseline code has no `TurnSource` or `PromptWithOpts` surface yet.
- `internal/acp/handlers.go` currently allows all terminal commands and file writes subject only to normal permission policy.
- Task-referenced future files `internal/network/delivery.go` and `internal/network/manager.go` are not present in this branch yet, so this task should limit itself to the session/ACP surfaces that later network work will consume.
- The worktree already contains unrelated task-tracking edits under `.compozy/tasks/agh-network/`; leave them untouched.
- `exec.LookPath` resolves the terminal executable before `Cmd.Env` is applied, so tests that need a fake `agh` binary must adjust the parent process `PATH`, not only the terminal request environment.
- Verified results: `go test ./internal/session ./internal/acp`, `go test -tags integration ./internal/acp`, `go test ./internal/session -cover`, `go test ./internal/acp -cover`, and `make verify` all pass.

## Files / Surfaces
- `internal/session/interfaces.go`
- `internal/session/manager_prompt.go`
- `internal/session/manager_hooks.go`
- `internal/session/manager_start.go`
- `internal/session/session.go`
- `internal/session/manager_test.go`
- `internal/session/manager_hooks_test.go`
- `internal/acp/handlers.go`
- `internal/acp/permission.go`
- `internal/acp/types.go`
- `internal/acp/handlers_test.go`
- `internal/acp/client_test.go`
- `internal/acp/client_integration_test.go`

## Errors / Corrections
- The caller mentioned `.compozy/tasks/agh-network/MEMORY.md`, but the actual shared workflow memory path in this repo is `.compozy/tasks/agh-network/memory/MEMORY.md`; use the provided workflow-memory block paths.
- The task spec mentioned future delivery files that are not present yet; the implementation stayed scoped to the session/ACP contract those future files will consume.

## Ready for Next Run
- Task 05 is complete. Task 06 should consume `PromptNetwork()` plus the `TurnSourceNetwork` / `network_owned` contract instead of re-implementing provenance or tool-guard logic.
