# TechSpec: Per-Session ACP Provider Override

## Executive Summary

This TechSpec defines how AGH lets one `agent_name` keep its default ACP runtime while allowing each individual session to choose a different provider for that conversation. No `_prd.md` exists for this feature, so this document is based on the user request, codebase exploration, and the accepted ADRs in `adrs/`.

The implementation extends the existing provider-resolution path instead of introducing a second driver registry. The chosen provider becomes part of session identity: AGH resolves the effective runtime from `agent + optional provider override`, persists the resulting provider in on-disk session metadata and the global session index, and reuses that persisted provider on resume. The primary trade-off is deliberate: AGH favors deterministic runtime identity and explicit operator-visible failure over silent fallback convenience. That requires schema migration, one-time repair of legacy metadata, and a more explicit web creation flow.

## System Architecture

### Component Overview

- `internal/config`
  Purpose: resolve `AgentDef` plus workspace-visible provider config into one `ResolvedAgent`.
  Boundary: owns provider lookup, provider default command/model resolution, and provider/global/agent MCP layer merging.

- `internal/session`
  Purpose: create, start, persist, query, and resume sessions.
  Boundary: accepts optional provider override in `CreateOpts`, validates it against the resolved workspace config during startup, persists the effective provider, and reuses it on resume.

- `internal/store` and `internal/store/globaldb`
  Purpose: persist session metadata on disk and maintain the global `sessions` read model in SQLite.
  Boundary: add `provider` to `SessionMeta`, `SessionInfo`, and the SQLite `sessions` table; migrate existing local DBs in place; reconcile repaired metadata into the global index.

- `internal/api/contract`, `internal/api/core`, `internal/api/spec`, `internal/cli`, `internal/extension`
  Purpose: expose explicit session-creation surfaces and session read models.
  Boundary: add optional `provider` to create-session requests, expose effective `provider` in session payloads, extend workspace detail with provider options, update OpenAPI/codegen, and surface provider in CLI output.

- `web/src/systems/session` and workspace-facing web surfaces
  Purpose: provide the explicit create-session dialog and display the effective provider in session UI.
  Boundary: replace direct quick-create with a single dialog-driven flow, prefill the selected agent/workspace/default provider, and render dedicated inline errors when resume fails because the persisted provider is unavailable.

### Data Flow

Session create flow:

1. A caller selects `agent_name` and may provide `provider`.
2. AGH resolves the workspace first, then resolves the selected agent inside that workspace.
3. AGH resolves the effective runtime from `agent + provider override` using workspace-merged provider config.
4. Provider validation completes inside `prepareSessionStartRuntime`, before `writeMeta(session)` and before `driver.Start`.
5. AGH writes session metadata and the global session row with the effective provider.
6. Transport startup uses the resolved runtime command/model/MCP layers and returns a session payload that includes `agent_name` and `provider`.

Session resume flow:

1. AGH reads on-disk session metadata.
2. If legacy inactive metadata has `provider == ""`, AGH repairs it once by resolving the provider from the stored agent plus resolved workspace config and persisting the result.
3. AGH validates the persisted provider against the current workspace-merged config.
4. If validation succeeds, AGH resumes with that exact provider.
5. If validation fails, AGH returns an explicit error and does not fall back to the current agent default.

## Implementation Design

### Core Interfaces

The feature keeps the existing package layout and extends current types instead of adding new packages.

```go
type CreateOpts struct {
	AgentName     string
	Provider      string
	Name          string
	Workspace     string
	WorkspacePath string
	Channel       string
	Type          Type
}
```

```go
func (c *Config) ResolveSessionAgent(
	agent AgentDef,
	providerOverride string,
) (ResolvedAgent, error)
```

```go
type SessionProviderOptionPayload struct {
	Name string `json:"name"`
}
```

Resolution helper semantics:

- `ResolveSessionAgent` is the only new resolution entry point.
- When `providerOverride == ""`, it behaves like current `ResolveAgent`.
- When `providerOverride != ""`, it clones the input `AgentDef`, overwrites `Provider`, clears explicit `Command` and `Model`, then reuses the shared provider-resolution path.
- `ResolvedAgent.Provider` is the source-of-truth field that flows into `session.Session`, `session.Info`, `store.SessionMeta`, `store.SessionInfo`, and transport payloads.
- Provider validation must run against `spec.workspace.Config`, not daemon-global config.

