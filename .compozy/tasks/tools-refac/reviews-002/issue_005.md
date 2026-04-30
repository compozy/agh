---
provider: coderabbit
pr: "85"
round: 2
round_created_at: 2026-04-30T19:49:37.693355Z
status: valid
file: internal/api/udsapi/hosted_mcp.go
line: 43
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-0Ilo,comment:PRRC_kwDOR5y4QM687orU
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Stop returning and logging raw hosted-MCP backend errors.**

These branches still surface `err` verbatim after the SSE path was sanitized. A wrapped bind/projection error can leak secret material through JSON error payloads or server logs.
 
As per coding guidelines, `Raw claim_token (agh_claim_*), MCP auth tokens, OAuth codes, PKCE verifiers, and secret bindings must NEVER appear in logs, status APIs, settings views, error payloads, channel messages, SSE, web UI, or memory`.


Also applies to: 58-61, 117-123, 157-160, 180-182

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/udsapi/hosted_mcp.go` around lines 40 - 43, The code is
returning/logging raw backend errors from hosted-MCP calls (e.g., the result of
h.HostedMCP.Bind) which can leak secrets; replace any direct use of err in
core.RespondError and logs with a sanitized/generic message or a wrapped error
that does NOT include the original err string. Concretely: for each call site
(e.g., h.HostedMCP.Bind) stop passing err into core.RespondError or logger;
instead call hostedMCPStatus(err) for mapping to an HTTP status, but provide a
fixed client-facing message like "hosted-mcp backend error" (or an error created
via errors.New) to RespondError, and if you must log, log only non-secret
context (no err.Error()) or log a redacted marker; apply this same pattern to
the other occurrences referenced (the other h.HostedMCP.* error branches at the
listed ranges). Ensure no raw err is serialized into the response or server
logs.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  The UDS hosted-MCP handlers map backend errors to status codes but still pass
  the raw backend error to `core.RespondError`, and the stream loop logs the raw
  projection error. Because hosted-MCP errors can wrap auth or bind material, the
  fix must preserve status mapping from the original error while returning and
  logging only fixed, non-secret messages.
