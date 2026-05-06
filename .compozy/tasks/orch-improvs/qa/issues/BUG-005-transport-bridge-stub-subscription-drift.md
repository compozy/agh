# BUG-005: Transport Integration Bridge Stubs Missed Subscription Methods

**Severity:** Medium  
**Priority:** P2  
**Type:** Functional  
**Status:** Fixed

## Environment

- **Build:** local dev build from task_32 QA execution
- **OS:** macOS, isolated AGH lab from `qa/bootstrap-manifest.json`
- **Browser:** not applicable
- **URL:** HTTP/UDS transport integration tests
- **Live provider/LLM:** not required

## Summary

HTTP and UDS transport integration bridge service stubs no longer implemented the full bridge service interface after task bridge subscription methods were added.

## Behavioral Impact

- **Operator/User Goal:** Transport parity tests could not compile against the runtime contract.
- **Agent Behavior:** Agent-manageable bridge notification paths lacked reliable integration coverage.
- **Business Outcome:** Interface drift could hide transport regressions.
- **Cross-Surface State:** Test transport stubs diverged from production bridge service authority.

## Reproduction

```bash
go test -race -parallel=4 -count=1 -tags integration ./internal/api/httpapi ./internal/api/udsapi
```

Observed before the fix:

- Build failed because integration bridge services were missing `DeleteBridgeTaskSubscription` and related subscription methods.

## Expected

Transport integration stubs must implement the full `core.BridgeService` contract and delegate bridge task subscription operations to the same store abstraction used by runtime code.

## Root Cause

The test bridge service stubs embedded registry behavior but were not updated when `BridgeTaskSubscriptionStore` became part of the bridge service contract.

## Fix

`internal/api/httpapi/httpapi_integration_test.go` and `internal/api/udsapi/udsapi_integration_test.go` now carry a `taskSubscriptions bridgepkg.BridgeTaskSubscriptionStore` field and delegate the subscription CRUD methods.

## Verification

- `go test -race -parallel=4 -count=1 -tags integration -run '^TestHTTPTransportApprovalFlowUsesSharedRuntimeHarness$' ./internal/api/httpapi`
- `go test -race -parallel=4 -count=1 -tags integration -run 'TestUDSTransport(ApprovalFlowMatchesHTTP|ObserveHarnessLifecycleParityMatchesHTTP)' ./internal/api/udsapi`
- `make test-e2e-runtime`

## Impact

- **Users Affected:** Developers relying on transport parity tests.
- **Frequency:** Always at compile time once the interface changed.
- **Workaround:** None.

## Related

- Test Case: TC-REG-001

