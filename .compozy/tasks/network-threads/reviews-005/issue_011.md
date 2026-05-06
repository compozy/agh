---
provider: coderabbit
pr: "105"
round: 5
round_created_at: 2026-05-06T02:28:33.373448Z
status: resolved
file: internal/network/manager.go
line: 1092
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_033z,comment:PRRC_kwDOR5y4QM6-SXvc
---

# Issue 011: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Avoid double-counting durable sent/received messages.**

For durable envelopes this path now does both `record*Observed(...)` and `recordAudit* (...)`, but `recordAuditSent` / `recordAuditReceived` already increment the same counters and emit the same `"network.message.sent"` / `"network.message.received"` log lines. That will inflate `MessagesSent`/`MessagesReceived` and duplicate logs for every persisted conversation message.

 


Also applies to: 1351-1393

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/network/manager.go` around lines 1080 - 1092, The durable delivery
path is double-counting and double-logging because recordSentDelivery and
recordReceivedDelivery call both record*Observed(...) and recordAudit*(...),
while recordAuditSent/recordAuditReceived already increment the same counters
and emit the same logs; update recordSentDelivery and recordReceivedDelivery to
call recordAuditSent/recordAuditReceived only when durable is false (i.e., skip
the audit call for durable deliveries) or alternatively ensure recordAudit*
no-ops when the observed path has already incremented — adjust the logic in
recordSentDelivery, recordReceivedDelivery and mirror the same change in the
other affected block (around the 1351-1393 region) referencing the same
functions so counters/logs are not duplicated.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - Durable deliveries currently call `recordSentObserved` / `recordReceivedObserved` and then always call `recordAuditSent` / `recordAuditReceived`.
  - `recordAuditSent` and `recordAuditReceived` already increment `MessagesSent` / `MessagesReceived` and emit the same `network.message.sent` / `network.message.received` logs, so durable deliveries are double-counted and double-logged.
  - Fix plan: make durable delivery recording choose one accounting path instead of both, and add regression coverage. The narrowest regression test lives in `internal/network/manager_test.go`, which is outside the listed code files; if touched, it will be limited to the minimal assertions needed for this bug.

## Resolution

- Split audit persistence from counter/log emission in `internal/network/manager.go` so durable deliveries still write audit rows without double-counting sent/received metrics.
- Added the minimal regression assertion in out-of-scope `internal/network/manager_test.go` to prove durable sent/received counts stay at `1` each. This file was touched only because the scoped production bug needed direct regression coverage.
- Verified with fresh full `make verify` (passed).
