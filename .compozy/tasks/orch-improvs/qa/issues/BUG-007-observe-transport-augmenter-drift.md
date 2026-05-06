# BUG-007: UDS Observe Parity Expected Stale Augmenter Sequence

**Severity:** Low  
**Priority:** P2  
**Type:** Functional  
**Status:** Fixed

## Environment

- **Build:** local dev build from task_32 QA execution
- **OS:** macOS, isolated AGH lab from `qa/bootstrap-manifest.json`
- **Browser:** not applicable
- **URL:** UDS observe transport parity E2E
- **Live provider/LLM:** acpmock fixture

## Summary

The UDS observe parity test timed out because it expected two prompt augmenter events while runtime truth now emits durable memory, skills, and situation augmenters.

## Behavioral Impact

- **Operator/User Goal:** Observe parity could not prove the full runtime prompt pipeline.
- **Agent Behavior:** The actual runtime emitted a richer and correct event sequence.
- **Business Outcome:** Stale tests could mask real parity regressions by timing out before assertion.
- **Cross-Surface State:** Test expectation drifted from HTTP/UDS observable event truth.

## Reproduction

```bash
go test -race -parallel=4 -count=1 -tags integration \
  -run '^TestUDSTransportObserveHarnessLifecycleParityMatchesHTTP$' ./internal/api/udsapi
```

Observed before the fix:

- The test timed out after observing three `harness.augmenter_applied` events because it expected only two.

## Expected

UDS observe parity should assert the current runtime event sequence, including durable memory, skills, and situation augmenters.

## Root Cause

`internal/api/udsapi/transport_parity_integration_test.go` still encoded the older two-augmenter lifecycle.

## Fix

The expected event type lists now include the third `harness.augmenter_applied` event.

## Verification

- `go test -race -parallel=4 -count=1 -tags integration -run 'TestUDSTransport(ApprovalFlowMatchesHTTP|ObserveHarnessLifecycleParityMatchesHTTP)' ./internal/api/udsapi`
- `make test-e2e-runtime`

## Impact

- **Users Affected:** Developers running UDS transport parity tests.
- **Frequency:** Always after the runtime started emitting the third augmenter.
- **Workaround:** None.

## Related

- Test Case: TC-REG-001

