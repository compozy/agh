# TechSpec: Settings UI

## Executive Summary

The `settings-ui` effort adds a first-class settings surface to AGH across both the daemon and `web/`. Because `.compozy/tasks/settings-ui/_prd.md` does not exist, this document uses the section analyses under `.compozy/tasks/settings-ui/analysis/` as the authoritative input. The implementation introduces a consolidated `/api/settings/*` namespace backed by canonical AGH config overlays, plus a nested `/_app/settings/*` shell in the web client with one route per Paper settings screen.

The primary trade-off is choosing file-backed canonical configuration with explicit restart-aware behavior instead of building a new settings store or a broad hot-reload framework. This keeps daemon boot behavior and web edits aligned, but it also means most changes will persist immediately and only some will activate without restart. In v1, this design also makes two explicit production-grade constraints:

- TOML-backed settings writes must use comment-preserving document edits instead of blind whole-file re-encoding.
- Mutating settings over HTTP are supported only when the HTTP server is bound to a loopback host; remote HTTP management is out of scope for v1.

For daemon restart in AGH's default detached mode, the implementation uses a dedicated relaunch helper process plus a persisted restart-operation state file rather than in-process `exec` or overlapping old/new daemon lifecycles.

## Design References

All settings screens live in the `AGH` Paper file (page `Page 1`). PNG exports are committed under `docs/design/paper/settings/` and kept in sync with the Paper artboards listed below. Each section page, collection page, and the combined Hooks & Extensions surface has a 1:1 artboard.

| Screen | Local export | Paper artboard (node id) |
|--------|--------------|--------------------------|
| General | `docs/design/paper/settings/AGH Settings — General@2x.png` | `AGH Settings — General` (`VP8-0`) |
| Providers | `docs/design/paper/settings/AGH Settings — Providers@2x.png` | `AGH Settings — Providers` (`YKG-0`) |
| MCP Servers | `docs/design/paper/settings/AGH Settings — MCP Servers@2x.png` | `AGH Settings — MCP Servers` (`YRR-0`) |
| Environments | `docs/design/paper/settings/AGH Settings — Environments@2x.png` | `AGH Settings — Environments` (`YZ2-0`) |
| Memory | `docs/design/paper/settings/AGH Settings — Memory@2x.png` | `AGH Settings — Memory` (`Z6D-0`) |
| Skills | `docs/design/paper/settings/AGH Settings — Skills@2x.png` | `AGH Settings — Skills` (`ZDO-0`) |
| Automation | `docs/design/paper/settings/AGH Settings — Automation@2x.png` | `AGH Settings — Automation` (`ZKZ-0`) |
| Network | `docs/design/paper/settings/AGH Settings — Network@2x.png` | `AGH Settings — Network` (`ZSA-0`) |
| Observability | `docs/design/paper/settings/AGH Settings — Observability@2x.png` | `AGH Settings — Observability` (`ZZL-0`) |
| Hooks & Extensions | `docs/design/paper/settings/AGH Settings — Hooks & Extensions@2x.png` | `AGH Settings — Hooks & Extensions` (`106W-0`) |

Task-to-screen mapping:

| Task | Screens covered |
|------|-----------------|
| `task_01` — Comment-preserving config editors and write targets | Foundational — underpins all 10 settings screens |
| `task_02` — Settings service orchestration in `internal/settings` | Foundational — underpins all 10 settings screens |
| `task_03` — Daemon relaunch helper and restart operation store | Primary: `General` (restart action); applies to every restart-required surface |
| `task_04` — Settings API contract and OpenAPI surface | Foundational — underpins all 10 settings screens |
| `task_05` — Shared settings handlers in `api/core` | Foundational — underpins all 10 settings screens |
| `task_06` — HTTP settings transport and loopback mutation policy | Foundational — underpins all 10 settings screens |
| `task_07` — UDS settings transport and parity coverage | Foundational — underpins all 10 settings screens |
| `task_08` — Settings entrypoint and route shell | Shared shell — frames all 10 settings screens |
| `task_09` — `web/src/systems/settings` domain scaffold | Shared system — feeds all 10 settings screens |
| `task_10` — General, Memory, and Observability pages | `General`, `Memory`, `Observability` |
| `task_11` — Skills, Automation, and Network summary pages | `Skills`, `Automation`, `Network` |
| `task_12` — Providers and Environments collection pages | `Providers`, `Environments` |
| `task_13` — MCP Servers scoped collection page | `MCP Servers` |
| `task_14` — Hooks and Extensions page | `Hooks & Extensions` |

## System Architecture

### Component Overview

Main components and boundaries:

- `internal/settings`: new domain package that owns settings read models, section mutation orchestration, scope validation, write-target resolution, source precedence metadata, restart metadata, and collection CRUD for settings-backed catalogs.
- `internal/config`: remains the source of truth for schema validation and precedence rules, and is extended with a comment-preserving TOML overlay editor plus a new MCP JSON writer that preserves unrelated document content.
- `internal/api/core`, `internal/api/httpapi`, `internal/api/udsapi`: expose `/api/settings/*`, map section and collection operations to the settings service, register HTTP extension routes needed by the Hooks & Extensions screen, and enforce loopback-only policy for HTTP mutation endpoints in v1.
- `internal/daemon`: wires the settings service and adds restart orchestration, including a detached relaunch helper entrypoint and restart-operation state persisted under the AGH home directory.
- `web/src/systems/settings`: new frontend system that owns settings DTOs, adapters, query keys, query options, mutation hooks, and section-level UI helpers.
- `web/src/routes/_app/settings.tsx` plus child routes under `web/src/routes/_app/settings/`: new settings shell with section navigation and route-level orchestration.
- Existing operational pages (`/automation`, `/network`, `/skills`) remain separate destinations. Settings sections summarize configuration and link to those pages instead of embedding their operational panels.

High-level flow:

```text
web settings route
    -> /api/settings/<section>
    -> internal/settings service
    -> aghconfig overlay loader + workspace resolver + runtime summaries
    -> section read model

web save action
    -> /api/settings/<section> mutation
    -> internal/settings validates merged config
    -> writes canonical target file
    -> computes applied-now vs restart-required
    -> returns structured mutation result
```

## Implementation Design

### Core Interfaces

Primary daemon-facing service contract:

```go
type Service interface {
    GetSection(ctx context.Context, req SectionRequest) (SectionEnvelope, error)
    UpdateSection(ctx context.Context, req SectionUpdateRequest) (MutationResult, error)
    ListCollection(ctx context.Context, req CollectionRequest) (CollectionEnvelope, error)
    PutCollectionItem(ctx context.Context, req CollectionItemPutRequest) (MutationResult, error)
    DeleteCollectionItem(ctx context.Context, req CollectionItemDeleteRequest) (MutationResult, error)
}
```

Shared mutation result shape:

```go
type WriteTargetKind string

type MutationResult struct {
    Section         SectionName `json:"section"`
    Scope           ScopeKind   `json:"scope"`
    WriteTarget     WriteTargetKind `json:"write_target,omitempty"`
    WorkspaceID     string      `json:"workspace_id,omitempty"`
    Applied         bool        `json:"applied"`
    RestartRequired bool        `json:"restart_required"`
    RestartScope    string      `json:"restart_scope,omitempty"`
    Warnings        []string    `json:"warnings,omitempty"`
}
```

Restart-action response shapes:

```go
type RestartActionResponse struct {
    OperationID        string `json:"operation_id"`
    Status             string `json:"status"`
    StatusURL          string `json:"status_url"`
    ActiveSessionCount int    `json:"active_session_count"`
}

type RestartActionStatus struct {
    OperationID        string     `json:"operation_id"`
    Status             string     `json:"status"`
    OldPID             int        `json:"old_pid"`
    NewPID             int        `json:"new_pid,omitempty"`
    ActiveSessionCount int        `json:"active_session_count"`
    FailureReason      string     `json:"failure_reason,omitempty"`
    StartedAt          time.Time  `json:"started_at"`
    UpdatedAt          time.Time  `json:"updated_at"`
    CompletedAt        *time.Time `json:"completed_at,omitempty"`
}
```

