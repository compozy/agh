---
provider: coderabbit
pr: "88"
round: 1
round_created_at: 2026-05-02T18:22:40.559088Z
status: resolved
file: internal/api/contract/authored_context.go
line: 1062
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_IrdE,comment:PRRC_kwDOR5y4QM69XbzD
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Redact `LastError` before returning session health.**

Line 1060 copies daemon error text straight into a public status payload, and the detector below still does not block secret bindings like `secret_ref`/`env:`/`vault:` or OAuth/PKCE fields. A failed auth or secret-resolution path can therefore leak exactly the material the new authored-context APIs are supposed to redact.
 
As per coding guidelines, "Raw `claim_token` (`agh_claim_*`), MCP auth tokens, OAuth codes, PKCE verifiers, and secret bindings must NEVER appear in logs, status APIs, settings views, error payloads, channel messages, SSE, web UI, or memory; use hash forms (`claim_token_hash`) over the wire".


Also applies to: 1243-1274

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/contract/authored_context.go` around lines 1048 - 1062, The
SessionHealthPayload currently returns raw daemon error text via LastError (see
SessionHealthPayload return and the normalized.LastError usage); change this to
never expose raw error/secret material by sanitizing or redacting
normalized.LastError before returning (e.g., replace with nil, sanitized string,
or a claim_token_hash) and ensure the same sanitization is applied to the other
occurrences referenced (lines ~1243-1274). Update or call the project’s
secret-sanitizer utility (or add a sanitizeLastError function) to strip secret
bindings (secret_ref/env:/vault:, OAuth/PKCE codes, MCP/claim tokens) and return
only safe hashed/placeholder values in SessionHealthPayload.LastError and any
related status/error payloads.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: `SessionHealthPayloadFromDomain` copied `heartbeat.SessionHealth.LastError` directly into the public `SessionHealthPayload.LastError` field. That error string can originate from daemon/provider failure paths and may contain credential material. The authored-context redaction guard also omitted secret-reference keys and secret binding value prefixes (`env:`, `vault:`, `vault://`) plus OAuth/PKCE key names, so a payload containing those bindings could pass validation.
- Fix approach: sanitize `LastError` with the existing diagnostics redactor before returning the DTO, then strengthen the authored-context unsafe key/value detector to reject secret-reference/OAuth/PKCE binding fields. Add regression coverage for sanitized session health errors and the expanded detector.
- Resolution: Implemented the fix in `internal/api/contract/authored_context.go`, added regression coverage in `internal/api/contract/authored_context_test.go`, regenerated `openapi/agh.json` with `make codegen`, and verified with `go test ./internal/api/contract -count=1`, `go test -race ./internal/api/contract -count=1`, the AGH test-convention helper, `make codegen-check`, and `make verify`.
