# TC-INT-007 — TypeScript extension publishes executable read-only tool through registry

- **Priority:** P0
- **Type:** Integration / extension-host
- **Trace:** Task 07, ADR-001, ADR-008

## Test Steps

1. Install fixture extension `ts_test_ext` with `tool.provider` capability and one read-only tool defined via `extension.tool(...)`.
2. Daemon completes initialize handshake; manifest/runtime descriptor reconciliation passes.
3. Operator view shows tool callable; session view shows tool when session lineage permits.
4. Invoke tool via CLI/HTTP/UDS/hosted MCP — all surfaces succeed with consistent result.
5. Schema digest mismatch (tampered manifest) → tool drops to operator-only with `extension_runtime_mismatch`.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/extension -run TestTSExtensionExecutableTool`; `bun test sdk/typescript`
