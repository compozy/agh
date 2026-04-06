---
status: resolved
file: internal/memory/dream.go
line: 275
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoCV,comment:PRRC_kwDOR5y4QM61T6Hv
---

# Issue 017: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Wrap resolver and memory-store errors with operation-level context.**

`prepareWorkspace` currently returns raw dependency errors at two call sites. Wrapping them at source improves diagnosability and keeps error context precise.



<details>
<summary>Proposed fix</summary>

```diff
 func (s *Service) prepareWorkspace(ctx context.Context, workspaceRef string) (string, error) {
 	trimmedRef := strings.TrimSpace(workspaceRef)
 	if trimmedRef == "" {
 		return "", nil
 	}
 	if s.workspaceResolver == nil {
 		return "", errors.New("memory: workspace resolver is required")
 	}

 	resolved, err := s.workspaceResolver.Resolve(ctx, trimmedRef)
 	if err != nil {
-		return "", err
+		return "", fmt.Errorf("memory: resolve workspace %q: %w", trimmedRef, err)
 	}
 	if strings.TrimSpace(resolved.ID) == "" {
 		return "", errors.New("memory: workspace id is required")
 	}
 	if s.memStore != nil {
-		if err := s.memStore.ForWorkspace(resolved.RootDir).EnsureDirs(); err != nil {
-			return "", err
+		if err := s.memStore.ForWorkspace(resolved.RootDir).EnsureDirs(); err != nil {
+			return "", fmt.Errorf("memory: ensure workspace memory dirs for %q: %w", resolved.RootDir, err)
 		}
 	}

 	return strings.TrimSpace(resolved.ID), nil
 }
```
</details>

As per coding guidelines, "Use explicit error returns with wrapped context: `fmt.Errorf(\"context: %w\", err)`".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func (s *Service) prepareWorkspace(ctx context.Context, workspaceRef string) (string, error) {
	trimmedRef := strings.TrimSpace(workspaceRef)
	if trimmedRef == "" {
		return "", nil
	}
	if s.workspaceResolver == nil {
		return "", errors.New("memory: workspace resolver is required")
	}

	resolved, err := s.workspaceResolver.Resolve(ctx, trimmedRef)
	if err != nil {
		return "", fmt.Errorf("memory: resolve workspace %q: %w", trimmedRef, err)
	}
	if strings.TrimSpace(resolved.ID) == "" {
		return "", errors.New("memory: workspace id is required")
	}
	if s.memStore != nil {
		if err := s.memStore.ForWorkspace(resolved.RootDir).EnsureDirs(); err != nil {
			return "", fmt.Errorf("memory: ensure workspace memory dirs for %q: %w", resolved.RootDir, err)
		}
	}

	return strings.TrimSpace(resolved.ID), nil
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/memory/dream.go` around lines 265 - 275, In prepareWorkspace, wrap
errors returned from s.workspaceResolver.Resolve(ctx, trimmedRef) and from
s.memStore.ForWorkspace(resolved.RootDir).EnsureDirs() with operation-level
context using fmt.Errorf (e.g., fmt.Errorf("resolve workspace %q: %w",
trimmedRef, err) and fmt.Errorf("ensure workspace dirs for %q: %w",
resolved.RootDir, err)) so callers get precise, wrapped error traces while
keeping the original errors intact.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - `prepareWorkspace` currently returns raw resolver and memory-store errors.
  - That loses the operation context needed to understand whether failure came from workspace resolution or workspace memory directory preparation.
  - I will wrap both call sites with precise `fmt.Errorf(...: %w)` context and extend tests to lock the error contract.
  - Test coverage for this requires touching `internal/memory/dream_test.go`, which is outside the listed batch files but is the minimal validation surface for the fix.
