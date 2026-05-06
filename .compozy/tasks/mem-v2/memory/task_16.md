# Task Memory: task_16.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Bind the Memory v2 public contract to shared API core handlers and register the same final Slice 1 route family in HTTP and UDS.
- Task references checked: `_techspec.md` `API Endpoints`, `Agent Manageability Plan`, `Development Sequencing` step 26, ADR-009, ADR-011, and ADR-006.

## Important Decisions

- Memory v2 transport business logic now lives in `internal/api/core/memory.go`; HTTP and UDS remain thin route registration layers.
- Legacy memory route shapes were hard-cut from transport registration: no `GET /memory/search`, no `POST /memory/consolidate`, and no `PUT /memory/:filename` family remains in HTTP/UDS route lists.
- Mutation responses expose redaction-safe `MemoryDecisionPayload` data only; raw WAL fields such as `post_content`, `prior_content`, and raw LLM responses stay internal.
- Some final Slice 1 routes are registered now but return deterministic `memory.unsupported` 501 envelopes until later tasks wire their daemon/native-tool services.
- Route parity does not import one transport from another; `internal/api/testutil` now owns the shared Memory v2 route-key expectation used by both transport test suites.

## Learnings

- The package boundary checker forbids `internal/api/udsapi` importing `internal/api/httpapi`, even from tests. Transport parity tests must compare each transport against a neutral API/testutil contract.
- `make codegen-check` can fail with `mage_output_file.go` when run in parallel with other Mage-backed Make targets. Sequential rerun passed with no generated drift.
- Recall/search route tests must use non-trivial query text with at least three meaningful lexical tokens because task 06 intentionally skips trivial recall queries.

## Files / Surfaces

- Shared core handlers: `internal/api/core/memory.go`, `internal/api/core/errors.go`.
- HTTP/UDS route registration: `internal/api/httpapi/routes.go`, `internal/api/udsapi/routes.go`.
- Parity guard: `internal/api/testutil/memory_routes.go`, `internal/api/httpapi/handlers_test.go`, `internal/api/udsapi/handlers_test.go`.
- Handler and transport coverage: `internal/api/core/memory_workspace_test.go`, `internal/api/httpapi/memory_test.go`, `internal/api/udsapi/memory_test.go`, HTTP/UDS integration tests.

## Errors / Corrections

- First full `make verify` failed because `internal/api/udsapi/handlers_test.go` imported `internal/api/httpapi` for parity. Replaced that cross-transport import with a neutral route contract helper and validated `make boundaries`.
- A redaction assertion initially rejected `post_content_hash`; corrected it to reject only raw forbidden fields (`"post_content":`, `"prior_content":`, `"raw_response":`) while still requiring the public hash.
- A shell inspection command with backticks accidentally launched an extra `make verify` via command substitution. That run reached green test/build/boundary output before the outer `rg` failed on the resulting regex; ignore it as validation noise.

## Ready for Next Run

- Focused validation passed: `go test ./internal/api/... -count=1`.
- Race validation passed: `go test -race ./internal/api/core ./internal/api/testutil ./internal/api/httpapi ./internal/api/udsapi -count=1`.
- Integration validation passed: `go test -tags integration ./internal/api/httpapi -run 'TestHTTPMemory' -count=1` and `go test -tags integration ./internal/api/udsapi -run 'TestUDSMemory|TestMemoryRoutesMatchHTTPTransport' -count=1`.
- Guardrails passed: `make lint`, `make codegen-check` sequential, `make boundaries`, and `git diff --check`.
- Final post-state full `make verify` passed with Bun tests 330 files / 2090 tests, Go lint `0 issues`, Go tests `DONE 8357 tests`, and package boundaries `OK`.
- Next task after state update should be `task_17` (CLI Memory Hard Cut).
