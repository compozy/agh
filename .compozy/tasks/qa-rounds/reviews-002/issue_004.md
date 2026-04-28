---
status: resolved
file: internal/cli/agent.go
line: 39
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4189206979,nitpick_hash:e5724ed5c1bb
review_hash: e5724ed5c1bb
source_review_id: "4189206979"
source_review_submitted_at: "2026-04-28T13:29:54Z"
---

# Issue 004: Optional: deduplicate workspace query construction between commands.
## Review Comment

The same `commandWorkspaceFlag` + `AgentQuery` block appears twice. A tiny helper would keep command handlers slimmer and reduce drift later.

Also applies to: 73-77

## Triage

- Decision: `VALID`
- Notes:
  - Both `agent list` and `agent info` repeat the same `commandWorkspaceFlag` and `AgentQuery{Workspace: workspace}` construction.
  - The duplication is small but real; centralizing it reduces drift at the CLI boundary where workspace flag validation must stay consistent.
  - Fix approach: add a small helper that returns `AgentQuery` from the command workspace flag and use it in both handlers without changing command behavior.

## Resolution

- Added `agentQueryFromCommand` to centralize workspace flag parsing and `AgentQuery` construction.
- Updated `agent list` and `agent info` to use the shared helper without changing behavior.
- Verified with targeted `go test -race ./internal/cli -run 'TestAgentListAndInfoCommands|TestAgentCommandsPassWorkspaceQuery|TestAgentWorkspaceFlagRejectsEmptyExplicitValue' -count=1`.
- Verified the repository gate with `make verify` after code changes.
