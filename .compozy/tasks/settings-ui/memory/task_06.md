# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Expose `/api/settings/*` and required `/api/extensions` routes on HTTP, enforce loopback-only HTTP mutation policy for privileged routes, and prove the behavior with route inventory, handler/server tests, and parity integration coverage.

## Important Decisions
- Registered settings and extension HTTP routes explicitly in `internal/api/httpapi/routes.go` so route inventory tests can catch drift instead of relying on implicit matcher middleware.
- Kept the loopback-only policy entirely in `internal/api/httpapi` via a route-attached guard keyed off the configured bind host; the shared settings service and `api/core` handlers remain transport-agnostic.
- Added HTTP extension service injection and mirrored the existing UDS extension handler semantics on HTTP to keep payloads and status behavior aligned for the Hooks & Extensions screen.

## Learnings
- Before this run, HTTP had no `/api/settings/*` or `/api/extensions` registration, no extension-service injection, and no bind-host mutation guard; UDS already exposed the extension surface that HTTP needed to match.
- The new HTTP package tests reach `81.4%` statement coverage with route inventory, non-loopback 403 assertions, loopback success paths, and real-server bind-host behavior checks.
- The parity integration needed to compare the runtime harness’ existing shared extension projections over HTTP and UDS; the mock-only install fixture path from helper tests is not available in the real integration runtime.

## Files / Surfaces
- `internal/api/httpapi/routes.go`
- `internal/api/httpapi/handlers.go`
- `internal/api/httpapi/middleware.go`
- `internal/api/httpapi/server.go`
- `internal/api/httpapi/extensions.go`
- `internal/api/httpapi/helpers_test.go`
- `internal/api/httpapi/handlers_test.go`
- `internal/api/httpapi/server_test.go`
- `internal/api/httpapi/transport_parity_integration_test.go`
- `internal/daemon/daemon.go`

## Errors / Corrections
- Initial parity integration tried to install an extension from `/extensions/telegram-reference` over the real runtime harness and failed with `400` because that mocked fixture path is not present in the live integration environment; corrected the test to compare the real shared extension list/status projections already exposed by both HTTP and UDS.

## Ready for Next Run
- Verification evidence is clean: `go test ./internal/api/httpapi -count=1 -cover` (`81.4%`), `go test -tags integration ./internal/api/httpapi -run TestHTTPTransportExtensionParityMatchesUDS -count=1`, and a post-commit `make verify` pass completed on commit `28c32bdd`.
- Commit boundary should include only the HTTP transport/test files plus `internal/daemon/daemon.go`; workflow memory and task tracking files stay unstaged for this task run.
