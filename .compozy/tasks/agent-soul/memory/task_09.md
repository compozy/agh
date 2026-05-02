# Task Memory: task_09.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement the Heartbeat wake service as an advisory policy consumer that can reorient an already eligible session through the existing synthetic prompt path.
- Preserve task ownership boundaries: no task-run creation, `ClaimNextRun`, claim-token handling, lease renewal, session creation, or network greet changes.

## Important Decisions
- The wake service consumes Task 05-08 primitives: latest valid persisted Heartbeat snapshots, session health read models, wake state/events, and `session.PromptSynthetic`.
- Wake decisions fail closed when current config disables Heartbeat, no valid policy exists, a snapshot envelope is invalid, active/quiet-hour policy blocks the wake, session health is not wake-eligible, cooldown/coalescing applies, or the prompt gate is busy.
- Synthetic Heartbeat prompts use `agent_heartbeat_wake` metadata and Heartbeat-specific wake event/policy fields; they do not carry task ownership credentials.
- Scheduler dispatch now has an optional batch waker path so the Heartbeat service can apply `MaxWakesPerCycle` across one scheduler cycle instead of per target.
- `WakeMany` preserves one decision per request, including failed placeholder decisions on partial errors, so scheduler batch results cannot drift out of order.

## Learnings
- `session.PromptSynthetic` currently queues behind active prompts; Heartbeat wake needs an explicit busy/no-queue mode so prompt-gate races are auditable as `session_prompt_active_race`.
- The existing mechanical scheduler wakes sessions for queued task runs but does not claim task runs; Heartbeat integration must remain a policy layer on top of that notification path.
- The scheduler's previous per-target dispatch was insufficient for config-bound per-cycle wake limits; batch dispatch is the runtime enforcement point for this task.

## Files / Surfaces
- Implemented: `internal/heartbeat/wake.go` and `internal/heartbeat/wake_test.go`.
- Implemented: `internal/session/synthetic_prompt.go` prompt-gate option and synthetic metadata fields in `internal/acp/types.go`.
- Implemented: scheduler batch dispatch in `internal/scheduler`, daemon scheduler/harness adapters, and daemon integration tests.

## Errors / Corrections
- Corrected scheduler integration to use batch wake dispatch after self-review found `WakeMany` was implemented but not used by the scheduler.
- Corrected `WakeMany` to preserve decision cardinality/order on partial errors.

## Ready for Next Run
- Task 09 implementation was committed in `f83ef970` (`feat: add heartbeat wake service`); tracking and workflow memory updates remain local artifacts.
- Verification evidence: focused Go tests, race-focused tests, heartbeat coverage at 80.3%, targeted golangci-lint, and full `make verify` passed before and after the commit.
