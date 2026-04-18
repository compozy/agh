---
status: resolved
file: internal/api/httpapi/routes.go
line: 240
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575kRk,comment:PRRC_kwDOR5y4QM65B60G
---

# Issue 007: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Guard the log-tail endpoint on non-loopback HTTP.**

`GET /api/settings/observability/log-tail` currently stays readable without `privilegedMutationGuard()`. Unlike the other settings reads, this streams raw daemon logs, so a daemon bound to `0.0.0.0` can expose prompts, tokens, paths, and other operational data to remote clients.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/httpapi/routes.go` around lines 237 - 240, The log-tail route
currently allows unauthenticated remote reads of daemon logs; update the
observability route registration to require the same guard used elsewhere so
remote clients cannot access it. Specifically, change the registration of
StreamSettingsObservabilityLogTail (observability.GET("/log-tail",
handlers.StreamSettingsObservabilityLogTail)) to include the privileged guard
(e.g., observability.GET("/log-tail", privileged,
handlers.StreamSettingsObservabilityLogTail) or the equivalent
privilegedMutationGuard), so the endpoint is protected from non-loopback HTTP.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  Root cause confirmed in `registerSettingsRoutes`: `/api/settings/observability/log-tail` is registered without `privilegedMutationGuard()`, even though it exposes raw daemon log lines over HTTP. I will apply the same loopback-only guard used for privileged settings mutations and add/adjust coverage in the HTTP server tests.
