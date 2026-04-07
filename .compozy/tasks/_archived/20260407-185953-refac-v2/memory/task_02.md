# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Create `internal/api/contract` as the canonical shared daemon DTO package, migrate shared request/response payloads out of `internal/apicore`, preserve existing JSON shapes, and finish with verified tests plus tracking updates.

## Important Decisions
- Moved shared daemon DTOs and shared error payloads into `internal/api/contract`, while keeping SSE helpers and observe cursors in `internal/apicore`.
- Kept HTTP prompt/AI SDK stream payloads local to `internal/httpapi/prompt.go`; only the shared timestamped base event shape is reused through `apicore` conversions.
- Reused the existing DTO names (`SessionPayload`, `MemoryWriteRequest`, etc.) inside `api/contract` to minimize churn during the cutover and keep downstream migrations simpler.

## Learnings
- `internal/api/contract` is a types-only package, so package-level statement coverage reports as `[no statements]`; coverage enforcement remains meaningful in the consuming packages (`apicore`, `httpapi`, `udsapi`).
- The handler, conversion, and shared test surfaces were already narrow enough that the cutover only required import/type updates plus explicit JSON-parity tests.

## Files / Surfaces
- `internal/api/contract/contract.go`
- `internal/api/contract/contract_test.go`
- `internal/apicore/{payloads.go,conversions.go,handlers.go,workspaces.go,memory.go,errors.go}`
- `internal/apicore/{conversions_parsers_test.go,error_paths_test.go,more_coverage_test.go,memory_workspace_test.go}`
- `internal/httpapi/{shared.go,shared_test.go,prompt_contract_test.go}`
- `internal/udsapi/{shared.go,shared_test.go}`

## Errors / Corrections
- Initial wide `apply_patch` attempt failed because the patch combined too many files; split the edit into file-scoped patches before continuing.

## Ready for Next Run
- Verification completed successfully with:
  - `go test ./internal/api/contract ./internal/apicore ./internal/httpapi ./internal/udsapi -count=1`
  - `go test -cover ./internal/api/contract ./internal/apicore ./internal/httpapi ./internal/udsapi -count=1`
  - `go test -tags integration ./internal/httpapi ./internal/udsapi -count=1`
  - `make test-integration`
  - `make verify`
