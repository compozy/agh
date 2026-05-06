# TC-INT-001: CLI and UDS Memory Surface Parity

**Priority:** P1
**Status:** Not Run

## Preconditions

- CLI is configured to talk to the daemon over UDS.
- A writable workspace is available.

## Steps

1. Write one workspace memory through CLI.
2. Show and search the same memory through CLI.
3. Validate that the underlying UDS responses expose the same scope and workspace_id.

**Expected:** CLI output matches the daemon UDS contract for write, show, and search, including scope, workspace_id, and content visibility.

## Required Evidence

- CLI output snapshots.
- UDS payload captures for write, show, and search.
- Recorded workspace_id values.
