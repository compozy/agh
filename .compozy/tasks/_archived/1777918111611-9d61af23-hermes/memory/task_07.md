# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implement Task 07 memory visibility: `agh memory health`, `agh memory history`, typed API/CLI paths, bounded/redacted operation history, and future context-ref/provider-hook interfaces without runtime prompt integration.
- Required grounding completed before code edits: shared workflow memory, current task memory, repo guidance, `_techspec.md`, `_tasks.md`, all Hermes ADRs, Task 02 health outputs, and Task 10 QA dependency.

## Important Decisions

- Runtime prompt assembly must remain unchanged; future context references/hooks are interface-only in this task.
- Health/history must build on Task 02 typed health patterns (`health.persistence`/`health.retention`) and existing memory/store surfaces where practical.
- Operation history reuses the existing memory catalog database and extends `memory_operation_log` with structured scope, workspace, filename, operation, summary, and timestamp fields.
- `agh memory health` and `agh memory history` are backed by direct typed API routes (`GET /api/memory/health`, `GET /api/memory/history`); `/api/observe/health` continues to include memory health through the same read model.
- Web follow-up for this slice is generated OpenAPI type refresh only; no runtime UI prompt/context-ref behavior is surfaced.

## Learnings

- Task 02 introduced typed observe health base fields and generated API/web/SDK contract updates; Task 07 API contract changes likely require `make codegen` and generated client updates.
- Shared memory records Tasks 01-06 as complete, including durable migrations and typed health foundations.
- `packages/site` memory CLI docs are manually enumerated, so new memory subcommands require explicit MDX pages plus memory `meta.json` and index updates.
- `packages/site` source generation also reflects unrelated existing network docs changes in `packages/site/lib/source.ts`; leave that unrelated change out of the Task 07 commit.

## Files / Surfaces

- Expected backend surfaces: `internal/memory`, `internal/api/contract`, `internal/api/core`, `internal/cli`, `internal/api/spec`, `internal/store/globaldb` if operation history needs a global durable table.
- Expected downstream surfaces: `openapi/agh.json`, generated web/SDK contracts, `packages/site` memory CLI docs, and Task 10 QA planning notes.
- Implemented backend/API/CLI surfaces: `internal/memory/{types,catalog,store}.go`, `internal/api/{contract,core,spec,httpapi,udsapi}`, `internal/cli/{client,memory}.go`.
- Added tests for durable history filters/redaction/bounds/restart persistence, core API health/history states, CLI command/client behavior, transport routes, and future interface boundaries.
- Documentation touched for Task 07: `packages/site/content/runtime/cli-reference/memory/{index,meta,health,history}.mdx`, `packages/site/content/runtime/core/memory/system.mdx`, `packages/site/content/runtime/api-reference/index.mdx`, `packages/site/content/runtime/core/skills/bundled.mdx`.

## Errors / Corrections

- Initial CLI compile failed because `formatOptionalTime` collided with automation helpers and the client test reused a session-history variable. Renamed the memory helper and test variable.
- Initial `make verify` failed on `gosec` G202 because the operation history query used dynamic SQL composition. Replaced it with a single static parameterized query using optional predicates.
- Self-review found disabled memory health could be reported as `unavailable` when the store was nil. `memoryHealth` now returns `disabled` before probing the store when memory is not configured.

## Ready for Next Run

- Targeted checks passed: `go test ./internal/memory ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/cli`, `make codegen-check`, `bun run --cwd web typecheck:raw`, `bun run --cwd packages/site typecheck`.
- Final verification passed: `make verify` completed successfully after the self-review fix, including lint, tests, build, web checks, and package boundary checks.
- Commit created: `26f6ab1d feat: add memory visibility surfaces`.
- Post-commit verification passed: `make verify` completed successfully with 5830 Go tests and package boundary checks.
