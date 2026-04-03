---
status: completed
domain: Kernel
type: Feature Implementation
scope: Full
complexity: medium
dependencies:
    - task_08
    - task_14
---

# Task 15: SessionManager

## Overview
Implement the SessionManager that creates, tracks, and manages multiple concurrent sessions within a single kernel process. The SessionManager owns a map[string]*Session protected by sync.RWMutex, enforces resource limits (max_sessions, max_agents_per_session, max_total_agents), handles session naming (--name flag, auto-slug from goal, xid suffix on collision), manages session lifecycle (starting -> active -> stopping -> stopped), creates session directories (~/.agh/sessions/{xid}/), captures the workspace directory (CWD or --dir flag), loads workspace-specific config and roles overrides, writes per-session meta.json, maintains ~/.agh/sessions/index.json, and supports session resume from persisted state.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE docs/plans/2026-03-30-multi-session-design.md for SessionManager architecture
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement SessionManager struct with map[string]*Session protected by sync.RWMutex per docs/plans/2026-03-30-multi-session-design.md
- MUST implement Create method: validate limits, capture workspace (CWD or --dir flag), generate session ID (xid), resolve name, create session directory at ~/.agh/sessions/{xid}/, write goal.md, write meta.json (workspace path, created_at, name), update ~/.agh/sessions/index.json, load workspace config override (.agh/config.toml from workspace), merge config (global + workspace), load workspace roles (.agh/roles/ from workspace), merge roles (global + workspace), open SQLite database, start writer goroutine, init per-session registries, init PTY manager, create suture supervisor, create NATS subscriptions, start health check, register session, spawn supervisor + advisor, set state to active
- MUST implement Start method: transition session from starting to active
- MUST implement Stop method: transition to stopping, stop suture tree, close PTYs, drain channels, unsubscribe NATS, close WebSocket connections, drain writer, checkpoint SQLite, close DB, deregister, update index.json, set state to stopped
- MUST implement Resume method: find session directory on disk (~/.agh/sessions/{xid}/), read goal.md, read meta.json for workspace path, open existing SQLite, create new session with same name, inject historical context into supervisor prompt
- MUST implement List method: return all active sessions (fast path: read index.json)
- MUST implement Get method: return session by name or ID
- MUST enforce max_sessions limit on Create with error "max sessions reached (N/N). Stop a session first."
- MUST enforce max_agents_per_session and max_total_agents limits
- MUST implement session naming: --name flag verbatim, else slug from goal text, append xid suffix on collision
- MUST create session directory at ~/.agh/sessions/{xid}/ with goal.md, meta.json, and session.db
- MUST write meta.json per session: {workspace, name, id, created_at, goal}
- MUST maintain ~/.agh/sessions/index.json with all sessions for fast listing
- MUST load workspace-specific config and roles on session creation (merge with global)
- MUST support --dir flag to override CWD as workspace directory
- MUST use context.Context for all lifecycle operations with proper cancellation
</requirements>

## Subtasks
- [x] 15.1 Implement SessionManager struct with map[string]*Session, sync.RWMutex, and reference to kernel global resources
- [x] 15.2 Implement Create method with full session start sequence (capture workspace, validate limits, generate ID, resolve name, create ~/.agh/sessions/{xid}/, write meta.json, update index.json, load workspace config/roles, merge with global, init subsystems, spawn bootstrap agents)
- [x] 15.3 Implement Stop method with full session stop sequence (stop supervisor tree, close PTYs, drain channels, close DB, update index.json)
- [x] 15.4 Implement Resume method (find session directory at ~/.agh/sessions/{xid}/, read goal.md + meta.json, restore workspace, open existing SQLite, inject historical context)
- [x] 15.5 Implement List and Get methods for session lookup (List reads index.json for fast path)
- [x] 15.6 Implement session naming logic (--name flag, auto-slug, xid suffix on collision)
- [x] 15.7 Implement resource limit enforcement (max_sessions, max_agents_per_session, max_total_agents)
- [x] 15.8 Implement workspace capture (CWD or --dir flag) and workspace config/roles loading
- [x] 15.9 Implement meta.json and index.json persistence

## Implementation Details
Refer to docs/plans/2026-03-30-multi-session-design.md for session lifecycle (Start, Stop, Resume sections), session naming rules, and resource limit enforcement.

### Relevant Files
- `docs/plans/2026-03-30-multi-session-design.md` — SessionManager spec, session lifecycle, naming, limits
- `docs/spec-v2/09-resilience.md` — graceful shutdown per session

### Dependent Files
- `internal/kernel/types.go` — Session and Kernel structs (from task_08)
- `internal/kernel/kernel.go` — Kernel boot (from task_14)
- `internal/config/` — LimitsConfig with session limits
- `internal/state/` — SQLite store per session
- `internal/registry/` — per-session registries
- `internal/pty/` — per-session PTY manager
- `internal/transport/` — NATS subscriptions scoped to session

## Deliverables
- internal/session/manager.go — SessionManager struct with Create, Start, Stop, Resume, List, Get methods
- internal/session/naming.go — session naming logic (slug generation, collision handling)
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for session lifecycle **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Create captures workspace from CWD
  - [x] Create captures workspace from --dir flag when provided
  - [x] Create initializes session with correct ID, name, goal, workspace, state=starting
  - [x] Create creates session directory at ~/.agh/sessions/{xid}/
  - [x] Create writes goal.md with goal text
  - [x] Create writes meta.json with workspace, name, id, created_at, goal
  - [x] Create updates ~/.agh/sessions/index.json
  - [x] Create loads workspace config (.agh/config.toml) and merges with global config
  - [x] Create loads workspace roles (.agh/roles/) and merges with global roles
  - [x] Create opens SQLite database at ~/.agh/sessions/{xid}/session.db
  - [x] Create transitions session state to active after full initialization
  - [x] Create rejects when max_sessions reached with correct error message
  - [x] Create rejects when max_total_agents would be exceeded
  - [x] Stop transitions session through stopping -> stopped
  - [x] Stop closes all session resources (PTY, NATS, SQLite, WebSocket)
  - [x] Stop updates index.json
  - [x] Resume finds existing session directory at ~/.agh/sessions/{xid}/ and reads goal.md + meta.json
  - [x] Resume restores workspace path from meta.json
  - [x] Resume opens existing SQLite database (read historical state)
  - [x] Resume creates new session with historical context in supervisor prompt
  - [x] List returns all active sessions (reads index.json)
  - [x] Get returns session by name
  - [x] Get returns session by ID
  - [x] Get returns error for non-existent session
  - [x] Session naming: --name flag used verbatim
  - [x] Session naming: goal text slugified correctly ("Build REST API" -> "build-rest-api")
  - [x] Session naming: xid suffix appended on name collision
  - [x] Concurrent Create/Stop operations are thread-safe (no data races)
- Integration tests:
  - [x] Full lifecycle: Create -> verify active -> Stop -> verify stopped
  - [x] Multiple concurrent sessions created and managed independently
  - [x] Session stop cleans up all resources without leaks
- Test coverage target: >=80%
- All tests must pass with -race flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- SessionManager correctly manages multiple concurrent sessions
- Resource limits enforced with clear error messages
- Session naming follows priority: --name > slug > slug+xid
- Session directories created at ~/.agh/sessions/{xid}/
- meta.json and index.json correctly maintained
- Workspace config/roles loaded and merged with global on session creation
- No data races under concurrent access
