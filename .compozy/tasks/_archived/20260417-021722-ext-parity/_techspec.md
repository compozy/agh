# TechSpec: Shared Extensibility Resource Runtime

## Executive Summary

AGH will replace fragmented, domain-specific extensibility catalogs with a persisted shared resource runtime that becomes the authoritative desired-state control plane for the first-wave extensibility families: hooks, tools, agents, MCP servers, skills, automation definitions, bridge instances, bundles, and bundle activations. Extensions, daemon-owned config, and bundle activations will publish canonical resource records into one store; domain projectors will reconcile those records into the execution subsystems that actually run behavior.

The first implementation is intentionally narrower than the original draft. It keeps the same architectural direction but removes first-version machinery that is not justified yet. Consistency will be driven by full-snapshot reconcile, not by a public watch contract. Store writes will validate kind-specific schemas before persistence, use optimistic concurrency for direct mutations, and stamp owner, source, and scope from caller authority instead of trusting payload metadata. Extension snapshots can only mutate records owned by their own source, daemon-owned records are immune to snapshot overwrite, and extension read surfaces are source-scoped by default so secrets do not leak across workspaces or sources. The persistence kernel still operates on encoded JSON bytes because the transport and SQLite boundary require that shape, but domain-facing stores, codecs, and projector adapters hand concrete Go types to the rest of the system. The daemon will own a named reconcile driver with per-kind single-flight semantics, bounded coalescing, topology-aware ordering, and explicit shutdown semantics. The trade-off is still a meaningful platform refactor, but the initial control plane becomes smaller, more deterministic, and safer to decompose into execution tranches.

## System Architecture

### Component Overview

```text
extension manifests / daemon config / bundle activations / extension snapshots
                                 |
                                 v
                  internal/resources (new)
   persisted store + kind validators + authority stamping + CAS
                                 |
                                 v
                 reconcile driver (boot + post-commit)
                  |            |             |             |
                  v            v             v             v
         hooks projector   tool/agent    automation     bridges
         + tool wiring     publishers     projector      projector
                                 |                          |
                                 +------------+-------------+
                                              v
                                bundles projector and execution subsystems
```

Main components and boundaries:

- `internal/resources` (new): canonical persisted resource store, kind codecs, typed store façades, projector adapters, read and mutation authorization, optimistic concurrency, owner/source stamping, and list primitives for full-snapshot reconcile.
- `internal/resources` reconcile driver (new component, implementation may live in the same package): per-kind single-flight scheduler that coalesces repeated triggers into one pending rerun, honors projector dependency order, enforces reconcile timeouts, and runs after boot and after successful commits.
- `internal/extension/surfaces` (new): static resource-surface metadata consumed by the manifest loader, capability checker, handshake builder, and SDK/codegen. This is a table-driven package, not a runtime-pluggable catalog abstraction.
- `internal/extension`: split current manager responsibilities into process supervision, handshake negotiation, and resource publishing. The extension layer stops being the authoritative catalog for hooks, tools, agents, bundles, and MCP servers.
- `internal/config`: extends `ExtensionsConfig` with resource-grant and rate-limit policy under a new `[extensions.resources]` section so source-tier ceilings and operator allowlists have one named configuration home.
- `internal/hooks`: consumes `hook.binding` records, replaces hand-maintained hook-family wiring with taxonomy-driven bindings, and closes the missing `tool.*` and `permission.*` runtime paths.
- `internal/automation`: reconciles `automation.job` and `automation.trigger` records into execution state; automation runs remain an operational read model.
- `internal/bridges`: reconciles `bridge.instance` records into bridge runtime state; delivery state, routes, and health remain operational read models.
- `internal/bundles`: treats bundles and bundle activations as resources; bundle activations compose owned downstream resources instead of writing bespoke activation inventory.
- `internal/api/httpapi`, `internal/api/udsapi`, and contract types: expose one canonical operator-facing resource CRUD surface while preserving domain-specific operational endpoints where the data is not desired state.

Data flow:

1. Extension manifests and daemon-owned config publish static resource records into `internal/resources`.
2. Dynamic extension providers publish complete source snapshots through a generic extension service instead of feature-specific flags such as `provide_tools`.
3. `internal/resources` derives a concrete `MutationActor` from the caller boundary, narrows scope, validates the kind-specific spec, enforces concurrency rules, stamps owner and source metadata, and persists the canonical record.
4. After commit, the reconcile driver schedules the affected kind and its dependents using per-kind single-flight semantics.
5. The projector for that kind reads the full persisted snapshot, builds the next runtime state off-path, and swaps it atomically into the execution subsystem.
6. Existing operational subsystems continue to expose runtime-only state such as hook runs, automation runs, bridge routes, and health.
7. Bundle activations create owned resources through the same store, so deleting an activation removes only the resources owned by that activation through owner-filtered deletes.

## Implementation Design

### Core Interfaces

Mutation actor:

```go
type MutationActor struct {
	Kind          MutationActorKind
	ID            string
	SessionNonce  string
	Source        ResourceSource
	MaxScope      ResourceScope
	GrantedKinds  []ResourceKind
	GrantedScopes []ResourceScopeKind
}
```