### Data Models

Runtime and persistence model changes:

- `session.CreateOpts`
  Add `Provider string`.

- `session.Session`
  Add `Provider string` as the in-memory runtime field persisted through `Meta()`.

- `session.Info`
  Add `Provider string` so list/get/status/read-model consumers can observe the effective runtime.

- `store.SessionMeta`
  Add `Provider string \`json:"provider,omitempty"\``.

- `store.SessionInfo`
  Add `Provider string`.

- `contract.CreateSessionRequest`
  Add `provider`.

- `contract.SessionPayload`
  Add `provider`.

- `contract.WorkspaceDetailPayload`
  Add `providers []SessionProviderOptionPayload`.

Global DB schema:

- Add `provider TEXT NOT NULL DEFAULT ''` to the SQLite `sessions` table.
- Extend the in-place session migration path in `internal/store/globaldb/migrate_workspace.go` so existing DBs receive the new column via `ALTER TABLE ... ADD COLUMN`.
- Extend copy-style migrations that rebuild `sessions` to create and populate the new column as well.

Legacy metadata repair:

- Blank provider is allowed only as a transient pre-feature state on inactive session metadata.
- The first read of such metadata must resolve and persist the effective provider before resume or global reconcile proceeds.
- If repair cannot resolve the stored agent or provider anymore, AGH fails explicitly.

Explicit non-goals for this feature:

- No new driver registry or separate runtime catalog.
- No per-session model override.
- No schema change to `SessionEventPayload` or `AgentEventPayload`; per-event provider duplication stays out of scope for this feature.

### API Endpoints

HTTP and UDS share the same contract changes.

| Method | Path | Change |
| --- | --- | --- |
| `POST` | `/api/sessions` | Add optional `provider` to the request body. Create the session with that provider when supplied. |
| `GET` | `/api/sessions` | Return `provider` in each `SessionPayload`. |
| `GET` | `/api/sessions/{id}` | Return `provider` in the `SessionPayload`. |
| `POST` | `/api/sessions/{id}/resume` | Resume with the persisted provider. Return an explicit error if that provider is unavailable. |
| `GET` | `/api/workspaces/{id}` | Return `providers` in `WorkspaceDetailPayload` so the web dialog can render a picker without a separate endpoint. |

Create-session request shape:

```json
{
  "agent_name": "coder",
  "provider": "codex",
  "workspace": "ws_123",
  "channel": "team-red"
}
```

Session response shape:

```json
{
  "session": {
    "id": "sess_123",
    "agent_name": "coder",
    "provider": "codex",
    "state": "starting"
  }
}
```

Extension and CLI surfaces:

- `agh session new` adds `--provider`.
- CLI session list/detail output surfaces the effective provider by default.
- Extension Host API `sessions.create` accepts an optional `provider` field because it is an explicit session-creation surface.

Non-interactive internal creators:

- `internal/automation/dispatch.go`
- `internal/daemon/task_runtime.go`
- `internal/memory/consolidation/runtime.go`
- `internal/api/core/network_details.go`
- `internal/extension/host_api_bridges.go`

These paths continue to pass an empty provider in this first cut, which means they intentionally use the agent default runtime.

## Integration Points

No new external service integration is introduced by this feature.

Existing ACP subprocess launch remains unchanged:

- `internal/session.Manager` still owns one injected `AgentDriver`.
- The selected provider only changes the resolved runtime command/model/MCP layer passed into that existing ACP transport adapter.

Generated artifact boundary:

