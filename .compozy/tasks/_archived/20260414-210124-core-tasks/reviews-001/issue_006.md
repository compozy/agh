---
status: resolved
file: internal/api/core/tasks_internal_test.go
line: 205
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562anq,comment:PRRC_kwDOR5y4QM63mgQ0
---

# Issue 006: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Assert the expected failure, not just “some error”.**

These branches only check `err != nil`, so the tests still pass if a helper starts returning the wrong validation or decode error. Pin each case to the expected error type/message to keep the coverage meaningful. 

As per coding guidelines, `MUST have specific error assertions (ErrorContains, ErrorAs)`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/tasks_internal_test.go` around lines 168 - 205, The tests
currently only assert err != nil; change each assertion to verify the specific
expected error type or message (using ErrorContains/ErrorAs or errors.Is) for
the functions attachTaskRunSessionIDFromRequest, failTaskRunFromRequest,
validateTaskChannel, enqueueTaskRunFromRequest, requiredPathID,
handlers.parseTaskListQuery, parseTaskRunListQuery, and decodeOptionalJSON so
they fail if a different error is returned; replace generic nil checks with
targeted assertions that match the exact validation/decoding error text or
exported error value for each case.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the negative helper coverage only asserts `err != nil`, so the tests would still pass if those helpers started returning the wrong validation or decode error.
- Fix approach: assert the expected sentinel or error substring for each negative case so the tests only pass when the correct validation path fires.

## Resolution

- Strengthened the negative helper coverage to assert the task validation sentinel and the expected error substrings instead of only checking for a non-nil error.
- Verified in the final `make verify` run.
