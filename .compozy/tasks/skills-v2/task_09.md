---
status: completed
title: "Integrate MCP + hooks into daemon boot and session manager"
type: backend
complexity: high
dependencies:
  - task_04
  - task_05
  - task_07
---

# Task 9: Integrate MCP + hooks into daemon boot and session manager

## Overview

Wire the MCPResolver and HookRunner into the daemon composition root. Inject skill-resolved MCP servers into session StartOpts during Manager.Create/Resume. Add a post-notifier hook dispatch phase to notifierFanout that runs subprocess hooks after built-in notifiers. This is the integration task that connects all skills-v2 components to the runtime.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST inject skills Registry (or an active-skills resolver) as a new dependency on the session Manager
- MUST call MCPResolver.Resolve() during Manager.Create() to get skill-declared MCP servers
- MUST merge skill MCP servers with existing resolved.MCPServers via `MergeMCPServers()`
- MUST construct MCPResolver in daemon boot with config's AllowedMarketplaceMCP
- MUST add a post-notifier hook phase to notifierFanout (not an ordinary Notifier)
- MUST wire HookRunner into the hook phase with access to skills Registry and workspace resolver
- MUST resolve session's workspace for hook dispatch (Session carries WorkspaceID string, need full ResolvedWorkspace)
- MUST extend SessionManagerDeps if needed for new dependencies
- MUST preserve existing notifier ordering: built-in notifiers first, then hook phase
</requirements>

## Subtasks
- [x] 9.1 Add skills Registry dependency to session Manager (or a narrower skill-lookup interface)
- [x] 9.2 Call MCPResolver.Resolve() in Manager.Create() and merge into StartOpts.MCPServers
- [x] 9.3 Construct MCPResolver in daemon boot.go with config
- [x] 9.4 Add post-notifier hook phase to notifierFanout struct and dispatch methods
- [x] 9.5 Wire HookRunner into hook phase with skills Registry and workspace resolver access
- [x] 9.6 Write integration tests for MCP injection and hook dispatch

## Implementation Details

Changes span `internal/daemon/boot.go`, `internal/daemon/daemon.go`, `internal/daemon/notifier.go`, and `internal/session/manager_lifecycle.go`.

Key challenge (per Codex review): `Manager.Create()` does not currently call `registry.ForWorkspace()` directly — that happens in prompt assembly. MCP resolution needs its own injected dependency. Similarly, `session.Session` carries `WorkspaceID` as string, not `ResolvedWorkspace`, so hook dispatch needs the workspace resolver.

See TechSpec "Integration Points" sections 3, 4, 5.

### Relevant Files
- `internal/daemon/boot.go` — skills registry creation (line 114), notifier composition (line 193), session manager creation (line 193)
- `internal/daemon/daemon.go` — SessionManagerDeps (line 85), Daemon struct (line 134)
- `internal/daemon/notifier.go` — notifierFanout (line 10), OnSessionCreated/Stopped dispatch
- `internal/session/manager_lifecycle.go` — Manager.Create (line 16), StartOpts population (line 101)
- `internal/skills/mcp.go` — MCPResolver (from task_04)
- `internal/skills/hooks.go` — HookRunner (from task_05)

### Dependent Files
- `internal/session/interfaces.go` — may need extended Notifier or new interface
- `internal/config/provider.go` — MergeMCPServers used for combining skill + agent MCP servers

### Related ADRs
- [ADR-001: MCP Consent Model](adrs/adr-001.md) — MCPResolver enforces trust tiers at runtime
- [ADR-002: Hybrid Hook Execution Model](adrs/adr-002.md) — post-notifier hook phase ordering

## Deliverables
- Updated daemon boot.go with MCPResolver and HookRunner wiring
- Updated daemon notifier.go with post-notifier hook phase
- Updated session manager with skill MCP server injection
- Updated SessionManagerDeps if needed
- Integration tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] MCPResolver.Resolve() output merged correctly with resolved.MCPServers
  - [x] MergeMCPServers deduplicates by name (skill MCP + agent MCP)
  - [x] notifierFanout calls built-in notifiers before hook phase
  - [x] Hook phase receives correct workspace context for skill lookup
- Integration tests:
  - [x] Session created with skill declaring MCP server → MCP server in StartOpts
  - [x] Session created → on_session_created hook subprocess executed with correct payload
  - [x] Session stopped → on_session_stopped hook subprocess executed
  - [x] Hook failure does not block session creation or termination
  - [x] Marketplace skill MCP server blocked when not in consent list
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes (fmt, lint, test, build)
- Existing session lifecycle tests still pass
- MCP servers from skills appear in agent StartOpts
- Hooks fire on session create/stop without blocking
