# Task Memory: task_11.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement public Tool Registry daemon contracts for task_11: DTOs, shared core handlers, HTTP/UDS route parity, session projections, invoke paths through executable registry backends, regenerated OpenAPI, generated web TypeScript contracts, drift/parity tests, full verification, task tracking updates, and one local commit.

## Important Decisions
- Source of truth: task_11 plus TechSpec sections "Data Models", "API Endpoints", "Agent Manageability Plan", "Implementation Steps" 13-14, ADR-006, ADR-007, ADR-010, and AGH contract/codegen co-ship rules.
- Existing dirty task/spec/memory artifacts are treated as pre-existing unless this task updates task_11 tracking or this task memory.
- Shared handlers map API requests onto registry abstractions (`ToolRegistry`, `ToolsetRegistry`, `ToolApprovalIssuer`) rather than backend packages.
- Local HTTP/UDS approval-required invokes use daemon-memory single-use approval tokens; hosted MCP still uses the existing ACP approval bridge path because it does not pass operator scope.

## Learnings
- Prior tasks completed executable native, extension-host, remote MCP call-through, and hosted MCP session exposure. Task_11 should expose those existing registry capabilities through `internal/api/core` and transports instead of creating a new execution path.
- `RuntimeRegistry` already owns operator/session projection and executable dispatch; API handlers only need DTO conversion, scope shaping, status mapping, and route parity.
- Approval-token consumption must classify a matching expired token before pruning expired records; otherwise expired approvals collapse into generic token mismatch responses.

## Files / Surfaces
- Expected surfaces: `internal/api/contract`, `internal/api/core`, `internal/api/httpapi`, `internal/api/udsapi`, `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, and focused API contract/parity tests.
- Touched so far: `internal/api/contract/tools.go`, `internal/api/core/{interfaces,handlers,errors,tools}.go`, HTTP/UDS route/server wiring, `internal/tools/{approval_token,reason,registry,toolset}.go`, daemon tool registry approval wiring, and OpenAPI spec registry.

## Errors / Corrections
- First compile-focused pass found only stale route-count expectations in HTTP/UDS tests after adding nine tool routes per transport; expected route lists were updated.
- Focused tests found a `core_test` helper-name collision and expired approval tokens being reported as mismatches after eager pruning; fixed the test helper name and moved expired-token classification ahead of pruning.
- `make lint` initially flagged funlen, hugeParam, rangeValCopy, and long-line issues; fixed by passing `ToolView` by pointer, avoiding large range-value copies, extracting approval bridge helpers, and wrapping lines.
- Package boundaries rejected an HTTP API test importing UDS API directly; moved transport parity coverage to `internal/daemon/tools_transport_parity_test.go`, where both transport packages can be composed without violating boundaries.

## Verification / Current State
- `make codegen`, `make codegen-check`, `make bun-typecheck`, and `make bun-test` passed against regenerated contracts. `make bun-test` passed 257 files / 1838 tests.
- Full `make verify` passed after final tracking updates: codegen/typecheck/format/oxlint, Vitest, web build, `golangci-lint` with 0 issues, Go tests with 6900 tests, package boundaries, and build.
- Local commit created: `fd953726` (`feat: add tool registry contracts`).
- Post-commit `make verify` passed: codegen-check/typecheck/format/oxlint, Vitest 257 files / 1838 tests, web build, `golangci-lint` with 0 issues, Go tests with 6900 tests, package boundaries, and build.

## Ready for Next Run
- Task complete. Shared workflow memory was promoted with the task_11 commit hash and downstream API-route/codegen handoff notes.
