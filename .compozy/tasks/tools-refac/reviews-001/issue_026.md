---
provider: coderabbit
pr: "85"
round: 1
round_created_at: 2026-04-30T14:00:14.99254Z
status: resolved
file: internal/cli/tool_operator.go
line: 31
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-ulKq,comment:PRRC_kwDOR5y4QM680KJX
---

# Issue 026: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Expose `approval_token` on `tool invoke`.**

The request type supports `approval_token`, but this command never captures or forwards it. Approval-gated tools can be discovered and approvals can be minted, yet there is no way to complete the second invoke from the CLI.

<details>
<summary>Suggested wiring</summary>

```diff
 type toolInvokeFlags struct {
 	scope                toolScopeFlags
 	input                string
 	inputFile            string
 	toolCallID           string
 	turnID               string
 	correlationID        string
+	approvalToken        string
 	sensitiveInputFields []string
 }
@@
 				request := ToolInvokeRequest{
 					SessionID:            strings.TrimSpace(flags.scope.sessionID),
 					WorkspaceID:          strings.TrimSpace(flags.scope.workspaceID),
 					AgentName:            strings.TrimSpace(flags.scope.agentName),
 					ToolCallID:           strings.TrimSpace(flags.toolCallID),
 					TurnID:               strings.TrimSpace(flags.turnID),
 					CorrelationID:        strings.TrimSpace(flags.correlationID),
+					ApprovalToken:        strings.TrimSpace(flags.approvalToken),
 					Input:                input,
 					SensitiveInputFields: trimNonEmptyStrings(flags.sensitiveInputFields),
 				}
@@
 	cmd.Flags().StringVar(&flags.toolCallID, "tool-call-id", "", "Optional caller tool-call id")
 	cmd.Flags().StringVar(&flags.turnID, "turn-id", "", "Optional caller turn id")
 	cmd.Flags().StringVar(&flags.correlationID, "correlation-id", "", "Optional correlation id")
+	cmd.Flags().StringVar(&flags.approvalToken, "approval-token", "", "Single-use approval token for approval-gated tools")
```
</details>




Also applies to: 166-175, 185-191

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/tool_operator.go` around lines 23 - 31, The tool invoke flow
never captures or forwards an approval token; update the CLI to accept and pass
approval_token by adding an approvalToken string field to the toolInvokeFlags
struct, wire a corresponding CLI flag parser where toolInvokeFlags is populated
(the same location that sets input/inputFile/toolCallID/etc.), and include this
approvalToken value in the invoke request payload when calling the function that
sends the tool invocation (the code that constructs the request object for the
tool invoke API). Ensure the field name is approvalToken in toolInvokeFlags and
map it to the API request's approval_token property so approval-gated tools can
complete the second invoke from the CLI.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `contract.ToolInvokeRequest` already supports `approval_token`, and the core handler forwards it to `tools.CallRequest`, but `agh tool invoke` has no flag or request mapping for it. Approval-gated tools therefore cannot be completed from the CLI after an approval token is minted. The fix is to add `toolInvokeFlags.approvalToken`, bind `--approval-token`, trim it, forward it as `ToolInvokeRequest.ApprovalToken`, and cover the CLI path in integration tests.
- Resolution: Added `--approval-token`, forwarded the trimmed value to `ToolInvokeRequest.ApprovalToken`, and covered the CLI path in the integration test; verified with focused integration tests and `make verify`.
