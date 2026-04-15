---
status: resolved
file: extensions/bridges/gchat/provider.go
line: 850
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57LwmV,comment:PRRC_kwDOR5y4QM64DQz7
---

# Issue 005: _⚠️ Potential issue_ | _🔴 Critical_

## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Do not let instance config choose outbound auth/API/cert URLs.**

These fields are powerful enough to exfiltrate credentials: `oauth_token_url` receives the signed service-account assertion, `api_base_url` receives the bearer token, and the cert URLs create request-path SSRF. Please keep those overrides operator/test-only or validate them against a strict Google allowlist before accepting them from `provider_config`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@extensions/bridges/gchat/provider.go` around lines 826 - 850, The code
currently accepts operator-influential URLs from provider config (apiBaseURL,
tokenURL, directCertsURL, pubsubCertsURL) when building resolvedInstanceConfig;
change this to prevent untrusted provider_config overrides by only using
operator-controlled defaults or validated values: update the logic that sets
apiBaseURL, tokenURL, directCertsURL, and pubsubCertsURL (where
firstNonEmpty(...) and normalizeURL(...) are used) to ignore cfg values unless a
flagged operator/test mode is active, or validate them against a strict
allowlist via a new helper (e.g., isAllowedURL(url string) bool) and only accept
the value if it passes that check; ensure the resolvedInstanceConfig
construction uses the safe/validated URLs and add unit tests for both rejection
and acceptance of allowed URLs.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `resolveInstanceConfig()` currently accepts tenant-controlled `api_base_url`, `oauth_token_url`, `verification.direct_certs_url`, and `verification.pubsub_certs_url` values directly from `provider_config`.
  - That gives instance config control over bearer-token and signed-assertion destinations, plus cert-document fetch locations. The current test and integration harness already rely on env-driven API/token overrides, so there is room to remove the unsafe tenant precedence.
  - Planned fix: stop honoring tenant config for API/token endpoints, validate provider-config cert URL overrides against a strict Google allowlist, support operator/test cert overrides via process env, and add rejection/acceptance tests.
  - Resolution: Google Chat now ignores tenant `api_base_url` and `oauth_token_url` overrides, keeps operator/test endpoint overrides in env, and only accepts provider-config cert URLs when they stay on the strict `www.googleapis.com` allowlist; unit coverage now exercises both rejection and acceptance cases.
  - Verification: `go test -race ./extensions/bridges/gchat -count=1` and `make verify` both passed after the fix.
