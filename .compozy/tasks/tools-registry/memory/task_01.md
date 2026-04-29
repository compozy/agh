# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Task 01 core Tool Registry contracts in `internal/tools`: canonical ToolID grammar/reasons, descriptor/backend/source/availability/result/error models, provider/handle interfaces, validators, and package-boundary-safe tests.
- Success evidence required: focused unit tests, >=80% `internal/tools` coverage, `go test ./internal/tools -race`, `make boundaries`, full `make verify`, self-review, task tracking updates, and one local commit after clean verification.

## Important Decisions
- Scope stays inside `internal/tools` plus boundary checks only if needed. Later task consumers (`extension`, `mcp`, `hooks`, `api`) should not be edited unless compilation requires adapting to the hard-cut contracts.
- Preserve greenfield identity rules: one canonical `ToolID`, no dotted IDs, no aliases, no truncation/hash suffix for over-length IDs.
- `ToolID` and `ToolsetID` validate during text/JSON marshal; encode-before-validate callers can now fail before resource validation if they attempt to marshal an invalid ID. Tests should use syntactically valid IDs when they specifically need resource-codec validation errors.
- The task's requested `make boundaries` command now delegates through the Makefile to the existing Mage `Boundaries` task.

## Learnings
- TechSpec says `internal/tools` owns contracts and dispatch-facing interfaces but must not import daemon/API/CLI/extension/MCP/session/task/skills/network packages.
- ADR-001/ADR-010 require `native_go`, `extension_host`, and `mcp` executable backend contracts in MVP; this task defines contracts only, not executable adapters.
- ADR-007 sets the ToolID grammar: lowercase ASCII segments separated by reserved `__`, max length 64, stable `id_too_long` reason for over-length IDs.
- Extension manifest tool publication currently derives canonical IDs as `ext__<extension_name>__<tool_key>` using `CanonicalToolID`; this is an adapter bridge until later tasks add executable registry/runtime reconciliation.

## Files / Surfaces
- Code surface touched: `internal/tools/*.go`, `internal/extension/resource_publication.go`, `internal/daemon/tool_mcp_resources.go`, and test/benchmark adapters in `internal/extension`, `internal/daemon`, and `internal/api`.
- Validation surface touched: `Makefile` now includes `make boundaries`; `magefile.go` boundary logic was reused unchanged.

## Errors / Corrections
- Initial focused coverage for `internal/tools` was 59.2%, below the task's 80% target. Added branch coverage for ID helpers, descriptor/backend/source validators, availability, result envelopes, error formatting, resource codec errors, and package boundary test; coverage is now 93.6%.
- `make boundaries` initially failed because the Makefile had no target even though Mage had `Boundaries`. Added the Makefile wrapper and confirmed `make boundaries` passes.

## Ready for Next Run
- Current run has read required memory, repo guidance, `_techspec.md`, `_tasks.md`, `task_01.md`, and ADR-001 through ADR-010 before editing code.
- Focused verification passed: `go test ./internal/tools -race -count=1`, `go test ./internal/tools -coverprofile=/tmp/tools.cover -count=1` (93.6%), AGH test-shape checks for `internal/tools/*_test.go`, `make boundaries`, and `go test ./internal/extension ./internal/daemon ./internal/api/... -count=1`.
- Full verification passed: `make verify` completed web format/lint/typecheck/tests/build, Go lint, race-enabled Go tests, Go build, and package boundaries.
- Task tracking files were updated after clean verification: `task_01.md` status/checklists and `_tasks.md` row 01 are marked completed.
- Final local task commit: `2cebdfe9 feat: add core tool registry contracts`.
- Post-amend verification passed: `make verify` completed successfully after the goconst follow-up fix was amended into the task commit.
