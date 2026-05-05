# Task Memory: task_13.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Execute Real-Scenario QA for `tools-refac` task_13 against current `HEAD` (`04986daf` at run start), using the saved dossier under `.compozy/tasks/tools-refac/qa/`.
- Required outcome: fresh isolated QA manifest/evidence, `qa/issues/BUG-NNN.md` for reproduced defects, root-cause fixes with regression coverage if defects appear, updated `qa/verification-report.md`, clean final `make verify`, task tracking updates, and one local commit after verification.

## Important Decisions
- Treat old commit `8c5d78d7 feat: harden tools refac QA surface` as historical/off-branch evidence only. It was authored on ancestor `d5316f5b` and cannot be reapplied wholesale without dropping later task 04-12 work from current `HEAD`.
- Execute against the current saved dossier rather than shrinking scope or reusing old evidence.

## Learnings
- The previous ledger/shared memory overstated completion for this branch. Current branch `tools-registry` still has `task_13.md` pending and no current-HEAD QA execution artifacts beyond the task_12 dossier.
- Real hosted MCP binding exposed a macOS path identity defect: daemon expected binary came from `/Users/.../dev/...`, while UDS peer inspection reported `/Users/.../Dev/...`. Fixed by accepting `os.SameFile` executable identity after normalized string comparison fails.
- Isolated MCP auth status QA exposed source-local failure leakage: one remote MCP server needing login made the whole registry unavailable, including `agh__mcp_auth_status`. Fixed by skipping auth-blocked MCP sources during dynamic discovery while preserving builtin tools.
- Automation runtime QA exposed boot-order capture: native tools were registered before automation boot, so automation tools kept a nil manager. Fixed with dynamic automation manager lookup in native tools.
- Runtime E2E exposed stale prompt-diagnostics assumptions after hosted MCP started writing `session_new` lifecycle diagnostics. Fixed tests to filter prompt diagnostics explicitly.
- Runtime E2E also exposed stale UDS/HTTP observe parity expectations: the current turn emits both durable-memory and situation augmenter events. Updated the expected sequence to include both before parity comparison.
- Runtime E2E artifact capture exposed a stale unaugmented-prompt assertion: transcripts now include situation context plus the original user prompt, so the regression should prove the user prompt survives augmentation rather than require exact adjacency.

## Files / Surfaces
- `.compozy/tasks/tools-refac/qa/test-plans/*`
- `.compozy/tasks/tools-refac/qa/test-cases/*`
- `.compozy/tasks/tools-refac/qa/verification-report.md`
- Expected runtime surfaces: CLI, HTTP, UDS, hosted MCP, built-in tools, autonomy, config/hooks/automation/extensions, codegen, web generated/types/tests, site docs/build.
- Fixed production/test files: `internal/mcp/hosted.go`, `internal/mcp/hosted_test.go`, `internal/tools/mcp.go`, `internal/tools/mcp_test.go`, `internal/daemon/native_tools.go`, `internal/daemon/native_automation_tools.go`, `internal/testutil/acpmock/diagnostics.go`, `internal/daemon/daemon_memory_e2e_integration_test.go`, `internal/daemon/daemon_mock_agents_integration_test.go`, `internal/api/udsapi/transport_parity_integration_test.go`, `internal/testutil/e2e/runtime_harness_integration_test.go`.

## Errors / Corrections
- Corrected stale memory before implementation: the on-branch state is not complete despite an off-branch commit object existing locally.
- Filed `BUG-001.md` for hosted MCP binary validation rejecting the same executable through a different case-preserved macOS path; reverified with `hosted-mcp-transcript-after-fix.json`.
- Filed `BUG-002.md` for auth-blocked remote MCP discovery making builtin status tools unavailable; reverified with `mcp-auth-status-tool-after-fix.json`.
- Filed `BUG-003.md` for automation tools retaining `dependency_missing` after automation boot; reverified with `tool-invoke-automation-jobs-list-after-fix.json`.
- Filed `BUG-004.md` for runtime E2E tests counting hosted MCP lifecycle diagnostics as prompt diagnostics; focused race/integration tests now pass.
- Filed `BUG-005.md` for UDS/HTTP observe parity expecting only one turn augmenter; focused race/integration test now passes.
- Filed `BUG-006.md` for runtime harness artifact capture expecting exact unaugmented echo text; focused race/integration test now passes.

## Verification Evidence
- Fresh isolated QA lab: `tools-refac-real-scenario-20260430-074748-514234`.
- Final report: `.compozy/tasks/tools-refac/qa/verification-report.md`.
- `make test-e2e-runtime` passed: daemon 22 tests, HTTP 8 tests, UDS 14 tests, runtime harness 6 tests.
- `make verify` passed after all code fixes: bun lint 0 warnings/0 errors, bun tests 258 files / 1845 tests, Go tests 7097 tests, package boundaries respected.
- Pre-commit `make verify` also passed after report/tracking updates: lint 0 warnings/0 errors, Go tests 7097 tests, package boundaries respected.
- Local commit created: `29de5ffe fix: harden tools refac QA surfaces`.
- Post-commit `make verify` passed from the committed tree: Go tests 7097 tests, package boundaries respected.
- Isolated QA daemon stopped after final verification; stop evidence is `.compozy/tasks/tools-refac/qa/logs/bootstrap/daemon-stop-final.json`.
- Focused package coverage passed: `internal/mcp` 80.6%, `internal/tools` 80.8%.
- Tracking updated: `.compozy/tasks/tools-refac/task_13.md` status completed and `_tasks.md` row 13 completed.

## Ready for Next Run
- Continue from current branch state; do not cherry-pick `8c5d78d7` wholesale.
- No task-local follow-up remains. Uncommitted `.compozy/tasks/tools-refac/*` task/memory/log artifacts remain in the workspace by workflow policy; unrelated pre-existing guidance-file edits remain untouched.
