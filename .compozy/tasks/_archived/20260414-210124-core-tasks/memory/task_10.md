# Task Memory: task_10.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Integrate automation with explicit task-backed work for task_10 without forcing all automation jobs into the task domain.
- Required outputs: direct automation task creation/enqueue, agent-mediated `task.create` with agent authorship and automation-linked origin, non-overlap between automation/task execution state, required unit/integration coverage, clean verification, tracking updates, and one local commit.

## Important Decisions
- Use the approved PRD/techspec/ADR design as the brainstorming baseline instead of starting a new design approval loop.
- Keep the integration explicit and daemon-owned: automation decides when a job is task-backed, and task-backed work must stop using the normal automation dispatch runtime once materialized.
- Model task-backed automation explicitly on `automation.Job.Task`; jobs without that block stay on the existing session-backed dispatch path.
- Represent direct task-backed automation runs as `delegated` activation records linked to the canonical `task_id` / `task_run_id` instead of creating a second automation-owned execution lifecycle.
- Preserve automation provenance for automation-launched agent sessions through a manager-owned session actor registry that is cleaned up when the session stops.

## Learnings
- Current automation dispatch always persists an `automation.Run`, creates a session, prompts it, and finalizes that run; this is the pre-change evidence that task-backed automation is not implemented and would currently duplicate task execution state.
- `task.ActorContext` currently allows `agent_session` actors only with `agent_session` origin, so the task_10 requirement for agent-authored tasks with automation-linked origin will need an intentional provenance path.
- Session creation currently has no general metadata field for automation provenance, so the automation-linked agent path needs explicit daemon-owned context propagation.
- Automation API/spec surfaces also needed task-aware job/run fields so direct task-backed jobs can be configured and delegated runs can expose their linked task/task-run identifiers.
- The global automation store needed schema support for both job-level task config and delegated run linkage; otherwise task-backed activation could not survive restart/reload paths.

## Files / Surfaces
- `internal/automation/manager.go`
- `internal/automation/dispatch.go`
- `internal/automation/types.go`
- `internal/automation/model/types.go`
- `internal/automation/model/validate.go`
- `internal/task/actors.go`
- `internal/task/manager_test.go`
- `internal/task/manager_integration_test.go`
- `internal/daemon/boot.go`
- `internal/daemon/daemon.go`
- `internal/api/core/automation.go`
- `internal/api/core/conversions.go`
- `internal/api/core/automation_test.go`
- `internal/api/contract/automation.go`
- `internal/api/contract/contract_test.go`
- `internal/api/spec/spec.go`
- `internal/api/spec/spec_test.go`
- `internal/store/globaldb/global_db.go`
- `internal/store/globaldb/global_db_automation.go`
- `internal/store/globaldb/global_db_automation_test.go`
- `internal/config/automation.go`
- `internal/automation/dispatch_test.go`
- `internal/automation/manager_test.go`
- `internal/automation/manager_integration_test.go`
- `openapi/agh.json`
- `sdk/typescript/src/generated/contracts.ts`
- `web/src/generated/agh-openapi.d.ts`

## Errors / Corrections
- `make verify` initially failed because generated API artifacts were stale after the automation contract/spec changes; running `make codegen` and then rerunning `make verify` resolved the mismatch.

## Ready for Next Run
- Task complete. Commit `3cc4f2a` contains the implementation and generated artifacts; workflow-memory and task-tracking files were intentionally left unstaged.

## Verification
- `go test ./internal/store/globaldb`
- `go test ./internal/automation ./internal/task ./internal/store/globaldb ./internal/api/core ./internal/api/contract ./internal/api/spec ./internal/daemon`
- `go test -tags integration ./internal/automation`
- `go test -cover ./internal/automation ./internal/task` with `internal/automation` at `80.0%` and `internal/task` at `80.1%`
- `make codegen`
- `make verify`
- Post-commit `make verify` passed on the committed state after the staged-file hooks reformatted the staged diff during `git commit`.
