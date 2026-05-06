# TC-SEC-001: Scope and Lifecycle Inputs Reject Ambiguity

**Priority:** P1
**Status:** Not Run

## Preconditions

- HTTP and UDS endpoints are reachable.
- Operator can craft raw requests.

## Steps

1. Exercise provider lifecycle endpoints with the route-selected provider.
2. Exercise memory write/search APIs with explicit workspace_id bindings.
3. Verify that no unsupported legacy selector field changes the targeted resource.

**Expected:** Scope binding stays explicit, lifecycle targets remain unambiguous, and requests rely on supported fields only.

## Required Evidence

- Raw request/response captures.
- Operator notes confirming the selected provider and workspace_id.
