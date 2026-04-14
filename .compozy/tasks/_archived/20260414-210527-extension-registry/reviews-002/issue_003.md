---
status: resolved
file: internal/extension/registry.go
line: 403
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM563lMa,comment:PRRC_kwDOR5y4QM63oCtD
---

# Issue 003: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Wrap the upsert error before returning it.**

Line 401 returns the raw DB error for `replaceExisting`, so callers lose which install failed. This branch should keep the same context-rich wrapping as the rest of the registry writes.

<details>
<summary>💡 Proposed fix</summary>

```diff
 	if err != nil {
 		if config.replaceExisting {
-			return err
+			return fmt.Errorf("extension: persist %q: %w", info.Name, err)
 		}
 		return mapRegistryConstraintError(err, info.Name)
 	}
```
</details>
As per coding guidelines, `**/*.go`: Use explicit error returns with wrapped context: `fmt.Errorf("context: %w", err)`.

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	if err != nil {
		if config.replaceExisting {
			return fmt.Errorf("extension: persist %q: %w", info.Name, err)
		}
		return mapRegistryConstraintError(err, info.Name)
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/registry.go` around lines 399 - 403, The branch that
returns raw DB errors when config.replaceExisting is true should wrap the error
with context before returning; update the return in that branch to use
fmt.Errorf to include a descriptive message mentioning info.Name and wrap the
original err (same pattern used elsewhere and similar to
mapRegistryConstraintError usage) so callers know which install failed.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  Marked completed (resolved).
