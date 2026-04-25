---
status: resolved
file: internal/memory/store.go
line: 356
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59mLie,comment:PRRC_kwDOR5y4QM67SmDl
---

# Issue 012: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don't persist workspace-bound search/reindex events with `scope=""`.**

When `opts.Scope` is omitted but `opts.Workspace` resolves to a concrete workspace, these records are written with `Workspace=<root>` and `Scope=""`. The readers still treat `scope=""` as globally/legacy visible in `internal/memory/catalog.go`, so a search or reindex from workspace A can surface in workspace B history and in unrelated `HealthStats`. Please either emit a concrete scope for new records or make empty-scope rows match only the same `workspace_root`.



Also applies to: 394-404

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/memory/store.go` around lines 349 - 356, The issue is that when
opts.Scope is empty but opts.Workspace resolves to a concrete workspace, you
currently log records with Scope="" which is treated as global; update the code
that constructs the OperationRecord for calls to s.logCatalogEvent (e.g., the
OperationSearch and OperationReindex records) to replace an empty normalized
scope with a workspace-specific scope: compute scopeNorm := scope.Normalize();
if scopeNorm == "" && workspaceRoot != "" { scopeNorm = workspaceRoot }; then
set OperationRecord.Scope = scopeNorm so events are persisted with a concrete,
workspace-bound scope (apply the same change to the other logging site around
lines 394-404).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `Search` and `Reindex` log catalog operations with `Scope: scope.Normalize()`, which is empty when callers omit scope but a workspace-bound store resolves a concrete workspace. Catalog readers treat empty scope as global/legacy-visible, allowing workspace operation history and health stats to bleed across workspaces.
- Fix approach: derive the operation-log scope from the resolved workspace when scope is omitted, persisting new workspace-bound search/reindex events as `scope="workspace"` with the canonical workspace root.
