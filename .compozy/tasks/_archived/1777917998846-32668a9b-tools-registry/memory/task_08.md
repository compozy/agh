# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Task 08: public `sdk/go` extension SDK for third-party out-of-process Go tool providers.
- Required outcomes: function-based tool registration, subprocess JSON-RPC runtime for Task 07 protocol, Host API client primitives, digest fixture parity, Go create-extension template, external-package conformance tests, >=80% SDK coverage, clean `make verify`, tracking updates, and one local commit.

## Important Decisions
- Source of truth loaded before implementation: `_techspec.md`, `_tasks.md`, Task 08, ADR-001, ADR-008, ADR-009, shared workflow memory, and Task 06/07 ledgers.
- Public SDK must not import `internal/*`; protocol types/constants should be mirrored or generated from public wire contracts, not coupled to daemon internals.
- Go extension tools are third-party subprocess handlers behind `extension_host`; daemon built-ins remain first-party `native_go` providers.

## Learnings
- Task 07 completed `tool.provider`, `provide_tools`, `tools/call`, TypeScript SDK tool authoring, and create-extension JSON-manifest template support in commit `f88f47b9`.
- Shared digest fixtures already exist at daemon, TypeScript SDK, and Go SDK fixture paths from Task 06/07 handoffs.
- Baseline before implementation: `go test ./sdk/go` fails because `sdk/go` has no Go files; `sdk/create-extension/templates/go-tool-provider` does not exist; existing internal extension subprocess integration test passes with the pre-Go-SDK helper runtime.
- Public Go SDK implementation mirrors Task 07 TypeScript protocol names and wire fields without importing daemon `internal/*`.
- SDK focused coverage reached 81.5% with external-package tests.
- Create-extension now has a `go-tool-provider` template; focused test builds the generated extension with a local module replacement.
- Registry integration now compiles a real Go SDK extension binary and proves read-only dispatch through `Registry.Call` plus mutating approval gating.
- Focused validation passed for SDK coverage, extension registry tests, create-extension tests/typecheck, combined Go package tests, and `git diff --check`.
- The project-local `scripts/check-test-conventions.py` referenced by `agh-test-conventions` is absent, so that heuristic could not be run.
- First full `make verify` failed at lint on new-code issues; corrected ready-callback error handling, internal session pointer passing, long lines, typed nil-context validation coverage, and context-aware subprocess commands in tests.
- Post-correction focused validation passed: `go test ./sdk/go -cover -count=1` (81.2%), `go test ./internal/extension -run 'TestExtensionToolProvider(GoSDKSubprocessIntegration|SubprocessIntegration)' -count=1`, and `make lint`.
- Self-review tightened registration to reject duplicate explicit tool IDs across handlers and added test coverage.
- Final pre-commit validation passed: `make verify` completed with 6801 tests and package-boundary checks passing.
- Created local code-only commit `58ad2dba` (`feat: add public Go extension SDK`).
- Post-commit validation passed: `make verify` completed with 6801 tests and package-boundary checks passing.
- Post-commit SDK coverage passed: `go test ./sdk/go -cover -count=1` reported 82.4% statement coverage.

## Files / Surfaces
- Expected implementation surfaces: `sdk/go/**`, `sdk/create-extension/src/index.ts`, `sdk/create-extension/src/index.test.ts`, digest fixtures, and narrowly scoped daemon/runtime tests only if needed for Go SDK conformance.
- Touched implementation surfaces: `sdk/go/**`, `sdk/create-extension/src/index.ts`, `sdk/create-extension/src/index.test.ts`, `sdk/create-extension/templates/go-tool-provider/**`, and `internal/extension/tool_provider_test.go`.

## Errors / Corrections
- Corrected lint failures found by full `make verify`; no tests were weakened.
- Corrected self-review duplicate-ID ambiguity without expanding task scope.

## Ready for Next Run
- Current state: implementation complete, tracking updated, local commit created and post-commit verified.
