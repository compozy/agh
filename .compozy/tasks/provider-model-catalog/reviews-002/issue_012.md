---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/config/provider.go
line: 1163
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245938208,nitpick_hash:6ab9df4a196e
review_hash: 6ab9df4a196e
source_review_id: "4245938208"
source_review_submitted_at: "2026-05-07T16:46:43Z"
---

# Issue 012: Consider using explicit scheme constants for clarity.
## Review Comment

Using `string(MCPServerTransportHTTP)` as the HTTP scheme works because the constant happens to equal `"http"`, but this couples URL validation to MCP transport terminology. Consider using explicit `"http"` and `"https"` strings or defining separate URL scheme constants.

---

## Triage

- Decision: `invalid`
- Notes:
  - This is a readability suggestion, not a functional defect in the current branch.
  - `validateAbsoluteHTTPURL` intentionally accepts only `"http"` and `"https"`, and `string(MCPServerTransportHTTP)` currently resolves to the canonical `"http"` scheme.
  - Changing this would be cosmetic only, so it is out of scope for a review-remediation batch focused on real defects and required test hardening.
  - Resolved as invalid after branch inspection and full verification.
