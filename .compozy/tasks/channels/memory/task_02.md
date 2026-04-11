# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implement task 02: add the daemon-owned channel registry and policy-driven routing layer in `internal/channels/`, with unit and integration coverage.

## Important Decisions

- Added a narrow `channels.Service` over a small store interface so registry behavior stays in `internal/channels/` while task 01 `globaldb` helpers remain persistence-only.
- Canonical routing keys are rebuilt from persisted instance scope/workspace/policy, and mismatched caller-provided scope, workspace, or routing-key hash values are rejected.
- `ResolveOrCreateRoute` reuses the stored session ownership for a canonical key and refreshes route activity; `UpsertRoute` is the explicit rebind/update path.
- Channel lifecycle validation now enforces `enabled=false => status=disabled`, active instances cannot report `disabled`, and disabled instances only transition back through `starting`.

## Learnings

- The channel registry tests need to live in `package channels_test` because `globaldb` already imports `channels`; using the external test package preserves real-SQLite coverage without an import cycle.
- Staticcheck flags literal `nil` contexts in tests during `make verify`, so nil-context guard coverage should use a helper function that returns `context.Context(nil)`.

## Files / Surfaces

- `internal/channels/types.go`
- `internal/channels/routing.go`
- `internal/channels/lifecycle.go`
- `internal/channels/dimensions.go`
- `internal/channels/registry.go`
- `internal/channels/registry_test.go`
- `internal/channels/registry_integration_test.go`

## Errors / Corrections

- Moved the new registry tests to `package channels_test` after the initial version created a `channels -> globaldb -> channels` import cycle.
- Replaced a literal nil context in the guard-clause test after `make verify` failed on staticcheck `SA1012`.

## Ready for Next Run

- Task 02 implementation is complete and verified.
- Verification evidence:
  - `go test ./internal/channels`
  - `go test -cover ./internal/channels` with `80.2%` coverage
  - `go test -tags integration ./internal/channels`
  - `make verify`
