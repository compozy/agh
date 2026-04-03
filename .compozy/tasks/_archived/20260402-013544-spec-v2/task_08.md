---
status: completed
domain: Kernel
type: Feature Implementation
scope: Full
complexity: medium
dependencies:
    - task_01
    - task_02
    - task_03
    - task_04
    - task_05
    - task_06
    - task_07
---

# Task 8: Multi-Session Rework of Completed Tasks

## Overview
Update the completed foundation code (types, config, NATS transport, registry) to support multi-session architecture. This introduces the Session and Kernel structs, adds session-scoped NATS subject prefixes, updates the UDS transport to use a single global socket with session routing in the payload, and splits registries into global (RoleCatalog, DriverRegistry) vs per-session (AgentRegistry, WorkgroupRegistry) ownership.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE docs/plans/2026-03-30-multi-session-design.md for all multi-session architecture decisions
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add Session struct to internal/kernel/types.go per docs/plans/2026-03-30-multi-session-design.md Session section
- MUST add Kernel struct to internal/kernel/types.go per docs/plans/2026-03-30-multi-session-design.md Kernel section
- MUST add session lifecycle states: starting, active, stopping, stopped
- MUST add config limit fields: max_sessions, max_agents_per_session, max_total_agents to LimitsConfig per docs/plans/2026-03-30-multi-session-design.md Resource Limits section
- MUST update NATS subjects to include session prefix: agh.s.{sid}.wg.{wg}.agent.{ag}, agh.s.{sid}.wg.{wg}.broadcast, etc. per docs/plans/2026-03-30-multi-session-design.md NATS Subject Hierarchy section
- MUST update ScopeValidator to include SessionID field and validate session prefix in subjects
- MUST update UDS transport to use 1 global socket (~/.agh/daemon.sock) with HTTP over UDS (Gin) and session routing in payload per docs/plans/2026-03-30-multi-session-design.md UDS Transport section
- MUST ensure RoleCatalog and DriverRegistry remain global (shared across sessions)
- MUST ensure AgentRegistry and WorkgroupRegistry are per-session (each Session owns its own instances)
- MUST implement $AGH_HOME env var support to override default ~/.agh/ location
- MUST implement config merge: global (~/.agh/config.toml) + workspace (.agh/config.toml), deep merge with workspace wins
- MUST implement roles merge: global (~/.agh/roles/) + workspace (.agh/roles/), workspace wins on name collision
- MUST add Session.Workspace field capturing CWD (or --dir override) for workspace-relative path resolution
- MUST acquire daemon.lock (gofrs/flock) before binding the UDS socket
- MUST write daemon.json with PID, socket path, and start time after successful lock
- MUST store session metadata in ~/.agh/sessions/{xid}/meta.json (workspace path, created_at, name)
- MUST maintain session index at ~/.agh/sessions/index.json for fast session listing
- MUST NOT break existing tests — update them to work with session-scoped subjects
- MUST pass `make verify`

**New dependencies:** gofrs/flock (daemon locking), gin-gonic/gin (HTTP over UDS)
</requirements>

## Subtasks
- [x] 8.1 Add Session struct (ID, Name, Goal, State, Store, AgentRegistry, WorkgroupRegistry, PtyManager, Supervisor, WsHub, NATSSubscriptions, SessionDir) and Kernel struct (NATS, UDS, HTTP, Config, RoleCatalog, DriverRegistry, SessionManager, Logger) to internal/kernel/types.go
- [x] 8.2 Add session lifecycle state type and constants (starting, active, stopping, stopped) with valid transitions
- [x] 8.3 Add config limit fields (max_sessions, max_agents_per_session, max_total_agents) to LimitsConfig with TOML tags and validation
- [x] 8.4 Update NATS subject builder to include session prefix (agh.s.{sid}.*) and update ScopeValidator with SessionID field
- [x] 8.5 Update UDS transport to single global socket (~/.agh/daemon.sock) with HTTP over UDS (Gin) and session context in request payload
- [x] 8.6 Validate registry ownership: RoleCatalog/DriverRegistry global, AgentRegistry/WorkgroupRegistry per-session; update constructors if needed
- [x] 8.7 Add Session.Workspace field and --dir flag support for workspace-relative path resolution
- [x] 8.8 Implement $AGH_HOME env var support (default ~/.agh/)
- [x] 8.9 Implement config merge (global + workspace) and roles merge (global + workspace)
- [x] 8.10 Implement daemon.lock acquisition (gofrs/flock) and daemon.json writing (PID, socket path, start time)
- [x] 8.11 Implement session metadata: meta.json per session, index.json for fast listing at ~/.agh/sessions/

