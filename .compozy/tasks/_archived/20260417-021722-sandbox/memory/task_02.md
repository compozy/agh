# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Extract ACP local subprocess launch and local file/terminal/permission side effects behind Task 01 `internal/sandbox` Launcher/ToolHost interfaces while preserving local ACP behavior.
- Completion needs existing ACP tests unchanged, new local launcher/toolhost behavior tests, coverage evidence, `make verify`, tracking updates, and one local commit.

## Important Decisions
- Use Task 01 `internal/sandbox` interfaces from ACP rather than redefining duplicate ACP-local contracts.
- Keep ACP protocol request/response shaping in `AgentProcess`; move file IO, permission policy evaluation/path resolution, and terminal process side effects into `localToolHost`.
- Keep task scope to driver-level `WithLauncher()` and `WithToolHost()` injection; `StartOpts` stays protocol/session input only.
- Add `Launcher` to `sandbox.Prepared` so Task 03 can return the local launcher alongside `Launch` and `ToolHost`.

## Learnings
- Pre-change ACP still hardcodes local launch in `Driver.spawnProcess` and direct OS/terminal behavior in `AgentProcess` handlers.
- Task 01 already added `internal/sandbox.Launcher`, `Handle`, `LaunchSpec`, `ToolHost`, permission operation, and permission decision types.
- Final ACP coverage is 81.0% with the new local launcher/toolhost tests.

## Files / Surfaces
- Source surfaces touched: `internal/acp/client.go`, `internal/acp/types.go`, `internal/acp/permission.go`, `internal/acp/handlers.go`, `internal/acp/launcher.go`, `internal/acp/tool_host.go`, `internal/acp/launcher_tool_host_test.go`, `internal/acp/client_test.go`, `internal/acp/client_integration_test.go`, and `internal/sandbox/types.go`.

## Errors / Corrections
- Initial `make verify` found a staticcheck context issue and `Driver.Stop` gocyclo >20; fixed by using a real test context and extracting stop helpers.
- A subprocess-based launcher override test was flaky under race due process shutdown; replaced with an in-process fake handle that preserves transport lifecycle.
- Self-review removed a per-start `Launcher`/`ToolHost` `StartOpts` expansion to honor the task's driver-option scope.

## Ready for Next Run
- Task 02 implementation commit: `5ea386ea` (`refactor: extract acp launcher and tool host`).
- Relevant commands passed: `go test ./internal/acp`, `go test -race ./internal/acp`, `go test -cover ./internal/acp` (81.0%), `go test -tags integration ./internal/acp`, `make verify`, and `env -u NO_COLOR make verify`.
