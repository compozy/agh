# TC-FUNC-044 — Go SDK schema digest matches TypeScript SDK and daemon fixtures

- **Priority:** P1
- **Type:** Functional / digest parity
- **Trace:** Task 08, ADR-008

## Test Steps

1. Run shared digest fixture sets across daemon, TypeScript SDK, and Go SDK.
   - **Expected:** Byte-for-byte parity.
2. Modify a fixture schema; rebuild — all three implementations report new digest, still in agreement.
3. Confirm Go SDK `go-tool-provider` template scaffolds and produces a buildable extension whose manifest schema digests pass daemon validation.

## Automation

- **Target:** Unit + Integration
- **Status:** Existing
- **Command/Spec:** `go test ./sdk/go -run TestDigestParity`
