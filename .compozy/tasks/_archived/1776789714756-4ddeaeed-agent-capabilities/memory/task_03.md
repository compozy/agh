# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Project the task 02 capability join payload into both `peer_card.capabilities` and `peer_card.ext["agh.capabilities_brief"]` from one centralized helper, while keeping the no-catalog path empty-but-valid.

## Important Decisions
- Build the brief projection in `internal/network` near peer-card construction so greet publishing, peer listing, and API payload conversion all reuse the same peer-card shape.
- Normalize capability IDs once inside the projection helper and reuse that same ordered result for both the `Capabilities` slice and the brief ext entries.
- Keep `agh.capabilities_brief` as a raw JSON ext payload with only `id` and `summary`, and rely on the existing `PeerCard`/API clone paths instead of adding special-case copy logic.

## Learnings
- `internal/network/manager.go` currently only copies capability IDs from the join payload into `PeerCard.Capabilities`; no code path populates `agh.capabilities_brief` yet.
- Existing ext-clone behavior is already centralized in `clonePeerCard()`, `normalizeAndValidatePeerCard()`, and `core.NetworkPeerPayloadFromInfo()`, so task 03 should populate the brief ext key once and rely on those clone paths.
- A small amount of extra API handler coverage was needed to keep `internal/api/core` at the task target after adding the new network assertions; peer/channel detail error-path tests were enough to bring the package back to 80.0%.

## Files / Surfaces
- `.compozy/tasks/agent-capabilities/memory/MEMORY.md`
- `internal/network/peer.go`
- `internal/network/capability_brief.go`
- `internal/network/manager.go`
- `internal/network/validate.go`
- `internal/network/router.go`
- `internal/network/peer_test.go`
- `internal/network/manager_test.go`
- `internal/network/manager_integration_test.go`
- `internal/network/router_test.go`
- `internal/network/router_integration_test.go`
- `internal/api/core/network.go`
- `internal/api/core/network_details.go`
- `internal/api/core/network_test.go`

## Errors / Corrections
- Accidentally pointed `gofmt` at Markdown memory files during the first format pass; reran formatting on Go files only before testing.

## Ready for Next Run
- Task 03 is implemented and verified. Code evidence:
  - `go test ./internal/network -count=1`
  - `go test ./internal/api/core -count=1`
  - `go test -tags integration ./internal/network -run TestManagerJoinPublishesProjectedCapabilityBriefInInitialAndReconnectGreets -count=1`
  - `go test ./internal/network -count=1 -cover` -> `81.2%`
  - `go test ./internal/api/core -count=1 -cover` -> `80.0%`
  - `make verify`
  - post-commit `make verify`
- Local code commit created: `b3f31b93` (`feat: project brief peer capabilities`).
