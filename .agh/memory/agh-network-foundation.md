---
name: Agh Network Foundation
description: Verified AGH Network architecture milestones and integration seams completed on 2026-04-11.
type: project
---

# AGH Network Foundation

## Verified Milestones

- On 2026-04-11, session `Space` became first-class runtime metadata across create, resume, persisted `meta.json`, the global sessions index, daemon contracts, and CLI session surfaces.
- On 2026-04-11, AGH Network runtime bootstrapping was verified around a daemon-owned `internal/network.Manager` that joins and leaves session peers during session lifecycle events.
- On 2026-04-11, bundled `agh-network` skill injection was verified for session start and resume, but only when a session has a non-empty `Space`.

## Durable Architecture Rules

- `internal/session` must stay decoupled from `internal/network` through late-bound callbacks. The stable seams are `session.NetworkPeerLifecycle` for join and leave, plus `session.TurnEndNotifier` for delivery wakeups after turns finish.
- Network participation is opt-in. Sessions without `Space` do not join runtime spaces and do not receive bundled network guidance.
- Network diagnostics may expose listener, space, and peer status, but broker credentials and token material must remain runtime-only and must not appear in persisted status or info payloads.
- Session and CLI surfaces should continue to show `Space` anywhere operators need to inspect or target a network-capable session.

## Source Anchors

- `internal/session/manager.go`
- `internal/session/manager_helpers.go`
- `internal/session/manager_network_skill.go`
- `internal/network/manager.go`
- `internal/store/globaldb/global_db_session.go`
- `internal/cli/session.go`
