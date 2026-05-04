---
status: resolved
file: internal/diagnostics/redact.go
line: 12
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:567e1e80fb92
review_hash: 567e1e80fb92
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 012: Quoted JSON secrets still bypass redaction.
## Review Comment

`secretPattern` only matches unquoted `key:value` / `key=value` shapes, so inputs like `"access_token":"abc"` or `"refresh_token":"abc"` remain intact unless the value also happens to match the bearer-token regex. Since this helper feeds persisted diagnostics, that can leak credentials into crash bundles and failure summaries.

## Triage

- Decision: `VALID`
- Notes: `secretPattern` only allows an unquoted secret key because it begins with a word boundary on the key name. JSON strings such as `"access_token":"abc"` therefore bypass the diagnostic redactor. Extend the pattern to preserve optional key quotes while redacting quoted JSON values.
