# Task Memory: task_10.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Finish task 10 as the hardening pass over the already-implemented AGH Network runtime.
- Close the remaining runtime reliability risks around reconnect/re-greet, queue pressure, busy-session delivery, shutdown drain, and degraded/recovered diagnostics.
- Prove the final behavior through real integration paths across CLI, UDS, daemon, network, and session layers.

## Important Decisions
- Treat the existing broad integration failure in `internal/session` crash-classification-on-resume as part of task 10 scope because it blocks the required resume/rejoin reliability validation.
- Keep implementation focused on root-cause fixes and missing evidence; do not widen `internal/network` into orchestration behavior outside the v0 boundary.
- Treat shutdown-drain accounting as a runtime correctness issue, not just a logging issue: interrupted in-flight prompt deliveries must not be counted as delivered, and shutdown diagnostics must include both queued and in-flight work.

## Learnings
- Current unit baseline is green for `internal/network`, `internal/cli`, `internal/api/udsapi`, `internal/daemon`, `internal/session`, and `internal/observe`.
- Current touched-package coverage baseline is already at or above the 80% target for the main task-10 surfaces.
- The broad tagged integration command still fails in `internal/session`: `TestManagerIntegrationResumeClassifiesCrashAndActivates` loses the crash stop reason on resume.
- Existing integration coverage does not yet exercise the full direct/retry/resume/reconnect reliability matrix required by task 10.
- `session.activate()` was clearing repaired crash classification unconditionally; crash-classified resumes need to preserve that stop metadata so degraded provenance survives recovery.
- `StopWithCause()` could return before `proc.Done()` closed, which left stop finalization to the watcher path and created a real cleanup race during resume integration flows.
- Network shutdown could cancel `drainPromptEvents()` and still log/count the envelope as delivered; tracking in-flight deliveries separately fixes both the false delivery metric and the undercounted shutdown `pending_messages`.
- Final task-10 evidence is green:
  - `go test ./internal/network ./internal/session`
  - `go test ./internal/daemon ./internal/cli`
  - `go test -tags integration ./internal/network ./internal/cli ./internal/daemon ./internal/api/udsapi ./internal/session -count=1`
  - `go test -cover ./internal/network ./internal/cli ./internal/api/udsapi ./internal/daemon ./internal/session ./internal/observe`
  - `make verify`
  - Coverage: `network` 80.9%, `cli` 80.0%, `api/udsapi` 81.7%, `daemon` 80.2%, `session` 81.9%, `observe` 81.6%

## Files / Surfaces
- `internal/session/{session.go,manager_lifecycle.go,manager_start.go,manager_stop_integration_test.go}`
- `internal/network/{manager.go,delivery.go,transport.go,*_test.go,*_integration_test.go}`
- `internal/cli/cli_integration_test.go`
- `internal/daemon/daemon_integration_test.go`
- `internal/api/udsapi/udsapi_integration_test.go`
- `internal/session/{manager_helpers.go,manager_test.go,session_test.go,stop_reason.go}`
- `internal/network/{delivery_test.go,manager_test.go}`

## Errors / Corrections
- Baseline failure reproduced: `go test -tags integration ./internal/network ./internal/cli ./internal/daemon ./internal/api/udsapi ./internal/session`
  - `internal/session`: `TestManagerIntegrationResumeClassifiesCrashAndActivates` expects repaired `agent_crashed` stop metadata to survive resume, but the resumed active session currently clears that classification.
- Correction applied: `prepareResumeStart()` now marks crash-classified resumes to preserve repaired stop metadata through activation, and `Session.activate()` honors that flag.
- Correction applied: `StopWithCause()` now waits on `proc.Done()` after a successful driver stop before finalizing metadata/removal, eliminating the stop/resume race.
- Correction applied: `deliveryCoordinator` now tracks in-flight work, logs `network.message.delivery_interrupted` on shutdown cancellation, suppresses false delivered accounting, and lets shutdown diagnostics report `queued_messages` plus `inflight_messages`.

## Ready for Next Run
- Task tracking is updated in the working tree and verification is complete; the remaining action is the local source-code commit while leaving tracking-only artifacts unstaged.
