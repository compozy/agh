# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State
- Task 01 completed the foundational environment data model and workspace selection flow. `Config.ResolveEnvironment` now implements `Workspace.EnvironmentRef` -> `defaults.environment` -> implicit `local`.

## Shared Decisions
- Daytona startup policy treats `snapshot` as authoritative when both `snapshot` and `image` are configured; `image` remains as operator documentation/fallback input.
- Task 01 defines the `environment.ToolHost` contract only. Task 02 owns the concrete ACP/local ToolHost extraction and behavior preservation.

## Shared Learnings
- Workspace CRUD now carries `environment_ref` through Go contracts, HTTP/UDS handlers, CLI `workspace add/edit`, OpenAPI, and the generated web client type surface.
- ACP now exposes local runtime constructors `acp.NewLocalLauncher` and `acp.NewLocalToolHost`; local provider work should compose these into `environment.Prepared`.
- The provider registry type lives in `internal/environment` without importing provider subpackages; daemon/session wiring should import `internal/environment/local` from the composition root and use `local.NewRegistry()` or `environment.NewRegistry(local.NewProvider(...))` to avoid parent/child import cycles.
- Restart reconciliation must inspect persisted session metadata before `observer.Reconcile`, because observer reconciliation normalizes crashed non-terminal sessions to `stopped`.
- Remote providers can implement optional `environment.EnvironmentFinder` so daemon boot can find partial creates by `agh_environment_id` without creating a new sandbox during cleanup.
- Task 07 completed boot-time environment reconciliation after resource boot reconcile and before observer reconciliation. Reconciliation is non-blocking, reattaches recoverable remote sessions by persisted provider state, finds partial creates by `agh_environment_id`, destroys terminal/unrecoverable instances, and skips `Prepare` when no existing instance is known.
- Environment provider sync methods now accept `environment.SyncOptions` and return `environment.SyncResult`; future providers should honor `ExcludePatterns` and populate files/bytes/error stats for lifecycle hook payloads.
- Session-owned environment `ToolHost` is the execution path for extension Host API `environment/exec`; future provider implementations should ensure prepared tool hosts support terminal execution consistently.

## Open Risks
- Task 05 added the Daytona SSH non-PTY validation harness, but live validation is not complete because `DAYTONA_API_KEY` was unavailable in the execution environment. Task 06 must not assume SSH is approved until `DAYTONA_API_KEY=... go test -tags integration ./internal/environment/daytona -run TestDaytonaSSHNonPTYValidation -count=1 -v` passes and the validation report records observed latency/artifact evidence.
- Task 06 now has a local Daytona provider implementation with `make verify` passing, but live Daytona E2E cases still skipped without `DAYTONA_API_KEY`; do not mark task 06 complete or use it as a dependency until credentialed Daytona validation passes.

## Handoffs
- Task 02 consumed `internal/environment.ToolHost`, `Launcher`, and `LaunchSpec` via ACP aliases instead of redefining ACP-local equivalents.
- `environment.Prepared` now has `Launcher`, `Launch`, and `ToolHost`; Task 03 should populate all three for the local provider.
- Task 03 added `PrepareRequest.Permissions` as a string permission-mode seam so provider-created local `ToolHost` instances can preserve non-default ACP permission behavior during Task 04 session integration.
- Task 03 completed the local provider in `internal/environment/local`; local prepare is path-preserving, sync/destroy are no-ops, and local registry resolution uses `BackendLocal`/`DefaultBackend`.
- Task 04 completed session environment lifecycle integration: session manager now requires an injected environment registry, persists/restores `SessionEnvironmentMeta`, calls prepare/sync/destroy around ACP launch/stop, exposes environment data through session/API/CLI payloads, and daemon boot composes the local provider registry.
- Later Daytona tasks can rely on ACP `StartOpts` supporting per-start `Launcher` and `ToolHost` overrides returned by providers.
- Later remote providers should implement `environment.Finder` when they support label lookup; daemon restart cleanup depends on that optional interface for partial-create recovery.
