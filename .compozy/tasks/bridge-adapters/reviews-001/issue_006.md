---
status: resolved
file: extensions/bridges/gchat/provider.go
line: 989
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57Lwmc,comment:PRRC_kwDOR5y4QM64DQ0E
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Add timeouts to the webhook listener.**

This endpoint is exposed to external callers, but the server is using zero-value timeout settings. Please set `ReadHeaderTimeout` and `IdleTimeout` at minimum.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@extensions/bridges/gchat/provider.go` around lines 972 - 989, The http.Server
created in this block (httpServer) must set timeouts to avoid hanging external
requests: when constructing httpServer (used with p.serveWebhookHTTP and
Serve(ln)), add ReadHeaderTimeout and IdleTimeout (and optionally
ReadTimeout/WriteTimeout) with sensible durations (e.g., ReadHeaderTimeout: 5s,
IdleTimeout: 2m) instead of zero values; keep using ln, actualAddr, and the same
serve goroutine and error handling but ensure the server struct includes these
timeout fields before assigning p.server and starting Serve.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The Google Chat webhook listener is also built without explicit `ReadHeaderTimeout` or `IdleTimeout`, so it inherits the zero-value server timeouts.
  - This listener is exposed to external callers and should not allow indefinitely pinned slow connections.
  - Planned fix: add defensive server timeouts and cover them with a focused server-construction test.
  - Resolution: the Google Chat webhook server now sets explicit `ReadHeaderTimeout` and `IdleTimeout` values, and the initialization/config test asserts those timeout settings on the published server instance.
  - Verification: `go test -race ./extensions/bridges/gchat -count=1` and `make verify` both passed after the fix.
