---
status: resolved
file: internal/api/core/automation_additional_test.go
line: 114
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM565Hy-,comment:PRRC_kwDOR5y4QM63qGaR
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Status-only subtests make this coverage too shallow.**

Every case here only checks `200/204`, so a handler can return the wrong payload or miss the endpoint-specific behavior and this table still passes. Please assert at least one response field or one stub side effect per route; if you keep the table, the subtest names should also follow the required `Should...` pattern.



As per coding guidelines, `**/*_test.go`: MUST use `t.Run("Should...")` pattern for ALL test cases; MUST test meaningful business logic, not trivial operations.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/automation_additional_test.go` around lines 87 - 114, The
table-driven subtests (using performRequest, fixture.Engine and t.Run) only
assert HTTP status codes, which is too shallow and the t.Run names don't follow
the "Should..." pattern; update each subtest to use t.Run("Should ...") and add
at least one meaningful assertion per route: for GETs assert a specific JSON
field/value or structure in resp.Body (e.g., job id/name for
"/automation/jobs/job-1", trigger fields for "/automation/triggers/trigger-1",
run entries for runs endpoints) and for DELETEs assert side-effects such as a
subsequent GET to the same resource returns 404 or that a list endpoint no
longer contains the deleted id (performRequest + resp.Body checks); keep using
the existing table but include an expected-check function or switch on
request.path to validate payloads/side effects rather than only status.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The test only checks HTTP status codes and uses non-compliant subtest labels, so a broken payload mapper or missing delete side effect would still pass. I will rename the subtests to `Should...`, make the automation stub stateful for deletes, and assert endpoint-specific payload fields or delete side effects.
  Resolution: Reworked the endpoint table to use `Should...` names, stateful delete behavior, wrapped-response assertions for job/trigger/run payloads, and follow-up 404 checks after deletes.
