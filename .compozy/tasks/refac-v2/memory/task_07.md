# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Move canonical replay assembly from `internal/session` into `internal/transcript` without changing transcript endpoint semantics or transcript sequencing/role behavior.
- Finish with replay-focused tests under `internal/transcript`, one real API transcript integration test, and fresh coverage/verification evidence.

## Important Decisions
- `internal/transcript` now owns the replay DTOs (`Message`, `Role`, `ToolResult`), assembly logic, and canonical event-envelope marshaling via `MarshalAgentEvent`.
- `session.Manager.Transcript` stays as the public manager surface but now only queries persisted events and delegates to `transcript.Assemble`.
- Internal consumers that typed transcript responses against `session` were updated to depend on `transcript.Message` directly instead of keeping session-local aliases.

## Learnings
- Direct transcript unit tests must use persisted-like `store.SessionEvent` inputs with non-empty event IDs for assistant buffering; the persisted session DB supplies those IDs in real flows.
- The task coverage target required focused transcript-package tests beyond repo-wide gates because `make verify` does not print package coverage and `internal/transcript` started below the threshold.

## Files / Surfaces
- `internal/transcript/transcript.go`
- `internal/transcript/transcript_test.go`
- `internal/session/transcript.go`
- `internal/session/transcript_test.go`
- `internal/session/manager_prompt.go`
- `internal/session/additional_test.go`
- `internal/api/core/interfaces.go`
- `internal/api/testutil/apitest.go`
- `internal/api/core/{handlers_test.go,error_paths_test.go}`
- `internal/api/httpapi/{handlers_test.go,httpapi_integration_test.go}`
- `internal/api/udsapi/handlers_test.go`
- `internal/daemon/{daemon.go,daemon_test.go}`

## Errors / Corrections
- Initial direct transcript unit tests missed that assistant buffering relies on persisted event IDs; fixed by using realistic event IDs in the transcript package tests.
- `internal/transcript` coverage initially landed at 73.9%; added focused `MarshalAgentEvent` tests to reach `80.7%`.

## Ready for Next Run
- Verification evidence is complete:
  - `go test -cover ./internal/transcript ./internal/session ./internal/api/httpapi -count=1`
  - `go test -tags integration ./internal/api/httpapi -run TestHTTPSessionTranscriptEndpointWithRealSessionManager -count=1`
  - `make verify`
- Remaining closeout is task tracking updates and the single local commit for code changes only.
