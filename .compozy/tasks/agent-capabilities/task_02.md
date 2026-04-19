---
status: completed
title: Capability-Aware Runtime Join Plumbing
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 02: Capability-Aware Runtime Join Plumbing

## Overview

Propagate the loaded capability catalog from the runtime/session layer to the network join boundary so local peers are built with capability context from the start. This task closes the current gap where `session` only hands `sessionID`, `peerID`, and `channel` to the network manager, forcing the local peer path to fall back to `DefaultPeerCard(peerID)`.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and task_01 before starting (`_prd.md` is absent for this feature)
- REFERENCE TECHSPEC "Component Overview", "Data flow", and "Development Sequencing"
- DO NOT re-read agent files inside `internal/network` - pass normalized runtime data across the boundary
- PRESERVE existing join/leave lifecycle invariants, peer identity, and nil/no-channel no-op behavior
- TESTS REQUIRED - add concrete session and network regression coverage for the new boundary
- GREENFIELD: prefer a clean interface evolution over adapter shims that keep the old impoverished join contract alive
</critical>

<requirements>
- MUST evolve the `session.NetworkPeerLifecycle` boundary so the network join path receives capability-aware runtime input derived from task_01
- MUST keep local catalog parsing in `internal/config`; `internal/network` may consume normalized capability data but MUST NOT parse filesystem formats
- MUST preserve current `joinNetworkPeer` no-op behavior for nil sessions, blank channels, and missing lifecycle handlers
- MUST preserve `peer_id`, `session_id`, and channel identity semantics while adding capability context
- MUST keep leave behavior and shutdown semantics unchanged apart from using the richer local peer registration input
- SHOULD keep the new join payload narrow and runtime-owned rather than leaking config package internals unnecessarily across packages
</requirements>

## Subtasks
- [x] 2.1 Design the capability-aware join payload or interface addition consumed by `internal/session` and `internal/network`
- [x] 2.2 Update the session activation path to gather loaded capability data and send it through the late-bound network lifecycle
- [x] 2.3 Update the network manager join preparation path to consume the richer input instead of relying only on `DefaultPeerCard(peerID)`
- [x] 2.4 Preserve existing leave, stop, and resume behavior while accommodating the new join payload
- [x] 2.5 Add focused unit and integration regressions around the evolved session/network boundary

## Implementation Details

See TechSpec "Component Overview", "Data flow", and build order step 4. The important architectural rule is that `internal/session` owns the runtime context of an agent session and should hand `internal/network` the normalized projection input it needs; `internal/network` should not discover or infer capabilities by itself.

### Relevant Files
- `internal/session/interfaces.go` - current `NetworkPeerLifecycle` interface that only exposes `JoinChannel(ctx, sessionID, peerID, channel)`
- `internal/session/manager_helpers.go` - `joinNetworkPeer` currently derives only `peerID` and channel before calling the network lifecycle
- `internal/session/manager_start.go` - session startup path where loaded agent data is available before network join
- `internal/network/manager.go` - `prepareJoinLocalPeer` currently creates local peers from `DefaultPeerCard(request.peerID)`
- `internal/session/manager_test.go` - existing session manager coverage to extend around the network join seam

### Dependent Files
- `internal/network/manager_test.go` - join-path tests need updates once the network manager accepts capability-aware input
- `internal/session/manager_integration_test.go` - integration coverage should exercise session activation with the evolved lifecycle contract
- `internal/network/peer.go` - local peer registration continues to normalize the final peer card built from this richer input
- `internal/api/core/network.go` - later tasks depend on the enriched local peer card flowing through API payload conversion unchanged

### Related ADRs
- [ADR-001: Explicit Capability Catalogs](adrs/adr-001.md) - capability data must originate from the runtime catalog, not network inference
- [ADR-002: Dual Storage Modes Without Merge](adrs/adr-002.md) - the runtime boundary should not need to know local file-mode details after task_01
- [ADR-003: Soft Outcome-Oriented Capability Model](adrs/adr-003.md) - the runtime payload should carry the structured capability meaning needed for later projection

## Deliverables
- Evolved session-to-network join contract carrying capability-aware runtime data
- Session activation path that passes normalized capability context during network join
- Network join preparation updated to build local peers from provided runtime capability input instead of unconditional defaults
- Updated unit tests in `internal/session` and `internal/network` for the new join seam **(REQUIRED)**
- Integration tests covering capability-aware join behavior through real session activation paths **(REQUIRED)**
- Test coverage >=80% for touched session/network packages **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `joinNetworkPeer()` remains a no-op when the session is nil, the channel is blank, or no lifecycle is installed
  - [x] `joinNetworkPeer()` forwards the same `session_id`, `peer_id`, and `channel` as before while adding capability-aware runtime input
  - [x] The evolved join payload carries a deterministic empty capability projection when task_01 loaded no catalog, instead of leaving `PeerCard.Capabilities` nil
  - [x] `prepareJoinLocalPeer()` builds the local peer from supplied runtime capability input rather than always calling `DefaultPeerCard(peerID)` as the authoritative source
  - [x] Leave and stop paths still use only `session_id` and do not regress because of the join payload change
- Integration tests:
  - [x] Activating a session whose agent directory contains a capability catalog reaches the network join path once and registers the same `peer_id` with enriched peer-card input
  - [x] Activating a session with no capability catalog still joins successfully and keeps capability projection empty-but-valid
  - [x] Resume or restart flows do not double-register the local peer or lose capability context across a join/leave cycle
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- The runtime/network boundary carries capability-aware input without leaking filesystem parsing into `internal/network`
- Local peer registration no longer depends solely on `DefaultPeerCard(peerID)` when capability data exists
