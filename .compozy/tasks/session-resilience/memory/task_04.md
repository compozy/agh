# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Complete task 04 by adding resume repair classification and infrastructure validation, exposing `Config.Session.Limits.Timeout`, verifying the full stop/resume behavior end-to-end, and closing the task with clean verification.

## Important Decisions

- Kept the existing `dispatchSessionPreResume` / `dispatchSessionPostResume` boundaries as the required hook seams instead of introducing a parallel no-op seam, so resume repair stays compatible with the real hook architecture already present in the repo.
- Inserted crash classification and infrastructure validation immediately after `ReadSessionMeta()` and before workspace resolution / ACP startup, matching ADR-003’s repair-before-start requirement.
- Validation failures are returned as an aggregated diagnostic after all four independent checks run, while each failed check is also logged with structured metadata.

## Learnings

- Resumed sessions need their in-memory stop fields initialized from repaired `SessionMeta`, otherwise downstream API/state checks can miss the persisted stop metadata after resume.
- Real ACP integration tests are race-prone if they resume immediately after `Stop()`. Waiting for the manager to reach the final stopped state avoids false failures caused by delayed watcher finalization.
- HTTP API crash propagation tests are easiest to drive with an injected `Wait` error in the integration driver rather than process-level kill mechanics.

## Files / Surfaces

- `internal/session/resume_repair.go`
- `internal/session/resume_repair_test.go`
- `internal/session/manager_lifecycle.go`
- `internal/session/session.go`
- `internal/session/manager_stop_integration_test.go`
- `internal/config/config.go`
- `internal/config/merge.go`
- `internal/config/config_test.go`
- `internal/api/httpapi/httpapi_integration_test.go`

## Errors / Corrections

- Initial resume integration coverage was flaky because `Stop()` returned before the stop watcher finalized metadata. Added `waitForStoppedSession()` in the real ACP harness tests and re-ran verification cleanly.

## Ready for Next Run

- Full verification is green, including targeted session/config/httpapi tests, integration stop/resume coverage, package coverage over 80% for `internal/session` and `internal/config`, and a clean `make verify`.
- Local code commit created: `8c2f532` (`feat: add resume repair pipeline`). Tracking and workflow memory updates remain intentionally unstaged.
