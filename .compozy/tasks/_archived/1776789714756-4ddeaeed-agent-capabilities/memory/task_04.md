# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement explicit rich `whois` discovery on envelope `ext`, triggered only by `agh.include=["capability_catalog"]`, with optional `agh.capability_ids` filtering, deterministic empty-catalog behavior, and an oversized-response guard.

## Important Decisions
- Keep `PeerCard` brief and store full local capability metadata in runtime/network-local state instead of `PeerCard.Ext`.
- Evolve the runtime-owned session-to-network capability projection so local `whois` responders can serve the full structured catalog without re-reading agent files inside `internal/network`.
- Treat unknown AGH `ext` keys on `whois` requests as ignorable; parse only the known rich-discovery keys.
- Guard rich `whois` response emission by measuring the serialized envelope against the 1 MiB protocol limit and fail before publish if it would exceed that limit.

## Learnings
- Current `internal/network` code has no handling for `agh.include`, `agh.capability_ids`, or `agh.capability_catalog`; `handleWhois` always returns a minimal body-only response today.
- The existing join seam from task 02 only preserves capability `id` and `summary`, so rich discovery requires carrying more runtime-owned capability fields through that seam.
- Directed `whois` integration coverage must discover the target peer first, because `Router.Send()` enforces presence preflight for directed messages before the request is published.

## Files / Surfaces
- `internal/session/interfaces.go`
- `internal/session/manager_integration_test.go`
- `internal/session/network_peer.go`
- `internal/network/capability_catalog.go`
- `internal/network/manager.go`
- `internal/network/peer.go`
- `internal/network/router.go`
- `internal/network/validate.go`
- `internal/network/validate_test.go`
- `internal/network/router_test.go`
- `internal/network/router_integration_test.go`
- `internal/network/manager_test.go`

## Errors / Corrections
- `make verify` initially failed on `funlen` for `handleWhois`; fixed by splitting request handling and reply construction into focused helpers before rerunning the full gate.

## Ready for Next Run
- Verification evidence:
  - `go test ./internal/session ./internal/network`
  - `go test -tags integration ./internal/network`
  - `go test -cover ./internal/network` => `81.6%`
  - `go test -cover ./internal/session` => `80.9%`
  - `make verify`
