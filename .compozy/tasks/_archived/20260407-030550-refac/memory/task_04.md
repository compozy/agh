# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Extract the task_04 domain-level dedup helpers across `internal/session`, `internal/acp`, `internal/store`, `internal/skills`, `internal/workspace`, and `internal/cli` without changing behavior.
- Verification target for this run: task-specific unit/integration coverage at or above 80% for the touched packages, plus repository-wide `make verify`.

## Important Decisions
- Canonical raw JSON cloning now lives in `internal/acp.CloneRawMessage`; session transcript/prompt code and ACP handlers both reuse it.
- Shared file snapshot state moved into the new leaf package `internal/filesnap`, with `Snapshot`, `FromPath`, `Equal`, and `Clone` as the single canonical implementation for skills and workspace.
- `GlobalDB.checkReady(ctx, action)` centralizes nil receiver / nil context validation, but `Close` keeps its existing nil-receiver behavior instead of being forced through the new helper.
- CLI list-style bundles now route through generic `listBundle[T]`; `sessionHistoryBundle` keeps its original JSON payload while reusing the shared human/toon row builder.

## Learnings
- `internal/cli` coverage landed at 79.9% after the initial refactor verification; a focused real command-path test (`agent list` / `agent info`) was enough to lift the package back to the 80% task threshold.
- Staticcheck `SA1012` flags literal `nil` contexts even in tests, so nil-context validation needs a helper-produced nil instead of a direct literal call site.

## Files / Surfaces
- `internal/session/{manager_helpers.go,manager_lifecycle.go,manager_prompt.go,transcript.go,manager_test.go}`
- `internal/acp/{handlers.go,rawjson.go,handlers_test.go}`
- `internal/store/{validation.go,types.go,global_db.go,global_db_*.go,global_db_test.go,store_helpers_test.go}`
- `internal/filesnap/{filesnap.go,filesnap_test.go}`
- `internal/skills/{types.go,registry.go,loader.go,watcher.go,loader_test.go}`
- `internal/workspace/{scanner.go,clone.go,resolver.go,resolver_test.go}`
- `internal/cli/{format.go,agent.go,memory.go,observe.go,session.go,skill.go,workspace.go,render_test.go,agent_commands_test.go}`

## Errors / Corrections
- Removed an unused `errors` import from `internal/store/types.go` after the first package test pass.
- Fixed the new CLI test to use the actual `AgentMCPServer` type instead of a non-existent record alias.
- Reworked the `checkReady` nil-context test to avoid staticcheck `SA1012`, then reran `go test ./internal/store` and `make verify`.

## Ready for Next Run
- Task 04 is complete in local code-only commit `5a60b8a` (`refactor: deduplicate domain helpers`).
- Post-commit `make verify` passed on `HEAD`, and the remaining worktree dirt is limited to workflow/tracking files outside the commit.
