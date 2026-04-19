# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Complete task 05 by teaching downstream consumers that `synthetic_reentry` is daemon-originated prompt input, not a `user_message`.
- Delivered outputs: distinct transcript rendering, a dedicated synthetic hook input class, extension-host turn/seed discovery that works for synthetic initiating events, mixed-turn ordering/tool pairing regression coverage, and transport/session transcript coverage.

## Important Decisions
- Use the canonical transcript assembler, existing hook input-class mapping, and existing extension-host stored-event replay helpers rather than introducing a parallel replay or hook bus.
- Treat the current seam gaps as the pre-change signal:
  - transcript only renders `user_message`
  - hooks only classify `network` specially
  - extension-host turn discovery scans only `user_message`
- Render `synthetic_reentry` as transcript role `system` to distinguish daemon-originated prompt input while preserving existing user/network transcript behavior.
- Scope transcript tool-call/result lifecycle identity by `turnID + tool_call_id` so mixed-turn transcripts cannot collide when different turns reuse the same tool call id.

## Learnings
- Task 04 already persists synthetic input as `synthetic_reentry` with canonical payload text and metadata, so task 05 can stay focused on read/classification surfaces instead of persistence changes.
- Transcript transport consumers currently tolerate unknown/non-user transcript roles on the frontend by mapping them into the UI system role.
- The UDS integration harness driver only emits assistant and done events for prompts, so synthetic transcript parity there is a 6-message replay shape rather than the 12-message tool-rich HTTP harness shape.

## Files / Surfaces
- `internal/transcript/transcript.go`
- `internal/transcript/transcript_test.go`
- `internal/session/manager_hooks.go`
- `internal/session/manager_hooks_test.go`
- `internal/session/transcript.go`
- `internal/session/transcript_test.go`
- `internal/extension/host_api.go`
- `internal/extension/host_api_test.go`
- `internal/api/httpapi/httpapi_integration_test.go`
- `internal/api/udsapi/udsapi_integration_test.go`

## Errors / Corrections
- An earlier `make test-integration` session wedged on a stale `internal/cli` process after tool polling stopped receiving output; revalidated the real package gate with `go test -race -parallel=4 -tags integration ./internal/cli -count=1 -timeout=2m -v` and reran `make test-integration` cleanly from a fresh process.

## Ready for Next Run
- Current state: task 05 is complete.
- Verification evidence:
  - `go test -coverprofile=/tmp/transcript.cover ./internal/transcript` -> `coverage: 81.2% of statements`
  - `go test -coverprofile=/tmp/session.cover ./internal/session` -> `coverage: 81.0% of statements`
  - `go test -coverprofile=/tmp/extension.cover ./internal/extension` -> `coverage: 80.6% of statements`
  - `make verify` -> pass
  - `make test-integration` -> pass (`DONE 5720 tests, 3 skipped in 56.808s`)
