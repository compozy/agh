# Task Memory: task_10.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Wire typed hook dispatch into the session manager lifecycle and runtime paths required by task_10: session create/resume/stop, input submission, prompt assembly, event recording, and agent lifecycle.
- Preserve the current session manager cleanup/finalization semantics while adding sync barriers where required and async observation where the taxonomy marks events async-only.
- Finish with task-specific tests, clean `make verify`, tracking updates, and one local commit.

## Important Decisions
- Keep `internal/session` on a narrow local hook-dispatch interface instead of importing the concrete `internal/hooks.Hooks` runtime directly; `internal/daemon` will adapt the real runtime into that seam.
- Treat the approved task spec, techspec, and ADRs as the validated design context for this implementation run; no extra design loop is needed unless a contradiction appears.
- Keep `session.Notifier` for legacy observer/dream fan-out only; session lifecycle hook execution now flows through `SessionManagerDeps.Hooks` so post-create/post-stop dispatch is not duplicated.

## Learnings
- Current baseline before edits: `internal/session` only has `Notifier`; `startupPrompt` only assembles and returns a string; `recordEvent` writes directly to the recorder; and `internal/hooks.OnAgentEvent` is still a no-op for the richer task_10 agent lifecycle events.
- The permission escalation invariant initially missed ACP's `reject-once` and `reject-always` deny states, which allowed a subprocess `permission.request` patch to escalate a rejected request; fixing the classifier closed the real bug and made the end-to-end test pass.
- Task 09-style daemon tests that manually invoked `Notifier.OnSessionCreated`/`OnSessionStopped` no longer exercised the load-bearing path after task_10; they now need to call `SessionManagerDeps.Hooks.DispatchSessionPostCreate/PostStop` instead.

## Files / Surfaces
- `internal/session/manager.go`
- `internal/session/manager_lifecycle.go`
- `internal/session/manager_helpers.go`
- `internal/session/manager_prompt.go`
- `internal/session/manager_hooks.go`
- `internal/session/interfaces.go`
- `internal/daemon/hooks_bridge.go`
- `internal/daemon/boot.go`
- `internal/daemon/daemon.go`
- `internal/daemon/daemon_integration_test.go`
- `internal/session/manager_test.go`
- `internal/session/manager_hooks_test.go`
- `internal/session/manager_integration_test.go`
- `internal/daemon/notifier_test.go`
- `internal/daemon/daemon_test.go`
- `internal/hooks/dispatch.go`
- `internal/hooks/dispatch_integration_test.go`
- `internal/hooks/pipeline.go`
- `internal/hooks/permission.go`
- `internal/hooks/permission_test.go`

## Errors / Corrections
- Corrected the permission deny-state classifier so `reject-once` and `reject-always` are treated as denials by the dispatcher guard.
- Corrected daemon integration tests to use the new hook-dispatch seam instead of the legacy notifier callbacks.

## Ready for Next Run
- Task 10 is implemented and verified. The remaining follow-on work is task 11, which can reuse the same session-owned hook-dispatch seam for `turn.*`, `message.*`, and `context.*` wiring.
