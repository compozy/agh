---
provider: coderabbit
pr: "88"
round: 1
round_created_at: 2026-05-02T18:22:40.559088Z
status: pending
file: internal/api/httpapi/httpapi_integration_test.go
line: 913
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4215360648,nitpick_hash:e18502587d4c
review_hash: e18502587d4c
source_review_id: "4215360648"
source_review_submitted_at: "2026-05-02T18:22:08Z"
---

# Issue 010: Add an explicit non-leak assertion for webhook secret responses.
## Review Comment

This test now writes `webhook_secret_value` but does not verify the secret never comes back in API payloads; that’s the critical regression to guard here.

As per coding guidelines, "Raw `claim_token` (`agh_claim_*`), MCP auth tokens, OAuth codes, PKCE verifiers, and secret bindings must NEVER appear in logs, status APIs, settings views, error payloads, channel messages, SSE, web UI, or memory; use hash forms (`claim_token_hash`) over the wire".

---

## Triage

- Decision: `UNREVIEWED`
- Notes:
