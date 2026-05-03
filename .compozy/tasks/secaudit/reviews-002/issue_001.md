---
provider: coderabbit
pr: "90"
round: 2
round_created_at: 2026-05-03T03:57:53.330715Z
status: resolved
file: extensions/bridges/teams/provider.go
line: 2834
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_KWjl,comment:PRRC_kwDOR5y4QM69Zj0M
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_ | _🏗️ Heavy lift_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _🏗️ Heavy lift_

**Don't allow cleartext loopback endpoints for credentialed flows.**

`accessToken()` now trusts any URL accepted here, and that request carries `client_secret` in the POST body. The new `http` loopback branch means `token_url=http://127.0.0.1:...` is treated as valid, so a misconfigured or malicious instance can exfiltrate the bot secret to any local listener in cleartext. If loopback support is only needed for tests, split the validator or gate it behind a dedicated dev/test-only switch instead of allowing it in production.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@extensions/bridges/teams/provider.go` around lines 2823 - 2834, The validator
validTeamsCredentialedURL currently allows http loopback hosts which enables
cleartext token_url endpoints and can leak client_secret via accessToken();
change the validator to disallow any "http" scheme for credentialed flows by
removing or disabling the http case (isLoopbackTeamsHost) and only accept
"https" hosts (login.botframework.com or login.microsoftonline.com); if loopback
support is required for tests, add a separate test-only gate (e.g. an explicit
dev/test flag like ENABLE_TEAMS_LOOPBACK_FOR_TESTING) or provide a separate
test-only validator function and ensure accessToken() continues to call the
credentialed validator that rejects plain-http loopback URLs.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Reasoning: `validTeamsCredentialedURL()` still accepts `http://localhost` and loopback IPs, and both `fetchTeamsOpenIDMetadata()` and `teamsBotClient.accessToken()` trust that validator before issuing credentialed requests. `accessToken()` posts `client_secret` to the configured `token_url`, so the current branch still allows cleartext loopback credential exfiltration if a credentialed URL is misconfigured.
- Fix approach: make the credentialed validator reject plain-HTTP endpoints by default, add an explicit test-only loopback override for the provider tests, and keep the production credentialed flows on the strict validator.
- Resolution: credentialed Teams auth URLs now reject loopback `http` unless the dedicated test-only switch is set, and the provider tests opt into that switch only where local mock auth endpoints are required.
- Verification: `go test ./extensions/bridges/teams ./internal/network -count=1 -race`, `bunx vitest run packages/site/lib/public-install-contract.test.ts`, `sh -n packages/site/public/install.sh`, `make verify`.
