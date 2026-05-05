# TC-INT-009 — Go SDK extension publishes executable read-only tool through registry

- **Priority:** P0
- **Type:** Integration / Go SDK
- **Trace:** Task 08, ADR-009

## Test Steps

1. Compile a Go subprocess extension that uses `aghsdk.Tool[...]` to define a read-only tool.
2. Install and enable in daemon.
3. Manifest/runtime reconciliation passes; digests match shared fixtures.
4. Invoke via CLI/HTTP/UDS/hosted MCP.
   - **Expected:** All surfaces return identical result.
5. External-package compilation: building a Go program that imports only `github.com/pedronauck/agh/sdk/go` works.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./sdk/go ./internal/extension -run TestGoSDKExecutableTool`