- `openapi/agh.json` is regenerated from `internal/api/spec/spec.go`.
- `web/src/generated/agh-openapi.d.ts` is regenerated from `openapi/agh.json`.

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|---------------------|-----------------|
| `internal/config` | modified | Medium risk. Add session-aware resolution semantics and make `ResolvedAgent.Provider` the single source of truth. | Implement `ResolveSessionAgent` and MCP-layer tests. |
| `internal/session` | modified | High risk. Create/start/resume/query flows now own provider override, validation ordering, and legacy repair. | Add `Provider` to runtime/read-model types and validate before persistence. |
| `internal/store/types.go` | modified | Medium risk. On-disk and global read models gain a new field. | Extend `SessionMeta` and `SessionInfo`. |
| `internal/store/globaldb` | modified | High risk. SQLite `sessions` schema must migrate safely in place and reconcile repaired metadata correctly. | Add migration, register/scan support, and migration tests. |
| `internal/api/contract` and `internal/api/core` | modified | Medium risk. Session and workspace contracts change. | Add `provider` to session payloads and provider options to workspace detail. |
| `internal/api/spec` and `cmd/agh-codegen` outputs | modified | Medium risk. Checked-in OpenAPI and TS types must stay in sync. | Regenerate `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts`. |
| `internal/cli/session.go` | modified | Low risk. CLI create/list/detail surfaces provider. | Add `--provider` and display provider in output. |
| `internal/extension/host_api.go` | modified | Low risk. Explicit Host API create path must accept provider. | Extend RPC params and tests. |
| Internal non-interactive session creators | modified | Low risk. They must continue to be explicit about using the agent default. | Pass empty provider and document this contract in code/tests. |
| `web/src/systems/session` and workspace-facing UI | modified | Medium risk. Quick-create becomes a dialog flow; resume error UI becomes explicit. | Build the dialog, wire all create entrypoints through it, and show inline resume failure state. |

## Testing Approach

### Unit Tests

- `internal/config`
  Test `ResolveSessionAgent` for:
  - no override keeps current behavior
  - override swaps provider, command, and default model
  - override clears explicit agent command/model influence
  - provider-owned MCP layer is replaced while global and agent layers remain

- `internal/session`
  Test:
  - `CreateOpts.Provider` propagation
  - `Session.Meta()` and `sessionInfoFromMeta()` round-trip provider
  - validation runs before `writeMeta`
  - legacy blank-provider repair persists once
  - explicit error when persisted provider is unavailable

- `internal/store/globaldb`
  Test:
  - `migrateSessionColumns` adds `provider` idempotently
  - copy-style migrations preserve `provider`
  - `scanSessionInfo` reads provider
  - `registerSession` upserts provider
  - reconcile does not keep blank providers after repair

- `internal/api/core` and `internal/api/contract`
  Test:
  - request decoding accepts optional provider
  - session payload conversion emits provider
  - workspace detail payload emits sorted provider options

- `web`
  Test:
  - dialog opens from every create entrypoint
  - default provider is preselected from the chosen agent
  - provider picker renders workspace-visible providers
  - resume failure renders dedicated inline state instead of only a toast

### Integration Tests

- HTTP/UDS integration
  - create session with explicit provider and verify returned payload/provider
  - resume session after agent default changes and verify persisted provider wins
  - resume fails explicitly when persisted provider is removed

- Session manager integration
  - starting session with invalid provider fails before metadata/global DB write
  - legacy metadata with blank provider repairs once and resumes deterministically

- Global DB integration
  - opening an existing DB migrates `sessions.provider`
  - reconcile persists repaired provider into the global index

- Web/API integration
  - workspace detail returns provider options and the web dialog consumes them through generated types

- Verification commands
  - `make verify`
  - `make codegen-check`
  - targeted web tests and typecheck as needed while implementing

## Development Sequencing

### Build Order

1. Extend core data models and contracts.
   Add `provider` to session create/read models, add workspace provider options payloads, and update OpenAPI schema definitions. This step has no dependencies.

2. Add session-aware config resolution.
   Implement `ResolveSessionAgent` and provider-option assembly from workspace-merged config. This step depends on step 1.

3. Add session lifecycle semantics.
   Thread `CreateOpts.Provider` through create/start/resume/query paths, perform validation inside `prepareSessionStartRuntime`, and persist provider through session metadata. This step depends on steps 1 and 2.

4. Add migration and legacy repair.
   Extend SQLite session migration, update register/scan/reconcile behavior, and add one-time blank-provider repair for inactive metadata. This step depends on steps 1 and 3.

5. Update explicit and automatic creators.
   Extend HTTP/UDS/CLI/Host API explicit create surfaces to accept `provider`, and keep automatic internal creators pinned to agent default by passing empty provider. This step depends on steps 1 through 4.

6. Update web flows.
   Replace direct quick-create with the dialog flow, consume workspace provider options, and add inline resume failure state. This step depends on steps 1, 2, and 5.

7. Regenerate artifacts and close verification.
   Run codegen, update generated files, add/adjust tests, and pass the full verification gate. This step depends on steps 1 through 6.

### Technical Dependencies

