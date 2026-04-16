# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State

- Task 01 implementation and verification are complete; later tasks can assume the raw persistence kernel and schema exist.
- Task 02 implementation and verification are complete; later tasks can assume typed codecs, typed stores, and opaque projector registration adapters exist in `internal/resources`.
- Task 03 implementation and verification are complete; later tasks can assume daemon lifecycle owns a shared reconcile driver runtime.
- Task 04 implementation and verification are complete; later tasks can assume extension resource grants are daemon-derived from a shared surface registry plus operator/source/session ceilings.
- Task 05 implementation and verification are complete; later tasks can assume extension initialize now carries daemon-issued `session_nonce` plus granted resource kinds/scopes, and extensions can use `resources/list`, `resources/get`, and `resources/snapshot` through the negotiated Host API and TypeScript SDK.
- Task 07 implementation and verification are complete; later tasks can assume `hook.binding` is the first migrated desired-state family and the shared reconcile runtime now drives authoritative hook binding projection.
- Task 08 implementation and verification are complete; later tasks can assume `tool` and `mcp_server` are authoritative resource families, daemon boot syncs config and extension manifest declarations into canonical resources, extension dynamic tool publication now flows through `resources/snapshot`, and `provide_tools` is no longer part of the authoritative tool path.
- Task 09 implementation and verification are complete; later tasks can assume `agent` and `skill` are authoritative resource families, with config/workspace/extension declarations projected from canonical records.
- Task 10 implementation and verification are complete; later tasks can assume `automation.job` and `automation.trigger` desired state is projected from canonical resources while runs, overlays, webhook secrets, locks, and execution history remain automation-owned operational state.
- Task 11 implementation and verification are complete; later tasks can assume `bridge.instance` desired state is projected from canonical resources while delivery, route, health/status, degradation, assigned-instance visibility, and provider reporting remain bridge-owned operational state.
- Task 12 implementation and verification are complete; later tasks can assume `bundle` and `bundle.activation` are authoritative resource families, and bundle activation fan-out writes owned `automation.job`, `automation.trigger`, and `bridge.instance` records through canonical typed stores.

## Shared Decisions

- The canonical desired-state persistence boundary lives in `internal/resources`; `globaldb` only bootstraps the SQLite tables and indexes.
- Raw resource schema bootstrap now lives in `internal/resources.SchemaStatements()`, and `globaldb` appends those statements instead of duplicating the resource table definitions.
- Extension snapshot sequencing uses persisted source-session state in `resource_source_state` with explicit `ActivateSourceSession` and `ResetSource` APIs on the raw kernel.
- Task 02 layers the typed boundary in `internal/resources` through `KindCodec`, `RegisterCodec`/`ResolveCodec`, `NewStore`, and opaque `ProjectorRegistration` adapters; later boot wiring should register codecs explicitly instead of decoding raw JSON in domain packages.
- The raw reconcile input stays internal to `internal/resources`; single-kind domain code uses `NewTypedProjectorRegistration`, and `bundle.activation` is the only explicit mixed-kind adapter seam for now via `NewBundleActivationProjectorRegistration`.
- Task 03 keeps the reconcile runtime inside `internal/resources` but makes daemon boot/shutdown the sole owner of `ReconcileDriver` lifecycle.
- The default daemon reconcile factory currently boots an empty driver with no registered projectors; later family migrations must register projectors explicitly and trigger the driver from committed resource writes.
- Task 04 adds `internal/extension/surfaces` as the canonical static map from manifest resource families to first-wave resource kinds, including daemon-only kinds that extensions may not publish.
- Effective extension resource grants are now computed in `CapabilityChecker` as the intersection of surface legality, source-tier max scope, operator allowlist/max scope, manifest request, and session scope ceiling; extension manager state stores that computed snapshot for later startup and handshake consumers.
- Task 05 keeps generic `resources/*` Host API methods scoped to same-source desired-state reads and snapshot publication; bridge operational reads remain on `bridges/instances/*` and are not folded into generic resource access.
- Daemon runtime wiring now shares one `internal/resources` kernel between extension Host API handlers and manager-side source-session activation so nonce enforcement stays server-authoritative.
- Task 07 keeps `internal/hooks` resource-agnostic and moves hook-binding resource codecs/stores/projectors into `internal/daemon`, where daemon boot publishes native/config/agent/skill/extension declarations into canonical `hook.binding` resources whenever the shared kernel is present.
- Task 08 keeps `internal/tools` and `internal/config` resource-agnostic and places `tool` / `mcp_server` projectors plus static publication sync in `internal/daemon`, where one daemon-owned source syncer rebuilds config and manifest declarations under the canonical resource runtime.
- Task 09 keeps skill file/content loading, provenance checks, and MCP sidecar parsing in `internal/skills`; daemon-owned source sync publishes the resulting desired `agent`, `skill`, and attachment-derived `mcp_server` records into the canonical runtime.
- Task 10 keeps automation codecs/projectors in `internal/automation` with daemon wiring for stores/projector registration; daemon boot starts automation before resource reconcile so `RunBoot()` can rebuild scheduler and trigger runtime from persisted resource records.
- Task 11 keeps bridge operational surfaces on family-specific APIs (`bridges/instances/list|get|report_state`, delivery, routes, health/degradation) while the underlying desired bridge configuration is now canonical `bridge.instance` resource state.
- Task 12 keeps bundle activation as the explicit mixed-kind projector outlier: `bundle.activation` depends only on `bundle`, then writes owner-indexed downstream resource records and relies on post-commit triggers for automation/bridge projection.
- Bundle-managed bridge instances are now published through a daemon-managed `bridge.instance` resource source; direct legacy bridge-definition writes are no longer the desired-state authority after projection.
- Bundle activation cleanup is activation-scoped through `owner_kind=bundle.activation` and `owner_id=<activation-id>` and may only target the allowlisted owned kinds `automation.job`, `automation.trigger`, and `bridge.instance`.
- Hook runtime mutation now flows through `BuildBindingState` plus atomic `ApplyBindingState`, so projector build failures cannot partially corrupt the live dispatch table.
- Automation runtime mutation now flows through side-effect-free `BuildJobResourceState` / `BuildTriggerResourceState` and atomic apply methods, so failed reconcile preserves the previously applied scheduler and trigger engine.
- Bridge runtime mutation now flows through side-effect-free `BuildResourceState` and atomic `ApplyResourceState`; live extension reload failures roll the daemon-visible bridge projection back to the prior snapshot.
- Shared resource write paths now trigger reconcile after successful commits: operator CRUD writes in `internal/api/core/resources.go` and extension snapshots in `internal/extension/host_api*.go` both call the daemon reconcile driver instead of relying on direct projector callbacks.

