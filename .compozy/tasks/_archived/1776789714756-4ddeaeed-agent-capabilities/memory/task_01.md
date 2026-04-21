# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add config-owned capability catalog structs, loader, validation, and agent integration for task 01, with the required unit/integration coverage and no scope expansion into runtime/network projection.

## Important Decisions
- Reuse strict JSON decode/EOF checks from `mcpjson.go`, but keep capability parsing and validation rules in a separate loader instead of treating capability files as generic blobs.
- Extend `AgentDef` with normalized capability data so later tasks consume runtime state rather than reparsing local files.
- Preserve the new capability field across workspace, daemon, and extension clone helpers so later tasks can trust loaded `AgentDef` state after resolution/resource projection hops.

## Learnings
- `LoadAgentDefFile()` currently reads `AGENT.md`, merges `mcp.json`, and validates; capability loading belongs in the same local config-owned path.
- `LoadWorkspaceAgentDefs()` already enforces workspace/additional/global precedence by loading full agent defs in order and skipping later duplicates by agent name.
- Missing capability catalogs now return `nil` from `LoadAgentCapabilities()`, while explicit file/directory catalogs normalize to a non-nil `CapabilityCatalog` with trimmed strings and strict JSON/TOML validation.

## Files / Surfaces
- `internal/config/capabilities.go`
- `internal/config/capabilities_test.go`
- `internal/config/agent_capabilities_test.go`
- `internal/config/agent.go`
- `internal/config/agent_test.go`
- `internal/config/mcpjson.go`
- `internal/config/mcpjson_test.go`
- `internal/config/agent_resource.go`
- `internal/config/agent_resource_test.go`
- `internal/workspace/clone.go`
- `internal/extension/manager.go`
- `internal/daemon/agent_skill_resources.go`

## Errors / Corrections
- Initial `make verify` run failed on local lint findings in `internal/config/capabilities.go` (`goconst`, unused fields); fixed by extracting extension constants and removing dead layout fields before rerunning the full gate.

## Ready for Next Run
- Task 01 is complete. Verification evidence:
  - `go test ./internal/config -count=1`
  - `go test ./internal/config -count=1 -coverprofile=/tmp/agh-internal-config.cover.out`
  - `make verify`
