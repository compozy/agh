# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Compose a daemon-owned channel runtime that wires the channel registry, outbound target seam, Host API channel methods, delivery broker, and channel-capable extension launch resolver into daemon boot and shutdown without exposing new API/CLI surfaces yet.
- Add daemon/unit integration coverage for boot composition, bound-secret launch resolution, disable/stop behavior, restart route continuity, and clean runtime shutdown.

## Important Decisions
- Treat the accepted task spec, techspec, and ADRs as the approved design baseline for this implementation task; no separate brainstorming/design artifact is needed.
- Compose a daemon-owned `channelRuntime` in `internal/daemon/channels.go` that wraps `channels.Service` and `channels.Broker`, then inject that runtime into boot, session notifier wiring, the default extension-manager Host API construction, and daemon shutdown.
- Keep channel launch selection keyed by extension name for now and fail fast when multiple enabled channel instances map to the same extension, rather than guessing which launch payload to use.

## Learnings
- The real daemon boot path initially skipped channel runtime composition because `bootRuntime()` called `composeChannelRuntime()` before assigning `state.registry`; the new daemon integration tests exposed that ordering bug, and the fix was to publish `state.registry` / `state.workspaceResolver` before composing the runtime.
- The default extension-manager factory now injects `WithHostAPIChannelRegistry`, `WithHostAPIChannelDedupStore`, `WithHostAPIDeliveryBroker`, and `WithChannelRuntimeResolver`, so channel-capable adapters boot with daemon-owned routing, delivery, and bound-secret resolution without extra transport wiring.
- `internal/daemon` package unit coverage is now `80.0%`, and the full daemon integration suite plus repo-wide `make verify` passed after the composition fix.

## Files / Surfaces
- `internal/daemon/channels.go`
- `internal/daemon/boot.go`
- `internal/daemon/daemon.go`
- `internal/daemon/channels_test.go`
- `internal/daemon/daemon_test.go`
- `internal/daemon/daemon_integration_test.go`

## Errors / Corrections
- Corrected a real boot-ordering defect in `internal/daemon/boot.go`: the daemon now assigns `state.registry` before attempting channel runtime composition, otherwise the runtime is silently skipped on real boots.

## Ready for Next Run
- Task 07 is verified complete. Follow-on API/CLI/observability tasks can depend on `Daemon.channels` being a composed runtime that survives extension restarts inside one daemon lifetime and owns bound-secret launch resolution.