## Implementation Details
Refer to docs/plans/2026-03-30-multi-session-design.md for the complete multi-session architecture. This task modifies existing code from tasks 01-07 to prepare for multi-session support without breaking current tests.

### Relevant Files
- `docs/plans/2026-03-30-multi-session-design.md` — complete multi-session design
- `docs/spec-v2/08-data-models.md` — type definitions

### Dependent Files
- `internal/kernel/types.go` — add Session and Kernel structs
- `internal/config/config.go` — add limit fields, config merge logic
- `internal/transport/nats.go` — update subject builder
- `internal/transport/uds.go` — update to global socket (~/.agh/daemon.sock) with HTTP over UDS (Gin)
- `internal/transport/scope.go` — update ScopeValidator with SessionID
- `internal/registry/agents.go` — verify per-session ownership
- `internal/registry/workgroups.go` — verify per-session ownership
- `internal/registry/roles.go` — verify global ownership, add merge logic for global + workspace
- `internal/registry/drivers.go` — verify global ownership

## Deliverables
- Updated internal/kernel/types.go with Session and Kernel structs
- Updated internal/config/ with new limit fields
- Updated internal/transport/ with session-scoped NATS subjects and global UDS
- Updated internal/transport/scope.go with SessionID in ScopeValidator
- Updated tests for all modified packages
- Unit tests with 80%+ coverage **(REQUIRED)**
- `make verify` passes

## Tests
- Unit tests:
  - [x] Session struct fields correctly initialized (ID, Name, Goal, State)
  - [x] Session state transitions: starting -> active, active -> stopping, stopping -> stopped
  - [x] Invalid state transitions rejected (e.g., stopped -> active)
  - [x] Kernel struct holds global resources (Config, RoleCatalog, DriverRegistry)
  - [x] LimitsConfig validates max_sessions > 0, max_agents_per_session > 0, max_total_agents > 0
  - [x] LimitsConfig defaults applied when fields omitted from TOML
  - [x] NATS subject builder produces agh.s.{sid}.wg.{wg}.agent.{ag} format
  - [x] NATS subject builder produces agh.s.{sid}.system.ready.{ag} format
  - [x] ScopeValidator rejects cross-session publish (agent in session A publishes to session B subject)
  - [x] ScopeValidator allows same-session publish
  - [x] UDS socket created at ~/.agh/daemon.sock (not /tmp/)
  - [x] UDS request includes session field in payload (HTTP over UDS with Gin)
  - [x] UDS routes request to correct session based on payload
  - [x] $AGH_HOME env var overrides default ~/.agh/ location
  - [x] Config merge: global ~/.agh/config.toml + workspace .agh/config.toml, workspace wins
  - [x] Roles merge: global ~/.agh/roles/ + workspace .agh/roles/, workspace wins on collision
  - [x] daemon.lock acquired before UDS socket bind
  - [x] daemon.json written with correct PID, socket path, start time
  - [x] Session.Workspace captures CWD correctly
  - [x] meta.json written per session with workspace path, created_at, name
  - [x] index.json maintained at ~/.agh/sessions/index.json
  - [x] Existing transport tests pass with updated subject format
  - [x] Existing registry tests pass without modification
- Test coverage target: >=80%
- All tests must pass with -race flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- Session and Kernel types defined with all required fields
- NATS subjects include session prefix
- ScopeValidator enforces session isolation
- UDS uses single global socket at ~/.agh/daemon.sock (HTTP over UDS with Gin)
- Config and roles merge correctly (global + workspace)
- Session metadata persisted (meta.json, index.json)
- No regressions in existing test suites
