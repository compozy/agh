# BUG-006: Transport Approval Fixture Auto-Rejected Permission Requests

**Severity:** Medium  
**Priority:** P1  
**Type:** Functional  
**Status:** Fixed

## Environment

- **Build:** local dev build from task_32 QA execution
- **OS:** macOS, isolated AGH lab from `qa/bootstrap-manifest.json`
- **Browser:** not applicable
- **URL:** HTTP/UDS approval transport parity tests
- **Live provider/LLM:** acpmock fixture

## Summary

HTTP/UDS approval parity tests could not observe a pending permission because the shared fixture auto-rejected writes.

## Behavioral Impact

- **Operator/User Goal:** Approval parity between HTTP and UDS could not be validated.
- **Agent Behavior:** The agent emitted a rejected permission event instead of an interactive pending request.
- **Business Outcome:** Transport parity coverage falsely failed before reaching the approval boundary.
- **Cross-Surface State:** HTTP/UDS test expectations did not match fixture policy.

## Reproduction

```bash
go test -race -parallel=4 -count=1 -tags integration \
  -run '^TestHTTPTransportApprovalFlowUsesSharedRuntimeHarness$' ./internal/api/httpapi
go test -race -parallel=4 -count=1 -tags integration \
  -run '^TestUDSTransportApprovalFlowMatchesHTTP$' ./internal/api/udsapi
```

Observed before the fix:

- `approvedRequestID` stayed empty because no pending permission was registered.

## Expected

The fixture must allow read operations automatically and require interactive approval for writes so parity tests can exercise the real approval path.

## Root Cause

`internal/testutil/acpmock/testdata/permission_env_fixture.json` set the approver fixture to `permissions: "deny-all"`.

## Fix

The fixture now uses `permissions: "approve-reads"`.

## Verification

- `go test ./internal/testutil/acpmock ./internal/testutil/acpmock/cmd/acpmock-driver -count=1`
- `go test -race -parallel=4 -count=1 -tags integration -run '^TestHTTPTransportApprovalFlowUsesSharedRuntimeHarness$' ./internal/api/httpapi`
- `go test -race -parallel=4 -count=1 -tags integration -run 'TestUDSTransport(ApprovalFlowMatchesHTTP|ObserveHarnessLifecycleParityMatchesHTTP)' ./internal/api/udsapi`
- `make test-e2e-runtime`

## Impact

- **Users Affected:** Transport parity QA.
- **Frequency:** Always in the affected fixture path.
- **Workaround:** None.

## Related

- Test Case: TC-INT-002

