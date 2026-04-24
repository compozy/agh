---
status: resolved
file: internal/store/globaldb/global_db_network_messages_test.go
line: 218
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58qIeT,comment:PRRC_kwDOR5y4QM66CAk2
---

# Issue 014: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Cover cursor tie-breaks with equal timestamps.**

This only exercises strictly increasing `Timestamp` values. If `ListNetworkMessages` compares cursors by timestamp alone, entries that share the same timestamp can still be skipped or duplicated while this test passes. Please add at least one same-timestamp pair with different `MessageID`s so the before/after cases validate the `(timestamp, message_id)` ordering contract.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/globaldb/global_db_network_messages_test.go` around lines 127
- 206, TestGlobalDBListNetworkMessagesSupportsMessageIDCursors currently only
uses strictly increasing Timestamp values; add at least one pair of entries that
share the exact same Timestamp but have different MessageID values (e.g.,
"msg-2a" and "msg-2b") and write them via globalDB.WriteNetworkMessage so the
test covers tie-break ordering; then update the subsequent ListNetworkMessages
queries and assertions (the before and after checks) to account for the
deterministic (timestamp, message_id) ordering expected by ListNetworkMessages
so the test verifies no skips/duplicates when timestamps are equal.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Reasoning: this is the same gap as issue 013 expressed against the before/after cursor cases. Without equal-timestamp fixtures, the test can pass even if tie-break ordering skips or duplicates rows.
- Fix plan: cover both issues with one test update that inserts same-timestamp messages and verifies the cursor windows remain stable.
- Resolution: the same equal-timestamp fixture update now verifies the before and after cursor windows do not skip or duplicate rows when timestamps match.
- Verification: `go test ./internal/store/globaldb` and `make verify`
