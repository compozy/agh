---
provider: coderabbit
pr: "85"
round: 2
round_created_at: 2026-04-30T19:49:37.693355Z
status: valid
file: internal/cli/cli_integration_test.go
line: 1767
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4207230527,nitpick_hash:e9013c7e22f3
review_hash: e9013c7e22f3
source_review_id: "4207230527"
source_review_submitted_at: "2026-04-30T17:35:37Z"
---

# Issue 009: Make stale-lease stderr redaction checks consistent with other paths.
## Review Comment

These blocks only reject `agh_claim_`. They should also reject `"claim_token"` (and avoid printing raw stderr on failure) to enforce the same redaction contract.

As per coding guidelines, "Raw `claim_token` ... must NEVER appear in logs, ... error payloads ... or memory."

Also applies to: 1900-1905

## Triage

- Decision: `VALID`
- Notes:
  The stale-lease stderr checks reject only token-like values and print raw
  stderr on failure. They should also reject the `claim_token` field name and
  use sanitized assertion messages so a failure cannot echo secret-bearing
  stderr into CI logs.
