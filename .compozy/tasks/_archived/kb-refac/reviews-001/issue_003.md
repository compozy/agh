---
status: resolved
file: internal/api/httpapi/middleware.go
line: 83
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56LrWQ,comment:PRRC_kwDOR5y4QM62twbW
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Match the full origin, not just the hostname.**

`resolveAllowedOrigin()` currently accepts any origin that shares the same host name, even when the port or scheme differs. That turns CORS from “same origin” into “same host”, so a page on another port can call this API as long as it runs on the same host. Please compare against the full origin tuple (`scheme`, `host`, and `port`) or an explicit allowlist instead.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/httpapi/middleware.go` around lines 61 - 83, The function
resolveAllowedOrigin currently compares only hostnames; change it to compare
full origin tuples (scheme, hostname, port) instead. Parse requestHost and
boundHost into URLs (like you do for origin), normalize scheme and port (use
default ports for http/https when port is empty), build a canonical origin
string (scheme://hostname:port) for origin, request and bound, and then compare
those full canonical origins in the switch cases instead of
originHost/requestHostname/boundHostname; keep special-case loopback handling
but apply it to the full origin (or allow any loopback port if intended), and
preserve wildcard logic by matching bound origin appropriately (e.g., allow
wildcard host only when bound indicates it). Update helper calls (canonicalHost,
hostOnly, isLoopbackHost, isWildcardHost) or add small helpers to normalize and
compare full origins used by resolveAllowedOrigin.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Reasoning: `resolveAllowedOrigin` currently compares only canonical host names for the normal path, so a non-loopback origin on a different port can be treated as same-origin. That weakens the intended CORS boundary from origin matching to host matching.
- Fix approach: Canonicalize request, bound, and origin values as origin tuples, compare full origins for the normal path, preserve the explicit loopback-development allowance, and add regression tests for port-sensitive behavior.
