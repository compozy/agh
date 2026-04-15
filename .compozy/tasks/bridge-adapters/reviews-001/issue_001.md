---
status: resolved
file: extensions/bridges/discord/provider.go
line: 754
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57Lwly,comment:PRRC_kwDOR5y4QM64DQzH
---

# Issue 001: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Block tenant-controlled API host overrides.**

`provider_config.api_base_url` flows straight into the outbound client and becomes the destination for `Authorization: Bot ...` requests. That turns bridge instance config into a token exfiltration / SSRF primitive. Keep host overrides env/test-only, or strictly allowlist Discord hosts before accepting them.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@extensions/bridges/discord/provider.go` around lines 752 - 754, The code
reads cfg.APIBaseURL into apiBaseURL (via firstNonEmpty + normalizeURL) allowing
tenant-controlled API hosts which can be used for token exfiltration/SSRF;
change this so provider_config.api_base_url is not accepted from tenant-managed
config: either only allow env/test overrides (remove cfg.APIBaseURL from
firstNonEmpty call) or validate the resolved host against a strict allowlist of
Discord hosts before using it (e.g., check the hostname of the URL returned by
normalizeURL and reject/ignore values not in the allowlist), ensuring apiBaseURL
is only set when the source is an allowed env/test value.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `resolveInstanceConfig()` still accepts `provider_config.api_base_url` ahead of the process-level env/default path, so a tenant-managed bridge instance can redirect bot-token-authenticated Discord API traffic to an arbitrary host.
  - The integration harness already uses `AGH_BRIDGE_DISCORD_API_BASE_URL` for test overrides, so the safe fix is to remove tenant config precedence here instead of preserving this provider-config override.
  - Planned fix: resolve the Discord API base URL from env/default only and add a regression test proving provider config cannot override it.
  - Resolution: `resolveInstanceConfig()` now ignores `provider_config.api_base_url` and resolves the Discord API base URL from the operator-controlled env/default path only; the config regression test now proves a tenant override is ignored.
  - Verification: `go test ./extensions/bridges/discord -count=1` and `make verify` both passed after the fix.
