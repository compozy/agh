---
status: resolved
file: extensions/bridges/gchat/provider.go
line: 1827
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57Lwmu,comment:PRRC_kwDOR5y4QM64DQ0a
---

# Issue 008: _⚠️ Potential issue_ | _🟠 Major_

## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Avoid live cert fetches on every webhook verification.**

Signature verification currently blocks on a remote cert download for each request, and it does so via `http.DefaultClient` without an explicit timeout. That makes webhook availability depend on Google cert endpoint latency and turns verification into an easy cascading-failure point. Cache the keys with expiry and use a bounded client for refreshes.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@extensions/bridges/gchat/provider.go` around lines 1711 - 1827, The
verification currently calls fetchGoogleX509Keys on every request (used by
verifyDirectBearerToken and verifyPubSubBearerToken) and blocks on
http.DefaultClient with no timeout; change fetchGoogleX509Keys to use an
in-memory cache keyed by certsURL with an expiry/TTL and a singleflight or mutex
to coalesce concurrent refreshes, have verification read cached keys and only
trigger a refresh when expired (or background refresh), and use a bounded
http.Client with an explicit timeout/context when performing the remote GET;
ensure errors when refreshing fall back to existing cached keys (if present) to
avoid making webhook handling dependent on the cert endpoint latency.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `verifyDirectBearerToken()` and `verifyPubSubBearerToken()` call `fetchGoogleX509Keys()` on every request, and that helper currently uses `http.DefaultClient` with no timeout and no cache.
  - This makes each webhook verification depend on live cert-endpoint latency and introduces an avoidable availability bottleneck.
  - Planned fix: add a bounded cert-fetch client plus an in-memory URL-keyed cache with expiry and stale-on-refresh-failure behavior, then cover cache reuse and timeout-bound fetching in tests.
  - Resolution: verification now goes through a bounded `googleX509KeyCache` with a dedicated timeout-limited client, cache-expiry parsing, and stale-entry fallback when refresh fails; focused tests cover cache reuse, stale fallback, and bounded refresh behavior.
  - Verification: `go test -race ./extensions/bridges/gchat -count=1` and `make verify` both passed after the fix.
