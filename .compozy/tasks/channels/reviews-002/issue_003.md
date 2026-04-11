---
status: resolved
file: internal/channels/delivery_broker_test.go
line: 250
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56Tbvc,comment:PRRC_kwDOR5y4QM624BPK
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Wait for ack state, not just recorded calls.**

`waitForCalls(t, transport, 2)` only proves the fake transport appended two requests. In `fakeDeliveryTransport.DeliverChannel`, that append happens before the handler returns its `DeliveryAck`, so `Snapshot()` can still observe `LastAckedSeq == 1` here and make this test flaky. Gate the assertion on the snapshot fields you expect, or add an explicit ack notification path to the fake transport.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/channels/delivery_broker_test.go` around lines 234 - 250, The test
currently waits only for transport call count via waitForCalls(t, transport, 2)
which races with fakeDeliveryTransport.DeliverChannel appending requests before
the DeliveryAck is applied; change the test to wait for the actual ack state
before asserting snapshot fields: either add an explicit ack-notify channel on
fakeDeliveryTransport that is closed/sent to when the handler applies the
DeliveryAck and use that in the test, or poll broker.Snapshot(ctx,
reg.DeliveryID) until snapshot.LastAckedSeq == 2 (with a short timeout) before
asserting RemoteMessageID and ReplaceRemoteMessageID; update uses of
waitForCalls, fakeDeliveryTransport.DeliverChannel, and broker.Snapshot
accordingly.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `Valid`
- Notes:
  The root cause is real: `fakeDeliveryTransport.DeliverChannel` records the call before the handler returns, while broker ack state is only updated after the handler returns. Waiting only on recorded calls can race `Snapshot()` and make `LastAckedSeq` assertions flaky. The fix is to add explicit transport state-change signaling and wait for ack completion before snapshot assertions.
  Resolved in `internal/channels/delivery_broker_test.go` by adding transport notifications, ack counting, and `waitForAcks(...)`, then verified with `go test ./internal/channels -count=1` and `make verify`.
