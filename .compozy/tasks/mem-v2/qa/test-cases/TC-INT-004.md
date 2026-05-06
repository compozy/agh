# TC-INT-004: MemoryProvider Registry And Extension Host Lifecycle

**Priority:** P1
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 40 minutes
**Created:** 2026-05-05
**Last Updated:** 2026-05-05

## Objective

Verify the bundled local MemoryProvider is active, extension providers can register through the Host API, collision rejection is deterministic, and provider-first recall falls back only for absent or not-implemented providers.

## Preconditions

- [ ] Isolated daemon booted with bundled local provider.
- [ ] Fixture extension provider or test harness is available.
- [ ] Provider collision fixture declares a tool that conflicts with an `agh__memory_*` built-in.

## Test Steps

1. **Run provider/extension tests**
   - Input: `go test ./internal/memory/provider/... ./internal/extension ./internal/daemon -run "TestMemoryProvider|TestLocalProvider|TestHostAPI|TestBoot" -count=1`
   - **Expected:** Provider interface, registry, local adapter, Host API, and daemon boot tests pass.

2. **Verify active bundled provider**
   - Input: `agh memory provider list -o json` or API equivalent.
   - **Expected:** Bundled local provider is present and active.

3. **Register fixture provider**
   - Input: install/enable fixture extension provider.
   - **Expected:** Provider registration succeeds and active-provider selection is deterministic.

4. **Exercise provider-first recall**
   - Input: search/recall through Host API path with provider implementing recall.
   - **Expected:** Provider recall is used; fallback occurs only when provider is absent or returns `ErrNotImplemented`.

5. **Collision rejection**
   - Input: register a provider tool colliding with `agh__memory_search`.
   - **Expected:** Registration fails without changing active provider; `memory.provider.collision` event is emitted.

## Evidence To Capture

- Go test logs.
- Provider list/status JSON.
- Extension install/enable logs.
- Collision error and event row.

