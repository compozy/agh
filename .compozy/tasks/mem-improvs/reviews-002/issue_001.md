---
status: resolved
file: internal/memory/store.go
line: 443
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575RSy,comment:PRRC_kwDOR5y4QM65BgwH
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Health stats miss first-time workspace indexing once the catalog has any rows.**

`entryCount` is checked globally, not for the requested filters. If the catalog already has global entries, a later `HealthStats()` call for a brand-new workspace skips `reindexScopes()`, so that workspace's markdown files never make it into `entries` and also don't count as orphaned. The API can therefore report a clean workspace health snapshot even though that workspace has never been indexed. Please make readiness/reindex decisions per requested scope/workspace instead of per whole catalog.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/memory/store.go` around lines 427 - 443, The current logic uses
s.catalog.entryCount(ctx) and only reindexes when the global count is zero,
which skips initial indexing for a new workspace if the catalog already has
rows; change this to check counts per requested scope/workspace and call
s.reindexScopes for each missing scope: for ScopeGlobal still check a global
count and call s.reindexScopes(ctx, ScopeGlobal, "") when its count==0, and for
each filter where filter.scope==ScopeWorkspace call a per-workspace count (e.g.,
a catalog method that accepts a workspaceRoot or use a filtered query) and call
s.reindexScopes(ctx, ScopeWorkspace, filter.workspaceRoot) when that
workspace-specific count==0 so new workspaces get indexed and show up in
HealthStats.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `HealthStats()` checks `s.catalog.entryCount(ctx)` across the whole catalog, so any existing global rows make a first-time workspace look "ready" even when that workspace has never been indexed.
- Impact: workspace health can under-report indexed/orphaned files because the workspace scope is skipped entirely.
- Fix plan: add per-scope catalog readiness/count checks and use them for workspace-specific reindex decisions. Apply the same readiness rule to the shared catalog bootstrap path so new workspaces are indexed independently.
- Resolution: replaced the global-only readiness check with per-scope catalog counts, added scope-specific readiness helpers, and reused them in both `HealthStats()` and the shared search bootstrap path so a fresh workspace is indexed even when global rows already exist.
- Verification: `go test ./internal/memory`; `make verify`
