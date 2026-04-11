# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement task 03 inside `internal/network`: local peer membership by session, remote peer cache by `peer_id` scoped to space, greet/leave/whois handling, heartbeat freshness/expiry, sender-side directed-send preflight, and message routing with dedup/lifecycle-aware rejection.
- Required evidence: unit tests for presence/routing rules, integration tests for greet/discovery/heartbeat/direct+broadcast flow, touched-package coverage >=80%, and clean `make verify`.

## Important Decisions
- Source of truth is `_techspec.md` plus RFC 003 subject/presence sections. Keep manager boot wiring and delivery worker orchestration out of scope for this task.
- Reuse the existing `Envelope`, `ValidateOptions`, `RouteToken`, transport helpers, lifecycle helpers, and audit interfaces instead of introducing parallel protocol types.
- Keep presence and routing foundations concentrated in `internal/network` via a `PeerRegistry` plus `Router`, so later manager, delivery, and API tasks compose the same runtime surfaces instead of rebuilding protocol behavior.

## Learnings
- `internal/network` already has protocol validation, lifecycle state handling, transport, and audit foundations from tasks 01-02; `peer.go` and `router.go` are the missing task-03 surfaces.
- Workspace already has unrelated tracking-file changes in `.compozy/tasks/agh-network`; those files must be left untouched unless task tracking is updated at closeout.
- The task-level presence policy is now enforced locally: remote peers expire at `last_seen + 2*greet interval`, direct sends fail before publish when the target is absent or stale, and fresh greet traffic repopulates presence without manager involvement.
- Replay handling is split between sender and receiver responsibilities: outbound routing chooses broadcast vs directed transport subjects, while inbound routing deduplicates by route token and generates deterministic receipts for duplicate or terminal direct deliveries.

## Files / Surfaces
- `internal/network/envelope.go`
- `internal/network/validate.go`
- `internal/network/lifecycle.go`
- `internal/network/transport.go`
- Added `internal/network/peer.go`
- Added `internal/network/router.go`
- Added `internal/network/peer_test.go`
- Added `internal/network/router_test.go`
- Added `internal/network/router_integration_test.go`

## Errors / Corrections
- `make verify` exposed a test-helper leak into production code; `router.go` now uses a local `ptrString` helper instead of the test-only helper name.

## Ready for Next Run
- Implementation and verification are complete. Keep tracking/memory artifacts unstaged unless a later workflow explicitly requires them in version control, and let follow-up tasks build on `PeerRegistry` and `Router` instead of reintroducing presence/routing logic elsewhere.
