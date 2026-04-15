---
status: resolved
file: extensions/bridges/github/provider.go
line: 219
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57Lwmx,comment:PRRC_kwDOR5y4QM64DQ0e
---

# Issue 009: _⚠️ Potential issue_ | _🟠 Major_

## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Reuse a `githubClient` per instance instead of recreating it per call.**

`githubClient` caches installation tokens in-memory, but this factory returns a fresh client on every auth check and delivery. In app mode that turns every operation into a new JWT + installation-token exchange, which adds latency and can hit GitHub auth rate limits. Keep the client, or at least the token cache, scoped to the bridge instance.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@extensions/bridges/github/provider.go` around lines 212 - 219, The apiFactory
currently returns a new githubClient for every call which prevents reuse of the
in-memory installation-token cache; change the implementation so each bridge
instance reuses a single githubClient (or at least a shared token cache) per
resolvedInstanceConfig instead of recreating it on every auth check/delivery.
Concretely, create and store a githubClient keyed by resolvedInstanceConfig (or
attach it to the provider instance) and have provider.apiFactory return that
stored client; keep using the same githubClient struct, http.Client with
Timeout, and now: func() time.Time { return provider.now() } when constructing
the cached client.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The default `apiFactory` still constructs a fresh `githubClient` for every call, which throws away the client’s in-memory installation-token cache each time.
  - In app mode that means repeated JWT signing and installation-token exchanges for auth checks, webhook-driven delivery, and subsequent updates on the same instance.
  - Planned fix: cache the default `githubClient` per bridge instance inside the provider, reset stale entries on reconciliation, and add a regression test showing repeated default-factory calls reuse the same client.
  - Resolution: the default GitHub API factory now caches and reuses one `githubClient` per bridge instance via `p.apiClients`, preserving the installation-token cache across auth checks and deliveries; focused unit coverage now asserts repeated factory calls reuse the same client.
  - Verification: `go test ./extensions/bridges/github -count=1` and `make verify` both passed after the fix.
