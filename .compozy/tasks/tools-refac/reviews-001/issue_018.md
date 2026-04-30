---
provider: coderabbit
pr: "85"
round: 1
round_created_at: 2026-04-30T14:00:14.99254Z
status: resolved
file: internal/cli/client_tools.go
line: 284
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-ulKB,comment:PRRC_kwDOR5y4QM680KIg
---

# Issue 018: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Narrow the sensitive-key matcher to credential-shaped names.**

Matching any key that merely contains `token` will also redact benign fields like `completion_tokens`, `prompt_tokens`, or `token_count`. That drops useful usage/accounting data from otherwise safe tool responses.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/client_tools.go` around lines 268 - 284, The
sensitiveToolFieldName matcher is too broad (it treats any key containing
"token" as sensitive), so update the function to only redact credential-shaped
names: modify the markers list to remove the generic "token" and instead detect
token-like segments (e.g., keys that equal "token" or "access_token" or end with
"_token"/"-token"), and/or split the normalized key on non-alphanumeric
separators and check segments for exact marker matches (including plural
"tokens" where appropriate); adjust sensitiveToolFieldName (and its use of
normalized variable) to use this stricter segment-or-suffix matching so fields
like "completion_tokens" or "token_count" are not redacted.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `sensitiveToolFieldName` currently redacts any key containing `token`, which incorrectly hides benign usage fields like `completion_tokens`, `prompt_tokens`, or `token_count`. Narrow the matcher to credential-shaped token keys while preserving redaction for `token`, `access_token`, `refresh_token`, suffix-style token fields, auth, password, secret, API key, and PKCE fields.
