---
status: resolved
file: internal/api/core/handlers_test.go
line: 149
severity: medium
author: claude-code
provider_ref:
---

# Issue 016: Unsynchronized counter mutation in parallel SSE test

## Review Comment

`TestBaseHandlersStreamingAndObserveEndpoints` is marked `t.Parallel()` and uses `sessionCalls` and `observeCalls` counters (lines 149-150) that are mutated inside stub callbacks. These callbacks execute from the SSE polling goroutine while assertions read the counters after the test completes. With the `-race` flag and unlucky timing, this is a data race.

```go
sessionCalls := 0  // written from SSE poll goroutine
observeCalls := 0  // read from test goroutine for assertion
```

**Fix:** Use `atomic.Int32` for the counters:

```go
var sessionCalls atomic.Int32
var observeCalls atomic.Int32
// In stubs: sessionCalls.Add(1)
// In assertions: assert sessionCalls.Load() >= 1
```

Also affects: `createCalled` bool in `TestBaseHandlersSessionEndpoints` (line ~29) has the same pattern but is lower risk since the HTTP request is synchronous.

## Triage

- Decision: `valid`
- Root cause: The SSE endpoint test mutates shared counters from callback execution paths while the parent test runs in parallel. That is a real data race under `-race`, even if the test usually passes.
- Fix approach: Replace the shared counters/flags with atomic values and keep the assertions semantically equivalent.
- Resolution: Implemented with atomic counters and validated by the passing `-race` test suite in `make verify`.
