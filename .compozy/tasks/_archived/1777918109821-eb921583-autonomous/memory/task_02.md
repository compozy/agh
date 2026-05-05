# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Task 02 by adding transport-agnostic autonomy DTOs under `internal/api/contract`, OpenAPI schema/operation registration, generated web contract updates, and contract tests proving claim-token safety. Runtime handler/business behavior remains out of scope.

## Important Decisions
- Do not expose raw `claim_token` in read/list/detail/SSE/web/channel DTOs; reserve it for the synchronous claim response contract only.
- Treat `coordination_channel_id` and channel display metadata as stable fields on agent context and claim response DTOs.
- Register `/api/agent/*` OpenAPI operations for this task without adding HTTP/UDS runtime route wiring.

## Learnings
- Task 01 handoff established `[autonomy.coordinator]` config and `RuntimeDeps.CoordinatorConfig`; Task 02 can expose coordinator config read DTOs without implementing coordinator behavior.
- Current worktree already contains pre-existing autonomy task/ADR edits and an untracked memory directory. Do not revert or stage unrelated tracking/document changes.
- Pre-change signal: local contract/spec search found no `/agent/me`, `/agent/context`, `coordination_channel_id`, `claim_token`, or `claim_token_hash` in `internal/api/contract`, `internal/api/spec`, `openapi/agh.json`, or web task/session contract consumers.
- Focused Go tests passed after DTO/spec changes: `go test ./internal/api/contract ./internal/api/core ./internal/api/spec`.
- Final verification passed after the lint fix for `NormalizeAgentContextPayload`: `make verify` completed with Go lint `0 issues`, `DONE 6017 tests`, and package-boundary checks OK.
- Focused coverage evidence: `internal/api/contract` 80.5%, `internal/api/core` 80.0%, and `internal/api/spec` 93.4%.
- Raw `claim_token` appears only in synchronous claim/command surfaces; read and lease DTOs expose `claim_token_hash`. Metadata validation rejects raw `claim_token` keys, including nested metadata.
- `make codegen` and `make codegen-check` are clean. Shared Go contract generation also updates `sdk/typescript/src/generated/contracts.ts` for the newly exported coordination/task-run DTOs.

## Files / Surfaces
- Expected implementation surfaces: `internal/api/contract`, `internal/api/spec`, `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, and affected `web/src/systems/{tasks,session}` types/fixtures if generated contracts require it.
- Added/updated: `internal/api/contract/agents.go`, `internal/api/contract/{contract.go,tasks.go,responses.go}`, `internal/api/core/agent_contracts.go`, `internal/api/spec/spec.go`, `web/src/systems/{tasks,session}/types.ts`, and task/session fixtures.
- Generated surfaces updated: `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, and `sdk/typescript/src/generated/contracts.ts`.

## Errors / Corrections

## Ready for Next Run
- Task 02 implementation, validation, tracking updates, and local code commit are complete. Code commit: `f7d9ecfb feat: add agent contract dtos`.
