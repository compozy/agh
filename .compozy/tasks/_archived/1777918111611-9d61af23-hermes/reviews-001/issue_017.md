---
status: resolved
file: internal/mcp/auth/metadata.go
line: 81
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:d97ac42307f9
review_hash: d97ac42307f9
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 017: Build the RFC 8414 well-known URL in the correct order.
## Review Comment

Per RFC 8414, for issuers with a path component, the metadata endpoint must be `https://host/.well-known/oauth-authorization-server/<issuer-path>`. The current code at line 85 appends the well-known path *after* the issuer path, resulting in `https://host/<issuer-path>/.well-known/oauth-authorization-server`, which will never resolve successfully.

## Triage

- Decision: `VALID`
- Notes: `resolveMetadataURL` appends `/.well-known/oauth-authorization-server` after the issuer path. For issuers with a path component, RFC 8414 discovery places the well-known prefix before the issuer path. Rebuild the URL as `/.well-known/oauth-authorization-server/<issuer-path>`.