Internal raw persistence kernel:

```go
type rawStore interface {
	PutRaw(ctx context.Context, actor MutationActor, draft RawDraft) (RawRecord, error)
	DeleteRaw(ctx context.Context, actor MutationActor, kind ResourceKind, id string, expectedVersion int64) error
	ApplySourceSnapshotRaw(ctx context.Context, actor MutationActor, snapshot SourceSnapshot) error
	GetRaw(ctx context.Context, actor MutationActor, kind ResourceKind, id string) (RawRecord, error)
	ListRaw(ctx context.Context, actor MutationActor, filter ResourceFilter) ([]RawRecord, error)
}
```

Raw mutation shape used only at persistence and transport boundaries:

```go
type RawDraft struct {
	Kind            ResourceKind
	ID              string
	Scope           ResourceScope
	ExpectedVersion int64
	SpecJSON        []byte
}
```

Source-scoped snapshot shape:

```go
type SourceSnapshot struct {
	SourceVersion int64
	Records       []RawDraft
}
```

Persisted raw record shape:

```go
type RawRecord struct {
	Kind      ResourceKind
	ID        string
	Version   int64
	Scope     ResourceScope
	Owner     ResourceOwner
	Source    ResourceSource
	SpecJSON  []byte
	CreatedAt time.Time
	UpdatedAt time.Time
}
```

Typed domain mutation and record shapes:

```go
type Draft[T any] struct {
	ID              string
	Scope           ResourceScope
	ExpectedVersion int64
	Spec            T
}

type Record[T any] struct {
	Kind      ResourceKind
	ID        string
	Version   int64
	Scope     ResourceScope
	Owner     ResourceOwner
	Source    ResourceSource
	Spec      T
	CreatedAt time.Time
	UpdatedAt time.Time
}
```

Typed codec boundary:

```go
type KindCodec[T any] interface {
	Kind() ResourceKind
	DecodeAndValidate(ctx context.Context, scope ResourceScope, raw []byte) (T, error)
	Encode(spec T) ([]byte, error)
	MaxBytes() int
}
```

Reconcile driver boundary:

```go
type ReconcileDriver interface {
	Trigger(ctx context.Context, kind ResourceKind, reason ReconcileReason) error
	RunBoot(ctx context.Context) error
	Close(ctx context.Context) error
}
```

Typed store façade used by domain code:

```go
type Store[T any] interface {
	Put(ctx context.Context, actor MutationActor, draft Draft[T]) (Record[T], error)
	Delete(ctx context.Context, actor MutationActor, id string, expectedVersion int64) error
	Get(ctx context.Context, actor MutationActor, id string) (Record[T], error)
	List(ctx context.Context, actor MutationActor, filter ResourceFilter) ([]Record[T], error)
}
```

Internal reconcile adapter boundary:

```go
type ProjectionInput struct {
	Kind         ResourceKind
	Revision     int64
	Records      []RawRecord
	Dependencies map[ResourceKind][]RawRecord
}

type ProjectionPlan interface {
	Kind() ResourceKind
	Revision() int64
	OperationCount() int
}

type Projector interface {
	Kind() ResourceKind
	DependsOn() []ResourceKind
	Build(ctx context.Context, input ProjectionInput) (ProjectionPlan, error)
	Apply(ctx context.Context, plan ProjectionPlan) error
}
```

Typed projector boundary for single-kind domain code:

```go
type TypedProjector[T any] interface {
	Kind() ResourceKind
	DependsOn() []ResourceKind
	Build(ctx context.Context, records []Record[T]) (ProjectionPlan, error)
	Apply(ctx context.Context, plan ProjectionPlan) error
}
```

Implementation rules:

- `internal/resources` owns canonical desired state only. It does not become a generic runtime-state bucket.
- Operational data such as automation runs, hook runs, bridge delivery state, and session health remain in their domain systems.
- Consistency is driven by full-snapshot reconcile. V1 does not expose a `Watch` store contract, `resources/watch`, or a resource event stream.
- A local invalidation hint may exist later inside the daemon, but reconcile correctness cannot depend on it.
- Raw JSON bytes exist only at transport and persistence boundaries. The shared runtime keeps that raw kernel internal to `internal/resources`.
- `MutationActor` is derived by the daemon from the actual caller path: operator HTTP or UDS request, daemon-internal call, or extension session.
- `MutationActor.SessionNonce` is generated by the daemon with at least 128 bits from `crypto/rand` and encoded as a stable opaque string.
- `RawDraft.Scope` is mandatory on every mutation. Omitted scope is rejected. `global` requires empty `scope_id`; `workspace` requires a non-empty `scope_id`.
- Domain packages do not manipulate `RawDraft`, `RawRecord`, or `[]byte` directly. They receive typed `Store[T]`, `Record[T]`, and `TypedProjector[T]` adapters.
- `KindCodec[T]` is the only shared place where raw JSON is decoded into typed domain specs or encoded back to bytes.
- A record may be decoded again when it crosses a new daemon boundary such as boot rebuild or a later reconcile pass, but the runtime must not repeatedly reparse the same record inside one validation or projection pass.
- `Store[T]` keeps `MutationActor` on `Get` and `List` intentionally, so the typed façade itself remains the server-authoritative enforcement point for source, scope, and grant filtering.
- `Get` and `List` enforce the same authority boundary as writes. Extension actors can read only records from their own `Source` and granted kinds within `MaxScope`. Cross-source reads are denied in v1 unless the daemon later adds an explicit public-read surface for a kind.
- `Put` and `Delete` use optimistic concurrency. Direct creates use `ExpectedVersion=0`; updates and deletes must match the current stored version.
- `MutationActorKindExtension` cannot call direct `Put` or `Delete` in v1. Extension publication uses `ApplySourceSnapshot`; operator and daemon actors use direct CRUD.
- `ApplySourceSnapshotRaw` is serialized per `(source_kind, source_id)`, may create new records for that source, and may update or delete only records whose stored `Source` exactly matches the actor source.
- If a snapshot record targets an existing `(kind, id)` owned by another source or by the daemon, snapshot apply fails with conflict instead of overwriting that record.
- The daemon pins one active `SessionNonce` per live extension session. Snapshots with a nonce different from the active session are rejected before acquiring the per-source lock.
- A new live extension session starts with `SourceVersion=1`. `resource_source_state` initialization or session reset happens inside the same per-source transaction using `INSERT ... ON CONFLICT DO UPDATE`, so concurrent bootstrap calls cannot both win.
- Operator-driven source reset deletes the source-owned records and the corresponding `resource_source_state` row in the same transaction.
- The reconcile driver runs one in-flight reconcile per kind. Concurrent triggers for the same kind set one dirty flag and at most one queued rerun after the current pass finishes.
- The reconcile driver uses a bounded coalescing window before reruns, defaulting to `50ms`. When a kind is idle, reconcile should begin within `250ms` of commit. Under a write storm, the driver must never hold more than one in-flight pass plus one pending rerun per kind.
- One committed `Put`, `Delete`, or `ApplySourceSnapshot` call triggers at most one reconcile request per affected kind, regardless of how many records were changed inside that commit.
- `Trigger` returns an error only for driver-closed, unknown-kind, or enqueue-path failures. Projector failures are recorded in health state, logs, and metrics; they do not bubble back through an already committed write.
- Every reconcile pass runs with a per-kind timeout derived from daemon config and propagates that deadline into `Projector.Build` and `Projector.Apply`.
- Repeated projector failures open a degraded circuit for that kind, emit health state, and suppress busy-loop reruns until a bounded backoff elapses or a new committed write arrives.
- `Close` stops accepting new triggers, drains or cancels in-flight work within the caller deadline, and guarantees no owned goroutine outlives driver shutdown.
- `RunBoot` schedules migrated kinds in topology order from `Projector.DependsOn()`. Post-commit triggers schedule the written kind first, then its dependents using a reverse dependency index built from the registered projectors.
- In the initial topology, `hook.binding`, `tool`, `agent`, `skill`, `mcp_server`, `automation.job`, `automation.trigger`, `bridge.instance`, and `bundle` are root kinds. `bundle.activation` depends on `bundle`.
- `DependsOn()` expresses reconcile ordering only. It does not represent ownership fan-out. `bundle.activation` materializes owned `automation.*` and `bridge.instance` records through store writes during `Apply`, and those writes trigger their own kinds without creating a reverse dependency edge back to `bundle.activation`.
- `provide_tools` is removed as a one-off negotiated feature once tools migrate. Dynamic contribution uses `resources/snapshot`.
- `session.HookSet` and `internal/daemon/hooks_bridge.go` stop hand-enumerating supported families. Binding support is derived from hook taxonomy plus persisted `hook.binding` records.
- `ProjectionInput` is internal to `internal/resources`. The reconcile driver assembles it and the registered projector adapter is responsible for decoding raw records into typed domain inputs before domain `Build` logic runs.
- Kinds whose build logic depends only on their primary resource kind register a `TypedProjector[T]` behind a thin adapter that implements the internal `Projector` interface.
- `TypedProjector[T]` is the standard contract for single-kind domain projection. Mixed-kind outliers, such as `bundle.activation`, do not implement `TypedProjector[T]` directly; they use package-local `Projector` adapter code to decode dependency kinds once and call explicit domain build functions without leaking raw JSON through shared call sites.
- `ProjectionPlan` is projector-private data plus generic metadata for logging, timeout handling, and contract tests. One plan may represent multiple object changes, but `Apply` treats the full plan as one atomic daemon-visible transition.
- Data projectors must build the next runtime state without mutating live state and swap atomically during `Apply`.
- External-state projectors such as bridges and MCP servers may not open speculative live connections during `Build`. They compute a validated delta plan off-path, atomically swap the daemon-visible registry or desired runtime map during `Apply`, then converge external processes or connections with rollback or degraded-state marking if side effects fail.

### Data Models

First-wave canonical resource kinds:

- `hook.binding`
- `tool`
- `agent`
- `mcp_server`
- `skill`
- `automation.job`
- `automation.trigger`
- `bridge.instance`
- `bundle`
- `bundle.activation`

Manifest model:

