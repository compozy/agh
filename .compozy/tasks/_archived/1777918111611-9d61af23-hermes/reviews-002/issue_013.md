---
status: resolved
file: internal/memory/store.go
line: 490
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59mLif,comment:PRRC_kwDOR5y4QM67SmDm
---

# Issue 013: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Reuse `normalizeScopeAndWorkspace` in `History`.**

`History` is the only API here that doesn't derive the store-bound workspace when `query.Workspace` is empty. On a workspace-scoped store, `History(ctx, OperationHistoryQuery{})` therefore drops `s.workspaceDir`, which can under-filter local results or, with the current SQL, pull in other workspaces from the shared catalog.

<details>
<summary>Suggested fix</summary>

```diff
	normalized := query
-	normalized.Scope = query.Scope.Normalize()
-	normalized.Workspace = canonicalWorkspaceRoot(query.Workspace)
+	scope, workspaceRoot, err := s.normalizeScopeAndWorkspace(query.Scope, query.Workspace)
+	if err != nil {
+		return nil, err
+	}
+	normalized.Scope = scope
+	normalized.Workspace = workspaceRoot
	normalized.Operation = query.Operation.Normalize()
	return s.catalog.listOperations(ctx, normalized)
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	normalized := query
	scope, workspaceRoot, err := s.normalizeScopeAndWorkspace(query.Scope, query.Workspace)
	if err != nil {
		return nil, err
	}
	normalized.Scope = scope
	normalized.Workspace = workspaceRoot
	normalized.Operation = query.Operation.Normalize()
	return s.catalog.listOperations(ctx, normalized)
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/memory/store.go` around lines 487 - 490, The History method
currently does manual normalization (setting normalized.Scope,
normalized.Workspace, normalized.Operation) but omits the store-bound workspace
behavior; replace that manual handling by calling the existing
normalizeScopeAndWorkspace helper so the store's workspace is applied when
query.Workspace is empty. Specifically, in History use
normalizeScopeAndWorkspace(&normalized, s.workspaceDir) (or the helper's actual
signature) to set Scope and Workspace, then still call Operation.Normalize() on
normalized.Operation if needed, removing the direct canonicalWorkspaceRoot and
Scope.Normalize calls in this block.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `History` normalizes query workspace manually and does not apply the store-bound workspace default when `query.Workspace` is empty, so workspace stores can under-filter operation history.
- Fix approach: reuse `normalizeScopeAndWorkspace` in `History` and keep operation normalization, preserving existing validation while deriving the bound workspace root.
