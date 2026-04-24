# TC-NET-002: Network Delivery Retry Backoff

**Priority:** P0
**Type:** Regression / Reliability
**Status:** Pass
**Created:** 2026-04-24

## Objective

Verify that a temporary `PromptNetwork` failure does not drop the inbound message and does not immediately spin a new worker in a retry loop.

## Preconditions

- `internal/network` delivery coordinator constructed with a fake prompter.
- First prompt attempt returns an error.
- Retry scheduler is instrumented by the test.

## Test Steps

1. Accept a directed delivery for an idle session.
   **Expected:** one prompt attempt is made.

2. Make the first prompt attempt fail.
   **Expected:** message is requeued at the front, retry attempt increments, and a retry is scheduled with the base backoff delay.

3. Before running the scheduled retry callback, inspect call count and queue depth.
   **Expected:** call count is still one and queue depth is one.

4. Run the scheduled retry callback.
   **Expected:** second prompt attempt receives the same network message and can complete normally.

5. Validate retry delay function.
   **Expected:** retry delays grow exponentially and cap at the configured maximum.

## Current Evidence

- `go test ./internal/network` passed on 2026-04-24.
- Covered by `TestDeliveryCoordinatorRetriesPromptFailuresAfterWorkerExit`.
- Covered by `TestDeliveryCoordinatorRetryDelayUsesExponentialCap`.
