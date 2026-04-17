---
status: resolved
file: internal/api/udsapi/transport_parity_integration_test.go
line: 131
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57wEbO,comment:PRRC_kwDOR5y4QM640qzf
---

# Issue 004: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Check the webhook run count before indexing `delivery.Runs[0]`.**

If webhook delivery returns zero runs, this test panics before it can report a useful transport-parity failure. Add an explicit `len(delivery.Runs) == 1` assertion before extracting `runID`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/udsapi/transport_parity_integration_test.go` around lines 118 -
131, The test assumes delivery.Runs has at least one element and will panic;
before extracting runID add an explicit assertion that len(delivery.Runs) == 1
(or at least > 0) and fail the test with a clear message if not. Locate the call
to runtimeHarness.DeliverGlobalWebhook (seedTransportWebhookTrigger -> delivery)
and insert a check on delivery.Runs (e.g., if len(delivery.Runs) != 1 {
t.Fatalf("expected 1 webhook run, got %d: %+v", len(delivery.Runs), delivery) })
before referencing delivery.Runs[0].ID.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: the UDS parity test indexes `delivery.Runs[0]` without first asserting that webhook delivery actually produced a run, so a regression would panic instead of failing with context.
- Fix plan: add an explicit run-count assertion before indexing the slice. While touching the transport parity lane, apply the same guard to the matching HTTP helper path.
- Resolution: added explicit webhook run-count assertions before indexing the returned runs in the UDS parity test and its matching HTTP transport sibling.
- Verification: `go test ./internal/api/httpapi ./internal/api/udsapi` passed. Historical note: the later blocker about a missing `driver/dist/index.js` was stale; the shipped mock driver is `internal/testutil/acpmock/cmd/acpmock-driver`.