Error handling conventions:

- Validation failures return `400` with section- and field-specific context.
- HTTP mutation attempts from non-loopback HTTP bindings return `403`.
- Missing workspaces, providers, environments, MCP servers, hooks, or extensions return `404`.
- Conflicting names, invalid scope combinations, invalid write-target requests, or duplicate resource definitions return `409`.
- File write failures, restart orchestration failures, and transport parity errors return `500`.
- All daemon-side errors must preserve wrapped context with section, scope, and write target.

### Data Models

Section envelopes:

- `general`
  - daemon runtime status
  - resolved config paths
  - defaults
  - limits
  - permissions
  - session timeout
  - restart action metadata
- `memory`
  - memory config
  - dream config
  - memory health
  - consolidate action metadata
- `skills`
  - engine config
  - disabled skills
  - marketplace config
  - discovered and disabled counts
  - operational links
- `automation`
  - engine config
  - manager summary
  - operational links
- `network`
  - network config
  - runtime status summary
  - operational links
- `observability`
  - observe config
  - transcript config
  - DB usage metrics
  - log tail capability metadata
- `hooks-extensions`
  - hook declarations
  - extension marketplace config
  - extension resource policy
  - installed extension summaries
  - transport parity status

Collection resources:

- `providers`
  - config-backed provider catalog with resolved command, default model, API key env name, command availability, environment presence state, builtin-versus-overlay source metadata, and fallback metadata for builtin providers
- `mcp-servers`
  - scoped collection with `scope`, optional `workspace_id`, effective precedence summary, `effective_source`, `shadowed_sources`, `available_targets`, and target file metadata
- `environments`
  - sandbox profiles with computed workspace usage counts
- `hooks`
  - hook declarations stored in canonical config overlays

Scope rules:

- Global scope is the default for all settings sections.
- Workspace scope is explicitly supported in v1 for `mcp-servers` because the product surface needs it there first. The underlying config loader already supports broader global and workspace overlay loading, but v1 intentionally limits workspace editing to sections with clear UX and write-target semantics.
- Additional workspace-scoped section editing can be added later when the product surface requires it and the write target is unambiguous.

Persistence targets:

- Global config writes go to `HomePaths.ConfigFile`.
- Workspace-scoped config writes go to `<workspace-root>/.agh/config.toml`.
- MCP sidecar writes go to `~/.agh/mcp.json` or `<workspace-root>/.agh/mcp.json` as appropriate.
- TOML-backed writes must use comment-preserving document mutation. If a requested edit cannot be applied without rewriting unrelated TOML structure, the daemon must reject that mutation as unsupported instead of silently canonicalizing the file.
- MCP sidecar writes use a new JSON writer that preserves unknown top-level keys and untouched server definitions.
- No new daemon-owned settings store is introduced.

Collection mutation semantics:

- `PUT /providers/:name`, `PUT /sandboxes/:name`, and `PUT /hooks/:name` are full replacements of the named overlay entry in the selected canonical target.
- `DELETE /providers/:name`, `DELETE /sandboxes/:name`, and `DELETE /hooks/:name` remove only the selected overlay entry. When a builtin provider exists with the same name, deleting the overlay reveals the builtin definition again.
- `mcp-servers` accepts an explicit `target=auto|config|sidecar` selector. `target=auto` edits the highest-precedence source in the selected scope when the server already exists, and writes new servers to `mcp.json` by default.
- `DELETE /mcp-servers/:name?target=auto` removes the highest-precedence definition in the selected scope only. Lower-precedence definitions may become effective again and must be reported as `shadowed_sources` in the next read model.

Runtime apply matrix:

If a field does not have an explicit live-apply surface in this matrix, it defaults to `restart_required` in v1.

| Section | Field or action | Behavior | Notes |
|---------|-----------------|----------|-------|
| `general` | `defaults.agent`, `defaults.provider`, `defaults.sandbox` | `restart_required` | default resolution remains tied to the daemon's loaded config snapshot in v1 |
| `general` | `limits.max_sessions`, `limits.max_concurrent_agents` | `restart_required` | daemon-wide admission controls remain startup-configured |
| `general` | `session.limits.timeout`, `permissions.mode` | `restart_required` | applies to future daemon activity after reload |
| `general` | `http.host`, `http.port`, `daemon.socket` | `restart_required` | listener and socket rebinding require controlled restart |
| `general` | restart action | `action_trigger` | runs the relaunch helper flow |
| `providers` | `providers.*` | `restart_required` | provider registry is reloaded from canonical config on daemon boot |
| `mcp-servers` | `mcp-servers.*` | `restart_required` | effective MCP catalog is rebuilt from config and sidecars on reload |
| `environments` | `environments.*` | `restart_required` | environment catalog is reloaded from config on daemon boot |
| `memory` | `memory.*`, `memory.dream.*` | `restart_required` | store paths and dream thresholds are startup-configured |
| `memory` | consolidate action | `action_trigger` | reuses `/api/memory/consolidate` |
| `skills` | `skills.disabled` | `applied_now` | persist config and update the registry for future task and session resolution; existing sessions are unchanged |
| `skills` | other `skills.*` marketplace and policy fields | `restart_required` | no safe live policy-reload surface in v1 |
| `automation` | `automation.*` | `restart_required` | manager wiring and schedules are startup-configured |
| `network` | `network.*` | `restart_required` | listener topology and peer runtime require restart |
| `observability` | `observability.*` | `restart_required` | transcript and observe policy are reloaded on boot |
| `hooks-extensions` | `hooks.*` | `restart_required` | hook declaration graph is rebuilt on reload |
| `hooks-extensions` | extension install, enable, disable | `action_trigger` | reuses extension service operations and applies immediately |
| `hooks-extensions` | extension marketplace and resource policy | `restart_required` | config-backed policy has no live reload in v1 |

### API Endpoints

Section reads and updates:

- `GET /api/settings/general`
- `PATCH /api/settings/general`
- `GET /api/settings/memory`
- `PATCH /api/settings/memory`
- `GET /api/settings/skills`
- `PATCH /api/settings/skills`
- `GET /api/settings/automation`
- `PATCH /api/settings/automation`
- `GET /api/settings/network`
- `PATCH /api/settings/network`
- `GET /api/settings/observability`
- `PATCH /api/settings/observability`
- `GET /api/settings/hooks-extensions`
- `PATCH /api/settings/hooks-extensions`

Collection resources:

- `GET /api/settings/providers`
- `GET /api/settings/providers/:name`
- `PUT /api/settings/providers/:name`
- `DELETE /api/settings/providers/:name`
- `GET /api/settings/mcp-servers?scope=global|workspace&workspace_id=...`
- `PUT /api/settings/mcp-servers/:name?scope=global|workspace&workspace_id=...&target=auto|config|sidecar`
- `DELETE /api/settings/mcp-servers/:name?scope=global|workspace&workspace_id=...&target=auto|config|sidecar`
- `GET /api/settings/sandboxes`
- `GET /api/settings/sandboxes/:name`
- `PUT /api/settings/sandboxes/:name`
- `DELETE /api/settings/sandboxes/:name`
- `GET /api/settings/hooks`
- `PUT /api/settings/hooks/:name`
- `DELETE /api/settings/hooks/:name`

Settings actions:

