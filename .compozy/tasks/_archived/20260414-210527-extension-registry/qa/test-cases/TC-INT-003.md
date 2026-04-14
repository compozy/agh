# TC-INT-003: ClawHub Retry on Transient Failures

| Field | Value |
|-------|-------|
| **Priority** | P1 (High) |
| **Type** | Integration |
| **Estimated Time** | 3 min |
| **Module** | `internal/registry/clawhub/client.go` |

## Objective

Validate that the ClawHub client retries on transient HTTP failures with exponential backoff.

## Preconditions

- Mock server that returns 500 on first 2 calls, then 200 on 3rd call.

## Test Steps

| Step | Action | Expected |
|------|--------|----------|
| 1 | Call `Search(ctx, "test", SearchOpts{})` against flaky server | **Expected:** Eventually succeeds after retries. Call count on server = 3. |
| 2 | Mock server returns 500 on all 3+ attempts | **Expected:** Fails after max retries with error mentioning retry exhaustion. |
| 3 | Verify backoff timing (1s, 2s, 4s pattern) | **Expected:** Delays between requests increase exponentially. |

## Edge Cases

- Context cancelled during backoff: should return context error immediately.
- HTTP 429 (rate limit): should also trigger retry with longer backoff.