- Keep family-oriented resource declarations for operator readability, but derive runtime metadata from a static `internal/extension/surfaces` table.
- Add first-class manifest contribution for `tools` and automation definitions alongside the already existing hook, agent, skill, MCP, and bundle contributions.
- Bridge-capable extension manifests continue declaring provider metadata such as display name, secret slots, and config-schema hints, but concrete `bridge.instance` desired state remains operator-, daemon-, or bundle-authored in v1.
- Static manifest contributions and dynamic extension snapshots produce the same persisted raw-record shape after daemon stamping, while domain code consumes typed `Record[T]` values through codecs and store adapters.

SQLite persistence in `globaldb`:

| Table                   | Purpose                                                                                              |
| ----------------------- | ---------------------------------------------------------------------------------------------------- |
| `resource_records`      | Canonical desired-state records for all covered kinds                                                |
| `resource_source_state` | Tracks the active snapshot session and last accepted snapshot version per `(source_kind, source_id)` |
| existing run tables     | Remain for runtime-only state such as `automation_runs`, hook runs, and operational logs             |

`resource_records` columns:

- `kind TEXT NOT NULL`
- `id TEXT NOT NULL`
- `version INTEGER NOT NULL`
- `scope_kind TEXT NOT NULL CHECK (scope_kind IN ('global', 'workspace'))`
- `scope_id TEXT`
- `owner_kind TEXT NOT NULL`
- `owner_id TEXT NOT NULL`
- `source_kind TEXT NOT NULL`
- `source_id TEXT NOT NULL`
- `spec_json TEXT NOT NULL`
- `created_at TEXT NOT NULL`
- `updated_at TEXT NOT NULL`

`resource_source_state` columns:

- `source_kind TEXT NOT NULL`
- `source_id TEXT NOT NULL`
- `session_nonce TEXT NOT NULL`
- `last_snapshot_version INTEGER NOT NULL`
- `updated_at TEXT NOT NULL`

Indexes:

- primary key: `(kind, id)`
- `idx_resource_kind` on `(kind)`
- `idx_resource_scope` on `(scope_kind, scope_id, kind)`
- `idx_resource_owner` on `(owner_kind, owner_id, kind)`
- `idx_resource_source` on `(source_kind, source_id, kind)`
- primary key on `resource_source_state`: `(source_kind, source_id)`

Ownership model:

- Bundle-produced records are linked through `owner_kind` and `owner_id`, which replaces `bundle_activation_inventory`.
- Extension-published records use `source_kind=extension` and `source_id=<extension name>`, stamped by the daemon from the active extension session.
- Snapshot sequencing for extension-published records uses the daemon-issued `SessionNonce` for that active extension session.
- Daemon-owned records use `source_kind=daemon` or a more specific system source, stamped by the daemon write path.
- In v1, `bridge.instance` records are not extension-publishable. They are authored by operator or daemon flows, then consumed by managed bridge-provider extensions through family-specific operational Host APIs.
- Source reset deletes both source-owned records and `resource_source_state` for that source in one transaction so a new session can bootstrap from `SourceVersion=1`.
- User payloads never supply authoritative `owner` or `source` values.

### Authority and Validation Rules

Mutation authority rules:

- Operator APIs can create daemon-owned resources only through operator-authorized write endpoints.
- Extension subprocesses can publish only the resource kinds and scopes granted during handshake.
- Extension subprocesses do not receive direct resource-mutation methods in v1. They can call only `resources/list`, `resources/get`, and `resources/snapshot`.
- `resources/snapshot` is source-scoped: an extension can replace only the records currently owned by its own source identity and granted kinds.
- Workspace-scoped callers cannot create global resources.
- Delete operations are allowed only for daemon authority or for records owned by the calling source within the granted kind set.

Read authority rules:

- Operator and daemon actors can read any record within their authorized scope boundary.
- Extension `resources/list` and `resources/get` calls are filtered by `GrantedKinds`, `MaxScope`, and the calling `Source`.
- Record specs are not public by default because several kinds may embed credentials, URLs, or other secrets. Cross-source reads are denied in v1.
- Managed bridge-provider extensions are the explicit exception to generic same-source reads: they do not use `resources/list` or `resources/get` for assigned `bridge.instance` visibility, and instead continue using family-specific authorized bridge Host APIs backed by canonical desired state.
- If a future kind needs public read semantics, that must be declared explicitly in `internal/extension/surfaces` and reviewed as a separate security decision.

Grant-authority model:

- The extension manifest declares requested resource families.
- `internal/extension/surfaces` declares which kinds are extension-publishable and which scopes are legal for each kind.
- `CapabilityChecker` computes the granted kinds and scopes as the intersection of manifest requests, static surface policy, source-tier ceilings, operator configuration, and session mode.
- The named source-tier ceilings remain defined in `internal/extension/capability.go`.
- The named operator policy lives in a new `[extensions.resources]` config section under `internal/config.ExtensionsConfig`, merged with the existing config overlay path.
- Effective grant precedence is: surface legality -> source-tier ceiling -> operator config allowlist -> manifest request -> session-mode scope narrowing.
- The initialize handshake carries the computed grants to the extension session. `resources/snapshot` enforcement uses those daemon-computed grants, not extension self-declaration.