- `POST /api/settings/actions/restart`
  - available on UDS and on HTTP only when the HTTP server is bound to a loopback host
  - spawns the internal helper subcommand `agh daemon relaunch`
  - returns `202 Accepted` with `operation_id`, `status`, `status_url`, and `active_session_count`
  - writes restart state to `HomePaths.HomeDir/restarts/<operation_id>.json`
  - current daemon records `stopping`, performs graceful shutdown, and releases lock plus `daemon.json`
  - helper waits for singleton resource release, launches the replacement daemon with the same executable, inherited environment, and detached process-group semantics, then updates operation state through `waiting_release`, `starting`, and terminal success or failure
  - replacement daemon marks the operation `ready` only after boot succeeds and fresh daemon discovery state is written
- `GET /api/settings/actions/restart/:operation_id`
  - returns current restart state from the persisted operation record
  - survives loss of the initiating connection and replacement-daemon boot failures
- `POST /api/memory/consolidate`
  - reused directly for the Memory screen "Trigger now" action
- `GET /api/settings/observability/log-tail`
  - streams daemon log output from `HomePaths.LogFile`
  - uses SSE in v1
  - closes the stream on log rotation or reader error; the client reconnects explicitly

Supporting operational HTTP parity required by the settings surface:

- `GET /api/extensions`
- `POST /api/extensions`
- `GET /api/extensions/:name`
- `POST /api/extensions/:name/enable`
- `POST /api/extensions/:name/disable`

Transport and security policy:

- Read-only settings routes are available on both HTTP and UDS.
- Settings mutations, restart actions, and HTTP extension mutations are exposed on HTTP only when the HTTP server is bound to a loopback host.
- When HTTP is bound to a non-loopback host, these HTTP mutation routes return `403` and UDS remains the authoritative privileged transport.
- V1 does not introduce authenticated remote HTTP management for settings. That requires a separate security design.

Response behavior:

- Every mutation returns `MutationResult`.
- `MutationResult` reports semantic `write_target` instead of absolute filesystem paths.
- Every section read returns both `config` and `runtime` data where the Paper screen mixes static config and live state.
- Collection reads that compose multiple sources must return source metadata needed by the UI, especially `effective_source`, `shadowed_sources`, and `available_targets`.
- HTTP and UDS must expose the same `/api/settings/*` contract. Existing `/api/extensions` routes must also be registered on HTTP for the Hooks & Extensions screen, subject to the loopback-only mutation policy above.

## Integration Points

This design adds no new external service dependencies. Its integration points are internal system boundaries plus the local host:

- Host filesystem
  - read and write `~/.agh/config.toml`
  - read and write optional workspace `.agh/config.toml`
  - read and write MCP JSON sidecars
  - read and write restart-operation state files under `~/.agh/restarts/`
  - read daemon log file for log-tail UI
- Host process lifecycle
  - restart helper launches a replacement AGH process using the same executable and detached process-group semantics
- Workspace resolver
  - resolves `workspace_id` to canonical workspace roots before any scoped write
- Existing runtime services
  - memory consolidate
  - skill registry state
  - observe health
  - network status
  - automation status
  - extension service status

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|---------------------|-----------------|
| `internal/config` | modified | Adds comment-preserving TOML overlay editor and a new MCP sidecar writer; medium-to-high risk because persistence becomes operator-facing | Replace blind re-encoding with structured document edits and validate merged config before commit |
| `internal/settings` | new | New cross-cutting service for settings orchestration; medium risk because it spans multiple domains | Keep one package with file-level splits and shared section abstractions |
| `internal/api/contract` | modified | Adds settings DTOs and mutation result payloads; low-to-medium risk | Keep OpenAPI and generated web types aligned |
| `internal/api/spec` | modified | Adds settings OpenAPI surface; low risk | Regenerate artifacts after every contract change |
| `internal/api/core` | modified | Adds settings handlers and action endpoints; medium risk | Centralize mapping to the settings service |
| `internal/api/httpapi` | modified | Adds `/api/settings/*` routes, HTTP extension parity, and loopback-only mutation enforcement; medium risk | Keep route inventory aligned with UDS and apply transport policy consistently |
| `internal/api/udsapi` | modified | Adds `/api/settings/*` routes for CLI and transport parity; low risk | Mirror HTTP semantics and payloads |
| `internal/daemon` | modified | Adds restart helper orchestration, persisted operation state, and settings service wiring; medium risk | Reuse detached spawn logic safely and persist restart outcomes durably |
| `web/src/systems/settings` | new | Adds frontend settings system; low risk if isolated | Follow existing `app-renderer-systems` conventions |
| `web/src/routes/_app/settings*` | new | Adds shared settings shell and section routes; low risk | Keep route-level orchestration and pure UI components |
| `web/src/components/app-sidebar.tsx` | modified | Existing Settings button becomes navigational | Link to `/settings` and preserve current visual style |

