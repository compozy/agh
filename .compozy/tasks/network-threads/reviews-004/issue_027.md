---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/network/manager.go
line: 1094
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_0DsO,comment:PRRC_kwDOR5y4QM6-RRZX
---

# Issue 027: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Don't drop audit records just because the conversation write succeeded.**

`durable=true` now bypasses `RecordSent` / `RecordReceived` completely. That works only if the conversation store and the audit sink are the same backend; with `WithManagerConversationStore(...)` they can be different, and this silently removes sent/received audit rows for every persisted conversation message.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/network/manager.go` around lines 1080 - 1094, The current
recordSentDelivery and recordReceivedDelivery skip audit recording when durable
is true; instead, ensure that when durable is true you still call the audit
methods in addition to the observed handlers. Modify recordSentDelivery to call
m.recordSentObserved(sessionID, envelope) when durable is true but do not return
— afterwards always call m.recordAuditSent(ctx, sessionID, envelope). Likewise
modify recordReceivedDelivery to call m.recordReceivedObserved(sessionID,
envelope) when durable is true and then always call m.recordAuditReceived(ctx,
sessionID, envelope); keep the ctx parameter for the audit calls.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: `recordSentDelivery` and `recordReceivedDelivery` currently return immediately after recording the durable observed side effect. When the durable conversation store and the audit sink are different implementations, that skips sent/received audit rows for successfully persisted conversation messages.
- Fix approach: keep the durable observed-path bookkeeping, but always follow it with the corresponding audit write so persisted conversation messages still produce audit records.
- Verification: fixed in scoped code and validated with fresh `make verify`.
