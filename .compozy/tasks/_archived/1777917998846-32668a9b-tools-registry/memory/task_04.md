# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement the provider-agnostic Task 04 dispatch path in `internal/tools`: schema/input validation, call-time policy and availability recheck, canonical `tool_id` hooks, result budgeting/redaction, cancellation propagation, deterministic observability events, focused tests, clean `make verify`, task tracking, and one local commit.

## Important Decisions
- Treat the approved TechSpec/ADRs as the design source of truth; the generic brainstorming design gate is already satisfied by this PRD execution flow.
- Keep dispatch provider-agnostic inside `internal/tools`; concrete native/extension/MCP adapters remain for Tasks 05, 07, and 09.

## Learnings
- Task 03 left `Registry.Call` as a revalidation stub that returns `ErrToolBackendFailed` with "tool dispatch is not wired" after policy passes.
- Baseline `internal/tools` coverage before Task 04 edits is 86.2%.
- Existing tool hook payloads and matchers still expose `tool_name` and `tool_namespace`; Task 04 must hard-cut registry-owned hook identity to canonical `tool_id`.
- Result metadata with sensitive key names must be removed, not value-redacted in place, because `ToolResult.Validate` rejects secret-shaped metadata keys.
- Tool-family hook matchers now use `tool_id`; ACP permission-family hooks continue to use `tool_name` because that is the permission request vocabulary.
- Pre-call hook decisions fail closed on any non-callable decision; dispatch appends `hook_denied` if the hook omitted a reason.
- Result/observability events hash redacted input/result envelopes and record redaction paths instead of carrying raw tool input or secrets.

## Files / Surfaces
- Touched: `internal/tools/registry.go`, `internal/tools/dispatch.go`, `internal/tools/result_limit.go`, `internal/tools/schema.go`, `internal/tools/dispatch_test.go`, `internal/tools/registry_test.go`, hook payload/matcher files, config/settings/CLI/API/daemon/extension hook matcher propagation, and generated OpenAPI/TypeScript/web contracts.

## Errors / Corrections
- Pre-change signal: `go test ./internal/tools -run TestRuntimeRegistryCallFailsClosedBeforeDispatchTask -count=1` passes because dispatch is not wired yet.
- Initial focused hook/tools test failed on removed `ToolName`/`ToolNamespace` fields and the old dispatch-stub expectation; corrected tests to assert canonical `tool_id` and real provider calls.
- Test-convention helper accepts one file per run; new dispatch tests and expanded registry tests pass, while older hook test files still trigger pre-existing inline-test warnings if scanned wholesale.
- Focused affected packages passed after migration: `go test ./internal/tools ./internal/hooks ./internal/config ./internal/settings ./internal/cli ./internal/api/core ./internal/api/httpapi ./internal/daemon ./internal/extension -count=1`.
- Focused `internal/tools` coverage is 81.8%; focused race for `internal/tools` and `internal/hooks` passed.
- `make verify` initially failed on `internal/tools/dispatch.go` funlen after tightening hook denial; extracted callable-target gating to keep the dispatch pipeline readable and lint-clean.
- Final `make verify` passed: frontend format/lint/typecheck/tests/build, Go lint, 6640 Go tests, build, and package boundary checks.

## Ready for Next Run
- Completed in local commit `6be9d30c` after clean `make verify`; tracking and memory files were intentionally left unstaged.
