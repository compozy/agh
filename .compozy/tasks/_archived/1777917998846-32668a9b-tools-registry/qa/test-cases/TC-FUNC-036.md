# TC-FUNC-036 — MCP descriptor normalization preserves `output_schema`

- **Priority:** P1
- **Type:** Functional / MCP normalization
- **Trace:** Task 09, ADR-010, TechSpec MCP

## Objective

Prove `MCPToolDescriptor` preserves normalized `output_schema` for external discovery: raw schema bytes when surfaced directly by the library; otherwise one canonical JSON encoding of the decoded schema.

## Test Steps

1. Configure fake MCP server returning `tools/list` with both `inputSchema` and `outputSchema` byte-stable in raw form.
   - **Expected:** Daemon-internal `MCPToolDescriptor` carries those bytes verbatim; downstream digesting uses them.
2. Configure fake MCP server returning a decoded schema only (library exposes parsed object).
   - **Expected:** Daemon canonicalizes to a single JSON encoding and treats it as authoritative.
3. Confirm registry `Descriptor.input_schema` and `Descriptor.output_schema` reflect the normalized payload.

## Automation

- **Target:** Unit + Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/mcp -run TestMCPDescriptorNormalization`
