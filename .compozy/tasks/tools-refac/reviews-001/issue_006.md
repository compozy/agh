---
provider: coderabbit
pr: "85"
round: 1
round_created_at: 2026-04-30T14:00:14.99254Z
status: resolved
file: internal/api/core/tools.go
line: 111
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-ulJg,comment:PRRC_kwDOR5y4QM680KH0
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Scope defaults are merged but not propagated into downstream request payloads.**

At Lines [106]-[108] and [153]-[155], `ApprovalRequest`/`CallRequest` use raw `req.*` values even though scope is normalized earlier (Lines [97]-[99], [146]-[148]). If scope comes from query/path defaults, downstream handlers can receive empty session/workspace/agent fields.



<details>
<summary>💡 Proposed fix</summary>

```diff
  grant, err := h.ToolApprovals.CreateToolApproval(c.Request.Context(), scope, toolspkg.ApprovalRequest{
  	ToolID:      id,
- 	SessionID:   req.SessionID,
- 	WorkspaceID: req.WorkspaceID,
- 	AgentName:   req.AgentName,
+ 	SessionID:   scope.SessionID,
+ 	WorkspaceID: scope.WorkspaceID,
+ 	AgentName:   scope.AgentName,
  	Input:       cloneRawMessage(req.Input),
  	InputDigest: req.InputDigest,
  })
...
  result, err := h.Tools.Call(c.Request.Context(), scope, toolspkg.CallRequest{
  	ToolID:               id,
  	ToolCallID:           req.ToolCallID,
  	TurnID:               req.TurnID,
- 	SessionID:            req.SessionID,
- 	WorkspaceID:          req.WorkspaceID,
- 	AgentName:            req.AgentName,
+ 	SessionID:            scope.SessionID,
+ 	WorkspaceID:          scope.WorkspaceID,
+ 	AgentName:            scope.AgentName,
  	CorrelationID:        req.CorrelationID,
  	Input:                cloneRawMessage(req.Input),
  	SensitiveInputFields: append([]string(nil), req.SensitiveInputFields...),
  	ApprovalToken:        req.ApprovalToken,
  })
```
</details>


Also applies to: 145-160

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/tools.go` around lines 96 - 111, The scope defaults
computed by operatorToolScope are merged into the local scope variable but the
downstream payloads (ApprovalRequest in ToolApprovals.CreateToolApproval and
CallRequest in the tool call path) still use raw
req.SessionID/req.WorkspaceID/req.AgentName; update those payload constructions
to use scope.SessionID, scope.WorkspaceID, and scope.AgentName (and keep Input
as cloneRawMessage(req.Input) and InputDigest as req.InputDigest) so the
normalized defaults are propagated to ToolApprovals.CreateToolApproval and the
CallRequest creation; ensure the same change is applied in both the approval
branch (where ApprovalRequest is built) and the call branch (where CallRequest
is built).
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `CreateToolApproval` and `InvokeTool` normalize request/default scope into `scope`, but downstream `ApprovalRequest` and `CallRequest` still use raw request fields. If scope is provided by query/defaults, the registry/approval layer receives empty identifiers. Propagate `scope.SessionID`, `scope.WorkspaceID`, and `scope.AgentName` into those payloads.
