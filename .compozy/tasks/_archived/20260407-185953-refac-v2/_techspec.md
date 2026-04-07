# TechSpec: Refac V2 — Package Graph Reorganization

## Executive Summary

No `_prd.md` exists for `.compozy/tasks/refac-v2/`. This TechSpec therefore uses the `20260406-summary.md` report set, `20260406-codex-review.md`, and live codebase inspection as the source of truth. Where the analysis documents and the live tree differ, the live repository state wins.

`refac-v2` targets a broad package-graph reorganization rather than a path-preserving cleanup. The target architecture introduces `internal/api/contract` as the canonical shared daemon API surface, re-roots transport code under `internal/api/*`, extracts `internal/frontmatter` and `internal/transcript`, moves dream orchestration into `internal/memory/consolidation`, and splits persistence into `internal/store/sessiondb` and `internal/store/globaldb`. The primary technical trade-off is higher short-term import churn and tighter rollout discipline in exchange for clearer package ownership, reduced parser/DTO duplication, and lower cross-boundary drift risk.

## System Architecture

### Component Overview

The target architecture is:

- `internal/api/contract`
  - Owns transport-agnostic request and response DTOs shared by CLI, HTTP, and UDS consumers.
- `internal/api/core`
  - Owns shared handler logic, query parsing, SSE helpers, error shaping, and transport-facing service interfaces.
  - Absorbs the responsibilities currently split between `internal/apicore` and `internal/apisupport`.
- `internal/api/httpapi`
  - Owns HTTP server lifecycle, route registration, static asset serving, CORS behavior, and HTTP-only payload details.
- `internal/api/udsapi`
  - Owns UDS server lifecycle, route registration, and UDS-only transport behavior.
- `internal/api/testutil`
  - Owns shared API test harnesses, stubs, request helpers, and SSE assertions used by transport and CLI tests.
- `internal/frontmatter`
  - Owns line-ending normalization, delimiter detection, shared error sentinels, and generic frontmatter decoding for `config`, `memory`, and `skills`.
- `internal/transcript`
  - Owns canonical transcript assembly from persisted session events.
  - `session` no longer owns replay-specific message shaping.
- `internal/store`
  - Keeps shared persistence types, narrow interfaces, validation, and minimal SQLite helpers.
- `internal/store/sessiondb`
  - Owns per-session SQLite persistence, writer-loop lifecycle, token usage writes, event queries, and turn-history queries.
- `internal/store/globaldb`
  - Owns global registry persistence, workspace storage, observability summaries, permission logs, and token stats.
- `internal/memory/consolidation`
  - Owns dream gating, lock management, consolidation prompt orchestration, and daemon-triggered scheduling.
- `internal/daemon`
  - Remains the sole composition root and wiring layer.
  - It must not retain dream domain logic after the move.
- `internal/session`, `internal/workspace`, `internal/skills`, `internal/observe`, `internal/config`
  - Remain top-level domain packages, but depend on narrower shared boundaries.

Primary data flow in the target design:

- CLI uses `internal/api/contract` and talks to `internal/api/udsapi`.
- HTTP and UDS transports both depend on `internal/api/core`.
- `api/core` depends on runtime services in `session`, `workspace`, `observe`, `memory`, and shared persistence interfaces from `store`.
- `session` writes event data to `store/sessiondb` and relies on `transcript` for replay assembly.
- `observe`, `workspace`, and daemon status surfaces read from `store/globaldb`.
- `config`, `skills`, and `memory` all consume `frontmatter` rather than carrying local parsers.

## Implementation Design

### Core Interfaces

```go
package contract

type SessionRecord struct {
	ID            string    `json:"id"`
	Name          string    `json:"name,omitempty"`
	AgentName     string    `json:"agent_name"`
	WorkspaceID   string    `json:"workspace_id,omitempty"`
	WorkspacePath string    `json:"workspace_path,omitempty"`
	State         string    `json:"state"`
	ACPSessionID  string    `json:"acp_session_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
```

```go
package store

type SessionCatalog interface {
	RegisterSession(ctx context.Context, session SessionInfo) error
	UpdateSessionState(ctx context.Context, update SessionStateUpdate) error
	ListSessions(ctx context.Context, query SessionListQuery) ([]SessionInfo, error)
}

type EventRecorder interface {
	Record(ctx context.Context, event SessionEvent) error
	Query(ctx context.Context, query EventQuery) ([]SessionEvent, error)
	History(ctx context.Context, query EventQuery) ([]TurnHistory, error)
}
```

```go
package transcript

type Assembler interface {
	Assemble(events []store.SessionEvent) ([]Message, error)
}

