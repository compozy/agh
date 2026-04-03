---
status: resolved
file: internal/httpapi/server.go
line: 510
severity: high
author: claude-code
provider_ref:
---

# Issue 005: CORS wildcard with no auth allows unrestricted cross-origin access

## Review Comment

The CORS middleware at line 510 unconditionally sets `Access-Control-Allow-Origin: *` with no authentication or authorization middleware anywhere in the HTTP API. Any webpage in any browser can make requests to the daemon's HTTP API: create sessions, send prompts to agents, stop sessions, and read all session events. Since agents can execute arbitrary code (file operations, terminal commands), this is a significant security exposure.

Combined with the lack of rate limiting or request size limits, a malicious website could abuse the API while the user is browsing. The UDS server does not have this issue because it relies on filesystem permissions (chmod 0600).

**Suggested fix:** At minimum: (1) restrict allowed origins to `localhost` or the configured HTTP host, (2) add a configurable auth token, (3) warn on startup when binding to non-localhost addresses, (4) add request body size limits on POST endpoints.

## Triage

- Decision: `valid`
- Notes: The HTTP API currently returns `Access-Control-Allow-Origin: *` for every request and does not require authentication. Because the daemon binds to localhost by default, that still exposes the API to arbitrary browser origins targeting localhost. CORS needs to be restricted to safe origins instead of allowing all sites.
