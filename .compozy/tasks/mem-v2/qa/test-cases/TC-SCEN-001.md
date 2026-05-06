# TC-SCEN-001: Controller-Backed Write Is Searchable

**Priority:** P0
**Status:** Not Run

## Preconditions

- AGH daemon is running with Memory v2 enabled.
- A clean workspace is available with CLI, HTTP, and UDS access.
- Operator can inspect memory_decisions and memory_events.

## Steps

1. Create a new memory entry through CLI using the controller-backed write path.
2. Search for that content through CLI without reindex.
3. Repeat the same write/search flow through UDS.
4. Repeat the same write/search flow through HTTP.

**Expected:** The new memory is searchable without reindex on CLI, UDS, and HTTP, and the controller emits memory_decisions and memory_events records tied to the same workspace_id.

## Required Evidence

- CLI command transcript and returned search hit.
- UDS request/response pair showing workspace_id.
- HTTP request/response pair showing workspace_id.
- Captured memory_decisions row or API payload.
- Captured memory_events row or API payload.
