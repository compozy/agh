# Task Memory: task_10.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Task 10 over the existing tools-refac branch: add the agent-callable `agh__mcp_auth_status` built-in, align hosted MCP projection/approval semantics with the expanded registry surface, keep MCP login/logout on CLI/HTTP/UDS management surfaces, and add status/projection/approval/redaction coverage.

## Important Decisions
- Reuse existing `internal/mcp/auth` redacted status via the current `tools.MCPAuthStatusProvider`; do not introduce new OAuth storage, browser login, logout, or token-refresh side effects in the tool path.
- Treat hosted MCP as a session projection over `Registry.Call`; approval must remain daemon-mediated and must not accept client-supplied approval tokens from hosted MCP.

## Learnings
- Pre-change signal: `rg -n "ToolIDMCPAuth|ToolsetIDMCPAuth|agh__mcp_auth_status" internal/tools internal/daemon internal/mcp -g '*.go'` returned no matches, so the redacted auth status model exists but the built-in tool/toolset is not registered yet.
- Shared workflow memory shows tasks 01 and 03-09 are already implemented on this branch; Task 10 should only add MCP auth status plus hosted MCP parity tests/fixes.
- Existing `internal/mcp.CallExecutor` already implements `tools.MCPAuthStatusProvider`; daemon registry boot only needs to expose that executor through native tool deps after MCP provider construction.
- Existing hosted MCP already projects `Registry.List(record.scope())` and dispatches `Registry.Call(record.scope(), ...)` without a hosted `ApprovalToken`; parity risk is coverage, plus ensuring the new status built-in participates in normal session projection.
- Focused coverage evidence after implementation: `internal/tools` 80.8%, `internal/tools/builtin` 93.3%, `internal/mcp` 80.7%; `internal/daemon` remains package-wide 73.8% because it is a broad daemon package, while the new `native_mcp_auth_tools.go` handler functions cover 87.5%/100%/100%.
- Pre-commit `make verify` passed after all code/test changes with Go lint `0 issues`, `DONE 7094 tests`, and package boundaries OK.
- Code/test commit is `5fa9f805 feat: add mcp auth status tool`; post-commit `make verify` passed with lint `0 issues`, Vitest `257` files / `1838` tests, Go `DONE 7094 tests`, and package boundaries OK.

## Files / Surfaces
- Expected code surfaces: `internal/tools/mcp.go`, `internal/tools/builtin*`, `internal/daemon/native_tools*`, `internal/mcp/hosted.go`, `internal/daemon/tool_approval_bridge.go`, and focused tests.
- Management repair path to preserve: `internal/cli/mcp_auth.go` plus existing HTTP/UDS/settings auth status conversions.
- Actual edit set: built-in IDs/descriptors/toolsets, daemon native auth-status handler/provider wiring, result redaction exemption for public `token_present`, builtin/native/hosted MCP tests, CLI/API parity checks.

## Errors / Corrections
- Generic result redaction treated `token_present` as sensitive because the normalized field contains `token`; corrected by allowing only that exact public diagnostic field while keeping OAuth/token/secret field redaction intact.

## Ready for Next Run
- Task 10 is implemented, tracked as completed, committed locally as `5fa9f805`, and post-commit verified. Tracking/memory artifacts remain uncommitted by workflow instruction.
