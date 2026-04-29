# TC-FUNC-040 — `tool.provider` capability negotiation requires `provide_tools` and `tools/call`

- **Priority:** P1
- **Type:** Functional / extension protocol
- **Trace:** Task 07, ADR-001, ADR-008

## Test Steps

1. Extension declares `capabilities.provides = ["tool.provider"]` but does not implement `provide_tools`.
   - **Expected:** Initialize handshake fails; daemon refuses to enable `tool.provider`.
2. Extension implements `provide_tools` but not `tools/call`.
   - **Expected:** Same — both methods required.
3. Both implemented but mismatched runtime descriptors (digest mismatch, missing handler).
   - **Expected:** Tool remains operator-visible with `extension_runtime_mismatch`; session-hidden.
4. All match.
   - **Expected:** Tool callable via `Registry.Call`.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/extension -run TestToolProviderHandshake`
