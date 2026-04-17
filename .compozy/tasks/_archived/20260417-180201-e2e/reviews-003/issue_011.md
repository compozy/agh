---
status: resolved
file: internal/testutil/acpmock/cmd/acpmock-driver/main.go
line: 891
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57ziMh,comment:PRRC_kwDOR5y4QM645avT
---

# Issue 011: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Replace the fixed delivery delay with a real synchronization point.**

`pauseForDelivery` makes the harness timing-dependent instead of state-dependent. That is brittle for E2E tests: under load, 5ms may be too short for ordering guarantees and too long for larger streams.


As per coding guidelines, "Never use time.Sleep() in orchestration — use proper synchronization primitives".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/testutil/acpmock/cmd/acpmock-driver/main.go` around lines 882 - 891,
The pauseForDelivery function currently uses a fixed 5ms timer which makes tests
timing-dependent; replace this with a real synchronization point by waiting for
a delivery-complete signal instead of sleeping. Modify pauseForDelivery to
accept or access a delivery notification (e.g., a chan struct{} or sync.Cond)
from the driver harness and select on ctx.Done() and that notification channel
(or use a condition variable with a mutex), and ensure the code that completes a
delivery closes/signals that channel (or broadcasts the cond) so
pauseForDelivery reliably returns when delivery state is reached; update all
call sites to provide the notification channel if needed.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - The ACP Go SDK currently dispatches inbound messages concurrently in `Connection.receive()` by spawning `go c.handleInbound(&msg)` per frame. That means earlier `session/update` notifications and a later `session/request_permission` can legitimately race at the client even though wire order is preserved.
  - Removing `pauseForDelivery` caused the existing stable driver-sequence test to reorder permission events ahead of earlier assistant/tool updates, which is a real regression in the mock driver’s observable behavior.
  - A true non-timing fix would require ordered inbound handling or an explicit transport acknowledgement mechanism outside this batch’s code scope. The local sleep is therefore not safely replaceable here without changing lower-layer transport behavior.