Validation rules:

- The store resolves the validator by `ResourceKind` before persistence.
- Invalid specs are rejected before they reach any projector.
- Each kind validator enforces schema, semantic invariants, and a maximum payload size.
- Snapshot payloads enforce total record count and total byte ceilings per call.
- `resources/snapshot` is rate-limited per source and accepts at most one queued update while a snapshot for that source is in flight.
- Direct operator writes to `/api/resources` are rate-limited per actor to bound reconcile-flood abuse.

Transport authority rules:

- UDS resource writes are operator-only because the UDS API is the daemon's local control plane.
- HTTP `/api/resources` remains disabled until explicit operator auth middleware exists for those routes. The initial implementation exposes mutating resource APIs over UDS only.
- Anonymous or cross-workspace public resource reads or writes are out of scope.

Bundle safety rules:

- `bundle.activation` may own only allowlisted resource kinds declared by the bundle system.
- Activation deletion deletes only records owned by that activation. It cannot target arbitrary resources by ID.

### API Endpoints

Canonical operator-facing desired-state API added to UDS first, with HTTP exposure gated on explicit operator auth:

| Method | Path                       | Description                                           |
| ------ | -------------------------- | ----------------------------------------------------- |
| GET    | `/api/resources`           | List records by `kind`, `scope`, `owner`, or `source` |
| GET    | `/api/resources/:kind`     | List one resource kind                                |
| GET    | `/api/resources/:kind/:id` | Read one record                                       |
| PUT    | `/api/resources/:kind/:id` | Create or replace one desired-state record            |
| DELETE | `/api/resources/:kind/:id` | Delete one desired-state record                       |

Request shape for `PUT /api/resources/:kind/:id`:

- `scope`
- `expected_version` (`0` for create, current version for update)
- `spec`

Request shape for `DELETE /api/resources/:kind/:id`:

- `expected_version` (current version required)

Response shape:

- `record`

Status codes:

- `200` or `201` for write success
- `204` for delete success
- `403` for capability or scope violations
- `409` for stale `expected_version`, stale `source_version`, or snapshot ownership conflict
- `413` for oversized record or snapshot payloads
- `429` for rate-limited direct writes or source snapshots
- `422` for invalid kind-specific spec or scope mismatch

Extension protocol changes:

- Add Host API read surface:
  - `resources/list`
  - `resources/get`
- Add generic extension service for dynamic publication:
  - `resources/snapshot`
- Initialize grants expand to include `granted_resource_kinds` and `granted_resource_scopes`, computed by the daemon during handshake.
- The initialize handshake also carries the daemon-issued `session_nonce` for that live extension session.
- `resources/list` and `resources/get` expose only same-source records within granted kinds and scopes in v1.
- Bridge-provider extensions keep family-specific operational Host API methods for daemon-assigned instance visibility and runtime reporting:
  - `bridges/instances/list`
  - `bridges/instances/get`
  - `bridges/instances/report_state`
  - `bridges/messages/ingest`
- Those bridge methods are backed by the canonical `bridge.instance` desired-state authority after bridge migration, but they are not re-expressed as generic same-source `resources/list` or `resources/get` calls.
- `resources/snapshot` submits a complete desired-state snapshot for the extension source and granted kinds, plus a monotonic `source_version` for that active source session.
- The daemon serializes snapshot application per source, rejects non-active `session_nonce` values, rejects stale `source_version` for the active session, diffs against the currently persisted records for that source inside one transaction, and bumps `Version` on every changed record.
- Snapshot apply rejects attempts to overwrite daemon-owned or foreign-source records that already occupy the same `(kind, id)`.
- Direct `resources/put` and `resources/delete` methods are intentionally absent from the extension Host API in v1.
- Keep typed operational Host API methods where the action is not resource CRUD, such as session control, task execution, memory backend operations, bridge assigned-instance visibility, state reporting, or bridge message ingestion.

Operational endpoints that remain family-specific because they expose runtime state rather than desired state:

- `/api/hooks/runs`
- `/api/hooks/events`
- `/api/automation/runs`
- `/api/bridges/:id/routes`
- `/api/bridges/:id/test-delivery`
- `/api/bundles/preview`

## Integration Points

This design does not add new third-party services. The relevant system boundaries are:

- Extension subprocess protocol over JSON-RPC:
  - `resources/snapshot` replaces special-case feature flags
  - capability gating remains for side-effecting APIs
  - source identity and granted resource kinds come from the negotiated extension session, not from request payload
- SQLite global database:
  - persists canonical resource definitions
  - remains the single durable source of desired state for covered families
- Config system:
  - adds `[extensions.resources]` for operator allowlists, scope ceilings, and resource publish rate limits
  - keeps grant precedence explicit instead of spreading resource policy across ad hoc call sites
- HTTP, UDS, and CLI contracts:
  - move resource-definition mutation onto one canonical transport surface
  - keep UDS as the initial mutating control plane until HTTP operator auth exists
  - preserve operational read models where generic resource records are the wrong abstraction
- Daemon composition root:
  - wires kind codecs and projector adapters directly
  - wires the reconcile driver directly
  - drives full-snapshot reconcile after boot and after successful writes
  - preserves explicit dependency edges instead of introducing an event bus

