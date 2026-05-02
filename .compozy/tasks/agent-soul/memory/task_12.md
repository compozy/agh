# Task Memory: task_12.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add agent-operable CLI commands for managed Soul authoring, managed Heartbeat authoring/status/wake, and session health/status/inspect without direct filesystem mutation.
- Preserve UDS/API route semantics from Task 11 and body-level `expected_digest` CAS behavior from Task 10/11.

## Important Decisions
- CLI commands will use Task 10 contract DTO aliases through `DaemonClient`; no CLI-only response/request shapes.
- Add a global `--json` flag as a deterministic alias for the existing `-o json` output mode because the Task 12 spec explicitly names `--json`.
- Keep `agh session heartbeat` absent; session runtime state is exposed through `session health`, `session status`, and `session inspect`.
- Add the operator-scoped `POST /api/agents/{name}/soul/validate` route so CLI validation uses the same agent-management route family as inspect/write/delete/history/rollback.
- Preserve workspace-resolved `[agents]` config in `workspace.cloneConfig`; Soul/Heartbeat authoring through workspaces depends on these defaults/overlays.

## Learnings
- Baseline `go run ./cmd/agh agent soul --help` falls through to the existing `agent` parent help, confirming the Task 12 command tree is absent.
- Task 11 route memory says Soul validate exists, but local route review shows only the agent-caller `/api/agent/soul/validate` path; operator CLI validation may require an agent-scoped validate route that reuses shared core logic.
- Focused CLI integration exposed that managed authoring rejects global-home agent definitions for workspace writes with `path_escape`; integration coverage now uses a workspace-local `.agh/agents/<name>/AGENT.md` definition to exercise the intended managed mutation path.

## Files / Surfaces
- Touched: `internal/cli/{agent,session,client,format,root,authored_context}.go`, CLI tests/integration harness, `internal/api/{core,httpapi,udsapi,spec}/authored context` route/spec surfaces, generated OpenAPI/web types, and `internal/workspace/{clone.go,resolver_test.go}`.

## Errors / Corrections
- Corrected `workspace.cloneConfig` dropping `Agents`, which produced zero-valued Soul/Heartbeat config in resolved workspaces and surfaced as `agents.soul.max_body_bytes must be positive: 0`.
- Corrected the new integration scenario from global agent authoring to workspace-local authoring to satisfy managed path boundaries.
- Corrected the first full-gate lint pass by passing large CLI output records by pointer, removing a redundant workspace helper argument, and splitting long CLI declarations.

## Verification
- `go test ./internal/cli -run 'TestAgentSoulCommands|TestAgentHeartbeatCommands|TestSessionAuthoredContextCommands|TestSessionStatusReturnsHealthStatus|TestCommandPathsAndHelpers' -count=1` passed.
- `go test -tags integration ./internal/cli -run TestCLIAgentAuthoredContextIntegration -count=1 -timeout=90s` passed.
- `make lint` passed with `0 issues.`
- `make codegen-check` passed after OpenAPI and web type regeneration.
- `make cli-docs` passed and generated CLI reference pages for the new command groups.
- `make verify` passed after all implementation/tracking changes: Bun tests `264` files / `1872` tests, Go gate `7693` tests, and package boundaries OK.
- Fresh pre-commit `make verify` after task tracking updates also passed: Bun tests `264` files / `1872` tests, Go gate `7693` tests, and package boundaries OK.
- Created local code-only commit `2ee510ec` (`feat: add authored context CLI commands`).
- Post-commit `make verify` passed: Bun tests `264` files / `1872` tests, Go gate `7693` tests, and package boundaries OK.

## Ready for Next Run
- Task 12 implementation, tracking, commit, and post-commit verification are complete.
