# TC-INT-003: MemoryProvider Selection and Status

**Priority:** P1
**Status:** Not Run

## Preconditions

- Multiple providers are available in the environment.
- Provider status endpoints are reachable.

## Steps

1. List providers.
2. Select the active provider.
3. Enable and disable one provider through the lifecycle endpoints.
4. Fetch provider detail after each mutation.

**Expected:** The daemon reports consistent MemoryProvider status, active selection, and lifecycle state across list, select, enable, disable, and detail reads.

## Required Evidence

- Provider list payload.
- Provider detail payloads after each step.
- Active provider selection evidence.