## Impact Analysis

| Component                                                | Impact Type | Description and Risk                                                                                                                         | Required Action                                                                     |
| -------------------------------------------------------- | ----------- | -------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- |
| `internal/resources`                                     | new         | New central control plane with raw persistence kernel, typed codecs, and authority stamping; high risk because multiple domains depend on it | Build contract tests first and keep codecs and adapters explicit                    |
| `internal/store/globaldb`                                | modified    | Adds canonical resource persistence and removes authority from migrated definition tables; medium-high risk                                  | Keep schema deterministic and cut over by family                                    |
| `internal/config`                                        | modified    | Adds `[extensions.resources]` policy, scope, and rate-limit config; medium risk if precedence is vague                                       | Keep config shape minimal and precedence explicit                                   |
| `internal/extension`                                     | modified    | Manager splits into supervisor, negotiator, and publisher; high risk due to current coupling                                                 | Remove catalog ownership as families migrate                                        |
| `internal/extension/surfaces`                            | new         | Static source of manifest, capability, and handshake metadata; medium risk if consumers drift                                                | Keep manifest loader, capability checker, handshake, and SDK aligned from one table |
| `internal/session` and `internal/daemon/hooks_bridge.go` | modified    | Hook binding becomes taxonomy-driven and adds missing tool and permission wiring; medium risk                                                | Replace the manual family matrix                                                    |
| `internal/hooks`                                         | modified    | Hook records become resource-backed; medium risk                                                                                             | Align runtime dispatch with full-snapshot projector output                          |
| `internal/automation`                                    | modified    | Definitions move to resources while runs stay local; high risk                                                                               | Add projector and delete old definition authority                                   |
| `internal/bridges`                                       | modified    | Desired instance definitions move to resources; high risk                                                                                    | Separate desired state from runtime state clearly                                   |
| `internal/bundles`                                       | modified    | Activations become resource-owned composition instead of bespoke inventory; medium-high risk                                                 | Replace activation inventory with owner-indexed resources                           |
| `internal/api/*` and `sdk/typescript`                    | modified    | Canonical resource CRUD plus protocol changes; medium risk                                                                                   | Regenerate contracts, fixtures, and extension SDK helpers                           |

## Testing Approach

### Unit Tests

Add focused unit coverage for:

- `internal/resources`
  - CRUD
  - read filtering
  - actor derivation
  - filter semantics
  - owner/source stamping
  - scope narrowing
  - explicit-scope rejection
  - optimistic concurrency conflicts
  - stale snapshot rejection
  - stale session nonce rejection
  - per-source snapshot serialization
  - snapshot conflict on foreign-source or daemon-owned records
  - delete authorization
  - read authorization
  - per-kind payload-size enforcement
  - typed store adapters
  - codec encode/decode coverage
  - reconcile dirty-bit coalescing and shutdown
- `internal/extension/surfaces`
  - kind-to-manifest and kind-to-protocol mapping
  - granted-capability derivation
- typed projector adapters
  - primary-kind decode once per reconcile pass
  - explicit dependency decode for heterogeneous projector inputs
- hook binding runtime
  - taxonomy-driven family registration
  - real `tool.*` and `permission.*` dispatch entrypoints
- bundle ownership
  - bundle activation expansion into owned resource records
  - cleanup on activation deletion
  - rejection of non-allowlisted owned kinds
- automation and bridge validators
  - kind-specific spec validation
  - scope and ownership rules

### Integration Tests

Each family cutover must ship with:

- resource runtime contract tests using real SQLite
- migrated-domain integration tests using real projectors and full-snapshot reconcile paths
- extension subprocess fixtures proving static and dynamic resource publication
- boot and reconcile smoke coverage for rebuild behavior
- `make verify`

Critical scenarios:

- invalid specs are rejected before persistence
- domain projectors and validators do not consume `json.RawMessage` directly
- extension read paths cannot observe records from another source
- updates with stale `expected_version` are rejected
- extension manifest publishes tools and hook bindings into the canonical store
- `resources/snapshot` publishes dynamic records, deletes removed records from the same source, and replaces the old `provide_tools` path
- concurrent snapshots from the same source serialize and reject stale `source_version`
- snapshots with a non-active `session_nonce` are rejected
- snapshots cannot overwrite daemon-owned or foreign-source records that already use the same `(kind, id)`
- a restarted extension session with a new `SessionNonce` can replace prior source state without inheriting stale snapshot version counters
- workspace-scoped extensions cannot publish global resources
- operator resource writes create or remove automation jobs and triggers through the projector
- bundle activation creates owned automation and bridge records, and deletion removes only those owned records
- write storms against one kind coalesce to one in-flight pass plus one pending rerun without unbounded queue growth
- reconcile driver shutdown drains or cancels in-flight work within deadline
- projector failure preserves the previously applied runtime state for that kind because `Build` is side-effect free and `Apply` swaps atomically

## Development Sequencing

### Build Order

