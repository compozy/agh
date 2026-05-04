# TC-FUNC-046 — Hosted MCP `mcp.Tool` schemas use exact descriptor bytes via `RawInputSchema`/`RawOutputSchema`

- **Priority:** P1
- **Type:** Functional / hosted MCP
- **Trace:** Task 10, ADR-011, Safety Invariant 22

## Objective

Prove hosted MCP tool registration constructs `mcp.Tool` with `RawInputSchema` and `RawOutputSchema` taken byte-for-byte from `Descriptor.input_schema`/`Descriptor.output_schema`. Reflection helpers (`WithInputSchema`, `WithOutputSchema`) are forbidden for hosted tools.

## Test Steps

1. Code grep across `internal/mcp` proxy code: no occurrence of `WithInputSchema`/`WithOutputSchema` in hosted-tool registration code path.
2. Hosted MCP `tools/list` returns the byte-identical schema as the descriptor.
   - **Expected:** Byte-stable; `tools/list` JSON identical to descriptor JSON.

## Automation

- **Target:** Unit
- **Status:** Existing
- **Command/Spec:** `go test ./internal/mcp -run TestHostedSchemaByteStable`
