# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement task 01 foundation for execution sandboxes: core environment contracts, config profiles and validation/merge, workspace sandbox reference resolution/persistence/API/CLI plumbing, and session sandbox metadata.
- Status: implementation complete, tracking updated, local commit created, and post-commit verification passed.

## Important Decisions
- Use the approved task spec, TechSpec, and ADRs as the design source of truth for this implementation run.
- Follow the task-specific correction that `ToolHost` is defined with task 01 core environment contracts, while concrete ACP implementation work remains task 02.
- Keep session DB sandbox lifecycle columns out of task 01; task 01 adds only `SessionSandboxMeta` plus workspace `sandbox_ref` persistence.
- Keep generated API artifacts (`openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`) in the implementation changes because `sandbox_ref` changes the public workspace contract.

## Learnings
- Shared workflow memory currently has no durable sandbox-specific decisions beyond the PRD/TechSpec.
- Worktree already had untracked sandbox `_meta.md` and `memory/` files before code edits; preserve and update only required workflow memory/tracking files.
- `make verify` enforces generated OpenAPI freshness; API contract changes require `make codegen`.
- The final post-commit `make verify` run passed with exit code 0. Output included toolchain/environment warnings from Node `NO_COLOR`/`FORCE_COLOR` and macOS `ld -bind_at_load`, while project lint reported 0 issues.

## Files / Surfaces
- Expected surfaces: `internal/sandbox`, `internal/config`, `internal/workspace`, `internal/store`, `internal/store/globaldb`, `internal/api/contract`, `internal/api/core`, `internal/cli`, task tracking files.
- Touched implementation surfaces: `internal/sandbox`, `internal/config`, `internal/workspace`, `internal/store`, `internal/store/globaldb`, `internal/api/{contract,core,httpapi,udsapi}`, `internal/cli`, `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`.
- Also touched `internal/network/tasks.go` for a minimal `goconst` root-cause lint fix required by full `make verify`.

## Errors / Corrections
- Regenerated API artifacts after codegen freshness check failed.
- Changed environment resolution helpers to use `*config.Config` after `gocritic` flagged a large value copy.
- Added `taskIngressReasonStaleChannel` constant after `goconst` blocked full lint.
- A full race suite once exposed a nondeterministic failure in `extensions/bridges/discord`; direct `go test -race -count=1 -v ./extensions/bridges/discord` and `go test -race -count=10 ./extensions/bridges/discord` passed, and the subsequent full `make verify` passed.

## Ready for Next Run
- Task 01 implementation is committed as `9efa6efb` (`feat: add sandbox profile resolution`). Verification evidence: targeted unit tests passed; integration config/workspace/globaldb tests passed; per-package coverage targets passed; final post-commit `make verify` passed with exit code 0.
