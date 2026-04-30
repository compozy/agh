---
provider: coderabbit
pr: "85"
round: 1
round_created_at: 2026-04-30T14:00:14.99254Z
status: resolved
file: internal/cli/client.go
line: 1054
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-ulKF,comment:PRRC_kwDOR5y4QM680KIl
---

# Issue 015: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Redact streamed hosted-MCP errors before returning them.**

`error` events are surfaced verbatim here, so any sensitive value included by the daemon bypasses the scrubbing already added in `readAPIError` and goes straight to the CLI. Please route these frames through the same redaction/parsing path before returning them. 

As per coding guidelines, "Raw `claim_token` (`agh_claim_*`), MCP auth tokens, OAuth codes, PKCE verifiers, and secret bindings must NEVER appear in logs, status APIs, settings views, error payloads, channel messages, SSE, web UI, or memory; use hash forms (`claim_token_hash`) over the wire".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/client.go` around lines 1051 - 1054, The SSE handler currently
returns raw error frames (errors.New(strings.TrimSpace(string(event.Data))))
which bypasses redaction; instead pass event.Data through the existing
readAPIError parser/scrubber and return the resulting error (e.g., call
readAPIError(event.Data) and return its error), falling back to a sanitized
generic error only if readAPIError itself fails—this ensures SSEEvent error
frames are redacted the same way as other API errors. Refer to SSEEvent,
event.Data, readAPIError, and the current errors.New usage when making the
change.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `StreamHostedMCPProjection` turns SSE `error` frames into `errors.New(raw data)`. That bypasses the CLI's API error redaction path and can expose sensitive values if the daemon or an intermediate layer emits them. Add a helper that parses hosted SSE error payloads and redacts the final message before returning it.
