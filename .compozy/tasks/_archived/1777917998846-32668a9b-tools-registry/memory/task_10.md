# Task Memory: task_10.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement hosted AGH MCP session exposure for Task 10: session-bound bind nonce lifecycle, ACP stdio injection, `agh tool mcp --session --bind-nonce`, UDS peer/binary validation, live session projection parity, and approval bridge errors.
- Success requires all task unit/integration tests, >=80% relevant package coverage, full `make verify`, tracking updates, and one local commit after clean verification/self-review.

## Important Decisions
- Approved PRD/TechSpec/ADRs are the design source of truth; no separate brainstorming design doc is needed for this execution run.
- Hosted MCP is only the model/session exposure transport; all calls must re-enter `internal/tools.Registry.Call` and must not bypass registry policy, approval, hooks, redaction, or result budgets.
- The approval bridge will live inside `internal/tools.RuntimeRegistry` behind an injected interface so hosted MCP cannot make approval-required calls executable by setting `ApprovalAvailable=true` without an actual ACP/session permission path.
- ACP `mcpServers` injection will be narrowed to the single AGH-hosted stdio entry; existing remote HTTP/SSE MCP configs must stay daemon-owned and must not be converted into ACP stdio entries.
- UDS peer validation is implemented as fail-closed. Darwin uses `LOCAL_PEERPID` + `proc_pidpath` via cgo, Linux uses `SO_PEERCRED` + `/proc/<pid>/exe`, and unsupported builds return explicit unsupported validation errors.
- Hosted proxy projection changes use `mcp-go` `MCPServer.SetTools` so the library owns tool update/list_changed behavior; tests assert the next `tools/list` matches the updated projection.

## Learnings
- Shared memory confirms Task 09 completed daemon-owned external MCP call-through in local commit `51ab3547`; hosted MCP must build on it without exposing remote MCP servers directly through ACP.
- ADR-011 requires `mark3labs/mcp-go v0.49.0`, explicit hosted `mcp.Tool.RawInputSchema`/`RawOutputSchema`, and no hand-rolled MCP protocol/framing.
- Baseline gap: no existing `agh tool mcp` command or hosted MCP proxy code exists; only the Task 09 remote MCP executor is present.
- Baseline gap: `internal/acp/client.go` currently converts every config `MCPServer` to `McpServerStdio`, which would misrepresent HTTP/SSE remotes.
- Baseline gap: `RuntimeRegistry.dispatch` currently has no approval wait hook before provider execution; approval-required callable tools need an injected bridge before hosted exposure can be enabled.
- `mcp-go` v0.49.0 supports `server.NewMCPServer`, `server.NewStdioServer(...).Listen(ctx, in, out)`, `WithToolCapabilities(true)`, `MCPServer.SetTools`, and direct `mcp.Tool.RawInputSchema`/`RawOutputSchema`.
- Darwin can provide UDS peer PID via `LOCAL_PEERPID`; executable path requires libproc `proc_pidpath` or fail-closed behavior when unavailable.
- ACP SDK `mcpServers` supports HTTP/SSE variants, but AGH start payloads now intentionally inject only hosted stdio entries; remote configured MCP backends remain registry-side call-through providers.
- acpmock diagnostics now records session lifecycle `mcp_servers` only when non-empty, preserving existing prompt diagnostics while enabling deterministic hosted MCP observation.

## Files / Surfaces
- Active surfaces: `internal/acp`, `internal/session`, `internal/mcp`, `internal/tools`, `internal/cli`, `internal/api/udsapi`, `internal/daemon`, `internal/testutil/acpmock`, and `mcp-go` local module APIs.
- Implemented surfaces include hosted MCP service/proxy/peer files, UDS internal hosted routes, hidden `agh tool mcp`, daemon composition, session launch injection, ACP permission bridge, registry approval hook, and acpmock MCP diagnostics.

## Errors / Corrections
- Corrected a test fixture that attempted to pass an invalid blank stdio MCP server through `acp.Start`; ACP validates `StartOpts` before conversion, so remote HTTP/SSE skip behavior is asserted with valid remote entries.
- The stdio proxy integration test could not deterministically observe the library notification callback, so it asserts the AGH-owned projection stream result: after `SetTools`, `tools/list` returns the updated projection.
- Corrected approval outcome handling to avoid direct ACP SDK `Cancelled` field access because the repo lint auto-fix rewrites that spelling to `Canceled`; the bridge now treats a validated non-selected outcome as canceled without relying on the field name.
- Fixed Go lint issues found by full verification: checked hosted release/projection stream errors, avoided large `ToolView` value passing through approval bridge APIs, handled projection digest marshal errors, and removed redundant Darwin cgo conversions/imports.

## Ready for Next Run
- Focused tests passed: `go test ./internal/mcp ./internal/acp ./internal/session ./internal/testutil/acpmock ./internal/testutil/acpmock/cmd/acpmock-driver ./internal/daemon ./internal/tools ./internal/api/udsapi ./internal/cli`.
- Coverage evidence: `go test -coverprofile=/tmp/agh-mcp.cover ./internal/mcp` reports 81.0% statement coverage for `internal/mcp`.
- Full verification passed after self-review corrections: `make verify` exited 0 with `Found 0 warnings and 0 errors.`, `257` Vitest files / `1838` Vitest tests passing, `DONE 6880 tests in 46.208s`, and `OK: all package boundaries respected`.
- Local code commit created: `af576477` (`feat: expose hosted session mcp tools`).
- Post-commit `make verify` exited 0 with `Found 0 warnings and 0 errors.`, `257` Vitest files / `1838` Vitest tests passing, `DONE 6880 tests in 10.748s`, and `OK: all package boundaries respected`.
- Task 10 tracking is marked complete in `task_10.md` and `_tasks.md`; tracking/memory artifacts remained out of the code commit as required.
- Final handoff is ready. Remaining dirty worktree entries are `.compozy` tracking/spec/memory/QA artifacts only.
