# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Extract shared HTTP/UDS API runtime into `internal/apicore` and shared transport test infrastructure into `internal/apitest` without changing transport contracts, then finish with `internal/apicore` coverage >=80% and a clean `make verify`.

## Important Decisions
- `apicore.BaseHandlers` owns the shared session, agent, observe, daemon, memory, and workspace handlers; `httpapi` and `udsapi` keep only transport-specific concerns such as AI SDK streaming, static/CORS wiring, raw socket lifecycle, and approval-route differences.
- The shared event payload in `apicore` keeps `time.Time` timestamps, while the HTTP AI SDK prompt stream keeps its local string-based timestamp payload in `internal/httpapi/prompt.go`.
- Transport test compatibility aliases live in `internal/httpapi/shared_test.go` and `internal/udsapi/shared_test.go` so production packages stay free of test-only symbols.

## Learnings
- Staticcheck on embedded handler composition prefers calling promoted methods directly (`h.SetStreamDone(...)`, `h.SetHTTPPort(...)`) instead of spelling through the embedded field.
- Minimal test-only shims are safer than broad compatibility layers; unused aliases/functions in `shared_test.go` quickly trigger the `unused` linter.

## Files / Surfaces
- `internal/apicore/`
- `internal/apitest/`
- `internal/httpapi/`
- `internal/udsapi/`

## Errors / Corrections
- `internal/httpapi/prompt.go` still referenced `timeRFC3339Nano` after the shared-code trim; corrected it to use `time.RFC3339Nano` directly.
- Removing the test shim files entirely broke transport package tests; restored minimal `shared_test.go` wrappers for the remaining payload aliases and helper methods used by those suites.

## Ready for Next Run
- Task 03 is complete in local commit `1542208`.
- Verification is clean on the committed tree: `go test ./internal/httpapi ./internal/udsapi ./internal/apicore ./internal/apitest -count=1`, `go test ./internal/apicore -cover -count=1` (`80.4%`), and post-commit `make verify`.
- Tracking and workflow-memory updates are intentionally left unstaged so the code commit stays focused on production and test changes.