type Message struct {
	ID        string    `json:"id"`
	Role      Role      `json:"role"`
	Content   string    `json:"content"`
	ToolName  string    `json:"tool_name,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}
```

Error handling conventions:

- All cross-package methods return wrapped errors with package context.
- Shared sentinel errors stay in the owning domain package and are matched with `errors.Is`.
- Transport packages remain responsible for mapping domain errors to status codes; `api/core` owns shared mapping helpers.

### Data Models

Core technical entities in the target design are:

- `api/contract`
  - Session DTOs: `SessionRecord`, `SessionEventRecord`, `TurnHistoryRecord`, transcript response DTOs.
  - Agent DTOs: `AgentRecord`, MCP server DTOs.
  - Workspace DTOs: create/update requests, `WorkspaceRecord`, workspace detail payloads.
  - Observe DTOs: event summary records, health payloads.
  - Memory DTOs: list/read/write/delete/consolidate request and response types.
  - Daemon DTOs: daemon status payloads and permission approval payloads.
- `transcript`
  - `Message`, `ToolResult`, and assembly helpers that translate persisted session events into the canonical replay shape.
- `store/sessiondb`
  - Owns per-session `events.db`.
  - Persists ordered `SessionEvent` rows, token usage rows, and turn history queries keyed by session and sequence.
- `store/globaldb`
  - Owns `agh.db`.
  - Persists session index rows, workspace registrations, event summaries, permission logs, and token stats.
- `frontmatter`
  - Decodes a metadata header plus body content from Markdown-like files.
  - Used by `config` for `AGENT.md`, `skills` for `SKILL.md`, and `memory` for memory documents.
- `workspace`
  - Continues to own `Workspace`, `ResolvedWorkspace`, and workspace-facing sentinel errors.
  - `globaldb` stores persisted registrations; `workspace` owns resolution logic and runtime snapshots.

### API Endpoints

The refactor does not change the external daemon API paths. It changes ownership of request and response types and handler placement, not the public route surface.

Sessions:

- `GET /api/sessions`
  - Lists sessions.
  - Response: `contract.SessionRecord[]`
  - Status: `200`
- `POST /api/sessions`
  - Creates a session.
  - Request: `contract.CreateSessionRequest`
  - Response: `contract.SessionRecord`
  - Status: `201`, `400`, `409`
- `GET /api/sessions/:id`
  - Fetches one session.
  - Response: `contract.SessionRecord`
  - Status: `200`, `404`
- `DELETE /api/sessions/:id`
  - Stops a session.
  - Status: `204`, `404`
- `POST /api/sessions/:id/resume`
  - Resumes a session.
  - Response: `contract.SessionRecord`
  - Status: `200`, `404`, `409`
- `POST /api/sessions/:id/prompt`
  - Sends a prompt and streams agent events.
  - Request: transport-specific prompt request plus shared event DTOs.
  - Status: `200`, `400`, `404`
- `POST /api/sessions/:id/approve`
  - Approves or denies an interactive permission request.
  - Request: `contract.ApproveSessionRequest`
  - Status: `204`, `400`, `404`
- `GET /api/sessions/:id/events`
  - Lists persisted session events.
  - Response: `contract.SessionEventRecord[]`
  - Status: `200`, `404`
- `GET /api/sessions/:id/history`
  - Lists turn-grouped history.
  - Response: `contract.TurnHistoryRecord[]`
  - Status: `200`, `404`
- `GET /api/sessions/:id/transcript`
  - Returns canonical replay transcript.
  - Response: transcript response DTOs from `api/contract`
  - Status: `200`, `404`
- `GET /api/sessions/:id/stream`
  - Streams shared SSE events.
  - Response: SSE envelopes
  - Status: `200`

Agents:

- `GET /api/agents`
  - Lists resolved agents.
  - Response: `contract.AgentRecord[]`
- `GET /api/agents/:name`
  - Returns one agent definition.
  - Response: `contract.AgentRecord`

Observe:

- `GET /api/observe/events`
  - Lists observability events.
  - Response: `contract.ObserveEventRecord[]`
- `GET /api/observe/events/stream`
  - Streams observability events.
  - Response: SSE envelopes
- `GET /api/observe/health`
  - Returns health status.
  - Response: health DTO in `api/contract`

Memory:

- `GET /api/memory`
  - Lists memory documents.
- `GET /api/memory/:filename`
  - Reads one memory document.
- `PUT /api/memory/:filename`
  - Writes one memory document.
- `DELETE /api/memory/:filename`
  - Deletes one memory document.
- `POST /api/memory/consolidate`
  - Triggers dream consolidation.
  - Request and response DTOs move to `api/contract`.

Workspaces:

- `POST /api/workspaces`
  - Registers a workspace.
- `GET /api/workspaces`
  - Lists workspaces.
- `GET /api/workspaces/:id`
  - Returns one workspace with detail payload.
- `PATCH /api/workspaces/:id`
  - Updates mutable workspace fields.
- `DELETE /api/workspaces/:id`
  - Deletes a workspace registration.
- `POST /api/workspaces/resolve`
  - Resolves or registers a workspace by path.

Daemon:

- `GET /api/daemon/status`
  - Returns daemon status and runtime counts.

## Integration Points

No new external services are introduced. Existing system boundaries that must remain stable are:

- ACP subprocess protocol
  - `acp` remains the boundary for agent subprocess communication.
  - `api/contract` must not leak ACP transport internals.
- SQLite persistence
  - `store/sessiondb` owns `events.db`.
  - `store/globaldb` owns `agh.db`.
- Filesystem metadata parsing
  - `frontmatter` becomes the single parser used by `config`, `skills`, and `memory`.
- Workspace scanning
  - `workspace` continues to resolve runtime snapshots from persisted registrations plus filesystem state.

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|---------------------|-----------------|
| `internal/api/contract` | new | Medium risk. Becomes the shared daemon contract for CLI and transports. | Create package, migrate shared DTOs from `apicore` and CLI |
| `internal/api/core` | new/merged | Medium risk. Re-roots shared handlers and absorbs `apisupport`. | Move shared transport logic and retire old package names |
| `internal/api/httpapi` | modified | Medium risk. Reduced to HTTP binding and HTTP-only behavior. | Re-root package and keep external routes stable |
| `internal/api/udsapi` | modified | Medium risk. Reduced to UDS binding and UDS-only behavior. | Re-root package and keep external routes stable |
| `internal/api/testutil` | new | Low production risk, medium maintenance impact. | Centralize duplicated API harnesses and stubs |
| `internal/frontmatter` | new | Low risk. Replaces duplicated parsing logic in three packages. | Extract parser and migrate `config`, `memory`, `skills` |
| `internal/transcript` | new | Medium risk. Changes replay ownership but not endpoint behavior. | Move transcript assembly and update session/API callers |
| `internal/store` | modified | High risk. Must shrink to shared types and narrow interfaces only. | Carve out package-local responsibilities cleanly |
| `internal/store/sessiondb` | new | High risk. Owns per-session writer loop and event queries. | Move event persistence with parity tests |
| `internal/store/globaldb` | new | High risk. Owns global registry and workspace-backed persistence. | Move registry and observability storage with parity tests |
| `internal/memory/consolidation` | new | Medium risk. Removes domain logic from `daemon`. | Move dream orchestration, lock, and scheduling logic |
| `internal/daemon` | modified | Medium risk. Must stay composition-only after the move. | Replace domain logic with wiring to new boundaries |
| `internal/cli` | modified | Medium risk. Switches from local DTOs to `api/contract`. | Migrate request/response types and test coverage |

## Testing Approach

### Unit Tests

- `api/contract`
  - Add serialization and shape-parity tests for session, workspace, memory, observe, and daemon DTOs.
- `api/core`
  - Keep shared handler tests against fake `SessionManager`, `Observer`, `WorkspaceService`, and dream trigger implementations.
- `frontmatter`
  - Move existing parser cases from `config`, `memory`, and `skills` into shared table-driven tests.
- `transcript`
  - Move replay assembly tests out of `session` and assert output parity on representative event sequences.
- `store/sessiondb` and `store/globaldb`
  - Use real SQLite databases in `t.TempDir()`.
  - Do not replace SQL logic with mocks.
- `cli`
  - Add unit coverage around the new `api/contract` DTO use and any removed local contract types.

### Integration Tests

- `make verify` is required on every step that moves packages or shared contracts.
- `make test-integration` is required for every phase that touches:
  - `api/*`
  - `cli`
  - `daemon`
  - `session`
  - `store`
  - `workspace`
- API re-root phases must rerun:
  - `internal/httpapi` integration tests
  - `internal/udsapi` integration tests
  - `internal/cli` integration tests
- Persistence split phases must rerun:
  - `internal/store`
  - `internal/session`
  - `internal/observe`
  - `internal/workspace`
  - `internal/daemon`
- A phase cannot close until:
  - any temporary bridge introduced in that phase is removed
  - `make verify` passes
  - required integration suites pass

## Development Sequencing

### Build Order

1. Extract `internal/frontmatter` and migrate `config`, `memory`, and `skills` to it.
   - No dependencies.
2. Create `internal/api/contract` and migrate shared DTOs out of `internal/apicore` and `internal/cli/client.go`.
   - Depends on step 1.
3. Re-root `apicore`, `apisupport`, `httpapi`, `udsapi`, and `apitest` into `internal/api/core`, `internal/api/httpapi`, `internal/api/udsapi`, and `internal/api/testutil`.
   - Depends on step 2.
4. Split persistence into `internal/store/sessiondb` and `internal/store/globaldb`, leaving shared types and narrow interfaces in `internal/store`.
   - Depends on step 3.
5. Extract `internal/transcript` and move replay assembly out of `session`.
   - Depends on step 4.
6. Create `internal/memory/consolidation` and move dream orchestration out of `daemon`.
   - Depends on step 4.
7. Narrow consumer interfaces in `session`, `observe`, `workspace`, `daemon`, and `cli`, then remove any phase-local bridges and dead compatibility code.
   - Depends on steps 3, 4, 5, and 6.
8. Delete validated dead files, consolidate shared test helpers, and run the final cross-phase verification pass before task generation.
   - Depends on steps 1 through 7.

### Technical Dependencies

- No new external libraries are required.
- The plan depends on preserving the repository rule that `daemon` remains the sole composition root.
- Package moves must be sequenced to avoid import cycles between API, session, workspace, observe, and persistence boundaries.
- The current repository state already includes partial groundwork:
  - shared API logic in `internal/apicore`
  - file-level splits in several packages
  - helper deduplication work
- The TechSpec assumes live code remains authoritative when analysis documents are stale.

## Monitoring and Observability

Operational expectations for the refactor are:

- Preserve existing runtime visibility for:
  - daemon status
  - observe health
  - session event streaming
  - dream consolidation logging
  - workspace resolution logging
- Preserve or improve these structured fields during moved code paths:
  - `session_id`
  - `workspace_id`
  - `turn_id`
  - `reason`
  - `workspace_ref`
  - `error_type`
- Use test and validation gates as the primary phase-promotion signal:
  - failing `make verify` blocks the step
  - failing required integration tests block the phase
  - any regression in health or SSE integration behavior blocks promotion

## Technical Considerations

### Key Decisions

- Broad package-graph reorganization
  - Chosen over path-preserving cleanup so `refac-v2` defines a real target architecture instead of another local cleanup wave.
- Dedicated `internal/api/contract`
  - Chosen so CLI depends on a canonical contract package rather than server-core handler code.
- Explicit `store/sessiondb` and `store/globaldb`
  - Chosen because the code already has two concrete storage lifecycles and the current single package hides that boundary.
- Transport-specific payloads stay local when they are not part of the shared daemon contract
  - AI SDK streaming payloads remain an `api/httpapi` concern unless another consumer appears.
- Hybrid rollout with same-phase bridge removal
  - Chosen to keep migration practical without allowing transitional compatibility code to become permanent architecture.
- Live tree over report snapshot
  - Chosen because `20260406-codex-review.md` already identified stale assumptions in the analysis set.

### Known Risks

- Analysis drift
  - Some `refac-v2` findings were written from a partially stale snapshot.
  - Mitigation: use live package paths and file ownership as the implementation authority.
- Import-cycle risk during package moves
  - The new `api/*`, `transcript`, and `store/*` boundaries can create cycles if moved in the wrong order.
  - Mitigation: follow the build order strictly and keep `daemon` as wiring only.
- Contract leakage
  - `api/contract` could become a dumping ground for transport-local payloads.
  - Mitigation: keep AI SDK and transport-only envelopes local unless shared by multiple clients.
- Shared-helper sprawl in `internal/store`
  - The package could stay too broad if helpers are not kept intentionally small.
  - Mitigation: only retain common types, interfaces, validation, and minimal SQLite helpers in `internal/store`.
- Transitional bridge leftovers
  - Short-lived forwarders can survive longer than intended.
  - Mitigation: phase exit criteria require same-phase bridge removal plus green verification.

## Architecture Decision Records

- [ADR-001: Adopt a Broad Package-Graph Reorganization for Refac V2](adrs/adr-001.md) — Defines `refac-v2` as a target-architecture refactor rather than a path-preserving cleanup.
- [ADR-002: Make internal/api/contract the Canonical Shared API Contract](adrs/adr-002.md) — Establishes one shared daemon API DTO package for CLI and transports.
- [ADR-003: Split Persistence into store/sessiondb and store/globaldb](adrs/adr-003.md) — Makes the per-session and global SQLite boundaries explicit.
- [ADR-004: Use Phased Cutovers with Same-Phase Bridge Removal and Layered Verification](adrs/adr-004.md) — Defines rollout discipline, bridge policy, and required validation gates.
