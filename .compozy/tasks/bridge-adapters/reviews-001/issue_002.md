---
status: resolved
file: extensions/bridges/discord/provider.go
line: 894
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57Lwl2,comment:PRRC_kwDOR5y4QM64DQzQ
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Add defensive timeouts to the webhook server.**

This server is internet-facing, but it currently relies on the zero-value timeout settings. Please set at least `ReadHeaderTimeout` and `IdleTimeout` so slow clients cannot pin connections indefinitely.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@extensions/bridges/discord/provider.go` around lines 875 - 894, The
http.Server created as httpServer (used with p.serveWebhookHTTP and Serve(ln))
has zero timeouts and can be pinned by slow clients; set at least
ReadHeaderTimeout and IdleTimeout on that http.Server (e.g., ReadHeaderTimeout =
10*time.Second, IdleTimeout = 2*time.Minute) when constructing it so connections
can't be held indefinitely, keeping the rest of the flow (p.server,
p.serverAddr, Serve goroutine) unchanged.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `startServer()` builds an `http.Server` without `ReadHeaderTimeout` or `IdleTimeout`, leaving the public webhook listener on the Go zero-value timeout settings.
  - This is an internet-facing server, so slow-client pinning is a real resource-exhaustion risk.
  - Planned fix: set defensive webhook server timeouts and add a test that inspects the constructed server configuration.
  - Resolution: the Discord webhook server now sets explicit `ReadHeaderTimeout` and `IdleTimeout` constants, and the provider test asserts those timeout values on the initialized server.
  - Verification: `go test ./extensions/bridges/discord -count=1` and `make verify` both passed after the fix.
