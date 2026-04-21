# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Align network brief discovery, rich `whois` discovery, peer detail payloads, and daemon API contracts with the unified capability model from task_01/task_03.
- Replace API-visible raw capability ext blobs and string-only peer-card capability lists with typed unified capability payloads while preserving the underlying network `greet`/`whois` split.

## Important Decisions

- Treat the current task spec, techspec, and ADRs as the approved design source for this run; no separate design artifact is needed before implementation.
- Keep network protocol boundaries unchanged: `greet` remains brief, `whois` rich discovery remains explicit, and transferred artifacts stay `kind:"capability"`.
- Make the daemon API expose typed capability payloads rather than leaking `agh.capabilities_brief` / `agh.capability_catalog` through raw `ext`.
- Extend peer-state/runtime snapshots to retain rich capability catalogs where available so peer detail payloads can stay coherent with brief discovery.
- Preserve brief discovery summaries even when the cached rich `whois` catalog is filtered by capability id by merging filtered rich data over `greet` summaries instead of replacing them.

## Learnings

- `internal/network/capability_catalog.go` still drops unified fields (`version`, `digest`, `requirements`) from rich discovery projections and from `cloneNetworkPeerCapabilityCatalog`, so current rich discovery is not actually unified yet.
- `internal/api/contract` / `internal/api/core` still surface peer-card capabilities as `[]string` plus raw `ext`, and the generated OpenAPI/types reflect that split-model shape.
- Targeted baseline tests pass; the pre-change gap is a contract-shape mismatch rather than an already red test.
- The peer registry can safely cache remote rich capability catalogs as long as it invalidates them when the advertised brief capability id sequence changes.
- Contract changes propagated into CLI/e2e/frontend fixtures; `make verify` exposed stale consumers that still expected string-only capability lists.

## Files / Surfaces

- `internal/network/capability_brief.go`
- `internal/network/capability_catalog.go`
- `internal/network/peer.go`
- `internal/network/router.go`
- `internal/network/manager_test.go`
- `internal/network/router_test.go`
- `internal/network/router_integration_test.go`
- `internal/network/capability_catalog_test.go`
- `internal/api/contract/contract.go`
- `internal/api/contract/contract_test.go`
- `internal/api/core/network.go`
- `internal/api/core/network_details.go`
- `internal/api/core/network_test.go`
- `internal/api/udsapi/network_test.go`
- `internal/api/spec/spec_test.go`
- `internal/cli/network_test.go`
- `internal/cli/network_client_test.go`
- `internal/testutil/e2e/runtime_harness_helpers_test.go`
- `web/src/generated/agh-openapi.d.ts`
- `web/src/systems/network/components/network-peer-detail-panel.tsx`
- `web/src/systems/network/mocks/fixtures.ts`
- `openapi/agh.json`

## Errors / Corrections

- `make verify` initially failed on stale CLI/e2e/frontend test fixtures that still assumed string-only peer-card capabilities; those fixtures were updated to the typed brief payload.
- Self-review found that filtered rich `whois` catalogs could blank unrelated brief capability summaries in API peer cards; `internal/api/core/network.go` now merges filtered rich summaries over the `greet` brief summaries and has a regression test.

## Ready for Next Run

- Task 04 implementation is complete on this branch: runtime discovery, peer detail payloads, daemon/API contracts, OpenAPI types, and supporting CLI/frontend fixtures all use the unified capability model.
- Verification evidence: `go test ./internal/api/core ./internal/cli ./internal/network ./internal/api/contract ./internal/api/udsapi`, `go test -cover ./internal/network ./internal/api/core ./internal/api/contract ./internal/api/udsapi`, and `make verify` all passed after the final filtered-catalog fix.
