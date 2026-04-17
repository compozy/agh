# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Completed: shared-harness-based HTTP and UDS transport tests now cover HTTP approval-sensitive flow, HTTP webhook ingress, UDS/CLI-visible projection parity, and the documented UDS approval asymmetry.

## Important Decisions
- Keep transport tests narrow supplements to the daemon runtime lane; do not recreate network/automation/bridge/environment truth already covered in `internal/daemon`.
- Preserve the current UDS approval behavior as an explicit regression assertion: the route exists but returns `501 Not Implemented`.
- Add shared transport helpers under `internal/testutil/e2e` instead of importing `internal/cli` directly in transport-suite helpers; direct `internal/cli` imports create a test import cycle through `internal/daemon`.
- For the UDS approval asymmetry test, observe the pending permission request through the HTTP prompt stream and assert the UDS `501` response on the approval route itself. The fixture-backed ACP driver blocks until it receives a real decision, so waiting for a UDS prompt timeout is not a stable transport-parity proof.

## Learnings
- `internal/api/httpapi/httpapi_integration_test.go` and `internal/api/udsapi/udsapi_integration_test.go` still rely heavily on package-local `newIntegrationRuntime` helpers instead of the subprocess-backed shared runtime harness from task_01.
- The shared harness already exposes the key transport hooks needed for this task: HTTP/UDS JSON clients, session creation/prompting, HTTP approval, and signed webhook delivery.
- Existing UDS coverage only documents the approval gap indirectly (`extensions_additional_test.go`) and the webhook omission as `404`; it does not yet provide shared-harness parity assertions for task_07.
- The shared harness now also exposes transport-parity helpers for CLI-facing reads (`TransportClients()` + `CLI.RunJSON(...)`), narrow webhook projection comparison, UDS approval-gap validation, and UDS prompt streaming with event callbacks.
- Focused verification passed with integration coverage above the task threshold: `internal/testutil/e2e` 80.8%, `internal/api/httpapi` 84.5%, and `internal/api/udsapi` 84.8%.

## Files / Surfaces
- `internal/api/httpapi/httpapi_integration_test.go`
- `internal/api/httpapi/helpers_integration_test.go`
- `internal/api/httpapi/transport_parity_integration_test.go`
- `internal/api/udsapi/udsapi_integration_test.go`
- `internal/api/udsapi/transport_parity_integration_test.go`
- `internal/api/udsapi/routes.go`
- `internal/api/udsapi/sessions.go`
- `internal/testutil/e2e/{runtime_harness.go,automation_tasks.go,mock_agents.go,transport_parity.go,transport_parity_test.go}`

## Errors / Corrections
- Initial attempt to use the typed `internal/cli` daemon client from shared transport helpers created an import cycle for `httpapi`/`udsapi` integration tests through `internal/daemon`; corrected by using the harness-owned shell CLI helper instead.
- Initial attempt to prove the UDS `501` gap by waiting on a fixture-backed UDS prompt stream timed out because the mock ACP permission step blocks until a real decision arrives; corrected by observing the pending permission via HTTP SSE and asserting only the UDS approval route behavior.

## Ready for Next Run
- Task implementation is complete. The source/test changes were committed as `98b35df6` (`test: add transport parity e2e`), and `make verify` passed again on `HEAD` after the commit hook. Tracking and workflow-memory files remain intentionally unstaged for local continuity.