## Testing Approach

### Unit Tests

- `internal/config`
  - comment-preserving overlay edits for unrelated comments and untouched sections
  - MCP sidecar writes
  - MCP writer preservation of unknown top-level keys
  - merged validation failures
  - write-target resolution
- `internal/settings`
  - section read-model assembly
  - mutation diffing
  - scope validation
  - restart metadata
  - runtime-apply matrix enforcement
  - collection CRUD target selection
- `internal/api/core`
  - handler status mapping
  - invalid payloads
  - unsupported scope errors
  - restart action endpoint behavior
- `internal/api/httpapi`
  - loopback-only mutation enforcement
  - HTTP extension route registration
- `web/src/systems/settings`
  - adapter decoding
  - query options
  - mutation hooks
  - form serialization
  - restart banner state
  - source-metadata handling for multi-source resources
- `web/src/components/app-sidebar.test.tsx`
  - Settings navigation activation

### Integration Tests

- Global settings mutation persists to `HomePaths.ConfigFile` and reloads as effective daemon config.
- Global settings mutation preserves unrelated TOML comments and untouched sections.
- Workspace-scoped MCP server mutation persists to the correct workspace overlay or sidecar target.
- MCP server auto-target selection edits the highest-precedence source and reports shadowed lower-precedence definitions.
- Provider overlay delete reveals builtin provider fallback when applicable.
- Extension-related settings data and extension operations are visible through HTTP with the same effective shape expected by the web client.
- HTTP mutation routes return `403` when the daemon HTTP server is bound to a non-loopback host.
- Restart helper flow:
  - helper spawn failure
  - restart operation file written before shutdown
  - daemon shutdown and lock release
  - replacement daemon ready
  - replacement daemon boot failure
  - restart status polling after reconnect
- Web route coverage:
  - `/settings` shell load
  - section navigation
  - save flow
  - restart-required banner
  - restart-operation polling
  - operational links

Verification gates:

- `make verify`
- `make web-lint`
- `make web-typecheck`
- `make web-test`

## Development Sequencing

### Build Order

1. Extend `internal/config` with a comment-preserving TOML editor, a new MCP sidecar writer, explicit write-target handling, and merged-config validation on write. This step has no dependencies.
2. Add `internal/settings` with section mapping, scope resolution, source precedence metadata, collection CRUD orchestration, and the v1 runtime-apply matrix. This step depends on step 1.
3. Add settings DTOs, OpenAPI definitions, HTTP extension parity, and loopback-only mutation wiring in `internal/api/contract`, `internal/api/spec`, `internal/api/core`, `internal/api/httpapi`, and `internal/api/udsapi`. This step depends on step 2.
4. Add daemon integration for the relaunch helper, persisted restart-operation state, and any safe runtime-apply hooks. This step depends on steps 2 and 3.
5. Add `web/src/systems/settings`, the shared `/_app/settings/*` shell, sidebar navigation, and restart-operation polling. This step depends on step 3.
6. Implement each section page and connect operational links to existing pages. This step depends on step 5 and on the section endpoints from step 3.
7. Add regression tests and run backend and web verification gates. This step depends on steps 1 through 6.

### Technical Dependencies

