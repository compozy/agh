---
provider: coderabbit
pr: "85"
round: 1
round_created_at: 2026-04-30T14:00:14.99254Z
status: resolved
file: internal/api/core/tools.go
line: 445
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-ulJi,comment:PRRC_kwDOR5y4QM680KH2
---

# Issue 008: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Do not forward raw backend error text directly to API clients.**

`payload.Message` is populated from `toolErr.Error()` / `err.Error()`. This creates a leakage path for sensitive values if lower layers include secrets in wrapped errors.



<details>
<summary>🔐 Safer pattern</summary>

```diff
 	switch {
 	case errors.As(err, &toolErr):
 		payload.Code = toolErr.Code
-		payload.Message = toolErr.Error()
+		payload.Message = http.StatusText(status)
 		payload.ToolID = toolErr.ToolID
 		payload.ReasonCodes = append([]toolspkg.ReasonCode(nil), toolErr.ReasonCodes...)
 		payload.Layer = toolErrorLayer(toolErr.ReasonCodes)
 	case err != nil:
 		payload.Code = toolErrorCodeForStatus(status)
-		payload.Message = err.Error()
+		payload.Message = http.StatusText(status)
 		if reason, ok := toolspkg.ReasonOf(err); ok {
 			payload.ReasonCodes = []toolspkg.ReasonCode{reason}
 			payload.Layer = toolErrorLayer(payload.ReasonCodes)
 		}
```
</details>

As per coding guidelines, "Raw `claim_token` (`agh_claim_*`), MCP auth tokens, OAuth codes, PKCE verifiers, and secret bindings must NEVER appear in logs, status APIs, settings views, error payloads, channel messages, SSE, web UI, or memory; use hash forms (`claim_token_hash`) over the wire."

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/tools.go` around lines 421 - 445, The code currently
assigns payload.Message directly from toolErr.Error() and err.Error(), which may
leak secrets; replace those direct assignments in the error handling branch
(cases using toolErr, the err != nil branch, and the final payload.Message
defaults) with a sanitization step that returns a non-sensitive message — e.g.,
map known toolErr.Code and toolspkg.ReasonOf(err) to a safe canonical message or
call a new helper like sanitizeErrorMessage(err, toolErr.Code,
payload.ReasonCodes) that strips secret tokens (or returns
http.StatusText(status) / a code-based message) and only includes hashed
identifiers if needed; keep returning ReasonCodes and Layer (via toolErrorLayer)
but never expose raw err.Error() in payload.Message, and ensure
MaskInternalErrors still forces generic messages for 5xx responses.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `respondToolError` currently sends `toolErr.Error()` and `err.Error()` to clients. Lower layers may wrap sensitive input, approval tokens, MCP auth details, or claim tokens in those errors. Fix by mapping tool errors to safe status/code messages while preserving code, tool id, reason codes, and layer; keep `MaskInternalErrors` behavior for 5xx.
