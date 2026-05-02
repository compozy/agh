# Task Memory: task_11.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Task 11: expose Soul, Heartbeat, wake status, and session health through shared `internal/api/core` handlers plus HTTP/UDS route registrations with parity tests.
- Success requires deterministic error mapping for not found, validation, stale `expected_digest`, disabled config, and ineligible sessions; redaction must avoid raw prompt-only content.

## Important Decisions
- Reuse Task 10 DTOs/converters from `internal/api/contract/authored_context.go`; Task 11 should not introduce HTTP-only or UDS-only response shapes.
- Compose managed Soul/Heartbeat services once at the daemon/server dependency boundary and inject them into shared `internal/api/core` handlers for both transports.
- Preserve body-level `expected_digest` as the mutation CAS field; Heartbeat `If-Match` remains unsupported.
- Session Soul refresh needs CAS at the session service boundary, not just in the HTTP/UDS handler.
- HTTP now receives the same `AgentContextService` as UDS because `GET /api/agent/context` is registered on both transports.

## Learnings
- Shared workflow memory says Task 10 already added contract DTOs/OpenAPI/codegen and future Task 11 handlers should reuse `internal/api/contract/authored_context.go` DTOs instead of creating transport-specific shapes.
- Baseline route scan shows no Task 11 Soul/Heartbeat/session-health routes registered in `internal/api/httpapi/routes.go` or `internal/api/udsapi/routes.go`; the only matching route is the existing UDS task-run heartbeat endpoint.
- Focused pre-change test command passed but had no httpapi/udsapi route coverage for the new authored-context surface: `go test ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi -run 'TestAuthoredContext|Test.*Soul|Test.*Heartbeat|Test.*SessionHealth' -count=1`.
- `go test ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/daemon ./internal/session -run 'TestAuthoredContext|Test.*Soul|Test.*Heartbeat|Test.*SessionHealth|TestBoot' -count=1` passed after handler/wiring/test implementation.
- Transport parity tests belong in `internal/daemon`, the composition root allowed to import both `httpapi` and `udsapi`; placing that proof under `internal/api/core` violates package boundaries.
- Final pre-commit `make verify` passed after the transport parity test moved to `internal/daemon`, including codegen, Bun checks, Go lint/test/build, and boundary verification.
- Local code commit created: `ecf1382e feat: expose authored context routes`.
- Post-commit `make verify` passed, including codegen, Bun checks, Go lint/test/build, and boundary verification.

## Files / Surfaces
- Expected surfaces: `internal/api/core`, `internal/api/httpapi/routes.go`, `internal/api/udsapi/routes.go`, `internal/api/core/*_test.go`, route parity tests, and codegen/OpenAPI verification if spec metadata changes.
- Implemented surfaces include `internal/api/core/authored_context.go`, `internal/api/core/handlers.go`, `internal/api/core/interfaces.go`, HTTP/UDS server/route wiring, daemon runtime dependency composition, `internal/session/soul.go`, and `internal/api/core/authored_context_transport_test.go`.

## Errors / Corrections
- Initial worktree contains unrelated pre-existing dirty tracking files and untracked `.compozy/extensions/`; preserve them and avoid destructive git commands.
- Corrected an HTTP wiring gap: registering `/api/agent/context` on HTTP also required adding and passing `httpapi.WithAgentContext`.
- Corrected a boundary-gate failure: moved `authored_context_transport_test.go` from `internal/api/core` to `internal/daemon` so the test can exercise both transports without reversing dependency direction.

## Ready for Next Run
- Task 11 implementation is complete and committed in `ecf1382e`; only tracking/memory artifacts remain uncommitted by policy.
