# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implemented `internal/network` as the RFC v0 protocol owner with envelope types, validation helpers, lifecycle helpers, and task-required unit/integration tests.
- Verified package coverage at `81.9%` via `go test -count=1 -cover ./internal/network/...`.

## Important Decisions
- Use the existing PRD/techspec/RFC set as the approved design baseline for this implementation task.
- Model `proof` and `ext` as opaque raw-JSON maps so v0 preserves unknown payloads without interpreting them.
- Keep transport subject construction out of scope for task_01; expose `RouteToken` plus protocol/lifecycle helpers that later transport/router tasks can reuse directly.
- Return explicit lifecycle outcomes via `LifecycleResult` so later router code can distinguish opened, advanced, unchanged, ignored, and reject-direct cases.

## Learnings
- `direct` always requires `to` and `interaction_id`.
- `receipt` and `trace` always require `interaction_id`, and require `to` only when the message is targeted.
- `whois` responses require `reply_to`; `greet` should keep `to` and `interaction_id` unset.
- `space` must match `[a-z0-9][a-z0-9_-]{0,63}` and `peer_id` must match `[a-z0-9][a-z0-9._-]{0,127}`.
- Route tokens are the first 32 lowercase hex chars of `SHA-256(peer_id UTF-8 bytes)`.
- Terminal interaction states are `completed`, `failed`, and `canceled`; later non-terminal traces must not regress state.
- `receipt(rejected)` and `receipt(canceled)` produce terminal lifecycle transitions in the helper, while `accepted/duplicate/expired/unsupported` stay non-terminal and router policy remains a later concern.

## Files / Surfaces
- `.compozy/tasks/agh-network/_techspec.md`
- `docs/rfcs/003_agh-network-v0.md`
- `docs/rfcs/004_agh-network-v1.md`
- `.compozy/tasks/agh-network/adrs/adr-002.md`
- `.compozy/tasks/agh-network/adrs/adr-005.md`
- `internal/network/envelope.go`
- `internal/network/validate.go`
- `internal/network/lifecycle.go`
- `internal/network/validate_test.go`
- `internal/network/lifecycle_test.go`
- `internal/network/helpers_test.go`
- `internal/network/envelope_integration_test.go`

## Errors / Corrections
- Adjusted `ext` and `proof` round-trip assertions to compare decoded JSON semantics rather than formatting after re-marshaling.

## Ready for Next Run
- Task implementation, task-specific unit/integration tests, and `make verify` all passed.
- Tracking files still need the final task-complete update and the code-only local commit.
