---
provider: coderabbit
pr: "85"
round: 1
round_created_at: 2026-04-30T14:00:14.99254Z
status: resolved
file: internal/cli/tool.go
line: 39
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-ulKv,comment:PRRC_kwDOR5y4QM680KJe
---

# Issue 023: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Trim the validated IDs before handing them to the proxy.**

The checks use `strings.TrimSpace`, but Lines 57-58 still forward the original `sessionID` and `bindNonce`. A value like `" sess-1 "` passes validation and then binds with the spaces included.


<details>
<summary>Suggested fix</summary>

```diff
 		RunE: func(cmd *cobra.Command, _ []string) error {
-			if strings.TrimSpace(sessionID) == "" {
+			sessionID = strings.TrimSpace(sessionID)
+			if sessionID == "" {
 				return mcppkg.ErrHostedSessionRequired
 			}
-			if strings.TrimSpace(bindNonce) == "" {
+			bindNonce = strings.TrimSpace(bindNonce)
+			if bindNonce == "" {
 				return mcppkg.ErrHostedNonceRequired
 			}
```
</details>


Also applies to: 56-58

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/tool.go` around lines 35 - 39, The validation trims sessionID
and bindNonce but the original variables (sessionID, bindNonce) are later
forwarded untrimmed to the proxy; update the code so you assign the trimmed
values back (or use new trimmed variables) and pass those trimmed values to the
proxy call that binds the hosted session (the call that currently forwards
sessionID and bindNonce), keeping the same error checks that return
mcppkg.ErrHostedSessionRequired and mcppkg.ErrHostedNonceRequired when blank.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `newToolMCPCommand` validates `sessionID` and `bindNonce` with `strings.TrimSpace(...)` but then passes the original untrimmed variables to `mcp.RunHostedProxy`. A whitespace-padded value can pass validation and reach the hosted-MCP bind request with embedded spaces. The fix is to normalize both variables once before validation and pass the trimmed values onward.
- Resolution: Normalized `sessionID` and `bindNonce` before validation and forwarding to `mcp.RunHostedProxy`; verified with `go test -race ./internal/cli -count=1` and `make verify`.
