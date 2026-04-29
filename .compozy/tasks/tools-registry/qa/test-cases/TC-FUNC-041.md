# TC-FUNC-041 — TypeScript SDK `extension.tool(...)` registration with digest parity

- **Priority:** P2
- **Type:** Functional / TypeScript SDK
- **Trace:** Task 07, ADR-008

## Test Steps

1. Define a tool via `extension.tool("search", { readOnly: true, inputSchema: z.object({ q: z.string() }) }, handler)`.
2. Build the extension package; confirm SDK emits manifest snippet matching `extension.toml` declarations and a runtime descriptor with matching digest.
3. Daemon validates JCS schema digest against shared fixtures (`sdk/typescript/test-fixtures/digest/cases.json` ↔ `internal/extension/testdata/digest/cases.json`).
   - **Expected:** Byte-stable parity.
4. Tampered manifest schema (rename property) → daemon flags `extension_runtime_mismatch`.

## Automation

- **Target:** Unit + Integration
- **Status:** Existing
- **Command/Spec:** `bun test sdk/typescript`; `go test ./internal/extension -run TestSchemaDigestParityTypescript`
