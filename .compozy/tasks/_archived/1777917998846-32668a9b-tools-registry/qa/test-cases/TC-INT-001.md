# TC-INT-001 — Every executable path enters `internal/tools.Registry.Call`

- **Priority:** P0
- **Type:** Integration / boundary
- **Trace:** Task 04, ADR-003, Safety Invariant 1

## Objective

Prove CLI, HTTP, UDS, hosted MCP, and extension/MCP backend paths cannot bypass `Registry.Call`. Every public verb that executes a tool flows through dispatch.

## Test Steps

1. Source-level audit: HTTP/UDS handlers call `Registry.Call`; CLI calls UDS/HTTP client; hosted MCP `tools/call` re-enters `Registry.Call`.
2. End-to-end test: each surface invokes the same tool; capture telemetry events.
   - **Expected:** Each call produces a `tool.call_started` and a `tool.call_completed` (or `tool.call_failed`/`tool.call_denied`) event keyed by `correlation_id` showing dispatch entry.
3. Boundary check (`make boundaries`).

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/api/core ./internal/cli ./internal/mcp -run TestRegistryCallEntryPoint`
