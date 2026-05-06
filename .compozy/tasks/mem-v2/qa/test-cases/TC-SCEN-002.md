# TC-SCEN-002: Provider Lifecycle Stays Route-Scoped

**Priority:** P1
**Status:** Not Run

## Preconditions

- At least one configurable MemoryProvider is registered.
- Operator can call CLI, HTTP, and UDS lifecycle endpoints.

## Steps

1. Enable a provider through the CLI provider command.
2. Disable the same provider through the HTTP route that selects it by path.
3. Re-enable it through UDS and confirm the daemon reports the same provider name.

**Expected:** Provider lifecycle mutations always target the provider selected by the route path, and no request body field can redirect the action to a different MemoryProvider.

## Required Evidence

- CLI output for provider enable.
- HTTP enable/disable response bodies.
- UDS response body.
- Provider status snapshot before and after the sequence.
