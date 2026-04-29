# Task Memory: task_09.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement daemon-owned MCP backend call-through for Task 09: preserve MCP server transport/auth metadata, normalize external MCP tool descriptors, execute stdio/HTTP/SSE calls through `internal/mcp`, map redacted auth status to registry reasons, and verify with focused unit/integration coverage plus full `make verify`.

## Important Decisions
- Keep `internal/mcp/auth` as the durable auth owner. The executor may read token material only inside `internal/mcp` for header injection; `internal/tools`, daemon APIs, events, and CLI-facing DTOs stay redacted.
- Use `mark3labs/mcp-go` client constructors exactly as specified: stdio via `client.NewStdioMCPClient`, streamable HTTP via `client.NewStreamableHttpClient`, and SSE via `client.NewSSEMCPClient`.
- Add the new dependency with `go get github.com/mark3labs/mcp-go@v0.49.0`; `mcp-go` itself requires `go 1.25.5`, so the module Go directive bump is dependency-driven.
- Use concrete type `internal/mcp.CallExecutor` to satisfy Go lint stutter rules; `NewMCPCallExecutor` constructs it and it implements `tools.MCPCallExecutor`.
- Redact refresh failures at the package boundary: `ensureAuthorized` returns `mcp_auth_refresh_failed` without wrapping upstream refresh errors that might contain token material.

## Learnings
- Pre-change signal: `rg` finds no MCP call-through constructor usage or MCP provider wiring under `internal/mcp`, `internal/tools`, or `internal/daemon`.
- Pre-change signal: `go list -m github.com/mark3labs/mcp-go` fails with `not a known dependency`.
- Existing `cloneDaemonMCPServer` preserves only `Name`, `Command`, `Args`, and `Env`; it currently drops `Transport`, `URL`, and `Auth`.
- Existing `MCPToolDescriptor` has `InputSchema` but no `OutputSchema` or embedded `SourceRef`.
- Focused verification evidence so far: AGH test-shape checks pass for `internal/mcp/executor_test.go`, `internal/tools/mcp_test.go`, and `internal/daemon/tool_mcp_resources_clone_test.go`.
- Focused Go tests pass: `go test ./internal/mcp ./internal/tools ./internal/daemon -count=1`; race pass: `go test -race ./internal/mcp ./internal/tools ./internal/daemon -count=1`.
- `internal/mcp` package coverage is 81.8% from `go test ./internal/mcp -coverprofile=/tmp/agh-task09-mcp.cover -count=1`.
- `make lint` passes with `0 issues.` after renaming the concrete executor and fixing gocritic/staticcheck findings.
- Full `make verify` exposed scaffold/temp-module drift caused by the new dependency-driven root `go 1.25.5` directive: the Go create-extension template, SDK external-consumer fixture, and extension integration temp module still used `go 1.25.4`, which made generated modules require `go mod tidy` immediately after replacing AGH with the local repo.
- Final verification evidence before tracking: `git diff --check && make verify` passed with `Found 0 warnings and 0 errors.`, `257` Vitest files / `1838` Vitest tests passing, frontend build passing, `DONE 6843 tests in 25.037s`, and `OK: all package boundaries respected`.
- Local commit created: `51ab3547` (`feat: add daemon mcp call-through`).
- Post-commit verification evidence: `make verify` passed with `Found 0 warnings and 0 errors.`, `257` Vitest files / `1838` Vitest tests passing, frontend build passing, `DONE 6843 tests in 6.331s`, and `OK: all package boundaries respected`.

## Files / Surfaces
- `internal/daemon/tool_mcp_resources.go` for MCP server resource/config clone preservation.
- `internal/tools` for MCP canonical ID normalization, provider adapter, auth reason mapping, and `MCPToolDescriptor` contract.
- `internal/mcp` for daemon-owned executor and auth/header injection.
- `internal/daemon/native_tools.go` for registry provider composition.
- `go.mod` / `go.sum` for `mark3labs/mcp-go` and transitive module metadata.

## Errors / Corrections
- Initial MCP executor integration test hung because the timeout fake server held an active request during cleanup; replaced with a bounded timer/select fake handler.
- Initial expected canonical ID for `GitHub` incorrectly assumed camel-case splitting; corrected to case-folding only (`mcp__github__echo`).
- `make lint` rejected `mcp.MCPCallExecutor` / `MCPCallExecutorOption` as stuttering names; concrete type/options are now `CallExecutor` / `CallExecutorOption`.
- Corrected Go scaffold/temp-module version drift by aligning generated Go modules, SDK external-consumer fixture, and extension integration temp module with the root `go 1.25.5` directive.

## Ready for Next Run
- Task implementation, tracking, scoped local commit, and post-commit verification are complete.
