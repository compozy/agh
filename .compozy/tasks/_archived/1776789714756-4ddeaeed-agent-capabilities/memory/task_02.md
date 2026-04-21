# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implemented capability-aware runtime join plumbing so `internal/session` passes normalized capability context from task 01 into the network join boundary, with unit/integration coverage and no change to leave/no-op invariants.

## Important Decisions
- Evolved the late-bound join seam to `session.NetworkPeerJoin`, carrying `session_id`, `peer_id`, `channel`, and a runtime-owned capability projection instead of exposing config types in `internal/network`.
- Kept the runtime projection intentionally narrow: each capability carries only `id` and `summary`, which is enough for the downstream peer-card/discovery work planned in later tasks.
- Standardized the no-catalog case on a deterministic empty capability slice so join callers and network tests never depend on nil slice behavior.

## Learnings
- Task 01's `AgentDef.Capabilities` data was already available during `prepareSessionStartRuntime()`, so the session layer could project network-ready capability data without reparsing agent directories or teaching `internal/network` about config internals.
- `internal/network` still starts local peer registration from `DefaultPeerCard(peerID)`, but capability IDs supplied through the join payload are now the authoritative source for the resulting local peer card's `Capabilities` field.
- Resume coverage matters on this seam: it proved the richer payload survives a join/leave cycle without double registration or identity drift.

## Files / Surfaces
- `internal/session/interfaces.go`
- `internal/session/manager_helpers.go`
- `internal/session/manager_start.go`
- `internal/session/network_peer.go`
- `internal/session/manager_test.go`
- `internal/session/manager_hooks_test.go`
- `internal/session/manager_integration_test.go`
- `internal/network/manager.go`
- `internal/network/manager_test.go`
- `internal/daemon/daemon_test.go`
- `internal/daemon/daemon_integration_test.go`

## Errors / Corrections
- The first `make verify` rerun surfaced a long-line lint failure in `internal/session/network_peer.go`; the helper signature was wrapped and the full verification gate passed on the next run.

## Ready for Next Run
- Verification evidence:
  - `go test ./internal/session -count=1`
  - `go test ./internal/network -count=1`
  - `go test ./internal/daemon -count=1`
  - `go test -tags integration ./internal/session -run 'TestManagerIntegrationCapabilityAwareJoin' -count=1`
  - `go test -tags integration ./internal/daemon -run 'TestBootNetworkDeliversInboundMessagesThroughLateBoundLifecycle|TestBootNetworkShutdownTracksInterruptedInFlightDelivery' -count=1`
  - `go test ./internal/session -count=1 -cover` -> `81.2%`
  - `go test ./internal/network -count=1 -cover` -> `81.2%`
  - `make verify`
- Ready handoff: task 03 can treat `session.NetworkPeerJoin` as the stable ingress for brief capability projection into peer cards and API/network discovery.
