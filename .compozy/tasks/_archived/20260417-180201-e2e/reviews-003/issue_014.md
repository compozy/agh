---
status: resolved
file: internal/testutil/acpmock/driver_test.go
line: 362
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57ziM1,comment:PRRC_kwDOR5y4QM645avv
---

# Issue 014: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Assert the canceled stop reason explicitly here.**

Right now this only proves that cancellation surfaces an error event. It does not prove the behavior named by the test, so a regression in `stop_reason` emission would still pass.


As per coding guidelines, "Ensure tests verify behavior outcomes, not just function calls".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/testutil/acpmock/driver_test.go` around lines 316 - 362, Update the
test TestDriverControlBlockUntilCancelReturnsCanceledStopReason to assert the
stop reason on the emitted error event explicitly: after collecting events with
collectPromptEvents/normalizeEvents, iterate the normalized events to find the
event with "type" == acp.EventTypeError and assert its "stop_reason" field
equals the canceled stop reason constant (e.g. acp.StopReasonCanceled); fail the
test with a clear message if no error event has that stop_reason. This ensures
the test checks the specific stop_reason behavior rather than only the presence
of an error event.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - The current ACP client behavior does not surface a `stop_reason` on the emitted `error` event when the caller cancels the prompt context. Cancellation aborts the request in `internal/acp/client.go` before a prompt response is observed, so the test only sees an error event.
  - Asserting a canceled `stop_reason` in this test would therefore require a product-behavior change outside the scoped review file, not just a tighter assertion in `driver_test.go`.
  - The real regression in this area is the async-driver-control lifetime bug from issue 010, which is addressed separately.
