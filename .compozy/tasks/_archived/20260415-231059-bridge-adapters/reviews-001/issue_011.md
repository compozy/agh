---
status: resolved
file: extensions/bridges/github/provider.go
line: 604
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57Lwm7,comment:PRRC_kwDOR5y4QM64DQ0p
---

# Issue 011: _⚠️ Potential issue_ | _🔴 Critical_

## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Reject shared webhook paths across GitHub instances.**

Signature verification later accepts any secret from `configsForPath`, and routing then picks the instance from `payload.Repository.FullName`. If two instances share a webhook path, a request signed with instance A's secret can spoof repo B in the JSON body and be routed to B. Enforce unique `webhookPath` values during reconciliation, or bind verification to the selected repo before accepting the request.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@extensions/bridges/github/provider.go` around lines 587 - 604, The code
currently checks duplicate repos via seenRepos but does not reject shared
webhook paths, which lets a payload signed by instance A spoof instance B; add a
seenWebhookPaths map and in the same reconciliation loop (where
resolveInstanceConfig returns cfg) reject duplicate webhook paths by setting
cfg.configError (similar to the repo conflict) when cfg.webhookPath != "" and
seenWebhookPaths[cfg.webhookPath] exists, e.g. cfg.configError =
fmt.Errorf("github: webhook path %q is already owned by %q and cannot also
belong to %q", cfg.webhookPath, ownerInstance, cfg.instanceID); also record
seenWebhookPaths[cfg.webhookPath] = cfg.instanceID when non-empty so
configsForPath cannot contain multiple instances for the same path (or
alternatively ensure signature verification consults configsForPath and binds
verification to payload.Repository.FullName before accepting).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - GitHub reconciliation rejects duplicate repositories but still allows multiple instances to share the same webhook path.
  - Because signature verification accepts any matching secret from `configsForPath()` and routing later chooses the instance by repository identity in the payload, shared paths create a spoofing boundary between instances.
  - Planned fix: reject duplicate webhook paths during reconciliation and add a unit test covering the conflicting-path case.
  - Resolution: GitHub reconciliation now tracks `seenWebhookPaths` and marks duplicate webhook paths as configuration errors before routing is published; unit coverage now exercises the conflicting-path case directly.
  - Verification: `go test ./extensions/bridges/github -count=1` and `make verify` both passed after the fix.
