# TC-REG-001: Broad Memory v2 Regression Sweep

**Priority:** P1
**Status:** Not Run

## Preconditions

- CLI, HTTP, UDS, native tool, and extension host paths are all reachable.
- One workspace has seeded memories and session history.

## Steps

1. Run write, show, search, reindex, reset, reload, provider, extractor, and dream flows.
2. Verify the same behavior through CLI, HTTP, and UDS.
3. Confirm generated CLI/API docs still describe the exercised routes and fields.

**Expected:** Memory v2 remains coherent across public surfaces, generated CLI/API output, config lifecycle, Knowledge, Memory Settings, Session Inspector, and operator-visible storage artifacts.

## Required Evidence

- Consolidated command/output transcript.
- Matching HTTP and UDS payloads.
- Screenshots or notes from Knowledge, Memory Settings, and Session Inspector.
- Links or snapshots from generated CLI/API docs.
