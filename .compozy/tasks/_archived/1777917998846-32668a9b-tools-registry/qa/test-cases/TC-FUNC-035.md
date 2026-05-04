# TC-FUNC-035 — `MCPCallExecutor` lives inside `internal/mcp` (not `internal/tools`)

- **Priority:** P1
- **Type:** Functional / boundary
- **Trace:** Task 09, ADR-010

## Objective

Prove the executor implementation is owned by `internal/mcp` and the registry only depends on the executor interface, not on `internal/mcp/auth` directly.

## Test Steps

1. Boundary check: `make boundaries` confirms `internal/tools` does not import `internal/mcp/auth`.
2. Source-level grep proves `MCPCallExecutor` interface defined in `internal/tools/...` and implemented in `internal/mcp/...`.
3. Static check: `internal/tools` test suite uses an injected mock executor; never instantiates real auth client.

## Automation

- **Target:** Unit + boundaries
- **Status:** Existing
- **Command/Spec:** `make boundaries`; `go test ./internal/tools -run TestExecutorInterfaceBoundary`
