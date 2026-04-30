---
provider: coderabbit
pr: "85"
round: 1
round_created_at: 2026-04-30T14:00:14.99254Z
status: resolved
file: internal/cli/client_tools.go
line: 150
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-ulJ5,comment:PRRC_kwDOR5y4QM680KIX
---

# Issue 017: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Apply invoke-result redaction before returning the response.**

`InvokeTool` decodes the server payload and returns it as-is, so the new `sanitizeToolInvokeResponse` path never runs on successful tool calls. That means `preview`, `structured`, `content[*].data`, and metadata can still surface raw secrets even though this file already defines the redaction helpers for them.

<details>
<summary>Suggested fix</summary>

```diff
 func (c *unixSocketClient) InvokeTool(
 	ctx context.Context,
 	id string,
 	request ToolInvokeRequest,
 ) (ToolInvokeResponseRecord, error) {
@@
 	path := "/api/tools/" + url.PathEscape(strings.TrimSpace(id)) + "/invoke"
 	if err := c.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
 		return ToolInvokeResponseRecord{}, err
 	}
-	return response, nil
+	return sanitizeToolInvokeResponse(response), nil
 }
```
</details>


As per coding guidelines, `internal/**/*.go`: `Raw claim_token (agh_claim_*), MCP auth tokens, OAuth codes, PKCE verifiers, and secret bindings must NEVER appear ...`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/client_tools.go` around lines 133 - 150, InvokeTool currently
returns the decoded ToolInvokeResponseRecord without applying redaction; call
the existing sanitizeToolInvokeResponse helper on the response (e.g.,
sanitizeToolInvokeResponse(&response) or the appropriate signature) after doJSON
succeeds and before returning so preview, structured, content[*].data and
metadata are sanitized and secrets (agh_claim_*, MCP/OAuth/PKCE tokens, secret
bindings) are removed; update the return path in InvokeTool to run this
sanitization and return the sanitized response.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `InvokeTool` decodes `ToolInvokeResponseRecord` and returns it without calling the existing `sanitizeToolInvokeResponse`. Successful tool responses can therefore leak sensitive values in preview, structured JSON, content data, or metadata. Apply the sanitizer before returning.
