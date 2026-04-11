---
status: resolved
file: internal/skills/registry_workspace_cache.go
line: 75
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56LrYB,comment:PRRC_kwDOR5y4QM62twdh
---

# Issue 017: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Wrap propagated errors with local context.**

At Line 69 and Line 74, errors are returned directly. Please wrap them so failures in workspace loading are diagnosable.


<details>
<summary>Suggested fix</summary>

```diff
 	for _, skillPath := range resolved.Skills {
 		if err := checkRegistryContext(ctx); err != nil {
-			return workspaceLoad{}, err
+			return workspaceLoad{}, fmt.Errorf("skills: check registry context while loading workspace skills: %w", err)
 		}
 
 		source, include, err := skillSourceFromWorkspacePath(skillPath.Source)
 		if err != nil {
-			return workspaceLoad{}, err
+			return workspaceLoad{}, fmt.Errorf("skills: resolve workspace skill source %q: %w", skillPath.Source, err)
 		}
```
</details>
As per coding guidelines, `Use explicit error returns with wrapped context: fmt.Errorf("context: %w", err)`.

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
		if err := checkRegistryContext(ctx); err != nil {
			return workspaceLoad{}, fmt.Errorf("skills: check registry context while loading workspace skills: %w", err)
		}

		source, include, err := skillSourceFromWorkspacePath(skillPath.Source)
		if err != nil {
			return workspaceLoad{}, fmt.Errorf("skills: resolve workspace skill source %q: %w", skillPath.Source, err)
		}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/skills/registry_workspace_cache.go` around lines 68 - 75, The
returns from checkRegistryContext(ctx) and
skillSourceFromWorkspacePath(skillPath.Source) should be wrapped with local
context before propagating so failures in workspace loading are diagnosable;
replace direct returns like `return workspaceLoad{}, err` with wrapped errors
using fmt.Errorf to add context (e.g., "checking registry context failed: %w")
when handling the error from checkRegistryContext and similarly add context for
skillSourceFromWorkspacePath (e.g., "determining skill source from workspace
path failed: %w"), keeping references to the existing symbols
checkRegistryContext, skillSourceFromWorkspacePath, and the workspaceLoad return
value.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Reasoning: `workspaceLoadFromResolved` returns raw errors from `checkRegistryContext` and `skillSourceFromWorkspacePath`, which obscures whether the failure was cancellation or workspace-skill source resolution.
- Fix approach: Wrap both errors with local context so workspace loading failures are diagnosable.
