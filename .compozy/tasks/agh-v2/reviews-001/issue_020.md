---
status: resolved
file: internal/httpapi/stream.go
line: 321
severity: medium
author: claude-code
provider_ref:
---

# Issue 020: Error messages leak internal details to HTTP clients

## Review Comment

`respondError()` at line 321 passes `err.Error()` directly to the client JSON response. This can leak internal file paths, database errors, configuration details, and package-prefixed error messages. Combined with the wildcard CORS (issue_005), these details are exposed to any origin.

The same pattern exists in `internal/udsapi/handlers.go` at line ~844 (less concerning since UDS is local-only).

**Suggested fix:** For the HTTP API, sanitize error messages for 500-level responses (return generic "internal server error"). Only pass through user-facing messages for 400-level client errors. Consider a structured error type that separates internal details from client-facing messages.

## Triage

- Decision: `valid`
- Notes: The HTTP API currently reflects `err.Error()` directly to clients for all statuses, including 500-level failures. That leaks internal error strings and implementation details unnecessarily. Server-side failures should be sanitized before being returned over the public HTTP transport.
