---
status: completed
title: "Implement MCPResolver with trust-tier filtering"
type: backend
complexity: medium
dependencies:
  - task_01
---

# Task 4: Implement MCPResolver with trust-tier filtering

## Overview

Create the `MCPResolver` that collects MCP server declarations from active skills and applies trust-tier filtering per ADR-001. Bundled, user, additional, and workspace skills are auto-approved. Marketplace skills require explicit consent via config allowlist. Output is `[]aghconfig.MCPServer` ready for `StartOpts`.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `MCPResolver` struct in `internal/skills/mcp.go` with `allowedMarketplace []string` and logger
- MUST implement `Resolve(skills []*Skill) []aghconfig.MCPServer` method
- MUST auto-approve MCP servers from SourceBundled, SourceUser, SourceAdditional, SourceWorkspace
- MUST block MCP servers from SourceMarketplace unless the skill name is in the allowlist
- MUST convert `MCPServerDecl` to `aghconfig.MCPServer` (matching Name, Command, Args, Env fields)
- MUST deduplicate MCP servers by name (later skill in precedence order wins)
- MUST log blocked servers at Warn level with skill_name, mcp_server, source fields
- MUST log resolved servers at Info level
</requirements>

## Subtasks
- [x] 4.1 Create MCPResolver struct with constructor accepting config and logger
- [x] 4.2 Implement Resolve() with trust-tier logic and MCPServerDecl → MCPServer conversion
- [x] 4.3 Add name-based deduplication (higher precedence wins)
- [x] 4.4 Add structured logging for resolved and blocked servers
- [x] 4.5 Write unit tests for all trust tier scenarios

## Implementation Details

New file `internal/skills/mcp.go`. Converts `MCPServerDecl` (skill-defined) to `aghconfig.MCPServer` (ACP-compatible). The resolver is later injected into the session manager path (task_09).

See TechSpec "MCPResolver" and "Data Flow — MCP Lazy-Load" sections.

### Relevant Files
- `internal/skills/types.go` — MCPServerDecl, SkillSource, Skill struct
- `internal/config/provider.go` — MCPServer struct (lines 17-23), MergeMCPServers (line 181)

### Dependent Files
- `internal/session/manager_lifecycle.go` — will call MCPResolver.Resolve() (task_09)
- `internal/daemon/boot.go` — constructs MCPResolver (task_09)

### Related ADRs
- [ADR-001: MCP Consent Model](adrs/adr-001.md) — defines the trust tier rules this task implements

## Deliverables
- `internal/skills/mcp.go` with MCPResolver
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Bundled skill with MCP server → auto-approved, returned in output
  - [x] User skill with MCP server → auto-approved
  - [x] Additional skill with MCP server → auto-approved
  - [x] Workspace skill with MCP server → auto-approved
  - [x] Marketplace skill with MCP server, NOT in allowlist → blocked, warning logged
  - [x] Marketplace skill with MCP server, IN allowlist → approved
  - [x] Skill with no MCPServers → no output for that skill
  - [x] Duplicate MCP server name across skills → higher-precedence skill wins
  - [x] Empty skill list → empty output
  - [x] MCPServerDecl correctly converted to aghconfig.MCPServer fields
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make lint` passes
- Trust tier logic matches ADR-001 exactly
