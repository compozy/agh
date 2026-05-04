---
status: resolved
file: internal/mcp/auth/metadata.go
line: 19
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:6bac4643e49f
review_hash: 6bac4643e49f
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 015: Add a timeout to the default metadata client.
## Review Comment

Falling back to `http.DefaultClient` means discovery has no client timeout, so a slow or wedged issuer can hang login/refresh indefinitely unless every caller always supplies a deadline on `ctx`. Prefer a dedicated client with a sane timeout, or require callers to provide one explicitly.

As per coding guidelines, external-call hazards like blocking calls without timeouts on request threads must be fixed before release.

## Triage

- Decision: `VALID`
- Notes: `discoverMetadata` falls back to `http.DefaultClient`, which has no timeout. Callers may pass contexts, but the default client should still bound external OAuth discovery. Replace the fallback with a dedicated client with a sane timeout.
