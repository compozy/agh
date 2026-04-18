# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add the dedicated synthetic prompt submission path for harness work so daemon-originated turns do not persist as `user_message`.
- Keep synthetic submission daemon-owned, carry durable wake-up metadata, and preserve ordinary user/network prompt behavior.
- Add direct regression coverage for validation, persistence, metadata threading, and busy-session ordering.

## Important Decisions
- Use a narrow synthetic-only queue in `internal/session` because ACP already serializes live prompts per process; a general scheduler is unnecessary for task 04.
- Keep `PromptWithOpts` closed to synthetic callers and introduce a dedicated synthetic submission helper/path instead of widening the ordinary prompt API.
- Persist synthetic input with a dedicated event type and structured metadata envelope rather than overloading `acp.EventTypeUserMessage`.
- Preserve ordinary user and network prompt behavior by reserving exclusive prompt setup and queueing only for the daemon-owned synthetic path.

## Learnings
- The current baseline already exposes the required trust-boundary gap: `manager_prompt.go` rejects synthetic turns through `PromptWithOpts`, and `recordPromptInputEvent` always writes `acp.EventTypeUserMessage`.
- `session.beginPromptSetup()` does not queue turns; real single-turn enforcement currently happens deeper in ACP via `AgentProcess.beginPrompt()` returning `acp: prompt already in progress`.
- Transcript/hooks/extension host still key heavily off `user_message`; those downstream consumers are task-05 follow-on surfaces unless a minimal change becomes strictly necessary here.
- Task-04 coverage exposed a separate stop-lifecycle race: when stop preparation observes an already-exited process, `RequestStopWithCause` and `StopWithCause` must finalize immediately instead of leaving the session in `stopping`.

## Files / Surfaces
- `internal/session/interfaces.go`
- `internal/session/manager.go`
- `internal/session/manager_prompt.go`
- `internal/session/manager_lifecycle.go`
- `internal/session/session.go`
- `internal/session/synthetic_prompt.go`
- `internal/session/stop_reason.go`
- `internal/session/manager_test.go`
- `internal/session/manager_integration_test.go`
- `internal/acp/types.go`
- `internal/acp/types_test.go`
- `internal/transcript/transcript.go`

## Errors / Corrections
- An initial broad prompt-in-progress guard regressed ordinary extension prompt paths; the fix was to leave `beginPromptSetup()` unchanged and reserve `beginExclusivePromptSetup()` for synthetic submissions only.
- Coverage and verification exposed a pre-existing stop finalization gap for already-exited processes; `stop_reason.go` now finalizes those sessions immediately.

## Ready for Next Run
- Implementation and verification are complete. Follow-on work belongs to task 05 for transcript/hook/extension-host synthetic handling and task 07 for task-run completion reentry.