- The bootstrap-only overlay writer in `internal/config/bootstrap.go` must be replaced for settings mutations with a comment-preserving document editor.
- MCP sidecar persistence must be implemented as a new writer instead of assuming existing write support.
- Extension routes currently visible only through UDS must be registered on HTTP because the Hooks & Extensions screen depends on them.
- Detached daemon launch logic must be factored into shared relaunch code instead of staying CLI-only.
- Restart outcomes must be persisted outside the daemon process so the UI can reconcile after reconnect.
- Log tail streaming must safely read the structured daemon log file without blocking daemon shutdown.

## Monitoring and Observability

Operational visibility for this implementation:

- Metrics
  - `settings_reads_total`
  - `settings_mutations_total`
  - `settings_mutation_failures_total`
  - `settings_restart_required_total`
  - `settings_restart_attempts_total`
  - `settings_restart_failures_total`
- Structured log fields
  - `section`
  - `scope`
  - `workspace_id`
  - `write_target`
  - `applied`
  - `restart_required`
  - `restart_operation_id`
  - `restart_status`
  - `warning_count`
- UI observability
  - restart status banner
  - mutation warnings
  - last action result for manual triggers

## Technical Considerations

### Key Decisions

- Use one consolidated settings namespace instead of scattering settings across operational APIs.
- Use one dedicated `web/src/systems/settings` domain and one nested settings shell under `/_app/settings/*`.
- Persist through canonical config overlays and MCP sidecars instead of a new settings database.
- Use comment-preserving TOML edits for settings mutations instead of blind whole-file canonicalization.
- Keep settings restart-aware. Apply live only where the daemon already exposes a safe mutation surface.
- Keep operational pages separate and linked from settings instead of embedding them.
- Restrict HTTP settings and extension mutations to loopback-bound servers in v1 instead of introducing remote management without an auth design.
- Do not use `syscall.Exec` for daemon restart.
- Do not spawn the replacement daemon before the current daemon has released lock, socket, and daemon info resources.
- Reuse and factor detached launch logic from `internal/cli/daemon.go` into shared relaunch code.
- Treat restart as an asynchronous operation with explicit status, not as a synchronous HTTP round-trip.
- Persist restart-operation state on disk and expose it through a polling endpoint.
- Use explicit write-target and source-precedence metadata for multi-source resources instead of leaking filesystem paths to the client.
- If AGH is later installed under `launchd` or `systemd`, restart should be delegated to the service manager instead of using the helper path.

### Known Risks

- Workspace scope can become ambiguous if the UI exposes it too broadly beyond sections that need it.
- Most subsystem settings are boot-time today, so the first iteration will likely mark many mutations as `restart_required`.
- Comment-preserving TOML mutation is more complex than raw re-encoding and needs strong tests around unrelated document preservation.
- MCP dual-source behavior can still confuse operators even with explicit `effective_source` and `shadowed_sources` metadata.
- Helper spawn may fail after settings are already persisted.
- Replacement daemon boot may fail after the current daemon exits.
- Restart-status diagnostics must survive the initiating connection drop so the UI can explain failure on reconnect.

## Architecture Decision Records

- [ADR-001: Use a consolidated settings namespace with a dedicated settings shell](adrs/adr-001.md) — Centralizes settings under `/api/settings/*` and `/_app/settings/*` while keeping operational pages separate.
- [ADR-002: Persist settings by writing canonical config overlays instead of creating a new settings store](adrs/adr-002.md) — Keeps file-backed config as the only source of truth while requiring comment-preserving TOML edits and explicit MCP write-target semantics.
- [ADR-003: Keep settings mutations restart-aware and separate from operational workflows](adrs/adr-003.md) — Applies safe live changes only where runtime surfaces already exist and uses a helper-based relaunch flow for restart-required changes.
- [ADR-004: Restrict HTTP settings mutations to loopback-bound servers in v1](adrs/adr-004.md) — Avoids remote config writes and daemon restart over unauthenticated HTTP while keeping the local web app fully functional.