1. Add `internal/resources`, canonical schema, `MutationActor`, optimistic concurrency, source-scoped read enforcement, source-snapshot serialization, reconcile-driver contracts, rate limiting, and resource contract tests; no dependencies.
2. Add `internal/extension/surfaces`, `[extensions.resources]` config, handshake `session_nonce` plus resource grants, capability-checker integration, and SDK support for `resources/list`, `resources/get`, and `resources/snapshot`; depends on step 1.
3. Tranche 1, lane A: migrate hooks to resource-backed binding, including `tool.*` and `permission.*` runtime wiring; depends on steps 1 and 2.
4. Tranche 1, lane B: migrate tools, agents, skills, and MCP servers to canonical resource publication; depends on steps 1 and 2.
5. Run tranche-1 verification and confirm the reconcile-driver, CAS, snapshot ownership, read authority, and projector-swap pattern against hooks and tools before broader family migration; depends on steps 3 and 4.
6. Tranche 2: migrate automation definitions to full-snapshot projection and remove legacy definition authority; depends on step 5.
7. Tranche 2: migrate bridge instances to full-snapshot projection and remove legacy definition authority; depends on step 5.
8. Tranche 3: migrate bundles and bundle activations to owner-indexed resource composition; depends on steps 6 and 7.
9. Enable HTTP `/api/resources` only if operator auth middleware lands with the same tranche; otherwise keep HTTP disabled and continue using UDS as the mutating control plane; depends on step 1.
10. Remove obsolete family authorities as each tranche lands, then run the full cutover verification gate after steps 3 through 9; depends on steps 3 through 9.

Steps 3 and 4 are the recommended first execution tranche for a small team because they close shipped hook and tool gaps while validating the shared runtime pattern before automation, bridges, and bundles migrate.

### Technical Dependencies

Blocking dependencies before implementation starts:

- one canonical `internal/resources` package and schema must exist before any family migration
- extension protocol and SDK changes must land before dynamic resource publication can replace `provide_tools`
- reconcile-driver semantics, source-snapshot serialization, read authority, and optimistic concurrency must be specified before implementation begins
- HTTP resource mutation routes stay disabled until operator auth middleware exists for those routes
- hook migration should happen early because it closes concrete shipped gaps and validates the full-snapshot projection pattern on a narrower family
- bundle migration must happen after automation and bridges because bundle activations compose those families

Tranche-1 verification checklist:

- no legacy hook-family path remains authoritative for migrated hook bindings
- `tool.*` and `permission.*` hooks are wired end to end through the migrated runtime
- direct write CAS tests and source-snapshot ownership tests pass
- same-source read filtering and cross-source read denial tests pass
- reconcile driver single-flight, timeout, shutdown, and degraded-circuit tests pass
- projector atomic-swap tests pass for one data projector and one external-state projector
- `make verify` passes before tranche 2 starts

## Monitoring and Observability

Key metrics:

- `resource_write_total{kind,result}`
- `resource_delete_total{kind,result}`
- `resource_snapshot_total{source_kind,result}`
- `resource_snapshot_rejected_total{reason}`
- `resource_cas_conflict_total{kind}`
- `resource_snapshot_conflict_total{source_kind}`
- `resource_snapshot_rate_limited_total{source_kind}`
- `resource_read_denied_total{actor_kind,kind}`
- `resource_reconcile_requested_total{kind,reason}`
- `resource_reconcile_coalesced_total{kind}`
- `resource_reconcile_duration_seconds{kind}`
- `resource_reconcile_failures_total{kind}`
- `resource_reconcile_degraded_total{kind}`
- `resource_projection_applied_total{kind}`

Structured log fields:

- `resource_kind`
- `resource_id`
- `scope_kind`
- `scope_id`
- `owner_kind`
- `owner_id`
- `source_kind`
- `source_id`
- `actor_kind`
- `actor_id`
- `reconcile_reason`

Operational signals:

- repeated validation rejects for one source or kind
- repeated read denials for one source or scope
- repeated scope or capability denials
- repeated CAS or stale-snapshot conflicts
- repeated non-active nonce rejects for one source
- repeated snapshot rate limiting for one source
- boot rebuild duration regression after a family cutover
- projector failures causing stale runtime state
- orphaned owned resources after bundle activation deletion

## Technical Considerations

### Key Decisions

- Decision: the shared resource runtime is authoritative desired state for all covered families, not only extension-owned records.
  Rationale: eliminates split authority and removes per-domain catalog drift.
  Trade-off: broader migration and more subsystem touch points.
  Alternatives rejected: extension-only runtime, targeted gap patching.

- Decision: consistency is driven by full-snapshot reconcile, not by a public watch contract.
  Rationale: SQLite does not provide a durable notification primitive, and the codebase explicitly avoids event-bus architecture.
  Trade-off: reconcile does more repeated read work after writes and at boot.
  Alternatives rejected: `Watch` as a correctness mechanism, resource event streaming in v1.

- Decision: the store validates kind-specific specs and stamps owner, source, and scope from caller authority before persistence.
- Decision: raw JSON remains confined to the persistence and transport kernel, while domain code consumes typed stores and typed projector adapters.
  Rationale: AGH is a Go codebase that benefits from concrete types, and `json.RawMessage` leaking into projectors or validators would reintroduce type erasure at the exact call sites that need invariants.
  Trade-off: the runtime needs explicit codecs and thin adapters per kind.
  Alternatives rejected: raw JSON at all call sites, one generic projector API that forces mixed-kind dependencies into untyped blobs.