- `internal/workspace` must continue to provide workspace-merged config before provider validation runs.
- `cmd/agh-codegen` must regenerate `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts`.
- No new package or directory is required; implement inside current packages.

## Monitoring and Observability

- Structured logs on create, resume, and legacy repair failures should include:
  - `session_id`
  - `agent_name`
  - `provider`
  - `workspace_id`
  - `phase` (`create`, `resume`, `legacy_repair`)
  - `error`

- Structured logs on successful legacy repair should include:
  - `session_id`
  - `agent_name`
  - `provider`
  - `workspace_id`
  - `repaired=true`

- API error messages for unavailable persisted providers should name:
  - the session id
  - the missing provider

- No new alerting threshold is required for this feature. Operator observability comes from session payloads, explicit errors, and structured logs.

## Technical Considerations

### Key Decisions

- Decision: model per-session engine choice as `provider` override, not agent swap or a new driver registry.
  Rationale: existing provider resolution already expresses runtime choice.
  Trade-off: the UI and API must now distinguish agent identity from runtime identity.
  Alternatives rejected: separate driver catalog; treating runtime choice as another agent.

- Decision: make `ResolvedAgent.Provider` the source-of-truth runtime field.
  Rationale: prevents ad-hoc recomputation from `agent.Provider` and keeps persistence coherent.
  Trade-off: requires a dedicated helper and touches several read models.
  Alternatives rejected: scattered field mutation; preserving explicit agent command/model on override.

- Decision: use one-time legacy repair for blank provider metadata.
  Rationale: avoids breaking all local sessions while still converging to deterministic persistence.
  Trade-off: first repair depends once on current workspace and agent resolution.
  Alternatives rejected: perpetual fallback; immediate hard failure for every legacy session; dropping local session indexes.

- Decision: expose workspace-visible provider options through `WorkspaceDetailPayload`, not a new `/api/providers` endpoint.
  Rationale: keeps provider discovery scoped to the resolved workspace that already drives the create flow.
  Trade-off: workspace detail grows one more field.
  Alternatives rejected: separate provider-list endpoint; client-side inference from agent defaults only.

- Decision: keep raw per-event payloads unchanged in this feature.
  Rationale: session payloads already surface provider, and per-event duplication would widen scope without changing core behavior.
  Trade-off: operator tools that need provider per event still derive it from session context.
  Alternatives rejected: adding provider to every session event and raw agent-event payload now.

### Known Risks

- Risk: a legacy session may repair to a provider the operator no longer expects.
  Likelihood: medium.
  Mitigation: repair once, persist immediately, and log the repaired provider explicitly.

- Risk: provider override swaps provider-level MCP servers and may change runtime tooling in surprising ways.
  Likelihood: medium.
  Mitigation: add integration tests that assert provider MCP layer replacement while global and agent layers remain intact.

- Risk: moving from direct quick-create to dialog-based creation may feel like a regression.
  Likelihood: medium.
  Mitigation: prefill agent/workspace/default provider, route every create entrypoint through the same dialog, and keep submission one-click once open.

- Risk: generated artifacts may drift from contract changes.
  Likelihood: high during implementation.
  Mitigation: include `make codegen-check` in the verification gate and commit regenerated files.

- Risk: workspace-specific provider overrides could validate differently across workspaces.
  Likelihood: medium.
  Mitigation: validate only against `spec.workspace.Config` after workspace resolution and before session persistence.

## Architecture Decision Records

- [ADR-001: Model Session Driver Selection As A Provider Override](adrs/adr-001.md) — Represent per-session engine choice as a provider override layered on top of existing agent resolution.
- [ADR-002: Re-Resolve Provider-Owned Runtime Fields On Session Override](adrs/adr-002.md) — When a session changes provider, provider-owned runtime fields come from the selected provider and not from the agent definition.
- [ADR-003: Persist Effective Session Provider And Fail Explicitly On Mismatch](adrs/adr-003.md) — Persist the effective provider, reuse it on resume, and fail explicitly when it becomes unavailable.
- [ADR-004: Use Explicit Session Creation Surfaces For Provider Selection](adrs/adr-004.md) — Replace quick-create with explicit session-creation surfaces that expose provider choice and return effective provider state.
- [ADR-005: Migrate Session Provider State In Place And Repair Legacy Metadata Once](adrs/adr-005.md) — Add provider persistence to existing local data safely with in-place migration and one-time repair for legacy session metadata.
