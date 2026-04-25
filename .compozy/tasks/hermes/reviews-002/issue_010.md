---
status: resolved
file: internal/mcp/auth/service.go
line: 233
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59mLid,comment:PRRC_kwDOR5y4QM67SmDk
---

# Issue 010: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Make local logout independent of revocation success.**

Right now any metadata lookup or revocation error returns before `DeleteMCPAuthToken`, so a temporary network failure leaves the persisted token in place and the user is still logged in locally. Logout should clear local state even when remote revoke is best-effort.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/mcp/auth/service.go` around lines 219 - 233, The current flow
returns early on discoverMetadata or s.revoke errors which prevents
s.store.DeleteMCPAuthToken from running; change the logic so metadata discovery
and revoke attempts (discoverMetadata and s.revoke) are best-effort: capture/log
any metaErr or revokeErr but do NOT return immediately, ensure
s.store.DeleteMCPAuthToken(ctx, cfg.ServerName) always runs (e.g., by
restructuring the if-block or using a defer/cleanup path), and then return
statusFromToken(cfg, nil, s.now()), nil; keep unique calls discoverMetadata,
s.revoke, s.store.DeleteMCPAuthToken, and statusFromToken so reviewers can find
the changes.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `Logout` returns on metadata discovery or remote revocation errors before deleting the local durable token, so transient network failures keep the user logged in locally.
- Fix approach: treat metadata discovery and revocation as best-effort, always attempt local token deletion, and surface any remote cleanup failure as a redacted diagnostic in the returned logged-out status rather than preserving local credentials.
