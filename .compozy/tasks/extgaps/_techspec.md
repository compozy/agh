# TechSpec: Extension Bundles and Activation Runtime

## Executive Summary

This change closes the extension packaging gaps by adding a new bundle resource model on top of the existing extension manifest system. Extensions can now declare static bundle specs, the daemon exposes those bundles as an operator-visible catalog, and operators explicitly activate bundle profiles through HTTP or UDS APIs. Activation is daemon-owned and persisted, which lets the daemon reconcile package-managed automations, bridge presets, and effective network defaults across boot, reload, update, and deactivation.

The primary trade-off is introducing a new daemon subsystem and new persistence tables instead of reusing extension-hosted runtime mutation through the Host API. The delivered design favors explicit ownership, deterministic reconciliation, and lifecycle enforcement over a smaller implementation footprint.

## System Architecture

### Component Overview

```
extension manifest/resources
        │
        ▼
internal/extension
(bundle loading + validation)
        │
        ▼
internal/bundles.Service
(catalog, preview, activate, reconcile, network settings)
        │
   ┌────┼──────────────┬──────────────┐
   ▼    ▼              ▼              ▼
globaldb  automation.Manager   bridges store/runtime   API core
persisted activations          package sync            HTTP/UDS handlers
```

Main components and boundaries:

- `internal/extension`: adds `resources.bundles`, loads bundle files, validates profiles, channels, jobs, triggers, and bridge presets.
- `internal/bundles`: owns bundle cataloging, activation lifecycle, activation inventory, reconcile, and effective network settings.
- `internal/store/globaldb`: persists bundle activations and per-activation inventory; extends bridge persistence with instance `source`.
- `internal/automation`: adds `JobSourcePackage` and a daemon-facing `SyncManagedDefinitions(...)` entrypoint so bundle-managed jobs and triggers can be reconciled as owned resources.
- `internal/bridges`: adds `BridgeInstanceSourcePackage`, read-only enforcement for managed instances, and secret-binding APIs usable by bundle-created bridge instances.
- `internal/api/core`, `httpapi`, `udsapi`: expose catalog, preview, activation CRUD, and bundle-derived network settings; extend bridge APIs for secret bindings.
- `internal/daemon`: boots the bundle runtime after extensions and automations, injects it into HTTP/UDS servers, and re-runs reconcile on extension reloads.

Data flow for activation:

1. An extension manifest declares `resources.bundles`.
2. `internal/extension` loads bundle specs during extension load/start.
3. The daemon-owned bundle service exposes those specs through catalog APIs.
4. An operator activates a bundle profile through `/api/bundles/activations`.
5. The service persists the activation, validates default-channel claims, and reconciles:
   - package automation definitions
   - package bridge instances
   - activation inventory
   - effective network default state
6. Session creation and status endpoints consume the computed effective network settings.

## Implementation Design

### Core Interfaces

Primary runtime type:

```go
type Service struct {
    store             Store
    automation        AutomationSyncer
    extensions        ExtensionInfoLister
    loadExtension     ExtensionLoader
    workspaceResolver workspace.WorkspaceResolver
    configuredDefault string
}
```

Daemon-facing automation reconcile contract:

```go
type AutomationSyncer interface {
    SyncManagedDefinitions(
        ctx context.Context,
        source automation.JobSource,
        desiredJobs []automation.Job,
        desiredTriggers []automation.Trigger,
        desiredTriggerSecrets map[string]string,
    ) (automation.SyncStats, error)
}
```

Activation request shape:

```go
type ActivateRequest struct {
    ExtensionName               string
    BundleName                  string
    ProfileName                 string
    Scope                       Scope
    Workspace                   string
    BindPrimaryChannelAsDefault bool
}
```

Error handling conventions:

- invalid bundle definitions fail at extension load/validation time with `extension.ErrBundleInvalid`
- missing bundle/profile/activation map to `404`
- conflicting default-channel claims and active-bundle lifecycle guards map to `409`
- bundle webhook triggers are explicitly rejected with `400`

### Data Models

Bundle declarations in extension resources:

- `Manifest.Resources.Bundles []string`
- `BundleSpec`
- `BundleProfile`
- `BundleChannelsConfig`
- `BundleJob`
- `BundleTrigger`
- `BundleBridgePreset`
- `BundleBridgeSecretSlot`

Persisted activation model:

- `bundles/model.Activation`
  - `ID`
  - `ExtensionName`
  - `BundleName`
  - `ProfileName`
  - `Scope`
  - `WorkspaceID`
  - `BindPrimaryChannelAsDefault`
  - `CreatedAt`
  - `UpdatedAt`
- `bundles/model.InventoryItem`
  - `ActivationID`
  - `ResourceKind`
  - `ResourceID`
  - `ResourceName`
  - `RecordedAtUTC`

SQLite storage added in global DB:

- `bundle_activations`
- `bundle_activation_inventory`

Bundle-owned resource identity in existing domains:

- automation:
  - `automation.JobSourcePackage`
  - package-managed jobs/triggers are reconciled through `SyncManagedDefinitions`
- bridges:
  - `bridges.BridgeInstanceSourcePackage`
  - `bridges.ErrBridgeInstanceReadOnly`
  - persisted `bridge_instances.source` column

Network runtime state exposed by bundles:

- `ConfiguredDefaultChannel`
- `EffectiveDefaultChannel`
- `EffectiveDefaultSource`
- `DeclaredChannels`

### API Endpoints

Bundle APIs added in HTTP and UDS:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/bundles/catalog` | List bundle catalog from installed extensions |
| POST | `/api/bundles/preview` | Preview activation result for one bundle profile |
| GET | `/api/bundles/activations` | List persisted activations |
| POST | `/api/bundles/activations` | Create or upsert one activation |
| GET | `/api/bundles/activations/:id` | Get one activation |
| PATCH | `/api/bundles/activations/:id` | Update activation overlays |
| DELETE | `/api/bundles/activations/:id` | Deactivate and reconcile removal |
| GET | `/api/bundles/network/settings` | Return configured/effective default channel and declared channels |

Bridge secret-binding APIs added in HTTP and UDS:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/bridges/:id/secret-bindings` | List secret bindings for one bridge instance |
| PUT | `/api/bridges/:id/secret-bindings/:binding_name` | Upsert one secret binding |
| DELETE | `/api/bridges/:id/secret-bindings/:binding_name` | Remove one secret binding |

Existing payloads extended:

- `contract.NetworkStatusPayload`
  - `configured_default_channel`
  - `effective_default_channel`
  - `effective_default_source`
  - `declared_channels`

Session creation behavior changed indirectly:

- if `CreateSessionRequest.channel` is empty, `defaultSessionChannel(...)` now prefers the bundle runtime’s effective default channel before falling back to static config.

## Integration Points

This implementation does not introduce external third-party integrations beyond existing AGH subsystems. Its integration points are internal subsystem boundaries:

- Extension loader and registry:
  - loads bundle specs from extension roots
  - blocks disable/uninstall when active bundle activations exist
- Automation manager:
  - accepts package-managed definitions through daemon-side sync
- Bridge runtime and persistence:
  - stores package-managed bridge instances
  - exposes secret-binding operations for those instances
- Workspace resolver:
  - resolves workspace-bound activations
- HTTP/UDS API layers:
  - expose catalog, lifecycle, and network settings to operators

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|---------------------|-----------------|
| `internal/extension` | modified | Manifest/resource surface expanded with bundle loading and validation; low-to-medium risk because invalid specs now fail earlier | Keep bundle schema and manifest docs aligned |
| `internal/bundles` | new | New daemon-owned runtime governs activation/reconcile behavior; medium risk due to lifecycle centralization | Maintain reconcile invariants and ownership rules |
| `internal/store/globaldb` | modified | New activation tables and bridge source persistence; medium risk around schema correctness | Preserve migration coverage and ID stability |
| `internal/automation` | modified | Adds package source and daemon-managed sync path; medium risk around source ownership semantics | Keep CRUD/read-only rules aligned with package source |
| `internal/bridges` | modified | Adds package source and secret-binding APIs; medium risk around managed-instance mutability | Preserve read-only enforcement for base specs |
| `internal/api/*` | modified | New bundle routes and expanded network payloads; low-to-medium transport risk | Keep HTTP, UDS, OpenAPI, and SDK output in sync |
| `internal/daemon` | modified | New boot step and extension-reload reconcile; medium risk due to boot ordering dependency | Preserve bundle boot after extensions and automations |

## Testing Approach

### Unit Tests

Concrete test additions and updates in this delivery include:

- `internal/bundles/service_test.go`
  - activation materialization
  - default-channel conflict rejection
