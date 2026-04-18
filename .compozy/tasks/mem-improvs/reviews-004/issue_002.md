---
status: resolved
file: internal/memory/store.go
line: 582
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575hoy,comment:PRRC_kwDOR5y4QM65B3LV
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Normalize explicit workspace paths before using them.**

If a caller passes `<workspace>/.agh/memory` here, this helper keeps that suffix. The later `ForWorkspace(workspaceRoot)` call then resolves to `<workspace>/.agh/memory/.agh/memory`, so `Search`/`Reindex` read the wrong directory and can split one workspace into two catalog keys.  


<details>
<summary>🐛 Proposed fix</summary>

```diff
-	workspaceRoot := cleanDirPath(workspace)
+	workspaceRoot := deriveWorkspaceRoot(cleanDirPath(workspace))
 	if workspaceRoot == "" {
 		workspaceRoot = deriveWorkspaceRoot(s.workspaceDir)
 	}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/memory/store.go` around lines 579 - 582, The code currently uses
cleanDirPath(workspace) directly which preserves an explicit
"<workspace>/.agh/memory" suffix and causes ForWorkspace(workspaceRoot) to
resolve to "<workspace>/.agh/memory/.agh/memory"; fix by normalizing and
stripping any trailing ".agh/memory" before using workspaceRoot: after computing
workspaceRoot := cleanDirPath(workspace) (and after the deriveWorkspaceRoot
fallback), detect if workspaceRoot refers inside the memory subdir (compare
against filepath.Join(".agh","memory") or use strings.HasSuffix with
filepath.Join) and if so set workspaceRoot to the parent workspace directory
(remove the ".agh/memory" suffix) so subsequent calls like
ForWorkspace(workspaceRoot), Search and Reindex operate on the correct root.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `normalizeScopeAndWorkspace` currently uses `cleanDirPath(workspace)` directly, so an explicit `<workspace>/.agh/memory` input remains unchanged.
  - Later workspace-scoped operations call `ForWorkspace(workspaceRoot)`, which appends `.agh/memory` again and can target `<workspace>/.agh/memory/.agh/memory`.
  - That can split one logical workspace across two catalog keys and make workspace search/reindex target the wrong directory.
  - Fix approach: canonicalize explicit workspace inputs to the workspace root before returning from `normalizeScopeAndWorkspace`, while still falling back to `deriveWorkspaceRoot(s.workspaceDir)` when no explicit workspace is supplied.
  - A focused regression test is required in `internal/memory/store_test.go` even though it is outside the listed code files, because the bug is in workspace path normalization and needs direct coverage in the memory package tests.
  - Resolved by normalizing explicit workspace inputs and by making `Store.ForWorkspace` treat memory-dir inputs as the canonical workspace root.
  - Verified with `go test ./internal/memory -run TestStoreNormalizesExplicitWorkspacePaths` and the full `make verify` gate.
