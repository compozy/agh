# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implement the bundled local MemoryProvider plus provider registry mechanics for selection, collision rejection, and observability without introducing external-provider compatibility code.

## Important Decisions

- `internal/memory/provider/local` now implements the Hermes-style 10-hook `MemoryProvider` over a small contract-typed `Backend` interface, not over controller or recall internals directly.
- `internal/memory/provider/local/memstore` adapts `memory.Store` to the local provider backend. This keeps the provider package contract-clean while still exercising real Store seams for prompt snapshots, recall, controller decision application, and agent-scoped bindings.
- The provider registry lives in `internal/extension` because later Host API/config/daemon tasks need one shared registration and selection surface. It normalizes names, preserves active workspace selection, and rejects provider-name, provider-tool, and reserved built-in-tool collisions deterministically.
- Registry collision observability writes global `memory.provider.collision` event summaries without holding the registry lock during the event-writer call.
- `memory.provider.collision` is now an allowed global `store.EventSummary` type. Public transports still land later; this task only provides the registry and event seam.

## Learnings

- The strictest ADR-008 interpretation is achievable without blocking the bundled provider: keep `local.Provider` on contract DTOs plus a local `Backend`, and isolate concrete `memory.Store` adaptation in a subpackage.
- Full `go test -race ./internal/extension` can trip preexisting SQLite migration context-deadline failures under high package parallelism; the new registry tests pass under targeted `-race`, and full non-race extension tests pass.
- Provider package coverage is stable above the floor after adding fake-backend error-path tests: `local` 88.5%, `local/memstore` 96.7%.

## Files / Surfaces

- `internal/memory/provider/local/`
- `internal/memory/provider/local/memstore/`
- `internal/extension/memory_provider_registry.go`
- `internal/extension/memory_provider_registry_test.go`
- `internal/extension/host_api.go`
- `internal/store/types.go`
- `internal/store/store_helpers_test.go`

## Errors / Corrections

- Initial local provider implementation imported `internal/memory` directly. It passed tests but conflicted with the ADR-008 contract-boundary posture, so the provider was refactored onto a contract-typed backend plus a `memstore` adapter.
- Initial local provider coverage was 71.5%; fake-backend validation and adapter tests raised the new provider packages above the 80% target.
- `make lint` flagged `WriteRecord` as a large value parameter. The method must retain the value parameter to satisfy `memcontract.MemoryProvider`; the implementation now uses a narrow `nolint:gocritic` justification only on the interface method and passes pointers internally.
- `go test -race ./internal/extension` failed due unrelated SQLite migration context-deadline errors across existing Host API tests. Targeted registry race coverage passes, and full `make verify` passes.

## Ready for Next Run

- Task 07 passed focused tests, targeted race tests, coverage, `git diff --check`, `make lint`, and full `make verify` before tracking updates.
- Next loop iteration should execute `task_08` (Frozen Snapshot and Prompt Assembly).