- Decision: the store validates kind-specific specs and stamps owner, source, and scope from caller authority before persistence.
  Rationale: invalid or forged desired state should never enter the canonical store.
  Trade-off: more logic in the write path and codec registry.
  Alternatives rejected: projector-only validation, self-asserted metadata, deferred authorization.

- Decision: direct mutations use optimistic concurrency, and extension snapshots are serialized per source with monotonic `source_version`.
  Rationale: concurrent writes and retried snapshots must not silently overwrite newer desired state.
  Trade-off: more coordination in the write path and one more piece of source-state metadata.
  Alternatives rejected: silent last-writer-wins, unlocked snapshot apply, timestamp-only conflict detection.

- Decision: extension reads and snapshots are source-scoped by default, and extensions do not get direct resource CRUD in v1.
  Rationale: resource specs can carry credentials or other sensitive config, so the safest initial contract is own-source publication plus own-source readback.
  Trade-off: some cross-source inspection use cases must wait for an explicit public-read design.
  Alternatives rejected: unrestricted extension reads, extension-side direct `Put` and `Delete`.

- Decision: desired-state CRUD becomes generic, while operational runtime APIs stay typed.
  Rationale: resource definitions and runtime state are different concerns.
  Trade-off: the platform keeps both generic and domain-specific endpoints.
  Alternatives rejected: forcing runs, health, or delivery state into generic resource records.

- Decision: dynamic extension contribution uses `resources/snapshot` instead of special-case flags like `provide_tools`.
  Rationale: avoids reintroducing surface-by-surface protocol drift.
  Trade-off: protocol and SDK changes are larger up front.
  Alternatives rejected: shipping `provide_tools` end to end as another one-off mechanism.

- Decision: cut over by family with direct removal of replaced authority in each phase, but organize the middle phases as parallel lanes.
  Rationale: preserves architectural integrity while still supporting progressive tasks and shorter time to validation.
  Trade-off: sequencing and test isolation must be explicit.
  Alternatives rejected: big-bang rewrite, long-lived dual-write, fully serial family migration.

- Decision: the reconcile driver is a named control-plane component with single-flight per-kind scheduling and topology-aware ordering.
  Rationale: without an explicit scheduler, post-commit and boot-time reconcile behavior remains ambiguous and error-prone.
  Trade-off: one more foundational component must exist before domain migrations begin.
  Alternatives rejected: ad hoc post-write callbacks, projector self-scheduling, hidden dependency ordering.

### Known Risks

- Risk: `internal/resources` becomes too generic and starts hiding domain invariants.
  Mitigation: codecs, typed stores, and projector adapters remain kind-specific; runtime stores desired state only.

- Risk: typed adapters add another layer and drift from raw-store semantics.
  Mitigation: keep the raw kernel internal, keep adapters thin, and require adapter contract tests alongside codec tests.

- Risk: automation, bridges, and bundles create circular migration pressure.
  Mitigation: migrate automation and bridges in parallel lanes, then migrate bundle composition afterward.

- Risk: projector failure semantics become ambiguous across domains.
  Mitigation: require full-snapshot idempotent reconcile, side-effect-free `Build`, and atomic `Apply`.

- Risk: protocol churn breaks extension fixtures and SDKs.
  Mitigation: land protocol, SDK, and subprocess fixture tests in the same phase as resource negotiation.

## Architecture Decision Records

- [ADR-001: Adopt a Shared Resource Runtime as the Authoritative Extensibility Control Plane](adrs/adr-001.md) — Makes one persisted runtime the authoritative desired-state store for all covered extensibility families.
- [ADR-002: Migrate Covered Domains Through Phased Clean Cutovers](adrs/adr-002.md) — Delivers the refactor progressively without dual-write or deprecated compatibility layers.
- [ADR-003: Gate Every Domain Cutover With Contract, Integration, and Reconcile Verification](adrs/adr-003.md) — Requires `make verify` plus contract, integration, and boot/reconcile evidence for every cutover.
- [ADR-004: Use Snapshot-First Reconcile for Resource Consistency](adrs/adr-004.md) — Makes full-snapshot reconcile the correctness path and treats change notification as an optional local optimization only.
- [ADR-005: Make Resource Access Server-Authoritative](adrs/adr-005.md) — Requires daemon-side validation, source stamping, scope enforcement, and same-source read filtering for resource access.
- [ADR-006: Use a Topology-Aware Reconcile Driver](adrs/adr-006.md) — Defines the named single-flight scheduler that drives post-commit and boot-time reconcile in dependency order.
- [ADR-007: Use Optimistic Concurrency and Serialized Source Snapshots](adrs/adr-007.md) — Restores record `Version`, CAS writes, and per-source snapshot serialization with stale-version rejection.
- [ADR-008: Confine Raw JSON to the Persistence Boundary and Expose Typed Domain Adapters](adrs/adr-008.md) — Keeps raw bytes inside the runtime kernel while stores and projectors hand concrete Go types to domain code.
