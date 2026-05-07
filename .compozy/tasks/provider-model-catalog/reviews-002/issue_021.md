---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/modelcatalog/service.go
line: 127
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AYaql,comment:PRRC_kwDOR5y4QM6-7HZI
---

# Issue 021: _⚠️ Potential issue_ | _🟠 Major_ | _🏗️ Heavy lift_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _🏗️ Heavy lift_

**Global refreshes still bypass coalescing and freshness checks.**

The all-provider path uses the `"__all__"` flight key and skips `sourceHasFreshStatus` entirely. That means a global refresh can run concurrently with a provider-scoped refresh for the same source/provider and re-fetch upstream even when the stored status is still fresh, leaving persisted rows/status as last-writer-wins.

 

Based on learnings "Keep execution paths deterministic and observable in Go backend."


Also applies to: 188-196, 350-378

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/modelcatalog/service.go` around lines 119 - 127, The global-refresh
path currently forces providerKey="__all__" (via refreshFlightScopeKey) which
bypasses per-provider coalescing and the sourceHasFreshStatus checks, so global
refreshes can run concurrently with provider-scoped refreshes and re-fetch fresh
data; change the coalescing key and refresh logic so global refreshes still
honor per-provider/source flight keys and freshness checks: when handling a
global refresh in the code around
providerKey/refreshFlightScopeKey/withRefreshFlight (and inside refreshSources),
generate per-source/provider-scoped flight keys (or iterate providers and call
withRefreshFlight per provider) instead of using a single "__all__" key, and
ensure sourceHasFreshStatus is consulted in refreshSources for each
source/provider path so global refreshes don't skip freshness/coalescing (see
functions refreshFlightScopeKey, withRefreshFlight, refreshSources, and
sourceHasFreshStatus).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `CatalogService.Refresh` still maps an empty `ProviderID` to the synthetic `__all__` flight key, so global refreshes do not share the same per-provider flight as provider-scoped refreshes.
  - `refreshSource` only consults `sourceHasFreshStatus` when `opts.ProviderID` is non-empty, so the global path bypasses freshness gating and can re-fetch already-fresh provider/source pairs.
  - Fix plan: fan global refreshes out into provider-scoped refresh flights so they reuse the same coalescing key and freshness logic as direct provider refreshes.
