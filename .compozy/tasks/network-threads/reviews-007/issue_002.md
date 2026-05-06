---
provider: coderabbit
pr: "105"
round: 7
round_created_at: 2026-05-06T03:44:15.991789Z
status: resolved
file: internal/daemon/network_e2e_assertions_test.go
line: 37
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_16_F,comment:PRRC_kwDOR5y4QM6-TzFS
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Audit assertions still skip reply/trace/causation correlation keys.**

`internal/network/audit.go:214-230` and `internal/network/audit.go:260-279` persist `ReplyTo`, `TraceID`, and `CausationID`, but this helper never compares them. That means the new assertions only prove transcript emission for those keys; a regression in durable audit persistence would still pass here.

<details>
<summary>Suggested minimal follow-up</summary>

```diff
 type networkAuditExpectation struct {
 	MessageID string
 	Direction string
 	Kind      string
 	Surface   *string
 	ThreadID  *string
 	DirectID  *string
 	WorkID    *string
+	ReplyTo   *string
+	TraceID   *string
+	CausationID *string
 	PeerFrom  *string
 	PeerTo    *string
 	Reason    string
 }
```

```diff
 		if !optionalAuditFieldMatches(expectation.WorkID, entry.WorkID) {
 			continue
 		}
+		if !optionalAuditFieldMatches(expectation.ReplyTo, entry.ReplyTo) {
+			continue
+		}
+		if !optionalAuditFieldMatches(expectation.TraceID, entry.TraceID) {
+			continue
+		}
+		if !optionalAuditFieldMatches(expectation.CausationID, entry.CausationID) {
+			continue
+		}
```

```diff
 		if err := validateNetworkAuditEntry(audit, networkAuditExpectation{
 			MessageID: expectation.MessageID,
 			Direction: direction,
 			Kind:      expectation.Kind,
 			Surface:   auditFieldValue(expectation.Surface),
 			ThreadID:  auditFieldValue(expectation.ThreadID),
 			DirectID:  auditFieldValue(expectation.DirectID),
 			WorkID:    auditFieldValue(expectation.WorkID),
+			ReplyTo:   auditFieldValue(expectation.ReplyTo),
+			TraceID:   auditFieldValue(expectation.TraceID),
+			CausationID: auditFieldValue(expectation.CausationID),
 		}); err != nil {
```
</details>

 

As per coding guidelines, "Focus on critical paths: workflow execution, state management, error handling".


Also applies to: 84-92, 114-149

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/daemon/network_e2e_assertions_test.go` around lines 26 - 37, The
test helper networkAuditExpectation struct omits the ReplyTo, TraceID and
CausationID fields that the audit persistence code (functions that set
ReplyTo/TraceID/CausationID) writes, so extend the networkAuditExpectation type
to include pointer/string fields for ReplyTo, TraceID and CausationID and update
the assertion logic that compares actual persisted audit records (the helper
assertions referenced around the existing checks at 84-92 and 114-149) to
compare those new fields as well (treat nil vs empty appropriately), ensuring
the tests fail if durable audit persistence stops storing these correlation
keys.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `INVALID`
- Notes:
  - Fresh code inspection showed the review comment mapped the persistence target incorrectly. `internal/network/audit.go` writes `ReplyTo`, `TraceID`, and `CausationID` to `store.NetworkMessageEntry` / persisted conversation messages, not to `store.NetworkAuditEntry`.
  - `validateNetworkAuditEntry()` is intentionally limited to the fields that `NetworkAuditEntry` actually stores: message identity, direction, kind, container routing, peers, and rejection reason.
  - Extending `networkAuditExpectation` to assert `reply_to` / `trace_id` / `causation_id` would be a false test against fields that do not exist on the audited type. The correct persistence surface for those keys is the conversation-message timeline, not the audit snapshot helper covered by this issue.