## Shared Learnings

- Later transport and reconcile tasks should reuse the `internal/resources` nonce/version authority boundary instead of introducing separate source-session tracking.
- The generic JSON codec path validates typed specs by round-tripping through `DecodeAndValidate`, so later family migrations can keep validation at the codec boundary without adding a second validator interface.
- The reconcile runtime builds topology strictly from registered projector `DependsOn()` edges; ownership fan-out still happens through later store writes and follow-on triggers rather than synthetic dependency edges.
- Daemon boot now runs `ReconcileDriver.RunBoot()` before observer session reconcile so migrated families can rebuild desired state deterministically during startup.
- Workspace and marketplace extensions are now ceiling-limited to workspace publication scope even if manifests request global scope; bundled and user extensions may request global scope but still narrow through operator and session policy.
- SDK-facing resource Host API failures now normalize to HTTP-like JSON-RPC status codes (`403`, `409`, `413`, `429`), so later family migrations can reuse the same protocol/error surface instead of introducing method-specific error encodings.
- ACP session events are now the authoritative source for `tool.*` and `permission.*` hook dispatch; later hook-family work should extend the same agent-event translation path instead of reintroducing bespoke notifier plumbing.
- Tool and permission hook matching now supports agent/workspace scoping through `SessionContext`, so later family migrations can rely on the existing matcher model instead of special-casing those events.
- Operator CRUD and extension snapshot tests only rehydrate migrated kinds when their codec/projector is registered in the runtime under test; raw CRUD smoke that intentionally omits a projector should ignore trigger calls for unregistered kinds instead of relying on legacy side effects.
- Automation config/package enabled toggles are operational overlays after task 10; canonical `automation.job` / `automation.trigger` resource specs remain the desired defaults for managed sources.
- Bridge `bridge.instance` codecs validate provider-authored metadata through the live provider manifest lookup; later resource families with provider-owned metadata should keep that validation at the typed codec boundary.
- Legacy `bundle_activations` and `bundle_activation_inventory` tables are no longer part of the desired-state authority; active-bundle guards and boot rebuilds should read canonical `resource_records` instead.

## Open Risks

- UNCONFIRMED: Future non-first-wave resource families still need their own codecs/projectors and cutover plans before being treated as canonical desired state.

## Handoffs

- Future domain cutovers should inject projector registrations through the daemon reconcile factory rather than building sidecar schedulers.
- When a domain starts writing canonical resources, its post-commit path must trigger the shared driver after commit instead of adding direct projector callbacks.
- Later extensibility tasks can reuse the `hookBindingSourceSyncer` pattern when a legacy declaration source must be published into canonical resources before the bespoke authority is removed in the same tranche.
- Later family migrations should follow the same clean cutover pattern as task 08: move static declarations into a daemon-owned source syncer, move dynamic extension writes onto `resources/snapshot`, and delete one-off host API methods in the same tranche instead of leaving dual authority behind.
- Automation and later domain migrations can resolve agent and skill references through the canonical projections instead of reading config or skill registry discovery paths as definition authority.
- Task 12 can treat `bridge.instance` as canonical desired state and should compose bundle fan-out against resource-backed bridge sync rather than reintroducing direct `bridge_instances` writes.
- Later cleanup or extension lifecycle work must preserve bundle activation's owner-indexed deletion boundary; never delete downstream automation or bridge records without matching the specific activation owner.