- `internal/extension/registry_bundles_test.go`
  - disable/uninstall blocked by active bundle activations
- updated route inventory tests
  - `internal/api/httpapi/handlers_test.go`
  - `internal/api/udsapi/handlers_test.go`
- updated bridge persistence tests
  - `internal/store/globaldb/global_db_bridges_test.go`

### Integration Tests

Concrete verification executed for this delivery:

- `make codegen`
- `make verify`

`make verify` completed successfully after regenerating derived API artifacts and resolving final lint issues.

## Development Sequencing

### Build Order

1. Extend extension manifest/resources to load and validate bundle specs - no dependencies
2. Add bundle persistence models and global DB tables - depends on step 1
3. Introduce the daemon-owned bundle runtime in `internal/bundles` - depends on steps 1 and 2
4. Extend automation with `JobSourcePackage` and managed-definition sync - depends on step 3
5. Extend bridges with package source and secret-binding support - depends on step 3
6. Wire bundle APIs and network payload changes into API/core/http/uds - depends on steps 3, 4, and 5
7. Boot the bundle service from `internal/daemon` and reconcile on reload - depends on steps 3 through 6
8. Add and update regression tests, regenerate contracts, and run verification - depends on steps 1 through 7

### Technical Dependencies

Blocking dependencies that shaped the implementation:

- extension registry must expose the SQL DB handle used by the bundle service boot path
- automation manager must be available before `bootBundles(...)`
- extension runtime reload path must invoke bundle reconcile to keep package-managed resources current
- OpenAPI and generated SDK/types must be regenerated after API contract changes

## Monitoring and Observability

Operational visibility currently relies on existing daemon/API observability surfaces plus bundle-derived status payloads:

- daemon/network status now exposes:
  - configured default channel
  - effective default channel
  - effective default source
  - declared channels
- activation inventory persists the materialized resources per activation for inspection
- transport status codes distinguish:
  - missing bundle/profile/activation
  - conflicting default-channel claims
  - unsupported webhook triggers
  - active-bundle lifecycle conflicts

No new alerting or metrics subsystem was added in this delivery.

## Technical Considerations

### Key Decisions

- Decision: add `resources.bundles` as an additive extension resource instead of changing existing skills/agents/hooks/MCP semantics.
  - Rationale: preserve the existing extension-loading model and add a higher-level packaging layer beside it.
  - Trade-offs: one more resource family and validator path.
  - Alternatives rejected: overloading existing manifest sections to describe product/team packages.

- Decision: make activation daemon-owned and persisted.
  - Rationale: the daemon must own reconcile, inventory, and lifecycle enforcement.
  - Trade-offs: new runtime package and new DB tables.
  - Alternatives rejected: extension self-materialization via Host API.

- Decision: mark automation and bridge resources as package-owned in their own domains.
  - Rationale: reconcile and read-only enforcement need first-class ownership signals.
  - Trade-offs: extra source values, CRUD restrictions, and schema expansion.
  - Alternatives rejected: treating package resources as generic dynamic resources.

- Decision: compute effective default channels as runtime state.
  - Rationale: activation should not mutate config on disk.
  - Trade-offs: the system now carries both configured and effective defaults.
  - Alternatives rejected: rewriting daemon config or keeping channels informational only.

### Known Risks

- The current TechSpec reflects only shipped operator surfaces. No CLI workflow for bundle activation was implemented in this delivery.
- Bundle webhook triggers are explicitly unsupported; profiles that rely on webhook-style trigger materialization are rejected.
- The effective default channel is a runtime-computed value. Consumers that read only static network config can miss the active operational default.
- Bundle behavior depends on reconcile timing during boot and extension reload. Future changes in daemon boot ordering could regress activation correctness if not kept aligned with the current sequence.

## Architecture Decision Records

- [ADR-001: Use a daemon-owned bundle activation runtime](adrs/adr-001.md) — Moves bundle activation, persistence, and reconcile into a dedicated daemon subsystem.
- [ADR-002: Treat bundle-managed automations and bridges as package-owned resources](adrs/adr-002.md) — Adds explicit ownership markers so bundle-managed resources can be reconciled and protected from direct base-spec mutation.
- [ADR-003: Resolve bundle default channels as runtime state instead of mutating daemon config](adrs/adr-003.md) — Keeps static config unchanged and computes the effective network default from active bundle state.
