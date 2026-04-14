---
status: resolved
file: internal/automation/dispatch_test.go
line: 267
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562an5,comment:PRRC_kwDOR5y4QM63mgRK
---

# Issue 013: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Assert the expected missing-task-service failure.**

This only checks `err != nil`, so the test still passes if `Dispatch` fails for some unrelated reason before it reaches the task-service availability branch. Please assert the expected sentinel or message with `errors.Is` / `ErrorContains`.

As per coding guidelines, `**/*_test.go`: MUST have specific error assertions (ErrorContains, ErrorAs).

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/dispatch_test.go` around lines 261 - 267, The test
currently only checks that dispatcher.Dispatch returned a non-nil error; change
it to assert the specific "missing task service" failure by using errors.Is or
ErrorContains instead of a nil check: after calling dispatcher.Dispatch(...)
replace the generic err != nil check with an assertion that errors.Is(err,
<expected sentinel error for missing task service>) or
testutil.ErrorContains(err, "<expected message substring>"), referencing the
Dispatch function call and DispatchRequest/DispatchKindManual/job variables so
the test fails if Dispatch errors for any other reason.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the missing-task-service regression test only checks that `Dispatch` returns some error, so it could pass for an unrelated failure earlier in the dispatch path.
- Fix approach: assert the specific missing task service error/message alongside the failed run status.

## Resolution

- Tightened the missing-task-service regression test to assert the specific error message in addition to the failed automation run status.
- Verified in the final `make verify` run.
