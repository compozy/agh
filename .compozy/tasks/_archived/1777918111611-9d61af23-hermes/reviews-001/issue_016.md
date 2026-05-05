---
status: resolved
file: internal/mcp/auth/metadata.go
line: 68
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:87c7dcb9b227
review_hash: 87c7dcb9b227
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 016: Require HTTPS for OAuth discovery and credential endpoints.
## Review Comment

Both `resolveMetadataURL()` and `validateAbsoluteHTTPURL()` accept plaintext HTTP for critical OAuth endpoints, enabling credential interception during metadata discovery, token exchange, and revocation flows. Apply HTTPS enforcement to:
- Line 68–88: `resolveMetadataURL()` — enforce `https://` scheme only
- Line 113–121: `validateAbsoluteHTTPURL()` — reject `http://` scheme

For local development, carve out an explicit loopback-only exception (e.g., `localhost` or `127.0.0.1`).

## Triage

- Decision: `VALID`
- Notes: OAuth metadata, authorization, token, and revocation URLs currently allow plaintext `http` for any host. That exposes credential-bearing flows to interception. Enforce `https` for non-loopback hosts while retaining an explicit loopback `http` exception for local development and tests.
