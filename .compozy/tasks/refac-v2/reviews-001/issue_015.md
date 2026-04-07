---
status: resolved
file: internal/api/httpapi/server.go
line: 570
severity: medium
author: claude-code
provider_ref:
---

# Issue 015: CORS rejects cross-origin when server bound to wildcard

## Review Comment

When the server binds to `0.0.0.0` or `::`, `resolveAllowedOrigin` computes `boundHostname` as a wildcard. The `isWildcardHost` check returns true, so the third branch (`boundHostname != "" && !isWildcardHost(boundHostname) && originHost == boundHostname`) is always false. The only passing cases are exact-match with the request `Host` header or mutual loopback.

If the SPA is served from a different port during development (e.g., Vite dev server on port 5173), CORS will reject the request with no way to allow it. There is no configuration option to set an explicit CORS allow-list.

**Fix:** When `isWildcardHost(boundHostname)` is true, either allow any origin where `originHost` matches the request host (ignoring port), or add an explicit CORS allow-list configuration option for development scenarios.

## Triage

- Decision: `invalid`
- Analysis: `resolveAllowedOrigin` already allows cross-port browser requests when the origin host matches the request host, and it also allows loopback-to-loopback development flows such as `localhost` to `127.0.0.1`.
- Analysis: Binding the server to `0.0.0.0` or `::` is not itself a reason to allow unrelated origins. The reported failing case depends on a different host name than the one the browser is actually contacting, which is outside the current security policy rather than a regression in wildcard binding.
- Conclusion: The current CORS logic matches the documented same-host/loopback policy, so no code change is justified in this batch.
