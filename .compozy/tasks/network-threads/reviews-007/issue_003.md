---
provider: coderabbit
pr: "105"
round: 7
round_created_at: 2026-05-06T03:44:15.991789Z
status: resolved
file: internal/daemon/network_e2e_assertions_test.go
line: 283
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_16_H,comment:PRRC_kwDOR5y4QM6-TzFU
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**The split-transcript negative case can pass for the wrong reason.**

This fixture never provides `surface`, `direct-id`, `causation-id`, or `trust` in any transcript message, and the audit rows also omit the expected routing fields. So the test still fails even if `validateNetworkCorrelationSurfaces()` incorrectly starts aggregating attributes across multiple messages. It does not isolate the split-match behavior its name claims to cover.

<details>
<summary>Suggested fixture shape</summary>

```diff
 		messages := []transcript.UIMessage{
 			{
 				Role: transcript.UIRoleAssistant,
 				Parts: []transcript.UIMessagePart{{
 					Type:  "text",
-					Text:  `<network-message id="msg_direct_01" kind="say"></network-message>`,
+					Text:  `<network-message id="msg_direct_01" kind="say" surface="direct" direct-id="direct_test_01"></network-message>`,
 					State: "done",
 				}},
 			},
 			{
 				Role: transcript.UIRoleAssistant,
 				Parts: []transcript.UIMessagePart{
 					{
 						Type:  "text",
-						Text:  `<network-message work-id="work_patch_42" reply-to="msg_say_01" trace-id="trace_ops_patch_42"></network-message>`,
+						Text:  `<network-message work-id="work_patch_42" reply-to="msg_say_01" trace-id="trace_ops_patch_42" causation-id="msg_say_01" trust="untrusted"></network-message>`,
 						State: "done",
 					},
 				},
 			},
 		}
 		audit := []store.NetworkAuditEntry{
-			{MessageID: "msg_direct_01", Direction: "sent", Kind: "say"},
-			{MessageID: "msg_direct_01", Direction: "delivered", Kind: "say"},
+			{MessageID: "msg_direct_01", Direction: "sent", Kind: "say", Surface: "direct", DirectID: "direct_test_01", WorkID: "work_patch_42"},
+			{MessageID: "msg_direct_01", Direction: "delivered", Kind: "say", Surface: "direct", DirectID: "direct_test_01", WorkID: "work_patch_42"},
 		}
```
</details>

 

As per coding guidelines, "Ensure tests verify behavior outcomes, not just function calls".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/daemon/network_e2e_assertions_test.go` around lines 247 - 283, The
negative test fixture doesn't provide routing attributes in any single
message/audit row so a buggy validateNetworkCorrelationSurfaces can pass by
aggregating across messages; update the test data (the messages slice, audit
slice, and the networkCorrelationExpectation) so at least one transcript
UIMessage contains the explicit attributes surface, direct-id, causation-id and
trust (e.g., add attributes to the first messages[0].Parts[0].Text or include
them as metadata in that message), and ensure the audit entries include the
corresponding routing fields for the same MessageID (e.g., add audit rows with
DirectID/WorkID/TraceID/CausationID or the appropriate fields matching
"msg_direct_01"), so the negative case truly verifies that
validateNetworkCorrelationSurfaces (and the networkCorrelationExpectation) does
not succeed by combining attributes from multiple messages.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - The current negative fixture still splits the expected routing/correlation attributes across multiple transcript messages and leaves the audit rows without the matching routing fields that `validateNetworkAuditEntry()` actually checks.
  - That means the test does not fully isolate the failure mode it names: a buggy implementation could aggregate attributes across messages or ignore the missing routing metadata in audit rows and this fixture would not prove otherwise.
  - The fix hardens the fixture so the first transcript message now carries `surface`, `direct-id`, `causation-id`, and `trust`, the second message carries the remaining correlation fields, and the audit rows carry the routed identifiers already supported by `NetworkAuditEntry`. The case still expects rejection because no single message contains the full correlated attribute set.
