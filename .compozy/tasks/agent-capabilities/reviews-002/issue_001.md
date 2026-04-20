---
status: pending
file: internal/api/core/network_test.go
line: 1624
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58BM38,comment:PRRC_kwDOR5y4QM65LrXh
---

# Issue 001: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Strengthen these new error-path subtests.**

Both cases only assert the status code, so an unrelated 404/400 branch would still pass. Please decode the error payload and assert the expected message as well, and add `t.Parallel()` since these subtests are independent.

As per coding guidelines, "MUST have specific error assertions (ErrorContains, ErrorAs)" and "Add `t.Parallel()` for independent subtests in Go".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/network_test.go` around lines 1570 - 1624, Add t.Parallel()
at the start of each subtest ("Should return not found when a channel has no
presence or history" and "Should reject invalid channel message limits"), and
after performing the request decode the JSON error payload from resp.Body and
assert the error message contains the expected text (e.g., for the first case
assert error contains "not found" or "no presence or history" and for the second
assert it contains "invalid limit" or "bad request"); use the existing
performRequest result and the response body to unmarshal into the error shape
used by the handlers and use ErrorContains-style assertions rather than only
checking resp.Code so the tests validate the specific error path.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
