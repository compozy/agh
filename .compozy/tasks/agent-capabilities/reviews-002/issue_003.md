---
status: pending
file: internal/config/capabilities.go
line: 186
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58BM3_,comment:PRRC_kwDOR5y4QM65LrXl
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Fail fast when reserved catalog paths exist with the wrong type.**

If `capabilities.toml` / `capabilities.json` is accidentally a directory, or `capabilities` is a regular file, these helpers return `(false, nil)` and `LoadAgentCapabilities()` behaves as if no catalog exists. That silently drops capabilities for a misconfigured agent instead of surfacing the configuration error.

<details>
<summary>Suggested fix</summary>

```diff
 func existingCapabilityCatalogFile(path string) (bool, error) {
 	info, err := os.Stat(path)
 	if err != nil {
 		if errors.Is(err, os.ErrNotExist) {
 			return false, nil
 		}
 		return false, fmt.Errorf("config: stat capability catalog file %q: %w", path, err)
 	}
-	return !info.IsDir(), nil
+	if info.IsDir() {
+		return false, fmt.Errorf("config: capability catalog file %q must be a file", path)
+	}
+	return true, nil
 }
 
 func existingCapabilityCatalogDir(path string) (bool, error) {
 	info, err := os.Stat(path)
 	if err != nil {
 		if errors.Is(err, os.ErrNotExist) {
 			return false, nil
 		}
 		return false, fmt.Errorf("config: stat capability catalog directory %q: %w", path, err)
 	}
-	return info.IsDir(), nil
+	if !info.IsDir() {
+		return false, fmt.Errorf("config: capability catalog directory %q must be a directory", path)
+	}
+	return true, nil
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func existingCapabilityCatalogFile(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("config: stat capability catalog file %q: %w", path, err)
	}
	if info.IsDir() {
		return false, fmt.Errorf("config: capability catalog file %q must be a file", path)
	}
	return true, nil
}

func existingCapabilityCatalogDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("config: stat capability catalog directory %q: %w", path, err)
	}
	if !info.IsDir() {
		return false, fmt.Errorf("config: capability catalog directory %q must be a directory", path)
	}
	return true, nil
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/config/capabilities.go` around lines 167 - 186, The two helpers
existingCapabilityCatalogFile and existingCapabilityCatalogDir currently treat a
path existing with the wrong type as “not present” and return (false, nil);
change them to fail fast by returning a descriptive error when the path exists
but is the wrong kind (e.g., existingCapabilityCatalogFile should return an
error if os.Stat succeeds but info.IsDir() is true, and
existingCapabilityCatalogDir should return an error if os.Stat succeeds but
info.IsDir() is false), and ensure callers such as LoadAgentCapabilities
propagate that error instead of treating it as absent.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
