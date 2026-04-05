# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement task 04 end to end: prompt assembly, session startup propagation, daemon memory/dream wiring, required tests, verification, tracking, and one local commit.

## Important Decisions
- Treated the task spec + techspec as the approved design baseline and kept scope out of task 05 API/CLI handlers.
- Implemented prompt assembly in `internal/memory.Assembler` using `Store.LoadIndex(ScopeGlobal)` plus `Store.ForWorkspace(workspace).LoadIndex(ScopeWorkspace)`.
- Added `acp.StartOpts.SystemPrompt` plus one-time first-turn injection because ACP still has no dedicated startup prompt field.
- Reused the daemon composition root for all memory wiring: boot initializes the shared store and assembler, then daemon runtime owns the dream service, ticker, and session-stop trigger.

## Learnings
- The repo previously had a `PromptAssembler` seam in `session.Manager`, but nothing actually carried the assembled prompt into ACP startup; without the `SystemPrompt` runtime path, task 04 would have passed local manager tests while doing nothing in live sessions.
- The dream trigger is cleanest as a non-blocking queue owned by `daemon.Daemon`; fanout just signals `"session_stop"` and the ticker loop owns the gating / run logging.
- `make verify` stayed green after the wiring changes, and targeted race+coverage for touched packages cleared the 80% bar.

## Files / Surfaces
- `internal/memory/assembler.go`
- `internal/memory/assembler_test.go`
- `internal/acp/types.go`
- `internal/acp/client.go`
- `internal/acp/client_test.go`
- `internal/session/manager.go`
- `internal/session/manager_test.go`
- `internal/session/additional_test.go`
- `internal/daemon/daemon.go`
- `internal/daemon/daemon_test.go`
- `internal/daemon/daemon_integration_test.go`
- `.compozy/tasks/agh-memory-extensibility/task_04.md`
- `.compozy/tasks/agh-memory-extensibility/_tasks.md`

## Errors / Corrections
- Initial ACP regression test failed because the helper test harness did not propagate `StartOpts.SystemPrompt`; fixed `startHelperProcess(...)` to pass the field through.
- Initial daemon test/build failed because the new injected dream-service factory field was referenced but not stored on `Daemon`, and the expanded daemon test doubles needed `sync`; fixed both before re-running the suite.

## Ready for Next Run
- Verification evidence:
  - `go test ./internal/acp ./internal/session ./internal/daemon`
  - `go test -tags integration ./internal/daemon`
  - `go test -race -cover ./internal/memory ./internal/session ./internal/daemon ./internal/acp`
  - `make verify`
- Local commit created: `bc20b58` (`feat: wire memory assembler into daemon`)
- Tracking files still need to be staged only if the repo/workflow wants them in the commit.
