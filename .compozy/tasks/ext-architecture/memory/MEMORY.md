# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State
- Task 01 is implemented and verified; `internal/tools/` now exists as the minimal tool foundation for extension work.
- Task 04 is implemented and verified; `internal/extension/capability.go` now computes effective action/security grants with source-tier ceilings and returns typed `-32001`-equivalent denials.
- Task 06 is implemented and verified; `internal/extension/manager.go` now owns extension runtime lifecycle, resource registration, status snapshots, and crash recovery for installed extensions.
- Task 07 is implemented and verified; `internal/extension/host_api.go` now exposes the negotiated Host API method inventory with typed RPC errors and per-extension rate limiting.
- Task 08 is implemented and verified; daemon boot now starts the extension manager between hooks and servers, rebuilds hook declarations after extension startup, and stops extensions before session/server teardown.
- Task 09 is implemented and verified; `agh extension {list, install, enable, disable, status}` now supports daemon-online UDS flows, daemon-offline registry flows, human/json/toon output, and the required CLI integration coverage.
- Task 10 is implemented and verified; TypeScript extension authors now have `@agh/extension-sdk` in `sdk/typescript/`, `@agh/create-extension` in `sdk/create-extension/`, a testing harness export, and repo workspace/Vitest wiring for both packages.

## Shared Decisions
- The canonical extension/protocol `Tool` JSON shape remains `name`, `description`, `input_schema`, `read_only`, and `source`; hook compatibility is handled by accepting `tool_name` as a decode alias instead of changing the wire shape.
- `internal/subprocess/` uses a custom line-delimited JSON-RPC transport tuned to `_protocol.md` (10 MiB frame cap, no notifications, explicit drain gating); ACP reuses the package only for raw subprocess lifecycle by launching with transport disabled and keeping `coder/acp-go-sdk` on the raw stdio pipes.
- On-disk extension manifests use wrapped core metadata (`[extension]` / `"extension"`), but `internal/extension.LoadManifest` flattens those fields into the exported `Manifest`; downstream extension tasks should consume the flat struct, not expect nested core metadata.
- `internal/extension.Registry` verifies and stores checksums for the entire extension directory artifact, while `manifest_path` persists the resolved `extension.toml` or `extension.json` file inside that directory; the public `Install` path defaults the stored source tier to `user`, and non-user tiers currently flow through package-local `installWithSource`.
- Extension-provided skills are injected through `skills.Registry.RegisterExternal` / `RemoveExternal` overlays instead of copying files into bundled or user skill roots; future extension tasks should treat the skills registry as an in-memory overlay point for extension resources.
- `internal/extension/HostAPIHandler` is transport-agnostic but returns protocol-aligned `*subprocess.RPCError` values directly; `Manager.wrapHostHandler` injects the caller extension name into the request context so handler-level capability checks can run with full extension identity.
- Host API memory methods bridge onto the existing markdown-backed `memory.Store` rather than a new key/value backend; tags are preserved in an `agh-tags` HTML comment, and recall/search works by reading and scoring those persisted documents.
- The daemon composition root owns one boot-scoped `HostAPIHandler` and registers its `MethodHandlers()` into `extension.Manager`; future extension runtime work should extend that boot wiring rather than register ad hoc Host API handlers elsewhere.
- Extension hook declarations are injected into the live hooks runtime by chaining `extensionDeclarationProvider(...)` into the existing config declaration provider and calling `Hooks.Rebuild()` after the extension manager start attempt, even when startup logs a non-fatal failure.
- Extension CLI online flows are served by dedicated UDS `/api/extensions` endpoints backed by a daemon-owned `ExtensionService`; daemon-offline CLI flows should continue opening the global DB directly and use `extension.Registry` instead of inventing a second local state layer.
- The daemon should publish extension runtime/service wiring whenever the global DB is available, even if zero extensions are currently installed, so install/list/status operations can work immediately after the daemon starts.
- The TypeScript SDK ships as dual ESM/CJS output with a separate `@agh/extension-sdk/testing` subpath export, while scaffolding lives in the sibling `@agh/create-extension` workspace package; follow-on TypeScript extension tasks should depend on those packages instead of embedding ad hoc transport/runtime code.

## Shared Learnings
- Marketplace extensions now default to a read-only security ceiling of `session.read`, `memory.read`, `observe.read`, `skills.read`, and `tool.read`; Host API action grants are derived from that ceiling via the protocol's method-to-security map.
- `CapabilityChecker.CheckHostAPI()` denies in two stages: first the exact Host API method allowlist (`granted_actions`), then the mapped security family (`granted_security`). Future Host API wiring should preserve that ordering when translating denials into JSON-RPC errors.
- The extension manager removes both capability grants and external skill overlays during stop/disable flows, so future daemon and CLI tasks should rely on `Manager` accessors (`HookDeclarations`, `AgentDefinitions`, `MCPServers`, `Statuses`, `Get`) instead of maintaining duplicate extension runtime state.
- Observer-backed Host API integration tests must seed the workspace row in the shared global database before creating sessions, otherwise session registration and event-summary persistence can fail behind foreign-key boundaries.
- `globaldb.GlobalDB` now exposes `DB()` as the narrow composition-root seam for boot-time adapters like `extension.Registry`; future tasks that need raw SQL access from the live global registry should prefer that accessor over widening unrelated daemon interfaces.
- Daemon shutdown now stops the extension runtime before sessions and servers; future lifecycle changes must preserve that ordering so extensions can shut down cleanly while transport and session dependencies still exist.
- Install/enable/disable daemon flows should call the public `extension.Manager.Reload()` seam and rebuild hooks afterward; future runtime mutation paths should reuse that single restart point rather than partially mutating in-memory extension state.
- SDK startup is order-sensitive on the shared stdio channel: ready callbacks that call Host API methods must fire after the initialize response is emitted to avoid startup races in real subprocess integrations.

## Open Risks

## Handoffs
- Task 03 added `internal/extension/manifest.go` and `internal/extension/manifest_test.go`; task 04 can depend on `Manifest.Actions.Requires`, `Manifest.Security.Capabilities`, and the typed `ErrManifestNotFound` / validation / compatibility errors.
- Task 04 added `ExtensionSource`, `CapabilityChecker`, `CapabilityDeniedData`, and `ErrCapabilityDenied`; task 06 should call `Register()` during extension load, and task 07 should convert `ErrCapabilityDenied` into the protocol `-32001 capability_denied` response with the existing `Data` payload.
- Task 06 added `Manager.Start`, `Stop`, `HookDeclarations`, `AgentDefinitions`, `MCPServers`, `Statuses`, and `Get`; task 07 should query runtime extension state through the manager, and task 08 should wire the manager into boot-time hook/resource composition instead of reloading manifests separately.
- Task 07 added `HostAPIHandler`, `WithHostAPICapabilityChecker`, `WithHostAPIWorkspaceResolver`, `WithHostAPIRateLimit`, `WithHostAPINow`, and `MethodHandlers`; task 08 should instantiate it once at boot and register each returned handler with the extension manager instead of wiring method handlers ad hoc.
